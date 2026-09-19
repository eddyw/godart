# bench

This directory measures the cost of a text property at the boundary between a Go
tree and the Dart host. The fixtures separate conversion and widget costs.

## The question

A text property crosses the boundary 2 times at each build.

1. The Go build writes the text. gart emits `GoString.fromString` for each Go
   string literal, thus each evaluation encodes the text as UTF-8. A read of
   `flutter/bench/gen/lib/bench/bench.part.dart` shows the call.
2. The Dart host reads the text. `GoString.toDartString` of
   `runtime/goruntime/lib/src/string.dart` decodes the bytes into a Dart
   string.

The frame budget at 60 hertz is 16,700 microseconds. The measurement gives the
part of that budget that the pair takes, over a sweep of the node count and the
text length.

The fixture compares a complete tree walk (`A5`), a tree diff (`A10`), and
signal boundaries. These names identify benchmark strategies. They do not
identify current production architectures.

## Run it again

```sh
make build                 # from the root of the repository, writes build/gart
RUNS=9 flutter/bench/run.sh   # 9 process runs
cat flutter/bench/report.md
```

`RUNS` gives the number of process runs, and the default is 5. `dart run
bin/check.dart` from `dart/` prints what each diff path reports. It runs no
measurement.

The script runs `go vet` and `go test` over `go/`. It compiles `go/` with
`build/gart` into `gen/`. It then builds `dart/bin/bench.dart` with `dart
compile exe`, and it runs the executable. `dart/bin/report.dart` then takes the
median over each batch and writes `report.md`.

`gen/`, `bench.exe`, `out.csv`, and `report.md` stay out of git.

`flutter/bench/flutter` holds a second harness. It measures the Flutter work
that 1 unchanged node skips, and it needs a profile build of a real
application:

```sh
RUNS=3 FLUTTER=/path/to/flutter flutter/bench/flutter/run.sh
```

The script writes the macOS runner with `flutter create`, thus git holds no
such directory. The run opens a window of the application. The generated report holds the measurements.

**The machine must stand near idle.** A parallel build moves each number by a
factor of 3 or more. `run.sh` prints the load average before each process run
and after the last one, thus a reader can throw a run away.

## The shape

`go/bench.go` builds a `ui.Column` of N `ui.Text` children, thus the tree holds
N plus 1 nodes and N text properties. Each text holds L characters.

`dart/lib/walk.dart` reads the tree the way the real host reads it. It holds the
field cache of `GoNode` of `flutter/goflutter/lib/src/node.dart`. It reads each
field that the generated host of an application reads: 13 fields of a `Text`,
and 8 of a `Column` or a `Row`.

The walk builds no Flutter widget, for 2 reasons. The measurement runs under
`dart compile exe`, and `package:flutter` needs the Flutter engine. The widget
construction is also work that a Dart application does too, thus it hides the
price of the boundary.

## The cases

| Case | What it holds |
|---|---|
| `micro.fromString` | 1 call of `GoString.fromString`. |
| `micro.take` | 1 call of `GoString.take` over `utf8.encode`. `GoString.fromString` copies the bytes a second time, and this case gives the floor without that copy. |
| `micro.toDartString` | 1 call of `GoString.toDartString`. |
| `micro.decodePlain` | 1 call of `utf8.decode` over a plain `Uint8List`. The pair with the case above answers whether the unmodifiable view costs anything. |
| `go.held` | The Go build, with the text held in the model already. No conversion runs, thus the number gives the structure cost alone. |
| `go.literal` | The Go build, with the text from a Go literal. Each node pays 1 `GoString.fromString` of L characters. |
| `go.itoa` | The Go build, with `strconv.Itoa` and a concatenation for each node. |
| `dart.walk.text` | The host walk, with `toDartString` for each text property. It reads each field that the generated host reads. |
| `dart.walk.data` | The same walk, with the text property alone. The difference against `dart.walk.text` is the price of the optional fields. |
| `dart.walk.raw` | The same walk, with no decode. The difference against `dart.walk.text` is the price of the decode. |
| `frame.go` | 1 whole frame: `go.literal` and then `dart.walk.text`. |
| `frame.go.itoa` | 1 whole frame, with the text from a number. |
| `frame.dart` | The same shape in plain Dart, with a Dart literal. This is the baseline of an application that a developer writes in Dart. |
| `frame.dart.concat` | The same, with a concatenation of a literal and a number. |
| `diff.same` | The reconciler over 2 trees that hold the same text. This is the floor of the A10 cost, and a rebuild with no visible change hits it. |
| `diff.one` | The reconciler over 2 trees where 1 leaf text differs. |
| `diff.all` | The reconciler over 2 trees where each leaf text differs. |
| `diff.eq.same` | The same, through the generated `==` of the node. This is a negative control. Go identity equality does not compare widget fields. |
| `diff.eq.all` | The same for a tree where each leaf differs. |
| `frame.a5` | 1 whole frame of A5: the Go build, and then the walk of each node. |
| `frame.a10.same` | 1 whole frame of A10: the Go build, the diff, and the widget build of each node that changed. Nothing changed here. |
| `frame.a10.one` | The same, with 1 leaf changed. |
| `frame.a10.all` | The same, with each leaf changed. |
| `watch.signal` | A write of 1 signal and a read of the dirty set. No build and no walk run. |
| `watch.one` | 1 frame of the normal case: 1 signal changes, and the host rebuilds the 1 boundary that read it. The boundary holds 1 text node. |
| `watch.three` | The same, with a boundary of a `ui.Row` and 3 text nodes. |

`dart/bin/check.dart` runs no measurement. It prints the count that each diff
path reports for 5 shapes. A reader then sees that the field diff is right and
the `==` diff is wrong.

The sweep takes N of 20, 50, 200, and 500, and L of 5, 20, and 80 characters. 1
more case holds 20 runes in 37 bytes of UTF-8. That pair answers whether the
byte count or the rune count drives the cost.

## What the number leaves out

- The Flutter element and render work of a rebuild. That work is the same for a
  Go tree and for a Dart tree. The separate Flutter harness measures this work.
- The Flutter widget construction of the host, for the reason above.
- `setState` and the goroutine that an event handler starts. `watch.one` holds
  the Go graph, the boundary build, and the boundary walk alone.
- A `flutter run` on a device. The measurement runs an ahead of time build of
  the Dart SDK on macOS, and a phone runs slower.

## Interpretation

Use a profile application for Flutter cost measurements. Debug widget tests
validate harness behavior but do not establish production performance.

The tree diff still pays for Go tree construction. Signal boundaries can avoid
that construction outside the changed boundary. Compare the complete workload,
including Flutter element work, before a production design decision.

Record SDK versions, source revision, host, and machine load with each report.
Keep dated measurements with their validation artifacts.

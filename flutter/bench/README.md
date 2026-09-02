# bench

This directory measures the cost of a text property at the boundary between a
Go tree and the Dart host. `docs/flutter-first.md` names that cost as
unmeasured, and this directory answers it with a number.

## The question

A text property crosses the boundary 2 times at each build.

1. The Go build writes the text. gart emits `GoString.fromString` for each Go
   string literal, thus each evaluation encodes the text as UTF-8. A read of
   `flutter/bench/gen/lib/bench/bench.part.dart` shows the call.
2. The Dart host reads the text. `GoString.toDartString` of
   `runtime/goruntime/lib/src/string.dart:52` decodes the bytes into a Dart
   string.

The frame budget at 60 hertz is 16,700 microseconds. The measurement gives the
part of that budget that the pair takes, over a sweep of the node count and the
text length.

## Run it again

```sh
make build                 # from the root of the repository, writes build/gart
RUNS=9 flutter/bench/run.sh   # 9 process runs, about 7 minutes
cat flutter/bench/report.md
```

`RUNS` gives the number of process runs, and the default is 5.

The script runs `go vet` and `go test` over `go/`, it compiles `go/` with
`build/gart` into `gen/`, it builds `dart/bin/bench.dart` with
`dart compile exe`, and it runs the executable. `dart/bin/report.dart` then
takes the median over each batch and writes `report.md`.

`gen/`, `bench.exe`, `out.csv`, and `report.md` stay out of git.

**The machine must stand near idle.** A parallel build moves each number by a
factor of 3 or more. `run.sh` prints the load average before each process run
and after the last one, thus a reader can throw a run away.

## The shape

`go/bench.go` builds a `ui.Column` of N `ui.Text` children, thus the tree holds
N plus 1 nodes and N text properties. Each text holds L characters.

`dart/lib/walk.dart` reads the tree the way the real host reads it. It holds the
field cache of `GoNode` of `flutter/goflutter/lib/src/node.dart`, and it reads
each field that the generated host of an application reads: 13 fields of a
`Text`, and 8 of a `Column` or a `Row`.

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
| `go.itoa` | The Go build, with `strconv.Itoa` and a concatenation for each node. The measurement of 2026-09-01 holds this shape. |
| `dart.walk.text` | The host walk, with `toDartString` for each text property. It reads each field that the generated host reads. |
| `dart.walk.data` | The same walk, with the text property alone. The difference against `dart.walk.text` is the price of the optional fields. |
| `dart.walk.raw` | The same walk, with no decode. The difference against `dart.walk.text` is the price of the decode. |
| `frame.go` | 1 whole frame: `go.literal` and then `dart.walk.text`. |
| `frame.go.itoa` | 1 whole frame, with the text from a number. |
| `frame.dart` | The same shape in plain Dart, with a Dart literal. This is the baseline of an application that a developer writes in Dart. |
| `frame.dart.concat` | The same, with a concatenation of a literal and a number. |
| `watch.signal` | A write of 1 signal and a read of the dirty set. No build and no walk run. |
| `watch.one` | 1 frame of the normal case: 1 signal changes, and the host rebuilds the 1 boundary that read it. The boundary holds 1 text node. |
| `watch.three` | The same, with a boundary of a `ui.Row` and 3 text nodes. |

The sweep takes N of 20, 50, 200, and 500, and L of 5, 20, and 80 characters.
1 more case holds 20 runes in 37 bytes of UTF-8, thus a reader can see whether
the byte count or the rune count drives the cost.

## What the number leaves out

- The Flutter element and render work of a rebuild. That work is the same for a
  Go tree and for a Dart tree.
- The Flutter widget construction of the host, for the reason above.
- `setState` and the goroutine that an event handler starts. `watch.one` holds
  the Go graph, the boundary build, and the boundary walk alone.
- A `flutter run` on a device. The measurement runs an ahead of time build of
  the Dart SDK on macOS, and a phone runs slower.

## The result of 2026-09-03

The machine of the project: Apple M4 Max, 16 cores, macOS 27.0, Dart SDK 3.13.2
on macos_arm64. The compiler is `build/gart` of 2026-09-02 17:21.

8 process runs of 15 batches gave 120 batches for each cell. Each run stood at
a load average of 11 or under. 2 invocations of `run.sh` gave 18 runs in all,
and another agent loaded the machine over 10 of them. The report reads the 8
quiet runs, and `out.csv` holds them.

The spread of the middle half of the batches is 2 percent for each cell. A rare
batch reads 5 times the median, because the scheduler of macOS interrupts it.
`report.md` holds the quarters and the largest batch of each case.

### 1 crossing of the boundary

| Text | `GoString.fromString` | `GoString.toDartString` | The pair |
|---|---|---|---|
| 5 characters | 30 ns | 11 ns | 41 ns |
| 20 characters | 60 ns | 16 ns | 76 ns |
| 80 characters | 185 ns | 33 ns | 218 ns |
| 20 runes in 37 bytes | 79 ns | 67 ns | 146 ns |

The encode costs 3 to 6 times the decode. `GoString.fromString` scans for an
escape unit, it encodes, and `GoString.fromBytes` then copies the result a
second time. `GoString.take` over `utf8.encode` gives the same value and skips
that copy, and it costs 22, 42, and 126 nanoseconds for the 3 ASCII lengths.
That is a saving of 27 to 32 percent, and it needs a change of
`runtime/goruntime/lib/src/string.dart`.

A byte that holds no ASCII costs much more. 20 runes in 37 bytes decode in
67 nanoseconds, and 20 ASCII characters decode in 16.

### A rebuild of the whole tree

Each number is the median of 120 batches, in microseconds.

| Nodes | Text | The frame | Part of 16,700 us | The GoString share | Dart baseline |
|---|---|---|---|---|---|
| 20 | 5 | 5.80 | 0.03% | 1.05 (0.01%) | 0.142 |
| 20 | 20 | 6.45 | 0.04% | 1.79 (0.01%) | 0.143 |
| 20 | 80 | 9.37 | 0.06% | 4.68 (0.03%) | 0.144 |
| 50 | 5 | 14.1 | 0.08% | 2.76 (0.02%) | 0.315 |
| 50 | 20 | 15.8 | 0.09% | 4.45 (0.03%) | 0.322 |
| 50 | 80 | 22.9 | 0.14% | 11.5 (0.07%) | 0.319 |
| 200 | 5 | 55.2 | 0.33% | 11.0 (0.07%) | 1.30 |
| 200 | 20 | 61.4 | 0.37% | 17.8 (0.11%) | 1.31 |
| 200 | 80 | 91.4 | 0.55% | 48.4 (0.29%) | 1.31 |
| 200 | 20 runes, 37 bytes | 76.4 | 0.46% | 30.6 (0.18%) | - |
| 500 | 5 | 137.1 | 0.82% | 25.3 (0.15%) | 2.82 |
| 500 | 20 | 156.5 | 0.94% | 42.8 (0.26%) | 2.87 |
| 500 | 80 | 229.5 | 1.4% | 115.3 (0.69%) | 2.87 |

### A rebuild of 1 boundary

1 signal changes, thus 1 boundary rebuilds. This is the normal case of the
framework, and the table above holds the worst case.

| Nodes in the tree | Text | 1 text node | 3 text nodes | The signal alone |
|---|---|---|---|---|
| 200 | 5 | 0.554 | 1.35 | 0.103 |
| 200 | 20 | 0.577 | 1.47 | 0.103 |
| 200 | 80 | 0.714 | 1.86 | 0.103 |
| 500 | 5 | 0.539 | 1.34 | 0.101 |
| 500 | 20 | 0.572 | 1.45 | 0.101 |
| 500 | 80 | 0.722 | 1.88 | 0.101 |

A rebuild of 1 boundary takes 0.6 microseconds, which is 0.003 percent of the
frame budget. The node count of the whole tree does not reach the number, thus
the graph of `flutter/ui/signal.go` costs nothing at the scale of a screen.

### Where the time goes

The smallest batch of each case, for a tree of 200 nodes, in microseconds.

| Text | Go structure | Go `fromString` | Dart, the text field | Dart, the other fields | The frame |
|---|---|---|---|---|---|
| 5 | 1.97 | 6.01 | 7.31 | 37.3 | 52.0 |
| 20 | 1.96 | 11.5 | 8.04 | 37.4 | 58.3 |
| 80 | 1.85 | 35.9 | 11.9 | 37.7 | 87.6 |

The text conversions take 22 to 55 percent of the frame, and the read of the
optional fields takes 43 to 72 percent. A `ui.Text` holds 12 optional fields,
and the host reads each 1 through a map of the descriptor. Each such read costs
about 15 nanoseconds, thus a `Text` node costs about 190 nanoseconds before any
text work.

### The reading

**The cost of a text property does not matter at 60 hertz.** A dense screen of
200 nodes with a line of 20 characters for each 1 spends 61 microseconds of the
16,700 microsecond budget, which is 0.37 percent. The `GoString` part of that is
18 microseconds, which is 0.11 percent. The normal case, a rebuild of 1
boundary, costs 0.6 microseconds.

A screen would need about 20,000 text properties before the `GoString`
conversions took 10 percent of the frame. Flutter cannot lay out 20,000 widgets
in 1 frame, thus the layout reaches the budget long before the boundary does.

2 findings stand against the claim of the file.

1. The encode of the Go side costs more than the decode of the Dart side. A
   text that a Go literal writes pays 60 nanoseconds at each build for 20
   characters, and the read of it pays 16.
2. The read of the optional fields costs more than the text. A cheaper field
   read would gain more than a cheaper `GoString`.

### The cross check

3 numbers of `docs/flutter-design.md` come from a measurement of 2026-09-01 with
another harness. The 3 agree.

| Measure | 2026-09-01 | This harness |
|---|---|---|
| `GoString.fromString` of 13 characters | 47.1 ns | 30 ns at 5, and 60 ns at 20 |
| `GoString.toDartString` of 13 characters | 13.0 ns | 11 ns at 5, and 16 ns at 20 |
| A Dart tree of 200 nodes with a concatenation | 5.61 us | 6.01 us |
| A Go tree of 200 nodes with `strconv.Itoa` | 50.34 us | 49.7 us |

The row "The Dart walk of 200 nodes, with `toDartString` for each node" of that
file reads 2.89 microseconds. This harness reads 7.76 for the same shape,
because it reads the tree the way the generated host reads it: 1 `GoNode` for
each node, and a map lookup of the descriptor for each field.

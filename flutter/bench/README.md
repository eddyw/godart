# bench

This directory measures the cost of a text property at the boundary between a Go
tree and the Dart host. `docs/flutter-first.md` names that cost as unmeasured,
and this directory answers it with a number.

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

The directory answers 1 more claim of `docs/flutter-design.md`: the gain of the
Dart reconciler of A10 over the plain walk of A5. A5 is the host of today, and
it walks the whole Go tree at each rebuild. A10 keeps the tree of the frame
before. It diffs the new tree against that one, and it builds a widget only for
a node that changed.

## Run it again

```sh
make build                 # from the root of the repository, writes build/gart
RUNS=9 flutter/bench/run.sh   # 9 process runs, about 7 minutes
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
such directory. The run opens a window of the application. The last section of
this file holds the numbers.

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
| `go.itoa` | The Go build, with `strconv.Itoa` and a concatenation for each node. The measurement of 2026-09-01 holds this shape. |
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
| `diff.eq.same` | The same, through the generated `==` of the node. Read "The trap of the `==` diff" below. |
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
  Go tree and for a Dart tree. A10 aims at exactly that work, thus the reading
  of A10 below names the number that A10 must beat.
- The Flutter widget construction of the host, for the reason above.
- `setState` and the goroutine that an event handler starts. `watch.one` holds
  the Go graph, the boundary build, and the boundary walk alone.
- A `flutter run` on a device. The measurement runs an ahead of time build of
  the Dart SDK on macOS, and a phone runs slower.

## The result of 2026-09-03

**The host of this section reads each field through a map of the name.** The
last section of this file, "The cheaper field read", holds the host of today
and the numbers of it. The `GoString` numbers of this section still hold,
because the change touched no conversion.

The machine of the project: Apple M4 Max, 16 cores, macOS 27.0, Dart SDK 3.13.2
on macos_arm64. The compiler is `build/gart` of 2026-09-02 17:21.

8 process runs of 15 batches gave 120 batches for each cell. Each run stood at a
load average of 11 or under. 2 invocations of `run.sh` gave 18 runs in all, and
another agent loaded the machine over 10 of them. The report reads the 8 quiet
runs. `out.csv` now holds the later run of the reconciler section, which
reproduced each number of this section inside 2 percent.

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

A byte that holds no ASCII costs much more. 20 runes in 37 bytes decode in 67
nanoseconds, and 20 ASCII characters decode in 16.

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
text work. The last section of this file takes that map out.

### The reading

**The cost of a text property does not matter at 60 hertz.** A dense screen of
200 nodes with a line of 20 characters for each 1 spends 61 microseconds of the
budget. That is 0.37 percent of 16,700. The `GoString` part of that is 18
microseconds, which is 0.11 percent. The normal case, a rebuild of 1 boundary,
costs 0.6 microseconds.

A screen would need about 20,000 text properties before the `GoString`
conversions took 10 percent of the frame. Flutter cannot lay out 20,000 widgets
in 1 frame, thus the layout reaches the budget long before the boundary does.

2 findings stand against the claim of the file.

1. The encode of the Go side costs more than the decode of the Dart side. A
   text that a Go literal writes pays 60 nanoseconds at each build for 20
   characters, and the read of it pays 16.
2. The read of the optional fields costs more than the text. A cheaper field
   read would gain more than a cheaper `GoString`. The last section of this
   file holds that cheaper read, and it measures the gain.

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
file reads 2.89 microseconds. This harness reads 7.76 for the same shape. It
reads the tree the way the generated host reads it. Each node costs 1 `GoNode`,
and each field costs a map lookup of the descriptor.

## The reconciler of A10, 2026-09-03

The same machine and the same harness. 8 process runs of 15 batches gave 120
batches for each cell. Each run stood at a load average of 7 or under. A 9th run
met a load of 69, and it stands out of the data. The middle half of the batches
spreads by 3 percent.

The run measured each case of the text property again. The 2 sets agree inside 2
percent, which is the reproducibility of this harness.

### What the compiler really does

`docs/flutter-design.md` names the obstacle at `internal/emit/iface.go:236`.
That line number is wrong, and the statement behind it is wrong too.

`box` stands at `internal/emit/iface.go:789`, and it holds this rule: a struct
that a name already holds takes `$copy()`, and **a composite literal takes no
copy**. Its own comment says so, and it names a widget tree as the case. A Go
build writes a tree of composite literals, thus the box costs 1 object for each
node and not 2.

The conclusion of the file still holds, for another reason. A build allocates a
fresh `Text$` at each call, thus `identical(old, new)` is false between 2 frames
even when the text is the same. No identity test can stop the walk, and the diff
must read each field of both trees.

### The trap of the `==` diff

The emitter writes an `operator ==` for each node type that Go compares.
`Text$.operator ==` of `flutter/bench/gen` compares `Data` by value. It compares
each of the 12 pointer fields with `identical`, because Go compares a pointer by
its address.

A Go build allocates a fresh pointer for each optional field that it sets.
`ui.P("center")` gives a new `$Ptr` at each call, thus `==` reads a change that
no reader sees. `dart/bin/check.dart` proves it over 100 leaves:

```
plain, nothing changed: the field diff reports 0, and the == diff reports 0
plain, 1 leaf changed: the field diff reports 1, and the == diff reports 1
plain, each leaf changed: the field diff reports 100, and the == diff reports 100
1 optional field set, nothing changed: the field diff reports 0, and the == diff reports 100
1 optional field set, 1 leaf changed: the field diff reports 1, and the == diff reports 100
```

The same trap holds for a function field. `OnPressed` takes a fresh closure at
each build, thus every button reads as changed. The `==` diff is therefore no
reconciler for a real screen. Its numbers below stand only as a floor that no
correct implementation reaches.

### The measurement

Each number is the median of 120 batches, in microseconds. The frame cases hold
the Go build, thus A5 and A10 compare frame against frame.

| Nodes | Text | The diff, 0 changed | The diff, 1 changed | The diff, each changed | A5, the frame | A10, 0 changed | A10, 1 changed | A10, each changed |
|---|---|---|---|---|---|---|---|---|
| 20 | 5 | 9.23 | 8.86 | 1.71 | 5.23 | 9.56 | 9.44 | 6.53 |
| 20 | 20 | 9.18 | 8.83 | 1.91 | 5.32 | 9.52 | 9.44 | 6.74 |
| 20 | 80 | 9.20 | 8.90 | 2.48 | 5.61 | 9.54 | 9.48 | 7.70 |
| 50 | 5 | 22.4 | 22.0 | 3.49 | 12.4 | 23.0 | 22.9 | 15.2 |
| 50 | 20 | 22.4 | 21.9 | 3.88 | 12.7 | 23.0 | 22.8 | 15.9 |
| 50 | 80 | 22.5 | 21.8 | 5.50 | 13.3 | 22.8 | 22.7 | 18.3 |
| 200 | 5 | 87.9 | 87.2 | 11.6 | 48.7 | 90.3 | 90.2 | 57.9 |
| 200 | 20 | 88.2 | 87.7 | 13.5 | 49.3 | 89.6 | 89.6 | 60.1 |
| 200 | 80 | 88.2 | 87.4 | 19.9 | 53.1 | 89.7 | 89.2 | 70.3 |
| 500 | 5 | 216.2 | 216.1 | 26.8 | 121.4 | 224.6 | 221.2 | 142.3 |
| 500 | 20 | 217.0 | 215.3 | 31.8 | 122.6 | 222.0 | 222.8 | 147.5 |
| 500 | 80 | 218.4 | 217.3 | 47.0 | 131.9 | 222.2 | 221.3 | 171.7 |

The `==` diff of 0 changed nodes reads 4.13 microseconds at 200 nodes and 10.1
at 500. It is wrong for each node that sets an optional field or a callback.

### A10 loses every case

**A10 never beats A5.** At 200 nodes with 20 characters:

| The case | A5 | A10 | A10 against A5 |
|---|---|---|---|
| Nothing changed | 49.3 | 89.6 | 1.82 times |
| 1 leaf changed | 49.3 | 89.6 | 1.82 times |
| Each leaf changed | 49.3 | 60.1 | 1.22 times |

The reason is plain. A5 reads 13 fields of 1 tree for each node. A10 reads 13
fields of 2 trees. It reads twice as much before it decides anything, and it
then builds the nodes that changed.

The order of the 3 rows carries the point. **A10 costs most when nothing
changed**, which is the case that A10 exists for. It costs least when everything
changed, because the compare of the first field already answers. The other 12
fields of such a node never run.

The diff of a tree where nothing changed takes 88.2 microseconds at 200 nodes,
which is 0.53 percent of the frame budget. The walk that it replaces takes 47.9.

### The break even that A10 would need

A10 pays 88.2 minus 47.9, thus 40.3 microseconds more than A5 at 200 nodes. That
is 202 nanoseconds for each node. A10 gives that back only if the work it skips
for 1 unchanged node costs more than 202 nanoseconds.

This harness cannot measure that work, because it builds no Flutter widget. The
work is the widget construction and the short cut of `Element.updateChild` at
`widgets/framework.dart:4027`. The descriptor read already costs 240 nanoseconds
for each node. A10 would need each unchanged node to cost over 202 nanoseconds
of Flutter work on top of that. That number is unmeasured, and a measurement of
it needs a Flutter harness.

### The answer that decides the item

**A reconciler has no case left after the signal graph.** The number is 0.588
microseconds against 88.2.

The signal graph of `flutter/ui/signal.go` already rebuilds 1 boundary alone. A
`ui.Watch` around the part that a signal drives costs **0.588 microseconds** for
a frame that changes 1 text, at any tree size. The floor of A10 for a tree of
200 nodes is **88.2 microseconds**. A10 reaches that floor only after Go built
the whole tree again.

A10 is thus **150 times more expensive** than the answer that the project
already ships. The 2 answer the same question: which part of the screen must
rebuild. The signal graph answers it in Go before the tree exists, and A10
answers it in Dart after the tree exists. The second answer costs a full tree
build plus a full 2-tree compare to recover information that the first answer
never lost.

A10 would earn its cost only for a screen that holds no signal and rebuilds the
whole tree at each frame. Such a screen costs 61.8 microseconds today with A5,
which is 0.37 percent of the frame budget, thus it needs no help.

### What an honest A10 must also carry

The reconciler of `dart/lib/diff.dart` is favourable to A10 on 6 counts. Each 1
would make the real number worse.

1. **A key.** `ui.Keyed` gives a Flutter `ValueKey`. A real reconciler matches
   2 children by key and not by index, thus a row that moves keeps its state.
   That needs a map of the key for each list at each frame.
2. **A child list of another length.** This harness marks the whole subtree as
   changed. A real reconciler finds each insert and each removal, which is the
   expensive part of every reconciler. Flutter itself runs a bidirectional sync
   with a key map at `widgets/framework.dart`.
3. **A nested struct.** The `Text` of this harness leaves `Style` nil. A real
   diff walks into a `*TextStyle`, an `Insets`, and a `Shape`. It compares each
   field of those, thus the field count for each node grows.
4. **A function field.** A node holds `OnPressed func()`, and a build allocates
   a fresh closure. No compare of 2 closures gives a useful answer, thus every
   node that holds a callback reads as changed.
5. **The storage of the tree before.** A10 holds the whole previous tree
   between 2 frames. Each node costs 1 `Iface`, 1 node object, 1 `GoString`,
   and 1 `Uint8List`. The live set doubles, and the objects reach the old
   generation of the Dart heap. A5 lets the whole tree die in the nursery. A
   batch of 20 milliseconds does not show that cost, and a long run would.
6. **No decode.** The diff compares the bytes of 2 `GoString` values, and it
   allocates no Dart string. A real host must decode the text of a node that
   changed anyway.

## The cheaper field read, 2026-09-03

The section "Where the time goes" named the read of the optional fields as the
largest single cost of a frame: 37.4 microseconds of the 58.3 that a tree of
200 nodes took. This section holds the change that answers that number.

### What the host did

`GoNode` of `flutter/goflutter/lib/src/node.dart` held 1 `Map<String,
FieldDesc>` for each node kind. A read of 1 field hashed the name of the field,
it probed that map, and it then called the getter closure of the descriptor.
The construction of 1 `GoNode` paid 1 more map lookup, on the name of the type.

A probe over a tree of 200 nodes split the 15.9 nanoseconds of 1 optional field
read. The tree holds 2,395 such reads, and each 1 of them reads a nil pointer.

| The part of 1 read | The cost |
|---|---|
| The lookup of the name in the map | 7.1 ns |
| The call of the getter closure of the descriptor | 5.5 ns |
| The test `v is $Ptr<T>` against a type parameter | 0.6 ns |
| The null test, the bounds test, and the call frames | 2.7 ns |

Half of the cost stood in the map, and no part of that half read a value.

### The change

1. `GoNode` holds `desc.fields`, which is a `List<FieldDesc>` that the
   descriptor carries already. It builds no map and it reads no cache, thus the
   construction of 1 node costs nothing.
2. Each reader takes the position of the field beside the name of it:
   `n.textOpt('TextAlign', 2)`. The read takes the descriptor at that position,
   and it compares the name of it with `identical`. Dart holds 1 object for
   each string literal of a program, thus the compare of 2 pointers answers.
3. A position that misses falls back to a scan of the name over the list. A Go
   struct of another field order thus still reads right, and a caller that
   names no position reads right too.
4. `textOpt`, `sizeOpt`, `countOpt`, and `flagOpt` each name the element type
   of the pointer. A test against `$Ptr<GoString>` costs less than a test
   against `$Ptr<T>` of a type parameter.
5. `nodegen` writes the position. `writeGo` and `writeDart` of
   `flutter/nodegen/lib` read 1 list of parameters for each widget, thus the
   position of a parameter is the position of the Go struct field.
6. `convert.dart`, `host.dart`, and `identity.dart` name the position of each
   field of `flutter/ui/types.go`, `node.go`, and `identity.go` too.

`dart/lib/walk.dart` mirrors the new `GoNode`, thus the measurement below reads
the tree the way the host of today reads it.

### The measurement of the change

The same machine. 3 process runs of each build gave 90 batches for each cell,
and the 2 builds ran in turn, thus a load of the machine hits both. Each number
is the smallest batch, in microseconds.

| Nodes | Text | Optional fields, before | after | Dart walk, before | after | Frame, before | after |
|---|---|---|---|---|---|---|---|
| 20 | 5 | 3.88 | 2.28 | 4.81 | 2.95 | 5.45 | 3.68 |
| 20 | 20 | 3.87 | 2.23 | 4.87 | 2.98 | 6.15 | 4.20 |
| 20 | 80 | 3.91 | 2.21 | 5.24 | 3.30 | 8.76 | 6.76 |
| 50 | 5 | 9.70 | 5.38 | 11.8 | 6.97 | 13.0 | 8.49 |
| 50 | 20 | 9.51 | 5.26 | 11.8 | 7.04 | 14.7 | 9.98 |
| 50 | 80 | 9.70 | 5.43 | 12.9 | 8.03 | 21.4 | 16.6 |
| 200 | 5 | 38.2 | 21.0 | 45.5 | 27.0 | 50.7 | 33.9 |
| 200 | 20 | 38.2 | 20.3 | 46.1 | 27.2 | 56.9 | 40.0 |
| 200 | 80 | 38.0 | 20.8 | 49.7 | 31.1 | 86.0 | 70.1 |
| 200 | 20 runes, 37 bytes | 37.6 | 20.4 | 55.2 | 37.1 | 69.3 | 52.1 |
| 500 | 5 | 94.7 | 54.2 | 112.2 | 69.0 | 125.0 | 82.9 |
| 500 | 20 | 93.5 | 53.5 | 114.6 | 70.8 | 144.5 | 98.6 |
| 500 | 80 | 93.1 | 51.8 | 123.5 | 78.0 | 211.1 | 166.7 |

The read of the optional fields costs 43 to 47 percent less. 1 such read falls
from 15.9 nanoseconds to 8.5. The whole Dart walk costs 37 to 41 percent less,
and the whole frame of 200 nodes with 20 characters falls from 56.9
microseconds to 40.0.

The Go side did not move, as it must not. `go.held` of 200 nodes reads 1.97
microseconds before and 1.99 after, and `go.literal` reads 12.2 and 12.5.

4 more cases of the sweep gain from the same change, at 200 nodes and 20
characters:

| Case | Before | After |
|---|---|---|
| `dart.walk.raw` | 39.8 | 25.9 |
| `frame.a5` | 46.6 | 30.1 |
| `diff.same`, the floor of A10 | 86.8 | 53.1 |
| `watch.three`, 1 boundary of 3 texts | 1.31 | 1.01 |

### What stands after the change

The floor of the read is the getter closure of the descriptor, at 5.5
nanoseconds of the 8.5 that 1 optional field now takes. That closure lives in
the Dart that `gart` writes, and a call of it is an indirect call that the
branch predictor of the processor misses. A read of the Dart field itself costs
under 1 nanosecond, thus a host that named the generated class would beat the
descriptor by another factor of 5. Such a host reads a generated class for each
node type, and the reading of `flutter/nodegen/README.md` gives the size of
that price: 7.2 megabytes of the snapshot.

The guard of the name costs 1.7 nanoseconds of the 8.5. A build that dropped it
would read 6.8, and a Go struct of another field order would then give a wrong
widget with no report. The measurement kept the guard.

## The Flutter work that 1 unchanged node skips, 2026-09-03

The section "The reconciler of A10" left 1 condition open: A10 earns its cost
only when the Flutter work that it skips for each unchanged node exceeds 202
nanoseconds. That harness built no Flutter widget, thus it could not read the
number. `flutter/bench/flutter` reads it now.

**The answer is 645 nanoseconds, thus the condition holds. A10 does beat A5
once the Flutter work stands in the sum. A10 still loses to the signal graph.**

### The harness

`flutter/bench/flutter/run.sh` measures it. `lib/skip.dart` holds a root that
builds a fresh `Column` at each frame. The children of the fresh case are new
`Text` objects, and the children of the reuse case are the objects of the frame
before. `Element.updateChild` of `widgets/framework.dart` ends at its first
test when the new widget is the object that the element already holds, thus the
difference of the 2 cases is the Flutter work of 1 unchanged node.

The loop drives `BuildOwner.buildScope` and not the pump of the tester. A
profile run holds a real binding, thus a pump waits for the next frame of the
screen and that wait of 8.33 milliseconds hides the work. The build scope holds
each part that an unchanged node skips: the widget of the child, the update of
its element, and the write of its render object. The render object reads the
same text in both cases, thus no layout follows either one.

**The run must be a profile build.** A widget test runs in debug mode: the Dart
VM interprets the code, and each assert of the framework runs. The debug number
is 6744 nanoseconds for each node, which is 10.5 times the profile number.
`flutter test` of this package thus proves that the harness runs, and it names
no number that a reader may use.

### The measurement

6 profile runs on the machine of the project, 2000 build scopes for each cell
after a warm up of 200.

| Nodes | Fresh, microseconds | Reuse, microseconds | Skipped, nanoseconds for each node |
|---|---|---|---|
| 50 | 32.8 | 1.1 | 634 |
| 200 | 132.0 | 3.8 | 645 |
| 500 | 347.9 | 8.7 | 679 |

The 3 rows agree, and the fresh column grows with the node count at a constant
rate. The reuse column gives the cost of the identity test alone: **16.8
nanoseconds for each node**.

### A10 against A5, with the Flutter work

The table of "The reconciler of A10" holds the Go build and the Dart walk. The
2 columns below add the Flutter work: A5 gives a fresh widget for each node,
and A10 gives the object of the frame before for each unchanged node.

At 200 nodes and 20 characters, for a frame where nothing changed:

| Part | A5 | A10 |
|---|---|---|
| The Go build and the Dart walk or diff | 49.3 | 89.6 |
| The Flutter work | 129.0 | 3.4 |
| **The frame** | **178.3** | **93.0** |

**A5 costs 1.92 times A10.** The same ratio holds at 20 nodes (1.85) and at 500
nodes (1.93). A10 loses 1 case alone: a frame where each node changed costs
189.1 against 178.3, because A10 pays the diff and gains nothing.

### The reading

The number reverses the A5 comparison of decision 249 and it leaves the
decision itself. 3 points hold.

1. **The condition of decision 249 is met.** 645 nanoseconds stands above the
   202 that A10 must beat, by a factor of 3.2.
2. **A10 beats A5 by about 1.9 times** for a frame where nothing changed.
   Decision 249 read the opposite, because it counted no Flutter work.
3. **A10 still loses to the signal graph by about 76 times.** A `ui.Watch`
   rebuilds 1 boundary of 1 text in 0.588 microseconds of Go work plus 0.645 of
   Flutter work, thus 1.23 in total. A10 reaches 93.0 at 200 nodes, and it
   reaches that floor only after Go built the whole tree again.

The 6 counts of "What an honest A10 must also carry" still stand, and each 1
makes the A10 column worse. The measurement adds a seventh: a host of A10 must
hold the Dart widget of each node between 2 frames and find it again for each
node. This harness holds the widget in a plain list, and a real host needs a
map of the node. That lookup stands in no column above.

# rfwbench

This directory measures the per frame cost of A9, `package:rfw`.
`docs/flutter-design.md` named that cost as the last claim of the matrix with
no number behind it, and this directory answers it.

## The question

A9 writes the user interface as data. The Go side builds a widget library, and
the Dart host decodes the blob and renders it with the `rfw` runtime. A frame
that changes the tree thus pays an encode and a decode of the whole library.

The frame budget at 60 hertz is 16,700 microseconds. `bin/bench.dart` gives the
part of that budget that the pair takes, over a sweep of the node count.

The measure reads `package:rfw/formats.dart`, which holds the wire format
alone and reads no Flutter library. `dart run` thus measures it with no engine.

## Run it again

The machine must stand near idle. Compile ahead of time, because the just in
time run gives a number 3 times larger on the first sweep.

```sh
dart pub get
dart compile exe bin/bench.dart -o bench.exe
./bench.exe > out.csv
```

## The result of 2026-09-04

The tree holds a `Column` of n rows, and each row holds a `Text` inside a
`Padding`. That is the shape of a list of a real application. Each number is
the middle value of 5 batches, and 3 whole runs agreed inside 2 percent.

| Nodes | Bytes | Encode | Decode | Total | The budget of 1 frame |
|---|---|---|---|---|---|
| 10 | 1,752 | 4.8 | 4.6 | 9.4 | 0.06% |
| 50 | 8,272 | 23.0 | 21.2 | 44.2 | 0.26% |
| 100 | 16,422 | 47.0 | 42.0 | 89.0 | 0.53% |
| 200 | 32,822 | 90.0 | 79.9 | 169.8 | 1.02% |
| 500 | 82,022 | 251.1 | 201.9 | 453.0 | 2.71% |
| 1000 | 164,022 | 496.6 | 411.8 | 908.4 | 5.44% |

Each column of a time holds microseconds.

## What the numbers say

**The cost grows with the node count, and the frame budget holds.** 1000 nodes
take 5.44% of 1 frame. A user interface of that size holds each frame at 60
hertz with room to spare.

**A9 costs 20 times the Go tree of A5.** `flutter/bench` measured a Go tree of
200 nodes at 8.66 microseconds. A9 takes 169.8 microseconds for the same node
count. The cause is the shape: A5 passes a live object, and A9 passes 32,822
bytes through an encoder and a decoder.

**The blob is large.** 200 nodes give 32,822 bytes, thus each node costs about
164 bytes. A frame that changes 1 row still writes and reads the whole
library, because the format holds no partial update.

## The lazy list

`_listLookup` of `rfw-1.1.3/lib/src/flutter/runtime.dart:759` walks from index
0 to the target index at each call, and `ListView` of `core_widgets.dart:479`
calls it for each index of the list. A first scroll of n rows is thus O(n²).

`_argsCache` of the same file holds the result by key, thus a second visit to
1 index is O(1). A scroll back up costs nothing more.

The source of the package holds no comment about that cost. The walk itself
shows it, and `docs/flutter-design.md` held the wrong attribution until
2026-09-04.

# The state model of a Flutter application in Go

`docs/flutter-design.md` holds the choice of the binding: a Go tree of plain
structs with a Dart host. This file holds the choice of the state model that
runs on that binding, and the measurements of 2026-09-02 behind it.

The rule that shaped each answer: **the whole state layer must run under
`go test`, with no Flutter, no Dart, and no gart.**

## The ground truth of Flutter, 2026-09-02

Flutter 3.47.0 stable, framework revision `4cf2416426` of 2026-08-11. Dart
3.13.0. DevTools 2.60.0.

### What the Flutter team recommends

The architecture guide of `docs.flutter.dev` names MVVM with 2 layers and 1
option:

| Layer | Parts |
|---|---|
| UI | View, and View Model |
| Data | Repository, and Service |
| Domain, optional | Use case, also named Interactor |

The case study writes the view model as a `ChangeNotifier`, the view as a
`ListenableBuilder`, and each action as a hand written `Command0` or `Command1`
that also extends `ChangeNotifier`. For dependency injection the guide states:
"teams at Google recommend using `package:provider`".

### What the SDK holds, and what it does not

| In the SDK | Position |
|---|---|
| `Listenable`, `ValueListenable` | `foundation/change_notifier.dart:62`, `:94` |
| `ChangeNotifier`, a mixin class | `foundation/change_notifier.dart:139` |
| `ValueNotifier` | `foundation/change_notifier.dart:542` |
| `ListenableBuilder`, `AnimatedBuilder` | `widgets/transitions.dart:1132`, `:1223` |
| `ValueListenableBuilder` | `widgets/value_listenable_builder.dart:71` |
| `InheritedWidget` | `widgets/framework.dart:1855` |
| `StreamBuilder`, `FutureBuilder` | `widgets/async.dart:406`, `:536` |

**`Command` is not an SDK class.** The guide writes it in application code.
**`Signal` is not an SDK class.** No file of the framework declares one.

### What the community uses

| Package | Version | Weekly downloads | Likes | Model |
|---|---|---|---|---|
| `flutter_riverpod` | 3.4.2 | 2.85M | 2.9k | A provider graph with caching |
| `flutter_bloc` | 9.1.1 | 1.79M | 8.07k | A reducer over events, plus Cubit |
| `provider` | 6.1.5+1 | 1.05M | 11k | A wrapper over `InheritedWidget` |
| `signals` | 7.1.0 | 19.6k | 704 | `signal`, `computed`, `effect`, `batch`, `untracked` |

Each 1 of the 4 is third-party. `provider` carries the recommendation of the
guide, and it is not a package of the Flutter team.

## The 2 axes

The question holds 2 independent choices. Each combination is valid.

| Axis | The question | The candidates |
|---|---|---|
| **A. How the state changes** | What does a Go developer write to move the state forward? | Plain mutation through a method. A reducer. A command object. |
| **B. How a change reaches the screen** | What tells the host to rebuild, and at what granularity? | A dirty flag over the whole tree. A listenable for each store. A signal graph. |

Redux and bloc are axis A. Signals and `ChangeNotifier` are axis B. MVVM names a
layout of both, and the Command pattern is axis A.

## The choice

**Axis B takes a signal graph. Axis A stays open, and the package holds all 3
shapes.**

The reason for the signal graph is not speed. It is wall 3 of
`docs/flutter-design.md`: the host builds a fresh Dart widget at each pass, thus
Flutter never takes its identity short cut. A dirty flag then rebuilds the whole
mapped subtree. A signal graph rebuilds only the boundaries that read the value
that changed, and it recovers the exact cost that wall 3 loses.

The graph needs no compiler change. It reads generics, a closure, and a package
level read stack, and each 1 already stands inside the subset.

## The package

The Go module `gart.dev/ui` lives at `flutter/ui`. `flutter/goflutter` holds the
Dart host, and `flutter/example/counter` holds the proof application.

### The signal graph

| Name | Meaning |
|---|---|
| `Signal[T]` | 1 value. `Get` records a dependency, `Peek` does not, `Set` marks each reader, and `Update` reads and writes. |
| `Computed[T]` | A value that a function derives. It runs the function again only after a read that follows a change. |
| `Effect(fn)` | Runs fn, and it runs fn again after a change of a signal that fn read. The result stops the effect. |
| `Batch(fn)` | Holds each notification until fn returns. A boundary that 2 writes touch rebuilds 1 time. |
| `Untracked(fn)` | Runs fn with no dependency record. |

The zero value of a `Signal` is ready, thus `&Counter{}` needs no constructor.

`Set` always marks the readers. It takes no equality test, because Go holds no
equality operation for each type. A `Computed` gives the deduplication when a
program needs it.

### The 3 shapes of axis A

| Shape | Use |
|---|---|
| A method that writes a signal | The plain case. `func (c *Counter) Add() { c.n.Update(...) }` |
| `Store` | The shape of `ChangeNotifier`. Embed it, write a field, and call `Notify`. |
| `Reducer[S, A]` | The shape of Redux and of bloc. The reduce function is pure, thus a table test reads it with no UI. |
| `Command`, `Command1[A]` | The shape of the Command pattern of the Flutter guide. The work runs on a goroutine, and the build reads `Busy` and `Err`. |

### The rule of the 1 goroutine

The graph runs on 1 goroutine. gart puts all Go code in 1 Dart isolate, thus the
rule costs nothing there. A plain Go program keeps the same rule through 2
functions:

- `Post(fn)` gives fn to the goroutine that owns the graph.
- `Pump()` runs each posted function.

A `Command` writes no signal from its worker goroutine. It posts, and the host
pumps. `go test -race` gives a clean run.

## The field decorator

The question: can a Go field assignment notify, like `counter.value = 1` of a
Dart signal?

The answer holds 2 parts.

**A wrapper type works today and needs no compiler change.** `c.n.Set(1)` and
`c.n.Get()` run the same way under `go test` and under gart. Milestone 1 uses
this form.

**A struct tag would give `c.n++`, and it needs stage 5.** Dart holds a real
setter and Go holds none, thus stage 5 can write a getter and a setter pair for
a marked field. The cost is 1 split: under `go test` the value matches and the
notification does not. `FieldDesc` already carries the raw tag at
`internal/emit/iface.go:236`, thus the mechanism exists.

The limit of a setter is the limit of a signal. `c.items[0] = y`, `c.m["k"] = v`,
and `*c.ptr = v` each write through the field, and no setter of any language
sees them. A signal of `package:signals` holds the same limit.

### The copy hazard, and its answer

A copy of a reactive value splits its subscriber list. The Dart output makes the
failure exact: `Signal$.$copy()` calls `_d.$copy()`, which builds a new `dep$`
that holds the subscriber slice of the moment. The 2 lists then drift apart, and
no error names it.

`go vet` gives the whole answer with no compiler work. `Signal`, `Computed`,
`Store`, `Reducer`, `Command`, and `Command1` each hold a `noCopy` field with a
`Lock` and an `Unlock` method. The copylocks check then reports each copy:

```
main.go:19:9: assignment copies lock value to bad: gart.dev/ui.Signal[int] contains gart.dev/ui.noCopy
main.go:22:11: assignment copies lock value to worse: probe.dev/vet.state contains gart.dev/ui.Signal[int] contains gart.dev/ui.noCopy
```

The report follows the value through an enclosing struct.

## The const marker

The question: does a marker for a const value pay, for Flutter or for the
compiler?

**For Flutter: no, and the cause is not the marker.** A Dart const constructor
needs each argument at compile time. A Go string is a `GoString`, and `GoString`
holds a `final Uint8List bytes` at `runtime/goruntime/lib/src/string.dart:17`.
Dart holds no const `Uint8List`, thus no `GoString` is ever const. A change of
the storage type to a const list would cost the typed data of each string
operation. And a tree that Go builds carries its values at run time, thus no
const widget stands at the end of that road. Only the approach that generates a
Dart widget class at build time reaches a const subtree.

**For the compiler: the marker is not needed, because the compiler already
sees it.** The box of an interface value copied the struct at
`internal/emit/iface.go:774`. A composite literal builds a value that no other
expression names, thus the copy is dead. The rule needs no annotation, and a
measure of the counter application gives:

| Copies in the generated build | Before | After |
|---|---|---|
| `flutter/example/counter/gen/lib/app/app.part.dart` | 39 | 2 |

A tree of 200 nodes thus allocates 200 objects in place of 400. The whole test
suite of the compiler, which compares each program of `testdata` against
`go run`, passes with the change.

The general lesson stands: a marker that the **user writes** and the compiler
trusts is a soundness hole, and a marker that the **compiler derives** is sound.
Each further copy of stage 4 needs the same treatment: a derived rule, and not a
tag.

## What milestone 1 proved

6 widget tests pass under Flutter 3.47.0, in
`flutter/example/counter/test/counter_test.dart`:

| Test | What it proves |
|---|---|
| The Go tree builds real Flutter widgets | A `Scaffold`, an `AppBar`, a `TextField`, and a `Switch` come from Go structs. |
| A Go closure drives the count | The event runs Go, and the screen follows. |
| A text field keeps its focus across a Go rebuild | Flutter owns the controller and the focus. |
| A goroutine fills a lazy list of 100,000 rows | A `Command` blocks on a goroutine, and the build never waits. |
| A Go error reaches the screen | The error of the command drives the body. |
| One signal rebuilds one boundary | The count changes, and the greeting keeps its element. |

The Go side runs alone too. 30 tests of `gart.dev/ui` and 5 tests of the
application pass under `go test`, and under `go test -race`.

## The compiler work that milestone 1 needed

The design document said "none". 3 items came up, and each 1 is small.

| Item | Cause | Change |
|---|---|---|
| A dependency module wrote no Dart | `compiled` of `internal/driver/driver.go` accepted the main module and the standard surface only. Stage 2 accepted the import, and stage 5 then failed with "the package gart.dev/ui has no import prefix". | `compiled` now accepts a path whose first element holds a dot, and `libPath` writes it under the directory `dep`. |
| The host had no signal after a goroutine woke | `$schedule` gave no hook, thus a host could only poll. A 16 millisecond poll also breaks `pumpAndSettle` of a widget test. | `$onIdle` runs after the run queue empties, and `$hosted` turns off the deadlock report for a host that starts a goroutine. |
| `flutter analyze` gave 1034 issues in the generated code | The header of a generated file named 8 lints, and a Flutter application analyzes its whole directory. | The header now names 18. The example gives a clean run. |

## What stays open

- The struct tag of a field, and the split that it costs under `go test`.
- A `Signal` inside a slice or a map. The element is not addressable in Go,
  thus a program must hold a pointer.
- The other 5 copy points of stage 4. Only the box of an interface value took
  the derived rule.
- A `Computed` that a program drops must call `Drop`. No test names a leak yet.
- The equality test of `Set`. A `comparable` constraint would give one, and it
  would split the type in 2.

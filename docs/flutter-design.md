# The Flutter design

This file answers how a Go program that gart compiles drives a Flutter user
interface. It weighs 13 approaches, and it takes 1 decision.

This file holds the whole survey. `docs/flutter-options.md` held the option
survey of 2026-08-28, and 2026-09-03 retired that file. Each claim of it that
still stands is below. `docs/flutter-first.md` holds the order of the compiler
work. Read this file before a choice of the Flutter mechanism.

Each claim about the compiler names a file and a symbol, because a line number
rots. Each measurement ran on 2026-09-01, on the machine of the project:
Flutter 3.47.0, Dart SDK 3.13.0, macOS on arm64. The pass of 2026-09-03 ran the
Flutter and the Dart checks again, under Flutter 3.47.2 and Dart 3.13.2. The
section "The measurements" holds each number, and the section "The corrections"
holds each claim that a pass found to be false.

## The question

The user asks for the simplest and the most beautiful expression in Go. The Go
must be idiomatic, it must hold state, and it must give the best developer
experience.
The Go side must read as Go, and not as Dart in Go words. The answer must then
fit Dart and Flutter.

## The 3 constraints

The compiler of 2026-09-01 puts 3 constraints on each answer. No approach
escapes them.

### Constraint 1: a Go type is a final Dart class of its own library

Stage 5 writes 1 class head, `final class X {`, at `internal/emit/decl.go`.
1 Go package maps to 1 Dart library. A `final` class of 1 library accepts no
subclass in another library, and the section "The probes of the language"
proves that rule. Thus a Go type is not a Flutter `Widget` today, and no Dart
code extends a gart class.

The consequence reaches each value that crosses the boundary. A Go struct is
`X$`, and a Go interface value is `Iface?` at `internal/emit/type.go`. A Go
string is `GoString`, and a Go slice is `Slice<T>`. 3 Go types map to
a plain Dart type: `bool`, `float64` to `double`, and each integer type to
`GoInt`. `GoInt` is an extension type over the Dart `int` at
`runtime/goruntime/lib/src/goint_native.dart`, thus it erases on the Flutter
target and costs nothing.

### Constraint 2: a function that can block gives a wrong answer to Dart

The emitted Dart holds no `Future` and no `await`. A function that can block
writes its locals to a frame, sets `$blocked`, and returns the zero value:
`machineSave` of `internal/emit/machine.go`. Only a Go caller tests that field,
and `resume` of the same file reads it.

A probe proved the 2 failures. A Go `Build` method that reads a nil channel
gives `null` to a Dart caller, and `$blocked` holds the frame. A Go `Build`
method that reads a real channel gives the Dart caller a null check error from
`runtime/goruntime/lib/src/chan.dart`, which names no Go position.

A Flutter `build` method must be synchronous, and `StatefulElement.build` calls
`state.build(this)` synchronously at `widgets/framework.dart:5944`. Thus a Go
function in a build position must never block, and the compiler must prove it.
The information already exists: `blockingExports` of
`internal/driver/driver.go` writes the list into the archive. A probe of a
Go type with 2 methods that block gave the archive entry
`["Blocker.Build", "Blocker.Silent"]`.

### Constraint 3: the Go scheduler is a guest of the Dart event loop

`$schedule` at `runtime/goruntime/lib/src/sched.dart` runs the run queue
until it is empty, and it returns. A timer or a future calls `$wake`. Flutter
owns the loop already, and it needs no change.

A Flutter event handler that starts Go work costs 2 lines:
`() { $go(f); $schedule(); }`. `$go` and the finalizer `_cleanups` of
`sched.dart` already hold that pattern. A Go call that waits for a Dart future
reads `$await` of the same file, and that call needs an intrinsic of stage 5.

### What a probe of 2026-09-01 proved

The probe built a Go package with `gart build` and drove it from a Flutter
widget test. It stood outside the repository, and the session deleted it.

| Probe | Result | Compiler change |
|---|---|---|
| A Go tree of plain structs drives real Flutter widgets. A counter holds the state in Go, and a Go closure is the `onPressed` of an `ElevatedButton`. | Passes | None |
| A Go closure of the type `func(i int) Node` drives `ListView.builder`. The list holds 100,000 rows. | Passes. Go counted 43 calls. | None |
| A Flutter `TextField` keeps its text and its focus across a rebuild that Go drives. A Go closure reads each change. | Passes | None |
| A Go `panic` inside a Go build reaches `ErrorWidget.builder`. The error text is `panic: the Go build failed`. | Passes | None |
| A Go build that reads a nil channel gives `null` to Dart, and `$blocked` holds the frame. | Passes | None |
| A Go build that reads a real channel gives a Dart null check error. | Passes | None |
| A bodyless Go function of package A, that package B calls, compiles. Stage 5 writes `$flutter.Text(GoString.fromString('hello'))`, and the body panics. `go build` rejects the same package with "missing function body". | Passes | None |

The probe corrects 1 statement of `docs/flutter-first.md`. `checkBodyless` at
`checkBodyless` of `internal/verify/verify.go` rejects a read of a bodyless
function of the **same** package only, because `bodyless` reads the files of 1
package. A cross-package call already reaches a correct Dart call site. Thus
item 1 of `docs/flutter-first.md` is smaller than the file states.

## What Flutter asks of a language

The 3 constraints above come from gart. This section holds the facts of the
Flutter side. The pass of 2026-09-03 measured each 1 again against Flutter
3.47.2 at `/path/to/flutter`.

### The size of each layer

The count holds each line that is not blank and not a comment.

| Layer | Files | Code lines |
|---|---|---|
| `foundation` | 42 | 5,433 |
| `painting` | 48 | 13,272 |
| `physics` | 7 | 400 |
| `animation` | 8 | 2,232 |
| `gestures` | 27 | 8,028 |
| `scheduler` | 5 | 1,064 |
| `semantics` | 5 | 4,198 |
| `services` | 53 | 14,775 |
| `rendering` | 48 | 31,943 |
| `widgets` | 186 | 81,804 |
| `material` | 198 | 134,343 |
| `cupertino` | 52 | 30,512 |

A widget layer needs 3 of them: `painting`, `rendering`, and `widgets`. Their
sum is 127,019 code lines, and that number is the port cost of A6. The base
under `material` adds `foundation`, `physics`, `animation`, `gestures`,
`scheduler`, `semantics`, and `services`, for 163,149 code lines.

34 percent of the apparent size of `material` is a data table and not logic.
`icons.dart` of `material` holds 11,658 code lines of icon codepoint
constants, and `material/animated_icons/data/` holds 34,066 code lines of path
data in 14 files. Hand written `material` is thus 88,619 code lines.
`icons.dart` of `cupertino` holds 6,372 code lines of the same form, thus hand
written `cupertino` is 24,140 code lines.

A port must generate each table from the upstream source. Go declares no const
class instance, thus a Go icon table needs a code generator of its own. That
cost stands on top of the port cost of A6.

### The barrier of Material is not native code

Material and Cupertino are pure Dart on top of `widgets`. A search for
`SystemChannels.`, `MethodChannel(`, and `dart:ffi` over the 250 files of the 2
layers gives no hit. The whole platform contact is 5 wrapper classes of
`package:flutter/services.dart`: `HapticFeedback`, `SystemChrome`,
`SystemNavigator`, `SystemSound`, and `HardwareKeyboard`.

Material reaches the engine at 3 points only: `FragmentProgram.fromAsset` of
`material/ink_sparkle.dart`, the `Platform` read of `material/about.dart`, and
`PictureRecorder` of `material/color_scheme.dart`.

Thus no native barrier blocks a Go Material. A different barrier blocks it.
Material declares 33 private `RenderObject` subclasses, and Cupertino declares
15. Only A4 and A6 reach such a class from Go, and note 11 of the matrix holds
that row.

### The contract of each base class is small

A Go type that stands as a Flutter widget satisfies a small contract.
`StatelessWidget` declares 1 abstract member, `build`. `StatefulWidget`
declares 1, `createState`. `State` declares `build`, plus `initState`,
`didUpdateWidget`, `didChangeDependencies`, and `dispose`.
`RenderObjectWidget` declares `createRenderObject`. `RenderObject` declares
`performResize` and `performLayout`, and `RenderBox` supplies each 1 except
`performLayout`.

2 rules of `Widget` conflict with the emitter. `Widget` carries `@immutable`,
and each field of a gart class is mutable. Its `operator ==` and its
`hashCode` each carry `@nonVirtual`, and each delegates to object identity.
A2 and A4 must thus suppress `writeEqual` of `internal/emit/decl.go`. The
section "The probes of the language" measures the 3 warnings that such a class
gives.

## The feature matrix

The mark holds 3 values: `F` for full, `P` for partial, and `N` for none. Each
mark that needs a note carries 1, and "The notes of the matrix" holds the text.

| Code | Approach |
|---|---|
| A1 | A hand written binding through `syscall/dart` |
| A2 | A struct tag that makes a Go struct a Dart widget class |
| A3 | A generator that reads the Go AST and writes Dart widget classes |
| A4 | An emitter mode that gives a Go type a Dart superclass |
| A5 | A Go tree with a hand written Dart host |
| A6 | A full port of the widgets layer to Go |
| A7 | A generator that reads the Flutter API and writes the Go binding |
| A8 | A shadow package set that only gart builds |
| A9 | Remote Flutter Widgets, `package:rfw` |
| A10 | A Go tree with a Dart reconciler |
| A11 | An immediate mode user interface on a canvas |
| A12 | Go in a second isolate |
| A13 | A Go tree with a generated Dart host |

| Feature | A1 | A2 | A3 | A4 | A5 | A6 | A7 | A8 | A9 | A10 | A11 | A12 | A13 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| A stateless widget | F | F | F | F | F | F | F | F | F | F | N | N | F |
| A stateful widget with local state | F | P | F | F | F | F | F | F | P | F | F | N | F |
| A key | F | F | F | F | F | F | F | F | N | F | N | N | F |
| `ListView.builder` with a lazy list | P | F | F | F | F | F | P | P | P | F | P | N | F |
| A theme and `MediaQuery` | P | P | F | F | P | P | P | P | P | P | N | N | P |
| A gesture | F | F | F | F | F | F | F | F | P | F | P | N | F |
| A 60 Hz animation with `AnimationController` | P | P | F | F | P | F | P | P | P | P | F | N | P |
| Text input and focus | P | P | F | F | F | N | P | P | N | F | N | N | F |
| Navigation and a route | P | F | F | F | P | P | P | P | N | P | N | N | P |
| A `pub.dev` package | P | P | F | P | P | N | F | F | N | P | N | N | F |
| A custom `RenderObject` | N | N | P | F | P | F | N | P | N | P | F | N | P |
| Hot reload | P | P | P | P | P | P | P | P | F | P | P | P | P |
| The widget inspector and the Dart devtools | P | P | P | P | P | N | P | P | P | P | N | P | P |
| A `const` subtree and its rebuild cost | P | N | F | N | N | N | P | P | N | P | N | N | N |
| The app size | F | F | F | F | F | N | P | F | P | F | P | P | P |
| A platform plugin and a method channel | P | P | F | P | P | P | F | F | P | P | P | P | P |
| Semantics and accessibility | P | F | F | F | F | N | F | F | N | F | N | N | F |
| A platform view | F | F | F | F | F | N | F | F | N | F | N | N | F |
| An image and an asset | F | F | F | F | F | P | F | F | F | F | P | P | F |
| An error boundary | F | F | F | F | F | N | F | F | F | F | N | P | F |

### The notes of the matrix

1. **A12 answers a different question.** A second isolate is a deployment axis,
   and not a shape of a user interface. The user interface isolate still needs
   1 of A1 to A11. Each `N` of the A12 column means that A12 alone gives the
   feature no answer. Read the section of A12.
2. **A lazy list.** A5 and A13 give the mark from a real run. The Go closure
   ran 43 times for a list of 100,000 rows. A1, A7, and A8 need the rule of
   constraint 2 for a callback with a result, thus the mark is `P` today. A9
   holds a lazy `ListView.builder` too, and `_listLookup` of `rfw` walks from
   index 0 at each call, thus a first scroll is O(N²). A second visit to 1
   index reads a cache, thus a scroll back up is O(1) for each row.
3. **A theme.** No approach outside A3 and A4 lets Go call `Theme.of(context)`,
   because a Go value holds no `BuildContext`. The host reads the theme and
   passes a snapshot to the Go build. That gives the full behavior with a
   different shape, thus the mark is `P`.
4. **A 60 Hz animation.** The host owns the `AnimationController` and passes
   the value to the Go build at each frame. `TickerProvider` at
   `scheduler/ticker.dart:43` holds 1 method, and a Go type cannot implement a
   Dart interface today, thus Go cannot own the controller. The measured cost
   of a Go tree of 200 nodes is 8.66 microseconds, thus the frame budget holds.
   A9 pays 169.8 microseconds for the same node count, and that budget holds
   too. Read `flutter/rfwbench`.
5. **Text input.** A5 and A13 give the mark from a real run. The Dart host
   builds a real `TextField`, thus Flutter keeps the text, the controller, and
   the focus across a rebuild that Go drives. A9 holds no text input widget of
   any kind.
6. **Navigation.** The host owns the `Navigator`. A13 holds the stack of the
   pages in Go since 2026-09-05: `ui.Pages` carries 1 key and 1 child for each
   page, the host builds 1 `MaterialPage` for each child, and a back gesture of
   the platform reaches `OnPop`. The earlier Compass example demonstrated
   this model with 7 screens and Material page transitions. Commit `f2652a3`
   retains its source. A Go call that waits for the
   result of a pop still needs the `$await` intrinsic, because
   `Navigator.push` gives a `Future<T?>` at `widgets/navigator.dart:5060`. A
   named route and a URL of the web also stay out, thus the mark stays `P`.
7. **A `const` subtree.** A5, A9, A10, and A13 build a fresh Dart widget at
   each pass. Thus `Element.updateChild` at `widgets/framework.dart:4027` never
   takes its identity short cut. A2 and A4 cannot write a const constructor,
   because a Go string constant emits `GoString.fromString`, and that call is
   not const. A3 writes a Dart class that holds a Dart `String`, thus only A3
   reaches a real const widget.
8. **The app size.** A measurement of an ahead of time build gave 64 kilobytes
   for `goruntime` and a small Go package. A6 replaces 127,019 code lines of
   the Flutter base with Go, and a Go named type costs a type descriptor pair.
   A measurement of 5 named types gave 6,353 bytes of descriptor against 7,676
   bytes of code. A13 carries 1 descriptor pair for each generated node type,
   and the Dart tree shaker drops each descriptor that no code reads. The
   milestone 3 section holds the measurement of 2026-09-02.
9. **Hot reload.** Each approach that emits Dart from Go needs a `gart watch`
   pass first, and then the Flutter reload takes the new Dart. This chain is
   unverified. A9 needs no reload at all, because the widget tree is data.
10. **The inspector.** The inspector shows the Dart widget tree. For A5, A10,
    and A13 it names the host file, and not the Go file. gart writes a source
    map of its own, and the Dart devtools do not read it.
11. **A custom `RenderObject`.** A4 and A6 are the only 2 that give Go a real
    `RenderObject`. A5, A10, and A13 reach a `CustomPaint` node whose painter
    reads a command list that Go writes. A11 is that model for the whole
    screen.
12. **The error boundary.** A Go panic throws `GoPanic`, a Dart exception.
    `ComponentElement.performRebuild` at `widgets/framework.dart:5823` catches
    it and calls `ErrorWidget.builder` of the same file. A probe proved the path. A6
    holds no `ComponentElement`, thus it needs an error boundary of its own.

## The approaches

### A1: A hand written binding through `syscall/dart`

**The mechanism.** A Go package holds 1 struct for each Flutter widget, and
each struct gives a Dart handle through a call of the interop package. A hand
written Dart host maps a property bag to a Flutter constructor.

```go
type Counter struct{ n int }

func (c *Counter) node() dart.Object {
	return dart.New("Column", dart.List("children",
		dart.New("Text", dart.Str("data", fmt.Sprintf("count %d", c.n))),
		dart.New("Button", dart.Str("label", "add"),
			dart.Fn("tap", func() { c.n++; dart.Dirty() }))))
}
```

**The state model.** The state is a plain Go struct. Go rebuilds the handle
tree at each pass, and the host holds no Go value.

**The compiler work.** `intrinsic` of `internal/emit/call.go` is a flat switch
on the name of an interop function, and `intrinsicText` holds 1 case for each
Go stub of `internal/overlay/std/syscall/dart/dart.go`. The 2 counts were 132
on 2026-08-28, and each 1 reads 174 on 2026-09-03.

A handle needs a named Go type whose Dart type is `Object?`. That was the whole
cost of this row, and 2026-09-03 paid it: `//gart:dart` gives an external Dart
declaration, and `dart.Object` of `syscall/dart` gives the type. Read decision
239 and decision 240. `unsafe.Pointer` was the only such type before that day,
and stage 2 still rejects `unsafe`. A callback with a result needs the rule of
constraint 2.

**The runtime cost.** Each property crosses as a string key. A text property
pays 47.1 nanoseconds for `GoString.fromString`.

Pros:

- The compiler work is small, and each new widget costs 1 Go struct.
- The Go side holds no Dart class name in a type position.

Cons:

- Each property crosses with no type check.
- Flet gives the price of such a host: 40,259 lines of hand written Dart.
- The Go code reads as a call sequence, and not as a tree.

### A2: A struct tag that makes a Go struct a Dart widget class

**The mechanism.** A tag on a blank marker field declares the struct a widget,
and stage 5 writes the whole Dart widget class from it.

```go
type Counter struct {
	_ struct{} `dart:"widget=StatefulWidget"`
}

type CounterState struct {
	_ struct{} `dart:"state=Counter"`
	n int
}

func (s *CounterState) Build(ctx dart.Context) dart.Node {
	return &Column{Children: []dart.Node{
		&Text{Data: fmt.Sprintf("count %d", s.n)},
		&Button{Label: "add", Tap: func() { s.SetState(func() { s.n++ }) }},
	}}
}
```

**The state model.** A widget struct must hold a final field for the const
constructor, thus the state moves to a second Go type. The Go source then holds
2 types for 1 screen, exactly as Dart does.

**The compiler work.** The largest of the 13. Stage 5 needs a second class
shape. `structClass` of `decl.go` writes the head, a final field, and a const
named constructor with `required` and a default. `writeEqual` of the same file
drops the equality operations, and `method` writes `@override` and the Dart
member name. `paramList` and the composite literal of `internal/emit/expr.go`
each need a named argument. Stage 2 needs 3 rules, and stage 3 needs the root
rule of constraint 2.

**The runtime cost.** The lowest of the 13. The Go widget is the Dart widget,
thus no tree crosses a boundary.

Pros:

- The Go type is a real Flutter `Widget`. Dart code and a `pub.dev` package
  each accept it.
- A key, semantics, and a route each work with no host.

Cons:

- A blank field with a tag is not idiomatic Go, and the tag grammar is a second
  language.
- The widget struct loses `$assign` and the value semantics of decision 2.
- A Go string constant blocks the const constructor.
- `Widget.operator ==` carries `@nonVirtual` at `widgets/framework.dart:365`,
  thus `writeEqual` of `decl.go` must not run for such a class.

### A3: A generator that reads the Go AST and writes Dart widget classes

**The mechanism.** A second tool reads the Go AST and writes 1 Dart widget
class for each marked Go type. gart itself does not change.

```go
//gart:widget stateful
type Counter struct {
	n int
}

func (c *Counter) Build(ctx dart.Context) dart.Node {
	return &Column{Children: []dart.Node{
		&Text{Data: fmt.Sprintf("count %d", c.n)},
		&Button{Label: "add", Tap: func() { c.n++; ctx.Dirty() }},
	}}
}
```

**The state model.** The generated Dart `State` holds the Go state object, and
it calls `setState` after each Go event.

**The compiler work.** None in the 6 stages. The driver at
`internal/driver/driver.go` must write the extra file into the output
package. `go/packages` parses each comment at `packages.go:774`, thus a
directive needs no loader change.

**The runtime cost.** 2 hops for each widget: Dart to Go and Go to Dart.

Pros:

- The Go side is plain Go with 1 comment directive.
- Only this approach reaches a real const Dart widget, because the generated
  class holds a Dart `String`.
- jaspr proves the form: 12,072 generated lines for a tag library.

Cons:

- A fix of the Dart output needs a change of the generator.
- The generator must track each Flutter release.
- The developer reads a Dart stack trace of generated code.

### A4: An emitter mode that gives a Go type a Dart superclass

**The mechanism.** An embedded Go mirror type names the Dart superclass, and
stage 5 writes `extends` and `@override`.

```go
type Counter struct {
	widgets.StatefulWidget
}

type CounterState struct {
	widgets.State

	n int
}

func (s *CounterState) Build(ctx widgets.BuildContext) widgets.Widget {
	return &widgets.Column{Children: []widgets.Widget{
		&widgets.Text{Data: fmt.Sprintf("count %d", s.n)},
		&material.Button{Tap: func() { s.SetState(func() { s.n++ }) }},
	}}
}

// The value of A4 stands 1 level lower. Only this approach reaches a
// RenderBox, and Material holds 25 private subclasses of one.
type Gauge struct {
	rendering.RenderBox

	Value float64
}

func (g *Gauge) PerformLayout() { g.SetSize(g.Constraints().Biggest()) }
```

**The state model.** The Go type is the Dart `State` or the Dart
`RenderObject`, thus the state model is the Flutter model.

**The compiler work.** `structClass` of `decl.go` writes the superclass, and
`method` writes
`@override` and the Dart member name. `internal/name/name.go` needs a map from
a Go method name to a Dart member name. `internal/emit/emit.go` needs an
import of a Dart package URI. A super call has no Go form, thus `dart.Super`
becomes a new intrinsic. The marker must select 1 of 3 Dart relations:
`extends`, `implements`, or `with`.

**The runtime cost.** The same as A2.

Pros:

- This is the only approach that reaches a custom `RenderBox` from Go.
- Go embedding is the natural Go form of a supertype.

Cons:

- Go holds 1 embedding form, and Dart holds 3 relations.
- The framework holds 108 mixins, and 38 of them call `super`. Go embedding
  gives method promotion and no super chain.
- Each inherited member needs a Go mirror declaration, thus A4 needs A7 under
  it.

### A5: A Go tree with a hand written Dart host

**The mechanism.** Go declares an immutable tree of plain Go structs. 1 Dart
host reads the tree and builds real Flutter widgets from it. The Go tree
crosses as an `Iface` pair, and the host reads `Iface.desc.name` at
`runtime/goruntime/lib/src/iface.dart`.

```go
type Counter struct{ n int }

func (c *Counter) Build(ctx ui.Ctx) ui.Node {
	return ui.Column{Children: []ui.Node{
		ui.Text{Data: fmt.Sprintf("count %d", c.n)},
		ui.Button{Label: "add", Tap: func() { c.n++; ctx.Dirty() }},
	}}
}
```

The host holds 1 case for each node kind:

```dart
case 'example.com/app/ui.Button':
  final go.Button$ b = n.value! as go.Button$;
  return ElevatedButton(
    onPressed: () { setState(() { $go(b.Tap!); $schedule(); }); },
    child: Text(b.Label.toDartString()));
```

**The state model.** The state is a plain Go struct with methods. The probe
holds this exact shape, and it works.

**The compiler work.** None, when the Go application is a library package. A
`package main` holds no library URI at `internal/driver/driver.go`, and
`writeMain` at `internal/emit/emit.go` writes `bin/main.dart`. Thus
`package main` needs item 4 of `docs/flutter-first.md`, at a small cost.

**The runtime cost.** Measured. A Go tree of 200 nodes with no text work costs
8.66 microseconds. The Dart walk of the same tree costs 2.89 microseconds. The
frame budget at 60 hertz is 16,700 microseconds.

Pros:

- It works today. 4 Flutter tests pass with no change of the compiler.
- The Go side is pure Go: no tag, no directive, and no Dart name.
- The host builds real Flutter widgets, thus Flutter keeps the element tree,
  the keys, the `State` of each Flutter widget, the focus, and the error
  boundary.
- The Go tree is data, thus a Go test of the tree needs no Flutter.

Cons:

- The host switch is hand written, and it grows with the node set.
- A Go value holds no `BuildContext`, thus the host must pass a theme snapshot.
- No const subtree, thus the mapped subtree rebuilds each pass.

### A6: A full port of the widgets layer to Go

**The mechanism.** Go owns the whole hierarchy. `painting`, `rendering`, and
`widgets` become Go packages, and the only Dart contact is the `dart:ui` handle
set.

```go
type Counter struct {
	widgets.Base

	n int
}

func (c *Counter) Build(ctx widgets.BuildContext) widgets.Widget {
	return &widgets.Column{Children: []widgets.Widget{
		&widgets.Text{Data: fmt.Sprintf("count %d", c.n)},
		&material.Button{Tap: func() { c.n++; c.MarkNeedsBuild() }},
	}}
}

func main() { widgets.RunApp(&Counter{}) }
```

**The state model.** The Go model, with a Go `Element` tree and a Go
`setState`.

**The compiler work.** Small for the emitter. About 95 intrinsics for the
`dart:ui` names, 12 of them opaque handles, plus the rule of constraint 2 for
the frame path.

**The runtime cost.** Go owns each frame. The Go tree cost of 43 nanoseconds
for each node then applies to the whole element tree and the whole render tree.

Pros:

- Each API is a Go interface, thus the whole stack reads as Go.
- Go reaches a custom `RenderObject` with no compiler change.

Cons:

- 127,019 code lines of base, plus 88,619 for Material and 24,140 for
  Cupertino.
- No `pub.dev` package works, and the inspector shows 1 leaf.
- Text input needs a port of `EditableText`, of the IME, and of the platform
  text input channel. That work alone is out of proportion.

### A7: A generator that reads the Flutter API and writes the Go binding

**The mechanism.** A Dart tool reads the public API of Flutter with
`package:analyzer`, and it writes the Go binding package: 1 Go struct for each
widget, 1 Go field for each named parameter, and 1 Go function for each named
constructor.

```go
// Generated by gartbind. Do not edit.
type Text struct {
	Data     string
	Style    *TextStyle
	MaxLines *int
	Overflow *TextOverflow
}
```

The counter then reads the generated names, and the shape follows the host that
consumes the binding. Under A13 it reads:

```go
func (c *Counter) Build(ctx ui.Ctx) ui.Node {
	return ui.Column{Children: []ui.Node{
		ui.Text{Data: fmt.Sprintf("count %d", c.n)},
		ui.ElevatedButton{
			OnPressed: func() { c.n++; ctx.Dirty() },
			Child:     ui.Text{Data: "add"},
		},
	}}
}
```

**The state model.** A7 writes no state model. It is a supplier of type
declarations for A1, A4, A8, A10, or A13.

**The compiler work.** None. The tool runs outside the compiler.

**The runtime cost.** None of its own.

Pros:

- It scales to the whole surface: about 1,650 public types, 1,671
  constructors, and 8,547 public fields.
- It re-runs for each Flutter release, and a diff of the output names each
  break.
- A `pub.dev` package is 1 more input.

Cons:

- A Go zero value carries no "not set" state, and Material reads `??` 3,744
  times. Each optional field needs a pointer, a bitmask, or a wrapper type.
- A Dart generic, a callback with a result, and a `BuildContext` each need a
  hand written rule.
- The generated Go package carries a type descriptor pair for each named type.

### A8: A shadow package set that only gart builds

**The mechanism.** A Go package declares Flutter-like functions and types, and
a marker names the Dart library and the Dart name. Stage 5 emits a direct call
of the Dart target.

```go
package flutter

//gart:dart package:flutter/material.dart Text
func Text(data string, opt ...TextOption) Widget { panic("gart only") }
```

The counter then calls the Dart names directly:

```go
func (c *Counter) build(ctx flutter.Context) flutter.Widget {
	return flutter.Column(
		flutter.Text(fmt.Sprintf("count %d", c.n)),
		flutter.ElevatedButton(
			flutter.OnPressed(func() { c.n++; flutter.Rebuild(ctx) }),
			flutter.Child(flutter.Text("add"))))
}
```

**The honesty cost, measured.** A probe compiled such a package 2 ways. With no
body, `go build` and `go vet` each reject the package with "missing function
body", and `gopls` reads the same error. With the stub body `panic("gart
only")`, `go build` accepts the package, and `gart` compiles the panic. Thus
the declarations must carry a stub body, and the marker must replace it. That
choice costs 1 line for each declaration, and it keeps the editor honest.

The probe also proved that gart already writes the correct call site for a
cross-package read of a bodyless function:
`$flutter.Text(GoString.fromString('hello'))`. Only the body needs the marker.

**The state model.** The state is a plain Go struct. The widget tree is a call
sequence over opaque handles.

**The compiler work.** Item 1 and item 2 of `docs/flutter-first.md`. A marker
that stage 5 reads, an import of a Dart URI at `internal/emit/emit.go`, a
handle type at `internal/emit/type.go`, and the value boundary of item 5.

**The runtime cost.** 1 Dart call for each Go call, with the conversion of each
argument.

Pros:

- The long tail costs no compiler change and no Dart edit. 1 Go declaration
  reaches 1 Dart name.
- It is the general escape hatch under each other approach.
- The generator of A7 writes the declarations.

Cons:

- The Go program reads as Dart in Go words.
- A handle must not be copied, and a Go struct copies at 5 points. Stage 2
  needs a rule.
- A Go type cannot implement a Dart interface, thus `TickerProvider`,
  `ScrollController`, and each delegate stay in Dart.

### A9: Remote Flutter Widgets, `package:rfw`

**The mechanism.** Go writes a widget description as data. A Dart host decodes
the blob and renders it with the first-party `rfw` runtime. No Dart code
generation runs.

```go
// Go writes the description. The host owns the count, because rfw holds no
// arithmetic and cannot write state.n = state.n + 1.
func counter(n int) rfw.Library {
	return rfw.Library{Widgets: []rfw.Widget{{
		Name: "Counter",
		Root: rfw.Call("Column", rfw.Map{"children": rfw.List(
			rfw.Call("Text", rfw.Map{"text": fmt.Sprintf("count %d", n)}),
			rfw.Call("ElevatedButton", rfw.Map{
				"onPressed": rfw.Event("add", nil),
				"child":     rfw.Call("Text", rfw.Map{"text": "add"}),
			}),
		)}),
	}}}
}
```

**The state model.** `rfw` holds state, and the state is set-only. A
`SetStateHandler` writes a literal or a reference. The format holds no
arithmetic and no comparison, thus `state.n = state.n + 1` has no form. The
counter example of the package increments in the host, and not in `rfw`.

**The compiler work.** Very small. The blob is a byte slice, thus Go writes it
with the packages that the surface already holds.

**The runtime cost.** Measured on 2026-09-04. `flutter/rfwbench` holds the
harness and the whole table.

| Nodes | Bytes | Encode | Decode | Total | The budget of 1 frame |
|---|---|---|---|---|---|
| 100 | 16,422 | 47.0 | 42.0 | 89.0 | 0.53% |
| 200 | 32,822 | 90.0 | 79.9 | 169.8 | 1.02% |
| 1000 | 164,022 | 496.6 | 411.8 | 908.4 | 5.44% |

Each column of a time holds microseconds, and the budget of 1 frame at 60
hertz is 16,700 of them. The frame budget holds, thus A9 renders at 60 hertz.

A9 costs 20 times the Go tree of A5, which `flutter/bench` measured at 8.66
microseconds for 200 nodes. The cause is the shape: A5 passes a live object,
and A9 passes 32,822 bytes through an encoder and a decoder. A frame that
changes 1 row writes and reads the whole library, because the format holds no
partial update.

The binary blob decodes in 1 pass. The text form is about 10 times slower, and
the package discourages it on a mobile target. `_listLookup` of the package
walks from index 0 to the target index at each call, thus a first scroll of a
lazy list is O(N²). `_argsCache` holds each result by key, thus a scroll back
up costs nothing more.

Pros:

- It is first-party, maintained, and version 1.1.3 of 2026-03-25.
- The user interface updates with no rebuild of the application.
- The lazy list is real: `ListView` maps to `ListView.builder`.

Cons:

- The package README states the ceiling: "Flutter proper lets you create
  widgets for compelling UIs with gestures and animations and so forth. With
  RFW you can use those widgets, but it doesn't let you *create* those
  widgets."
- No text input widget of any kind ships, thus a form needs a hand written
  local widget library.
- No widget accepts a key, thus a list item holds no identity across a
  reorder.
- 61 widgets ship in total: 39 core and 22 Material. No Cupertino library
  ships.
- No const subtree, and no page transition, no drag and drop, and no custom
  painter.

### A10: A Go tree with a Dart reconciler

**The mechanism.** Go builds an immutable tree of plain Go structs at each
frame. A Dart reconciler diffs the new tree against the tree of the frame
before, and it drives real Flutter widgets from the difference.

The Go side is identical to A5:

```go
func (c *Counter) Build(ctx ui.Ctx) ui.Node {
	return ui.Column{Children: []ui.Node{
		ui.Text{Data: fmt.Sprintf("count %d", c.n)},
		ui.Button{Label: "add", Tap: func() { c.n++; ctx.Dirty() }},
	}}
}
```

Only the host differs. It keeps the tree of the frame before, and it rebuilds
only the Flutter widget of a node that changed.

**The state model.** The same as A5.

**The compiler work.** The same as A5: none.

**The runtime cost.** The diff costs 1 more walk of the old tree. It saves the
rebuild of a subtree whose Go nodes did not change, thus it recovers part of
the const skip that A5 loses.

Pros:

- It keeps each property of A5, and it adds a subtree skip.
- jaspr proves that a Flutter-shaped reconciler is 4,086 code lines in 17
  files.

Cons:

- The host holds a second tree, thus the memory cost doubles.
- A build allocates a fresh node object for each node, thus `identical(old,
  new)` is false between 2 frames even when each field matches. No identity
  test can stop the walk, and the diff must compare each field.

  The earlier text named a copy of a box at `internal/emit/iface.go` as the
  cause. 2 parts of that are wrong. `box` stands at `internal/emit/iface.go`,
  and line 236 of that file wrote the descriptor of a map. The rule is also
  the opposite: a
  composite literal takes no copy, and the comment of that function names a
  widget tree as exactly the case that it holds at 1 object for each node.
- **Measured 2026-09-03. A10 loses in each case, and it has no case left.**
  `flutter/bench` holds the reconciler and the harness. Each number is a median
  of 120 batches at 200 nodes with a text of 20 characters.

  | Case | Microseconds | Against A5 |
  |---|---|---|
  | A5, the whole frame | 49.3 | 1.00 |
  | A10, nothing changed | 89.6 | 1.82 |
  | A10, 1 leaf changed | 89.6 | 1.82 |
  | A10, each leaf changed | 60.1 | 1.22 |
  | 1 boundary of the signal graph | 0.588 | 0.012 |

  The order of those rows is the answer. A10 costs the most when nothing
  changed, which is the case that it exists for, and the least when everything
  changed, because the first field that differs ends the compare. A5 reads 13
  fields of 1 tree for each node, and A10 reads 13 fields of 2 trees before it
  decides anything.

  The signal graph settles it. A `ui.Watch` rebuilds 1 boundary in 0.588
  microseconds at any tree size, and the floor of A10 is 88.2. A10 answers the
  same question 150 times more expensively, and it answers it in Dart after Go
  rebuilt the whole tree. The signal graph answers it in Go before the tree
  exists.

  A10 would need the Flutter work that it skips for each unchanged node to
  exceed 202 nanoseconds, which is its own cost for each node.
  `flutter/bench/flutter` measured that work on 2026-09-03: **645
  nanoseconds**, thus the condition holds by a factor of 3.2. A10 beats A5 by
  1.92 times for a frame where nothing changed, and the signal graph still
  beats A10 by 76 times. Read decision 253.

  The generated `==` cannot serve as the compare. It reads each pointer field
  with `identical`, which is correct for Go, and a build allocates a fresh
  pointer for each optional field that it sets. `ui.P("center")` thus reports a
  change that nobody can see, and a fresh closure makes each button report one.
  `flutter/bench/dart/bin/check.dart` proves it over 100 leaves.

### A11: An immediate mode user interface

**The mechanism.** Flutter gives a `CustomPaint` surface and the raw pointer
events. Go owns the layout, the hit test, and the paint. gio proves the model
in Go.

```go
type Counter struct {
	n   int
	btn ui.Click
}

func (c *Counter) Layout(gtx ui.Ctx) ui.Dim {
	if c.btn.Clicked(gtx) {
		c.n++
	}
	return ui.Column(gtx,
		ui.Label(gtx, fmt.Sprintf("count %d", c.n)),
		ui.Button(gtx, &c.btn, "add"))
}
```

**The state model.** Go owns each part. The state is a plain Go struct, and
the loop reads it at each frame.

**The compiler work.** About 20 intrinsics for the `Canvas`, the `Paint`, the
`Path`, and the `Paragraph` of `dart:ui`. Each one stays an opaque handle.

**The runtime cost.** Go repaints the whole surface at each frame. The frame
path holds no Flutter element tree, thus no rebuild cost exists.

Pros:

- The most Go of the 13. gio proves the shape with 2,237 stars.
- A 60 hertz animation is free, because Go owns the frame.
- No key, no element reuse, and no equality.

Cons:

- The whole loss list: no Material, no Cupertino, no text input, no IME, no
  accessibility, no platform view, no inspector, and no `pub.dev` package.
- The platform look is gone. Go draws every pixel.
- Text layout needs `Paragraph` of `dart:ui`, and the rules of a script that
  the developer does not read.

### A12: Go in a second isolate

**The mechanism.** The Go program runs in a second Dart isolate. The user
interface isolate holds the Flutter widgets, and a message protocol carries the
state and the events between the 2.

```go
// The Go side of the counter. The user interface isolate holds the widgets.
func main() {
	events, updates := isolate.Channels()
	n := 0
	for range events {
		n++
		updates <- Counter{N: n}
	}
}
```

**The state model.** Go owns the state, and each read costs a message and a
copy.

**The compiler work.** Small. `Isolate.run` already stands behind the worker of
`docs/design.md`.

**The runtime cost.** 2 isolates share no memory. Each message copies its
payload. A per-frame message costs a copy of the whole tree.

Pros:

- A long Go computation does not stall the frame.
- Each isolate holds its own scheduler, and the model of `docs/design.md`
  already states that rule.

Cons:

- It answers a different question. The user interface isolate still needs 1 of
  A1 to A11, thus A12 adds a cost and gives no widget.
- A `build` method cannot read the Go state, because a message is
  asynchronous.
- 2 isolates hold 2 copies of the program, thus the binary grows.

### A13: A Go tree with a generated Dart host

**The mechanism.** A5, with a generator for the node set and for the host. A
tool reads the public API of Flutter with `package:analyzer`, as A7 does. It
writes 2 files: the Go node structs, and the Dart host case for each 1. The
developer writes plain Go and reads no Dart.

```go
package app

import "gart.dev/ui"

// Counter holds the state of the screen. It is a plain Go struct.
type Counter struct {
	n int
}

// Build gives the widget tree of the current state.
func (c *Counter) Build(ctx ui.Ctx) ui.Node {
	return ui.Column(
		ui.Text("count %d", c.n),
		ui.Button("add", func() { c.n++; ctx.Dirty() }),
	)
}

func Root() ui.Node { return ui.Root(&Counter{}) }
```

A generated Go constructor keeps the tree short. `ui.Column` is a variadic
function that gives a `Column` node, and `ui.Text` takes a format. A named
option covers each other parameter: `ui.Text("hi", ui.Size(24))`. Go 1.27 does
not elide the type of an element of a slice literal, thus the function form
reads better than the struct literal form.

**The state model.** A plain Go struct with methods. `ctx.Dirty` marks the tree
for a rebuild, and the host calls `setState`.

**The compiler work.** None in the 6 stages for milestone 1. Milestone 2 adds
the rule of constraint 2, which is a filter over the list that the archive
already holds.

**The runtime cost.** The same as A5: 8.66 microseconds for a Go tree of 200
nodes, and 2.89 microseconds for the Dart walk. The generated node set adds a
type descriptor pair for each node type. The measure of 2026-09-05 reads the
Dart output of `gart.dev/ui` in `flutter/example/compass`: the library holds
1,911,012 bytes, the descriptions of the types hold 1,849,946 of them, and the
set holds 564 node types. Each node type thus costs about 3,280 bytes of Dart
source. The Dart tree shaker drops an unused descriptor, thus an ahead of time
build carries the types of the application alone.

**The pack split of 2026-09-05.** The node set stands in 3 Go packages now, and
an application compiles the packs that it names alone. compass names the core
and Material, thus the Dart of `gart.dev` falls from 3,064,035 bytes to
2,878,623, and the descriptor library of the core falls from 1,911,012 to
948,233. The generated host of compass falls from 3,195 lines to 1,324, because
the converter of an enum reaches the host only when a case of the host reads
that enum. compass names 40 node types of the 491 that its 2 packs hold, thus a
filter of the application would give the rest.

Pros:

- The Go side is the most beautiful of the 13, and it holds no tag, no
  directive, and no Dart name.
- The host covers the whole widget set, thus the hand written switch of A5 is
  gone.
- Each verified property of A5 holds: real Flutter widgets, real keys, real
  text input, a lazy list, and a real error boundary.
- A Go test of the tree needs no Flutter, because the tree is data.

Cons:

- The generator must track each Flutter release.
- No const subtree.
- A Go value holds no `BuildContext`.
- The descriptor cost of a large node set needs a measurement.

## The state model of the 3 best approaches

The 3 best are A13, A2, and A8. The table judges each 1 against 5 Go idioms.

| Go idiom | A13, a Go tree with a generated host | A2, a struct tag widget class | A8, a shadow declaration binding |
|---|---|---|---|
| A struct with methods | Full. The state is a plain Go struct, and `Build` is a method with a pointer receiver. | Broken. A const constructor needs a final field, thus a method must not write a field. The state moves to a second Go type. | Full for the state. The widget is a call, and not a struct. |
| A channel | Partial. A goroutine sends an update, and the receiver marks the tree dirty. `Build` must not read a channel. | Partial. The same rule holds. | Partial. The same rule holds. |
| `context.Context` | Full. A background call takes a context, and `defer cancel()` works since 2026-08-29. | Full | Full |
| An interface | Full. `ui.Node` is a Go interface, and a type switch over a node is plain Go. | Broken. A Go interface value is `Iface?` at `internal/emit/type.go`, and a Dart `List<Widget>` needs a `Widget`. | Partial. A handle is opaque, thus a Go interface over handles gives no type test. |
| A goroutine that owns state | Broken. `Build` runs on the Dart stack and must not block, thus it cannot ask a goroutine for a value. | Broken. The same rule holds. | Broken. The same rule holds. |

The 1 idiom that each 1 breaks:

| Approach | The idiom that it breaks | Why |
|---|---|---|
| A13 | A goroutine that owns state | The build path is synchronous, thus the state must stand in a struct that the goroutine and the build both reach. |
| A2 | A struct with methods | A widget struct holds only final fields, thus a method that writes a field is rejected. |
| A8 | Value semantics | A Dart handle must not be copied, and stage 4 copies a struct at 5 points. |

The break of A13 is the least severe. Each Go user interface library of the Go
world holds the same rule. gio, go-app, and Fyne each read the state
synchronously in the layout path. The break of A2 hits the first line of Go
that a developer writes. The break of A8 hits a rule of the compiler that
`docs/design.md` names as decision 2.

## The recommendation

**Take A13: a Go tree with a generated Dart host.** Start with the hand written
host of A5, and generate it in milestone 3.

The reason holds 5 parts:

1. **It works today.** 4 Flutter tests pass with no change of the compiler.
   Each other approach needs compiler work before the first screen runs.
2. **The Go side is the most beautiful.** A plain Go struct holds the state, a
   method gives the tree, and a closure holds the event. No tag, no directive,
   and no Dart name reaches a Go file.
3. **The Flutter model stays whole.** The host builds real Flutter widgets.
   Flutter then keeps the element tree, the key, the `State`, the focus, the
   lazy list, and the error boundary. A probe proved each 1. This is the point
   where A13 leaves `rfw` behind. The ceiling of `rfw` comes from the data model
   of `rfw`, and not from the shape of a data tree.
4. **The cost is measured and small.** A Go tree of 200 nodes plus the Dart
   walk costs 11.6 microseconds, which is 0.07 percent of a frame at 60 hertz.
   The runtime adds 64 kilobytes to an ahead of time binary.
5. **The cost that stays is a generator problem.** The 1 weak point of A5 is the
   hand written host, and A7 already answers it. The generator writes Dart from
   Dart, thus it needs no compiler change.

**The second choice is A2, a struct tag that makes a Go struct a Dart widget
class.**

A2 wins under 1 condition: the project must give Dart a real Flutter `Widget`.
2 cases hold that condition. The first case is a library: the project ships Go
widgets, and a Dart application or a `pub.dev` package constructs them. The
second case is a measurement. A profile shows that the rebuild of the mapped
subtree is the frame cost, and only a const subtree removes it. A13 gives no
const subtree, and A2 with the const work of A3 does.

Neither case holds today. Thus A2 stays the second choice, and the project
takes A13.

## The staged path

Each milestone names its compiler work and 1 proof program. The proof program
of milestone 1 already passes outside the repository.

### Milestone 1: the Go tree and a hand written host — done 2026-09-02

The directory `flutter/` holds it. `docs/flutter-state.md` holds the state
model, and `flutter/README.md` names each command.

**The deliverables.** The Go module `gart.dev/ui` at `flutter/ui` holds the
signal graph, the 3 state shapes, and 28 node types. The Dart package
`flutter/goflutter` holds the host widget, `GoWatch`, `GoTextField`, and
`GoApp`. `flutter/example/counter` holds the application.

**The proof.** 6 widget tests pass under Flutter 3.47.0: the whole tree, the
counter, the focus of a text field across a Go rebuild, a lazy list of 100,000
rows that a goroutine fills, a Go error on the screen, and 1 signal that
rebuilds 1 boundary. 35 Go tests pass under `go test` and under `go test
-race`, with no Flutter and no Dart.

**The compiler work.** The plan said none. 3 small items came up.

| Item | Change |
|---|---|
| A dependency module wrote no Dart. Stage 2 accepted the import, and stage 5 failed with "the package X has no import prefix". | `compiled` of `internal/driver/driver.go` now accepts a path whose first element holds a dot, and `libPath` writes it under `dep`. Decision 217. |
| The host had no signal after a goroutine woke, and a poll stops `pumpAndSettle` of a widget test from settling. | `$onIdle` and `$hosted` of `sched.dart`. Decision 218. |
| `flutter analyze` gave 1034 issues in the generated code. | The header of a generated file now names 18 lints. |

A library package needed no change, thus the claim of the plan holds for the
entry point. The `Ctx` interface of the plan is gone: `ui.Theme`, `ui.Media`,
and `ui.Dirty` are package functions, thus a `Build` method takes no parameter
and a closure captures no context.

1 more change came from the measure and not from a failure: the box of an
interface value writes no copy for a composite literal, thus a tree of 200
nodes allocates 200 objects in place of 400. Decision 219.

### Milestone 2: the synchronous guarantee — done 2026-09-02

`internal/analyze/sync.go` holds the pass, and decision 221 holds the choice.

**The shape.** A synchronous position is a named function type whose
declaration carries `//gart:sync`. The user took this candidate over a struct
tag and over a directive on each declaration, because a named type is the only
1 that stands as a struct field type and as a function parameter. A struct tag
reaches no parameter, and a directive on a declaration reaches no closure
literal.

```go
// Builder builds 1 node subtree.
//
//gart:sync
type Builder func() Node

type Watch struct {
	Build Builder
}
```

**The pass.** It runs after the fixed point of stage 3, because the block state
of a function is final only then. It reads the points-to sets of
`internal/analyze/flow.go`, thus it adds no analysis. For each cell of a
variable, a field, or a parameter whose type carries the marker, it reports
each function of the set that can block. A value that the flow cannot name
falls into the escape group of its signature, and the group flows into the cell,
thus an unresolved value gives a rejection and never a silent pass.

The report names 3 places: the function value, the position that holds it, and
the operation that blocks. `summary.blocksAt` of `internal/analyze/analyze.go`
keeps the first operation of each body, and `blockCause` follows the call chain
to it.

```
main.go:13:24: this function can block, and ui.Builder is a synchronous
position (Build at node.go:213:2 holds the value, and a channel receive at
main.go:14:3 blocks)
```

**The proof.** 9 tests of `internal/driver/sync_test.go` pass. They cover a
struct field, a function parameter, a local variable, a method value, a cause
1 call away, a declaration inside a `type` group, a directive on a type that is
not a function type, and a marker that stands in a dependency module. 1 test
proves the other side: a callback with no result still accepts a function that
blocks, because the host wraps it as `() { $go(f); $schedule(); }`.

`gart.dev/ui` now declares `Builder` and `ItemBuilder`, and `Watch.Build`,
`ListViewBuilder.Item`, and `RunWatch` each take 1. The Dart type of the field
does not change: it stays `Iface? Function()?`, and only the type descriptor of
the field names the new type. The 6 widget tests of the counter still pass.

### Milestone 3: the generated node set and the generated host — done 2026-09-02

`flutter/nodegen` holds the tool, and its README holds the mapping table and
the commands. Decisions 222 to 225 hold the choices.

**The compiler work.** None, as the plan said.

**The tool.** It reads `package:flutter/widgets.dart`,
`package:flutter/material.dart`, and `package:flutter/cupertino.dart` with
`package:analyzer` 8.4.1. The read takes about 5 seconds, thus the cost is
acceptable and the first open question is answered. The output:

| Output | Scope |
|---|---|
| `flutter/ui/**/widgets_gen.go`, 567 node types and 5,032 fields in 4 packs | The whole project |
| `<app>/lib/widgets_gen.dart` | 1 application |

**The 3 questions of the plan.**

| Question | The answer |
|---|---|
| How does a Go field state "not set"? | A pointer field. Decision 222. |
| How does a named constructor map to Go? | 1 Go struct of its own. Decision 223. |
| How does a callback with a result carry the synchronous rule? | The named type of milestone 2. `Widget Function(BuildContext)` maps to `ui.Builder`, and `Widget Function(BuildContext, int)` maps to `ui.ItemBuilder`. |

**The size of the output.** The second open question asked whether the Dart
tree shaker drops a type descriptor that no code reads. It does. The
measurement of 2026-09-02, on `flutter build macos --release`:

| The build | The snapshot |
|---|---|
| The hand written set of milestone 1, 28 nodes | 13,989,280 bytes |
| The generated set of 411 nodes, a shared host | 20,844,352 bytes |
| The generated set of 411 nodes, a host of the 18 cases that the counter uses | 13,628,480 bytes |

The Go node set compiles to 60,518 lines of Dart, and the shaker drops each
class and each descriptor that no code reaches. The whole cost of 7.2 megabytes
comes from the switch of the host, because a case names a Flutter widget. Thus
the host is per application. Decision 224.

**The proof.** The counter of `flutter/example/counter` now builds each widget
from the generated set. 6 widget tests pass under Flutter 3.47.0, and the 35
Go tests of `flutter/ui` and `flutter/example/counter/go` pass with no Flutter.
`flutter analyze` gives no issue for `goflutter`, for `nodegen`, and for the
application.

**What the mapping drops.** A parameter of a Dart type that no Go type carries
stays out. `dart run bin/nodegen.dart report ../goflutter -all` names each 1,
with the count of each cause and the count of each unmapped Dart type.

The reading of 2026-09-02 held 411 node types with 3576 fields, and 273 widget
constructors reached no node type. A measure of the blocking parameter of each
1 of the 273 gave a long tail: 176 distinct types, and the largest holds 11.
Thus no single Dart type unlocks a large group.

The 9 value types of the same day widened the mapping to 414 node types with
3851 fields, thus 270 constructors stayed out.

**The state of 2026-09-03.** The set holds 564 node types with 4975 fields, and
119 constructors stay out.

| The work | Node types | Fields | Dropped |
|---|---|---|---|
| The reading of 2026-09-02 | 411 | 3576 | 273 |
| The 9 value types, decision 231 | 414 | 3851 | 270 |
| 6 more value types, decision 246 | 423 | 3977 | 261 |
| The 3 identity objects, decision 247 | 427 | 4111 | 257 |
| The long tail of 2026-09-03, decisions 269 to 274 | 509 | 4806 | 174 |
| The 34 theme value types, decision 311 | 550 | 4894 | 133 |
| 4 value groups and 8 more themes, decision 313 | 564 | 4975 | 119 |

The set of 509 types costs an application 26,752 bytes less than the set of
427, because the tree shaker drops each type that the application does not
name. Read decision 275.

`ButtonStyle`, `ImageSource`, `Decoration`, `Size`, `Shadow`, and `Gradient`
took the value part. `FocusKey`, `ScrollKey`, and `StatesKey` took the identity
part: the host owns the object, and the Go program names it by a key.
`ui.TextInput` was the pattern for the 3.

The 4 constructors that the identity work recovered are the ones that hold such
a parameter and require it: `Focus.withExternalFocusNode`, `KeyboardListener`,
`PrimaryScrollController`, and `RawKeyboardListener`. The other 124 parameters
became fields of a node that already mapped.

Read the mapping table of `flutter/nodegen/README.md`.

## The measurements

Each number comes from an ahead of time build with `dart compile exe`, on the
machine of the project, on 2026-09-01. The Go source stood outside the
repository, and the session deleted it.

| Measure | Value |
|---|---|
| A Go tree of 200 nodes, with no text work | 8.66 microseconds |
| The same tree, with `strconv.Itoa` and a concatenation for each node | 50.34 microseconds |
| The Dart walk of 200 nodes, with `toDartString` for each node | 2.89 microseconds |
| A Dart tree of 200 nodes with a concatenation, the baseline | 5.61 microseconds |
| A Go build of 2 nodes | 121.1 nanoseconds |
| The same, plus the Dart walk | 167.2 nanoseconds |
| `GoString.fromString` of 13 characters | 47.1 nanoseconds |
| `GoString.toDartString` of 13 characters | 13.0 nanoseconds |
| The frame budget at 60 hertz | 16,700 microseconds |
| The ahead of time binary of `goruntime` and a small Go package | 64 kilobytes over a bare Dart program |
| The type descriptor bytes against the code bytes, for 5 named types | 6,353 against 7,676 |
| The Dart source of `goruntime` | 6,978 lines in 288 kilobytes |

**The 2 rows of `toDartString` moved on 2026-09-06.** A `GoString` holds the
decode after the first read now, thus a rebuild that meets the same label again
pays nothing. A screen of 500 labels of 20 bytes fell from 7.65 to 0.78
microseconds, and 200 labels of 500 bytes fell from 35.09 to 0.32. Read
decision 357 and `bench/README.md`.

**The 2026-09-01 date of that table holds.** The Dart source of `goruntime`
grew with the surface: it reads 8,457 lines in 340 kilobytes on 2026-09-03,
against the 6,978 lines in 288 kilobytes of the row above. Each other row of
the table names a speed, and no measurement of 2026-09-03 moved 1 of them by
more than 2%.

The last 2 rows carry the risk of a large node set. The measure of 2026-09-05
gives about 3,280 bytes of descriptor source for each node type of
`gart.dev/ui`, thus the 564 node types cost 1.85 megabytes of Dart source. The
Dart tree shaker drops a descriptor that no code reads, thus an ahead of time
build carries the node types of the application alone. The pack split of
2026-09-05 cut that cost: an application compiles the packs that it names
alone, and the descriptor library of the core held 948,233 bytes. The measure of
2026-09-02 named 1,271 bytes for each type, and it read a struct with fewer
fields than a node type holds.

**The node filter of 2026-09-06 cut that library to 310,204 bytes.** Read the
next section.

## The node filter of the Go side

The generated Dart host held a case for each of the 566 node types before
2026-09-02, and the host is per application since then. The Go side was not.
`gart.dev/ui` declares 273 node types, `gart.dev/ui/material` 218, and
`gart.dev/ui/cupertino` 75. compass names 27 of the 491 that its 2 packs hold.
The emitter writes each declaration of each package that the program imports,
thus the Dart output carried 464 dead classes and 464 dead descriptors.

`internal/nodefilter` reads each file that nodegen wrote and it keeps the types
of a name set. `glutter` gives the set: `flutter/nodescan` reads the module of
the application with the type information of the Go tool chain, and it names
each type of `gart.dev/ui` and of each pack under it that the code uses. The
answer is exact, and it already drives the host.

**The filter runs at the build.** `driver.Options.Filter` is a function, and
the driver calls it in front of each load. The result reaches
`packages.Config.Overlay`, thus the file on disk never changes: 2 applications
of 1 checkout each hold their own set, and a watch cycle that meets a new node
type reads it at once. The Go type checker then reads the filtered source, thus
an application that names a type outside the set gives a plain compile error
and no build reads a stale file.

The keep set grows 3 ways. It holds each name of the scan, each name that a
hand written file of the pack reads, and each declared type that a kept
declaration names. The walk of the names skips the name of a struct field and
the field of a selector, because such a name is no type: `Opacity float64` of
the node type `Opacity` would otherwise keep that type in each application.

**The measure of 2026-09-06.** The build is `flutter/example/compass`, first
with `gart build` and then with `glutter gen`, on the same commit and the same
compiler. Each number sums the bytes of every `.dart` file of the output. The
load average of the machine stood between 215 and 270 during the 2 runs,
because a second agent ran a test suite. A byte count reads no load.

| Measure | Before | After | Cut |
|---|---|---|---|
| The whole Dart output | 18,318,376 | 16,089,390 | 12.2 percent |
| The tree of `gart.dev/ui` | 2,922,881 | 693,895 | 76.3 percent |
| `ui.dart`, the descriptor library of the core | 948,233 | 310,204 | 67.3 percent |
| `ui/material.dart`, the descriptor library of Material | 875,599 | 70,580 | 91.9 percent |
| `ui/widgets_gen.part.dart`, the classes of the core | 350,524 | 27,527 | 92.1 percent |
| `ui/material/widgets_gen.part.dart` | 464,387 | 37,740 | 91.9 percent |
| Node classes of the 2 packs | 491 | 27 | 94.5 percent |

The other 3 examples read the same filter. counter keeps 21 node types of 37
names, apilist 13 of 21, and form 9 of 19. Each 1 passes `flutter analyze` and
its widget tests.

**The value files join the filter on 2026-09-06.** `flutter/ui/themes.go` and
`flutter/ui/types.go` each carry the line `// nodefilter:values` on the first
line, thus the filter reads them as it reads a file of nodegen. A type of such
a file stays when a kept declaration names it, and a function with no receiver
leaves the file when its signature names a type that the set drops.
`themes.part.dart` fell from 139,198 bytes to 8,021, and `types.part.dart` from
69,670 to 53,650.

## The theme of a Go program

`Theme.of(context)` needs a `BuildContext`, and a Go value holds none. The host
thus read the theme of Flutter and it wrote 5 values into `ui.ThemeData` before
each build. A Go program could name no theme of its own, and the port of the
Flutter sample compass_app drew the default theme of Material.

**The Go tree holds the application now.** `material.MaterialApp` is a node of
the generated set, and its fields `Theme` and `DarkTheme` each take a
`ui.ThemeStyle`. That value carries a color scheme, a text theme, and the
decoration of an input since 2026-09-06.

| Type | Content |
|---|---|
| `ui.ColorScheme` | Each of the 47 colors of the Flutter class. The 9 that the Flutter constructor demands stand as plain fields, and each other 1 is a pointer. |
| `ui.TextTheme` | The 15 named styles of the Flutter `TextTheme`. |
| `ui.InputDecorationStyle` | The decoration of a text field. `themes.go` already held it. |

`Resolve` of `ui.ColorScheme` gives the color that Flutter derives for each nil
pointer: `primaryContainer` reads `primary`, `outline` reads `onSurface`,
`shadow` reads black, and so on. A Go build thus reads no transparent color for
a field that the program left out, and a zero value gives no silent wrong
paint.

`ui.UseTheme(light, dark)` names the 2 themes, and `ui.Theme()` gives the 1 of
the brightness that the host reports. A program names the same 2 values in the
MaterialApp node, thus 1 source holds the theme and Flutter draws with it.

`GoApp` can stand above that MaterialApp. No `MediaQuery` stands over it in
that place, thus it reads `MediaQueryData.fromView(View.of(context))` and the
brightness of the platform. `ui.HostTheme` holds the 5 values that the host
writes, and a program that calls no `UseTheme` reads them as before.

**A variable font needs an axis.** `fontWeight` of Flutter names a face of a
family, and a variable font holds 1 face. `ui.TextStyle.Variations` names the
axis `wght` with a number, thus a Go program reads Rubik at the weight 800.

## The calls of the host

`ui.Call` of `flutter/ui/ask.go` reached the host since 2026-09-04, and no name
of the library rode on it. 5 calls stand in `flutter/ui/calls.go` now.

| Call | What the host runs |
|---|---|
| `DateRange` | `showDateRangePicker` of Material. |
| `SnackBar` | A `SnackBar` of the `ScaffoldMessenger`. The answer names the button that the user pressed. |
| `Share` | The share sheet of the platform. The host holds no such sheet, thus the application answers this name. |
| `SaveValue`, `ReadValue` | A value that survives a restart. The host holds a map in memory, thus an application that needs more names `shared_preferences`. |

`goHostCall` of `goflutter` answers each 1 with no code of the application, and
a handler that `goSetCall` installs answers its own names first. Each call
needs a goroutine, because it parks until the host answers.

**A dialog and a snack bar each need a `BuildContext` under the application.**
The node `ui.Anchor` keeps the context of its place while it stands on the
screen, and the 2 calls read it. A program puts 1 Anchor under the `Home` of
the application.

## The probes of the language

Each probe below ran again on 2026-09-03, under Dart 3.13.2 and Flutter 3.47.2.
Each 1 stood outside the repository, and the session deleted it.

| The probe | The result |
|---|---|
| A `final class` of library A, that library B extends | `dart analyze` rejects it with `invalid_use_of_type_outside_library`. The same extension inside 1 library passes. This is the proof of constraint 1. |
| A const constructor of a class that holds a mutable field | `dart analyze` rejects it with `const_constructor_with_non_final_field`. A const widget thus needs a final field, and A2 pays that cost. |
| 2 `const W(1)` values, against 2 `W(1)` values | `identical` gives true for the const pair and false for the other pair. Only a canonical const widget reaches the short cut of `Element.updateChild`. |
| A gart-shaped class that extends `StatelessWidget`, with a mutable field and a field-wise `==` | `flutter analyze` gives 3 warnings and 0 errors: `must_be_immutable` 1 time, and `invalid_override_of_non_virtual_member` 2 times. The class compiles and it mounts. |
| An async closure for a `void Function()` parameter | Dart accepts it, and it runs. The event handler boundary of constraint 3 thus holds. |
| `augment class` in a part file | Dart 3.13.2 gives a parse error, with and without `--enable-experiment=augmentations`. An augmentation reaches no approach. |

Dart macros are cancelled and will not ship. `https://dart.dev/language/macros`
states it. Thus Dart holds no compile time mechanism that writes a widget
class, and each approach that needs generated Dart needs a tool of its own.

## The prior art

The survey of 2026-08-28 read 15 projects, and the pass of 2026-09-03 checked
each state again.

| Project | Shape | State on 2026-09-03 | What it settles |
|---|---|---|---|
| flutter_rust_bridge | An FFI bridge. Rust writes plain functions and structs, and Dart writes each widget. | Active, 5,394 stars | Its own `frb(ui_state)` still leaves the tree in Dart. A synchronous mode exists only because `build` cannot await. |
| flutter-go-bridge | The Go form of the same bridge | Stalled, 103 stars, last push 2024-05-30 | It carries functions only. |
| go-flutter | An engine embedder. Go writes a plugin, and Dart writes each widget. | Dead, 5,931 stars. The last functional commit is 2022-07-24, and each later commit bumps a dependency. | An embedder answers a deployment question, and not the widget question. |
| Flet | A protocol binding for Python, with a hand written Dart adapter for each control. | Active, 16,636 stars | It gives the price of the A1 host: 40,259 lines of hand written Dart. |
| rfw | The first-party data-driven widget tree of Flutter | Active | Read A9. Its README states its own ceiling. |
| RemoteFlutterSharp | A C# domain language over `rfw` | Archived, 12 stars | The only C# attempt, and it stopped at a proof of concept. |
| ClojureDart | Compile to Dart with real subclasses: `:extends`, `^super`, `^:mixin`, and a `widget` macro that replaces `StatelessWidget`, `StatefulWidget`, `State`, `Builder`, and `StatefulBuilder`. | Active, 1,637 stars | The only prior art in which a second language authors a Flutter widget. A2 and A4 are thus possible, and not only plausible. |
| Dotlin | Kotlin to Dart. A marker on the supertype writes `implements` or `with`. | Archived 2024-03-06, 221 stars | It answers the relation question of A4: the marker carries the choice. It ships no Flutter story. |
| ts2dart | TypeScript to Dart, built for AngularDart | Archived, 185 stars, last push 2016 | The 1 industrial compile-to-Dart effort before gart. |
| jaspr | A reimplementation of the Flutter framework shape for the web | Active, 2,455 stars | The reconciler is 4,086 code lines in 17 files, and the tag library is 12,072 generated lines. It sizes A3 and A10. |
| gio | An immediate mode Go user interface. A config struct, 1 `Layout` method, and closure children. | Active, 2,242 stars | It proves A11 in Go. It needs no key, no element reuse, and no equality. |
| go-app | A declarative Go user interface. An embedded base, a `Render` method, and builder chains. | Active, 8,959 stars | The closest Go shape to a Flutter API. |
| Fyne | A retained mode Go user interface, with positional constructors and setters | Active, 28,668 stars | The most adopted Go user interface, and the opposite model of A13. |
| Vugu | A template file plus a generated Go `Build` method | Experimental, 5,002 stars | A second file format and a second tool chain step. A13 needs neither. |
| Compose Multiplatform | A vendor reimplementation of Material 3 in Kotlin, with a compiler plugin | Active, 19,336 stars | It is the true price of A6: a funded platform team over years. |

The table answers 1 question of the survey. No project reaches a Flutter widget
from Go, and the 2 that tried are dead or stalled. gart thus copies no path,
and ClojureDart is the nearest example.

## The corrections

The pass of 2026-09-01 and the pass of 2026-09-03 each re-read the option
survey of 2026-08-28 against the code. The Flutter facts of that file all hold.
A re-run against Flutter 3.47.0 verified 14 of 14 claims about
`widgets/framework.dart`, `rendering/object.dart`, `scheduler/ticker.dart`, and
`animation/animation_controller.dart`. A second re-run against Flutter 3.47.2
verified each size and each count of the section "What Flutter asks of a
language".

3 claims about the compiler were false on 2026-09-01.

| The claim of 2026-08-28 | The state of 2026-09-01 |
|---|---|
| `asyncWord` writes ` async` | False. No function of that name exists. The emitter writes no `async`. |
| `resultText` wraps the result in `Future<...>` | False. `resultText` of `internal/emit/decl.go` wraps nothing. |
| A signature that blocks gives `FutureOr<R> Function(...)` | False. `funcType` of `internal/emit/type.go` writes `R Function(...)` for each signature. `FutureOr` reaches no line of `type.go` or `decl.go`. |

The 3 rows date from the async model of 2026-08-28. The unwind model of
2026-08-29 removed each 1, and `docs/flutter-first.md` names the change.

6 more claims are false on 2026-09-03. The compiler grew, thus each row below
states a limit that the compiler no longer holds.

| The claim of 2026-08-28 | The state of 2026-09-03 |
|---|---|
| A Dart handle type in Go needs an emitter change, and only `unsafe.Pointer` gives `Object?` | False. `Object` of `syscall/dart` is that type, and `dartBoundary` of `internal/verify/dart.go` accepts it. Read decision 239 and decision 240. |
| A call of an arbitrary Dart constructor or method is out of reach, because `intrinsic` is a flat switch | False. A `//gart:dart` declaration names any Dart symbol, and `dartCall` of `internal/emit/dart.go` writes the call. The boundary carries `bool`, `int`, `float64`, `string`, and `dart.Object`. |
| An import of a Dart library that is not `goruntime` needs an emitter change | False. `dartPrefix` and `dartOrder` of `internal/emit/dart.go` and of `internal/emit/emit.go` write 1 prefixed import for each Dart library. |
| A guarantee that a function stays synchronous needs a new pass after stage 3 | False. `internal/analyze/sync.go` holds that pass, and milestone 2 above records it. |
| A box into an interface copies the struct again | False. `box` of `internal/emit/iface.go` writes no copy for a composite literal. A widget tree thus costs 1 object for each node. Decision 219. |
| The repository holds no mention of Flutter and no mention of a widget, thus each option starts from 0 | False. `flutter/` holds `gart.dev/ui` with 564 node types, the Dart package `goflutter`, the tool `nodegen`, the tool `nodescan`, and 3 example applications. |

2 more claims are incomplete.

| Claim | The correction |
|---|---|
| "A Dart caller reads the zero value and sees no report" of `docs/flutter-first.md` | True for a block point that reads no goroutine, for example a nil channel. A real channel reads `$current!` at `runtime/goruntime/lib/src/chan.dart:61`, thus the Dart caller gets a null check error of Dart that names no Go position. |
| "A Go function declaration with no body already parses, and stage 2 rejects a read of it" of `docs/flutter-first.md` | The rejection of `checkBodyless` of `internal/verify/verify.go` covers a read inside the same package only. A cross-package call compiles today, and stage 5 already writes the correct Dart call site. |

1 conclusion of the option survey does not hold.

The survey named the ceiling of `rfw` as the risk of the approach that this
file calls A5. That transfer is wrong. The ceiling of `rfw` comes from the data
model of `rfw`: no arithmetic, no key, no text input, no const, and an O(N²)
list index. A Go tree with a Dart host holds none of those limits, because the
host builds real Flutter widgets in Dart code. A probe proved the key, the text
input, the focus, the lazy list, and the error boundary.

**Why this file names no line number of the compiler.** The option survey gave
a line number for each claim about `internal/emit` and `internal/verify`. The
pass of 2026-09-01 mapped 20 of them to their new position. 2 days later 18 of
those 20 positions were wrong again, because the files grew. A line number thus
rots faster than a pass corrects it, and a symbol name does not. The archive
notice of `docs/flutter-options.md` holds no line number for that reason.

1 more note stands. "Only `unsafe.Pointer` gives `Object?`" was incomplete even
on 2026-08-28. `dartType` of `internal/emit/type.go` gives `Object?` at 6 more
points:

- a `dart.Object` of `syscall/dart`,
- a named type that names itself through a pointer, a slice, or a map,
- a signature that names such a type,
- a core struct of a constraint that holds a type parameter,
- a pointer to a type parameter,
- the depth cap of 64.

## The unverified claims

Each claim below stands in this file with no measurement behind it.

- The cost of a per-frame update of A9 at 60 hertz. The package publishes no
  benchmark.

**Hot reload, measured 2026-09-02.** The chain works. A probe ran the counter
with `flutter run -d macos`, held `gart watch` on the Go module, and changed 1
line of the Go source.

| Step | Result |
|---|---|
| `gart watch` | It read the change and wrote the Dart again. 1 package compiled, and 13 stayed cached. |
| The Flutter hot reload | It took the new Dart. 3 of 828 libraries, 601 milliseconds and 1,804 milliseconds over 2 runs. |
| The new Go code | It runs. A `fmt.Println` that the change added wrote its line after the reassemble. |

The probe sent SIGUSR1 to the process that `--pid-file` names, thus it needed
no keystroke.

**A bodyless Go declaration, measured 2026-09-02.** The earlier text said that
`go build` and `go vet` each reject such a declaration, and that the editor
reads the same tool chain. 2 of the 3 parts are false.

| Tool | Result |
|---|---|
| `go build ./...` | Rejects. `missing function body`, exit 1. |
| `go vet ./...` | Accepts. No diagnostic, exit 0. |
| `gopls check` | Accepts. No diagnostic. A control file with a type error gives the diagnostic, thus the probe stands. |

The editor thus stays quiet for item 1 of `docs/flutter-first.md`, and only
`go build` names the declaration. A module that carries 1 still needs a twin
file behind a build tag, for the same cause that `dart.Object` needs 1.

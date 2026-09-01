# The Flutter design

This file answers how a Go program that gart compiles drives a Flutter user
interface. It weighs 13 approaches, and it takes 1 decision.

`docs/flutter-options.md` holds the option survey of 2026-08-28. This file
re-reads each option against the state of 2026-09-01, and it adds 7 more.
`docs/flutter-first.md` holds the order of the compiler work. Read this file
before a choice of the Flutter mechanism.

Each claim about the compiler names a file and a line. Each measurement ran on
2026-09-01, on the machine of the project: Flutter 3.47.0, Dart SDK 3.13.0,
macOS on arm64. The section "The measurements" holds each number, and the
section "The corrections" names each claim of `docs/flutter-options.md` that
this pass found to be false.

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

Stage 5 writes 1 class head, `final class X {`, at `internal/emit/decl.go:43`.
1 Go package maps to 1 Dart library. A `final` class of 1 library accepts no
subclass in another library, and a probe of `docs/flutter-options.md` proved
that rule. Thus a Go type is not a Flutter `Widget` today, and no Dart code
extends a gart class.

The consequence reaches each value that crosses the boundary. A Go struct is
`X$`, and a Go interface value is `Iface?` at `internal/emit/type.go:85`. A Go
string is `GoString` at `:111`, and a Go slice is `Slice<T>`. 3 Go types map to
a plain Dart type: `bool`, `float64` to `double`, and each integer type to
`GoInt` at `:118`. `GoInt` is an extension type over the Dart `int` at
`runtime/goruntime/lib/src/goint_native.dart:10`, thus it erases on the Flutter
target and costs nothing.

### Constraint 2: a function that can block gives a wrong answer to Dart

The emitted Dart holds no `Future` and no `await`. A function that can block
writes its locals to a frame, sets `$blocked`, and returns the zero value:
`internal/emit/machine.go:349`. Only a Go caller tests that field, at `:469`.

A probe proved the 2 failures. A Go `Build` method that reads a nil channel
gives `null` to a Dart caller, and `$blocked` holds the frame. A Go `Build`
method that reads a real channel gives the Dart caller a null check error from
`runtime/goruntime/lib/src/chan.dart:61`, which names no Go position.

A Flutter `build` method must be synchronous, and `StatefulElement.build` calls
`state.build(this)` synchronously at `widgets/framework.dart:5944`. Thus a Go
function in a build position must never block, and the compiler must prove it.
The information already exists: `blockingExports` of
`internal/driver/driver.go:274` writes the list into the archive. A probe of a
Go type with 2 methods that block gave the archive entry
`["Blocker.Build", "Blocker.Silent"]`.

### Constraint 3: the Go scheduler is a guest of the Dart event loop

`$schedule` at `runtime/goruntime/lib/src/sched.dart:125` runs the run queue
until it is empty, and it returns. A timer or a future calls `$wake`. Flutter
owns the loop already, and it needs no change.

A Flutter event handler that starts Go work costs 2 lines:
`() { $go(f); $schedule(); }`. `$go` stands at `sched.dart:168`, and the
finalizer of `sched.dart:188` already holds that pattern. A Go call that waits
for a Dart future reads `$await` at `sched.dart:240`, and that call needs an
intrinsic of stage 5.

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
`internal/verify/verify.go:211` rejects a read of a bodyless function of the
**same** package only, because `bodyless` at `:44` reads the files of 1
package. A cross-package call already reaches a correct Dart call site. Thus
item 1 of `docs/flutter-first.md` is smaller than the file states.

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
   holds a lazy `ListView.builder` too, and its index lookup is O(N) for each
   index, thus a full scroll is O(N²). The source of `rfw` names that cost.
3. **A theme.** No approach outside A3 and A4 lets Go call `Theme.of(context)`,
   because a Go value holds no `BuildContext`. The host reads the theme and
   passes a snapshot to the Go build. That gives the full behavior with a
   different shape, thus the mark is `P`.
4. **A 60 Hz animation.** The host owns the `AnimationController` and passes
   the value to the Go build at each frame. `TickerProvider` at
   `scheduler/ticker.dart:43` holds 1 method, and a Go type cannot implement a
   Dart interface today, thus Go cannot own the controller. The measured cost
   of a Go tree of 200 nodes is 8.66 microseconds, thus the frame budget holds.
5. **Text input.** A5 and A13 give the mark from a real run. The Dart host
   builds a real `TextField`, thus Flutter keeps the text, the controller, and
   the focus across a rebuild that Go drives. A9 holds no text input widget of
   any kind.
6. **Navigation.** The host owns the `Navigator`. A Go call that waits for the
   result of a pop needs the `$await` intrinsic, because `Navigator.push` gives
   a `Future<T?>` at `widgets/navigator.dart:5060`.
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
   and the effect of the Dart tree shaker on those descriptors is unverified.
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
    it and calls `ErrorWidget.builder` at `:5838`. A probe proved the path. A6
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

**The compiler work.** `intrinsic` at `internal/emit/call.go:124` is a flat
switch on the name of an interop function, and `intrinsicText` at `:211` holds
132 cases. `internal/overlay/std/syscall/dart/dart.go` holds 132 Go stubs. A
handle needs a named Go type whose Dart type is `Object?`, and only
`unsafe.Pointer` gives that type at `internal/emit/type.go:115`. Stage 2
rejects `unsafe` at `internal/verify/verify.go:105`. A callback with a result
needs the rule of constraint 2.

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
shape. `decl.go:43` writes the head, `:48` a final field, and `:56` a const
named constructor with `required` and a default. `:202` drops the equality
operations, and `:250` writes `@override` and the Dart member name.
`paramList` at `:466` and the composite literal of `internal/emit/expr.go:662`
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
  thus `writeEqual` at `decl.go:202` must not run for such a class.

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
`internal/driver/driver.go:504` must write the extra file into the output
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

**The compiler work.** `decl.go:43` writes the superclass, and `:250` writes
`@override` and the Dart member name. `internal/name/name.go` needs a map from
a Go method name to a Dart member name. `internal/emit/emit.go:570` needs an
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
`runtime/goruntime/lib/src/iface.dart:124`.

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
`package main` holds no library URI at `internal/driver/driver.go:504`, and
`writeMain` at `internal/emit/emit.go:634` writes `bin/main.dart`. Thus
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
that stage 5 reads, an import of a Dart URI at `internal/emit/emit.go:570`, a
handle type at `internal/emit/type.go:115`, and the value boundary of item 5.

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

**The runtime cost.** The binary blob decodes in 1 pass. The text form is about
10 times slower, and the package discourages it on a mobile target. Each index
of a lazy list re-walks the list from the start, thus a full scroll is O(N²).
The source of the package names that cost.

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
- A Go struct has value semantics, and a box into an interface copies it at
  `internal/emit/iface.go:236`. Thus the diff cannot compare 2 nodes by
  identity, and it must compare each field.
- The measurement of A5 gives 2.89 microseconds for 200 nodes, thus the diff
  saves little at that size. The gain needs a measurement, and it is
  unverified.

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
type descriptor pair for each node type, at about 1,271 bytes of Dart source
for each 1. The effect of the Dart tree shaker on an unused descriptor is
unverified.

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
| An interface | Full. `ui.Node` is a Go interface, and a type switch over a node is plain Go. | Broken. A Go interface value is `Iface?` at `internal/emit/type.go:85`, and a Dart `List<Widget>` needs a `Widget`. | Partial. A handle is opaque, thus a Go interface over handles gives no type test. |
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

### Milestone 3: the generated node set and the generated host

**The compiler work.** None. 1 new tool reads the public API of Flutter with
`package:analyzer`. It writes the Go node structs, and the Dart host case for
each 1. The tool must answer 3 questions:

| Question | The 3 candidates |
|---|---|
| How does a Go field state "not set"? | A pointer field, a bitmask per struct, or a wrapper type `Opt[T]`. |
| How does a named constructor map to Go? | A Go function of the package, a variadic option, or a field of the struct. |
| How does a callback with a result carry the synchronous rule? | A named Go type, a directive, or an interface of the node package. |

**The proof program.** A Material application: a `Scaffold`, an `AppBar`, a
`ListView.builder` over a list that an HTTP GET fills, a `TextField` with a
validation in Go, a theme, and a route. This is program 2 and program 3 of
`docs/flutter-first.md` in 1 application. It needs `encoding/json` with the
marshal methods of `time`, which is task 2 of the open list.

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

The last 2 rows carry the risk of a large node set. Each named Go type costs
about 1,271 bytes of descriptor source. A generated set of 300 node types then
costs about 380 kilobytes of Dart source. The effect of the Dart tree shaker on
a descriptor that no code reads is unverified.

## The corrections

This pass re-read each claim of `docs/flutter-options.md` against the code. The
Flutter facts of that file all hold. A re-run of each Flutter measurement
against Flutter 3.47.0 verified 14 of 14 claims about `widgets/framework.dart`,
`rendering/object.dart`, `scheduler/ticker.dart`, and
`animation/animation_controller.dart`.

3 claims about the compiler are false.

| Claim of `docs/flutter-options.md` | The state of 2026-09-01 |
|---|---|
| `asyncWord` writes ` async` at `internal/emit/decl.go:303` | False. No function of that name exists. The emitter writes no `async`. |
| `resultText` wraps the result in `Future<...>` at `internal/emit/decl.go:316` | False. `resultText` stands at `internal/emit/decl.go:512`, and it wraps nothing. |
| A signature that blocks gives `FutureOr<R> Function(...)` at `internal/emit/type.go:81` | False. `funcType` at `internal/emit/type.go:128` writes `R Function(...)` for each signature. `FutureOr` reaches no line of `type.go` or `decl.go`. |

The 3 rows date from the async model of 2026-08-28. The unwind model of
2026-08-29 removed each 1. `docs/flutter-first.md` names the change, and
`docs/flutter-options.md` still holds the 3 rows.

2 more claims are incomplete.

| Claim | The correction |
|---|---|
| "A Dart caller reads the zero value and sees no report" of `docs/flutter-first.md` | True for a block point that reads no goroutine, for example a nil channel. A real channel reads `$current!` at `runtime/goruntime/lib/src/chan.dart:61`, thus the Dart caller gets a null check error of Dart that names no Go position. |
| "A Go function declaration with no body already parses, and stage 2 rejects a read of it" of `docs/flutter-first.md` | The rejection of `internal/verify/verify.go:211` covers a read inside the same package only. A cross-package call compiles today, and stage 5 already writes the correct Dart call site. |

1 conclusion of `docs/flutter-options.md` does not hold.

Option 5 of that file names its risk as the ceiling of `rfw`. That transfer is
wrong. The ceiling of `rfw` comes from the data model of `rfw`: no
arithmetic, no key, no text input, no const, and an O(N²) list index. A Go tree
with a Dart host holds none of those limits, because the host builds real
Flutter widgets in Dart code. A probe proved the key, the text input, the
focus, the lazy list, and the error boundary.

Each line number that `docs/flutter-options.md` gives for `internal/emit` and
for `internal/verify` is now stale, because the files grew. The table below
holds the current position of each cited item.

| Item | The line of 2026-08-28 | The line of 2026-09-01 |
|---|---|---|
| The class head, `final class X {` | `decl.go:36` | `decl.go:43` |
| A mutable struct field | `decl.go:41` | `decl.go:48` |
| The positional constructor | `decl.go:49` | `decl.go:56` |
| `$assign` | `decl.go:162` | `decl.go:171` |
| `writeEqual` | `decl.go:191` | `decl.go:202` |
| The 2 `@override` lines | `decl.go:199`, `:206` | `decl.go:210`, `:217` |
| The method head | `decl.go:218` | `decl.go:250` |
| The receiver and its `$copy` | `decl.go:227` | `decl.go:331` |
| `paramList` | `decl.go:295` | `decl.go:466` |
| A pointer to a struct | `type.go:33` | `type.go:70` |
| A function value | `type.go:43` | `type.go:128` |
| An interface value, `Iface?` | `type.go:47` | `type.go:85` |
| A Go string, `GoString` | `type.go:60` | `type.go:111` |
| `unsafe.Pointer` to `Object?` | `type.go:64` | `type.go:115` |
| The composite literal | `expr.go:438` | `expr.go:662` |
| `intrinsic` | `call.go:46` | `call.go:124`, and `intrinsicText` at `:211` |
| A `FieldDesc` with the raw tag | `iface.go:101` | `iface.go:236` |
| The rejection of `unsafe` | `verify.go:123` | `verify.go:105` |
| The build tag `dart` | `load.go:18` | `load.go:20` |
| A string constant as `final` | `emit.go:326` | `emit.go:695`, in `constDecl` |

2 more rows of that file need a note.

- "Only `unsafe.Pointer` gives `Object?`" is incomplete. `dartType` gives
  `Object?` at 4 more points: a cyclic named type at `internal/emit/type.go:36`,
  a pointer to a type parameter at `:65`, a core struct that holds a parameter
  at `:52`, and the depth cap at `:17`.
- Each Dart sketch of that file writes `int Count` for a Go `int` field. The
  emitter writes `GoInt Count` since 2026-09-01. `GoInt` is an extension type
  over the Dart `int`, thus the 2 forms are equal on the Flutter target.

## The unverified claims

Each claim below stands in this file with no measurement behind it.

- Hot reload of a Flutter application after a `gart watch` pass. The chain is
  mechanically clear, and no probe ran it.
- The effect of the Dart tree shaker on a type descriptor that no code reads.
- The gain of the Dart reconciler of A10 over the plain walk of A5.
- The ability of `package:analyzer` to read the whole Flutter public API at an
  acceptable cost.
- The cost of a per-frame update of A9 at 60 hertz. The package publishes no
  benchmark.
- The exact behavior of `gopls` for a bodyless Go declaration. `go build` and
  `go vet` each reject it, and the editor reads the same tool chain.

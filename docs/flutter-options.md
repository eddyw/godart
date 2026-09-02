# Flutter options

This file lists each option that reaches Flutter from Go through gart. It takes
no decision and it gives no recommendation. The user decides later.

`docs/flutter-design.md` replaces this file. That file re-reads each option
against the state of 2026-09-01, it adds 7 more, and it takes the decision. Its
section "The corrections" names 3 claims of this file that are false and 1
conclusion that does not hold. Read that file first.

The file stands from 2026-08-28. The block model changed on 2026-08-29, and 2
rows below carry the correction. **Each line number of this file is stale.** A
line number rots on the next commit, and the table "The corrections" of
`docs/flutter-design.md` holds the position of each cited item on 2026-09-01.
Read the file name and the symbol, and not the number. `docs/flutter-first.md` holds the order of the
work, and it names the state of 2026-09-01.

Each number comes from a measurement of the Flutter SDK at
`/path/to/flutter` or from the gart source. The section "Facts"
holds the file path and the line number of each claim.

## The question

The user asks which options reach Flutter from this new flavor of Go. The user
cares most about the core framework, and asks whether a Go Material and a Go
Cupertino beat a binding. The user also asks what a binding looks like, and asks
for several options that feel like natural Go.

## What Flutter needs from a language

### The layer stack

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

The 3 layers that a widget layer needs are `painting`, `rendering`, and
`widgets`. Their sum is 127,019 code lines. The full base under `material`, with
`foundation`, `physics`, `animation`, `gestures`, `scheduler`, `semantics`, and
`services`, is 163,149 code lines.

34 percent of the apparent size of `material` is a data table and not logic.
`material/icons.dart` holds 11,658 code lines of icon codepoint constants, and
`material/animated_icons/data/` holds 34,066 code lines of path data. Hand
written `material` is 88,619 code lines. `cupertino/icons.dart` holds 6,372 code
lines of the same form, thus hand written `cupertino` is 24,140 code lines.

A port must generate the tables from the upstream source. Go has no const class
instance, thus a Go icon table needs a code generator.

### Material and Cupertino reach no native code

Material and Cupertino are pure Dart on top of `widgets`. Their whole import set
outside Flutter is `dart:ui`, `dart:math`, `dart:async`, `dart:collection`, 1
`dart:io` read of `Platform`, `dart:developer`, `package:vector_math`,
`package:collection`, and `package:material_color_utilities`.

A search for `SystemChannels.`, `MethodChannel(`, and `dart:ffi` over all 250
files gives no hit. The whole platform contact is 5 wrapper classes of
`package:flutter/services.dart`: `HapticFeedback`, `SystemChrome`,
`SystemNavigator`, `SystemSound`, and `HardwareKeyboard`.

The only 3 reaches to the engine are `ui.FragmentProgram.fromAsset` in
`material/ink_sparkle.dart:424`, the `Platform` read in
`material/about.dart:12`, and `ui.PictureRecorder` in
`material/color_scheme.dart:2133`.

Thus no native barrier blocks a Go Material. A different barrier blocks it:
Material and Cupertino define 48 private `RenderObject` subclasses and 1
`RenderObjectElement` subclass, and their classes extend more than 30 distinct
base classes.

### The features that a widget class needs

The list holds the hardest feature first.

| Feature | Why Flutter needs it | Evidence |
|---|---|---|
| Class inheritance | 18 abstract classes of `widgets/framework.dart` and 7 of `rendering` accept a subclass. Each contract is small, and inheritance delivers it. | `widgets/framework.dart:312`, `:523`, `:771`, `:916`, `:1524`, `:1855`, `:1893`, `:1940`, `:1958`, `:1994`, `:3570`, `rendering/object.dart:2003`, `rendering/box.dart:1572` |
| A named optional parameter, with null for unset | `Container` takes 15 named parameters. `material` declares 2,929 nullable named positions and 607 required ones. It reads `??` 3,744 times. | `widgets/container.dart:255`, `material/theme_data.dart:1484` |
| A const constructor | The subtree skip of `Element.updateChild` fires only on a canonical const widget. A const constructor needs each field final. | `widgets/framework.dart:4027`, `widgets/framework.dart:525` |
| A synchronous frame pipeline | `drawFrame` is `void`, and `flushLayout`, `flushPaint`, and `flushSemantics` are `void`. No frame step awaits. | `rendering/binding.dart:691`, `rendering/object.dart:1137`, `:1239`, `:1293` |
| Mixin linearization | The framework holds 108 mixins, and 38 of them call `super`. Go embedding gives method promotion and no super chain. | `rendering/proxy_box.dart:66`, `rendering/object.dart:4178`, `widgets/ticker_provider.dart:325` |
| A reified generic as a run time key | `Element` keys `_inheritedElements` on `widget.runtimeType`, and it reads the map with the type parameter `T`. | `widgets/framework.dart:5055`, `:5096`, `:6276` |
| A getter and a setter | `material` declares 2,065 getters and 170 setters. `rendering` declares 615 and 304. Each 1 is sugar for a Go method. | `material/text_button.dart:501`, `:590` |
| A named constructor and a factory constructor | `widgets` declares 361 named constructors and 41 factory constructors. `material` declares 385 and 22. Each 1 maps to a Go package function. | `widgets/text.dart:507`, `foundation/key.dart:44` |
| An extension | The whole framework holds 13 extensions and 5 extension types. Only 2 extensions are public. | `rendering/box.dart:980`, `rendering/flex.dart:20` |

The exact contract of each widget class is small. `StatelessWidget` declares
only `Widget build(BuildContext context)` at `widgets/framework.dart:572`.
`StatefulWidget` declares only `State createState()` at
`widgets/framework.dart:800`. `State<T>` declares `build` at
`widgets/framework.dart:1463`, plus `initState`, `didUpdateWidget`,
`didChangeDependencies`, and `dispose`. `RenderObjectWidget` declares
`createRenderObject` at `widgets/framework.dart:1914`. `RenderObject` declares
`performResize` at `rendering/object.dart:2988` and `performLayout` at
`rendering/object.dart:3017`, and `RenderBox` supplies each 1 except
`performLayout`.

2 rules of `Widget` conflict with gart directly. `Widget` carries `@immutable`
at `widgets/framework.dart:311`. Its `operator ==` at
`widgets/framework.dart:365` and its `hashCode` at `widgets/framework.dart:369`
carry `@nonVirtual` and delegate to object identity.

The vsync mixin has an escape hatch. `AnimationController` takes `required
TickerProvider vsync` at `animation/animation_controller.dart:262`, and
`TickerProvider` at `scheduler/ticker.dart:43` is an abstract class with the
single method `createTicker` at `scheduler/ticker.dart:52`. Thus a Go state type
satisfies vsync with 1 method and needs no mixin.

`WidgetsFlutterBinding` at `widgets/binding.dart:2128` is the most mixin
dependent class of the framework. It stays off the path of the user, because
`runApp` calls `ensureInitialized` itself.

The `dart:ui` boundary is finite. `painting`, `rendering`, and `widgets` name 95
distinct `dart:ui` symbols. `painting/basic_types.dart:12` re-exports 42
`dart:ui` value types as the public boundary. About 55 of the 95 names are
immutable value types that map to a Go struct. About 12 are native handles that
must stay opaque: `Canvas`, `Path`, `Paint`, `Picture`, `PictureRecorder`,
`Image`, `Shader`, `Gradient`, `ImageFilter`, `ParagraphBuilder`, `Paragraph`,
and `SceneBuilder`. A `base class` of `dart:ui` accepts no implementation
outside `dart:ui`, thus a Go type never substitutes for a `Shader`.

The public surface of a binding, for a count of the work: `painting` holds 103
public classes and enums, 132 constructors, and 292 public final fields.
`rendering` holds 251, 206, and 238. `widgets` holds 790, 785, and 2,550.
`material` holds 414, 432, and 4,485. `cupertino` holds 92, 116, and 982. The
sum for `painting`, `rendering`, `widgets`, `material`, and `cupertino` is about
1,650 public types, 1,671 constructors, and 8,547 public fields.

## What gart gives today

Each row states 1 of 3 states: present, an emitter change gives it, or out of
reach.

| Item | State | Evidence |
|---|---|---|
| A Dart class for a Go struct | Present | `internal/emit/decl.go:36` |
| A pointer to a struct as a bare Dart object reference | Present | `internal/emit/type.go:33` |
| A struct tag at compile time and at run time | Present | `internal/emit/iface.go:101` |
| A reified type as a run time key, through `TypeDesc` and `Dict` | Present | `internal/emit/iface.go:64`, `internal/emit/decl.go:118` |
| A Dart call of a Go package that is not `main` | Present | `internal/driver/driver.go:438` |
| A build tag for a second target | Present | `internal/load/load.go:18` |
| A Dart superclass, `extends` | An emitter change gives it. `decl.go:36` holds the only class head, and a `final` class still extends. | `internal/emit/decl.go:36` |
| `implements` and `with` | An emitter change gives it. Go embedding gives no choice between the 3 Dart relations, thus the source needs a marker. | `internal/emit/decl.go:36` |
| `@override` on a method | An emitter change gives it. The emitter writes `@override` only for `==` and `hashCode`. | `internal/emit/decl.go:199`, `:206` |
| A final field | An emitter change gives it. | `internal/emit/decl.go:41` |
| A named parameter, an optional parameter, a default value | An emitter change gives it. The parameter list and the composite literal each hold 1 site. | `internal/emit/decl.go:295`, `internal/emit/expr.go:438` |
| A getter and a setter | An emitter change gives it, with a marker for the choice. | `internal/emit/decl.go:208` |
| A Dart handle type in Go | An emitter change gives it. Only `unsafe.Pointer` maps to `Object?`, and stage 2 rejects `unsafe`. | `internal/emit/type.go:64`, `internal/verify/verify.go:123` |
| A Flutter dependency in the pubspec, and a `lib/main.dart` entry point | An emitter change and a driver change give it. | `internal/driver/driver.go:468`, `:438` |
| An import of a Dart library that is not `goruntime` | An emitter change gives it. The library file imports only `goruntime` and the generated Go packages. | `internal/emit/emit.go:213` |
| A super call | Out of reach. No stage of the compiler writes `super`, and the subset holds no Go form for it. | `internal/emit/decl.go:36` |
| A const constructor | Out of reach. It needs each field final, and `$assign` writes a field in place by decision 2. | `internal/emit/decl.go:41`, `:162` |
| A const struct literal | Out of reach. A Go string constant emits `GoString.fromString`, and that call is not const. | `internal/emit/emit.go:326` |
| A Dart subclass of a gart class | Out of reach. A `final` class accepts no subclass outside its library, and 1 Go package is 1 Dart library. | `internal/emit/decl.go:36` |
| A Go interface value as a Dart `Widget` | Out of reach. An interface value is the pair `Iface`, with a `TypeDesc` and an `Object?`. | `runtime/goruntime/lib/src/iface.dart:66`, `internal/emit/type.go:47` |
| A field-wise `==` and `hashCode` on a widget | Present, and Flutter forbids it. The 2 members of `Widget` carry `@nonVirtual`. | `internal/emit/decl.go:191`, `widgets/framework.dart:365` |
| Value semantics of a struct | Present, and it works against the Flutter model. A value receiver opens with `$copy`, and a box into an interface copies again. | `internal/emit/decl.go:227`, `internal/emit/iface.go:310` |
| A guarantee that a function stays synchronous | A new pass after stage 3 gives it. `blockingExports` already reads the block set of each exported function, thus the rule needs no new analysis. The row before the block model change read "out of reach". | `internal/driver/driver.go:274`, `internal/analyze/analyze.go:357` |
| A call of an arbitrary Dart constructor or method | Out of reach. `intrinsic` is a flat switch on the name of an interop function. | `internal/emit/call.go:46` |
| A generic Dart class that a Dart caller constructs | Out of reach. A generic gart class takes a `Dict` parameter, and it gets no type descriptor. | `internal/emit/decl.go:32`, `internal/emit/iface.go:41` |

The repository holds no mention of Flutter and no mention of a widget. Each
option starts from 0.

The async model was the hardest item of the table on 2026-08-28. Stage 5 emitted
`Future<T> f() async` then, and a chain of 20 frames cost 3,900 nanoseconds
against 28.9 for the synchronous chain. A 60 frames per second build tree does
not pay that cost.

The unwind model of 2026-08-29 removed that cost. The output holds no `Future`
and no `await`. A function that can block keeps its Dart return type, and the
extra cost is about 1.6 nanoseconds for each frame. A build path of Go code thus
runs at the speed of the synchronous chain.

1 danger replaces the cost. A function that really blocks writes its frame to
`$blocked` and returns the zero value, and only a Go caller tests that field.
A Dart caller of such a function reads the zero value and sees no report. Thus a
`build` method needs the rule of the row above. `docs/block-model.md` holds the
model, and `docs/flutter-first.md` holds the rule.

The event handler boundary stays safe. A handler gives no result, thus the
adapter starts a goroutine and runs the scheduler: `() { $go(f); $schedule(); }`.
`onPressed` and `onChanged` then carry Go code that blocks. The finalizer of
`sched.dart:188` already holds that pattern.

## The options

### Option 1: A hand written binding through `syscall/dart`

A Go package holds 1 struct per Flutter widget, and a hand written Dart adapter
maps a property bag to a Flutter constructor.

```go
package ui

import "syscall/dart"

// Node is a value that gives a Dart widget handle.
type Node interface {
	Node() dart.Object
}

// Text holds the parameters of the Dart class Text.
type Text struct {
	Data string
	Size float64
}

func (t Text) Node() dart.Object {
	return dart.New("Text",
		dart.Str("data", t.Data),
		dart.Num("fontSize", t.Size))
}

type Center struct {
	Child Node
}

func (c Center) Node() dart.Object {
	return dart.New("Center", dart.Obj("child", c.Child.Node()))
}

func main() {
	dart.RunApp(Center{Child: Text{Data: "hello", Size: 24}}.Node())
}
```

gart emits the Go side with no change of the class shape. It replaces each
interop call with a runtime call.

```dart
final class Text$ {
  GoString Data;
  double Size;

  Text$(this.Data, this.Size);
  Text$.$zero() : Data = GoString.empty, Size = 0;

  Text$ $copy() => Text$(Data, Size);

  Object? Node() {
    final Text$ t = this.$copy();
    return $dartNew(GoString.fromString('Text'), Slice<Object?>.of(<Object?>[
      $dartStr(GoString.fromString('data'), t.Data),
      $dartNum(GoString.fromString('fontSize'), t.Size),
    ]));
  }
}
```

The adapter is hand written Dart in a new runtime package.

```dart
// runtime/goflutter/lib/src/widgets.dart. Hand written, not generated.
Widget $dartNew(GoString name, Slice<Object?> props) {
  final Map<String, Object?> p = $bag(props);
  switch (name.toDartString()) {
    case 'Text':
      return Text(p['data']! as String,
          style: TextStyle(fontSize: p['fontSize'] as double?));
    case 'Center':
      return Center(child: p['child'] as Widget?);
  }
  throw StateError('no widget ${name.toDartString()}');
}
```

The compiler must change: stage 5 `internal/emit/call.go:46` takes 1 case per
interop function, and `internal/overlay/std/syscall/dart/dart.go` takes 1 Go
stub for each 1. Stage 5 `internal/emit/type.go:64` needs a named Go type whose
Dart type is `Object?`, thus a Dart handle enters the subset. Stage 2
`internal/verify/verify.go:46` needs a rule that keeps a handle out of a copy.
The driver `internal/driver/driver.go:468` needs a `flutter:` section in the
pubspec, and `:438` needs a `lib/main.dart` layout.

The developer experience: the Go side reads as a plain struct literal with field
names, which is the closest Go form to a Dart named argument list. It feels like
Go, and each property crosses the boundary as a string key with no type check.

The cost and the risk: Flet gives the price. It holds 120 Dart control files,
40,259 lines of hand written Dart adapter, a 456 line switch, and 758 Python
control classes. The risk is that the behavior stays in Dart, because focus,
disposal, a ticker, and an inherited widget dependency all live in the adapter.

### Option 2: A struct tag transform that turns a Go struct into a widget class

A tag on a blank marker field declares the struct a widget, and the compiler
writes the whole Dart widget class from it.

```go
package app

import "syscall/dart"

type Counter struct {
	_     struct{} `dart:"widget=StatelessWidget"`
	Title string   `dart:"required"`
	Count int      `dart:"default=0"`
	OnTap func()   `dart:"sync"`
}

func (c *Counter) Build(ctx dart.Context) dart.Node {
	return &Column{Children: []dart.Node{
		&Text{Data: c.Title},
		&ElevatedButton{OnPressed: c.OnTap, Child: &Text{Data: "add"}},
	}}
}
```

Go accepts a tag only on a field, thus a blank field carries the tag of the
whole type. The tag reaches the compiler through `types.Struct.Tag(i)` and the
run time through `FieldDesc`.

```dart
final class Counter extends StatelessWidget {
  final GoString Title;
  final int Count;
  final void Function()? OnTap;

  const Counter({required this.Title, this.Count = 0, this.OnTap, super.key});

  @override
  Widget build(BuildContext context) {
    final Counter c = this;
    return Column(children: <Widget>[
      Text(c.Title.toDartString()),
      ElevatedButton(onPressed: c.OnTap, child: const Text('add')),
    ]);
  }
}
```

The class holds no `$copy`, no `$assign`, no `operator ==`, and no `hashCode`.

The compiler must change in stage 2, stage 3, and stage 5. Stage 5
`internal/emit/decl.go:36` writes the class head with `extends`. Line 41 writes
a final field. Line 49 writes a const named constructor with `required` and a
default. Line 191 drops the equality operations, and line 218 writes `@override`
and the Dart method name. Stage 5 `internal/emit/decl.go:295` and
`internal/emit/expr.go:438` map a field name to a named argument. Stage 2
`internal/verify/verify.go:46` needs 3 new rules. It must reject the address of
a widget struct, and a write to a field of it. It must also reject a call that
blocks and that a `Build` method reaches. Stage 3
`internal/analyze/analyze.go:357` must mark a `Build` method as a
must-stay-synchronous root. `internal/emit/emit.go:326` still blocks a const
literal that holds a Go string constant.

The developer experience: the developer writes 1 Go struct and 1 method, and the
field-keyed literal reads like the Dart named argument list. It feels like Go,
and the tag grammar is a second small language that the developer must learn.

The cost and the risk: the emitter needs a second class shape, and each change
sits at 1 site. The risk is a split of the struct model, because a widget struct
loses `$assign` and the value semantics of decision 2.

### Option 3: A code generator that reads the Go AST and writes Dart widget classes

A second tool reads the Go AST and writes 1 Dart widget class per marked Go
type, and gart itself does not change.

```go
package app

import "syscall/dart"

//gart:widget stateless
type Greeting struct {
	Name string
}

func (g *Greeting) Build(ctx dart.Context) dart.Node {
	return &Center{Child: &Text{Data: "hello " + g.Name}}
}
```

The tool writes a separate Dart file that imports the generated Go library and
forwards each call.

```dart
// GENERATED by gartwidget. Do not edit.
import 'package:flutter/widgets.dart';
import 'package:goruntime/goruntime.dart';
import 'package:app/app.dart' as go;

class Greeting extends StatelessWidget {
  const Greeting({required this.name, super.key});

  final String name;

  @override
  Widget build(BuildContext context) {
    final go.Greeting$ g = go.Greeting$(GoString.fromString(name));
    return $node(g.Build($ctx(context)));
  }
}
```

The compiler must change: no stage changes. The tool reads `internal/meta` or
the Go AST, and the driver `internal/driver/driver.go:438` must write the extra
file into the output package. The same tool runs in the other direction and
reads the Flutter public API to write the Go binding structs of option 1.

The developer experience: the developer writes plain Go and 1 comment directive,
and never reads the generated Dart. The Go side feels like Go, and a fix of the
Dart output needs a change of the generator.

The cost and the risk: the generator writes the 1,650 public types and 8,547
public fields of the binding, and jaspr proves that form with 12,072 lines. The
risk is a 2 hop cost per widget, and the generator must track each Flutter
release.

### Option 4: An emitter mode that gives a Go type a Dart superclass

An embedded Go mirror type names the Dart superclass, and the emitter writes
`extends` and `@override` for the whole class.

```go
package ui

import "flutter/rendering"

// Gauge draws an arc. The embedded type names the Dart superclass.
type Gauge struct {
	rendering.RenderBox

	Value float64
}

func (g *Gauge) PerformLayout() {
	g.SetSize(g.Constraints().Biggest())
}

func (g *Gauge) Paint(ctx rendering.PaintingContext, off rendering.Offset) {
	ctx.Canvas().DrawArc(g.Size().Rect(), 0, g.Value, false, g.Brush())
}
```

```dart
final class Gauge extends RenderBox {
  double Value;

  Gauge(this.Value);

  @override
  void performLayout() {
    final Gauge g = this;
    g.size = g.constraints.biggest;
  }

  @override
  void paint(PaintingContext ctx, Offset off) {
    final Gauge g = this;
    ctx.canvas.drawArc(g.size.rect(), 0, g.Value, false, g.brush());
  }
}
```

The compiler must change: stage 5 `internal/emit/decl.go:36` writes the
superclass, and `:218` writes `@override` plus the Dart name of the method.
Stage 5 `internal/name/name.go` needs a map from a Go method name to a Dart
member name. Stage 5 `internal/emit/emit.go:213` needs an import of a Dart
package URI. The embedded mirror type needs 1 Go declaration per inherited
member, thus this option needs option 1 or option 3 under it. A super call has
no Go form, thus `dart.Super` becomes a new intrinsic in
`internal/emit/call.go:46`. The marker must also select 1 of the 3 Dart
relations: `extends`, `implements`, or `with`.

The developer experience: the developer writes Go embedding, which is the
natural Go form, and gets a real Dart subclass. It feels like Go until an
inherited member has no Go mirror, and then the error names a Dart class.

The cost and the risk: this option is the only 1 that reaches a custom
`RenderBox`, and Material needs exactly that in 48 places. The risk is that Go
embedding maps to 3 Dart relations, and that 38 mixins need a super chain with
no Go form.

### Option 5: A Go native widget tree that the runtime maps to Flutter elements

Go declares its own widget interface, and 1 hand written Dart host widget walks
the Go tree and builds real Flutter widgets from it.

```go
package main

import "flutter/ui"

type Counter struct {
	n int
}

func (c *Counter) Build(ctx ui.Context) ui.Node {
	return ui.Column{Children: []ui.Node{
		ui.Text{Data: fmt.Sprintf("count %d", c.n)},
		ui.Button{Label: "add", Tap: func() {
			c.n++
			ctx.Dirty()
		}},
	}}
}

func main() {
	ui.Run(&Counter{})
}
```

The Go tree reaches Dart as an `Iface` pair, and the host reads the name of the
`TypeDesc`.

```dart
class GoHost extends StatefulWidget {
  const GoHost({super.key});

  @override
  State<GoHost> createState() => _GoHostState();
}

class _GoHostState extends State<GoHost> {
  @override
  Widget build(BuildContext context) => _map(go.$rootNode());
}

Widget _map(Iface? node) {
  switch (node!.desc.name) {
    case 'ui.Column':
      final go.Column$ c = node.value! as go.Column$;
      return Column(children: <Widget>[
        for (int i = 0; i < c.Children.length; i++) _map(c.Children[i]),
      ]);
    case 'ui.Text':
      return Text((node.value! as go.Text$).Data.toDartString());
  }
  throw StateError('no node ${node.desc.name}');
}
```

The compiler must change: very little. Stage 5 needs the handle type of
`internal/emit/type.go:64` and 3 intrinsics in `internal/emit/call.go:46` for
`Run`, `Dirty`, and the root read. Stage 3 `internal/analyze/analyze.go:357`
still needs the rule that keeps `Build` synchronous, because the host calls it
inside a Dart `build`.

The developer experience: the Go side is pure Go with no tag, no directive, and
no Dart name, and the tree is a plain struct literal. It feels the most like Go
of each option, and the developer never reads a Dart class.

The cost and the risk: the `_map` switch is hand written, and it grows with the
widget set. The risk is the ceiling that the Flutter team states for `rfw`: a
data tree uses a prebuilt widget and it creates none.

The `rfw` README names the failures: a page transition, a drag and drop, and a
custom painter. The 456 line switch of Flet is the measure of such an adapter. A
second shape of the same option escapes part of the ceiling. jaspr proves that a
Flutter-shaped reconciler is 4,086 code lines, and that a 7 method
`RenderObject` boundary hosts a whole component tree. Thus the Go tree drives 1
custom Dart `RenderObject` through a thin leaf, and Go owns the layout and the
paint.

### Option 6: A full port of the widgets layer to Go

Go owns the whole hierarchy. `painting`, `rendering`, and `widgets` become Go
packages, and the only Dart contact is the `dart:ui` handle set.

```go
package widgets

type Widget interface {
	Key() Key
	CreateElement() Element
}

type Stateless interface {
	Widget
	Build(ctx BuildContext) Widget
}

type Element interface {
	Mount(parent Element, slot any)
	Update(w Widget)
	Rebuild()
}
```

```go
package app

type Greeting struct {
	widgets.Base

	Name string
}

func (g *Greeting) Build(ctx widgets.BuildContext) widgets.Widget {
	return &widgets.Center{Child: &widgets.Text{Data: "hello " + g.Name}}
}
```

gart emits the Go widget classes with no change at all. The leaf reaches
`dart:ui` through the intrinsic table, and 1 hand written Dart file binds the
frame callback of the engine to the Go pipeline.

```dart
// The Go render tree owns the frame. 1 hand written file binds the engine.
void main() {
  go.$init();
  PlatformDispatcher.instance.onDrawFrame = () => go.DrawFrame();
  PlatformDispatcher.instance.onPointerDataPacket = (ui.PointerDataPacket p) =>
      go.HandlePointer($pointerBytes(p));
  PlatformDispatcher.instance.scheduleFrame();
}

// A paint call of the Go rendering layer reaches a dart:ui handle.
void $drawRect(Object? canvas, double l, double t, double r, double b,
        Object? paint) =>
    (canvas! as ui.Canvas).drawRect(ui.Rect.fromLTRB(l, t, r, b), paint! as ui.Paint);
```

The compiler must change: stage 3 `internal/analyze/analyze.go:357` needs the
synchronous rule for the frame path. Stage 5 `internal/emit/call.go:46` needs
about 95 intrinsics for the `dart:ui` names, and 12 of them stay opaque handles.
`internal/overlay/std/syscall/dart/dart.go` needs the same count of Go stubs.
Stage 5 needs no `extends`, no `const`, no named parameter, and no mixin,
because Go declares each type.

The developer experience: each API is a Go interface, thus the whole stack reads
as Go, and no Dart name appears in a Go file. It feels the most like Go of each
option, and the developer gives up each Flutter package of pub.dev.

The cost and the risk: the port is 127,019 code lines of base, plus 88,619 for
Material and 24,140 for Cupertino. The risk is the ratio: jaspr shows a
reconciler is 4,086 lines, and Material is 176,201 lines.

## A comparison table

| Option | Developer experience | Compiler work | Risk | Code size of the output | Material with no port |
|---|---|---|---|---|---|
| 1. A hand written binding | Go struct literals, a string key per property, no type check across the boundary | Small. 1 case per call in `call.go:46`, 1 handle type in `type.go:64` | Medium. The behavior stays in the Dart adapter | About 40,000 hand written Dart lines for a Flet-size surface | Yes, for each widget that the adapter covers |
| 2. A struct tag transform | 1 Go struct and 1 `Build` method, plus a tag grammar to learn | Medium. 7 emitter sites in `decl.go` and `expr.go`, 3 rules in `verify.go`, 1 root rule in `analyze.go` | High. A second struct model, and a Go string constant blocks `const` | About the size of the Go source, 1 Dart class per Go widget | Yes, as a consumer of Material |
| 3. A code generator | Plain Go plus 1 comment directive, and no Dart to read | Small. No stage changes, 1 new tool, 1 driver change | Medium. A 2 hop cost per widget, and each Flutter release needs a run | The Go source plus the generated binding, about 12,000 lines for a jaspr-size tag library | Yes, and the generator writes the binding |
| 4. An emitter superclass mode | Go embedding gives a real Dart subclass, and an inherited member needs a Go mirror | Medium. `decl.go:36` and `:218`, `name.go`, `emit.go:213`, plus a `Super` intrinsic | High. Go embedding maps to 3 Dart relations, and 38 mixins need a super chain | The Go source, plus the mirror declarations of each base class | Yes, and it also reaches a custom `RenderBox` |
| 5. A Go native tree | Pure Go, no tag and no Dart name, a plain struct literal tree | Very small. 1 handle type and 3 intrinsics | High. The `rfw` ceiling: Go arranges a widget and it does not create 1 | The Go source plus a hand written map switch that grows with the widget set | Yes, for each widget that the switch covers |
| 6. A full port | Each API is a Go interface, and the whole stack reads as Go | Small for the emitter. 95 `dart:ui` intrinsics and 1 synchronous rule | Very high. 127,019 code lines of base, and no pub.dev package still works | 127,019 code lines of base, plus 88,619 for Material | No. Material needs the full 88,619 line port |

## The open questions

- Which option does the project take? The user decides.
- How does a Go widget field state "not set"? Material reads `??` 3,744 times,
  and the Go zero value carries no such state. The 3 candidates are a pointer
  field, a presence bitmask per struct, and a generic wrapper type `Opt[T]`.
- How does the compiler prove that a `Build` method never blocks? Stage 3
  spreads the async state per signature, thus 1 `Build` method that blocks makes
  each `Build` call async. The rule belongs in stage 2 or stage 3, and it needs
  a definition of a widget entry point.
- Does a widget struct keep the value semantics of decision 2? A const
  constructor needs a final field, and `$assign` needs a mutable field. The 2
  rules do not hold at the same time.
- How does a Go string constant reach a `const` Dart expression? A Go string
  literal emits `GoString.fromString`, and that call is not const. A widget mode
  may map `string` to a Dart `String`, and the workstream rejects that map as a
  silent change of meaning.
- Which Dart relation does Go embedding select? Dart holds `extends`,
  `implements`, and `with`, and Go holds 1 embedding form. Dotlin solved it with
  a marker on the supertype, and gart needs the same decision.
- How does a Go type reach the 38 mixins that call `super`? Go embedding gives
  method promotion and no super chain. The vsync mixin has an escape hatch
  through `TickerProvider`, and the other 37 do not.
- Does the widget class suppress `writeEqual`? Flutter marks `Widget.operator
  ==` as `@nonVirtual`, and `dart analyze --fatal-infos` fails the CI step on
  the override. The field-wise `==` also makes `Element.updateChild` skip a real
  update.
- How does a Go widget carry a Dart getter? `material` declares 2,065 getters,
  and Go has no syntax to mark a zero argument method as a getter. The 3
  candidates are a struct tag, a naming rule, and a build tag.
- How does a widget tree of 5 levels read in Go? ClojureDart needed a dedicated
  `nest` macro for the `child` chain, and Go 1.27 does not elide the type of an
  element of a `[]Widget` literal.
- Does the descriptor overhead matter for a large Go widget library? The emitter
  writes a `TypeDesc` pair per named type, and a probe measured 3,233 bytes of
  descriptor against 3,863 bytes of code. `$init` also retains each global of an
  imported package.
- Which Go UI shape does the project follow? gio uses a config struct with a
  `Layout` method and closure children. go-app uses an embedded base struct, a
  `Render` method, and builder chains. Fyne uses a retained object with setters,
  and that is the most adopted Go form.
- Does the project need a `flutter` build tag? `internal/load/load.go:18` holds
  the `dart` tag, and a second constant gives a Go package a headless build and
  a Flutter build of the same source.

## Facts

Each fact below carries its evidence. The measurements ran on 2026-08-28.

### The Flutter SDK

The root is `/path/to/flutter`. The framework source is
`packages/flutter/lib/src`. The whole package holds 682 files, 567,179 lines,
and 328,237 code lines.

| Fact | Evidence |
|---|---|
| The layer code line counts of the table above | `find <layer> -name '*.dart' -exec cat {} + \| grep -vcE '^\s*(//\|$)'` over `packages/flutter/lib/src` |
| `material/icons.dart` holds 11,658 code lines of constants | `material/icons.dart:1`, 29,454 raw lines |
| `material/animated_icons/data/` holds 34,066 code lines in 14 files | `material/animated_icons/data/` |
| `cupertino/icons.dart` holds 6,372 code lines | `cupertino/icons.dart:1` |
| Material and Cupertino make no platform channel call and use no `dart:ffi` | `grep -rn "SystemChannels.\|MethodChannel(\|dart:ffi" material/*.dart cupertino/*.dart` gives no output |
| The 3 engine reaches of Material | `material/ink_sparkle.dart:424`, `material/about.dart:12`, `material/color_scheme.dart:2133` |
| Material defines 33 `RenderObject` subclasses and Cupertino 15 | `material/chip.dart:1730`, `material/input_decorator.dart:722`, `material/app_bar.dart:2206` |
| Cupertino subclasses the element tree | `cupertino/text_selection_toolbar.dart:792` |
| `Widget` carries `@immutable` | `widgets/framework.dart:311` |
| `Widget` is an abstract class | `widgets/framework.dart:312` |
| `Widget.operator ==` carries `@nonVirtual` | `widgets/framework.dart:365` |
| `Widget.hashCode` carries `@nonVirtual` | `widgets/framework.dart:369` |
| `Widget.canUpdate` reads `runtimeType` and `key` | `widgets/framework.dart:382` |
| An override of `==` on a widget with children is an O(N²) tree walk | `widgets/framework.dart:3581` |
| `Element.operator ==` is `identical(this, other)` and `@nonVirtual` | `widgets/framework.dart:3599` |
| `Element.updateChild` skips the update when `child.widget == newWidget` | `widgets/framework.dart:4027` |
| `StatelessWidget` is abstract, with a const constructor | `widgets/framework.dart:523`, `:525` |
| `StatelessWidget.build` is the only abstract member | `widgets/framework.dart:572` |
| `StatefulWidget` is abstract | `widgets/framework.dart:771` |
| `StatefulWidget.createState` is the only abstract member | `widgets/framework.dart:800` |
| `State<T extends StatefulWidget>` uses `Diagnosticable` | `widgets/framework.dart:916` |
| `State.build` is abstract | `widgets/framework.dart:1463` |
| `ProxyWidget`, `InheritedWidget`, `RenderObjectWidget` are abstract | `widgets/framework.dart:1524`, `:1855`, `:1893` |
| `LeafRenderObjectWidget`, `SingleChildRenderObjectWidget`, `MultiChildRenderObjectWidget` | `widgets/framework.dart:1940`, `:1958`, `:1994` |
| `RenderObjectWidget.createRenderObject` is abstract | `widgets/framework.dart:1914` |
| `Element` implements `BuildContext` | `widgets/framework.dart:3570` |
| `Element` keys the inherited map on `widget.runtimeType` | `widgets/framework.dart:5055`, `:5096` |
| `findAncestorStateOfType` reads `ancestor.state is T` | `widgets/framework.dart:6276` |
| `RenderObject` uses `DiagnosticableTreeMixin` | `rendering/object.dart:2003` |
| `RenderObject.performResize` is abstract | `rendering/object.dart:2988` |
| `RenderObject.performLayout` is abstract | `rendering/object.dart:3017` |
| `RenderBox` is abstract | `rendering/box.dart:1572` |
| `RenderBox.performLayout` has a body | `rendering/box.dart:2876` |
| `drawFrame` is `void` and it holds no `await` | `rendering/binding.dart:691` |
| `RenderProxyBoxMixin` holds 2 `on` constraints | `rendering/proxy_box.dart:66` |
| `ContainerRenderObjectMixin` calls `super` | `rendering/object.dart:4178` |
| `TickerProviderStateMixin` calls `super` | `widgets/ticker_provider.dart:325` |
| `TickerProvider` is an abstract class with 1 method | `scheduler/ticker.dart:43`, `:52` |
| `AnimationController` takes `required TickerProvider vsync` | `animation/animation_controller.dart:262` |
| `WidgetsFlutterBinding` linearizes 7 mixins | `widgets/binding.dart:2128` |
| `Container` takes 15 named parameters | `widgets/container.dart:255` |
| `Text` takes 1 positional and 16 named parameters | `widgets/text.dart:507` |
| `ThemeData.copyWith` takes about 100 nullable named parameters | `material/theme_data.dart:1484` |
| `Key` holds a const redirect factory | `foundation/key.dart:44` |
| `painting` re-exports 42 `dart:ui` value types | `painting/basic_types.dart:12` |
| A private layout extension type | `rendering/box.dart:980`, `rendering/flex.dart:20` |
| `material` overrides 16 getters in 1 defaults class | `material/text_button.dart:501`, `:590` |
| `dart:ui` holds 19 files, 25,724 lines, and 230 external members | `bin/cache/pkg/sky_engine/lib/ui/` |
| A `dart:ui` value type has a const constructor | `bin/cache/pkg/sky_engine/lib/ui/geometry.dart:9`, `:115`, `:346` |

### The gart source

The repository is `/path/to/godart`.

| Fact | Evidence |
|---|---|
| The emitter writes 1 class head, `final class X {` | `internal/emit/decl.go:36` |
| A struct field is mutable | `internal/emit/decl.go:41` |
| The constructor is positional and not const | `internal/emit/decl.go:49` |
| A generic class carries a `Dict` field and a `Dict` parameter | `internal/emit/decl.go:32`, `:44` |
| The bound of a type parameter comes from a Go type set | `internal/emit/decl.go:91`, `internal/emit/type.go:161` |
| `dictParams` writes 1 dictionary parameter per type parameter | `internal/emit/decl.go:118` |
| `$assign` writes a field in place | `internal/emit/decl.go:162` |
| `writeEqual` writes a field-wise `==` and `hashCode` | `internal/emit/decl.go:191`, `:199`, `:206`, `:210` |
| The 2 `@override` lines of the whole emitter | `internal/emit/decl.go:199`, `:206` |
| The method head carries no annotation | `internal/emit/decl.go:218` |
| A value receiver opens with `$copy`, and a pointer receiver binds `this` | `internal/emit/decl.go:227` |
| `paramList` writes a positional parameter with no default | `internal/emit/decl.go:295` |
| `asyncWord` writes ` async` | `internal/emit/decl.go:303` |
| `resultText` wraps the result in `Future<...>` | `internal/emit/decl.go:316` |
| A pointer to a struct is the bare Dart reference | `internal/emit/type.go:33` |
| A func type is nullable, and `func()` gives `void Function()?` | `internal/emit/type.go:43` |
| An interface type gives `Iface?` | `internal/emit/type.go:47` |
| A Go `string` gives `GoString` | `internal/emit/type.go:60` |
| Only `unsafe.Pointer` gives `Object?` | `internal/emit/type.go:64` |
| A signature that blocks gives `FutureOr<R> Function(...)` | `internal/emit/type.go:81` |
| A composite literal writes positional arguments | `internal/emit/expr.go:438` |
| A string constant uses `final` and not `const` | `internal/emit/emit.go:326` |
| A library file imports only `goruntime` and the generated Go packages | `internal/emit/emit.go:213`, `:217` |
| `writeDescriptors` skips a generic named type | `internal/emit/iface.go:41` |
| The `TypeDesc` line holds the name, the kind, and the method table | `internal/emit/iface.go:64` |
| A `FieldDesc` carries the raw Go struct tag | `internal/emit/iface.go:101` |
| A box into an interface copies the struct | `internal/emit/iface.go:310` |
| `intrinsic` is a flat switch on the interop function name | `internal/emit/call.go:46` |
| The Go stubs of the interop package all panic | `internal/overlay/std/syscall/dart/dart.go:1` |
| `blockingIntrinsic` names each interop call that gives up control | `internal/analyze/analyze.go:291` |
| `spread` marks a whole signature group async | `internal/analyze/analyze.go:357` |
| `signatureKey` excludes the receiver | `internal/analyze/analyze.go:409` |
| The verifier holds no rule about a call that blocks in a synchronous position | `internal/verify/verify.go:46` |
| Stage 2 rejects an import of `unsafe` | `internal/verify/verify.go:123` |
| `libraryURI` gives no URI for a main package | `internal/driver/driver.go:438` |
| The pubspec holds 1 dependency and no `flutter:` section | `internal/driver/driver.go:468`, `:474` |
| The build tag of the load is `dart` | `internal/load/load.go:18`, `:81` |
| `Iface` holds a `TypeDesc` and an `Object?` | `runtime/goruntime/lib/src/iface.dart:66` |
| `TypeDesc` holds the name, the kind, the method table, and the fields | `runtime/goruntime/lib/src/iface.dart:28` |
| `FieldDesc` holds the name, the tag, and the exported flag | `runtime/goruntime/lib/src/iface.dart:5` |
| The emitted class of a Go struct, in a real golden file | `testdata/golden/multi/lib/greet/greet.part.dart:12` |
| A Dart caller reads an exported Go function | `testdata/golden/multi/lib/greet/greet.part.dart:47` |
| An async chain of 20 frames costs 3,900 nanoseconds against 28.9 synchronous | `docs/parity.md:194` |
| No file of the repository mentions Flutter or a widget | `grep -rni "flutter\|widget"` over each `.go`, `.md`, and `.dart` file gives no hit |

### The verified experiments

| Fact | Evidence |
|---|---|
| A cross-library `final class G extends F` fails with `invalid_use_of_type_outside_library`. Same-library extension is legal. | `dart analyze` on a 2 library reproduction |
| A const constructor with a mutable field fails with `const_constructor_with_non_final_field` | `dart analyze` on `class M { const M(this.a); int a; }` |
| 2 `const W(1)` values are identical, and 2 `W(1)` values are not | `dart run` on a reproduction with `identical` |
| A gart-shaped class that extends `StatelessWidget` compiles and mounts, with 3 warnings and 0 errors: `must_be_immutable`, and `invalid_override_of_non_virtual_member` 2 times | `flutter test` in `/tmp/wtest` with the Flutter SDK of the machine |
| Dart accepts an async closure for a `void Function()` parameter | `dart analyze` and `dart run` on a `typedef VoidCallback = void Function();` reproduction |
| A Go `Build` method that blocks makes the interface dispatch `Future<Iface?> _render(Iface? w) async`, and it casts the method table entry to `FutureOr<Iface?> Function(Object?, int)` | A `gart build` probe of a Go widget whose `Build` reads a channel |
| Dart augmentations are not available in Dart 3.13.0 stable. `augment class` gives a parse error with and without `--enable-experiment=augmentations`. | `dart analyze` in a reproduction with a part file |
| Dart macros are cancelled and will not ship | `https://dart.dev/language/macros` |
| Go 1.27 does not add an untyped composite literal. Elision covers only an array, a slice, and a map with an identical element type. | `/usr/local/go/doc/go_spec.html:8842`, `:3374` |

### The prior art

| Project | Shape | State | Number |
|---|---|---|---|
| flutter_rust_bridge | An FFI bridge. Rust writes plain functions and structs, and Dart writes each widget. Its own `#[frb(ui_state)]` still leaves the tree in Dart. | Active, 5,386 stars | A sync mode exists only because `build` cannot await |
| flutter-go-bridge | The Go analogue of the above, across FFI, on the Flutter master channel | Stalled, 103 stars, last push 2024-05-30 | Functions only |
| go-flutter | An engine embedder. Go writes a plugin, and Dart writes each widget. | Dead. The last functional commit is 2022-07-24. | 5,931 stars |
| Flet | A protocol binding for Python. A hand written Dart adapter per control. | Active, 16,600 stars | 120 Dart control files, 40,259 Dart lines, a 456 line switch, 758 Python control classes |
| rfw | The first-party data-driven widget tree of Flutter | Active | Its README states that it uses a widget and it does not create 1 |
| RemoteFlutterSharp | A C# DSL over `rfw` | Archived 2026-01-31, a proof of concept | The only C# attempt |
| ClojureDart | Compile to Dart with real subclasses. `reify :extends`, `^super`, `^:mixin`, `^:const`, and a `widget` macro that replaces `StatelessWidget`, `StatefulWidget`, `State`, `Builder`, and `StatefulBuilder`. | Active, 1,636 stars, self-described production ready | The only prior art in which a second language authors a Flutter widget |
| Dotlin | Kotlin to Dart. A marker on the supertype selects `implements` or `with`. Relaxed const. | Archived 2024-03-06, 221 stars | No Flutter story |
| ts2dart | TypeScript to Dart, built for AngularDart | Archived under `dart-archive` | The 1 industrial compile-to-Dart effort |
| jaspr | A reimplementation of the Flutter framework shape for the web | Active, 2,449 stars, a Google Open Source blog post | The reconciler is 4,086 code lines in 17 files. The HTML tag library is 12,072 generated lines. The `RenderObject` boundary holds 7 methods. |
| gio | An immediate mode Go UI. A config struct literal, 1 `Layout` method, and closure children. | Active, 2,237 stars | It needs no key, no element reuse, and no equality |
| go-app | A declarative Go UI. An embedded `app.Compo`, a `Render` method, builder chains, and `app.If`. | Active, 8,956 stars | The closest Go shape to a Flutter API |
| Fyne | A retained mode Go UI with positional constructors and setters | Active, 28,644 stars | The most adopted Go UI, and the opposite model |
| Vugu | A template file plus a generated Go `Build` method | Experimental, 5,003 stars | A second file format and a second toolchain step |
| Compose Multiplatform | A vendor-scale reimplementation of Material 3 in Kotlin, with a compiler plugin that rewrites each `@Composable` signature | Active | A funded platform team over years |

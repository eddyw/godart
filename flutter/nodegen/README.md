# nodegen

nodegen reads the public API of Flutter with `package:analyzer`, and it writes
the node set of milestone 3. It needs no change of the compiler.

## The 2 outputs

| Output | Scope | Command |
|---|---|---|
| `flutter/ui/widgets_gen.go` | The whole project. 1 Go struct for each Flutter widget constructor. | `nodegen go` |
| `<app>/lib/widgets_gen.dart` | 1 application. 1 case for each node type that the Go source of the application names. | `nodegen host` |

```sh
cd flutter/nodegen
dart pub get
dart run bin/nodegen.dart go   ../goflutter ../ui/widgets_gen.go   # runs gofmt
dart run bin/nodegen.dart host ../goflutter ../example/counter/lib/widgets_gen.dart ../example/counter/go
```

The first argument names a directory that holds a Flutter application with a
`.dart_tool/package_config.json`. `flutter/goflutter` is such a directory, thus
the tool reads the Flutter SDK of that package. The read takes about 5 seconds.

`build.sh` of an application runs the host mode after `gart build`.

## Why the host is per application

A shared host would hold a case for each of the 427 node types, thus it would
name each Flutter widget. The tree shaker of Dart then keeps the whole
framework. The measurement of 2026-09-02 read a set of 411 nodes, on an ahead
of time build for macOS:

| The host | The snapshot |
|---|---|
| The hand written set of milestone 1, 28 nodes | 13,989,280 bytes |
| The generated set, a shared host of 411 cases | 20,844,352 bytes |
| The generated set, a host of the 18 cases that the counter uses | 13,628,480 bytes |

The Go node set costs nothing that the application does not use. The shaker
drops each Dart class and each type descriptor that no code reaches. The whole
cost of 7.2 megabytes comes from the switch of the host.

`usedNodes` of `bin/nodegen.dart` reads each `ui.<Name>` of the Go source of the
application. The set is a superset of the node types that the application
builds. A name that the scan misses gives a clear error at run time, and never a
wrong widget.

## The mapping

`lib/read.dart` holds the rules. A widget reaches the node set when each
parameter that its constructor demands maps to a Go type.

| Dart | Go | Note |
|---|---|---|
| `String`, `double`, `int`, `bool` | `string`, `float64`, `int`, `bool` | |
| `Color` | `ui.Color` | A named type over `uint32`. |
| `IconData` | `string` | The host reads `goIcons`. |
| `Duration` | `int` | Milliseconds. |
| `EdgeInsets`, `EdgeInsetsGeometry` | `ui.Insets` | |
| `TextStyle` | `ui.TextStyle` | A subset of the fields. |
| `BoxConstraints` | `ui.Constraints` | A maximum of 0 means no limit. |
| `Offset` | `ui.Offset` | |
| `Alignment`, `AlignmentGeometry`, `Curve` | `string` | A name of the table of `convert.dart`. |
| An enum | `string` | The name of the value. |
| `Widget` | `ui.Node` | |
| A subtype of `Widget`, for example `PreferredSizeWidget` | `ui.Node` | The host casts the result. |
| `List<Widget>`, `List<String>` | `[]ui.Node`, `[]string` | |
| `void Function()` | `func()` | The host runs it as a goroutine. |
| `void Function(T)` for a mapped scalar or enum | `func(T)` | |
| `Widget Function(BuildContext)` | `ui.Builder` | A synchronous position. Read decision 221. |
| `Widget Function(BuildContext, int)` | `ui.ItemBuilder` | A synchronous position. |
| `MouseCursor` | `string` | A name of the table of `convert.dart`. |
| `ScrollPhysics` | `string` | A name: `bouncing`, `clamping`, `never`, `always`, `page`, `range`, `fixedExtent`. |
| `Locale` | `string` | A language tag, for example `en` or `en_US`. |
| `BorderSide` | `ui.BorderSide` | |
| `BorderRadius`, `BorderRadiusGeometry` | `ui.Radius` | The 4 corners. |
| `ShapeBorder`, `OutlinedBorder` | `ui.Shape` | The host gives an `OutlinedBorder`. |
| `VisualDensity` | `ui.Density` | |
| `IconThemeData` | `ui.IconStyle` | The name differs, because a widget holds the name `IconTheme`. |
| `WidgetStateProperty<Color?>` | `ui.StateColor` | 1 color for each state. |
| `Size` | `ui.Size` | A width and a height. |
| `ButtonStyle` | `ui.ButtonStyle` | A color for each state, and 1 value of each other field. |
| `ImageProvider` | `ui.ImageSource` | A kind and a source: `network`, `asset`, or `memory`. |
| `Decoration`, `BoxDecoration` | `ui.Decoration` | The host gives a `BoxDecoration`. |
| `FocusNode` | `ui.FocusKey` | The host owns the object, by the name of the key. |
| `ScrollController` | `ui.ScrollKey` | The host owns the object, by the name of the key. |
| `WidgetStatesController` | `ui.StatesKey` | The host owns the object, by the name of the key. |

A parameter that the constructor does not demand takes a pointer field, and a
nil pointer means "the default of Flutter". A parameter of any other type does
not reach the node set.

The last 3 rows hold an identity, and not a value. A Go tree is data and it
rebuilds on each change, thus a Go value cannot be the object itself: a new
tree would give a new controller and the widget would lose its focus, its
offset, or its pressed state. The Go program names the object with a key, and
`GoIdentities` of `goflutter` keeps it across each rebuild. Read decision 247.

A `FileImage` stands outside `ui.ImageSource`, because it reads a `File` of
`dart:io`, and the host also builds for the web. A Go program reads the file
and gives the bytes under the kind `memory`.

A named constructor takes a Go struct of its own. `ListView.builder` gives
`ui.ListViewBuilder`, and the host switches on the name of the type descriptor.

## What the tool drops

Run the report to read the numbers of the current Flutter SDK:

```sh
dart run bin/report.dart ../goflutter
```

It names each widget that the reader dropped, and it counts each Dart type that
no Go type carries.

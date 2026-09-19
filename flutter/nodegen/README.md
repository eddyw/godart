# nodegen

nodegen reads the public API of Flutter with `package:analyzer`, and it writes
the node set of milestone 3. It needs no change of the compiler.

## The packs

A pack holds the node types of 1 Dart package. Flutter moved Material and
Cupertino out of the framework, thus each 1 stands as a package of pub.dev.

| Pack | The Dart package | The Go package | Node types |
|---|---|---|---|
| The core | `package:flutter/widgets.dart` | `gart.dev/ui` | 274 |
| Material | `package:material_ui/material_ui.dart` | `gart.dev/ui/material` | 217 |
| Cupertino | `package:cupertino_ui/cupertino_ui.dart` | `gart.dev/ui/cupertino` | 72 |
| SVG | `package:flutter_svg/flutter_svg.dart` | `gart.dev/ui/svg` | 4 |

`packs` of `lib/read.dart` holds the list. The core package declares the
interface `ui.Node`, and each pack gives its structs the method `IsNode` that
answers `ui.Mark`. A struct of another module joins the node set the same way,
thus a package of pub.dev such as `macos_ui` needs no change of the reader.

The host imports the Dart package of each case that it holds, and no other 1.
An application that names no Material widget thus needs no `material_ui`.

## The 3 modes

| Output | Scope | Command |
|---|---|---|
| `flutter/ui/**/widgets_gen.go` | The whole project. 1 Go struct for each Flutter widget constructor, in the package of its pack. | `nodegen go` |
| `<app>/lib/widgets_gen.dart` | 1 application. 1 case for each node type that the application names. | `nodegen host` |
| The yield of the reader | A report on the terminal. | `nodegen report` |

```sh
cd flutter/nodegen
dart pub get
dart run bin/nodegen.dart go     ../goflutter ../ui   # 1 file for each pack, and it runs gofmt
dart run bin/nodegen.dart report ../goflutter -all
dart run bin/nodegen.dart host   ../goflutter /path/to/app/lib/widgets_gen.dart nodes.txt
```

The first argument names a directory that holds a Flutter application with a
`.dart_tool/package_config.json`. `flutter/goflutter` is such a directory, thus
the tool reads the Flutter SDK of that package. The read takes about 6 seconds.

The last argument of the host mode names a file that holds 1 node type for each
line. A single dash reads that list from the standard input, and `-all` writes
a case for each node type of the set. `-all` gives the whole host, thus
`flutter analyze` of `goflutter` then proves that each case of the set
compiles:

```sh
dart run bin/nodegen.dart host ../goflutter ../goflutter/lib/all_gen_check.dart -all
cd ../goflutter && flutter analyze && rm lib/all_gen_check.dart
```

## Where the name list comes from

`flutter/nodescan` is a Go command that reads the application module with
`golang.org/x/tools/go/packages`, and it names each type of `gart.dev/ui` and
of each pack under it that the module uses. The answer is exact, because it reads the type information of
the Go tool chain.

```sh
cd <root of the repository>
go run ./flutter/nodescan /path/to/app/go > nodes.txt
```

`glutter gen` of an application runs `nodescan` and then the host mode.

An earlier reader took each `ui.<Name>` of the Go source with a regular
expression. Such a scan reads a name of a comment and a name of a string, and
it misses each name that an import alias or a dot import carries. The 3
examples gave the same host under the 2 sources on 2026-09-03, because each
extra name of the regular expression named a function of the package and no
node type carries such a name.

## Why the host is per application

A shared host would hold a case for each of the 566 node types, thus it would
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

## Why the Go node set is per application too

The Go file of a pack holds each node type, and the emitter writes a Dart class
and a type descriptor for each 1. compass names 27 of the 491 types that its 2
packs hold, thus 464 classes and 464 descriptors stood in its Dart output.

`internal/nodefilter` of the compiler reads the file that this tool wrote, and
it keeps the types that the application names. `glutter` gives that set from the
same `nodescan` run that writes the host, and the filter reaches the load
through `packages.Config.Overlay`. The file on disk never changes, thus 2
applications of 1 checkout each hold their own set and no `glutter doctor`
reports a stale file.

The filter cut the Dart output of compass from 18,318,376 bytes to 16,089,390,
and the tree of `gart.dev/ui` from 2,922,881 bytes to 693,895. Read the section
"The node filter of the Go side" of `docs/flutter-design.md`.

A node type that the application does not name today reaches the build the
moment 1 line names it, because the driver runs the scan in front of each load.

`flutter/ui/themes.go` and `flutter/ui/types.go` each carry the line
`// nodefilter:values` on the first line, thus the filter reads the value types
of the node set the same way. A type of such a file stays when a kept
declaration names it. The Dart output of compass held 139,198 bytes of theme
value types, and it holds 8,021 now.

## The icons

A Go `ui.Icon` node names its icon with a string, and `goIcons` of the host
maps that name to the Material set. Material holds 8,825 icons, thus a map of
each 1 would keep the whole table in each application.

`nodescan` writes each string constant of the module after the node names, with
a quotation mark in front. This tool keeps each 1 that names an icon of
Material, and it writes `goAppIcons` and `goGenInstall` into the host of the
application. The compass application names 61 such icons.

`goIcon` reports an unknown name. A name that the Go source builds at run time
reaches no string constant, and the application then adds it to `goIcons`
itself.

## The mapping

`lib/read.dart` holds the rules. A widget reaches the node set when each
parameter that its constructor demands maps to a Go type. A constructor that
demands no such parameter also reaches the set, and each parameter of that node
then takes the default of Flutter.

| Dart | Go | Note |
|---|---|---|
| `String`, `double`, `int`, `bool` | `string`, `float64`, `int`, `bool` | |
| `Color` | `ui.Color` | A named type over `uint32`. |
| `IconData` | `string` | The host reads `goIcons`. |
| `Duration` | `int` | Milliseconds. |
| `DateTime` | `int` | The milliseconds of the Unix epoch. A value of 0 means no date. |
| `TimeOfDay` | `int` | The minutes after midnight. |
| `Object` | `string` | A tag, a key, or a message. A Dart String is such an Object. |
| `Uint8List` | `[]byte` | |
| `List<int>`, `List<double>`, `List<bool>` | `[]int`, `[]float64`, `[]bool` | |
| `Matrix4` | `[]float64` | 16 numbers, in the order of the columns. |
| `EdgeInsets`, `EdgeInsetsGeometry` | `ui.Insets` | |
| `TextStyle` | `ui.TextStyle` | A subset of the fields. |
| `BoxConstraints` | `ui.Constraints` | A maximum of 0 means no limit. |
| `Offset` | `ui.Offset` | |
| `Rect`, `RelativeRect` | `ui.Rect` | 4 edges. A RelativeRect reads each 1 as a distance from the parent. |
| `Alignment`, `AlignmentGeometry`, `Curve` | `string` | A name of the table of `convert.dart`. |
| An enum | `string` | The name of the value. |
| `Key` | `string` | The host builds a `ValueKey<String>`. A parameter named `key` reaches the set only when the constructor demands it, because `ui.Keyed` gives a key to any child. |
| `Widget` | `ui.Node` | |
| A subtype of `Widget`, for example `PreferredSizeWidget` | `ui.Node` | The host casts the result. |
| `List<Widget>`, `List<String>` | `[]ui.Node`, `[]string` | |
| `void Function()` | `func()` | The host runs it as a goroutine. |
| `void Function(T)` for a mapped scalar or enum | `func(T)` | |
| `void Function(RangeValues)` | `func(float64, float64)` | The host builds no Go struct, thus a range crosses as 2 numbers. |
| `Future<void> Function()` | `ui.Task` | The end of the goroutine ends the work. |
| `Future<bool> Function()` | `ui.Ask` | The same, and the result gives the answer. |
| `Future<bool?> Function(<enum>)` | `ui.AskText` | The same. The Go side reads the name of the value of that enum, for example the direction of a `Dismissible`. |
| `Widget Function(BuildContext)` | `ui.Builder` | A synchronous position. Read decision 221. |
| `Widget Function(BuildContext, int)` | `ui.ItemBuilder` | A synchronous position. |
| `Widget Function(BuildContext, Orientation)` | `ui.OrientBuilder` | The same. The argument takes `portrait` or `landscape`. |
| `Widget Function(BuildContext, BoxConstraints)` | `ui.BoxBuilder` | The same. 4 numbers, and a maximum of 0 means no limit. |
| `List<Widget> Function(BuildContext, bool)` | `ui.SliverBuilder` | The same. |
| `MouseCursor` | `string` | A name of the table of `convert.dart`. |
| `ScrollPhysics` | `string` | A name: `bouncing`, `clamping`, `never`, `always`, `page`, `range`, `fixedExtent`. |
| `Locale` | `string` | A language tag, for example `en` or `en_US`. |
| `BorderSide` | `ui.BorderSide` | |
| `BorderRadius`, `BorderRadiusGeometry` | `ui.Radius` | The 4 corners. |
| `ShapeBorder`, `OutlinedBorder` | `ui.Shape` | The host gives an `OutlinedBorder`. |
| `VisualDensity` | `ui.Density` | |
| `IconThemeData` | `ui.IconStyle` | The name differs, because a widget holds the name `IconTheme`. |
| `ThemeData` | `ui.ThemeStyle` | A subset of the fields. The name differs, because `app.go` holds `ui.ThemeData`. |
| `ActionIconThemeData` | `ui.ActionIconStyle` | 1 builder for each button of an app bar. |
| `BadgeThemeData` | `ui.BadgeStyle` | A subset of the fields. |
| `BottomNavigationBarThemeData` | `ui.BottomNavigationBarStyle` | The same. |
| `ButtonBarThemeData` | `ui.ButtonBarStyle` | The same. Flutter holds ButtonBar as deprecated. |
| `ButtonThemeData` | `ui.ButtonThemeStyle` | The same. The name differs, because `ui.ButtonStyle` holds the Flutter `ButtonStyle`. |
| `CarouselViewThemeData` | `ui.CarouselViewStyle` | The same. |
| `CheckboxThemeData` | `ui.CheckboxStyle` | The same. |
| `ChipThemeData` | `ui.ChipStyle` | The same. |
| `CupertinoThemeData` | `ui.CupertinoStyle` | The same. The text theme stays out. |
| `DataTableThemeData` | `ui.DataTableStyle` | The same. |
| `DatePickerThemeData` | `ui.DatePickerStyle` | The same. |
| `DividerThemeData` | `ui.DividerStyle` | The same. |
| `DrawerThemeData` | `ui.DrawerStyle` | The same. |
| `DropdownMenuThemeData` | `ui.DropdownMenuStyle` | The same. It carries 1 `ui.MenuStyle`. |
| `ElevatedButtonThemeData` | `ui.ButtonStyle` | The theme holds 1 `ButtonStyle` alone. |
| `ExpansionTileThemeData` | `ui.ExpansionTileStyle` | A subset of the fields. |
| `FilledButtonThemeData` | `ui.ButtonStyle` | The theme holds 1 `ButtonStyle` alone. |
| `FloatingActionButtonThemeData` | `ui.FloatingActionButtonStyle` | A subset of the fields. |
| `IconButtonThemeData` | `ui.ButtonStyle` | The theme holds 1 `ButtonStyle` alone. |
| `MenuBarThemeData` | `ui.MenuStyle` | The theme holds 1 `MenuStyle` alone. |
| `MenuButtonThemeData` | `ui.ButtonStyle` | The theme holds 1 `ButtonStyle` alone. |
| `MenuStyle` | `ui.MenuStyle` | 1 value for each field, and 1 color for each state. |
| `MenuThemeData` | `ui.MenuStyle` | The theme holds 1 `MenuStyle` alone. |
| `NavigationBarThemeData` | `ui.NavigationBarStyle` | A subset of the fields. |
| `NavigationDrawerThemeData` | `ui.NavigationDrawerStyle` | The same. |
| `NavigationRailThemeData` | `ui.NavigationRailStyle` | The same. |
| `OutlinedButtonThemeData` | `ui.ButtonStyle` | The theme holds 1 `ButtonStyle` alone. |
| `PopupMenuThemeData` | `ui.PopupMenuStyle` | A subset of the fields. |
| `ProgressIndicatorThemeData` | `ui.ProgressIndicatorStyle` | The same. |
| `RadioThemeData` | `ui.RadioStyle` | The same. |
| `ScrollbarThemeData` | `ui.ScrollbarStyle` | The same. Radius takes a circular value. |
| `SearchBarThemeData` | `ui.SearchBarStyle` | The same. |
| `SearchViewThemeData` | `ui.SearchViewStyle` | The same. |
| `SegmentedButtonThemeData` | `ui.SegmentedButtonStyle` | 1 `ButtonStyle` and 1 icon. |
| `SliderThemeData` | `ui.SliderStyle` | A subset of the fields. The shape of each part stays out. |
| `SnackBarThemeData` | `ui.SnackBarStyle` | A subset of the fields. |
| `SwitchThemeData` | `ui.SwitchStyle` | The same. |
| `TextButtonThemeData` | `ui.ButtonStyle` | The theme holds 1 `ButtonStyle` alone. |
| `TextSelectionThemeData` | `ui.TextSelectionStyle` | 3 colors. |
| `TimePickerThemeData` | `ui.TimePickerStyle` | A subset of the fields. |
| `ToggleButtonsThemeData` | `ui.ToggleButtonsStyle` | The same. |
| `TooltipThemeData` | `ui.TooltipStyle` | The same. |
| `WidgetStateProperty<Color?>` | `ui.StateColor` | 1 color for each state. |
| `WidgetStateProperty<Color>` | `ui.StateColor` | The same. A state with no color gives a transparent color, because the parameter takes no null. |
| `Size` | `ui.Size` | A width and a height. |
| `ButtonStyle` | `ui.ButtonStyle` | A color for each state, and 1 value of each other field. |
| `ImageProvider` | `ui.ImageSource` | A kind and a source: `network`, `asset`, or `memory`. |
| `Decoration`, `BoxDecoration` | `ui.Decoration` | The host gives a `BoxDecoration`. |
| `InlineSpan`, `TextSpan` | `ui.Span` | A text, a style, and the children. |
| `SliverChildDelegate` | `ui.SliverChildren` | A list, or a builder with a count. |
| `SliverGridDelegate` | `ui.GridSpec` | A count of columns, or a largest size. |
| `RangeValues` | `ui.Range` | |
| `BottomNavigationBarItem` | `ui.BarItem` | |
| `NavigationRailDestination` | `ui.RailItem` | |
| `Step` | `ui.Step` | |
| `ContextMenuButtonItem` | `ui.MenuButtonItem` | |
| `DataColumn`, `DataRow` | `ui.DataColumn`, `ui.DataRow` | A cell reads `ui.DataCell`. |
| `FocusNode` | `ui.FocusKey` | The host owns the object, by the name of the key. |
| `FocusScopeNode` | `ui.ScopeKey` | The same. |
| `ScrollController` | `ui.ScrollKey` | The same. |
| `TextEditingController` | `ui.TextKey` | The same. |
| `WidgetStatesController` | `ui.StatesKey` | The same. |
| `Animation<double>` | `ui.AnimationKey` | The same. The Go side names the target, and the host runs the animation. |
| `LayerLink` | `ui.LinkKey` | The same. |
| `SliverOverlapAbsorberHandle` | `ui.OverlapKey` | The same. |
| `OverlayPortalController` | `ui.OverlayKey` | The same. The Go side names whether the child stands on the screen. |
| `MenuController` | `ui.MenuKey` | The same. The Go side names whether the menu stands open. |
| `PageStorageBucket` | `ui.StorageKey` | The same. |

A parameter that the constructor does not demand takes a pointer field, and a
nil pointer means "the default of Flutter". A parameter of any other type does
not reach the node set.

A constructor that demands a private field formal, for example
`required this._borderRadius` of `CupertinoFocusHalo.withRRect`, leaves the
set. No code outside the library of that class can name such a parameter.

The name of the value type of a theme strips "ThemeData" and adds "Style". A
field that Flutter carries for each state holds 1 value in the Go struct, as
`ui.ButtonStyle` does. The host gives that value to each state. Read
`flutter/ui/themes.go`.

The key types hold an identity, and not a value. A Go tree is data and it
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
dart run bin/nodegen.dart report ../goflutter -all
```

It names each widget that the reader dropped, the cause of each drop, and each
Dart type that no Go type carries.

The largest groups that stay out on 2026-09-03, with 119 dropped constructors:

| The group | Count | The cause |
|---|---|---|
| A generic widget, for example `Radio<T>` | 17 | A node holds no type argument, thus the host cannot name the instantiation. |
| A delegate that a Dart subclass must implement | 12 | A Go type cannot implement a Dart interface. |
| A live object of the host, for example `EditableTextState` or `ViewportOffset` | 12 | The object lives inside 1 build of Flutter, and a Go value cannot hold it. |
| A builder that takes an `Animation<double>` | 9 | The animation belongs to 1 item of a list, and the key table holds 1 object for each name. |
| A `BuildContext` | 5 | A Go value holds none. |

## Capability validation

`make flutter-test` verifies generated fields against the retained Go declarations.
It also runs the maintained `flutter/testdata/core` fixture from Go through Flutter.
The fixture covers signals, callbacks, lazy lists, focus, controllers, disposal,
constructor defaults, and widget reassembly. It needs no local example files.

`nodegen report <project> -all` lists unsupported constructor members with their types.
A widget name alone does not imply complete support for its fields.
For example, `CustomPaint` does not yet expose `CustomPainter` callbacks.

The host omits unset arguments with inaccessible defaults. Dart then supplies
the original default. Explicit Go values still reach those parameters.

# nodegen

nodegen reads the public API of Flutter with `package:analyzer`, and it writes
the node set of milestone 3. It needs no change of the compiler.

## The 3 modes

| Output | Scope | Command |
|---|---|---|
| `flutter/ui/widgets_gen.go` | The whole project. 1 Go struct for each Flutter widget constructor. | `nodegen go` |
| `<app>/lib/widgets_gen.dart` | 1 application. 1 case for each node type that the application names. | `nodegen host` |
| The yield of the reader | A report on the terminal. | `nodegen report` |

```sh
cd flutter/nodegen
dart pub get
dart run bin/nodegen.dart go     ../goflutter ../ui/widgets_gen.go   # runs gofmt
dart run bin/nodegen.dart report ../goflutter -all
dart run bin/nodegen.dart host   ../goflutter ../example/counter/lib/widgets_gen.dart nodes.txt
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
`golang.org/x/tools/go/packages`, and it names each type of `gart.dev/ui` that
the module uses. The answer is exact, because it reads the type information of
the Go tool chain.

```sh
cd <root of the repository>
go run ./flutter/nodescan flutter/example/counter/go > nodes.txt
```

`build.sh` of an application runs `nodescan` and then the host mode.

An earlier reader took each `ui.<Name>` of the Go source with a regular
expression. Such a scan reads a name of a comment and a name of a string, and
it misses each name that an import alias or a dot import carries. The 3
examples gave the same host under the 2 sources on 2026-09-03, because each
extra name of the regular expression named a function of the package and no
node type carries such a name.

## Why the host is per application

A shared host would hold a case for each of the 564 node types, thus it would
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
| `Widget` | `ui.Node` | |
| A subtype of `Widget`, for example `PreferredSizeWidget` | `ui.Node` | The host casts the result. |
| `List<Widget>`, `List<String>` | `[]ui.Node`, `[]string` | |
| `void Function()` | `func()` | The host runs it as a goroutine. |
| `void Function(T)` for a mapped scalar or enum | `func(T)` | |
| `void Function(RangeValues)` | `func(float64, float64)` | The host builds no Go struct, thus a range crosses as 2 numbers. |
| `Future<void> Function()` | `ui.Task` | The end of the goroutine ends the work. |
| `Future<bool> Function()` | `ui.Ask` | The same, and the result gives the answer. |
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

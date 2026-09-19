# Widget and service bindings

Nodegen reads resolved Dart libraries through `package:analyzer`. It writes
Go declarations and the Dart host adapters for an application.
Application-owned generation is experimental. The shared core supplies Go
nodes, signals, callbacks, controller identities, and supported value adapters.

## Generate application bindings

Use `glutter bind` with an explicit manifest. The [manifest contract](#application-owned-bindings-experimental)
defines selected libraries, constructors, and service functions.
Use `glutter gen` afterward to compile Go and generate the host.

The generator resolves packages from the application. Two libraries can export
widgets with the same name because each binding retains its package identity.
Unsupported members appear in the surface receipt. A widget name does not
establish complete support for all constructor parameters.

## Repository core and compatibility packages

The repository maintains the shared core and legacy Material, Cupertino, and
SVG packages. Existing applications can retain these packages during migration.
New application libraries use manifests without an edit to the built-in pack list.

| Dart library | Repository Go package |
|---|---|
| `package:flutter/widgets.dart` | `gart.dev/ui` |
| `package:material_ui/material_ui.dart` | `gart.dev/ui/material` |
| `package:cupertino_ui/cupertino_ui.dart` | `gart.dev/ui/cupertino` |
| `package:flutter_svg/flutter_svg.dart` | `gart.dev/ui/svg` |

These maintenance commands run from `flutter/nodegen`:

```sh
dart pub get
dart run bin/nodegen.dart go ../goflutter ../ui
dart run bin/nodegen.dart report ../goflutter -all
dart run bin/nodegen.dart host ../goflutter ../goflutter/lib/all_gen_check.dart -all
cd ../goflutter && flutter analyze && rm lib/all_gen_check.dart
```

The first argument names a project with resolved package metadata.
`report` lists constructor coverage and unsupported Dart types.
`host` accepts a file of node names, `-` for standard input, or `-all`.
`-all` validates the complete host but is unsuitable for normal application output.

## Application host and node filter

`flutter/nodescan` reads Go type information and emits the used node identities.
Glutter passes that list to the Go declaration filter and the Dart host generator.
The host contains cases for the application's node types. An unrestricted host
would retain unused Flutter constructors through its dispatch table.

The Go filter uses an overlay and leaves installed declaration files unchanged.
Value declarations marked `// nodefilter:values` remain when retained types need them.
The next build discovers newly used node types before compilation.

The scanner also reports string constants. The host retains matching Material
icons and installs them through `goGenInstall`. Dynamic icon names require an
explicit host registration because static discovery cannot infer their values.

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
| `Widget Function(BuildContext)` | `ui.Builder` | A synchronous callback position. |
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
`GoIdentities` of `goflutter` keeps it across each rebuild.

A `FileImage` stands outside `ui.ImageSource`, because it reads a `File` of
`dart:io`, and the host also builds for the web. A Go program reads the file
and gives the bytes under the kind `memory`.

A named constructor takes a Go struct of its own. `ListView.builder` gives
`ui.ListViewBuilder`, and the host switches on the name of the type descriptor.

## Capability validation

`make flutter-test` verifies generated fields against the retained Go declarations.
It also runs the maintained `flutter/testdata/core` fixture from Go through Flutter.
The fixture covers signals, callbacks, lazy lists, focus, controllers, disposal,
constructor defaults, and widget reassembly. It needs no local example files.

`nodegen report <project> -all` lists unsupported constructor members with their types.
A widget name alone does not imply complete support for its fields.
`CustomPaint` exposes synchronous Go painters for rectangles, circles, and lines.
The canvas expires when the callback returns. Change `Painter.Revision` when
the callback or its output changes. Paths, text, hit tests, and painter semantics
remain outside this adapter.

## Application-owned bindings (experimental)

`glutter bind` reads `glutter.bindings.json` from the Flutter application.
It uses that application's resolved dependencies and writes typed Go packages.
The shared core supplies nodes, signals, callbacks, controllers, and value types.
A Go design library can compose these primitives without a Dart widget class
for each Go component. A Dart design library uses generated constructor bindings.

After the application's first `glutter gen`, resolve dependencies with
`flutter pub get`. Add a manifest before Go code imports the new binding package:

```json
{
  "version": 1,
  "libraries": [{
    "uri": "package:my_app/brand.dart",
    "go_import": "example.com/my_app/bindings/brand",
    "directory": "go/bindings/brand",
    "symbols": ["Badge"],
    "functions": ["greeting"]
  }]
}
```

Run `glutter bind`, then import the generated package and run `glutter gen`.
Repeat `glutter bind` after a dependency API or manifest change.
Commit the manifest, generated Go declarations, and surface receipt together.
The receipt lists generated files and unsupported members. The generator removes
obsolete files from its previous receipt and refuses to replace handwritten files.
Application code must not edit generated files.

`symbols` selects Go constructor names, including named constructor variants.
`functions` selects public Dart top-level functions. Both lists use explicit names.
Equal widget names in different packages retain separate identities.
An unsupported required constructor parameter stops generation.
An unsupported optional parameter appears in the surface receipt and a warning.
Custom object parameters, custom enums, and generic functions remain unsupported.

The service adapter accepts required `String`, `int`, `double`, and `bool`
parameters. Results can use those types directly, through `Future`, or through
`Stream`. A generated Go function returns a value and an error. Call it from
a goroutine. A stream exposes `Next` and `Cancel`. Runtime disposal cancels
active streams. Dart exceptions become Go errors.

Services currently use JSON. They do not preserve object identity, arbitrary
Go string bytes, non-finite floats, or integers beyond the target's exact range.
Widget callbacks retain the existing direct adapters. This prototype does not
provide general Dart object interop or the full Canvas API.

The maintained `flutter/testdata/core` fixture covers Material and Cupertino
bindings, Go and Dart brand components, services, paint, and existing core behavior.
The repository retains its existing design-library packages during migration.

The host omits unset arguments with inaccessible defaults. Dart then supplies
the original default. Explicit Go values still reach those parameters.

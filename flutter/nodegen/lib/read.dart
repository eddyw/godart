import 'dart:convert';
import 'dart:io';

import 'package:analyzer/dart/analysis/analysis_context_collection.dart';
import 'package:analyzer/dart/analysis/results.dart';
import 'package:analyzer/dart/element/element.dart';
import 'package:analyzer/dart/element/nullability_suffix.dart';
import 'package:analyzer/dart/element/type.dart';

/// A pack holds the node types of 1 Dart package, and it names the Go package
/// that carries them.
///
/// The core pack holds the widgets of the Flutter framework itself, and the
/// hand written nodes of flutter/ui stand beside them. Each other pack holds
/// the widgets of 1 package of pub.dev, thus a package that an application
/// does not name costs it no Dart.
class Pack {
  const Pack(this.uri, this.short, this.goPath, this.goPackage);

  /// The Dart library that the reader walks.
  final String uri;

  /// The short name of the pack. The report groups by it.
  final String short;

  /// The Go import path of the pack.
  final String goPath;

  /// The Go package name.
  final String goPackage;

  /// The core pack declares the interface Node itself, thus its structs need
  /// no import and no qualified type.
  bool get isCore => goPath == 'gart.dev/ui';
}

/// The packs that the node set covers.
///
/// Flutter moved Material and Cupertino out of the framework, thus each 1
/// stands here as a package of pub.dev. Read
/// https://pub.dev/packages/material_ui.
const packs = <Pack>[
  Pack('package:flutter/widgets.dart', 'widgets', 'gart.dev/ui', 'ui'),
  Pack(
    'package:material_ui/material_ui.dart',
    'material_ui',
    'gart.dev/ui/material',
    'material',
  ),
  Pack(
    'package:cupertino_ui/cupertino_ui.dart',
    'cupertino_ui',
    'gart.dev/ui/cupertino',
    'cupertino',
  ),
  Pack(
    'package:flutter_svg/flutter_svg.dart',
    'flutter_svg',
    'gart.dev/ui/svg',
    'svg',
  ),
];

/// corePack holds the widgets of the Flutter framework, and the hand written
/// nodes of flutter/ui stand beside them.
final Pack corePack = packs[0];

/// materialPack holds the widgets of Material. The icon table of the host
/// reads the class Icons of that package.
final Pack materialPack = packs[1];

/// The libraries that the generated host may import, in the order of the
/// search. A name that no 1 of them exports does not compile in the host, thus
/// the reader drops the field that reads it.
const candidates = <String>[
  'package:material_ui/material_ui.dart',
  'package:flutter_svg/flutter_svg.dart',
  'package:cupertino_ui/cupertino_ui.dart',
  'package:flutter/widgets.dart',
  'package:flutter/painting.dart',
  'package:flutter/rendering.dart',
  'package:flutter/animation.dart',
  'package:flutter/gestures.dart',
  'package:flutter/services.dart',
  'package:flutter/semantics.dart',
  'package:flutter/foundation.dart',
  'package:flutter/scheduler.dart',
  'package:flutter/physics.dart',
];

/// A parameter of a widget constructor that the node set carries.
class Param {
  Param({
    required this.dartName,
    required this.goName,
    required this.type,
    required this.required,
    required this.nullable,
    required this.defaultCode,
    required this.named,
    this.omitDefault = false,
    this.doc = '',
  });

  /// The first sentence of the dartdoc of the field of the Flutter class, or
  /// an empty text.
  final String doc;

  /// The name of the Dart parameter.
  final String dartName;

  /// The name of the Go field.
  final String goName;

  final GoType type;

  /// A parameter that the constructor demands.
  final bool required;

  /// A Dart type that accepts null.
  final bool nullable;

  /// The default of the Dart parameter, verbatim, or null.
  final String? defaultCode;

  /// A named parameter. A positional parameter takes its place in the call.
  final bool named;

  final bool omitDefault;

  /// A field that a call may leave unset takes a pointer, thus the host can
  /// tell "not set" from the zero value.
  bool get optional => !required && type.pointer;

  String get goDecl => '$goName ${optional ? "*" : ""}${type.goType}';
}

/// 1 node type of the generated set.
class WidgetSpec {
  WidgetSpec({
    required this.dartClass,
    required this.ctor,
    required this.goName,
    required this.pack,
    required this.params,
    required this.isConst,
    required this.needed,
    this.doc = '',
  });

  /// The Flutter class, for example ListView.
  final String dartClass;

  /// The named constructor, or an empty text for the unnamed 1.
  final String ctor;

  /// The Go struct, for example ListViewBuilder.
  final String goName;

  /// The pack that exports the class.
  final Pack pack;

  final List<Param> params;
  final bool isConst;

  /// The libraries that a default of this widget names. The host imports each
  /// 1 only when it holds the case.
  final Set<String> needed;

  /// The first paragraph of the dartdoc of the Flutter class, or an empty
  /// text.
  final String doc;

  /// The kind of the node, as the Dart host reads it from the type
  /// descriptor.
  String get kind => '${pack.goPath}.$goName';

  String get dartCall => ctor.isEmpty ? dartClass : '$dartClass.$ctor';
}

/// A widget that the reader dropped, and the cause.
class Skipped {
  Skipped(this.name, this.param, this.shape);

  /// The Go name of the constructor.
  final String name;

  /// The Dart parameter that blocks the constructor.
  final String param;

  /// The Dart type of that parameter, or the shape of the block. The report
  /// counts each drop of 1 shape together.
  final String shape;

  String get cause => 'the parameter $param holds $shape';
}

class Reading {
  final unsupportedMembers =
      <({String kind, String member, String type, String reason})>[];
  final noticeSources = <Pack, Set<String>>{};
  final widgets = <WidgetSpec>[];
  final skipped = <Skipped>[];
  final unmapped = <String, int>{};

  /// enums holds the constant names of each Dart enum that a field reads.
  final enums = <String, List<String>>{};

  /// enumLibs gives the library that exports each such enum. The generated
  /// host imports each 1, because material.dart and cupertino.dart do not
  /// export them all.
  final enumLibs = <String, String>{};

  /// visible gives the library that exports each name that the host can write.
  final visible = <String, String>{};

  /// needed holds each library that the generated host must import.
  final needed = <String>{};

  /// icons holds the name of each icon of the Material set. nodescan gives the
  /// string constants of the application, and the host keeps each 1 that this
  /// set holds.
  final icons = <String>{};
}

/// read walks the library of each pack and maps each widget constructor.
Future<Reading> read(String root) async {
  final path = Directory(root).absolute.resolveSymbolicLinksSync();
  final collection = AnalysisContextCollection(includedPaths: <String>[path]);
  final session = collection.contextFor(path).currentSession;
  final out = Reading();
  final seen = <String>{};

  for (final uri in candidates) {
    final result = await session.getLibraryByUri(uri);
    if (result is! LibraryElementResult) continue;
    for (final name in result.element.exportNamespace.definedNames2.keys) {
      out.visible.putIfAbsent(name, () => uri);
    }
  }

  final iconLib = await session.getLibraryByUri(materialPack.uri);
  if (iconLib is LibraryElementResult) {
    final Object? icons =
        iconLib.element.exportNamespace.definedNames2['Icons'];
    if (icons is ClassElement) {
      for (final f in icons.fields) {
        final n = f.name;
        if (n == null || n.startsWith('_') || !f.isStatic) continue;
        if (f.type.getDisplayString() != 'IconData') continue;
        out.icons.add(n);
      }
    }
  }

  for (final pack in packs) {
    final result = await session.getLibraryByUri(pack.uri);
    if (result is! LibraryElementResult) {
      throw StateError('${pack.uri} does not resolve: $result');
    }
    for (final entry in result.element.exportNamespace.definedNames2.entries) {
      final element = entry.value;
      if (element is! ClassElement) continue;
      final name = element.name;
      if (name == null || name.startsWith('_')) continue;
      if (!isWidget(element) || element.isAbstract) continue;
      if (_skip.contains(name)) continue;
      for (final ctor in element.constructors) {
        if (!ctor.isPublic) continue;
        final goName = goStructName(name, ctor.name ?? '');
        if (!seen.add(goName)) continue;
        final sources = out.noticeSources.putIfAbsent(pack, () => <String>{});
        readCtor(out, pack, name, ctor, goName, docsOf(element, sources));
      }
    }
  }
  out.widgets.sort((a, b) => a.goName.compareTo(b.goName));
  return out;
}

void readCtor(
  Reading out,
  Pack pack,
  String className,
  ConstructorElement ctor,
  String goName,
  ClassDocs docs,
) {
  final params = <Param>[];
  final needed = <String>{};
  for (final p in ctor.formalParameters) {
    final pname = p.name;
    if (pname == null || pname.startsWith('_')) {
      // A private field formal, for example `required this._borderRadius` of
      // CupertinoFocusHalo.withRRect. No code outside the library can name it,
      // thus a constructor that demands 1 reaches no host.
      if (p.isRequired) {
        out.skipped.add(Skipped(goName, pname ?? '?', 'a private name'));
        return;
      }
      continue;
    }
    // A widget carries an optional key, and ui.Keyed gives that key to any
    // child. A widget that demands a key, for example Dismissible, needs the
    // key on itself, thus the node holds 1 field of its own.
    if (pname == 'key' && !p.isRequired) continue;
    final nullable = p.type.nullabilitySuffix == NullabilitySuffix.question;
    final go = mapType(out, p.type);
    if (go == null) {
      final key = p.type.getDisplayString();
      out.unsupportedMembers.add((
        kind: '${pack.goPath}.$goName',
        member: pname,
        type: key,
        reason: 'unsupported type',
      ));
      out.unmapped[key] = (out.unmapped[key] ?? 0) + 1;
      if (p.isRequired) {
        out.skipped.add(Skipped(goName, pname, key));
        return;
      }
      continue;
    }
    if (!p.isNamed && !p.isRequired) {
      // An optional positional parameter has no stable place in a call that
      // leaves an earlier 1 out.
      out.skipped.add(Skipped(goName, pname, 'a positional place of its own'));
      return;
    }
    final omitDefault =
        !p.isRequired &&
        (!nullable ||
            p.defaultValueCode != null && p.defaultValueCode != 'null') &&
        !usableDefault(out, needed, p.defaultValueCode);
    if (omitDefault && (!p.isNamed || !go.pointer)) {
      out.unsupportedMembers.add((
        kind: '${pack.goPath}.$goName',
        member: pname,
        type: p.type.getDisplayString(),
        reason: 'inaccessible default',
      ));
      // The host cannot write the call without the default of Flutter, and a
      // default that names a private symbol does not compile outside its
      // library.
      continue;
    }
    params.add(
      Param(
        dartName: pname,
        goName: goFieldName(pname),
        type: go,
        required: p.isRequired,
        nullable: nullable,
        defaultCode: p.defaultValueCode,
        named: p.isNamed,
        omitDefault: omitDefault,
        doc: firstSentence(docs.fields[pname] ?? ''),
      ),
    );
  }
  // A constructor with no mapped parameter still reaches the node set. Each
  // parameter of it then takes the default of Flutter, thus the node holds the
  // whole meaning of the widget. ui.CupertinoActivityIndicator is such a node.
  out.widgets.add(
    WidgetSpec(
      dartClass: className,
      ctor: ctor.name == 'new' ? '' : (ctor.name ?? ''),
      goName: goName,
      pack: pack,
      params: params,
      isConst: ctor.isConst,
      needed: needed,
      doc: firstParagraph(docs.owner),
    ),
  );
}

/// The widgets that the Go model cannot use. Builder exists only to give a
/// BuildContext, and a Go value holds none. The Go struct of Theme would carry
/// the name of the accessor ui.Theme, which gives the theme that the host
/// wrote. A Go program reads the field Theme of ui.MaterialApp, or the node
/// ui.AnimatedTheme.
const _skip = <String>{'Builder', 'Theme'};

/// The words of a default that name no declaration.
const _words = <String>{
  'const',
  'true',
  'false',
  'null',
  'new',
  'if',
  'else',
  'is',
  'as',
};

/// usableDefault reports whether the host can write the default verbatim.
///
/// A default that names a private symbol does not compile outside its library,
/// and a default that names a static member of the class without the class,
/// for example defaultThickness, does not resolve in the host either.
bool usableDefault(Reading out, Set<String> needed, String? code) {
  if (code == null) return false;
  if (RegExp(r'(^|[^A-Za-z0-9])_').hasMatch(code)) return false;
  final found = <String>{};
  for (final m in RegExp(r'(?<![.\w])[A-Za-z][A-Za-z0-9]*').allMatches(code)) {
    final word = m.group(0)!;
    if (_words.contains(word)) continue;
    final lib = out.visible[word];
    if (lib == null) return false;
    found.add(lib);
  }
  needed.addAll(found);
  out.needed.addAll(found);
  return true;
}

bool isWidget(InterfaceElement c) {
  if (c.name == 'Widget') return true;
  for (final t in c.allSupertypes) {
    if (t.element.name == 'Widget') return true;
  }
  return false;
}

/// A Go type and the read that the host performs.
class GoType {
  const GoType(this.goType, this.kind, {this.pointer = true});

  /// The Go type with no pointer.
  final String goType;

  /// The name of the read that the host writes. Read write_dart.dart.
  final String kind;

  /// A scalar that a call may leave unset takes a pointer.
  final bool pointer;
}

const _scalars = <String, GoType>{
  'String': GoType('string', 'str'),
  'Key': GoType('string', 'valueKey'),
  'double': GoType('float64', 'f64'),
  'int': GoType('int', 'i64'),
  'bool': GoType('bool', 'boolean'),
  'Color': GoType('Color', 'color'),
  'IconData': GoType('string', 'icon'),
  'Duration': GoType('int', 'millis'),
  'AlignmentGeometry': GoType('string', 'alignment'),
  'Alignment': GoType('string', 'alignmentXY'),
  'Curve': GoType('string', 'curve'),
  'Offset': GoType('Offset', 'offset'),
  'EdgeInsets': GoType('Insets', 'insets'),
  'EdgeInsetsGeometry': GoType('Insets', 'insets'),
  'TextStyle': GoType('TextStyle', 'textStyle'),
  'BoxConstraints': GoType('Constraints', 'constraints'),
  'MouseCursor': GoType('string', 'cursor'),
  'ScrollPhysics': GoType('string', 'physics'),
  'Locale': GoType('string', 'locale'),
  'BorderSide': GoType('BorderSide', 'borderSide'),
  'BorderRadius': GoType('Radius', 'radius'),
  'BorderRadiusGeometry': GoType('Radius', 'radius'),
  'ShapeBorder': GoType('Shape', 'shape'),
  'OutlinedBorder': GoType('Shape', 'shape'),
  'VisualDensity': GoType('Density', 'density'),
  'IconThemeData': GoType('IconStyle', 'iconStyle'),
  'WidgetStateProperty<Color>': GoType('StateColor', 'stateColorOn'),
  'WidgetStateProperty<Color?>': GoType('StateColor', 'stateColor'),
  'Size': GoType('Size', 'boxSize'),
  'ButtonStyle': GoType('ButtonStyle', 'buttonStyle'),
  'ImageProvider<Object>': GoType('ImageSource', 'image'),
  'Decoration': GoType('Decoration', 'decoration'),
  'BoxDecoration': GoType('Decoration', 'decoration'),
  'Rect': GoType('Rect', 'rect'),
  'RelativeRect': GoType('Rect', 'relRect'),
  'InlineSpan': GoType('Span', 'span'),
  'TextSpan': GoType('Span', 'span'),
  'SliverChildDelegate': GoType('SliverChildren', 'sliverChildren'),
  'SliverGridDelegate': GoType('GridSpec', 'gridSpec'),
  'RangeValues': GoType('Range', 'range'),
  'ThemeData': GoType('ThemeStyle', 'themeStyle'),
  // The theme of 1 widget family, and the value type of it. Read
  // flutter/ui/themes.go.
  'ActionIconThemeData': GoType('ActionIconStyle', 'actionIconStyle'),
  'AppBarThemeData': GoType('AppBarStyle', 'appBarStyle'),
  'BadgeThemeData': GoType('BadgeStyle', 'badgeStyle'),
  'BottomAppBarThemeData': GoType('BottomAppBarStyle', 'bottomAppBarStyle'),
  'BottomNavigationBarThemeData': GoType(
    'BottomNavigationBarStyle',
    'bottomNavigationBarStyle',
  ),
  'ButtonBarThemeData': GoType('ButtonBarStyle', 'buttonBarStyle'),
  'ButtonThemeData': GoType('ButtonThemeStyle', 'buttonThemeStyle'),
  'CardThemeData': GoType('CardStyle', 'cardStyle'),
  'CarouselViewThemeData': GoType('CarouselViewStyle', 'carouselViewStyle'),
  'CheckboxThemeData': GoType('CheckboxStyle', 'checkboxStyle'),
  'ChipThemeData': GoType('ChipStyle', 'chipStyle'),
  'CupertinoThemeData': GoType('CupertinoStyle', 'cupertinoStyle'),
  'DataTableThemeData': GoType('DataTableStyle', 'dataTableStyle'),
  'DatePickerThemeData': GoType('DatePickerStyle', 'datePickerStyle'),
  'DialogThemeData': GoType('DialogStyle', 'dialogStyle'),
  'DividerThemeData': GoType('DividerStyle', 'dividerStyle'),
  'DrawerThemeData': GoType('DrawerStyle', 'drawerStyle'),
  'DropdownMenuThemeData': GoType('DropdownMenuStyle', 'dropdownMenuStyle'),
  'ExpansionTileThemeData': GoType('ExpansionTileStyle', 'expansionTileStyle'),
  'FloatingActionButtonThemeData': GoType(
    'FloatingActionButtonStyle',
    'floatingActionButtonStyle',
  ),
  'InputDecorationThemeData': GoType(
    'InputDecorationStyle',
    'inputDecorationStyle',
  ),
  'ListTileThemeData': GoType('ListTileStyle', 'listTileStyle'),
  'MaterialBannerThemeData': GoType(
    'MaterialBannerStyle',
    'materialBannerStyle',
  ),
  'MenuStyle': GoType('MenuStyle', 'menuStyle'),
  'NavigationBarThemeData': GoType('NavigationBarStyle', 'navigationBarStyle'),
  'NavigationDrawerThemeData': GoType(
    'NavigationDrawerStyle',
    'navigationDrawerStyle',
  ),
  'NavigationRailThemeData': GoType(
    'NavigationRailStyle',
    'navigationRailStyle',
  ),
  'PopupMenuThemeData': GoType('PopupMenuStyle', 'popupMenuStyle'),
  'ProgressIndicatorThemeData': GoType(
    'ProgressIndicatorStyle',
    'progressIndicatorStyle',
  ),
  'RadioThemeData': GoType('RadioStyle', 'radioStyle'),
  'ScrollbarThemeData': GoType('ScrollbarStyle', 'scrollbarStyle'),
  'SearchBarThemeData': GoType('SearchBarStyle', 'searchBarStyle'),
  'SearchViewThemeData': GoType('SearchViewStyle', 'searchViewStyle'),
  'SegmentedButtonThemeData': GoType(
    'SegmentedButtonStyle',
    'segmentedButtonStyle',
  ),
  'SliderThemeData': GoType('SliderStyle', 'sliderStyle'),
  'SnackBarThemeData': GoType('SnackBarStyle', 'snackBarStyle'),
  'SwitchThemeData': GoType('SwitchStyle', 'switchStyle'),
  'TabBarThemeData': GoType('TabBarStyle', 'tabBarStyle'),
  'TextSelectionThemeData': GoType('TextSelectionStyle', 'textSelectionStyle'),
  'TimePickerThemeData': GoType('TimePickerStyle', 'timePickerStyle'),
  'ToggleButtonsThemeData': GoType('ToggleButtonsStyle', 'toggleButtonsStyle'),
  'TooltipThemeData': GoType('TooltipStyle', 'tooltipStyle'),
  // A theme of 1 button family holds 1 ButtonStyle alone, and 2 themes of a
  // menu each hold 1 MenuStyle alone.
  'ElevatedButtonThemeData': GoType('ButtonStyle', 'elevatedButtonTheme'),
  'FilledButtonThemeData': GoType('ButtonStyle', 'filledButtonTheme'),
  'IconButtonThemeData': GoType('ButtonStyle', 'iconButtonTheme'),
  'MenuButtonThemeData': GoType('ButtonStyle', 'menuButtonTheme'),
  'OutlinedButtonThemeData': GoType('ButtonStyle', 'outlinedButtonTheme'),
  'TextButtonThemeData': GoType('ButtonStyle', 'textButtonTheme'),
  'MenuThemeData': GoType('MenuStyle', 'menuTheme'),
  'MenuBarThemeData': GoType('MenuStyle', 'menuBarTheme'),
  'BottomNavigationBarItem': GoType('BarItem', 'barItem'),
  'NavigationRailDestination': GoType('RailItem', 'railItem'),
  'Step': GoType('Step', 'step'),
  'ContextMenuButtonItem': GoType('MenuButtonItem', 'menuItem'),
  // The place of a text selection toolbar, the shape of 1 selection, and the
  // semantic meaning of a subtree. Each 1 holds a value alone.
  'TextSelectionToolbarAnchors': GoType('ToolbarAnchors', 'toolbarAnchors'),
  'SelectionGeometry': GoType('SelectionGeometry', 'selectionGeometry'),
  'SemanticsProperties': GoType('SemanticsProperties', 'semanticsProps'),
  'List<DataColumn>': GoType('[]DataColumn', 'columns', pointer: false),
  'List<DataRow>': GoType('[]DataRow', 'rows', pointer: false),
  'List<ContextMenuButtonItem>': GoType(
    '[]MenuButtonItem',
    'menuItems',
    pointer: false,
  ),
  // A matrix holds 16 numbers, in the order of the columns.
  'Matrix4': GoType('[]float64', 'matrix', pointer: false),
  // A date and a time of the day each take a plain number. A date holds the
  // milliseconds of the Unix epoch, and a value of 0 means no date. A time of
  // the day holds the minutes after midnight.
  'DateTime': GoType('int', 'date'),
  'TimeOfDay': GoType('int', 'clock'),
  // A Dart Object parameter names a tag, a key, or a message. A Go value gives
  // a text there, and a Dart String is such an Object.
  'Object': GoType('string', 'tag'),
  'Uint8List': GoType('[]byte', 'bytes', pointer: false),
  'List<int>': GoType('[]int', 'counts', pointer: false),
  'List<double>': GoType('[]float64', 'sizes', pointer: false),
  'List<bool>': GoType('[]bool', 'flags', pointer: false),
  'List<BottomNavigationBarItem>': GoType(
    '[]BarItem',
    'barItems',
    pointer: false,
  ),
  'List<NavigationRailDestination>': GoType(
    '[]RailItem',
    'railItems',
    pointer: false,
  ),
  'List<Step>': GoType('[]Step', 'steps', pointer: false),
  // The objects below hold an identity that no Go value carries. The Go side
  // names 1, and the host owns the object. Read flutter/ui/identity.go.
  'FocusNode': GoType('FocusKey', 'focusKey'),
  'ScrollController': GoType('ScrollKey', 'scrollKey'),
  'WidgetStatesController': GoType('StatesKey', 'statesKey'),
  'TextEditingController': GoType('TextKey', 'textKey'),
  'Animation<double>': GoType('AnimationKey', 'animation'),
  'LayerLink': GoType('LinkKey', 'linkKey'),
  'SliverOverlapAbsorberHandle': GoType('OverlapKey', 'overlapKey'),
  'OverlayPortalController': GoType('OverlayKey', 'overlayKey'),
  'MenuController': GoType('MenuKey', 'menuKey'),
  'FocusScopeNode': GoType('ScopeKey', 'scopeKey'),
  'PageStorageBucket': GoType('StorageKey', 'storageKey'),
};

/// The argument kinds that a callback of 1 argument carries. cb1Expr of
/// write_dart.dart holds the read of each 1, and an enum joins them.
const _cb1Args = <String>{
  'str',
  'f64',
  'i64',
  'boolean',
  'color',
  'millis',
  'date',
  'clock',
};

/// mapType gives the Go type of a Dart type, or null when no form fits.
GoType? mapType(Reading out, DartType t) {
  if (t is FunctionType) return mapFunction(out, t);

  // The exact text of the type comes first, because the nullability of an
  // argument of a generic type changes the Dart type of the converter. A
  // WidgetStateProperty<Color?> takes no value of a WidgetStateProperty<Color>.
  final exact = t.getDisplayString();
  final raw =
      _scalars[exact.endsWith('?')
          ? exact.substring(0, exact.length - 1)
          : exact];
  if (raw != null) return raw;

  final display = exact.replaceAll('?', '');
  final scalar = _scalars[display];
  if (scalar != null) return scalar;

  if (display == 'Widget') {
    return const GoType('Node', 'node', pointer: false);
  }
  if (display == 'List<Widget>') {
    return const GoType('[]Node', 'nodes', pointer: false);
  }
  if (display == 'List<String>') {
    return const GoType('[]string', 'strs', pointer: false);
  }

  final element = t.element;
  if (element is InterfaceElement &&
      element is! EnumElement &&
      isWidget(element)) {
    // A parameter that takes a subtype of Widget, for example the appBar of a
    // Scaffold. The host builds the node and casts the result.
    final name = element.name;
    if (name == null || out.visible[name] == null) return null;
    out.needed.add(out.visible[name]!);
    return GoType('Node', 'widgetAs:$name', pointer: false);
  }
  if (element is EnumElement) {
    final name = element.name;
    if (name == null) return null;
    final lib = out.visible[name];
    if (lib == null) return null;
    final values = <String>[];
    for (final f in element.fields) {
      if (!f.isEnumConstant) continue;
      final n = f.name;
      // A private constant leaves the switch of the host without a case, thus
      // the whole enum stays out.
      if (n == null || n.startsWith('_')) return null;
      values.add(n);
    }
    if (values.isEmpty) return null;
    out.enumLibs[name] = lib;
    out.needed.add(lib);
    out.enums[name] = values;
    return GoType('string', 'enum:$name');
  }
  return null;
}

/// mapFunction maps a callback. A callback with no result takes any Go
/// function. A callback with a result is a synchronous position, thus it takes
/// a named type that carries //gart:sync.
GoType? mapFunction(Reading out, FunctionType t) {
  final params = t.formalParameters;
  final ret = t.returnType.getDisplayString().replaceAll('?', '');

  final kinds = params
      .map((p) => p.type.getDisplayString().replaceAll('?', ''))
      .toList();

  if (ret == 'void') {
    if (params.isEmpty) return const GoType('func()', 'cb0', pointer: false);
    if (params.length == 1) {
      // A pair of numbers stands for a range, because the host cannot build a
      // Go struct: it imports no output of a Go module.
      if (kinds.first == 'RangeValues') {
        return const GoType(
          'func(float64, float64)',
          'cbRange',
          pointer: false,
        );
      }
      final arg = mapType(out, params.first.type);
      if (arg == null) return null;
      if (!_cb1Args.contains(arg.kind) && !arg.kind.startsWith('enum:')) {
        return null;
      }
      return GoType('func(${arg.goType})', 'cb1:${arg.kind}', pointer: false);
    }
    return null;
  }
  // A callback that gives a Future runs Go work that the host waits for. The
  // host reads the handle of the goroutine, thus the end of the Go function
  // ends the Future. Read the Task type of flutter/ui/node.go.
  if (ret == 'Future<void>' && params.isEmpty) {
    return const GoType('Task', 'task', pointer: false);
  }
  if (ret == 'Future<bool>' && params.isEmpty) {
    return const GoType('Ask', 'ask', pointer: false);
  }
  // The same work with 1 enum argument, for example the confirmDismiss of a
  // Dismissible. The Go side reads the name of the value of that enum.
  if (ret == 'Future<bool>' && params.length == 1) {
    final arg = mapType(out, params.first.type);
    if (arg != null && arg.kind.startsWith('enum:')) {
      return const GoType('AskText', 'askText', pointer: false);
    }
  }
  if (ret == 'Widget') {
    if (kinds.length == 1 && kinds.first == 'BuildContext') {
      return const GoType('Builder', 'build0', pointer: false);
    }
    if (kinds.length != 2 || kinds.first != 'BuildContext') return null;
    switch (kinds[1]) {
      case 'int':
        return const GoType('ItemBuilder', 'build1', pointer: false);
      case 'Orientation':
        return const GoType('OrientBuilder', 'buildTurn', pointer: false);
      case 'BoxConstraints':
        return const GoType('BoxBuilder', 'buildBox', pointer: false);
    }
    return null;
  }
  if (ret == 'List<Widget>' &&
      kinds.length == 2 &&
      kinds.first == 'BuildContext' &&
      kinds[1] == 'bool') {
    return const GoType('SliverBuilder', 'buildSlivers', pointer: false);
  }
  // The size of 1 item of a list. The host gives the 4 fields of the
  // SliverLayoutDimensions as plain numbers, and a result below 0 stands for
  // the null of Dart. Read decision 273.
  if (ret == 'double' &&
      kinds.length == 2 &&
      kinds.first == 'int' &&
      kinds[1] == 'SliverLayoutDimensions') {
    return const GoType('ExtentBuilder', 'buildExtent', pointer: false);
  }
  return null;
}

/// goStructName joins a class and a named constructor. ListView.builder gives
/// ListViewBuilder, and the host switches on that name.
String goStructName(String className, String ctor) {
  if (ctor.isEmpty || ctor == 'new') return className;
  return className + ctor[0].toUpperCase() + ctor.substring(1);
}

/// goFieldName gives the exported Go name of a Dart parameter.
String goFieldName(String dart) => dart[0].toUpperCase() + dart.substring(1);

/// packageOf gives the name of the Dart package of a library URI.
String packageOf(String uri) => uri.split('/').first.replaceAll('package:', '');

/// stampOf gives the comment line that names each version of the read. glutter
/// doctor reads it, and it stops a build when the machine holds another
/// Flutter. A Flutter that drops a parameter of a constructor gives a Go field
/// that the host does not read, and no error stands.
String stampOf(String root, Pack pack) {
  final parts = <String>['flutter=${flutterVersion()}'];
  final name = packageOf(pack.uri);
  // The framework itself carries the version of Flutter, and its entry of the
  // lock file holds 0.0.0.
  if (name != 'flutter') {
    final version = lockVersion(root, name);
    if (version != null) parts.add('$name=$version');
  }
  return '// nodegen: ${parts.join(" ")}';
}

/// flutterVersion reads the version of the Flutter of the machine.
String flutterVersion() {
  final out = Process.runSync('flutter', <String>['--version', '--machine']);
  if (out.exitCode != 0) return 'unknown';
  final m = jsonDecode(out.stdout as String) as Map<String, dynamic>;
  return (m['frameworkVersion'] as String?) ?? 'unknown';
}

/// lockVersion reads the version of 1 package of the lock file of root.
String? lockVersion(String root, String name) {
  final file = File('$root/pubspec.lock');
  if (!file.existsSync()) return null;
  var found = false;
  for (final line in const LineSplitter().convert(file.readAsStringSync())) {
    if (line == '  $name:') {
      found = true;
      continue;
    }
    if (!found) continue;
    final trimmed = line.trim();
    if (trimmed.startsWith('version:')) {
      return trimmed.substring(8).trim().replaceAll('"', '');
    }
    if (line.length > 2 && line[2] != ' ') return null;
  }
  return null;
}

/// ClassDocs holds the dartdoc of 1 Flutter class and of each field of it.
class ClassDocs {
  ClassDocs(this.owner, this.fields);

  /// The dartdoc of the class.
  final String owner;

  /// The dartdoc of each field, by the name of the field. A constructor
  /// parameter carries no doc of its own, and the field of the same name
  /// carries 1.
  final Map<String, String> fields;
}

/// docsOf reads the dartdoc of a class and of each field of it, and of each
/// class above it.
ClassDocs docsOf(InterfaceElement element, Set<String> sources) {
  sources.add(element.firstFragment.libraryFragment.source.fullName);
  final fields = <String, String>{};
  void add(InterfaceElement c) {
    for (final f in c.fields) {
      final doc = f.documentationComment;
      final name = f.name;
      if (doc == null || name == null) continue;
      sources.add(f.firstFragment.libraryFragment.source.fullName);
      fields.putIfAbsent(name, () => doc);
    }
  }

  add(element);
  for (final t in element.allSupertypes) {
    add(t.element);
  }
  return ClassDocs(element.documentationComment ?? '', fields);
}

/// firstParagraph gives the first paragraph of 1 dartdoc, with no markup.
String firstParagraph(String doc) => clean(doc, sentence: false);

/// firstSentence gives the first sentence of 1 dartdoc, with no markup.
String firstSentence(String doc) => clean(doc, sentence: true);

/// The blocks of dartdoc that carry no text for a reader of the Go source.
final _blocks = RegExp(r'\{@[a-z]+[^}]*\}');

/// clean reads the head of 1 dartdoc and it gives plain text.
///
/// A dartdoc holds markdown, a sample block, and a link of the form [Name].
/// The Go doc comment holds the first paragraph or the first sentence of that
/// text, and no markup.
String clean(String doc, {required bool sentence}) {
  final lines = <String>[];
  for (final raw in doc.split('\n')) {
    var line = raw.trim();
    if (line.startsWith('///')) line = line.substring(3).trim();
    if (line.startsWith('//')) line = line.substring(2).trim();
    line = line.replaceAll(_blocks, '').trim();
    if (line.isEmpty) {
      if (lines.isEmpty) continue;
      break;
    }
    // A list, a heading, and a code fence each need more than 1 line, thus the
    // head of the doc stops here.
    if (line.startsWith('*') ||
        line.startsWith('-') ||
        line.startsWith('#') ||
        line.startsWith('```') ||
        line.startsWith('>')) {
      break;
    }
    lines.add(line);
    if (sentence && line.contains('. ')) break;
  }
  var text = lines.join(' ');
  text = text.replaceAllMapped(
    RegExp(r'\[([^\]]+)\]'),
    (m) => m.group(1)!.split('.').last,
  );
  text = text.replaceAll('`', '').replaceAll(RegExp(r'\s+'), ' ').trim();
  if (sentence) {
    final stop = text.indexOf('. ');
    if (stop > 0) text = text.substring(0, stop + 1);
  }
  if (text.length > 240) {
    final cut = text.lastIndexOf(' ', 240);
    text = '${text.substring(0, cut > 0 ? cut : 240)} ...';
  }
  return text;
}

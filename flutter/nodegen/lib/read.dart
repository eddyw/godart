import 'dart:io';

import 'package:analyzer/dart/analysis/analysis_context_collection.dart';
import 'package:analyzer/dart/analysis/results.dart';
import 'package:analyzer/dart/element/element.dart';
import 'package:analyzer/dart/element/nullability_suffix.dart';
import 'package:analyzer/dart/element/type.dart';

/// The Flutter libraries that the node set covers.
const libraries = <String>[
  'package:flutter/widgets.dart',
  'package:flutter/material.dart',
  'package:flutter/cupertino.dart',
];

/// The libraries that the generated host may import, in the order of the
/// search. A name that no 1 of them exports does not compile in the host, thus
/// the reader drops the field that reads it.
const candidates = <String>[
  'package:flutter/material.dart',
  'package:flutter/cupertino.dart',
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
  });

  /// The name of the Dart parameter.
  final String dartName;

  /// The name of the Go field.
  final String goName;

  final GoType type;

  /// A parameter that the constructor demands.
  final bool required;

  /// A Dart type that accepts null. Such a parameter needs no default.
  final bool nullable;

  /// The default of the Dart parameter, verbatim, or null.
  final String? defaultCode;

  /// A named parameter. A positional parameter takes its place in the call.
  final bool named;

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
    required this.library,
    required this.params,
    required this.isConst,
  });

  /// The Flutter class, for example ListView.
  final String dartClass;

  /// The named constructor, or an empty text for the unnamed 1.
  final String ctor;

  /// The Go struct, for example ListViewBuilder.
  final String goName;

  /// The Dart library that exports the class.
  final String library;

  final List<Param> params;
  final bool isConst;

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
}

/// read walks the 3 Flutter libraries and maps each widget constructor.
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

  for (final uri in libraries) {
    final result = await session.getLibraryByUri(uri);
    if (result is! LibraryElementResult) {
      throw StateError('$uri does not resolve: $result');
    }
    final short = uri.split('/').last.replaceAll('.dart', '');
    for (final entry in result.element.exportNamespace.definedNames2.entries) {
      final element = entry.value;
      if (element is! ClassElement) continue;
      final name = element.name;
      if (name == null || name.startsWith('_')) continue;
      if (!isWidget(element) || element.isAbstract) continue;
      if (_skip.contains(name)) continue;
      for (final ctor in element.constructors) {
        if (!ctor.isPublic) continue;
        if (ctor.formalParameters
            .any((p) => p.name == 'key' && p.isRequired)) {
          continue;
        }
        final goName = goStructName(name, ctor.name ?? '');
        if (!seen.add(goName)) continue;
        readCtor(out, short, name, ctor, goName);
      }
    }
  }
  out.widgets.sort((a, b) => a.goName.compareTo(b.goName));
  return out;
}

void readCtor(
  Reading out,
  String library,
  String className,
  ConstructorElement ctor,
  String goName,
) {
  final params = <Param>[];
  for (final p in ctor.formalParameters) {
    final pname = p.name;
    if (pname == null || pname.startsWith('_')) continue;
    if (pname == 'key') continue;
    final nullable = p.type.nullabilitySuffix == NullabilitySuffix.question;
    final go = mapType(out, p.type);
    if (go == null) {
      final key = p.type.getDisplayString();
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
    if (!p.isRequired && !nullable && !usableDefault(out, p.defaultValueCode)) {
      // The host cannot write the call without the default of Flutter, and a
      // default that names a private symbol does not compile outside its
      // library.
      continue;
    }
    params.add(Param(
      dartName: pname,
      goName: goFieldName(pname),
      type: go,
      required: p.isRequired,
      nullable: nullable,
      defaultCode: p.defaultValueCode,
      named: p.isNamed,
    ));
  }
  // A constructor with no mapped parameter still reaches the node set. Each
  // parameter of it then takes the default of Flutter, thus the node holds the
  // whole meaning of the widget. ui.CupertinoActivityIndicator is such a node.
  out.widgets.add(WidgetSpec(
    dartClass: className,
    ctor: ctor.name == 'new' ? '' : (ctor.name ?? ''),
    goName: goName,
    library: library,
    params: params,
    isConst: ctor.isConst,
  ));
}

/// The widgets that the Go model cannot use. Builder exists only to give a
/// BuildContext, and a Go value holds none. The Go struct of Theme would carry
/// the name of the accessor ui.Theme, which gives the theme that the host
/// wrote. A Go program reads the field Theme of ui.MaterialApp, or the node
/// ui.AnimatedTheme.
const _skip = <String>{'Builder', 'Theme'};

/// The words of a default that name no declaration.
const _words = <String>{
  'const', 'true', 'false', 'null', 'new', 'if', 'else', 'is', 'as',
};

/// usableDefault reports whether the host can write the default verbatim.
///
/// A default that names a private symbol does not compile outside its library,
/// and a default that names a static member of the class without the class,
/// for example defaultThickness, does not resolve in the host either.
bool usableDefault(Reading out, String? code) {
  if (code == null) return false;
  if (RegExp(r'(^|[^A-Za-z0-9])_').hasMatch(code)) return false;
  for (final m in RegExp(r'(?<![.\w])[A-Za-z][A-Za-z0-9]*').allMatches(code)) {
    final word = m.group(0)!;
    if (_words.contains(word)) continue;
    final lib = out.visible[word];
    if (lib == null) return false;
    out.needed.add(lib);
  }
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
  'BottomNavigationBarItem': GoType('BarItem', 'barItem'),
  'NavigationRailDestination': GoType('RailItem', 'railItem'),
  'Step': GoType('Step', 'step'),
  'ContextMenuButtonItem': GoType('MenuButtonItem', 'menuItem'),
  'List<DataColumn>': GoType('[]DataColumn', 'columns', pointer: false),
  'List<DataRow>': GoType('[]DataRow', 'rows', pointer: false),
  'List<ContextMenuButtonItem>':
      GoType('[]MenuButtonItem', 'menuItems', pointer: false),
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
  'List<BottomNavigationBarItem>':
      GoType('[]BarItem', 'barItems', pointer: false),
  'List<NavigationRailDestination>':
      GoType('[]RailItem', 'railItems', pointer: false),
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
  final raw = _scalars[exact.endsWith('?')
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
  if (element is InterfaceElement && element is! EnumElement &&
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
        return const GoType('func(float64, float64)', 'cbRange',
            pointer: false);
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
  // Go side reports the end itself, because the host sees no end of a
  // goroutine. Read the Task type of flutter/ui/node.go.
  if (ret == 'Future<void>' && params.isEmpty) {
    return const GoType('Task', 'task', pointer: false);
  }
  if (ret == 'Future<bool>' && params.isEmpty) {
    return const GoType('Ask', 'ask', pointer: false);
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

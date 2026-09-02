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
  Skipped(this.name, this.cause);
  final String name;
  final String cause;
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
        out.skipped.add(Skipped(goName, 'the parameter $pname holds $key'));
        return;
      }
      continue;
    }
    if (!p.isNamed && !p.isRequired) {
      // An optional positional parameter has no stable place in a call that
      // leaves an earlier 1 out.
      out.skipped
          .add(Skipped(goName, 'the parameter $pname is optional and positional'));
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
  if (params.isEmpty) {
    out.skipped.add(Skipped(goName, 'no parameter maps'));
    return;
  }
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
/// BuildContext, and a Go value holds none.
const _skip = <String>{'Builder'};

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
  'WidgetStateProperty<Color>': GoType('StateColor', 'stateColor'),
  'Size': GoType('Size', 'boxSize'),
  'ButtonStyle': GoType('ButtonStyle', 'buttonStyle'),
  'ImageProvider<Object>': GoType('ImageSource', 'image'),
  'Decoration': GoType('Decoration', 'decoration'),
  'BoxDecoration': GoType('Decoration', 'decoration'),
  // The 3 objects below hold an identity that no Go value carries. The Go
  // side names 1, and the host owns the object. Read flutter/ui/identity.go.
  'FocusNode': GoType('FocusKey', 'focusKey'),
  'ScrollController': GoType('ScrollKey', 'scrollKey'),
  'WidgetStatesController': GoType('StatesKey', 'statesKey'),
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
};

/// mapType gives the Go type of a Dart type, or null when no form fits.
GoType? mapType(Reading out, DartType t) {
  if (t is FunctionType) return mapFunction(out, t);

  final display = t.getDisplayString().replaceAll('?', '');
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

  if (ret == 'void') {
    if (params.isEmpty) return const GoType('func()', 'cb0', pointer: false);
    if (params.length == 1) {
      final arg = mapType(out, params.first.type);
      if (arg == null) return null;
      if (!_cb1Args.contains(arg.kind) && !arg.kind.startsWith('enum:')) {
        return null;
      }
      return GoType('func(${arg.goType})', 'cb1:${arg.kind}', pointer: false);
    }
    return null;
  }
  if (ret == 'Widget') {
    final kinds = params.map((p) => p.type.getDisplayString()).toList();
    if (kinds.length == 1 && kinds.first == 'BuildContext') {
      return const GoType('Builder', 'build0', pointer: false);
    }
    if (kinds.length == 2 &&
        kinds.first == 'BuildContext' &&
        kinds[1] == 'int') {
      return const GoType('ItemBuilder', 'build1', pointer: false);
    }
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

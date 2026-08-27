import 'iface.dart';
import 'int.dart';
import 'map.dart';
import 'print.dart';
import 'slice.dart';
import 'string.dart';

/// Writes 1 value in the format of a Go verb.
///
/// The verb holds the letter and the flags that come before it, for example
/// `v`, `+v`, or `q`. The Go package fmt reads the format text and gives the
/// width and the precision, thus this function needs neither.
GoString $goFormat(Iface? value, GoString verb) =>
    GoString.fromString(_format(value, verb.toDartString()));

/// Reads a float from text. The second value is false when the text holds no
/// float.
(double, bool) $parseFloat(GoString text) {
  final double? v = double.tryParse(text.toDartString().trim());
  if (v == null) return (0.0, false);
  return (v, true);
}

/// The kind of the value of an interface, in the words of the package reflect.
GoString $goKind(Iface? value) =>
    GoString.fromString(value == null ? 'nil' : value.desc.kind);

String _format(Iface? x, String verb) {
  if (verb.endsWith('T')) return x == null ? '<nil>' : x.desc.display;
  if (x == null) return verb.endsWith('v') ? '<nil>' : '%!$verb(<nil>)';
  return _value(x.value, x.desc, verb);
}

/// The number after the point of a verb, for example 2 in `.2f`. The result is
/// -1 when the verb holds no precision.
int _precision(String verb) {
  final int dot = verb.indexOf('.');
  if (dot < 0) return -1;
  var out = 0;
  for (var i = dot + 1; i < verb.length - 1; i++) {
    final int digit = verb.codeUnitAt(i) - 0x30;
    if (digit < 0 || digit > 9) break;
    out = out * 10 + digit;
  }
  return out;
}

String _value(Object? v, TypeDesc type, String verb) {
  final bool plus = verb.contains('+');
  final String letter = verb.substring(verb.length - 1);
  if (letter == 'T') return type.display;
  if (letter == 'v' || letter == 's' || letter == 'q') {
    final String? text = _viaMethod(v, type, letter);
    if (text != null) return text;
  }
  switch (type.kind) {
    case 'bool':
      return _bool(v as bool, letter);
    case 'string':
      return _string(v as GoString, letter);
    case 'float32':
    case 'float64':
      return _float(v as double, letter, _precision(verb));
    case 'uint':
    case 'uint64':
    case 'uintptr':
      return _int(v as int, letter, true);
    case 'int':
    case 'int8':
    case 'int16':
    case 'int32':
    case 'int64':
    case 'uint8':
    case 'uint16':
    case 'uint32':
      return _int(v as int, letter, false);
    case 'slice':
      return _slice(v, type, verb, plus);
    case 'array':
      return _array(v as List<Object?>, type, verb, plus);
    case 'map':
      return _map(v, type, verb, plus);
    case 'struct':
      return _struct(v, type, verb, plus);
    case 'ptr':
      if (v == null) return '<nil>';
      final TypeDesc? inner = type.elem?.call();
      if (inner != null && inner.kind == 'struct') {
        return '&${_struct(v, inner, verb, plus)}';
      }
      return '&${inner == null ? v : _value(v, inner, verb)}';
    case 'interface':
      if (v == null) return '<nil>';
      if (v is Iface) return _value(v.value, v.desc, verb);
      return '$v';
    case 'chan':
      return v == null ? '<nil>' : '<chan>';
    case 'func':
      return v == null ? '<nil>' : '<func>';
  }
  return '$v';
}

/// Reads the text of a value that holds the method Error or String. The Go
/// package fmt gives such a method the first place.
String? _viaMethod(Object? v, TypeDesc type, String letter) {
  for (final name in const <String>['Error', 'String']) {
    final Function? m = type.methods[name];
    if (m == null) continue;
    final Object? text = Function.apply(m, <Object?>[v]);
    if (text is! GoString) continue;
    return letter == 'q' ? _quote(text) : text.toDartString();
  }
  return null;
}

String _bool(bool v, String letter) {
  if (letter == 'v' || letter == 't') return v ? 'true' : 'false';
  return '%!$letter(bool=$v)';
}

String _string(GoString v, String letter) {
  switch (letter) {
    case 'v':
    case 's':
      return v.toDartString();
    case 'q':
      return _quote(v);
    case 'x':
      return _hexBytes(v, false);
    case 'X':
      return _hexBytes(v, true);
  }
  return '%!$letter(string=${v.toDartString()})';
}

String _int(int v, String letter, bool unsigned) {
  switch (letter) {
    case 'v':
    case 'd':
      return unsigned ? $ustring(v) : v.toString();
    case 'x':
      return _hexInt(v, unsigned, false);
    case 'X':
      return _hexInt(v, unsigned, true);
    case 'b':
      return v < 0 && !unsigned
          ? '-${(-v).toRadixString(2)}'
          : v.toRadixString(2);
    case 'o':
      return v < 0 && !unsigned
          ? '-${(-v).toRadixString(8)}'
          : v.toRadixString(8);
    case 'c':
      return $runeToString(v).toDartString();
    case 'q':
      return "'${$runeToString(v).toDartString()}'";
    case 'f':
    case 'e':
    case 'g':
      return _float(v.toDouble(), letter, -1);
  }
  return '%!$letter(int=$v)';
}

String _float(double v, String letter, int precision) {
  switch (letter) {
    case 'v':
    case 'g':
      return $fstring(v);
    case 'f':
      if (v.isNaN) return 'NaN';
      if (v.isInfinite) return v > 0 ? '+Inf' : '-Inf';
      return v.toStringAsFixed(precision < 0 ? 6 : precision);
    case 'e':
      return _exponent(v, precision < 0 ? 6 : precision);
    case 'd':
      return '%!d(float64=${$fstring(v)})';
  }
  return '%!$letter(float64=${$fstring(v)})';
}

String _exponent(double v, int digits) {
  if (v.isNaN) return 'NaN';
  if (v.isInfinite) return v > 0 ? '+Inf' : '-Inf';
  final String text = v.abs().toStringAsExponential(digits);
  final int e = text.indexOf('e');
  final int exponent = int.parse(text.substring(e + 1));
  final String sign = v.isNegative ? '-' : '';
  final String exponentSign = exponent < 0 ? '-' : '+';
  final String pad = exponent.abs().toString().padLeft(2, '0');
  return '$sign${text.substring(0, e)}e$exponentSign$pad';
}

String _hexInt(int v, bool unsigned, bool upper) {
  String text;
  if (v < 0 && !unsigned) {
    text = '-${(-v).toRadixString(16)}';
  } else if (v < 0) {
    text = (v.toUnsigned(64)).toRadixString(16);
  } else {
    text = v.toRadixString(16);
  }
  return upper ? text.toUpperCase() : text;
}

String _hexBytes(GoString v, bool upper) {
  final StringBuffer out = StringBuffer();
  for (final b in v.bytes) {
    out.write(b.toRadixString(16).padLeft(2, '0'));
  }
  final String text = out.toString();
  return upper ? text.toUpperCase() : text;
}

String _quote(GoString v) {
  final StringBuffer out = StringBuffer('"');
  for (final r in v.toDartString().runes) {
    switch (r) {
      case 0x22:
        out.write(r'\"');
      case 0x5C:
        out.write(r'\\');
      case 0x0A:
        out.write(r'\n');
      case 0x09:
        out.write(r'\t');
      case 0x0D:
        out.write(r'\r');
      default:
        if (r < 0x20) {
          out.write('\\x${r.toRadixString(16).padLeft(2, '0')}');
          break;
        }
        out.writeCharCode(r);
    }
  }
  out.write('"');
  return out.toString();
}

String _slice(Object? v, TypeDesc type, String verb, bool plus) {
  if (v is! Slice<Object?>) return '$v';
  final TypeDesc? elem = type.elem?.call();
  final List<String> parts = <String>[];
  for (var i = 0; i < v.len; i++) {
    final Object? item = v.array[v.off + i];
    parts.add(elem == null ? '$item' : _value(item, elem, verb));
  }
  return '[${parts.join(' ')}]';
}

String _array(List<Object?> v, TypeDesc type, String verb, bool plus) {
  final TypeDesc? elem = type.elem?.call();
  final List<String> parts = <String>[];
  for (final item in v) {
    parts.add(elem == null ? '$item' : _value(item, elem, verb));
  }
  return '[${parts.join(' ')}]';
}

String _map(Object? v, TypeDesc type, String verb, bool plus) {
  if (v is! GoMap<Object?, Object?>) return '$v';
  final TypeDesc? keyType = type.key?.call();
  final TypeDesc? elem = type.elem?.call();
  final Slice<Object?> keys = v.keys();
  final List<String> parts = <String>[];
  for (var i = 0; i < keys.len; i++) {
    final Object? k = keys.array[keys.off + i];
    final Object? value = v.get(k);
    final String keyText = keyType == null ? '$k' : _value(k, keyType, verb);
    final String valueText = elem == null
        ? '$value'
        : _value(value, elem, verb);
    parts.add('$keyText:$valueText');
  }
  parts.sort();
  return 'map[${parts.join(' ')}]';
}

String _struct(Object? v, TypeDesc type, String verb, bool plus) {
  final List<String> parts = <String>[];
  for (final field in type.fields) {
    final Object? value = field.get(v);
    final String text = _value(value, field.type(), verb);
    parts.add(plus ? '${field.name}:$text' : text);
  }
  return '{${parts.join(' ')}}';
}

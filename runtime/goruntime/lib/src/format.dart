import 'box.dart';
import 'iface.dart';
import 'int.dart';
import 'map.dart';
import 'panic.dart';
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

/// The deep comparison of the package reflect.
///
/// 2 values of 2 types are never equal. A slice, a map, a struct, an array, a
/// pointer, and an interface each compare their parts, and each other kind
/// compares with the equality of Go.
bool $deepEqual(Iface? a, Iface? b) {
  if (a == null || b == null) return a == null && b == null;
  if (a.desc.name != b.desc.name) return false;
  return _deepValue(a.value, b.value, a.desc);
}

bool _deepValue(Object? x, Object? y, TypeDesc desc) {
  switch (desc.kind) {
    case 'slice':
      if (x is! Slice || y is! Slice) return x == y;
      if (x.isNil != y.isNil) return false;
      if (x.len != y.len) return false;
      final TypeDesc elem = desc.elem!();
      for (var i = 0; i < x.len; i++) {
        if (!_deepValue(x[i], y[i], elem)) return false;
      }
      return true;
    case 'array':
      if (x is! List || y is! List) return x == y;
      if (x.length != y.length) return false;
      final TypeDesc elem = desc.elem!();
      for (var i = 0; i < x.length; i++) {
        if (!_deepValue(x[i], y[i], elem)) return false;
      }
      return true;
    case 'map':
      if (x is! GoMap || y is! GoMap) return x == y;
      if (x.isNil != y.isNil) return false;
      if (x.length != y.length) return false;
      final TypeDesc elem = desc.elem!();
      final Slice<Object?> keys = x.keys() as Slice<Object?>;
      for (var i = 0; i < keys.len; i++) {
        final Object? key = keys[i];
        if (!y.has(key)) return false;
        if (!_deepValue(x.get(key), y.get(key), elem)) return false;
      }
      return true;
    case 'struct':
      if (x == null || y == null) return x == y;
      for (final FieldDesc field in desc.fields) {
        if (!_deepValue(field.get(x), field.get(y), field.type())) return false;
      }
      return true;
    case 'ptr':
      if (identical(x, y)) return true;
      if (x == null || y == null) return false;
      if (x is $Ptr<Object?> && y is $Ptr<Object?>) {
        return _deepValue(x.v, y.v, desc.elem!());
      }
      return _deepValue(x, y, desc.elem!());
    case 'interface':
      return $deepEqual(x as Iface?, y as Iface?);
    case 'func':
      return x == null && y == null;
  }
  if (x is Iface || y is Iface) return $deepEqual(x as Iface?, y as Iface?);
  return x == y;
}

/// Writes a value into the target of errors.As.
///
/// The target holds a pointer, and the write reads the box of it. A target of a
/// pointer to an interface type panics: the runtime library holds no method
/// name of such a type, thus the check of the method set has no source.
bool $assignError(Iface? value, Iface? target) {
  if (value == null || target == null) return false;
  final TypeDesc desc = target.desc;
  if (desc.kind != 'ptr' || desc.elem == null) {
    $panic(
      Iface(
        $typeGoError,
        const GoError('errors: target must be a non-nil pointer'),
      ),
    );
  }
  final TypeDesc elem = desc.elem!();
  if (elem.kind == 'interface') {
    $panic(
      Iface(
        $typeGoError,
        const GoError(
          'errors.As with a pointer to an interface type is '
          'outside the subset',
        ),
      ),
    );
  }
  if (elem.name != value.desc.name) return false;
  final Object? box = target.value;
  if (box is $Ptr<Object?>) {
    box.v = value.value;
    return true;
  }
  return false;
}

/// The text of the String method of the zero value of the type of a value. The
/// package flag reads it, and Go builds the zero value through reflect.New.
GoString $zeroText(Iface? value) {
  if (value == null) return GoString.empty;
  final TypeDesc desc = value.desc;
  final Function? method = desc.methods['String()string'];
  if (method == null) return GoString.empty;
  Object? zero;
  if (desc.kind == 'ptr' && desc.elem != null) {
    zero = desc.elem!().zero?.call();
  } else {
    zero = desc.zero?.call();
  }
  return method(zero) as GoString;
}

/// The text of the type of an interface value, and the comparable flag. The
/// package internal/reflectlite reads it.
(GoString, bool) $typeInfo(Iface? value) {
  if (value == null) $panicNil();
  return value.desc.info;
}

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

/// _base writes the prefix of the base in front of a whole number. Go writes it
/// for the flag # only.
String _base(String text, String verb, String letter) {
  if (!verb.contains('#')) return text;
  String prefix;
  switch (letter) {
    case 'x':
      prefix = '0x';
    case 'X':
      prefix = '0X';
    case 'o':
      prefix = '0';
    case 'O':
      prefix = '0o';
    case 'b':
      prefix = '0b';
    default:
      return text;
  }
  if (text.startsWith('-')) return '-$prefix${text.substring(1)}';
  return '$prefix$text';
}

/// _sign writes the flag + in front of a value that is not negative. Go writes
/// that flag for a numeric verb only, and not for the verb v.
String _sign(String text, bool plus, String letter) {
  if (!plus || text.startsWith('-')) return text;
  switch (letter) {
    case 'd':
    case 'x':
    case 'X':
    case 'o':
    case 'O':
    case 'b':
    case 'e':
    case 'E':
    case 'f':
    case 'F':
    case 'g':
    case 'G':
      return '+$text';
  }
  return text;
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
      return _sign(
        _float(v as double, letter, _precision(verb), type.kind == 'float32'),
        plus,
        letter,
      );
    case 'uint':
    case 'uint64':
    case 'uintptr':
      return _sign(
        _base(_int(v as int, letter, true), verb, letter),
        plus,
        letter,
      );
    case 'int':
    case 'int8':
    case 'int16':
    case 'int32':
    case 'int64':
    case 'uint8':
    case 'uint16':
    case 'uint32':
      return _sign(
        _base(_int(v as int, letter, false), verb, letter),
        plus,
        letter,
      );
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
  // The key of a method table holds the name and the signature.
  for (final name in const <String>['Error()string', 'String()string']) {
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
      return _float(v.toDouble(), letter, -1, false);
  }
  return '%!$letter(int=$v)';
}

/// The text of a 32-bit float, in the format of Go.
///
/// Go writes the shortest decimal text that reads back as the same value of the
/// width of the value. A Dart double holds a 32-bit value with the precision of
/// 64 bits, thus the shortest text of 64 bits holds more digits than Go writes.
String $f32string(double v) {
  if (v.isNaN || v.isInfinite || v == 0) return $fstring(v);
  for (int digits = 1; digits <= 9; digits++) {
    final double short = double.parse(v.toStringAsPrecision(digits));
    if ($f32FromBits($f32ToBits(short)) == v) return $fstring(short);
  }
  return $fstring(v);
}

String _float(double v, String letter, int precision, bool single) {
  switch (letter) {
    case 'v':
    case 'g':
      return single ? $f32string(v) : $fstring(v);
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
  int at = 0;
  while (at < v.length) {
    final (int r, int size) = $decodeRune(v, at);
    at += size;
    // Go writes the escape of a byte that holds no rune of UTF-8.
    if (r == 0xFFFD && size == 1) {
      out.write('\\x${v[at - 1].toRadixString(16).padLeft(2, '0')}');
      continue;
    }
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
  final List<Object?> items = <Object?>[];
  for (var i = 0; i < v.len; i++) {
    items.add(v.array[v.off + i]);
  }
  return _list(items, elem, verb);
}

String _array(List<Object?> v, TypeDesc type, String verb, bool plus) {
  return _list(v, type.elem?.call(), verb);
}

/// _list writes each element of a slice or of an array. Go reads a list of
/// bytes as text for the verbs s, q, x, and X.
String _list(List<Object?> items, TypeDesc? elem, String verb) {
  final String letter = verb.substring(verb.length - 1);
  if (elem != null && elem.kind == 'uint8' && _textVerb(letter)) {
    final List<int> bytes = <int>[];
    for (final Object? item in items) {
      bytes.add((item as int) & 0xff);
    }
    return _string(GoString.fromBytes(bytes), letter);
  }
  final List<String> parts = <String>[];
  for (final item in items) {
    parts.add(elem == null ? '$item' : _value(item, elem, verb));
  }
  return '[${parts.join(' ')}]';
}

bool _textVerb(String letter) =>
    letter == 's' || letter == 'q' || letter == 'x' || letter == 'X';

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

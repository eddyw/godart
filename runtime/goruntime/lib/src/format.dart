import 'box.dart';
import 'chan.dart';
import 'complex.dart';
import 'goint.dart';
import 'iface.dart';
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
      // Go writes the zero of an octal only when the digits hold none, thus
      // the value 0 gives the text 0 and never 00.
      if (text.startsWith('0') || text.startsWith('-0')) return text;
      prefix = '0';
    case 'O':
      // _int already wrote 0o, because Go writes that prefix with no flag.
      // The flag # adds the leading zero of an octal on top of it.
      final bool neg = text.startsWith('-');
      final String digits = text.substring(neg ? 3 : 2);
      if (digits.startsWith('0')) return text;
      return '${neg ? '-' : ''}0o0$digits';
    case 'b':
      prefix = '0b';
    default:
      return text;
  }
  if (text.startsWith('-')) return '-$prefix${text.substring(1)}';
  return '$prefix$text';
}

/// _sign writes the flag + or the flag space in front of a value that is not
/// negative. Go writes each 1 for a numeric verb only, and not for the verb v.
/// The flag + wins when the verb holds both.
String _sign(String text, String verb, String letter) {
  if (text.startsWith('-')) return text;
  final bool plus = verb.contains('+');
  final bool space = verb.contains(' ');
  if (!plus && !space) return text;
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
      return plus ? '+$text' : ' $text';
  }
  return text;
}

String _value(Object? v, TypeDesc type, String verb, [int depth = 0]) {
  final bool plus = verb.contains('+');
  final String letter = verb.substring(verb.length - 1);
  if (letter == 'T') return type.display;
  final bool sharp = _goSyntax(verb);
  if (sharp) {
    final String? text = _goStringMethod(v, type);
    if (text != null) return text;
  } else if (letter == 'v' || letter == 's' || letter == 'q') {
    final String? text = _viaMethod(v, type, letter);
    if (text != null) return text;
  }
  if (sharp && _unsigned(type.kind)) return '0x${$uradix(v as GoInt, 16)}';
  switch (type.kind) {
    case 'bool':
      return _bool(v as bool, letter);
    case 'string':
      return _string(v as GoString, verb, letter);
    case 'float32':
    case 'float64':
      return _sign(
        _float(v as double, letter, _precision(verb), type.kind == 'float32'),
        verb,
        letter,
      );
    case 'complex64':
    case 'complex128':
      return _complex(v as GoComplex, type.kind, letter, verb);
    case 'uint':
    case 'uint64':
    case 'uintptr':
      return _sign(
        _base(_int(v as GoInt, letter, true, verb.contains('#')), verb, letter),
        verb,
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
        _base(
          _int(v as GoInt, letter, false, verb.contains('#')),
          verb,
          letter,
        ),
        verb,
        letter,
      );
    case 'slice':
      return _slice(v, type, verb, depth);
    case 'array':
      return _array(v as List<Object?>, type, verb, depth);
    case 'map':
      return _map(v, type, verb, depth);
    case 'struct':
      return _struct(v, type, verb, plus, depth);
    case 'ptr':
      final TypeDesc? inner = type.elem?.call();
      if (v == null) return sharp ? '(${type.display})(nil)' : '<nil>';
      if (sharp) {
        // The Dart form of a pointer to a struct is the object of the struct,
        // and each other pointer holds a cell.
        final Object? target = v is $Ptr<Object?> ? v.v : v;
        if (inner == null) return '&$target';
        return '&${_value(target, inner, verb, depth + 1)}';
      }
      if (inner != null && inner.kind == 'struct') {
        return '&${_struct(v, inner, verb, plus, depth)}';
      }
      return '&${inner == null ? v : _value(v, inner, verb, depth + 1)}';
    case 'interface':
      if (v == null) return sharp ? '${type.display}(nil)' : '<nil>';
      if (v is Iface) return _value(v.value, v.desc, verb, depth + 1);
      return '$v';
    case 'chan':
      // The Dart form of a nil channel is a Chan that holds the nil flag, and
      // not a null reference.
      if (v == null || (v is Chan<Object?> && v.isNil)) {
        return sharp ? '(${type.display})(nil)' : '<nil>';
      }
      return '<chan>';
    case 'func':
      if (v == null) return sharp ? '(${type.display})(nil)' : '<nil>';
      return '<func>';
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

/// True for the verb %#v, which asks for the Go syntax form of a value.
bool _goSyntax(String verb) => verb.endsWith('v') && verb.contains('#');

/// True for a kind that Go writes as hexadecimal in the Go syntax form.
bool _unsigned(String kind) {
  switch (kind) {
    case 'uint':
    case 'uint8':
    case 'uint16':
    case 'uint32':
    case 'uint64':
    case 'uintptr':
      return true;
  }
  return false;
}

/// The text of the method GoString. The verb %#v of Go reads that method, and
/// it reads neither String nor Error.
String? _goStringMethod(Object? v, TypeDesc type) {
  final Function? m = type.methods['GoString()string'];
  if (m == null) return null;
  final Object? text = Function.apply(m, <Object?>[v]);
  if (text is! GoString) return null;
  return text.toDartString();
}

/// The type name that stands in front of a composite literal.
///
/// reflect names a slice of bytes `[]uint8`, and the package fmt holds a case
/// of its own for the type `[]byte` of an argument. That case writes `[]byte`,
/// thus the name differs at the top only.
String _goName(TypeDesc type, int depth) =>
    depth == 0 && type.display == '[]uint8' ? '[]byte' : type.display;

/// The Go syntax form of a list of elements, with the name of the type.
String _goList(
  List<Object?> items,
  TypeDesc? elem,
  String name,
  String verb,
  int depth,
) {
  final List<String> parts = <String>[
    for (final item in items)
      elem == null ? '$item' : _value(item, elem, verb, depth + 1),
  ];
  return '$name{${parts.join(', ')}}';
}

String _bool(bool v, String letter) {
  if (letter == 'v' || letter == 't') return v ? 'true' : 'false';
  return '%!$letter(bool=$v)';
}

String _string(GoString v, String verb, String letter) {
  switch (letter) {
    case 'v':
      if (_goSyntax(verb)) return _quote(v);
      return v.toDartString();
    case 's':
      return v.toDartString();
    case 'q':
      return _quote(v);
    case 'x':
      return _base(_hexBytes(v, false), verb, letter);
    case 'X':
      return _base(_hexBytes(v, true), verb, letter);
  }
  return '%!$letter(string=${v.toDartString()})';
}

String _int(GoInt v, String letter, bool unsigned, bool sharp) {
  switch (letter) {
    case 'v':
    case 'd':
      return unsigned ? $ustring(v) : v.toString();
    case 'x':
      return _hexInt(v, unsigned, false);
    case 'X':
      return _hexInt(v, unsigned, true);
    case 'b':
      return _radix(v, 2, unsigned);
    case 'o':
      return _radix(v, 8, unsigned);
    case 'O':
      // Go writes the prefix of this verb with no flag, thus _base adds none.
      final text = _radix(v, 8, unsigned);
      return text.startsWith('-') ? '-0o${text.substring(1)}' : '0o$text';
    case 'c':
      return $runeToString(v).toDartString();
    case 'q':
      return _quoteRune($ix(v));
    case 'U':
      return _unicode(v, sharp);
    case 'f':
    case 'e':
    case 'g':
      return _float(v.toDouble(), letter, -1, false);
  }
  return '%!$letter(int=$v)';
}

/// The text of a code point. Go writes `U+0061`, with 4 hexadecimal digits at
/// least, and the flag # adds the character of a printable rune.
String _unicode(GoInt v, bool sharp) {
  final int n = $ix(v);
  final String hex = n.toRadixString(16).toUpperCase().padLeft(4, '0');
  if (!sharp || n < 0x20 || n > 0x10FFFF || (n >= 0x7F && n < 0xA0)) {
    return 'U+$hex';
  }
  return "U+$hex '${$runeToString(v).toDartString()}'";
}

/// The text of an integer in a radix. An unsigned value reads its 64-bit
/// pattern, and a negative signed value takes a minus sign.
/// The quoted form of 1 code point, in the escape table of Go.
///
/// The table of the graphic categories of Unicode lives in the package
/// strconv, and the overlay of fmt reads it for a rune of a call. A rune
/// inside a slice, a map, or a struct reaches this function, thus a rune above
/// the ASCII range writes itself here.
String _quoteRune(int r) {
  const Map<int, String> escapes = <int, String>{
    0x07: r'\a',
    0x08: r'\b',
    0x0C: r'\f',
    0x0A: r'\n',
    0x0D: r'\r',
    0x09: r'\t',
    0x0B: r'\v',
    0x27: r"\'",
    0x5C: r'\\',
  };
  final String? e = escapes[r];
  if (e != null) return "'$e'";
  if (r < 0x20 || r == 0x7F) {
    final String hex = r.toRadixString(16).padLeft(2, '0');
    return "'\\x$hex'";
  }
  if (r < 0 || r > 0x10FFFF || (r >= 0xD800 && r <= 0xDFFF)) {
    return "'\uFFFD'";
  }
  return "'${String.fromCharCode(r)}'";
}

String _radix(GoInt v, int radix, bool unsigned) {
  if (unsigned) return $uradix(v, radix);
  if (v < GoInt.zero) return '-${$uradix(-v, radix)}';
  return $uradix(v, radix);
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
      if (precision < 0) return single ? $f32string(v) : $fstring(v);
      return _gPrecision(v, precision);
    case 'G':
      if (precision < 0) return single ? $f32string(v) : $fstring(v);
      return _gPrecision(v, precision).toUpperCase();
    case 'f':
    case 'F':
      if (v.isNaN) return 'NaN';
      if (v.isInfinite) return v > 0 ? '+Inf' : '-Inf';
      return v.toStringAsFixed(precision < 0 ? 6 : precision);
    case 'e':
      return _exponent(v, precision < 0 ? 6 : precision);
    case 'E':
      return _exponent(v, precision < 0 ? 6 : precision).toUpperCase();
    case 'd':
      return '%!d(float64=${$fstring(v)})';
  }
  return '%!$letter(float64=${$fstring(v)})';
}

/// The text of a float in the form %g of Go, with a count of the significant
/// digits.
///
/// Go rounds the value to that count, it drops each trailing zero, and it then
/// takes the form %e when the decimal exponent is under -4 or not under the
/// count. This form holds the same rule, and it writes the text from the
/// rounded digits, thus no second rounding of Dart reaches the output.
String _gPrecision(double v, int prec) {
  if (v.isNaN) return 'NaN';
  if (v.isInfinite) return v > 0 ? '+Inf' : '-Inf';
  if (prec <= 0) prec = 1;
  final String sign = v.isNegative ? '-' : '';
  final String text = v.abs().toStringAsExponential(prec - 1);
  final int at = text.indexOf('e');
  final int exp10 = int.parse(text.substring(at + 1));
  String mant = text.substring(0, at).replaceAll('.', '');
  int nd = mant.length;
  while (nd > 1 && mant.codeUnitAt(nd - 1) == 0x30) {
    nd--;
  }
  mant = mant.substring(0, nd);
  final int dp = exp10 + 1;
  int eprec = prec;
  if (eprec > nd && nd >= dp) eprec = nd;
  final int exp = dp - 1;
  if (exp < -4 || exp >= eprec) {
    final int digits = (prec > nd ? nd : prec) - 1;
    return sign + _digitsExp(mant, exp, digits);
  }
  int frac = (prec > dp ? nd : prec) - dp;
  if (frac < 0) frac = 0;
  return sign + _digitsFixed(mant, dp, frac);
}

/// The form d.dddde±dd of a digit string with a decimal exponent.
String _digitsExp(String mant, int exp, int digits) {
  final StringBuffer out = StringBuffer();
  out.write(mant[0]);
  if (digits > 0) {
    out.write('.');
    for (var i = 1; i <= digits; i++) {
      out.write(i < mant.length ? mant[i] : '0');
    }
  }
  final String esign = exp < 0 ? '-' : '+';
  final int abs = exp < 0 ? -exp : exp;
  final String number = abs < 10 ? '0$abs' : '$abs';
  out.write('e$esign$number');
  return out.toString();
}

/// The form ddd.ddd of a digit string, a decimal point, and a count of the
/// digits after the point.
String _digitsFixed(String mant, int dp, int frac) {
  final StringBuffer out = StringBuffer();
  if (dp <= 0) {
    out.write('0');
  } else {
    for (var i = 0; i < dp; i++) {
      out.write(i < mant.length ? mant[i] : '0');
    }
  }
  if (frac > 0) {
    out.write('.');
    for (var i = 0; i < frac; i++) {
      final int at = dp + i;
      out.write(at >= 0 && at < mant.length ? mant[at] : '0');
    }
  }
  return out.toString();
}

/// The text of a complex number, in the form (1+2i) of Go. The imaginary part
/// always carries a sign.
String _complex(GoComplex c, String kind, String letter, String verb) {
  switch (letter) {
    case 'v':
    case 'g':
    case 'f':
    case 'e':
      final bool single = kind == 'complex64';
      final int precision = _precision(verb);
      final String re = _float(c.re, letter, precision, single);
      String im = _float(c.im, letter, precision, single);
      if (!im.startsWith('-') && !im.startsWith('+')) im = '+$im';
      return '($re${im}i)';
  }
  return '%!$letter($kind=${_complex(c, kind, 'v', 'v')})';
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

String _hexInt(GoInt v, bool unsigned, bool upper) {
  final String text = _radix(v, 16, unsigned);
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
    final (int r, int size) = $decodeRuneRaw(v, at);
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
      case 0x07:
        out.write(r'\a');
      case 0x08:
        out.write(r'\b');
      case 0x0C:
        out.write(r'\f');
      case 0x0A:
        out.write(r'\n');
      case 0x0D:
        out.write(r'\r');
      case 0x09:
        out.write(r'\t');
      case 0x0B:
        out.write(r'\v');
      default:
        if (r < 0x20 || r == 0x7F) {
          out.write('\\x${r.toRadixString(16).padLeft(2, '0')}');
          break;
        }
        out.writeCharCode(r);
    }
  }
  out.write('"');
  return out.toString();
}

String _slice(Object? v, TypeDesc type, String verb, int depth) {
  if (v is! Slice<Object?>) return '$v';
  final TypeDesc? elem = type.elem?.call();
  final List<Object?> items = <Object?>[];
  for (var i = 0; i < v.len; i++) {
    items.add(v.array[v.off + i]);
  }
  if (_goSyntax(verb)) {
    final String name = _goName(type, depth);
    if (v.isNil) return '$name(nil)';
    return _goList(items, elem, name, verb, depth);
  }
  return _list(items, elem, verb, depth);
}

String _array(List<Object?> v, TypeDesc type, String verb, int depth) {
  final TypeDesc? elem = type.elem?.call();
  if (_goSyntax(verb)) return _goList(v, elem, type.display, verb, depth);
  return _list(v, elem, verb, depth);
}

/// _list writes each element of a slice or of an array. Go reads a list of
/// bytes as text for the verbs s, q, x, and X.
String _list(List<Object?> items, TypeDesc? elem, String verb, int depth) {
  final String letter = verb.substring(verb.length - 1);
  if (elem != null && elem.kind == 'uint8' && _textVerb(letter)) {
    final List<int> bytes = <int>[];
    for (final Object? item in items) {
      bytes.add($ix(item as GoInt) & 0xff);
    }
    return _string(GoString.fromBytes(bytes), verb, letter);
  }
  final List<String> parts = <String>[];
  for (final item in items) {
    parts.add(elem == null ? '$item' : _value(item, elem, verb, depth + 1));
  }
  return '[${parts.join(' ')}]';
}

bool _textVerb(String letter) =>
    letter == 's' || letter == 'q' || letter == 'x' || letter == 'X';

String _map(Object? v, TypeDesc type, String verb, int depth) {
  if (v is! GoMap<Object?, Object?>) return '$v';
  final bool sharp = _goSyntax(verb);
  if (sharp && v.isNil) return '${type.display}(nil)';
  final TypeDesc? keyType = type.key?.call();
  final TypeDesc? elem = type.elem?.call();
  final List<(Object?, String, String)> rows = <(Object?, String, String)>[];
  // The walk reads each entry itself. A second read by the key would lose a
  // key that no lookup finds, and a float that is not a number is such a key.
  final GoMapIter<Object?, Object?> it = GoMapIter<Object?, Object?>(v);
  while (it.next()) {
    final Object? k = it.key;
    final Object? value = it.value;
    final String keyText = keyType == null
        ? '$k'
        : _value(k, keyType, verb, depth + 1);
    final String valueText = elem == null
        ? '$value'
        : _value(value, elem, verb, depth + 1);
    rows.add((k, keyText, valueText));
  }
  // Go orders the entries by the key, and not by the text of the key. A key of
  // 1000 thus stands after a key of 3. Read internal/fmtsort of Go.
  rows.sort((a, b) => _compareKeys(a.$1, b.$1, keyType, a.$2, b.$2));
  final List<String> parts = <String>[
    for (final row in rows) '${row.$2}:${row.$3}',
  ];
  if (sharp) return '${type.display}{${parts.join(', ')}}';
  return 'map[${parts.join(' ')}]';
}

/// The order of 2 map keys of 1 type. A kind that holds no natural order takes
/// the order of its text, which keeps the output stable.
int _compareKeys(
  Object? a,
  Object? b,
  TypeDesc? type,
  String aText,
  String bText,
) {
  switch (type?.kind) {
    case 'bool':
      if (a is bool && b is bool) {
        if (a == b) return 0;
        return a ? 1 : -1;
      }
    case 'int' || 'int8' || 'int16' || 'int32' || 'int64':
      if (a is GoInt && b is GoInt) {
        if (a == b) return 0;
        return a < b ? -1 : 1;
      }
    case 'uint' || 'uint8' || 'uint16' || 'uint32' || 'uint64' || 'uintptr':
      if (a is GoInt && b is GoInt) {
        if (a == b) return 0;
        // A value above the largest signed number reads as a negative number,
        // thus the 2 signs decide first.
        final bool an = a < $gi(0);
        final bool bn = b < $gi(0);
        if (an != bn) return an ? 1 : -1;
        return a < b ? -1 : 1;
      }
    case 'float32' || 'float64':
      if (a is double && b is double) {
        // Go puts each value that is not a number first.
        if (a.isNaN) return b.isNaN ? 0 : -1;
        if (b.isNaN) return 1;
        return a.compareTo(b);
      }
    case 'string':
      if (a is GoString && b is GoString) {
        return $compare(a, b);
      }
  }
  return aText.compareTo(bText);
}

String _struct(Object? v, TypeDesc type, String verb, bool plus, int depth) {
  final bool sharp = _goSyntax(verb);
  final List<String> parts = <String>[];
  for (final field in type.fields) {
    final Object? value = field.get(v);
    final String text = _value(value, field.type(), verb, depth + 1);
    parts.add(plus || sharp ? '${field.name}:$text' : text);
  }
  if (sharp) return '${type.display}{${parts.join(', ')}}';
  return '{${parts.join(' ')}}';
}

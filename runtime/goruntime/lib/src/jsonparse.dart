/// The JSON parse of the decoder.
///
/// `jsonDecode` of dart:convert gives a Dart num for each number, thus it
/// loses the text of the literal. 2 failures follow. A uint64 above the
/// largest int64 comes back as a double and then clamps, and
/// `Decoder.UseNumber` cannot give the literal that Go gives.
///
/// The parser below keeps each number as a [JsonNum], which holds the text.
/// The store of json.dart reads that text against the target type.
library;

/// JsonNum holds the text of 1 number literal of the input.
final class JsonNum {
  final String text;

  const JsonNum(this.text);

  double get asDouble => double.parse(text);

  /// asNum gives the Dart number of the literal. A whole literal that a Dart
  /// int holds keeps each digit, and each other literal takes the double.
  num get asNum => int.tryParse(text) ?? double.parse(text);

  @override
  String toString() => text;
}

/// JsonParseError names a position of the input that holds no valid JSON.
final class JsonParseError extends FormatException {
  JsonParseError(String message, int offset) : super(message, null, offset);
}

/// jsonParse reads 1 JSON value and each space after it.
Object? jsonParse(String source) {
  final _Parser p = _Parser(source);
  p.space();
  final Object? value = p.value();
  p.space();
  if (p.i != source.length) {
    throw JsonParseError('unexpected character', p.i);
  }
  return value;
}

const int _quote = 0x22;
const int _backslash = 0x5c;

final class _Parser {
  final String s;
  int i = 0;

  _Parser(this.s);

  Never fail(String what) => throw JsonParseError(what, i);

  void space() {
    while (i < s.length) {
      final int c = s.codeUnitAt(i);
      if (c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d) {
        i++;
        continue;
      }
      return;
    }
  }

  Object? value() {
    if (i >= s.length) fail('unexpected end of input');
    final int c = s.codeUnitAt(i);
    switch (c) {
      case 0x7b: // {
        return object();
      case 0x5b: // [
        return array();
      case _quote:
        return text();
      case 0x74: // t
        word('true');
        return true;
      case 0x66: // f
        word('false');
        return false;
      case 0x6e: // n
        word('null');
        return null;
    }
    return number();
  }

  void word(String want) {
    if (!s.startsWith(want, i)) fail('unexpected character');
    i += want.length;
  }

  Map<String, Object?> object() {
    i++; // {
    final Map<String, Object?> out = <String, Object?>{};
    space();
    if (i < s.length && s.codeUnitAt(i) == 0x7d) {
      i++;
      return out;
    }
    while (true) {
      space();
      if (i >= s.length || s.codeUnitAt(i) != _quote) fail('expected a key');
      final String key = text();
      space();
      if (i >= s.length || s.codeUnitAt(i) != 0x3a) fail('expected a colon');
      i++;
      space();
      out[key] = value();
      space();
      if (i >= s.length) fail('unexpected end of input');
      final int c = s.codeUnitAt(i);
      if (c == 0x2c) {
        i++;
        continue;
      }
      if (c == 0x7d) {
        i++;
        return out;
      }
      fail('expected a comma or a closing brace');
    }
  }

  List<Object?> array() {
    i++; // [
    final List<Object?> out = <Object?>[];
    space();
    if (i < s.length && s.codeUnitAt(i) == 0x5d) {
      i++;
      return out;
    }
    while (true) {
      space();
      out.add(value());
      space();
      if (i >= s.length) fail('unexpected end of input');
      final int c = s.codeUnitAt(i);
      if (c == 0x2c) {
        i++;
        continue;
      }
      if (c == 0x5d) {
        i++;
        return out;
      }
      fail('expected a comma or a closing bracket');
    }
  }

  String text() {
    i++; // the opening quote
    final int start = i;
    // The fast path holds no escape, thus the substring is the answer.
    while (i < s.length) {
      final int c = s.codeUnitAt(i);
      if (c == _quote) {
        final String out = s.substring(start, i);
        i++;
        return out;
      }
      if (c == _backslash) return escaped(start);
      if (c < 0x20) fail('a control character stands in a string');
      i++;
    }
    fail('unexpected end of input');
  }

  String escaped(int start) {
    final StringBuffer b = StringBuffer(s.substring(start, i));
    while (i < s.length) {
      final int c = s.codeUnitAt(i);
      if (c == _quote) {
        i++;
        return b.toString();
      }
      if (c != _backslash) {
        if (c < 0x20) fail('a control character stands in a string');
        b.writeCharCode(c);
        i++;
        continue;
      }
      i++;
      if (i >= s.length) fail('unexpected end of input');
      final int e = s.codeUnitAt(i);
      i++;
      switch (e) {
        case _quote:
          b.writeCharCode(_quote);
        case _backslash:
          b.writeCharCode(_backslash);
        case 0x2f: // /
          b.writeCharCode(0x2f);
        case 0x62: // b
          b.writeCharCode(0x08);
        case 0x66: // f
          b.writeCharCode(0x0c);
        case 0x6e: // n
          b.writeCharCode(0x0a);
        case 0x72: // r
          b.writeCharCode(0x0d);
        case 0x74: // t
          b.writeCharCode(0x09);
        case 0x75: // u
          b.writeCharCode(hex4());
        default:
          fail('an unknown escape stands in a string');
      }
    }
    fail('unexpected end of input');
  }

  int hex4() {
    if (i + 4 > s.length) fail('a short escape stands in a string');
    int out = 0;
    for (int k = 0; k < 4; k++) {
      final int c = s.codeUnitAt(i + k);
      int d;
      if (c >= 0x30 && c <= 0x39) {
        d = c - 0x30;
      } else if (c >= 0x61 && c <= 0x66) {
        d = c - 0x61 + 10;
      } else if (c >= 0x41 && c <= 0x46) {
        d = c - 0x41 + 10;
      } else {
        fail('a bad escape stands in a string');
      }
      out = out * 16 + d;
    }
    i += 4;
    return out;
  }

  JsonNum number() {
    final int start = i;
    if (i < s.length && s.codeUnitAt(i) == 0x2d) i++; // -
    digits();
    if (i < s.length && s.codeUnitAt(i) == 0x2e) {
      i++;
      digits();
    }
    if (i < s.length) {
      final int c = s.codeUnitAt(i);
      if (c == 0x65 || c == 0x45) {
        i++;
        if (i < s.length) {
          final int sign = s.codeUnitAt(i);
          if (sign == 0x2b || sign == 0x2d) i++;
        }
        digits();
      }
    }
    if (i == start) fail('unexpected character');
    return JsonNum(s.substring(start, i));
  }

  void digits() {
    final int start = i;
    while (i < s.length) {
      final int c = s.codeUnitAt(i);
      if (c < 0x30 || c > 0x39) break;
      i++;
    }
    if (i == start) fail('expected a digit');
  }
}

/// The scanner and the fast decoder of encoding/json.
///
/// The Go code of the package holds a recursive descent decoder, and the stack
/// of Dart holds about 1,400 levels of it. Go holds 10,000 levels. The scanner
/// below walks the bytes with a stack of its own, thus it holds each depth up
/// to the limit of Go, and it reports each syntax error at the offset that Go
/// reports.
///
/// The decoder builds the Go value of a target of a fast type. It writes the
/// target only after the whole document proved, thus a document that the build
/// cannot serve leaves the target as it stands and the Go code runs from the
/// start.
library;

import 'dart:typed_data';

import 'box.dart';
import 'goint.dart';
import 'iface.dart';
import 'map.dart';
import 'reflect.dart';
import 'slice.dart';
import 'string.dart';

/// The answers of [$jsonUnmarshal]. The Go patch of encoding/json holds the
/// same numbers.
///
/// A syntax error other than the depth limit gives [$jsonGoPath]. The Go code
/// then reads the document again and it builds the error itself, thus each
/// text and each offset stays the text and the offset of Go. The depth limit
/// needs the answer of the scanner, because the descent of the Go code
/// overflows the stack of Dart far below the limit of 10,000 levels.
const int $jsonDone = 0;
const int $jsonGoPath = 1;
const int $jsonMethod = 2;
const int $jsonMaxDepth = 3;

/// The limit of encoding/json/jsontext.
const int _maxDepth = 10000;

/// The mode of a slot: the value that a slot takes goes nowhere, into an
/// empty interface, or into a value of a known type.
const int _skip = 0;
const int _any = 1;
const int _typed = 2;

/// Reports whether the bytes hold 1 valid JSON value.
bool $jsonValid(Slice<GoInt> data) {
  final _Scanner s = _Scanner(_bytesOf(data), _Build.skipAll());
  s.run();
  return s.code == 0;
}

/// Decodes 1 JSON document into the target, when the target holds a fast type.
///
/// The results are the code and 2 numbers. [$jsonMaxDepth] holds the offset of
/// the bracket that broke the limit in the first number. [$jsonMethod] holds
/// the bounds of the value, thus the Go patch calls UnmarshalJSON with those
/// bytes. Each other answer holds 2 zeros.
(int, int, int) $jsonUnmarshal(Slice<GoInt> data, Iface? target) {
  if (target == null || target.desc.kind != 'ptr' || target.value == null) {
    return ($jsonGoPath, 0, 0);
  }
  final TypeDesc? elem = target.desc.elem?.call();
  if (elem == null) return ($jsonGoPath, 0, 0);
  final Uint8List b = _bytesOf(data);
  final _Build build = _Build.forTarget(elem, target.value);
  final _Scanner s = _Scanner(b, build);
  s.run();
  if (s.code != 0) return (s.code, s.at, 0);
  if (build.method) return ($jsonMethod, s.valueStart, s.valueEnd);
  if (build.dead) return ($jsonGoPath, 0, 0);
  build.write(elem, target.value, s.result);
  return ($jsonDone, 0, 0);
}

/// The bytes of a byte slice. The JS target holds a pair for each Go integer,
/// thus that target takes a copy.
Uint8List _bytesOf(Slice<GoInt> data) {
  // A Slice of the JS target holds no subtype relation to $ByteSlice, thus the
  // test reads the object itself.
  final Object slice = data;
  if (slice is $ByteSlice) {
    return Uint8List.sublistView(slice.bytes, slice.off, slice.off + slice.len);
  }
  final Uint8List out = Uint8List(data.len);
  for (int i = 0; i < data.len; i++) {
    out[i] = $ix(data.array[data.off + i]);
  }
  return out;
}

/// _GiveUp ends a build that cannot prove the decode. The scan goes on, thus a
/// syntax error after the point still reaches the caller.
final class _GiveUp implements Exception {
  const _GiveUp();
}

/// 1 open object or array of the document.
final class _Frame {
  final bool object;
  final int mode;

  /// The mode and the type of the next value inside. A struct changes the 2
  /// at each name, and each other frame holds the 2 for its life.
  int childMode;
  TypeDesc? childType;

  /// The values of an array, in both build modes. The typed build moves them
  /// into a slice at the close, because Go grows the slice in the same steps.
  List<Object?>? items;

  /// The map of an object of the any mode.
  GoMap<GoString, Iface?>? anyMap;
  GoString? anyKey;

  /// The typed build: the type of the container, the fresh value, the field
  /// of the current name, the fields seen so far, and the count of the
  /// elements of an array.
  TypeDesc? desc;
  Object? value;
  FieldDesc? field;
  Uint8List? seen;
  int count = 0;

  _Frame(this.object, this.mode, this.childMode, this.childType);
}

/// The scanner. It follows the grammar walk of encoding/json/jsontext, thus
/// the position of each error is the position that Go reports.
final class _Scanner {
  final Uint8List b;
  final int n;
  final _Build build;
  int p = 0;

  int code = 0;
  int at = 0;

  /// True when the last read of a value opened a container.
  bool opened = false;

  /// The bounds of the top level value, without the white space around it.
  int valueStart = 0;
  int valueEnd = 0;

  /// The value of the document, after a build.
  Object? result;

  final List<_Frame> stack = <_Frame>[];

  _Scanner(this.b, this.build) : n = b.length;

  /// Ends the scan with an answer other than done. The Go code builds the
  /// error of each answer but the depth limit.
  void _fail(int c, [int position = 0]) {
    code = c;
    at = position;
  }

  /// Steps over the white space of JSON.
  void _space() {
    while (p < n) {
      final int c = b[p];
      if (c != 0x20 && c != 0x09 && c != 0x0a && c != 0x0d) return;
      p++;
    }
  }

  /// Reads the whole document. A wrong document leaves the code set.
  ///
  /// The loop holds the open containers in [stack], thus a document of 3
  /// million levels costs no stack of Dart.
  void run() {
    _space();
    valueStart = p;
    bool wantName = false;
    while (true) {
      if (wantName && !_name()) return;
      opened = false;
      if (!_value()) return;
      if (opened) {
        wantName = stack.last.object;
        continue;
      }
      // A leaf ended, or a container closed. The delimiter of each open
      // container above it follows.
      while (true) {
        if (stack.isEmpty) {
          valueEnd = p;
          _space();
          if (p < n) _fail($jsonGoPath);
          return;
        }
        _space();
        if (p >= n) {
          _fail($jsonGoPath);
          return;
        }
        final _Frame top = stack.last;
        final int c = b[p];
        if (c == 0x2c) {
          p++;
          wantName = top.object;
          break;
        }
        if (top.object ? c == 0x7d : c == 0x5d) {
          p++;
          stack.removeLast();
          _close(top);
          if (code != 0) return;
          continue;
        }
        _fail($jsonGoPath);
        return;
      }
    }
  }

  /// Reads the name of a member and its colon.
  bool _name() {
    _space();
    if (p >= n || b[p] != 0x22) {
      _fail($jsonGoPath);
      return false;
    }
    final _Frame top = stack.last;
    final int start = p;
    final bool plain = _string();
    if (code != 0) return false;
    // The text reads the position of the scanner, thus it stands here and not
    // after the colon.
    final Uint8List? name = top.mode == _skip ? null : _text(start, plain);
    _space();
    if (p >= n || b[p] != 0x3a) {
      _fail($jsonGoPath);
      return false;
    }
    p++;
    if (name != null) _key(top, name);
    return true;
  }

  /// Reads 1 value at the position.
  ///
  /// A container that holds a member or an element joins the stack, and the
  /// flag [opened] then reports it to [run].
  bool _value() {
    _space();
    if (p >= n) {
      _fail($jsonGoPath);
      return false;
    }
    final _Frame? top = stack.isEmpty ? null : stack.last;
    final int mode = top == null ? build.rootMode : top.childMode;
    final int c = b[p];
    switch (c) {
      case 0x7b || 0x5b: // { and [
        final bool object = c == 0x7b;
        if (stack.length == _maxDepth) {
          _fail($jsonMaxDepth, p);
          return false;
        }
        p++;
        _space();
        if (p >= n) {
          _fail($jsonGoPath);
          return false;
        }
        final _Frame f = _open(top, mode, object);
        if (b[p] == (object ? 0x7d : 0x5d)) {
          p++;
          _close(f);
          return code == 0;
        }
        stack.add(f);
        opened = true;
        return true;
      case 0x22: // "
        final int start = p;
        final bool plain = _string();
        if (code != 0) return false;
        if (mode != _skip) _leafString(top, mode, start, plain);
        return code == 0;
      case 0x74: // t
        if (!_literal('true')) return false;
        if (mode != _skip) _leafBool(top, mode, true);
        return code == 0;
      case 0x66: // f
        if (!_literal('false')) return false;
        if (mode != _skip) _leafBool(top, mode, false);
        return code == 0;
      case 0x6e: // n
        if (!_literal('null')) return false;
        if (mode != _skip) _leafNull(top, mode);
        return code == 0;
    }
    if (c == 0x2d || (c >= 0x30 && c <= 0x39)) {
      final int start = p;
      if (!_number()) return false;
      if (mode != _skip) _leafNumber(top, mode, start, p);
      return code == 0;
    }
    _fail($jsonGoPath);
    return false;
  }

  /// Reads 1 of the 3 literals. A wrong byte names the literal, and a short
  /// input ends at the end of the input.
  bool _literal(String lit) {
    for (int i = 0; i < lit.length; i++) {
      if (p >= n) {
        _fail($jsonGoPath);
        return false;
      }
      if (b[p] != lit.codeUnitAt(i)) {
        _fail($jsonGoPath);
        return false;
      }
      p++;
    }
    return true;
  }

  /// Reads 1 number, with the grammar of jsonwire.ConsumeNumber.
  bool _number() {
    if (b[p] == 0x2d) p++;
    if (p >= n) {
      _fail($jsonGoPath);
      return false;
    }
    int c = b[p];
    if (c == 0x30) {
      p++;
    } else if (c >= 0x31 && c <= 0x39) {
      p++;
      while (p < n && b[p] >= 0x30 && b[p] <= 0x39) {
        p++;
      }
    } else {
      _fail($jsonGoPath);
      return false;
    }
    if (p < n && b[p] == 0x2e) {
      p++;
      if (p >= n) {
        _fail($jsonGoPath);
        return false;
      }
      c = b[p];
      if (c < 0x30 || c > 0x39) {
        _fail($jsonGoPath);
        return false;
      }
      while (p < n && b[p] >= 0x30 && b[p] <= 0x39) {
        p++;
      }
    }
    if (p < n && (b[p] == 0x65 || b[p] == 0x45)) {
      p++;
      if (p < n && (b[p] == 0x2b || b[p] == 0x2d)) p++;
      if (p >= n) {
        _fail($jsonGoPath);
        return false;
      }
      c = b[p];
      if (c < 0x30 || c > 0x39) {
        _fail($jsonGoPath);
        return false;
      }
      while (p < n && b[p] >= 0x30 && b[p] <= 0x39) {
        p++;
      }
    }
    return true;
  }

  /// Reads 1 string from its opening quote. The result is true when the bytes
  /// between the quotes are the text itself: no escape and no invalid UTF-8.
  bool _string() {
    p++;
    bool plain = true;
    while (true) {
      while (p < n) {
        final int c = b[p];
        if (c < 0x20 || c == 0x22 || c == 0x5c || c >= 0x80) break;
        p++;
      }
      if (p >= n) {
        _fail($jsonGoPath);
        return false;
      }
      final int c = b[p];
      if (c == 0x22) {
        p++;
        return plain;
      }
      if (c == 0x5c) {
        plain = false;
        if (n < p + 2) {
          _fail($jsonGoPath);
          return false;
        }
        final int e = b[p + 1];
        switch (e) {
          case 0x2f || 0x22 || 0x5c || 0x62 || 0x66 || 0x6e || 0x72 || 0x74:
            p += 2;
          case 0x75: // u
            if (n < p + 6) {
              if (_escapePrefix(p, false)) {
                _fail($jsonGoPath);
              } else {
                _fail($jsonGoPath);
              }
              return false;
            }
            if (_hex4(p + 2) < 0) {
              _fail($jsonGoPath);
              return false;
            }
            p += 6;
          default:
            _fail($jsonGoPath);
            return false;
        }
        continue;
      }
      if (c >= 0x80) {
        final int size = _runeSize(p);
        if (size > 1) {
          p += size;
          continue;
        }
        plain = false;
        if (!_fullRune(p)) {
          _fail($jsonGoPath);
          return false;
        }
        p++;
        continue;
      }
      _fail($jsonGoPath);
      return false;
    }
  }

  /// Reports whether the bytes from the position can start a \uXXXX escape.
  /// The Go code then waits for more input, and a complete buffer ends.
  bool _escapePrefix(int start, bool low) {
    for (int i = 0; start + i < n; i++) {
      final int c = b[start + i];
      if (i == 0 && c != 0x5c) return false;
      if (i == 1 && c != 0x75) return false;
      if (i == 2 && low && c != 0x64 && c != 0x44) return false;
      if (i == 3 && low && !_lowHalfDigit(c)) return false;
      if (i >= 2 && i < 6 && _hexDigit(c) < 0) return false;
    }
    return true;
  }

  static bool _lowHalfDigit(int c) =>
      (c >= 0x63 && c <= 0x66) || (c >= 0x43 && c <= 0x46);

  static int _hexDigit(int c) {
    if (c >= 0x30 && c <= 0x39) return c - 0x30;
    if (c >= 0x61 && c <= 0x66) return c - 0x61 + 10;
    if (c >= 0x41 && c <= 0x46) return c - 0x41 + 10;
    return -1;
  }

  /// The value of 4 hex digits at the position, or -1.
  int _hex4(int start) {
    int v = 0;
    for (int i = 0; i < 4; i++) {
      final int d = _hexDigit(b[start + i]);
      if (d < 0) return -1;
      v = v * 16 + d;
    }
    return v;
  }

  /// The size of the UTF-8 sequence at the position, and 1 for an invalid
  /// sequence. The rule is the rule of utf8.DecodeRune.
  int _runeSize(int i) {
    final int b0 = b[i];
    int size;
    int lo = 0x80;
    int hi = 0xbf;
    if (b0 >= 0xc2 && b0 <= 0xdf) {
      size = 2;
    } else if (b0 >= 0xe0 && b0 <= 0xef) {
      size = 3;
      if (b0 == 0xe0) lo = 0xa0;
      if (b0 == 0xed) hi = 0x9f;
    } else if (b0 >= 0xf0 && b0 <= 0xf4) {
      size = 4;
      if (b0 == 0xf0) lo = 0x90;
      if (b0 == 0xf4) hi = 0x8f;
    } else {
      return 1;
    }
    if (i + size > n) return 1;
    final int b1 = b[i + 1];
    if (b1 < lo || b1 > hi) return 1;
    for (int k = 2; k < size; k++) {
      final int bk = b[i + k];
      if (bk < 0x80 || bk > 0xbf) return 1;
    }
    return size;
  }

  /// The rule of utf8.FullRune: false only for a short prefix of a valid
  /// sequence at the end of the input.
  bool _fullRune(int i) {
    final int b0 = b[i];
    int size;
    int lo = 0x80;
    int hi = 0xbf;
    if (b0 >= 0xc2 && b0 <= 0xdf) {
      size = 2;
    } else if (b0 >= 0xe0 && b0 <= 0xef) {
      size = 3;
      if (b0 == 0xe0) lo = 0xa0;
      if (b0 == 0xed) hi = 0x9f;
    } else if (b0 >= 0xf0 && b0 <= 0xf4) {
      size = 4;
      if (b0 == 0xf0) lo = 0x90;
      if (b0 == 0xf4) hi = 0x8f;
    } else {
      return true;
    }
    final int have = n - i;
    if (have >= size) return true;
    if (have > 1 && (b[i + 1] < lo || b[i + 1] > hi)) return true;
    if (have > 2 && (b[i + 2] < 0x80 || b[i + 2] > 0xbf)) return true;
    return false;
  }

  // The build steps. Each 1 catches the give up of the typed build and turns
  // the rest of the scan into a check of the syntax alone.

  _Frame _open(_Frame? top, int mode, bool object) {
    try {
      return build.open(top, mode, object);
    } on _GiveUp {
      build.dead = true;
      return _Frame(object, _skip, _skip, null);
    }
  }

  void _close(_Frame f) {
    if (f.mode == _skip) {
      if (stack.isEmpty && !build.dead) result = null;
      return;
    }
    try {
      final Object? v = build.close(f);
      _put(stack.isEmpty ? null : stack.last, v);
    } on _GiveUp {
      build.dead = true;
    }
  }

  void _key(_Frame top, Uint8List name) {
    try {
      build.key(top, name);
    } on _GiveUp {
      build.dead = true;
    }
  }

  void _leafString(_Frame? top, int mode, int start, bool plain) {
    try {
      _put(top, build.string(top, mode, _text(start, plain)));
    } on _GiveUp {
      build.dead = true;
    }
  }

  void _leafBool(_Frame? top, int mode, bool v) {
    try {
      _put(top, build.boolean(top, mode, v));
    } on _GiveUp {
      build.dead = true;
    }
  }

  void _leafNull(_Frame? top, int mode) {
    try {
      _put(top, build.none(top, mode));
    } on _GiveUp {
      build.dead = true;
    }
  }

  void _leafNumber(_Frame? top, int mode, int start, int end) {
    try {
      _put(top, build.number(top, mode, b, start, end));
    } on _GiveUp {
      build.dead = true;
    }
  }

  void _put(_Frame? top, Object? v) {
    if (top == null) {
      result = v;
      return;
    }
    build.add(top, v);
  }

  /// The bytes of the string that starts at the quote and ends before the
  /// position. The rule of the escapes is the rule of jsonwire.AppendUnquote.
  Uint8List _text(int start, bool plain) {
    if (plain) {
      final int len = p - start - 2;
      final Uint8List out = Uint8List(len);
      out.setRange(0, len, b, start + 1);
      return out;
    }
    final BytesBuilder out = BytesBuilder();
    int i = start + 1;
    final int end = p - 1;
    int run = i;
    while (i < end) {
      final int c = b[i];
      if (c < 0x80 && c != 0x5c) {
        i++;
        continue;
      }
      if (run < i) out.add(Uint8List.sublistView(b, run, i));
      if (c == 0x5c) {
        final int e = b[i + 1];
        switch (e) {
          case 0x22 || 0x5c || 0x2f:
            out.addByte(e);
            i += 2;
          case 0x62:
            out.addByte(0x08);
            i += 2;
          case 0x66:
            out.addByte(0x0c);
            i += 2;
          case 0x6e:
            out.addByte(0x0a);
            i += 2;
          case 0x72:
            out.addByte(0x0d);
            i += 2;
          case 0x74:
            out.addByte(0x09);
            i += 2;
          default:
            int r = _hex4(i + 2);
            i += 6;
            if (r >= 0xd800 && r <= 0xdfff) {
              // A surrogate half needs the second half. A lone half gives
              // U+FFFD, and the next escape then stands on its own.
              int pair = -1;
              if (i + 6 <= end && b[i] == 0x5c && b[i + 1] == 0x75) {
                final int r2 = _hex4(i + 2);
                if (r2 >= 0xdc00 && r2 <= 0xdfff && r <= 0xdbff) {
                  pair = 0x10000 + ((r - 0xd800) << 10) + (r2 - 0xdc00);
                }
              }
              if (pair < 0) {
                r = 0xfffd;
              } else {
                r = pair;
                i += 6;
              }
            }
            _appendRune(out, r);
        }
        run = i;
        continue;
      }
      final int size = _runeSize(i);
      if (size > 1) {
        out.add(Uint8List.sublistView(b, i, i + size));
        i += size;
      } else {
        out.addByte(0xef);
        out.addByte(0xbf);
        out.addByte(0xbd);
        i++;
      }
      run = i;
    }
    if (run < end) out.add(Uint8List.sublistView(b, run, end));
    return out.takeBytes();
  }

  static void _appendRune(BytesBuilder out, int r) {
    if (r < 0x80) {
      out.addByte(r);
    } else if (r < 0x800) {
      out.addByte(0xc0 | (r >> 6));
      out.addByte(0x80 | (r & 0x3f));
    } else if (r < 0x10000) {
      out.addByte(0xe0 | (r >> 12));
      out.addByte(0x80 | ((r >> 6) & 0x3f));
      out.addByte(0x80 | (r & 0x3f));
    } else {
      out.addByte(0xf0 | (r >> 18));
      out.addByte(0x80 | ((r >> 12) & 0x3f));
      out.addByte(0x80 | ((r >> 6) & 0x3f));
      out.addByte(0x80 | (r & 0x3f));
    }
  }
}

/// The build of the Go value. The any mode builds the values of an empty
/// interface, and the typed mode builds a value of the type of the target.
final class _Build {
  final int rootMode;
  final TypeDesc? rootType;

  /// True when the target holds UnmarshalJSON. The Go patch calls the method
  /// with the bytes of the value.
  final bool method;

  /// True when the build writes no target. A give up sets it, and so does a
  /// target that only the Go code serves.
  bool dead;

  _Build._(this.rootMode, this.rootType, this.method, this.dead);

  /// A build that reads the syntax alone. The scan still gives each error.
  factory _Build.skipAll() => _Build._(_skip, null, false, true);

  /// The build for a target of the type.
  factory _Build.forTarget(TypeDesc elem, Object? pointer) {
    if (_isAny(elem)) {
      final Iface? held = (pointer as $Ptr<Object?>).v as Iface?;
      // Go merges into a pointer that the interface holds.
      if (held != null && held.desc.kind == 'ptr' && held.value != null) {
        return _Build.skipAll();
      }
      return _Build._(_any, null, false, false);
    }
    if (_hasUnmarshalMethod(elem)) {
      if (_callsUnmarshalJSON(elem)) {
        return _Build._(_skip, null, true, false);
      }
      return _Build.skipAll();
    }
    if (!_typedKind(elem) || !_isZero(_readTarget(elem, pointer), elem)) {
      return _Build.skipAll();
    }
    return _Build._(_typed, elem, false, false);
  }

  /// Writes the built value into the target.
  void write(TypeDesc elem, Object? pointer, Object? value) {
    if (pointer is $Ptr<Object?>) {
      pointer.v = value;
      return;
    }
    $assignValue(pointer, value, elem);
  }

  // The steps of the scan.

  _Frame open(_Frame? top, int mode, bool object) {
    if (mode == _skip) return _Frame(object, _skip, _skip, null);
    if (mode == _any) {
      final _Frame f = _Frame(object, _any, _any, null);
      if (object) {
        f.anyMap = GoMap<GoString, Iface?>(_nilIface);
      } else {
        f.items = <Iface?>[];
      }
      return f;
    }
    final TypeDesc t = top == null ? rootType! : top.childType!;
    if (_isAny(t)) return open(top, _any, object);
    if (object) {
      switch (t.kind) {
        case 'struct':
          final _Frame f = _Frame(true, _typed, _skip, null);
          f.desc = t;
          f.value = t.zero!();
          f.seen = Uint8List(t.fields.length);
          return f;
        case 'map':
          final TypeDesc key = t.key!();
          if (key.kind != 'string' || _hasUnmarshalMethod(key)) {
            throw const _GiveUp();
          }
          final TypeDesc elem = t.elem!();
          _checkTyped(elem);
          final _Frame f = _Frame(true, _typed, _modeFor(elem), elem);
          f.desc = t;
          f.value = t.make!(0);
          return f;
      }
      throw const _GiveUp();
    }
    switch (t.kind) {
      case 'slice' || 'array':
        final TypeDesc elem = t.elem!();
        if (elem.kind == 'uint8') throw const _GiveUp();
        _checkTyped(elem);
        final _Frame f = _Frame(false, _typed, _modeFor(elem), elem);
        f.desc = t;
        f.items = <Object?>[];
        return f;
    }
    throw const _GiveUp();
  }

  Object? close(_Frame f) {
    if (f.mode == _any) {
      if (f.object) return Iface(_anyMapDesc, f.anyMap);
      return Iface(_anySliceDesc, _anySlice(f.items!));
    }
    final TypeDesc t = f.desc!;
    switch (t.kind) {
      case 'struct' || 'map':
        return f.value;
      case 'slice':
        final List<Object?> items = f.items!;
        if (items.isEmpty) return t.make!(0);
        final int cap = _capFor(items.length);
        final Slice<Object?> out = t.make!(cap) as Slice<Object?>;
        final List<Object?> array = out.array;
        for (int i = 0; i < items.length; i++) {
          array[i] = items[i];
        }
        return out.range(0, items.length, cap);
      case 'array':
        final List<Object?> out = t.zero!() as List<Object?>;
        final List<Object?> items = f.items!;
        final int count = items.length < out.length ? items.length : out.length;
        for (int i = 0; i < count; i++) {
          out[i] = items[i];
        }
        return out;
    }
    throw const _GiveUp();
  }

  void key(_Frame f, Uint8List name) {
    if (f.mode == _any) {
      f.anyKey = GoString.take(name);
      return;
    }
    final TypeDesc t = f.desc!;
    if (t.kind == 'map') {
      f.anyKey = GoString.take(name);
      return;
    }
    final _StructInfo info = _structInfo(t);
    final int index = info.lookup(name);
    if (index < 0) {
      f.field = null;
      f.childMode = _skip;
      f.childType = null;
      return;
    }
    // A second member of 1 field merges into the first value in Go.
    if (f.seen![index] != 0) throw const _GiveUp();
    f.seen![index] = 1;
    final FieldDesc field = t.fields[index];
    final TypeDesc ft = field.type();
    f.field = field;
    f.childMode = _modeFor(ft);
    f.childType = ft;
  }

  void add(_Frame f, Object? v) {
    if (f.mode == _any) {
      if (f.object) {
        f.anyMap!.set(f.anyKey!, v as Iface?);
      } else {
        f.items!.add(v);
      }
      return;
    }
    if (f.object) {
      if (f.desc!.kind == 'map') {
        (f.value as GoMap<Object?, Object?>).set(f.anyKey, v);
        return;
      }
      f.field!.set(f.value, v);
      return;
    }
    f.items!.add(v);
  }

  Object? string(_Frame? top, int mode, Uint8List text) {
    if (mode == _any) return Iface($typeString, GoString.take(text));
    final TypeDesc t = _slotType(top);
    if (_isAny(t)) return Iface($typeString, GoString.take(text));
    if (t.kind == 'string') return GoString.take(text);
    throw const _GiveUp();
  }

  Object? boolean(_Frame? top, int mode, bool v) {
    if (mode == _any) return Iface($typeBool, v);
    final TypeDesc t = _slotType(top);
    if (_isAny(t)) return Iface($typeBool, v);
    if (t.kind == 'bool') return v;
    throw const _GiveUp();
  }

  Object? none(_Frame? top, int mode) {
    if (mode == _any) return null;
    final TypeDesc t = _slotType(top);
    return t.zero!();
  }

  Object? number(_Frame? top, int mode, Uint8List b, int start, int end) {
    if (mode == _any) return _anyNumber(b, start, end);
    final TypeDesc t = _slotType(top);
    if (_isAny(t)) return _anyNumber(b, start, end);
    switch (t.kind) {
      case 'float64':
        return _parseDouble(b, start, end);
      case 'int' || 'int64':
        return _parseInt(b, start, end, 64, true);
      case 'int32':
        return _parseInt(b, start, end, 32, true);
      case 'int16':
        return _parseInt(b, start, end, 16, true);
      case 'int8':
        return _parseInt(b, start, end, 8, true);
      case 'uint' || 'uint64' || 'uintptr':
        return _parseInt(b, start, end, 64, false);
      case 'uint32':
        return _parseInt(b, start, end, 32, false);
      case 'uint16':
        return _parseInt(b, start, end, 16, false);
      case 'uint8':
        return _parseInt(b, start, end, 8, false);
    }
    throw const _GiveUp();
  }

  Iface _anyNumber(Uint8List b, int start, int end) =>
      Iface($typeFloat64, _parseDouble(b, start, end));

  TypeDesc _slotType(_Frame? top) => top == null ? rootType! : top.childType!;

  /// The mode of a value of the type inside the typed build.
  int _modeFor(TypeDesc t) => _isAny(t) ? _any : _typed;

  /// _checkTyped rejects an element type that the typed build cannot serve.
  void _checkTyped(TypeDesc t) {
    if (_isAny(t)) return;
    if (_hasUnmarshalMethod(t) || !_typedKind(t)) throw const _GiveUp();
  }
}

Iface? _nilIface() => null;

/// The slice of []any that the build gives. The zero value of a slice type
/// shares 1 empty list, thus each slice here takes a list of its own.
Slice<Iface?> _anySlice(List<Object?> items) {
  if (items.isEmpty) return Slice<Iface?>(<Iface?>[], 0, 0, 0);
  final int cap = _capFor(items.length);
  final List<Iface?> array = List<Iface?>.filled(cap, null);
  for (int i = 0; i < items.length; i++) {
    array[i] = items[i] as Iface?;
  }
  return Slice<Iface?>(array, 0, items.length, cap);
}

/// The capacity that reflect.Value.Grow of the overlay gives a slice that the
/// decoder grows 1 element at a time.
int _capFor(int len) {
  int cap = 1;
  while (cap < len) {
    cap = cap * 2 + 1;
  }
  return cap;
}

/// True for an interface type with no method. Go decodes into such a type
/// exactly as into `any`.
bool _isAny(TypeDesc t) => t.kind == 'interface' && t.methods.isEmpty;

/// The names of the types that encoding/json/v2 decodes with a rule of its
/// own, in front of the default of the kind.
const Set<String> _specialNames = <String>{
  'time.Time',
  'time.Duration',
  'encoding/json/jsontext.Value',
  'encoding/json.Number',
};

/// True when the type or the pointer to it holds a method that the decoder of
/// Go calls in place of the default of the kind.
bool _hasUnmarshalMethod(TypeDesc t) {
  if (_specialNames.contains(t.name)) return true;
  return _tableHasUnmarshal(t.methods) ||
      _tableHasUnmarshal(t.ptr?.call().methods);
}

bool _tableHasUnmarshal(Map<String, Function>? table) {
  if (table == null) return false;
  for (final String key in table.keys) {
    if (key.startsWith('UnmarshalJSON(') ||
        key.startsWith('UnmarshalJSONFrom(') ||
        key.startsWith('UnmarshalText(')) {
      return true;
    }
  }
  return false;
}

/// True when the Go code calls UnmarshalJSON of the pointer to the type with
/// the bytes of the value, and nothing else.
bool _callsUnmarshalJSON(TypeDesc t) {
  if (_specialNames.contains(t.name)) return false;
  if (t.kind == 'ptr' || t.kind == 'interface') return false;
  const String key = 'UnmarshalJSON([]uint8)error';
  final bool has =
      t.methods.containsKey(key) ||
      (t.ptr?.call().methods.containsKey(key) ?? false);
  if (!has) return false;
  return !_tableHasPrefix(t.methods, 'UnmarshalJSONFrom(') &&
      !_tableHasPrefix(t.ptr?.call().methods, 'UnmarshalJSONFrom(');
}

bool _tableHasPrefix(Map<String, Function>? table, String prefix) {
  if (table == null) return false;
  for (final String key in table.keys) {
    if (key.startsWith(prefix)) return true;
  }
  return false;
}

/// True for a kind that the typed build serves.
bool _typedKind(TypeDesc t) {
  switch (t.kind) {
    case 'bool' ||
        'string' ||
        'int' ||
        'int8' ||
        'int16' ||
        'int32' ||
        'int64' ||
        'uint' ||
        'uint8' ||
        'uint16' ||
        'uint32' ||
        'uint64' ||
        'uintptr' ||
        'float64' ||
        'struct' ||
        'slice' ||
        'array' ||
        'map' ||
        'ptr':
      return t.zero != null;
  }
  return false;
}

/// The value that the target pointer holds now.
Object? _readTarget(TypeDesc elem, Object? pointer) {
  if (pointer is $Ptr<Object?>) return pointer.v;
  return pointer;
}

/// True when the value is the zero value of its type. The build replaces the
/// whole target, thus the target must hold nothing that Go would keep.
bool _isZero(Object? v, TypeDesc t) {
  switch (t.kind) {
    case 'bool':
      return v == false;
    case 'string':
      return (v as GoString).length == 0;
    case 'float64':
      return v is double && v == 0.0 && !v.isNegative;
    case 'ptr' || 'interface':
      return v == null;
    case 'slice':
      return (v as Slice<Object?>).isNil;
    case 'map':
      return (v as GoMap<Object?, Object?>).isNil;
    case 'struct':
      if (v == null) return false;
      for (final FieldDesc f in t.fields) {
        if (!_isZero(f.get(v), f.type())) return false;
      }
      return true;
    case 'array':
      final List<Object?> a = v as List<Object?>;
      final TypeDesc elem = t.elem!();
      for (final Object? x in a) {
        if (!_isZero(x, elem)) return false;
      }
      return true;
  }
  return v is int && v == 0;
}

/// The fields of a struct type that the typed build reads, by their JSON
/// name. A struct that holds a rule outside the build gives up.
final class _StructInfo {
  final List<int> indexes;
  final List<String> names;

  _StructInfo(this.indexes, this.names);

  /// The index of the field of a name. An equal name wins, and a name that
  /// differs only in the case of ASCII letters comes second. A name with a
  /// byte above 0x7f can fold in a way that the ASCII rule does not hold.
  int lookup(Uint8List name) {
    for (int i = 0; i < names.length; i++) {
      if (_sameBytes(names[i], name)) return indexes[i];
    }
    for (int k = 0; k < name.length; k++) {
      if (name[k] >= 0x80) throw const _GiveUp();
    }
    for (int i = 0; i < names.length; i++) {
      if (_foldBytes(names[i], name)) return indexes[i];
    }
    return -1;
  }

  static bool _sameBytes(String a, Uint8List b) {
    if (a.length != b.length) return false;
    for (int i = 0; i < b.length; i++) {
      if (a.codeUnitAt(i) != b[i]) return false;
    }
    return true;
  }

  static bool _foldBytes(String a, Uint8List b) {
    if (a.length != b.length) return false;
    for (int i = 0; i < b.length; i++) {
      int x = a.codeUnitAt(i);
      int y = b[i];
      if (x >= 0x61 && x <= 0x7a) x -= 0x20;
      if (y >= 0x61 && y <= 0x7a) y -= 0x20;
      if (x != y) return false;
    }
    return true;
  }
}

/// The cache reads the descriptor itself and not its name. 2 local types of 1
/// package hold 1 name, thus a cache of the name gives the fields of the wrong
/// type.
final Map<TypeDesc, _StructInfo> _structInfos = <TypeDesc, _StructInfo>{};

_StructInfo _structInfo(TypeDesc t) {
  final _StructInfo? held = _structInfos[t];
  if (held != null) return held;
  final List<int> indexes = <int>[];
  final List<String> names = <String>[];
  for (int i = 0; i < t.fields.length; i++) {
    final FieldDesc f = t.fields[i];
    if (f.embedded) throw const _GiveUp();
    if (!f.exported) continue;
    final String? name = _jsonName(f);
    if (name == null) continue;
    if (names.contains(name)) throw const _GiveUp();
    final TypeDesc ft = f.type();
    if (!_isAny(ft) && (_hasUnmarshalMethod(ft) || !_typedKind(ft))) {
      throw const _GiveUp();
    }
    indexes.add(i);
    names.add(name);
  }
  final _StructInfo out = _StructInfo(indexes, names);
  _structInfos[t] = out;
  return out;
}

/// The JSON name of a field, or null for a field that the tag drops. A tag
/// with an option other than omitempty and omitzero gives up, and so does a
/// name with a quote, a backslash, or a byte above 0x7f.
String? _jsonName(FieldDesc f) {
  final String? tag = _tagValue(f.tag, 'json');
  if (tag == null) return f.name;
  if (tag == '-') return null;
  final int comma = tag.indexOf(',');
  final String name = comma < 0 ? tag : tag.substring(0, comma);
  for (int i = 0; i < name.length; i++) {
    final int c = name.codeUnitAt(i);
    if (c >= 0x80 || c == 0x22 || c == 0x27 || c == 0x5c || c == 0x60) {
      throw const _GiveUp();
    }
  }
  if (comma >= 0) {
    for (final String opt in tag.substring(comma + 1).split(',')) {
      if (opt != 'omitempty' && opt != 'omitzero') throw const _GiveUp();
    }
  }
  return name.isEmpty ? f.name : name;
}

/// The value of 1 key of a Go struct tag, or null when the tag holds no such
/// key. A value with an escape gives up.
String? _tagValue(String tag, String want) {
  int i = 0;
  while (i < tag.length) {
    while (i < tag.length && tag.codeUnitAt(i) == 0x20) {
      i++;
    }
    if (i >= tag.length) return null;
    final int colon = tag.indexOf(':', i);
    if (colon < 0) throw const _GiveUp();
    final String key = tag.substring(i, colon);
    if (colon + 1 >= tag.length || tag.codeUnitAt(colon + 1) != 0x22) {
      throw const _GiveUp();
    }
    int end = colon + 2;
    while (end < tag.length && tag.codeUnitAt(end) != 0x22) {
      if (tag.codeUnitAt(end) == 0x5c) throw const _GiveUp();
      end++;
    }
    if (end >= tag.length) throw const _GiveUp();
    if (key == want) return tag.substring(colon + 2, end);
    i = end + 1;
  }
  return null;
}

/// The exact powers of 10 of a double.
final List<double> _pow10 = List<double>.generate(23, (int i) {
  double v = 1.0;
  for (int k = 0; k < i; k++) {
    v *= 10.0;
  }
  return v;
});

/// The double of a number literal, with the rounding of strconv.ParseFloat.
///
/// A literal of at most 15 digits and no exponent is the quotient of 2 exact
/// doubles, and 1 division rounds it once. Each other literal takes the parse
/// of Dart, which rounds correctly for each length. A value out of range gives
/// up, because Go reports it.
double _parseDouble(Uint8List b, int start, int end) {
  int i = start;
  final bool neg = b[i] == 0x2d;
  if (neg) i++;
  int mantissa = 0;
  int digits = 0;
  int fraction = 0;
  bool simple = true;
  while (i < end) {
    final int c = b[i];
    if (c >= 0x30 && c <= 0x39) {
      mantissa = mantissa * 10 + (c - 0x30);
      digits++;
      i++;
      continue;
    }
    if (c == 0x2e) {
      i++;
      while (i < end && b[i] >= 0x30 && b[i] <= 0x39) {
        mantissa = mantissa * 10 + (b[i] - 0x30);
        digits++;
        fraction++;
        i++;
      }
      if (i < end) simple = false;
      break;
    }
    simple = false;
    break;
  }
  if (simple && digits <= 15) {
    double v = mantissa.toDouble();
    if (fraction > 0) v = v / _pow10[fraction];
    return neg ? -v : v;
  }
  final double v = double.parse(String.fromCharCodes(b, start, end));
  if (v.isInfinite) throw const _GiveUp();
  return v;
}

/// The Go integer of a number literal, with the rules of the int and the uint
/// decoders of encoding/json/v2: no fraction, no exponent, no leading zero,
/// and a value inside the range of the type.
GoInt _parseInt(Uint8List b, int start, int end, int bits, bool signed) {
  int i = start;
  final bool neg = b[i] == 0x2d;
  if (neg) {
    if (!signed) throw const _GiveUp();
    i++;
  }
  final int count = end - i;
  if (count == 0 || count > 20) throw const _GiveUp();
  if (b[i] == 0x30 && count > 1) throw const _GiveUp();
  if ($isJS && count > 15) throw const _GiveUp();
  int v = 0;
  for (int k = i; k < end; k++) {
    final int c = b[k];
    if (c < 0x30 || c > 0x39) throw const _GiveUp();
    v = v * 10 + (c - 0x30);
  }
  // 20 digits can pass the largest uint64. The product wraps, thus the text
  // decides.
  if (count == 20 && _compareDigits(b, i, end, '18446744073709551615') > 0) {
    throw const _GiveUp();
  }
  final bool wide = count == 20;
  if (signed) {
    // The magnitude fits 63 bits, or it is exactly 2^63 with a minus sign.
    if (wide || v < 0) {
      if (!(neg && wide == false && v == -v)) throw const _GiveUp();
    }
    if (bits < 64) {
      final int limit = 1 << (bits - 1);
      if (neg ? v > limit : v >= limit) throw const _GiveUp();
    }
    return $gi(neg ? -v : v);
  }
  if (bits < 64) {
    if (wide || v < 0 || v >= (1 << bits)) throw const _GiveUp();
  }
  return $gi(v);
}

int _compareDigits(Uint8List b, int start, int end, String limit) {
  for (int i = 0; i < limit.length; i++) {
    final int d = b[start + i] - limit.codeUnitAt(i);
    if (d != 0) return d;
  }
  return 0;
}

// The descriptions of the 3 types of a generic value. Each 1 holds the text
// of the emitted description, thus a value of the build and a value of the
// emitted code share 1 name and 1 shape.

const TypeDesc _anyDesc = TypeDesc(
  'interface{}',
  'interface',
  <String, Function>{},
  text: 'interface {}',
  zero: _nilIface,
);

TypeDesc _anyDescOf() => _anyDesc;

final TypeDesc _anySliceDesc = TypeDesc(
  '[]interface{}',
  'slice',
  const <String, Function>{},
  text: '[]interface {}',
  elem: _anyDescOf,
  make: (int n) =>
      $makeSlice<Iface?>(n, n, (int k) => List<Iface?>.filled(k, null)),
  withType: <R>(R Function<T>() f) => f<Slice<Iface?>>(),
  comparable: false,
  zero: Slice<Iface?>.nil,
);

final TypeDesc _anyMapDesc = TypeDesc(
  'map[string]interface{}',
  'map',
  const <String, Function>{},
  text: 'map[string]interface {}',
  key: () => $typeString,
  elem: _anyDescOf,
  make: (int n) => GoMap<GoString, Iface?>(_nilIface),
  withType: <R>(R Function<T>() f) => f<GoMap<GoString, Iface?>>(),
  comparable: false,
  zero: () => GoMap<GoString, Iface?>.nil(_nilIface),
);

// The fast marshal of encoding/json.
//
// The Go code of the package walks the value with the arshalers of
// encoding/json/v2, and each field of a struct reaches reflect.Value.Field. A
// measure of 2026-09-09 gave 43 percent of the cost of a marshal to that 1
// call. The walk below reads the descriptor of the type and the getter of each
// field, thus it needs no Value and no handle of the reflect bridge.
//
// The walk holds each rule of the v1 options: the escape of the 5 characters
// that Go escapes beyond the JSON minimum, the shortest float text that reads
// back, the sorted keys of a map, the base64 form of a byte slice, and the
// array form of a byte array. A value that the walk cannot prove ends the
// build, and the Go code then marshals it from the start.

/// The depth that the fast marshal serves. A deeper value, and each value that
/// holds a cycle, goes back to the Go code. The Go code of this build reports
/// a cycle from the depth of 100, thus it needs no stack of its own for one.
const int _maxMarshalDepth = 500;

/// Marshals 1 Go value with the rules of encoding/json v1.
///
/// A nil result reports that the fast path cannot prove the value. The Go code
/// then marshals it from the start, thus each byte and each error keeps the
/// answer of Go.
Slice<GoInt> $jsonMarshal(Iface? value) {
  final _Enc e = _Enc();
  try {
    if (value == null) {
      e.text('null');
    } else {
      e.value(value.desc, value.value, 0);
    }
  } on _GiveUp {
    return Slice<GoInt>.nil();
  }
  return $gils(Slice<int>.of(Uint8List.sublistView(e.buf, 0, e.n)));
}

/// The buffer and the walk of the fast marshal.
final class _Enc {
  Uint8List buf = Uint8List(1024);
  int n = 0;

  void _room(int need) {
    if (n + need <= buf.length) return;
    int cap = buf.length * 2;
    while (cap < n + need) {
      cap *= 2;
    }
    final Uint8List out = Uint8List(cap);
    out.setRange(0, n, buf);
    buf = out;
  }

  void byte(int c) {
    if (n == buf.length) _room(1);
    buf[n++] = c;
  }

  void raw(Uint8List b, int start, int end) {
    final int len = end - start;
    if (len <= 0) return;
    _room(len);
    buf.setRange(n, n + len, b, start);
    n += len;
  }

  /// Writes the code units of a text of ASCII.
  void text(String s) {
    _room(s.length);
    for (int i = 0; i < s.length; i++) {
      buf[n++] = s.codeUnitAt(i);
    }
  }

  /// Writes 1 value of the type.
  void value(TypeDesc t, Object? v, int depth) {
    if (depth > _maxMarshalDepth) throw const _GiveUp();
    if (_hasMarshalMethod(t)) throw const _GiveUp();
    switch (t.kind) {
      case 'bool':
        text((v as bool) ? 'true' : 'false');
        return;
      case 'string':
        quote((v as GoString).bytes);
        return;
      case 'int' || 'int8' || 'int16' || 'int32' || 'int64':
        text((v as GoInt).toString());
        return;
      case 'uint' || 'uint8' || 'uint16' || 'uint32' || 'uint64' || 'uintptr':
        text($ustring(v as GoInt));
        return;
      case 'float64':
        number(v as double);
        return;
      case 'interface':
        final Iface? held = v as Iface?;
        if (held == null) {
          text('null');
          return;
        }
        value(held.desc, held.value, depth + 1);
        return;
      case 'ptr':
        if (v == null) {
          text('null');
          return;
        }
        final TypeDesc elem = t.elem!();
        value(elem, v is $Ptr<Object?> ? v.v : v, depth + 1);
        return;
      case 'slice':
        slice(t, v as Slice<Object?>, depth);
        return;
      case 'array':
        array(t, v as List<Object?>, depth);
        return;
      case 'map':
        object(t, v as GoMap<Object?, Object?>, depth);
        return;
      case 'struct':
        record(t, v, depth);
        return;
    }
    throw const _GiveUp();
  }

  /// Writes a slice. A slice of bytes takes the base64 form of Go, and a nil
  /// slice takes the null of the v1 options.
  void slice(TypeDesc t, Slice<Object?> s, int depth) {
    final TypeDesc elem = t.elem!();
    if (elem.kind == 'uint8' && !_hasMarshalMethod(elem)) {
      if (s.isNil) {
        text('null');
        return;
      }
      base64(s);
      return;
    }
    if (s.isNil) {
      text('null');
      return;
    }
    byte(0x5b);
    for (int i = 0; i < s.len; i++) {
      if (i > 0) byte(0x2c);
      value(elem, s.array[s.off + i], depth + 1);
    }
    byte(0x5d);
  }

  /// Writes an array. The v1 options hold FormatByteArrayAsArray, thus an
  /// array of bytes takes the form of an array of numbers.
  void array(TypeDesc t, List<Object?> a, int depth) {
    final TypeDesc elem = t.elem!();
    byte(0x5b);
    for (int i = 0; i < a.length; i++) {
      if (i > 0) byte(0x2c);
      value(elem, a[i], depth + 1);
    }
    byte(0x5d);
  }

  /// Writes a map. Only a key of the string kind stands, and the keys sort by
  /// their bytes, exactly as the deterministic order of the v1 options asks.
  void object(TypeDesc t, GoMap<Object?, Object?> m, int depth) {
    final TypeDesc key = t.key!();
    if (key.kind != 'string' || _hasMarshalMethod(key)) throw const _GiveUp();
    if (m.isNil) {
      text('null');
      return;
    }
    final TypeDesc elem = t.elem!();
    final Slice<Object?> keys = m.keys();
    final List<GoString> names = <GoString>[];
    for (int i = 0; i < keys.len; i++) {
      names.add(keys.array[keys.off + i]! as GoString);
    }
    if (names.length > 1) names.sort(_compareBytes);
    byte(0x7b);
    for (int i = 0; i < names.length; i++) {
      if (i > 0) byte(0x2c);
      quote(names[i].bytes);
      byte(0x3a);
      value(elem, m.get(names[i]), depth + 1);
    }
    byte(0x7d);
  }

  /// Writes a struct.
  void record(TypeDesc t, Object? v, int depth) {
    final _EncInfo info = _encInfo(t);
    byte(0x7b);
    bool first = true;
    for (int i = 0; i < info.indexes.length; i++) {
      final FieldDesc f = t.fields[info.indexes[i]];
      final Object? held = f.get(v);
      final TypeDesc ft = f.type();
      if (info.omitEmpty[i] && _legacyEmpty(held, ft)) continue;
      if (!first) byte(0x2c);
      first = false;
      byte(0x22);
      text(info.names[i]);
      byte(0x22);
      byte(0x3a);
      value(ft, held, depth + 1);
    }
    byte(0x7d);
  }

  /// Writes a JSON string, with the escapes of the v1 options: the 2 of the
  /// grammar, each control character, the 3 of HTML, and the 2 of JavaScript.
  /// A byte that no valid sequence of UTF-8 holds gives U+FFFD.
  void quote(Uint8List src) {
    byte(0x22);
    int i = 0;
    int run = 0;
    final int len = src.length;
    while (i < len) {
      final int c = src[i];
      if (c < 0x80) {
        if (_escapeASCII[c] == 0) {
          i++;
          continue;
        }
        raw(src, run, i);
        i++;
        run = i;
        _escaped(c);
        continue;
      }
      final int size = _sizeOf(src, i, len);
      if (size == 0) {
        raw(src, run, i);
        i++;
        run = i;
        byte(0xef);
        byte(0xbf);
        byte(0xbd);
        continue;
      }
      if (size == 3 &&
          src[i] == 0xe2 &&
          src[i + 1] == 0x80 &&
          src[i + 2] >= 0xa8 &&
          src[i + 2] <= 0xa9) {
        raw(src, run, i);
        i += 3;
        run = i;
        text(src[i - 1] == 0xa8 ? r'\u2028' : r'\u2029');
        continue;
      }
      i += size;
    }
    raw(src, run, len);
    byte(0x22);
  }

  void _escaped(int c) {
    switch (c) {
      case 0x22 || 0x5c:
        byte(0x5c);
        byte(c);
      case 0x08:
        text(r'\b');
      case 0x0c:
        text(r'\f');
      case 0x0a:
        text(r'\n');
      case 0x0d:
        text(r'\r');
      case 0x09:
        text(r'\t');
      default:
        text(r'\u00');
        byte(_hexDigits.codeUnitAt((c >> 4) & 0xf));
        byte(_hexDigits.codeUnitAt(c & 0xf));
    }
  }

  /// Writes the base64 form of a byte slice, with the alphabet and the padding
  /// of RFC 4648.
  void base64(Slice<Object?> s) {
    final Uint8List b = _sliceBytes(s);
    byte(0x22);
    int i = 0;
    while (i + 3 <= b.length) {
      final int x = (b[i] << 16) | (b[i + 1] << 8) | b[i + 2];
      byte(_b64.codeUnitAt((x >> 18) & 0x3f));
      byte(_b64.codeUnitAt((x >> 12) & 0x3f));
      byte(_b64.codeUnitAt((x >> 6) & 0x3f));
      byte(_b64.codeUnitAt(x & 0x3f));
      i += 3;
    }
    final int rest = b.length - i;
    if (rest == 1) {
      final int x = b[i] << 16;
      byte(_b64.codeUnitAt((x >> 18) & 0x3f));
      byte(_b64.codeUnitAt((x >> 12) & 0x3f));
      byte(0x3d);
      byte(0x3d);
    } else if (rest == 2) {
      final int x = (b[i] << 16) | (b[i + 1] << 8);
      byte(_b64.codeUnitAt((x >> 18) & 0x3f));
      byte(_b64.codeUnitAt((x >> 12) & 0x3f));
      byte(_b64.codeUnitAt((x >> 6) & 0x3f));
      byte(0x3d);
    }
    byte(0x22);
  }

  /// Writes a JSON number of a float64, with the rule of jsonwire.AppendFloat.
  ///
  /// Go takes the exponent form below 1e-6 and from 1e21, and the fixed form
  /// between the 2. The text holds the shortest digits that read back as the
  /// same value.
  void number(double v) {
    if (v.isNaN || v.isInfinite) throw const _GiveUp();
    if (v == 0) {
      if (v.isNegative) byte(0x2d);
      byte(0x30);
      return;
    }
    final double abs = v.abs();
    final String form = abs.toStringAsExponential();
    final int at = form.indexOf('e');
    String digits = form.substring(0, at).replaceAll('.', '');
    final int exp = int.parse(form.substring(at + 1));
    while (digits.length > 1 && digits.endsWith('0')) {
      digits = digits.substring(0, digits.length - 1);
    }
    if (v.isNegative) byte(0x2d);
    if (abs < 1e-6 || abs >= 1e21) {
      byte(digits.codeUnitAt(0));
      if (digits.length > 1) {
        byte(0x2e);
        text(digits.substring(1));
      }
      byte(0x65);
      if (exp < 0) {
        byte(0x2d);
        text(exp.abs().toString());
      } else {
        byte(0x2b);
        text(exp.toString().padLeft(2, '0'));
      }
      return;
    }
    if (exp < 0) {
      text('0.');
      for (int i = 0; i < -exp - 1; i++) {
        byte(0x30);
      }
      text(digits);
      return;
    }
    if (digits.length > exp + 1) {
      text(digits.substring(0, exp + 1));
      byte(0x2e);
      text(digits.substring(exp + 1));
      return;
    }
    text(digits);
    for (int i = 0; i < exp + 1 - digits.length; i++) {
      byte(0x30);
    }
  }
}

const String _hexDigits = '0123456789abcdef';
const String _b64 =
    'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';

/// The bytes of a byte slice of the walk.
Uint8List _sliceBytes(Slice<Object?> s) {
  final Object slice = s;
  if (slice is $ByteSlice) {
    return Uint8List.sublistView(slice.bytes, slice.off, slice.off + slice.len);
  }
  final Uint8List out = Uint8List(s.len);
  for (int i = 0; i < s.len; i++) {
    out[i] = $ix(s.array[s.off + i]! as GoInt);
  }
  return out;
}

/// The order of 2 Go strings. Go compares the bytes.
int _compareBytes(GoString a, GoString b) {
  final Uint8List x = a.bytes;
  final Uint8List y = b.bytes;
  final int n = x.length < y.length ? x.length : y.length;
  for (int i = 0; i < n; i++) {
    if (x[i] != y[i]) return x[i] - y[i];
  }
  return x.length - y.length;
}

/// The size of the UTF-8 sequence at the position, and 0 for a byte that no
/// valid sequence holds. The rule is the rule of utf8.DecodeRune.
int _sizeOf(Uint8List b, int i, int n) {
  final int b0 = b[i];
  int size;
  int lo = 0x80;
  int hi = 0xbf;
  if (b0 >= 0xc2 && b0 <= 0xdf) {
    size = 2;
  } else if (b0 >= 0xe0 && b0 <= 0xef) {
    size = 3;
    if (b0 == 0xe0) lo = 0xa0;
    if (b0 == 0xed) hi = 0x9f;
  } else if (b0 >= 0xf0 && b0 <= 0xf4) {
    size = 4;
    if (b0 == 0xf0) lo = 0x90;
    if (b0 == 0xf4) hi = 0x8f;
  } else {
    return 0;
  }
  if (i + size > n) return 0;
  final int b1 = b[i + 1];
  if (b1 < lo || b1 > hi) return 0;
  for (int k = 2; k < size; k++) {
    final int bk = b[i + k];
    if (bk < 0x80 || bk > 0xbf) return 0;
  }
  return size;
}

/// The table of escapeASCII of encoding/json/internal/jsonwire. The v1 options
/// hold EscapeForHTML, thus the 3 characters of HTML also stand.
final Uint8List _escapeASCII = Uint8List.fromList(<int>[
  1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, //
  1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, //
  0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, //
  0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, //
  0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //
  0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, //
  0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //
  0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //
]);

/// True when the type or the pointer to it holds a method that the encoder of
/// Go calls in place of the default of the kind.
bool _hasMarshalMethod(TypeDesc t) {
  if (_specialNames.contains(t.name)) return true;
  return _tableHasMarshal(t.methods) || _tableHasMarshal(t.ptr?.call().methods);
}

bool _tableHasMarshal(Map<String, Function>? table) {
  if (table == null) return false;
  for (final String key in table.keys) {
    if (key.startsWith('MarshalJSON(') ||
        key.startsWith('MarshalJSONTo(') ||
        key.startsWith('MarshalText(') ||
        key.startsWith('AppendText(')) {
      return true;
    }
  }
  return false;
}

/// True for the empty value of the v1 rule of omitempty. Go reads the Go value
/// and not the JSON form: isLegacyEmpty of encoding/json/v2 holds the list.
bool _legacyEmpty(Object? v, TypeDesc t) {
  switch (t.kind) {
    case 'bool':
      return v == false;
    case 'string':
      return (v as GoString).bytes.isEmpty;
    case 'float64' || 'float32':
      return v == 0.0;
    case 'ptr' || 'interface':
      return v == null;
    case 'slice':
      return (v as Slice<Object?>).len == 0;
    case 'array':
      return (v as List<Object?>).isEmpty;
    case 'map':
      return (v as GoMap<Object?, Object?>).length == 0;
    case 'int' ||
        'int8' ||
        'int16' ||
        'int32' ||
        'int64' ||
        'uint' ||
        'uint8' ||
        'uint16' ||
        'uint32' ||
        'uint64' ||
        'uintptr':
      return (v as GoInt) == $gi(0);
  }
  return false;
}

/// The fields of a struct type that the fast marshal writes, in the order of
/// the declaration. A struct that holds a rule outside the walk gives up.
final class _EncInfo {
  final List<int> indexes;
  final List<String> names;
  final List<bool> omitEmpty;

  _EncInfo(this.indexes, this.names, this.omitEmpty);
}

/// The cache reads the descriptor itself and not its name. Read the comment of
/// _structInfos.
final Map<TypeDesc, _EncInfo> _encInfos = <TypeDesc, _EncInfo>{};

_EncInfo _encInfo(TypeDesc t) {
  final _EncInfo? held = _encInfos[t];
  if (held != null) return held;
  final List<int> indexes = <int>[];
  final List<String> names = <String>[];
  final List<bool> omit = <bool>[];
  for (int i = 0; i < t.fields.length; i++) {
    final FieldDesc f = t.fields[i];
    if (f.embedded) throw const _GiveUp();
    if (!f.exported) continue;
    final String? tag = _tagValue(f.tag, 'json');
    String name = f.name;
    bool omitEmpty = false;
    if (tag != null) {
      if (tag == '-') continue;
      final int comma = tag.indexOf(',');
      final String head = comma < 0 ? tag : tag.substring(0, comma);
      for (int k = 0; k < head.length; k++) {
        final int c = head.codeUnitAt(k);
        if (c >= 0x80 || c == 0x22 || c == 0x27 || c == 0x5c || c == 0x60) {
          throw const _GiveUp();
        }
      }
      if (head.isNotEmpty) name = head;
      if (comma >= 0) {
        for (final String opt in tag.substring(comma + 1).split(',')) {
          if (opt == 'omitempty') {
            omitEmpty = true;
            continue;
          }
          if (opt.isEmpty) continue;
          throw const _GiveUp();
        }
      }
    }
    for (int k = 0; k < name.length; k++) {
      final int c = name.codeUnitAt(k);
      if (c >= 0x80 || c == 0x22 || c == 0x5c) throw const _GiveUp();
    }
    if (names.contains(name)) throw const _GiveUp();
    indexes.add(i);
    names.add(name);
    omit.add(omitEmpty);
  }
  final _EncInfo out = _EncInfo(indexes, names, omit);
  _encInfos[t] = out;
  return out;
}

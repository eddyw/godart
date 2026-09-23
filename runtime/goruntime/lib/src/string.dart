import 'dart:convert';
import 'dart:typed_data';

import 'goint.dart';
import 'int.dart';
import 'panic.dart';
import 'slice.dart';

/// A Go string: an immutable sequence of bytes.
///
/// A Dart `String` holds UTF-16 code units, thus it cannot hold the bytes of a
/// Go string. Convert between the 2 types only at an interop boundary.
final class GoString {
  /// The bytes of the string.
  ///
  /// A Go string never changes, and the compiler writes no store into this
  /// list. The storage of a string that a conversion or a concatenation
  /// builds is an unmodifiable view, thus a write of a Dart caller throws. A
  /// substring is a plain view of the storage of its source, because a scan
  /// that gives many parts pays 1 view object for each part and not 2. No
  /// code sends a string to another isolate, thus no rule of the isolate
  /// boundary holds the form of the storage.
  final Uint8List bytes;

  GoString._(this.bytes);

  /// The decode of the bytes, after the first read.
  ///
  /// A Go string never changes, thus 1 decode holds for the life of the value.
  /// A rebuild of a Flutter tree reads the same label again at each pass, and
  /// a large document that crosses the boundary again pays 1 decode only.
  String? _dart;

  /// The zero value of a Go string.
  static final GoString empty = GoString.fromBytes(const <int>[]);

  /// Encodes a Dart string as UTF-8. A code unit of the escape range gives the
  /// 1 byte that it stands for. Read toDartString.
  ///
  /// A text of ASCII takes 1 loop over the code units, and it holds that text
  /// as its own decode. The UTF-8 encoder of Dart reached 9 percent of the
  /// cost of fmt.Sprintf.
  ///
  /// _encode gives a fresh buffer that no other code holds, thus this factory
  /// adopts the buffer and makes no second copy.
  factory GoString.fromString(String s) {
    final int n = s.length;
    final Uint8List out = Uint8List(n);
    for (var i = 0; i < n; i++) {
      final int unit = s.codeUnitAt(i);
      if (unit >= 0x80) return GoString.take(_encode(s));
      out[i] = unit;
    }
    final g = GoString._(out.asUnmodifiableView());
    g._dart = s;
    return g;
  }

  /// Copies the bytes into unmodifiable storage.
  factory GoString.fromBytes(List<int> bytes) =>
      GoString._(Uint8List.fromList(bytes).asUnmodifiableView());

  /// The Go string of a Dart string that holds ASCII only.
  ///
  /// The caller must give a text whose code units each stand under 128. The
  /// factory holds the text as the decode too, thus a read at the boundary
  /// costs nothing and the UTF-8 encoder never runs.
  factory GoString.ascii(String s) {
    final out = Uint8List(s.length);
    for (var i = 0; i < s.length; i++) {
      out[i] = s.codeUnitAt(i);
    }
    final g = GoString._(out.asUnmodifiableView());
    g._dart = s;
    return g;
  }

  /// Wraps bytes that the caller never changes again.
  factory GoString.take(Uint8List bytes) =>
      GoString._(bytes.asUnmodifiableView());

  /// The number of bytes. This value is not the number of runes.
  int get length => bytes.length;

  /// The byte at index [i], with the bounds check of Go.
  int operator [](int i) {
    if (i < 0 || i >= bytes.length) $panicIndex($gi(i), bytes.length);
    return bytes[i];
  }

  /// The byte at a Go index, as a Go integer. The emitted code reads it.
  GoInt $at(GoInt i, [bool uns = false]) {
    final int n = $ix(i);
    if (n < 0 || n >= bytes.length) $panicIndex(i, bytes.length, uns);
    return $gi(bytes[n]);
  }

  /// Decodes the bytes as UTF-8.
  ///
  /// A byte that holds no rune of UTF-8 becomes 1 code unit of the range U+DC80
  /// to U+DCFF, thus GoString.fromString gives the same bytes again. Go writes
  /// such a byte with no change, and a Dart String holds runes only.
  String toDartString() => _dart ??= _decode();

  String _decode() {
    try {
      return utf8.decode(bytes);
    } on FormatException {
      return _escape(this);
    }
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    if (other is! GoString || other.bytes.length != bytes.length) return false;
    for (var i = 0; i < bytes.length; i++) {
      if (other.bytes[i] != bytes[i]) return false;
    }
    return true;
  }

  int? _hash;

  @override
  int get hashCode => _hash ??= _mix();

  /// The FNV-1a hash of the bytes. Object.hashAll builds an iterator of the
  /// byte list, and a Go map calls this getter for each key.
  ///
  /// The mask holds the value under 26 bits, thus the product stays under 2^50
  /// and a JS number carries it with no loss.
  int _mix() {
    var h = 0x01C9DC5;
    for (var i = 0; i < bytes.length; i++) {
      h = ((h ^ bytes[i]) * 0x01000193) & 0x03FFFFFF;
    }
    return h;
  }

  @override
  String toString() => toDartString();
}

/// The text in upper case, with the rules of Unicode.
GoString $upper(GoString v) =>
    GoString.fromString(v.toDartString().toUpperCase());

/// The text in lower case, with the rules of Unicode.
GoString $lower(GoString v) =>
    GoString.fromString(v.toDartString().toLowerCase());

GoString $concat(GoString a, GoString b) {
  if (a.length == 0) return b;
  if (b.length == 0) return a;
  final out = Uint8List(a.length + b.length);
  out.setRange(0, a.length, a.bytes);
  out.setRange(a.length, out.length, b.bytes);
  return GoString.take(out);
}

/// Compares 2 strings by byte, as Go does with `<`, `<=`, `>`, and `>=`.
int $compare(GoString a, GoString b) {
  final int n = a.length < b.length ? a.length : b.length;
  for (var i = 0; i < n; i++) {
    if (a.bytes[i] != b.bytes[i]) return a.bytes[i] < b.bytes[i] ? -1 : 1;
  }
  if (a.length == b.length) return 0;
  return a.length < b.length ? -1 : 1;
}

/// The Go expression `s[low:high]`. The result shares the storage, because a
/// Go string never changes. The view is the 1 object of the result next to
/// the string itself. Read the field bytes.
GoString $substring(
  GoString s,
  GoInt lowIndex,
  GoInt highIndex, [
  bool lowUns = false,
  bool highUns = false,
]) {
  final low = $ix(lowIndex);
  final high = $ix(highIndex);
  if (high < 0 || high > s.length) {
    $panicBound(SliceBound.aLen, highIndex, s.length, highUns);
  }
  if (low < 0 || low > high) {
    $panicBound(SliceBound.b, lowIndex, high, lowUns);
  }
  if (low == 0 && high == s.length) return s;
  return GoString._(
    Uint8List.view(s.bytes.buffer, s.bytes.offsetInBytes + low, high - low),
  );
}

/// The bytes of a string, as a slice that shares the storage.
///
/// The Go builtins append and copy read the source and hold it not, thus the
/// share reaches no other code and a write to the result never stands. The
/// conversion `[]byte(s)` must copy, and it reads $stringToBytes.
///
/// The JS form builds a pair for each byte, exactly as $stringToBytes does,
/// because a GoInt is an object there.
Slice<GoInt> $stringBytes(GoString s) {
  if ($isJS) return $stringToBytes(s);
  return Slice<GoInt>(
    s.bytes as List<GoInt>,
    0,
    s.bytes.length,
    s.bytes.length,
  );
}

/// The Go expression `append(b, s...)` for a byte slice and a string.
///
/// The bytes of the string reach the slice with no header between them, thus
/// the call allocates 1 object and not 2. Read $stringBytes for the share.
Slice<GoInt> $appendString(
  Slice<GoInt> s,
  GoString str,
  List<GoInt> Function(int) alloc,
) {
  final int n = str.bytes.length;
  if (n == 0) return s;
  if ($isJS) return $appendSlice(s, $stringToBytes(str), alloc);
  // strings.Builder, fmt, and encoding/json each reach this line for each
  // write. The element type stands here, thus 1 test of the class gives a
  // direct move of the bytes and a direct build of the header.
  if (s is $ByteSlice) {
    final int need = s.len + n;
    if (need <= s.cap) {
      final Uint8List dst = s.bytes;
      dst.setRange(s.off + s.len, s.off + need, str.bytes, 0);
      return $ByteSlice(dst, s.off, need, s.cap);
    }
  }
  return $appendRange(s, str.bytes as List<GoInt>, 0, n, alloc);
}

/// The Go conversion `[]byte(s)`. Go copies the bytes. The native form keeps
/// the dense byte list, because the list type erases there.
Slice<GoInt> $stringToBytes(GoString s) {
  if ($isJS) {
    return Slice.of(<GoInt>[for (final b in s.bytes) $gi(b)]);
  }
  return Slice.of(Uint8List.fromList(s.bytes) as List<GoInt>);
}

/// The Go conversion `string(b)` for a byte slice.
GoString $bytesToString(Slice<GoInt> b) {
  final out = Uint8List(b.len);
  // A byte slice holds a Uint8List, thus 1 move of the memory replaces the
  // loop. The loop stands for the JS target, where an element is a pair.
  if (b is $ByteSlice) {
    out.setRange(0, b.len, b.bytes, b.off);
    return GoString.take(out);
  }
  for (var i = 0; i < b.len; i++) {
    out[i] = $ix(b.array[b.off + i]);
  }
  return GoString.take(out);
}

/// The Go conversion `string(r)` for 1 rune.
GoString $runeToString(GoInt goRune) {
  return _runeToString($ix(goRune));
}

GoString _runeToString(int rune) {
  rune = _validRune(rune);
  if (rune < 0x80) return GoString.take(Uint8List.fromList(<int>[rune]));
  if (rune < 0x800) {
    return GoString.take(
      Uint8List.fromList(<int>[0xC0 | (rune >> 6), 0x80 | (rune & 0x3F)]),
    );
  }
  if (rune < 0x10000) {
    return GoString.take(
      Uint8List.fromList(<int>[
        0xE0 | (rune >> 12),
        0x80 | ((rune >> 6) & 0x3F),
        0x80 | (rune & 0x3F),
      ]),
    );
  }
  return GoString.take(
    Uint8List.fromList(<int>[
      0xF0 | (rune >> 18),
      0x80 | ((rune >> 12) & 0x3F),
      0x80 | ((rune >> 6) & 0x3F),
      0x80 | (rune & 0x3F),
    ]),
  );
}

/// The Go conversion `[]rune(s)`.
Slice<GoInt> $stringToRunes(GoString s) {
  final out = <int>[];
  var i = 0;
  while (i < s.length) {
    final (int rune, int size) = $decodeRuneRaw(s, i);
    out.add(rune);
    i += size;
  }
  if ($isJS) {
    return Slice.of(<GoInt>[for (final r in out) $gi(r)]);
  }
  return Slice.of(Int32List.fromList(out) as List<GoInt>);
}

/// The Go conversion `string(r)` for a rune slice.
///
/// The first pass counts the bytes and the second pass writes them, thus the
/// cost stays linear in the count of runes. A concatenation of 1 rune at a
/// time copied the whole result for each rune.
GoString $runesToString(Slice<GoInt> runes) {
  final int n = runes.len;
  int size = 0;
  for (int i = 0; i < n; i++) {
    size += _runeSize(_validRune($ix(runes.array[runes.off + i])));
  }
  final Uint8List out = Uint8List(size);
  int j = 0;
  for (int i = 0; i < n; i++) {
    final int rune = _validRune($ix(runes.array[runes.off + i]));
    if (rune < 0x80) {
      out[j++] = rune;
    } else if (rune < 0x800) {
      out[j++] = 0xC0 | (rune >> 6);
      out[j++] = 0x80 | (rune & 0x3F);
    } else if (rune < 0x10000) {
      out[j++] = 0xE0 | (rune >> 12);
      out[j++] = 0x80 | ((rune >> 6) & 0x3F);
      out[j++] = 0x80 | (rune & 0x3F);
    } else {
      out[j++] = 0xF0 | (rune >> 18);
      out[j++] = 0x80 | ((rune >> 12) & 0x3F);
      out[j++] = 0x80 | ((rune >> 6) & 0x3F);
      out[j++] = 0x80 | (rune & 0x3F);
    }
  }
  return GoString.take(out);
}

/// The rune that Go encodes for a value: the value itself, or U+FFFD for a
/// negative value, a surrogate, and a value above U+10FFFF.
int _validRune(int rune) {
  if (rune < 0 || rune > 0x10FFFF || (rune >= 0xD800 && rune <= 0xDFFF)) {
    return 0xFFFD;
  }
  return rune;
}

int _runeSize(int rune) {
  if (rune < 0x80) return 1;
  if (rune < 0x800) return 2;
  if (rune < 0x10000) return 3;
  return 4;
}

/// Reads 1 UTF-8 rune at a byte index. An invalid sequence gives U+FFFD with
/// a size of 1, exactly as a Go range loop does. An overlong sequence, a
/// surrogate, and a value above U+10FFFF are each invalid.
(GoInt, GoInt) $decodeRune(GoString s, GoInt i) {
  final (int rune, int size) = $decodeRuneRaw(s, $ix(i));
  return ($gi(rune), $gi(size));
}

(int, int) $decodeRuneRaw(GoString s, int i) {
  final b = s.bytes;
  final int b0 = b[i];
  if (b0 < 0x80) return (b0, 1);
  int size;
  int rune;
  int least;
  if (b0 >= 0xF8) {
    return (0xFFFD, 1);
  } else if (b0 >= 0xF0) {
    size = 4;
    rune = b0 & 0x07;
    least = 0x10000;
  } else if (b0 >= 0xE0) {
    size = 3;
    rune = b0 & 0x0F;
    least = 0x800;
  } else if (b0 >= 0xC0) {
    size = 2;
    rune = b0 & 0x1F;
    least = 0x80;
  } else {
    return (0xFFFD, 1);
  }
  if (i + size > b.length) return (0xFFFD, 1);
  for (var k = 1; k < size; k++) {
    if (b[i + k] & 0xC0 != 0x80) return (0xFFFD, 1);
    rune = (rune << 6) | (b[i + k] & 0x3F);
  }
  if (rune < least || rune > 0x10FFFF || (rune >= 0xD800 && rune <= 0xDFFF)) {
    return (0xFFFD, 1);
  }
  return (rune, size);
}

/// The first code unit of the escape range. A byte that holds no rune of UTF-8
/// takes the code unit of this base and the value of the byte.
const int _escapeBase = 0xDC00;

/// Decodes bytes that hold no valid UTF-8. Each byte of an invalid sequence
/// takes 1 code unit of the escape range.
String _escape(GoString s) {
  final StringBuffer out = StringBuffer();
  var i = 0;
  while (i < s.bytes.length) {
    final (int rune, int size) = $decodeRuneRaw(s, i);
    if (rune == 0xFFFD && size == 1) {
      out.writeCharCode(_escapeBase | s.bytes[i]);
    } else {
      out.writeCharCode(rune);
    }
    i += size;
  }
  return out.toString();
}

/// Encodes a Dart string as UTF-8. A code unit of the escape range gives the 1
/// byte that it stands for.
///
/// The result is always a fresh buffer that this function alone holds, thus a
/// caller adopts it with GoString.take and makes no copy. utf8.encode builds
/// the buffer with a sublist, and Uint8List.fromList builds a new list.
Uint8List _encode(String s) {
  var escaped = false;
  for (var i = 0; i < s.length; i++) {
    final int unit = s.codeUnitAt(i);
    if (unit >= 0xDC80 &&
        unit <= 0xDCFF &&
        (i == 0 ||
            s.codeUnitAt(i - 1) < 0xD800 ||
            s.codeUnitAt(i - 1) > 0xDBFF)) {
      escaped = true;
      break;
    }
  }
  if (!escaped) return utf8.encode(s);
  final List<int> out = <int>[];
  final StringBuffer part = StringBuffer();
  for (var i = 0; i < s.length; i++) {
    final int unit = s.codeUnitAt(i);
    if (unit >= 0xDC80 &&
        unit <= 0xDCFF &&
        (i == 0 ||
            s.codeUnitAt(i - 1) < 0xD800 ||
            s.codeUnitAt(i - 1) > 0xDBFF)) {
      if (part.isNotEmpty) {
        out.addAll(utf8.encode(part.toString()));
        part.clear();
      }
      out.add(unit & 0xFF);
      continue;
    }
    part.writeCharCode(unit);
  }
  if (part.isNotEmpty) out.addAll(utf8.encode(part.toString()));
  return Uint8List.fromList(out);
}

/// The smaller of 2 values, in the order of Go. A NaN passes through, and a
/// string compares by bytes.
Object? $minOf(Object? a, Object? b) {
  if (a is GoFloat32 && b is GoFloat32) {
    if ((a.bits & 0x7fffffff) == 0 && (b.bits & 0x7fffffff) == 0) {
      return GoFloat32.fromBits(a.bits | b.bits);
    }
    return a.isNaN || (!b.isNaN && a <= b) ? a : b;
  }
  if (a is GoString) return $compare(a, b as GoString) <= 0 ? a : b;
  if (a is GoInt) return a <= (b as GoInt) ? a : b;
  if (a is GoFloat64 &&
      b is GoFloat64 &&
      a == GoFloat64.zero &&
      b == GoFloat64.zero) {
    return a.isNegative ? a : b;
  }
  if (a is GoFloat64 && a.isNaN) return a;
  if (b is GoFloat64 && b.isNaN) return b;
  return (a as GoFloat64) <= (b as GoFloat64) ? a : b;
}

/// The larger of 2 values, in the order of Go.
Object? $maxOf(Object? a, Object? b) {
  if (a is GoFloat32 && b is GoFloat32) {
    if ((a.bits & 0x7fffffff) == 0 && (b.bits & 0x7fffffff) == 0) {
      return GoFloat32.fromBits(a.bits & b.bits);
    }
    return a.isNaN || (!b.isNaN && a >= b) ? a : b;
  }
  if (a is GoString) return $compare(a, b as GoString) >= 0 ? a : b;
  if (a is GoInt) return a >= (b as GoInt) ? a : b;
  if (a is GoFloat64 &&
      b is GoFloat64 &&
      a == GoFloat64.zero &&
      b == GoFloat64.zero) {
    return a.isNegative ? b : a;
  }
  if (a is GoFloat64 && a.isNaN) return a;
  if (b is GoFloat64 && b.isNaN) return b;
  return (a as GoFloat64) >= (b as GoFloat64) ? a : b;
}

/// The smaller of 2 values of an unsigned 64-bit type. A Go uint64 above 2 to
/// the power 63 holds a negative Dart int, thus the signed order inverts.
GoInt $uminOf(GoInt a, GoInt b) => $ule(a, b) ? a : b;

/// The larger of 2 values of an unsigned 64-bit type.
GoInt $umaxOf(GoInt a, GoInt b) => $ule(a, b) ? b : a;

/// The spread of a value of a union of a string and a byte slice. Only append
/// reads it, thus the string form shares the storage. Read $stringBytes.
Slice<GoInt> $spreadBytes(Object? v) {
  if (v is GoString) return $stringBytes(v);
  return v as Slice<GoInt>;
}

import 'dart:convert';
import 'dart:typed_data';

import 'panic.dart';
import 'slice.dart';

/// A Go string: an immutable sequence of bytes.
///
/// A Dart `String` holds UTF-16 code units, thus it cannot hold the bytes of a
/// Go string. Convert between the 2 types only at an interop boundary.
final class GoString {
  /// The bytes of the string. A write to this list throws.
  ///
  /// The storage is unmodifiable, thus Dart passes it between 2 isolates by
  /// reference and makes no copy.
  final Uint8List bytes;

  const GoString._(this.bytes);

  /// The zero value of a Go string.
  static final GoString empty = GoString.fromBytes(const <int>[]);

  /// Encodes a Dart string as UTF-8.
  factory GoString.fromString(String s) => GoString.fromBytes(utf8.encode(s));

  /// Copies the bytes into unmodifiable storage.
  factory GoString.fromBytes(List<int> bytes) =>
      GoString._(Uint8List.fromList(bytes).asUnmodifiableView());

  /// Wraps bytes that the caller never changes again.
  factory GoString.take(Uint8List bytes) =>
      GoString._(bytes.asUnmodifiableView());

  /// The number of bytes. This value is not the number of runes.
  int get length => bytes.length;

  /// The byte at index [i], with the bounds check of Go.
  int operator [](int i) {
    if (i < 0 || i >= bytes.length) $panicIndex(i, bytes.length);
    return bytes[i];
  }

  /// Decodes the bytes as UTF-8. An invalid byte becomes U+FFFD, as in Go.
  String toDartString() => utf8.decode(bytes, allowMalformed: true);

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    if (other is! GoString || other.bytes.length != bytes.length) return false;
    for (var i = 0; i < bytes.length; i++) {
      if (other.bytes[i] != bytes[i]) return false;
    }
    return true;
  }

  @override
  int get hashCode => Object.hashAll(bytes);

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
/// Go string never changes.
GoString $substring(GoString s, int low, int high) {
  if (low < 0 || high < low || high > s.length) {
    $panicSlice(low, high, s.length);
  }
  if (low == 0 && high == s.length) return s;
  return GoString.take(
    Uint8List.view(s.bytes.buffer, s.bytes.offsetInBytes + low, high - low),
  );
}

/// The Go conversion `[]byte(s)`. Go copies the bytes.
Slice<int> $stringToBytes(GoString s) => Slice.of(Uint8List.fromList(s.bytes));

/// The Go conversion `string(b)` for a byte slice.
GoString $bytesToString(Slice<int> b) {
  final out = Uint8List(b.len);
  for (var i = 0; i < b.len; i++) {
    out[i] = b.array[b.off + i];
  }
  return GoString.take(out);
}

/// The Go conversion `string(r)` for 1 rune.
GoString $runeToString(int rune) {
  if (rune < 0 || rune > 0x10FFFF || (rune >= 0xD800 && rune <= 0xDFFF)) {
    rune = 0xFFFD;
  }
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
Slice<int> $stringToRunes(GoString s) {
  final out = <int>[];
  var i = 0;
  while (i < s.length) {
    final (int rune, int size) = $decodeRune(s, i);
    out.add(rune);
    i += size;
  }
  return Slice.of(Int32List.fromList(out));
}

/// The Go conversion `string(r)` for a rune slice.
GoString $runesToString(Slice<int> runes) {
  var out = GoString.empty;
  for (var i = 0; i < runes.len; i++) {
    out = $concat(out, $runeToString(runes.array[runes.off + i]));
  }
  return out;
}

/// Reads 1 UTF-8 rune at a byte index. An invalid sequence gives U+FFFD with
/// a size of 1, exactly as a Go range loop does.
(int, int) $decodeRune(GoString s, int i) {
  final b = s.bytes;
  final int b0 = b[i];
  if (b0 < 0x80) return (b0, 1);
  int size;
  int rune;
  if (b0 >= 0xF0) {
    size = 4;
    rune = b0 & 0x07;
  } else if (b0 >= 0xE0) {
    size = 3;
    rune = b0 & 0x0F;
  } else if (b0 >= 0xC0) {
    size = 2;
    rune = b0 & 0x1F;
  } else {
    return (0xFFFD, 1);
  }
  if (i + size > b.length) return (0xFFFD, 1);
  for (var k = 1; k < size; k++) {
    if (b[i + k] & 0xC0 != 0x80) return (0xFFFD, 1);
    rune = (rune << 6) | (b[i + k] & 0x3F);
  }
  if (rune > 0x10FFFF || (rune >= 0xD800 && rune <= 0xDFFF)) {
    return (0xFFFD, 1);
  }
  return (rune, size);
}

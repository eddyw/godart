import 'dart:convert';
import 'dart:typed_data';

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

  GoString._(this.bytes);

  /// The zero value of a Go string.
  static final GoString empty = GoString.fromBytes(const <int>[]);

  /// Encodes a Dart string as UTF-8.
  factory GoString.fromString(String s) => GoString.fromBytes(utf8.encode(s));

  /// Copies the bytes into unmodifiable storage.
  factory GoString.fromBytes(List<int> bytes) =>
      GoString._(Uint8List.fromList(bytes).asUnmodifiableView());

  /// The number of bytes. This value is not the number of runes.
  int get length => bytes.length;

  /// The byte at index [i].
  int operator [](int i) => bytes[i];

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

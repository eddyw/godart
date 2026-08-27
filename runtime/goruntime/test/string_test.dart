import 'dart:typed_data';

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

void main() {
  group('GoString', () {
    test('keeps the text through a round trip', () {
      expect(
        GoString.fromString('hello, world').toDartString(),
        'hello, world',
      );
    });

    test('counts bytes, and not runes', () {
      expect(GoString.fromString('héllo').length, 6);
      expect(GoString.fromString('héllo').toDartString().length, 5);
    });

    test('holds the UTF-8 bytes', () {
      expect(GoString.fromString('é').bytes, <int>[0xC3, 0xA9]);
    });

    test('rejects a write to the storage', () {
      final s = GoString.fromString('go');
      expect(() => s.bytes[0] = 0, throwsUnsupportedError);
    });

    test('copies the source bytes', () {
      final source = Uint8List.fromList(<int>[103, 111]);
      final s = GoString.fromBytes(source);
      source[0] = 0;
      expect(s.toDartString(), 'go');
    });

    test('gives 2 equal values an equal hash code', () {
      final a = GoString.fromString('gart');
      final b = GoString.fromBytes(<int>[103, 97, 114, 116]);
      expect(a, b);
      expect(a.hashCode, b.hashCode);
    });

    test('replaces an invalid byte with U+FFFD', () {
      expect(GoString.fromBytes(<int>[0xFF]).toDartString(), '�');
    });

    test('has an empty zero value', () {
      expect(GoString.empty.length, 0);
      expect(GoString.empty, GoString.fromString(''));
    });
  });
}

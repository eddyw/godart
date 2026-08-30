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

    test('keeps a byte that holds no rune through a round trip', () {
      final GoString bad = GoString.fromBytes(<int>[0x68, 0xFF, 0x69]);
      expect(GoString.fromString(bad.toDartString()).bytes, bad.bytes);
      expect(bad.toDartString().length, 3);
    });

    test('keeps each byte of a sequence that holds no rune', () {
      final GoString bad = GoString.fromBytes(<int>[0xED, 0xA0, 0x80, 0x41]);
      expect(GoString.fromString(bad.toDartString()).bytes, bad.bytes);
    });

    test('has an empty zero value', () {
      expect(GoString.empty.length, 0);
      expect(GoString.empty, GoString.fromString(''));
    });
  });
  _moreTests();
}

void _moreTests() {
  group('string operations', () {
    test('concat joins the bytes', () {
      expect(
        $concat(GoString.fromString('go'), GoString.fromString('dart')),
        GoString.fromString('godart'),
      );
    });

    test('compare orders by byte', () {
      expect($compare(GoString.fromString('a'), GoString.fromString('b')), -1);
      expect($compare(GoString.fromString('ab'), GoString.fromString('a')), 1);
      expect($compare(GoString.fromString('a'), GoString.fromString('a')), 0);
    });

    test('a substring shares the storage', () {
      final s = GoString.fromString('godart');
      expect($substring(s, 2, 6).toDartString(), 'dart');
      expect(() => $substring(s, 0, 7), throwsA(isA<GoPanic>()));
    });

    test('a rune round trip keeps the text', () {
      final s = GoString.fromString('héllo');
      expect($runesToString($stringToRunes(s)), s);
    });

    test('decode reads 1 rune with its size', () {
      expect($decodeRune(GoString.fromString('é'), 0), (0xE9, 2));
      expect($decodeRune(GoString.fromBytes(<int>[0xFF]), 0), (0xFFFD, 1));
    });

    test('a byte round trip keeps the bytes', () {
      final s = GoString.fromString('godart');
      expect($bytesToString($stringToBytes(s)), s);
    });
  });
}

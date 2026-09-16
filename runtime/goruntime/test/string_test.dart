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

    test('preserves surrogate pairs beside escaped bytes', () {
      final bytes = <int>[0xF0, 0x9F, 0x92, 0xA9];
      expect(GoString.fromString('\u{1f4a9}').bytes, bytes);
      final mixed = GoString.fromBytes(<int>[0xFF, ...bytes, 0x80]);
      expect(GoString.fromString(mixed.toDartString()).bytes, mixed.bytes);
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

    test('gives the same text and the same hash at each read', () {
      final a = GoString.fromString('h\u00e9llo');
      final b = GoString.fromString('h\u00e9llo');
      expect(a.toDartString(), b.toDartString());
      expect(a.toDartString(), b.toDartString());
      expect(a.hashCode, b.hashCode);
      expect(a.hashCode, b.hashCode);
      final bad = GoString.fromBytes(<int>[0x68, 0xFF, 0x69]);
      expect(bad.toDartString(), bad.toDartString());
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

    test('gives the exact byte count with no spare tail', () {
      // utf8.encode fills a buffer of 3 bytes for each code unit, thus the
      // adopted buffer must hold the bytes of the text alone.
      expect(GoString.fromString('').bytes.length, 0);
      expect(GoString.fromString('go').bytes.length, 2);
      expect(GoString.fromString('\u00e9').bytes.length, 2);
      expect(GoString.fromString('\u65e5\u672c').bytes.length, 6);
    });

    test('rejects a write to an adopted buffer', () {
      final GoString plain = GoString.fromString('godart');
      expect(() => plain.bytes[0] = 0, throwsUnsupportedError);
      // The escape path builds the buffer with a different route.
      final GoString escaped = GoString.fromString(
        GoString.fromBytes(<int>[0x68, 0xFF]).toDartString(),
      );
      expect(escaped.bytes, <int>[0x68, 0xFF]);
      expect(() => escaped.bytes[0] = 0, throwsUnsupportedError);
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
      final Uint8List raw = Uint8List.fromList('godart'.codeUnits);
      final GoString s = GoString.take(raw);
      final GoString part = $substring(s, $gi(2), $gi(6));
      expect(part.bytes.offsetInBytes, s.bytes.offsetInBytes + 2);
      expect(part.bytes.length, 4);
      // A write to the adopted storage reaches the substring, thus the
      // substring holds a view and no copy.
      raw[2] = 0x44;
      expect(part.toDartString(), 'Dart');
      expect(() => $substring(s, $gi(0), $gi(7)), throwsA(isA<GoPanic>()));
    });

    test('a substring gives the hash of the equal value that stands alone', () {
      final part = $substring(GoString.fromString('godart'), $gi(2), $gi(6));
      final same = GoString.fromString('dart');
      expect(part, same);
      expect(part.hashCode, same.hashCode);
      expect(part.toDartString(), same.toDartString());
    });

    test('a rune round trip keeps the text', () {
      final s = GoString.fromString('héllo');
      expect($runesToString($stringToRunes(s)), s);
    });

    test('decode reads 1 rune with its size', () {
      expect($decodeRuneRaw(GoString.fromString('é'), 0), (0xE9, 2));
      expect($decodeRuneRaw(GoString.fromBytes(<int>[0xFF]), 0), (0xFFFD, 1));
    });

    test('decode rejects a sequence that Go rejects', () {
      // An overlong form of U+07FF, an overlong form of U+0000, a surrogate,
      // a value above U+10FFFF, and a start byte above 0xF7.
      for (final bytes in <List<int>>[
        <int>[0xE0, 0x9F, 0xBF],
        <int>[0xC0, 0x80],
        <int>[0xED, 0xA0, 0x80],
        <int>[0xF4, 0x90, 0x80, 0x80],
        <int>[0xF8, 0x80, 0x80, 0x80],
      ]) {
        expect($decodeRuneRaw(GoString.fromBytes(bytes), 0), (0xFFFD, 1));
      }
    });

    test('a byte round trip keeps the bytes', () {
      final s = GoString.fromString('godart');
      expect($bytesToString($stringToBytes(s)), s);
    });
  });
}

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

// The conformance test of the 32-bit float. Go reads a decimal into 32 bits
// with 1 rounding, and it writes the shortest decimal that reads back as the
// same 32-bit value. A read that gives a 64-bit value first and narrows it
// rounds 2 times, thus a decimal that stands exactly between 2 neighbour
// values of float32 takes the wrong neighbour.

final ByteData _bits = ByteData(8);

/// The 32-bit value of a bit pattern.
double fromBits(int pattern) {
  _bits.setUint32(0, pattern);
  return _bits.getFloat32(0);
}

/// The bit pattern of a 32-bit value.
int toBits(double v) {
  _bits.setFloat32(0, v);
  return _bits.getUint32(0);
}

/// The value that the text names, in 32 bits.
double parse32(String text) {
  final (double v, bool ok) = $parseFloat32(GoString.fromString(text));
  expect(ok, isTrue, reason: text);
  return v;
}

void main() {
  group('the read of a 32-bit float', () {
    test('a decimal at a middle point takes the neighbour of Go', () {
      // The 64-bit value of each text below lands exactly on a middle point
      // of 2 neighbour values of float32. Go reads the exact decimal, and it
      // thus rounds away from the even neighbour.
      expect(toBits(parse32('9000000000.0000001')), 1342577735);
      expect(toBits(parse32('7.038531e-26')), 363742205);
      expect(toBits(parse32('-7.038531e-26')), 363742205 | 0x80000000);
    });

    test('a decimal that is a middle point rounds to the even neighbour', () {
      // 2 to the power -150 stands between 0 and the smallest value of
      // float32. Go gives 0, because 0 holds the even mantissa.
      expect(parse32('7.0064923216240853546186479164495806564013e-46'), 0.0);
      // The next middle point stands between the 2 smallest values, and the
      // even neighbour is the second one.
      expect(
        toBits(parse32('2.1019476964872256063855943749348741969204e-45')),
        2,
      );
      // A text 1 step under the same middle point takes the lower neighbour.
      expect(toBits(parse32('2.1019476964872256063e-45')), 1);
    });

    test('a value outside the range gives an infinity', () {
      expect(parse32('1e39'), double.infinity);
      expect(parse32('-1e39'), double.negativeInfinity);
      expect(parse32('1e-46'), 0.0);
    });

    test('the plain values keep their bits', () {
      expect(toBits(parse32('1')), 0x3F800000);
      expect(toBits(parse32('0.5')), 0x3F000000);
      expect(toBits(parse32('3.1415927')), 0x40490FDB);
      expect(parse32('0'), 0.0);
    });

    test('a text that holds no float reports false', () {
      expect($parseFloat32(GoString.fromString('abc')).$2, isFalse);
    });
  });

  group('the text of a 32-bit float', () {
    test('the shortest text of Go stands', () {
      expect($f32string(fromBits(363742205)), '7.038531e-26');
      expect($f32string(fromBits(0x40490FDB)), '3.1415927');
      expect($f32string(1.0), '1');
      expect($f32string(0.5), '0.5');
    });

    test('each text reads back, and no shorter text does', () {
      // The set holds each binade, each boundary, and a spread of patterns.
      final List<int> patterns = <int>[
        1,
        2,
        3,
        0x007FFFFF,
        0x00800000,
        0x00800001,
        363742205,
        363742206,
        1342577734,
        1342577735,
        0x3F800000,
        0x40490FDB,
        0x7F7FFFFF,
      ];
      // A deterministic spread reaches each exponent. The step is odd, thus
      // the mantissa moves at each turn.
      int seed = 12345;
      for (int i = 0; i < 4000; i++) {
        seed = (seed * 1103515245 + 12345) & 0x7FFFFFFF;
        patterns.add(seed);
      }
      for (final int pattern in patterns) {
        final double v = fromBits(pattern);
        if (v.isNaN || v.isInfinite) continue;
        final String text = $f32string(v);
        // The text reads back as the same 32-bit value.
        expect(parse32(text), v, reason: 'the text $text of $pattern');
        if (v == 0) continue;
        // No text of fewer significant digits reads back. The nearest text of
        // a given count of digits is the best one of that count, thus the
        // first count that reads back is the shortest.
        int least = 9;
        for (int digits = 1; digits <= 9; digits++) {
          if (parse32(v.toStringAsPrecision(digits)) == v) {
            least = digits;
            break;
          }
        }
        expect(
          double.parse(text),
          double.parse(v.toStringAsPrecision(least)),
          reason: 'the text $text of $pattern is not the shortest',
        );
      }
    });
  });
}

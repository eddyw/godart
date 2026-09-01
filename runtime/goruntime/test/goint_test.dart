import 'package:goruntime/src/goint_js.dart' as js;
import 'package:goruntime/src/goint_native.dart' as nat;
import 'package:test/test.dart';

// The conformance test of the 2 forms of the Go integer. The native form on
// the VM holds the 64-bit semantics of Go, and the whole go test net proves
// it, thus the native result is the reference. The JS pair form must give
// the same value for each operation of the table.

const List<int> values = [
  0,
  1,
  2,
  -1,
  -2,
  10,
  -10,
  0x7F,
  0x80,
  0xFF,
  0x100,
  0x7FFF,
  0xFFFF,
  0x10000,
  0x7FFFFFFF,
  0x80000000,
  0xFFFFFFFF,
  0x100000000,
  4503599627370495,
  4503599627370496,
  9007199254740991,
  9007199254740992,
  4611686018427387904,
  -4611686018427387904,
  0x7FFFFFFFFFFFFFFF,
  -0x8000000000000000,
  6364136223846793005,
  -6148914691236517206,
  1442695040888963407,
  1311768467463790320,
];

const List<int> counts = [0, 1, 7, 15, 16, 31, 32, 33, 47, 63, 64, 100];

/// The exact value of a JS pair, on the VM.
int back(js.GoInt v) => v.$h * 4294967296 + v.$l;

js.GoInt js2(int v) => js.$gi2(v >> 32, v & 0xFFFFFFFF);

void main() {
  test('the 2 constructors give the value back', () {
    for (final v in values) {
      expect(back(js2(v)), v);
      expect(nat.$gi2(v >> 32, v & 0xFFFFFFFF).$v, v);
      if (v.abs() < 9007199254740992) {
        expect(back(js.$gi(v)), v);
        expect(js.$gi(v).$v, v);
      }
      expect(nat.$gi(v).$v, v);
    }
  });

  test('the binary operations match the native form', () {
    for (final a in values) {
      final na = nat.$gi(a);
      final ja = js2(a);
      for (final b in values) {
        final nb = nat.$gi(b);
        final jb = js2(b);
        expect(back(ja + jb), (na + nb).$v, reason: '$a + $b');
        expect(back(ja - jb), (na - nb).$v, reason: '$a - $b');
        expect(back(ja * jb), (na * nb).$v, reason: '$a * $b');
        expect(back(ja & jb), (na & nb).$v, reason: '$a & $b');
        expect(back(ja | jb), (na | nb).$v, reason: '$a | $b');
        expect(back(ja ^ jb), (na ^ nb).$v, reason: '$a ^ $b');
        expect(ja < jb, na.$v < nb.$v, reason: '$a < $b');
        expect(ja <= jb, na.$v <= nb.$v, reason: '$a <= $b');
        expect(ja == jb, na.$v == nb.$v, reason: '$a == $b');
        expect(js.$ult(ja, jb), nat.$ult(na, nb), reason: 'ult $a $b');
        expect(js.$ule(ja, jb), nat.$ule(na, nb), reason: 'ule $a $b');
        if (b != 0) {
          expect(back(js.$div(ja, jb)), nat.$div(na, nb).$v, reason: '$a / $b');
          expect(back(js.$mod(ja, jb)), nat.$mod(na, nb).$v, reason: '$a % $b');
          expect(
            back(js.$udiv(ja, jb)),
            nat.$udiv(na, nb).$v,
            reason: 'udiv $a $b',
          );
          expect(
            back(js.$umod(ja, jb)),
            nat.$umod(na, nb).$v,
            reason: 'umod $a $b',
          );
        }
      }
    }
  });

  test('the unary operations and the masks match the native form', () {
    for (final a in values) {
      final na = nat.$gi(a);
      final ja = js2(a);
      expect(back(~ja), (~na).$v, reason: 'not $a');
      expect(back(-ja), (-na).$v, reason: 'neg $a');
      expect(back(js.$i8(ja)), nat.$i8(na).$v, reason: 'i8 $a');
      expect(back(js.$i16(ja)), nat.$i16(na).$v, reason: 'i16 $a');
      expect(back(js.$i32(ja)), nat.$i32(na).$v, reason: 'i32 $a');
      expect(back(js.$u8(ja)), nat.$u8(na).$v, reason: 'u8 $a');
      expect(back(js.$u16(ja)), nat.$u16(na).$v, reason: 'u16 $a');
      expect(back(js.$u32(ja)), nat.$u32(na).$v, reason: 'u32 $a');
      expect(ja.toString(), na.$v.toString(), reason: 'text $a');
      expect(js.$ustring(ja), nat.$ustring(na), reason: 'utext $a');
      expect(ja.toDouble(), na.$v.toDouble(), reason: 'double $a');
      expect(js.$utodouble(ja), nat.$utodouble(na), reason: 'udouble $a');
      expect(ja.hashCode, js2(a).hashCode, reason: 'hash $a');
    }
  });

  test('the shifts match the native form', () {
    for (final a in values) {
      final na = nat.$gi(a);
      final ja = js2(a);
      for (final s in counts) {
        final ns = nat.$gi(s);
        final jss = js.$gi(s);
        expect(back(js.$shl(ja, jss)), nat.$shl(na, ns).$v, reason: '$a << $s');
        expect(back(js.$shr(ja, jss)), nat.$shr(na, ns).$v, reason: '$a >> $s');
        expect(
          back(js.$ushr(ja, jss)),
          nat.$ushr(na, ns).$v,
          reason: '$a >>> $s',
        );
      }
    }
  });

  test('the float bits round through the 2 forms', () {
    const List<double> floats = [
      0.0,
      1.5,
      -2.25,
      3.141592653589793,
      5e-324,
      1.7976931348623157e308,
      double.infinity,
      double.negativeInfinity,
    ];
    for (final f in floats) {
      expect(back(js.$f64ToBits(f)), nat.$f64ToBits(f).$v, reason: 'bits $f');
      expect(js.$f64FromBits(js2(nat.$f64ToBits(f).$v)), f, reason: 'back $f');
      expect(back(js.$f32ToBits(f)), nat.$f32ToBits(f).$v, reason: 'bits32 $f');
    }
    expect(back(js.$f64ToBits(0.0 / 0.0)), nat.$f64ToBits(0.0 / 0.0).$v);
    expect(js.$f64FromBits(js2(-1)).isNaN, true);
    final double negZero = -0.0;
    expect(back(js.$f64ToBits(negZero)), nat.$f64ToBits(negZero).$v);
  });

  test('the division of the smallest value by -1 wraps', () {
    const int min = -0x8000000000000000;
    expect(
      back(js.$div(js2(min), js2(-1))),
      nat.$div(nat.$gi(min), nat.$gi(-1)).$v,
    );
  });
}

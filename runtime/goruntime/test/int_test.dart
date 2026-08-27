import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

void main() {
  test('a narrow mask wraps like Go', () {
    expect($i8(127 + 1), -128);
    expect($u8(-1), 255);
    expect($i32(2147483647 + 1), -2147483648);
    expect($u32(-1), 4294967295);
    expect($i16(-32768 - 1), 32767);
  });

  test('division truncates toward zero', () {
    expect($div(-7, 2), -3);
    expect($mod(-7, 2), -1);
    expect($mod(7, -2), 1);
  });

  test('a division by zero panics', () {
    expect(() => $div(1, 0), throwsA(isA<GoPanic>()));
  });

  test('unsigned division reads the pattern as unsigned', () {
    expect($udiv(-1, 2), 9223372036854775807);
    expect($umod(-1, 10), 5);
    expect($ult(-1, 1), false);
    expect($ult(1, -1), true);
  });

  test('a shift past the width gives the Go result', () {
    expect($shl(1, 64), 0);
    expect($shr(-1, 64), -1);
    expect($ushr(-1, 63), 1);
  });

  test('an unsigned value prints without a sign', () {
    expect($ustring(-1), '18446744073709551615');
    expect($ustring(7), '7');
  });

  test('float32 rounds to 32-bit precision', () {
    expect($f32(0.1), closeTo(0.10000000149011612, 1e-18));
  });
}

import 'dart:typed_data';

import 'panic.dart';

// A Dart int on the VM holds 64 bits and wraps on overflow, thus int, int64,
// uint, uint64, and uintptr need no mask. A narrow type needs a mask after
// each operation.

int $i8(int v) => (v & 0xFF) - ((v & 0x80) << 1);

int $i16(int v) => (v & 0xFFFF) - ((v & 0x8000) << 1);

int $i32(int v) => (v & 0xFFFFFFFF) - ((v & 0x80000000) << 1);

int $u8(int v) => v & 0xFF;

int $u16(int v) => v & 0xFFFF;

int $u32(int v) => v & 0xFFFFFFFF;

/// Truncated division, as in Go. Dart rounds toward zero with `~/`.
int $div(int a, int b) {
  if (b == 0) $panicDivide();
  return a ~/ b;
}

/// The remainder keeps the sign of the dividend, as in Go. The Dart operator
/// `%` gives a value with the sign of the divisor, thus it does not fit.
int $mod(int a, int b) {
  if (b == 0) $panicDivide();
  return a.remainder(b);
}

/// Unsigned division of 2 64-bit patterns.
int $udiv(int a, int b) {
  if (b == 0) $panicDivide();
  if (b < 0) return $ult(a, b) ? 0 : 1;
  if (a >= 0) return a ~/ b;
  final int q = ((a >>> 1) ~/ b) << 1;
  return $ult(a - q * b, b) ? q : q + 1;
}

int $umod(int a, int b) => a - $udiv(a, b) * b;

/// Unsigned comparison of 2 64-bit patterns.
bool $ult(int a, int b) => (a ^ 0x8000000000000000) < (b ^ 0x8000000000000000);

bool $ule(int a, int b) => !$ult(b, a);

/// A left shift. Go gives 0 when the count reaches the width of the type.
int $shl(int v, int count) {
  if (count < 0) $panic(const GoError('negative shift amount'));
  if (count >= 64) return 0;
  return v << count;
}

/// An arithmetic right shift. Go keeps the sign bit.
int $shr(int v, int count) {
  if (count < 0) $panic(const GoError('negative shift amount'));
  if (count >= 64) return v < 0 ? -1 : 0;
  return v >> count;
}

/// A logical right shift, for an unsigned type.
int $ushr(int v, int count) {
  if (count < 0) $panic(const GoError('negative shift amount'));
  if (count >= 64) return 0;
  return v >>> count;
}

/// The decimal text of a 64-bit pattern that holds an unsigned value.
String $ustring(int v) {
  if (v >= 0) return v.toString();
  final int q = $udiv(v, 10);
  return '$q${v - q * 10}';
}

/// The value of an unsigned 64-bit pattern as a double.
double $utodouble(int v) {
  if (v >= 0) return v.toDouble();
  return (v >>> 1).toDouble() * 2.0 + (v & 1).toDouble();
}

final _f32 = Float32List(1);

/// Rounds a double to 32-bit precision, as float32 does.
double $f32(double v) {
  _f32[0] = v;
  return _f32[0];
}

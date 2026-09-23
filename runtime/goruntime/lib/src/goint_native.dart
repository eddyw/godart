import 'dart:typed_data';

import 'int.dart';
import 'panic.dart';

// The native form of the Go integer. The extension type erases at the
// compile, thus the VM, AOT, and the wasm target see a plain int, and the
// emitted arithmetic costs nothing. goint_js.dart holds the form of the JS
// target, and the 2 files hold 1 surface. Read docs/js-integers.md.

extension type const GoInt(int $v) implements int {
  static const GoInt zero = GoInt(0);

  GoInt operator +(int o) => GoInt($v + o);
  GoInt operator -(int o) => GoInt($v - o);
  GoInt operator *(int o) => GoInt($v * o);
  GoInt operator &(int o) => GoInt($v & o);
  GoInt operator |(int o) => GoInt($v | o);
  GoInt operator ^(int o) => GoInt($v ^ o);
  GoInt operator ~() => GoInt(~$v);
  GoInt operator -() => GoInt(-$v);
}

/// The Go integer of a Dart int. A literal and each boundary value take
/// this form.
GoInt $gi(int v) => GoInt(v);

/// The Go integer of 2 32-bit halves. A literal outside 53 bits takes this
/// form, because a JS number holds 53 bits of an integer.
GoInt $gi2(int high, int low) => GoInt((high << 32) | low);

GoInt $i8(GoInt v) => GoInt((v.$v & 0xFF) - ((v.$v & 0x80) << 1));

GoInt $i16(GoInt v) => GoInt((v.$v & 0xFFFF) - ((v.$v & 0x8000) << 1));

GoInt $i32(GoInt v) => GoInt((v.$v & 0xFFFFFFFF) - ((v.$v & 0x80000000) << 1));

GoInt $u8(GoInt v) => GoInt(v.$v & 0xFF);

GoInt $u16(GoInt v) => GoInt(v.$v & 0xFFFF);

GoInt $u32(GoInt v) => GoInt(v.$v & 0xFFFFFFFF);

/// Truncated division, as in Go. Dart rounds toward zero with `~/`.
GoInt $div(GoInt a, GoInt b) {
  if (b.$v == 0) $panicDivide();
  return GoInt(a.$v ~/ b.$v);
}

/// The remainder keeps the sign of the dividend, as in Go. The Dart operator
/// `%` gives a value with the sign of the divisor, thus it does not fit.
GoInt $mod(GoInt a, GoInt b) {
  if (b.$v == 0) $panicDivide();
  return GoInt(a.$v.remainder(b.$v));
}

/// Unsigned division of 2 64-bit patterns.
GoInt $udiv(GoInt a, GoInt b) {
  if (b.$v == 0) $panicDivide();
  if (b.$v < 0) return $ult(a, b) ? GoInt.zero : const GoInt(1);
  if (a.$v >= 0) return GoInt(a.$v ~/ b.$v);
  final int q = ((a.$v >>> 1) ~/ b.$v) << 1;
  return GoInt($ult(GoInt(a.$v - q * b.$v), b) ? q : q + 1);
}

GoInt $umod(GoInt a, GoInt b) => GoInt(a.$v - $udiv(a, b).$v * b.$v);

/// Unsigned comparison of 2 64-bit patterns.
bool $ult(GoInt a, GoInt b) =>
    (a.$v ^ 0x8000000000000000) < (b.$v ^ 0x8000000000000000);

bool $ule(GoInt a, GoInt b) => !$ult(b, a);

/// The count of a shift whose type is unsigned. Go holds no negative count of
/// such a type, thus a pattern that reads as 64 or more unsigned takes the
/// width and the shift gives the value that Go gives.
GoInt $ucount(GoInt c) => $ult(c, $gi(64)) ? c : $gi(64);

/// A left shift. Go gives 0 when the count reaches the width of the type.
GoInt $shl(GoInt v, GoInt count) {
  if (count.$v < 0) $panicShift();
  if (count.$v >= 64) return GoInt.zero;
  return GoInt(v.$v << count.$v);
}

/// An arithmetic right shift. Go keeps the sign bit.
GoInt $shr(GoInt v, GoInt count) {
  if (count.$v < 0) $panicShift();
  if (count.$v >= 64) return GoInt(v.$v < 0 ? -1 : 0);
  return GoInt(v.$v >> count.$v);
}

/// A logical right shift, for an unsigned type.
GoInt $ushr(GoInt v, GoInt count) {
  if (count.$v < 0) $panicShift();
  if (count.$v >= 64) return GoInt.zero;
  return GoInt(v.$v >>> count.$v);
}

/// The decimal text of a 64-bit pattern that holds an unsigned value.
String $ustring(GoInt v) {
  if (v.$v >= 0) return v.$v.toString();
  final GoInt q = $udiv(v, const GoInt(10));
  return '${q.$v}${v.$v - q.$v * 10}';
}

/// The value of an unsigned 64-bit pattern as a double.
/// The float64 value of a Go integer.
///
/// int.toDouble() is a virtual call, because a Dart int holds 2 forms. The
/// compiler does not fold it, and it costs 5 times an addition of 0.0. The
/// operator + of int reads a double operand and it gives the same value,
/// because a Go integer holds no negative zero and no NaN.
double $todouble(GoInt v) => v.$v + 0.0;

double $utodouble(GoInt v) {
  if (v.$v >= 0) return v.$v + 0.0;
  return ((v.$v >>> 1) + 0.0) * 2.0 + ((v.$v & 1) + 0.0);
}

/// The IEEE 754 bit pattern of a 64-bit value.
GoInt $f64ToBits(GoFloat64 v) => GoInt((v.high << 32) | v.low);

/// The 64-bit value of an IEEE 754 bit pattern.
GoFloat64 $f64FromBits(GoInt b) =>
    GoFloat64.fromBits((b.$v >>> 32) & 0xffffffff, b.$v & 0xffffffff);

/// The IEEE 754 bit pattern of a 32-bit value. The result holds 32 bits, thus
/// it stays positive.
GoInt $f32ToBits(GoFloat32 v) => GoInt(v.bits);

/// The 32-bit value of an IEEE 754 bit pattern.
GoFloat32 $f32FromBits(GoInt b) => GoFloat32.fromBits(b.$v);

/// The plain Dart int of a Go integer, for an index, a length, or another
/// value that a Dart API takes. The module form spares the caller a
/// parenthesis around a compound expression.
int $ix(GoInt v) => v.$v;

/// The Go integer of a double, with the truncation of Go.
GoInt $fromF64(double v) => GoInt(v.truncate());

/// The value of a float64 as an unsigned 64-bit integer. truncate of Dart
/// stops at the largest signed value, thus a value of at least 2 to the power
/// 63 needs the wrap of the unsigned range.
GoInt $fromF64U(double v) {
  final double t = v.truncateToDouble();
  if (t < 9223372036854775808.0) return GoInt(t.toInt());
  return GoInt(BigInt.from(t).toUnsigned(64).toSigned(64).toInt());
}

/// True inside the build of the JS target. The conditional import selects
/// this file per target, thus the value is a constant of the file.
const bool $isJS = false;

/// The text of the 64-bit pattern of a Go integer in a radix, read as an
/// unsigned value.
String $uradix(GoInt v, int radix) {
  if (v.$v >= 0) return v.$v.toRadixString(radix);
  return BigInt.from(v.$v).toUnsigned(64).toRadixString(radix);
}

// The allocators of a Go integer slice. The native form keeps the dense list
// of the width, because the list type erases to List<int> here. A write into
// a dense list wraps at the width, and each store of a narrow Go type is
// masked already.
List<GoInt> $goList8(int n) => Int8List(n) as List<GoInt>;
List<GoInt> $goList16(int n) => Int16List(n) as List<GoInt>;
List<GoInt> $goList32(int n) => Int32List(n) as List<GoInt>;
List<GoInt> $goList64(int n) => Int64List(n) as List<GoInt>;
List<GoInt> $goListU8(int n) => Uint8List(n) as List<GoInt>;
List<GoInt> $goListU16(int n) => Uint16List(n) as List<GoInt>;
List<GoInt> $goListU32(int n) => Uint32List(n) as List<GoInt>;
List<GoInt> $goListU64(int n) => Uint64List(n) as List<GoInt>;

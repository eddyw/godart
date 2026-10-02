import 'goint_division.dart'
    if (dart.library.js) 'goint_division_js.dart'
    as division;
import 'int.dart';
import 'panic.dart';

// The JS form of the Go integer, in the pair model of GopherJS. A JS number
// holds 53 bits of an integer, thus a 64-bit value is a pair of 32-bit
// halves. The high half holds a signed value, and the low half holds an
// unsigned value. Each step below stays under 53 bits, thus each step is
// exact. The division, the remainder, and the decimal text ride BigInt,
// because they are rare next to the other operations. goint_native.dart
// holds the form of the other targets, and the 2 files hold 1 surface.
// Read docs/js-integers.md.

const int _b32 = 4294967296;
const int _b16 = 65536;

/// The signed 32-bit value of the low 32 bits of x.
int _s32(int x) => x.toSigned(32);

/// The unsigned 32-bit value of the low 32 bits of x.
int _u32(int x) => x.toUnsigned(32);

final class GoInt {
  /// The high half, as a signed 32-bit value.
  final int $h;

  /// The low half, as an unsigned 32-bit value.
  final int $l;

  const GoInt._(this.$h, this.$l);

  static const GoInt zero = GoInt._(0, 0);

  /// The value as a plain Dart int. A value outside 53 bits loses its low
  /// bits, thus the getter serves a boundary value only: an index, a
  /// length, or a handle.
  int get $v => _b32 * $h + $l;

  GoInt operator +(GoInt o) {
    int low = $l + o.$l;
    int carry = 0;
    if (low >= _b32) {
      low -= _b32;
      carry = 1;
    }
    return GoInt._(_s32($h + o.$h + carry), low);
  }

  GoInt operator -(GoInt o) {
    int low = $l - o.$l;
    int borrow = 0;
    if (low < 0) {
      low += _b32;
      borrow = 1;
    }
    return GoInt._(_s32($h - o.$h - borrow), low);
  }

  GoInt operator -() => zero - this;

  /// The multiply splits each half into 16-bit parts, because a JS multiply
  /// of 2 values over 2^26 rounds. Each lane stays under 53 bits.
  GoInt operator *(GoInt o) {
    final int a0 = $l % _b16;
    final int a1 = $l ~/ _b16;
    final int ua = _u32($h);
    final int a2 = ua % _b16;
    final int a3 = ua ~/ _b16;
    final int b0 = o.$l % _b16;
    final int b1 = o.$l ~/ _b16;
    final int ub = _u32(o.$h);
    final int b2 = ub % _b16;
    final int b3 = ub ~/ _b16;
    final int c0 = a0 * b0;
    final int c1 = c0 ~/ _b16 + a1 * b0 + a0 * b1;
    final int c2 = c1 ~/ _b16 + a2 * b0 + a1 * b1 + a0 * b2;
    final int c3 = c2 ~/ _b16 + a3 * b0 + a2 * b1 + a1 * b2 + a0 * b3;
    final int low = (c1 % _b16) * _b16 + (c0 % _b16);
    final int high = _s32((c3 % _b16) * _b16 + (c2 % _b16));
    return GoInt._(high, low);
  }

  GoInt operator &(GoInt o) => GoInt._(_s32($h & o.$h), _u32($l & o.$l));

  GoInt operator |(GoInt o) => GoInt._(_s32($h | o.$h), _u32($l | o.$l));

  GoInt operator ^(GoInt o) => GoInt._(_s32($h ^ o.$h), _u32($l ^ o.$l));

  GoInt operator ~() => GoInt._(_s32(~$h), _u32(~$l));

  bool operator <(GoInt o) {
    if ($h != o.$h) return $h < o.$h;
    return $l < o.$l;
  }

  bool operator <=(GoInt o) => !(o < this);

  bool operator >(GoInt o) => o < this;

  bool operator >=(GoInt o) => !(this < o);

  @override
  bool operator ==(Object other) =>
      other is GoInt && $h == other.$h && $l == other.$l;

  @override
  int get hashCode => ($h ^ $l) & 0x3FFFFFFF;

  double toDouble() => _b32 * $h + $l.toDouble();

  /// The text in a radix. The signed value takes a minus sign, exactly as
  /// int.toRadixString gives it.
  String toRadixString(int radix) => _sbig(this).toRadixString(radix);

  @override
  String toString() {
    if ($h >= -2097152 && $h < 2097152) return $v.toString();
    return _sbig(this).toString();
  }
}

BigInt _sbig(GoInt a) =>
    BigInt.from(a.$h) * BigInt.from(_b32) + BigInt.from(a.$l);

BigInt _ubig(GoInt a) => _sbig(a).toUnsigned(64);

GoInt _ofParts((int, int) value) => GoInt._(_s32(value.$1), value.$2);

/// The Go integer of a Dart int. The value must hold 53 bits or less,
/// because a JS number holds no more of an integer.
GoInt $gi(int v) {
  final int low = v % _b32;
  return GoInt._(_s32((v - low) ~/ _b32), low);
}

/// The Go integer of 2 32-bit halves. A literal outside 53 bits takes this
/// form.
GoInt $gi2(int high, int low) => GoInt._(_s32(high), _u32(low));

GoInt $i8(GoInt v) {
  final int b = v.$l & 0xFF;
  return $gi(b - ((b & 0x80) << 1));
}

GoInt $i16(GoInt v) {
  final int b = v.$l & 0xFFFF;
  return $gi(b - ((b & 0x8000) << 1));
}

GoInt $i32(GoInt v) {
  final int s = _s32(v.$l);
  return GoInt._(s < 0 ? -1 : 0, v.$l);
}

GoInt $u8(GoInt v) => GoInt._(0, v.$l & 0xFF);

GoInt $u16(GoInt v) => GoInt._(0, v.$l & 0xFFFF);

GoInt $u32(GoInt v) => GoInt._(0, v.$l);

/// Truncated division, as in Go.
GoInt $div(GoInt a, GoInt b) {
  if (b.$h == 0 && b.$l == 0) $panicDivide();
  return _ofParts(division.divide(a.$h, a.$l, b.$h, b.$l));
}

/// The remainder keeps the sign of the dividend, as in Go.
GoInt $mod(GoInt a, GoInt b) {
  if (b.$h == 0 && b.$l == 0) $panicDivide();
  return _ofParts(division.remainder(a.$h, a.$l, b.$h, b.$l));
}

/// Unsigned division of 2 64-bit patterns.
GoInt $udiv(GoInt a, GoInt b) {
  if (b.$h == 0 && b.$l == 0) $panicDivide();
  return _ofParts(division.divide(_u32(a.$h), a.$l, _u32(b.$h), b.$l));
}

GoInt $umod(GoInt a, GoInt b) {
  if (b.$h == 0 && b.$l == 0) $panicDivide();
  return _ofParts(division.remainder(_u32(a.$h), a.$l, _u32(b.$h), b.$l));
}

/// Unsigned comparison of 2 64-bit patterns.
bool $ult(GoInt a, GoInt b) {
  final int ua = _u32(a.$h);
  final int ub = _u32(b.$h);
  if (ua != ub) return ua < ub;
  return a.$l < b.$l;
}

bool $ule(GoInt a, GoInt b) => !$ult(b, a);

/// The count of a shift whose type is unsigned. Go holds no negative count of
/// such a type, thus a pattern that reads as 64 or more unsigned takes the
/// width and the shift gives the value that Go gives.
GoInt $ucount(GoInt c) => $ult(c, $gi(64)) ? c : $gi(64);

/// A left shift. Go gives 0 when the count reaches the width of the type.
GoInt $shl(GoInt v, GoInt count) {
  final int s = count.$v;
  if (s < 0) $panicShift();
  if (s >= 64) return GoInt.zero;
  if (s == 0) return v;
  if (s < 32) {
    final int low = _u32(v.$l << s);
    final int high = _s32((v.$h << s) | (v.$l >> (32 - s)));
    return GoInt._(high, low);
  }
  if (s == 32) return GoInt._(_s32(v.$l), 0);
  return GoInt._(_s32(v.$l << (s - 32)), 0);
}

/// An arithmetic right shift. Go keeps the sign bit.
GoInt $shr(GoInt v, GoInt count) {
  final int s = count.$v;
  if (s < 0) $panicShift();
  if (s >= 64) return v.$h < 0 ? const GoInt._(-1, 0xFFFFFFFF) : GoInt.zero;
  if (s == 0) return v;
  if (s < 32) {
    final int low = _u32((v.$l >> s) | (v.$h << (32 - s)));
    final int high = _s32(v.$h >> s);
    return GoInt._(high, low);
  }
  if (s == 32) return GoInt._(v.$h < 0 ? -1 : 0, _u32(v.$h));
  return GoInt._(v.$h < 0 ? -1 : 0, _u32(v.$h >> (s - 32)));
}

/// A logical right shift, for an unsigned type.
GoInt $ushr(GoInt v, GoInt count) {
  final int s = count.$v;
  if (s < 0) $panicShift();
  if (s >= 64) return GoInt.zero;
  if (s == 0) return v;
  final int uh = _u32(v.$h);
  if (s < 32) {
    final int low = _u32((v.$l >> s) | (v.$h << (32 - s)));
    return GoInt._(_s32(uh >> s), low);
  }
  if (s == 32) return GoInt._(0, uh);
  return GoInt._(0, uh >> (s - 32));
}

/// The decimal text of a 64-bit pattern that holds an unsigned value.
String $ustring(GoInt v) {
  if (v.$h >= 0) return v.toString();
  return _ubig(v).toString();
}

/// The value of an unsigned 64-bit pattern as a double.
/// The float64 value of a Go integer. The JS form holds a pair of halves,
/// thus the method of the class already gives the value with no virtual call.
double $todouble(GoInt v) => v.toDouble();

GoFloat32 $tof32(GoInt v) => $integerFloat32(_sbig(v));

GoFloat32 $utof32(GoInt v) => $integerFloat32(_ubig(v));

double $utodouble(GoInt v) => _u32(v.$h) * 4294967296.0 + v.$l.toDouble();

/// The IEEE 754 bit pattern of a 64-bit value.
GoInt $f64ToBits(GoFloat64 v) => GoInt._(_s32(v.high), v.low);

/// The 64-bit value of an IEEE 754 bit pattern.
GoFloat64 $f64FromBits(GoInt b) => GoFloat64.fromBits(_u32(b.$h), b.$l);

/// The IEEE 754 bit pattern of a 32-bit value.
GoInt $f32ToBits(GoFloat32 v) => GoInt._(0, v.bits);

/// The 32-bit value of an IEEE 754 bit pattern.
GoFloat32 $f32FromBits(GoInt b) => GoFloat32.fromBits(b.$l);

/// The plain Dart int of a Go integer, for an index, a length, or another
/// value that a Dart API takes. A value outside 53 bits loses its low bits.
int $ix(GoInt v) => v.$v;

/// The Go integer of a double, with the truncation of Go. A double outside
/// 53 bits is a multiple of a power of 2, thus the 2 halves stay exact.
GoInt $fromF64(double v) {
  // Go permits any integer result for NaN or infinity, but no exception.
  if (!v.isFinite) return GoInt.zero;
  final double t = v.truncateToDouble();
  final double low = t % 4294967296;
  return GoInt._(_s32(((t - low) / 4294967296).toInt()), low.toInt());
}

/// The value of a float64 as an unsigned 64-bit integer. The pair form already
/// holds the whole 64-bit pattern, thus the 2 conversions agree.
GoInt $fromF64U(double v) => $fromF64(v);

/// True inside the build of the JS target. The conditional import selects
/// this file per target, thus the value is a constant of the file.
const bool $isJS = true;

/// The text of the 64-bit pattern of a Go integer in a radix, read as an
/// unsigned value.
String $uradix(GoInt v, int radix) => _ubig(v).toRadixString(radix);

// The allocators of a Go integer slice. The pair form fits no dense list,
// thus each width gives a plain list.
List<GoInt> $goList8(int n) => List<GoInt>.filled(n, GoInt.zero);
List<GoInt> $goList16(int n) => List<GoInt>.filled(n, GoInt.zero);
List<GoInt> $goList32(int n) => List<GoInt>.filled(n, GoInt.zero);
List<GoInt> $goList64(int n) => List<GoInt>.filled(n, GoInt.zero);
List<GoInt> $goListU8(int n) => List<GoInt>.filled(n, GoInt.zero);
List<GoInt> $goListU16(int n) => List<GoInt>.filled(n, GoInt.zero);
List<GoInt> $goListU32(int n) => List<GoInt>.filled(n, GoInt.zero);
List<GoInt> $goListU64(int n) => List<GoInt>.filled(n, GoInt.zero);

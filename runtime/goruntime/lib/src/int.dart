import 'dart:collection';
import 'dart:typed_data';

final _float32Data = ByteData(4);

// Numeric lists in JavaScript can erase a NaN payload. Raw bits preserve it.
final class GoFloat32 {
  final int bits;
  const GoFloat32.fromBits(this.bits);
  static const zero = GoFloat32.fromBits(0);
  factory GoFloat32.fromDouble(double value) {
    _float32Data.setFloat32(0, value);
    return GoFloat32.fromBits(_float32Data.getUint32(0));
  }
  double toDouble() {
    _float32Data.setUint32(0, bits);
    return _float32Data.getFloat32(0);
  }

  bool get isNaN => (bits & 0x7fffffff) > 0x7f800000;
  GoFloat32 operator -() => GoFloat32.fromBits(bits ^ 0x80000000);
  GoFloat32 operator +(GoFloat32 b) =>
      GoFloat32.fromDouble(toDouble() + b.toDouble());
  GoFloat32 operator -(GoFloat32 b) =>
      GoFloat32.fromDouble(toDouble() - b.toDouble());
  GoFloat32 operator *(GoFloat32 b) =>
      GoFloat32.fromDouble(toDouble() * b.toDouble());
  GoFloat32 operator /(GoFloat32 b) =>
      GoFloat32.fromDouble(toDouble() / b.toDouble());
  bool operator <(GoFloat32 b) => toDouble() < b.toDouble();
  bool operator <=(GoFloat32 b) => toDouble() <= b.toDouble();
  bool operator >(GoFloat32 b) => toDouble() > b.toDouble();
  bool operator >=(GoFloat32 b) => toDouble() >= b.toDouble();
  @override
  bool operator ==(Object other) =>
      other is GoFloat32 && toDouble() == other.toDouble();
  @override
  int get hashCode => toDouble().hashCode;
  @override
  String toString() => toDouble().toString();
}

GoFloat32 $f32(double value) => GoFloat32.fromDouble(value);
GoFloat32 $float32Zero() => GoFloat32.zero;

final class $Float32List extends ListBase<GoFloat32> {
  final Uint32List _bits;
  $Float32List(int length) : _bits = Uint32List(length);
  $Float32List.fromList(List<GoFloat32> values)
    : _bits = Uint32List(values.length) {
    for (var i = 0; i < values.length; i++) {
      _bits[i] = values[i].bits;
    }
  }
  @override
  int get length => _bits.length;
  @override
  set length(int value) => throw UnsupportedError('fixed length');
  @override
  GoFloat32 operator [](int index) => GoFloat32.fromBits(_bits[index]);
  @override
  void operator []=(int index, GoFloat32 value) => _bits[index] = value.bits;
}

final _float64Data = ByteData(8);

// JavaScript numeric lists can change NaN bits. Values retain both halves.
final class GoFloat64 {
  final int high;
  final int low;
  const GoFloat64.fromBits(this.high, this.low);
  static const zero = GoFloat64.fromBits(0, 0);
  factory GoFloat64.fromDouble(double value) {
    _float64Data.setFloat64(0, value);
    return GoFloat64.fromBits(
      _float64Data.getUint32(0),
      _float64Data.getUint32(4),
    );
  }
  double toDouble() {
    _float64Data.setUint32(0, high);
    _float64Data.setUint32(4, low);
    return _float64Data.getFloat64(0);
  }

  bool get isNaN =>
      (high & 0x7fffffff) > 0x7ff00000 ||
      (high & 0x7fffffff) == 0x7ff00000 && low != 0;
  bool get isNegative => (high & 0x80000000) != 0;
  GoFloat64 operator -() => GoFloat64.fromBits(high ^ 0x80000000, low);
  GoFloat64 operator +(GoFloat64 b) => $f64(toDouble() + b.toDouble());
  GoFloat64 operator -(GoFloat64 b) => $f64(toDouble() - b.toDouble());
  GoFloat64 operator *(GoFloat64 b) => $f64(toDouble() * b.toDouble());
  GoFloat64 operator /(GoFloat64 b) => $f64(toDouble() / b.toDouble());
  bool operator <(GoFloat64 b) => toDouble() < b.toDouble();
  bool operator <=(GoFloat64 b) => toDouble() <= b.toDouble();
  bool operator >(GoFloat64 b) => toDouble() > b.toDouble();
  bool operator >=(GoFloat64 b) => toDouble() >= b.toDouble();
  GoFloat64 abs() => GoFloat64.fromBits(high & 0x7fffffff, low);
  GoFloat64 floorToDouble() => $f64(toDouble().floorToDouble());
  GoFloat64 ceilToDouble() => $f64(toDouble().ceilToDouble());
  GoFloat64 truncateToDouble() => $f64(toDouble().truncateToDouble());
  @override
  bool operator ==(Object other) =>
      other is GoFloat64 && toDouble() == other.toDouble();
  @override
  int get hashCode => toDouble().hashCode;
  @override
  String toString() => toDouble().toString();
}

GoFloat64 $f64(double value) => GoFloat64.fromDouble(value);
GoFloat64 $float64Zero() => GoFloat64.zero;

final class $Float64List extends ListBase<GoFloat64> {
  final Uint32List _bits;
  $Float64List(int length) : _bits = Uint32List(length * 2);
  $Float64List.fromList(List<GoFloat64> values)
    : _bits = Uint32List(values.length * 2) {
    for (var i = 0; i < values.length; i++) {
      this[i] = values[i];
    }
  }
  @override
  int get length => _bits.length ~/ 2;
  @override
  set length(int value) => throw UnsupportedError('fixed length');
  @override
  GoFloat64 operator [](int index) =>
      GoFloat64.fromBits(_bits[index * 2], _bits[index * 2 + 1]);
  @override
  void operator []=(int index, GoFloat64 value) {
    _bits[index * 2] = value.high;
    _bits[index * 2 + 1] = value.low;
  }
}

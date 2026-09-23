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

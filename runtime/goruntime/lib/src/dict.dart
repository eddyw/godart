import 'dart:typed_data';

import 'complex.dart';
import 'goint.dart';
import 'iface.dart';
import 'int.dart';
import 'slice.dart';
import 'string.dart';

/// The run time data of 1 type argument.
///
/// A Dart type parameter gives no zero value and no Go equality, thus a
/// generic function takes 1 dictionary for each type parameter.
final class Dict<T> {
  final T Function() zero;
  final bool Function(T, T) equal;
  final int Function(T) hash;
  final List<T> Function(int) alloc;

  /// order gives a negative value, 0, or a positive value. A generic function
  /// reads it when the constraint holds more than 1 Dart type, because no Dart
  /// operator then fits each type argument.
  ///
  /// The function stops the program for a type that Go holds unordered.
  final int Function(T, T) order;

  /// mask keeps a value inside the width of the type argument. A generic
  /// function reads it after each arithmetic operation, because the width
  /// differs per type argument. A type that needs no mask holds the identity.
  final T Function(T) mask;

  /// desc describes the type argument. A conversion of a value of the type
  /// parameter to an interface reads it.
  final TypeDesc desc;

  /// copy gives a fresh value. A struct and an array hold value semantics,
  /// thus a copy point needs a deep copy. Null stands for the identity.
  final T Function(T)? copy;

  /// assign writes a value into a location and gives the value to store. A
  /// struct writes its fields in place, thus a pointer to the location stays
  /// correct. Null stores the source value.
  final T Function(T, T)? assign;

  /// length, at, slice, bytes, and text serve a type set that holds a byte
  /// slice and a string. Such a set holds 2 Dart types, thus no Dart member
  /// fits both. Each other type set holds null.
  final GoInt Function(T)? length;
  final GoInt Function(T, GoInt)? at;
  final T Function(T, GoInt, GoInt)? slice;
  final Slice<GoInt> Function(T)? bytes;
  final GoString Function(T)? text;
  final T Function(Slice<GoInt>)? fromBytes;
  final T Function(GoString)? fromText;

  const Dict(
    this.zero,
    this.equal,
    this.hash,
    this.alloc,
    this.order,
    this.mask,
    this.desc, {
    this.copy,
    this.assign,
    this.length,
    this.at,
    this.slice,
    this.bytes,
    this.text,
    this.fromBytes,
    this.fromText,
  });

  bool get unsigned64 =>
      desc.kind == 'uint' || desc.kind == 'uint64' || desc.kind == 'uintptr';

  bool less(T a, T b) {
    if ((a is double && a.isNaN) || (b is double && b.isNaN)) return false;
    return order(a, b) < 0;
  }

  bool lessEqual(T a, T b) => equal(a, b) || less(a, b);

  T add(T a, T b) {
    if (a is GoString) return $concat(a, b as GoString) as T;
    if (a is double) return mask((a + (b as double)) as T);
    if (a is GoComplex) return mask((a + (b as GoComplex)) as T);
    return mask(((a as GoInt) + (b as GoInt)) as T);
  }

  T quotient(T a, T b) {
    if (a is double) return mask((a / (b as double)) as T);
    if (a is GoComplex) return mask((a / (b as GoComplex)) as T);
    final GoInt x = a as GoInt, y = b as GoInt;
    return mask((unsigned64 ? $udiv(x, y) : $div(x, y)) as T);
  }

  T remainder(T a, T b) => mask(
    (unsigned64 ? $umod(a as GoInt, b as GoInt) : $mod(a as GoInt, b as GoInt))
        as T,
  );

  T shiftRight(T a, GoInt count) => mask(
    (unsigned64 ? $ushr(a as GoInt, count) : $shr(a as GoInt, count)) as T,
  );

  T copyOf(T v) => copy == null ? v : copy!(v);

  T assignOf(T dst, T src) => assign == null ? src : assign!(dst, src);
}

/// $identity gives the value again. A type that needs no mask reads it.
T $identity<T>(T v) => v;

/// $unordered stops the program. Go rejects an order operation on the type,
/// thus no path reaches this function.
int $unordered<T>(T a, T b) {
  throw StateError('the type holds no order');
}

GoInt _zeroInt() => GoInt.zero;
bool _equalInt(GoInt a, GoInt b) => a == b;
int _hashInt(GoInt v) => v.hashCode;
List<GoInt> _allocInt(int n) => List<GoInt>.filled(n, GoInt.zero);
int _orderInt(GoInt a, GoInt b) => a < b ? -1 : (a == b ? 0 : 1);
int _orderUint(GoInt a, GoInt b) => $ult(a, b) ? -1 : (a == b ? 0 : 1);

double _zeroDouble() => 0.0;
bool _equalDouble(double a, double b) => a == b;
int _hashDouble(double v) => v.hashCode;
List<double> _allocDouble(int n) => Float64List(n);
int _orderDouble(double a, double b) => a < b ? -1 : (a == b ? 0 : 1);

bool _zeroBool() => false;
bool _equalBool(bool a, bool b) => a == b;
int _hashBool(bool v) => v.hashCode;
List<bool> _allocBool(int n) => List<bool>.filled(n, false);

GoComplex _zeroComplex() => GoComplex.zero;
bool _equalComplex(GoComplex a, GoComplex b) => a == b;
int _hashComplex(GoComplex v) => v.hashCode;
List<GoComplex> _allocComplex(int n) =>
    List<GoComplex>.filled(n, GoComplex.zero);

GoString _zeroString() => GoString.empty;
bool _equalString(GoString a, GoString b) => a == b;
int _hashString(GoString v) => v.hashCode;
List<GoString> _allocString(int n) => List<GoString>.filled(n, GoString.empty);

/// A dictionary of an integer type. Each one holds the same operations, and
/// the description and the order tell them apart. An unsigned 64-bit type
/// reads the bit pattern as unsigned, thus its order differs.
Dict<GoInt> _signed(TypeDesc desc, GoInt Function(GoInt) mask) => Dict<GoInt>(
  _zeroInt,
  _equalInt,
  _hashInt,
  _allocInt,
  _orderInt,
  mask,
  desc,
);

Dict<GoInt> _unsigned(TypeDesc desc) => Dict<GoInt>(
  _zeroInt,
  _equalInt,
  _hashInt,
  _allocInt,
  _orderUint,
  $identity<GoInt>,
  desc,
);

final Dict<GoInt> $dictInt = _signed($typeInt, $identity<GoInt>);
final Dict<GoInt> $dictInt8 = _signed($typeInt8, $i8);
final Dict<GoInt> $dictInt16 = _signed($typeInt16, $i16);
final Dict<GoInt> $dictInt32 = _signed($typeInt32, $i32);
final Dict<GoInt> $dictInt64 = _signed($typeInt64, $identity<GoInt>);
final Dict<GoInt> $dictUint8 = _signed($typeUint8, $u8);
final Dict<GoInt> $dictUint16 = _signed($typeUint16, $u16);
final Dict<GoInt> $dictUint32 = _signed($typeUint32, $u32);
final Dict<GoInt> $dictUint = _unsigned($typeUint);
final Dict<GoInt> $dictUint64 = _unsigned($typeUint64);
final Dict<GoInt> $dictUintptr = _unsigned($typeUintptr);

const Dict<double> $dictFloat64 = Dict<double>(
  _zeroDouble,
  _equalDouble,
  _hashDouble,
  _allocDouble,
  _orderDouble,
  $identity<double>,
  $typeFloat64,
);

const Dict<double> $dictFloat32 = Dict<double>(
  _zeroDouble,
  _equalDouble,
  _hashDouble,
  _allocDouble,
  _orderDouble,
  $f32,
  $typeFloat32,
);

const Dict<bool> $dictBool = Dict<bool>(
  _zeroBool,
  _equalBool,
  _hashBool,
  _allocBool,
  $unordered<bool>,
  $identity<bool>,
  $typeBool,
);

const Dict<GoComplex> $dictComplex64 = Dict<GoComplex>(
  _zeroComplex,
  _equalComplex,
  _hashComplex,
  _allocComplex,
  $unordered<GoComplex>,
  $c64,
  $typeComplex64,
);

const Dict<GoComplex> $dictComplex128 = Dict<GoComplex>(
  _zeroComplex,
  _equalComplex,
  _hashComplex,
  _allocComplex,
  $unordered<GoComplex>,
  $identity<GoComplex>,
  $typeComplex128,
);

const Dict<GoString> $dictString = Dict<GoString>(
  _zeroString,
  _equalString,
  _hashString,
  _allocString,
  $compare,
  $identity<GoString>,
  $typeString,
  length: _lengthString,
  at: _atString,
  slice: _sliceString,
  bytes: $stringToBytes,
  text: $identity<GoString>,
  fromBytes: $bytesToString,
  fromText: $identity<GoString>,
);

GoInt _lengthString(GoString v) => $gi(v.length);
GoInt _atString(GoString v, GoInt i) => v.$at(i);
GoString _sliceString(GoString v, GoInt a, GoInt b) =>
    $substring(v, $ix(a), $ix(b));

import 'dart:typed_data';

import 'iface.dart';
import 'int.dart';
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

  /// desc describes the type argument. A conversion of a value of the type
  /// parameter to an interface reads it.
  final TypeDesc desc;

  const Dict(
    this.zero,
    this.equal,
    this.hash,
    this.alloc,
    this.order,
    this.desc,
  );
}

/// $unordered stops the program. Go rejects an order operation on the type,
/// thus no path reaches this function.
int $unordered<T>(T a, T b) {
  throw StateError('the type holds no order');
}

int _zeroInt() => 0;
bool _equalInt(int a, int b) => a == b;
int _hashInt(int v) => v.hashCode;
List<int> _allocInt(int n) => Int64List(n);
int _orderInt(int a, int b) => a < b ? -1 : (a == b ? 0 : 1);
int _orderUint(int a, int b) => $ult(a, b) ? -1 : (a == b ? 0 : 1);

double _zeroDouble() => 0.0;
bool _equalDouble(double a, double b) => a == b;
int _hashDouble(double v) => v.hashCode;
List<double> _allocDouble(int n) => Float64List(n);
int _orderDouble(double a, double b) => a < b ? -1 : (a == b ? 0 : 1);

bool _zeroBool() => false;
bool _equalBool(bool a, bool b) => a == b;
int _hashBool(bool v) => v.hashCode;
List<bool> _allocBool(int n) => List<bool>.filled(n, false);

GoString _zeroString() => GoString.empty;
bool _equalString(GoString a, GoString b) => a == b;
int _hashString(GoString v) => v.hashCode;
List<GoString> _allocString(int n) => List<GoString>.filled(n, GoString.empty);

/// A dictionary of an integer type. Each one holds the same operations, and
/// the description and the order tell them apart. An unsigned 64-bit type
/// reads the bit pattern as unsigned, thus its order differs.
Dict<int> _signed(TypeDesc desc) =>
    Dict<int>(_zeroInt, _equalInt, _hashInt, _allocInt, _orderInt, desc);

Dict<int> _unsigned(TypeDesc desc) =>
    Dict<int>(_zeroInt, _equalInt, _hashInt, _allocInt, _orderUint, desc);

final Dict<int> $dictInt = _signed($typeInt);
final Dict<int> $dictInt8 = _signed($typeInt8);
final Dict<int> $dictInt16 = _signed($typeInt16);
final Dict<int> $dictInt32 = _signed($typeInt32);
final Dict<int> $dictInt64 = _signed($typeInt64);
final Dict<int> $dictUint8 = _signed($typeUint8);
final Dict<int> $dictUint16 = _signed($typeUint16);
final Dict<int> $dictUint32 = _signed($typeUint32);
final Dict<int> $dictUint = _unsigned($typeUint);
final Dict<int> $dictUint64 = _unsigned($typeUint64);
final Dict<int> $dictUintptr = _unsigned($typeUintptr);

const Dict<double> $dictFloat64 = Dict<double>(
  _zeroDouble,
  _equalDouble,
  _hashDouble,
  _allocDouble,
  _orderDouble,
  $typeFloat64,
);

const Dict<double> $dictFloat32 = Dict<double>(
  _zeroDouble,
  _equalDouble,
  _hashDouble,
  _allocDouble,
  _orderDouble,
  $typeFloat32,
);

const Dict<bool> $dictBool = Dict<bool>(
  _zeroBool,
  _equalBool,
  _hashBool,
  _allocBool,
  $unordered<bool>,
  $typeBool,
);

const Dict<GoString> $dictString = Dict<GoString>(
  _zeroString,
  _equalString,
  _hashString,
  _allocString,
  $compare,
  $typeString,
);

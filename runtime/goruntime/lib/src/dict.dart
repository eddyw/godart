import 'dart:typed_data';

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

  const Dict(this.zero, this.equal, this.hash, this.alloc, this.order);
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

const Dict<int> $dictInt = Dict<int>(
  _zeroInt,
  _equalInt,
  _hashInt,
  _allocInt,
  _orderInt,
);

/// $dictUint holds an unsigned 64-bit type. Such a type reads the bit pattern
/// as unsigned, thus its order differs from the order of $dictInt.
const Dict<int> $dictUint = Dict<int>(
  _zeroInt,
  _equalInt,
  _hashInt,
  _allocInt,
  _orderUint,
);

const Dict<double> $dictDouble = Dict<double>(
  _zeroDouble,
  _equalDouble,
  _hashDouble,
  _allocDouble,
  _orderDouble,
);

const Dict<bool> $dictBool = Dict<bool>(
  _zeroBool,
  _equalBool,
  _hashBool,
  _allocBool,
  $unordered<bool>,
);

const Dict<GoString> $dictString = Dict<GoString>(
  _zeroString,
  _equalString,
  _hashString,
  _allocString,
  $compare,
);

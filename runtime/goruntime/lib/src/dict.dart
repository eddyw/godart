import 'dart:typed_data';

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

  const Dict(this.zero, this.equal, this.hash, this.alloc);
}

int _zeroInt() => 0;
bool _equalInt(int a, int b) => a == b;
int _hashInt(int v) => v.hashCode;
List<int> _allocInt(int n) => Int64List(n);

double _zeroDouble() => 0.0;
bool _equalDouble(double a, double b) => a == b;
int _hashDouble(double v) => v.hashCode;
List<double> _allocDouble(int n) => Float64List(n);

bool _zeroBool() => false;
bool _equalBool(bool a, bool b) => a == b;
int _hashBool(bool v) => v.hashCode;
List<bool> _allocBool(int n) => List<bool>.filled(n, false);

GoString _zeroString() => GoString.empty;
bool _equalString(GoString a, GoString b) => a == b;
int _hashString(GoString v) => v.hashCode;
List<GoString> _allocString(int n) => List<GoString>.filled(n, GoString.empty);

const Dict<int> $dictInt = Dict<int>(_zeroInt, _equalInt, _hashInt, _allocInt);

const Dict<double> $dictDouble = Dict<double>(
  _zeroDouble,
  _equalDouble,
  _hashDouble,
  _allocDouble,
);

const Dict<bool> $dictBool = Dict<bool>(
  _zeroBool,
  _equalBool,
  _hashBool,
  _allocBool,
);

const Dict<GoString> $dictString = Dict<GoString>(
  _zeroString,
  _equalString,
  _hashString,
  _allocString,
);

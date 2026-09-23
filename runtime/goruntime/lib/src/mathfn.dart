import 'dart:math' as math;

import 'int.dart';

/// The square root of a value, with the rounding of IEEE 754.
///
/// The portable Go form builds the result bit by bit in a loop of 54 steps,
/// and the compiler of Go replaces it with 1 machine instruction. dart:math
/// gives that instruction, and the 2 forms agree on each value: a negative
/// input gives a NaN, a negative zero gives a negative zero, and an infinity
/// passes through.
GoFloat64 $sqrt(GoFloat64 v) => $f64(math.sqrt(v.toDouble()));

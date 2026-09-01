import 'dart:typed_data';

// The integer helpers live in goint_native.dart and goint_js.dart, and the
// conditional export of goint.dart selects 1 form. This file keeps the
// float helper only.

final _f32 = Float32List(1);

/// Rounds a double to 32-bit precision, as float32 does.
double $f32(double v) {
  _f32[0] = v;
  return _f32[0];
}

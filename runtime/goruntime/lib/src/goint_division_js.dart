import 'dart:js_interop';

@JS('BigInt')
external JSBigInt _bigInt(int value);

@JS('BigInt.asUintN')
external JSBigInt _unsigned(int width, JSBigInt value);

@JS('Number')
external double _number(JSAny value);

final JSBigInt _base = _bigInt(4294967296);

JSAny _value(int high, int low) =>
    _bigInt(high).multiply(_base).add(_bigInt(low));

(int, int) _parts(JSAny value) {
  final bits = _unsigned(64, value as JSBigInt);
  return (
    _number(bits.divide(_base)).toInt(),
    _number(_unsigned(32, bits)).toInt(),
  );
}

// Nonnegative 32-bit operands fit exactly in a JS number.
(int, int) divide(int ah, int al, int bh, int bl) {
  if (ah == 0 && bh == 0) return (0, al ~/ bl);
  return _parts(_value(ah, al).divide(_value(bh, bl)));
}

(int, int) remainder(int ah, int al, int bh, int bl) {
  if (ah == 0 && bh == 0) return (0, al % bl);
  return _parts(_value(ah, al).modulo(_value(bh, bl)));
}

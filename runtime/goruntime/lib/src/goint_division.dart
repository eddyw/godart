final BigInt _base = BigInt.from(4294967296);
final BigInt _mask = BigInt.from(0xFFFFFFFF);

BigInt _value(int high, int low) =>
    BigInt.from(high) * _base + BigInt.from(low);

(int, int) _parts(BigInt value) {
  final bits = value.toUnsigned(64);
  return ((bits >> 32).toInt(), (bits & _mask).toInt());
}

(int, int) divide(int ah, int al, int bh, int bl) =>
    _parts(_value(ah, al) ~/ _value(bh, bl));

(int, int) remainder(int ah, int al, int bh, int bl) =>
    _parts(_value(ah, al).remainder(_value(bh, bl)));

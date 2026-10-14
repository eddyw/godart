@TestOn('node')
library;

import 'package:goruntime/src/goint_js.dart' as js;
import 'package:goruntime/src/panic.dart';
import 'package:test/test.dart';

js.GoInt pair(BigInt value) =>
    js.$gi2((value >> 32).toSigned(32).toInt(), value.toUnsigned(32).toInt());

BigInt signed(js.GoInt value) =>
    (BigInt.from(value.$h) << 32) + BigInt.from(value.$l);

void main() {
  final values = [
    '0',
    '1',
    '2',
    '3',
    '44487',
    '44488',
    '44489',
    '48271',
    '65535',
    '65536',
    '2147483646',
    '2147483647',
    '2147483648',
    '4294967294',
    '4294967295',
    '4294967296',
    '-1',
    '-2',
    '-2147483648',
    '-4294967295',
    '-4294967296',
    '-9223372036854775808',
    '9223372036854775807',
    '-6148914691236517206',
    '6364136223846793005',
  ].map(BigInt.parse).toList();

  test('JS division and remainder preserve signed and unsigned boundaries', () {
    for (final a in values) {
      for (final b in values) {
        if (b == BigInt.zero) continue;
        final x = pair(a);
        final y = pair(b);
        expect(signed(js.$div(x, y)), (a ~/ b).toSigned(64), reason: '$a / $b');
        expect(signed(js.$mod(x, y)), a.remainder(b), reason: '$a % $b');
        final unsignedA = a.toUnsigned(64);
        final unsignedB = b.toUnsigned(64);
        expect(
          signed(js.$udiv(x, y)).toUnsigned(64),
          unsignedA ~/ unsignedB,
          reason: 'unsigned $a / $b',
        );
        expect(
          signed(js.$umod(x, y)).toUnsigned(64),
          unsignedA.remainder(unsignedB),
          reason: 'unsigned $a % $b',
        );
      }
    }
  });

  test('all division forms preserve the Go zero-divisor panic', () {
    for (final operation in [js.$div, js.$mod, js.$udiv, js.$umod]) {
      for (final a in values) {
        expect(
          () => operation(pair(a), js.GoInt.zero),
          throwsA(isA<GoPanic>()),
        );
      }
    }
  });
}

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

final _values = [
  '0',
  '1',
  'ffffffff',
  '100000000',
  '7fffffffffffffff',
  '8000000000000000',
  'ffffffffffffffff',
  'aaaaaaaaaaaaaaaa',
].map((value) => BigInt.parse(value, radix: 16)).toList();

GoInt _word(BigInt value) =>
    $gi2((value >> 32).toSigned(32).toInt(), value.toUnsigned(32).toInt());

void main() {
  for (final accumulate in [false, true]) {
    test('word vectors preserve exact carries and sequential aliases '
        'accumulate=$accumulate', () {
      for (final multiplier in _values) {
        for (final initial in _values) {
          for (final layout in [0, 1, 2, 3, 4]) {
            for (final length in [0, 1, 7]) {
              final original = List<BigInt>.generate(
                24,
                (i) => _values[(i * 3 + layout) % _values.length],
              );
              final expected = List<BigInt>.of(original);
              final words = original.map(_word).toList();
              const z = 1;
              final x = layout == 0 ? 9 : (layout == 4 ? 0 : z);
              final y = layout < 2 ? 17 : (layout == 3 ? 0 : z);
              var carry = initial;
              for (var i = 0; i < length; i++) {
                final total = accumulate
                    ? expected[y + i] * multiplier + expected[x + i] + carry
                    : expected[x + i] * multiplier + carry;
                expected[z + i] = total.toUnsigned(64);
                carry = (total >> 64).toUnsigned(64);
              }
              final result = accumulate
                  ? $bigAddMulVVWW(
                      $indexPtr(words, $gi(z)),
                      $indexPtr(words, $gi(x)),
                      $indexPtr(words, $gi(y)),
                      length,
                      _word(multiplier),
                      _word(initial),
                    )
                  : $bigMulAddVWW(
                      $indexPtr(words, $gi(z)),
                      $indexPtr(words, $gi(x)),
                      length,
                      _word(multiplier),
                      _word(initial),
                    );
              expect(result, _word(carry));
              expect(words, expected.map(_word).toList());
            }
          }
        }
      }
    });
  }
}

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

Slice<int> makeInts(int len, int cap) =>
    $makeSlice<int>($gi(len), $gi(cap), (n) => List<int>.filled(n, 0));

void main() {
  test('make gives the length and the capacity', () {
    final s = makeInts(2, 4);
    expect(s.len, 2);
    expect(s.cap, 4);
    expect(s[0], 0);
  });

  test('an index out of range panics', () {
    final s = makeInts(2, 4);
    expect(() => s[2], throwsA(isA<GoPanic>()));
  });

  test('append keeps the array when the capacity is enough', () {
    final s = makeInts(1, 4);
    final t = $append(s, <int>[7], (n) => List<int>.filled(n, 0));
    expect(t.len, 2);
    expect(identical(t.array, s.array), true);
    expect(t[1], 7);
  });

  test('append allocates when the capacity is full', () {
    final s = makeInts(2, 2);
    final t = $append(s, <int>[7], (n) => List<int>.filled(n, 0));
    expect(t.len, 3);
    expect(t.cap >= 3, true);
    expect(identical(t.array, s.array), false);
  });

  test('a range shares the backing array', () {
    final s = makeInts(4, 4);
    final t = s.range($gi(1), $gi(3), $gi(4));
    t[0] = 9;
    expect(s[1], 9);
    expect(t.len, 2);
    expect(t.cap, 3);
  });

  test('copy returns the number of elements', () {
    final dst = makeInts(2, 2);
    final src = $append(makeInts(0, 3), <int>[
      1,
      2,
      3,
    ], (n) => List<int>.filled(n, 0));
    expect($copySlice(dst, src), 2);
    expect(dst[1], 2);
  });

  test('the zero value is nil', () {
    expect(Slice<int>.nil().isNil, true);
    expect(Slice<int>.nil().len, 0);
  });
}

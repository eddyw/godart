import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

Slice<int> makeInts(int len, int cap) =>
    $makeSlice<int>($gi(len), $gi(cap), (n) => List<int>.filled(n, 0));

void main() {
  test('integer views preserve shared writes and index panics', () {
    final source = Slice<GoInt>.of([$gi(1), $gi(2), $gi(3)]);
    final view = source.range($gi(1), $gi(3), $gi(3));
    view.$set($gi(0), $gi(7));
    expect(source.$at($gi(1)), $gi(7));
    expect(view.$at($gi(1)), $gi(3));
    expect(() => view.$at($gi(-1)), throwsA(isA<GoPanic>()));
    expect(() => view.$set($gi(2), $gi(9)), throwsA(isA<GoPanic>()));
    expect(source.$at($gi(2)), $gi(3));
  });

  test('integer append and overlapping copy preserve storage', () {
    final source = $makeSlice<GoInt>(
      $gi(3),
      $gi(4),
      (n) => List<GoInt>.filled(n, GoInt.zero),
    );
    for (var i = 0; i < 3; i++) {
      source.$set($gi(i), $gi(i + 1));
    }
    final grown = $append(source, [
      $gi(4),
    ], (n) => List<GoInt>.filled(n, GoInt.zero));
    expect(identical(source.array, grown.array), isTrue);
    final destination = grown.range($gi(1), $gi(4), $gi(4));
    expect($copySlice(destination, source), $gi(3));
    expect(
      [for (var i = 0; i < 4; i++) grown.$at($gi(i))],
      [$gi(1), $gi(1), $gi(2), $gi(3)],
    );
  });

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

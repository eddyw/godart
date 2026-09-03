import 'goint.dart';

import 'iface.dart';
import 'panic.dart';

/// A Go slice: a view on a backing array with a length and a capacity.
///
/// The header is a value, thus each field is final. An operation that changes
/// the header, for example append, returns a new header.
final class Slice<T> {
  final List<T> array;
  final int off;
  final int len;
  final int cap;

  /// True for the zero value of a slice type. Go compares such a slice with
  /// nil, but it allows len, cap, and append on it.
  final bool isNil;

  const Slice(this.array, this.off, this.len, this.cap) : isNil = false;

  const Slice._(this.array, this.off, this.len, this.cap, this.isNil);

  factory Slice.nil() => Slice._(<T>[], 0, 0, 0, true);

  /// Wraps a full backing array.
  factory Slice.of(List<T> array) =>
      Slice(array, 0, array.length, array.length);

  T operator [](int i) => $at(i);

  void operator []=(int i, T value) => $set(i, value);

  /// The element at index i. uns names an index of an unsigned type, which
  /// reads as a negative Dart int above 2^63.
  T $at(int i, {bool uns = false}) {
    if (i < 0 || i >= len) $panicIndex(i, len, uns);
    return array[off + i];
  }

  void $set(int i, T value, {bool uns = false}) {
    if (i < 0 || i >= len) $panicIndex(i, len, uns);
    array[off + i] = value;
  }

  /// The Go expression `s[low:high]`, and `s[low:high:max]` when three is
  /// true. byLen names the limit "length" in a report, which an array and a
  /// pointer to an array need.
  ///
  /// The order of the tests is the order of Go, thus a bound that breaks 2
  /// limits names the same 1 that Go names.
  Slice<T> range(
    int low,
    int high,
    int max, {
    bool three = false,
    bool byLen = false,
    bool lowUns = false,
    bool highUns = false,
    bool maxUns = false,
  }) {
    if (three) {
      if (max < 0 || max > cap) {
        $panicBound(
          byLen ? SliceBound.threeALen : SliceBound.threeACap,
          max,
          cap,
          maxUns,
        );
      }
      if (high < 0 || high > max) {
        $panicBound(SliceBound.threeB, high, max, highUns);
      }
      if (low < 0 || low > high) {
        $panicBound(SliceBound.threeC, low, high, lowUns);
      }
    } else {
      if (high < 0 || high > cap) {
        $panicBound(
          byLen ? SliceBound.aLen : SliceBound.aCap,
          high,
          cap,
          highUns,
        );
      }
      if (low < 0 || low > high) $panicBound(SliceBound.b, low, high, lowUns);
    }
    return Slice(array, off + low, high - low, max - low);
  }
}

/// The Go expression `make([]T, len, cap)`. The allocator returns a list of
/// the given length that holds a fresh zero value in each slot.
Slice<T> $makeSlice<T>(int len, int cap, List<T> Function(int) alloc) {
  if (len < 0) {
    $panic(Iface($typeGoError, const GoError('makeslice: len out of range')));
  }
  if (cap < len) {
    $panic(Iface($typeGoError, const GoError('makeslice: cap out of range')));
  }
  return Slice(alloc(cap), 0, len, cap);
}

/// The Go builtin `make` of a slice type with no capacity. The length comes
/// from 1 read of its expression, because the expression can hold a side
/// effect.
Slice<T> $makeSliceN<T>(int len, List<T> Function(int) alloc) =>
    $makeSlice(len, len, alloc);

/// The Go builtin `append`. The result shares the backing array when the
/// capacity is enough, exactly as in Go.
///
/// The assign function writes 1 element. A struct element needs a field copy on
/// the path that grows, because Go copies each element into the new array. A
/// reference write would give the 2 slices 1 object, and a pointer into the old
/// array would then follow a write to the new 1.
Slice<T> $append<T>(
  Slice<T> s,
  List<T> add,
  List<T> Function(int) alloc, [
  void Function(T, T)? assign,
]) {
  if (add.isEmpty) return s;
  final int need = s.len + add.length;
  if (need <= s.cap) {
    for (var i = 0; i < add.length; i++) {
      s.array[s.off + s.len + i] = add[i];
    }
    return Slice(s.array, s.off, need, s.cap);
  }
  int cap = s.cap == 0 ? need : s.cap;
  while (cap < need) {
    cap += cap < 256 ? cap : cap ~/ 4;
  }
  final List<T> array = alloc(cap);
  if (assign != null) {
    for (var i = 0; i < s.len; i++) {
      assign(array[i], s.array[s.off + i]);
    }
  } else {
    for (var i = 0; i < s.len; i++) {
      array[i] = s.array[s.off + i];
    }
  }
  for (var i = 0; i < add.length; i++) {
    array[s.len + i] = add[i];
  }
  return Slice(array, 0, need, cap);
}

/// Exchanges 2 elements of a slice. The package sort uses it, because a Go
/// sort works on a slice of an unknown element type.
void $swap(Object? value, int i, int j) {
  final Object? unwrapped = value is Iface ? value.value : value;
  if (unwrapped is! Slice<Object?>) {
    $panic(
      Iface($typeGoError, const GoError('sort: the value is not a slice')),
    );
  }
  final Slice<Object?> slice = unwrapped;
  if (i < 0 || i >= slice.len) $panicIndex(i, slice.len);
  if (j < 0 || j >= slice.len) $panicIndex(j, slice.len);
  final Object? a = slice.array[slice.off + i];
  slice.array[slice.off + i] = slice.array[slice.off + j];
  slice.array[slice.off + j] = a;
}

/// The length of a slice of an unknown element type.
int $sliceLen(Object? value) {
  final Object? unwrapped = value is Iface ? value.value : value;
  if (unwrapped is! Slice<Object?>) {
    $panic(
      Iface($typeGoError, const GoError('sort: the value is not a slice')),
    );
  }
  return unwrapped.len;
}

/// The Go builtin `copy`. The assign function writes 1 element. A struct
/// element needs a field copy, and not a reference write.
int $copySlice<T>(Slice<T> dst, Slice<T> src, [void Function(T, T)? assign]) {
  final int n = dst.len < src.len ? dst.len : src.len;
  if (assign != null) {
    for (var i = 0; i < n; i++) {
      assign(dst.array[dst.off + i], src.array[src.off + i]);
    }
    return n;
  }
  if (identical(dst.array, src.array) && dst.off > src.off) {
    for (var i = n - 1; i >= 0; i--) {
      dst.array[dst.off + i] = src.array[src.off + i];
    }
    return n;
  }
  for (var i = 0; i < n; i++) {
    dst.array[dst.off + i] = src.array[src.off + i];
  }
  return n;
}

/// The Go expression `append(s, other...)`.
Slice<T> $appendSlice<T>(
  Slice<T> s,
  Slice<T> other,
  List<T> Function(int) alloc, [
  void Function(T, T)? assign,
]) {
  final add = <T>[];
  if (assign != null && other.len > 0) {
    // Go copies each element of the source. A plain append copies its argument
    // at the call site, and a spread has no such point, thus the copy stands
    // here.
    final List<T> fresh = alloc(other.len);
    for (var i = 0; i < other.len; i++) {
      assign(fresh[i], other.array[other.off + i]);
      add.add(fresh[i]);
    }
    return $append(s, add, alloc, assign);
  }
  for (var i = 0; i < other.len; i++) {
    add.add(other.array[other.off + i]);
  }
  return $append(s, add, alloc, assign);
}

/// Writes the zero value into each element of a slice. The Go builtin clear
/// does that for a slice.
void $clearSlice<T>(Slice<T> s, T Function() zero) {
  for (var i = 0; i < s.len; i++) {
    s[i] = zero();
  }
}

/// The Go byte slice of a plain int slice of the runtime. The native form
/// shares the storage, because the element type erases there, and the JS
/// form copies each element into a pair.
Slice<GoInt> $gils(Slice<int> v) {
  if ($isJS) {
    final List<GoInt> out = <GoInt>[
      for (var i = 0; i < v.len; i++) $gi(v.array[v.off + i]),
    ];
    return Slice.of(out);
  }
  return Slice<GoInt>(v.array as List<GoInt>, v.off, v.len, v.cap);
}

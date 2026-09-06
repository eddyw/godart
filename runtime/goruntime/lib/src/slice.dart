import 'dart:typed_data';

import 'goint.dart';

import 'iface.dart';
import 'panic.dart';

/// A Go slice: a view on a backing array with a length and a capacity.
///
/// The header is a value, thus each field is final. An operation that changes
/// the header, for example append, returns a new header.
///
/// The real class of `array` varies: Uint8List, Int32List, Float64List, a
/// growable list, and more. Dart thus cannot resolve `array[i]` to 1 target,
/// and each read of a byte takes a megamorphic dispatch. `$ByteSlice` holds
/// the storage of a byte slice in a field of the type Uint8List, thus a call
/// site of `[]` sees 2 targets at most and each read reaches 1 list class.
final class Slice<T> {
  final List<T> array;
  final int off;
  final int len;
  final int cap;

  /// True for the zero value of a slice type. Go compares such a slice with
  /// nil, but it allows len, cap, and append on it.
  final bool isNil;

  /// Builds a header. A backing array of bytes gives a `$ByteSlice`.
  ///
  /// The JS target holds a pair for each Go integer, thus no byte slice holds
  /// a Uint8List there and the constant `$isJS` removes the test.
  factory Slice(List<T> array, int off, int len, int cap) {
    if (!$isJS && array is Uint8List) {
      return $ByteSlice(array as Uint8List, off, len, cap) as Slice<T>;
    }
    return Slice<T>._(array, off, len, cap, false);
  }

  const Slice._(this.array, this.off, this.len, this.cap, this.isNil);

  /// The zero value of a slice type. The backing array is 1 shared empty
  /// list, because a nil slice holds a length of 0 and a capacity of 0, thus
  /// no write ever reaches it. A List of Never fits each List of T.
  factory Slice.nil() => Slice<T>._(const <Never>[], 0, 0, 0, true);

  /// Wraps a full backing array.
  factory Slice.of(List<T> array) =>
      Slice<T>(array, 0, array.length, array.length);

  // The 2 operators hold the body of $at and of $set. A call of a helper
  // would stand in each read of a byte, and a scanner reads 2 megabytes.
  T operator [](int i) {
    if (i < 0 || i >= len) $panicIndex(i, len);
    return array[off + i];
  }

  void operator []=(int i, T value) {
    if (i < 0 || i >= len) $panicIndex(i, len);
    array[off + i] = value;
  }

  /// The element at index i. uns names an index of an unsigned type, which
  /// reads as a negative Dart int above 2^63.
  T $at(int i, [bool uns = false]) {
    if (i < 0 || i >= len) $panicIndex(i, len, uns);
    return array[off + i];
  }

  void $set(int i, T value, [bool uns = false]) {
    if (i < 0 || i >= len) $panicIndex(i, len, uns);
    array[off + i] = value;
  }

  /// Adds n elements of src at the end, inside the capacity, and gives the
  /// new header. The class of the result is the class of this header.
  ///
  /// 1 call holds the move and the header, thus the hot path of append takes
  /// 1 virtual call and no test of the storage.
  Slice<T> $grow(List<T> src, int srcOff, int n) {
    final int need = len + n;
    // `append(b, x)` adds 1 element, and that shape holds most of the calls.
    // A move of a range reads an iterator of the source, thus the direct
    // write costs less.
    if (n == 1) {
      array[off + len] = src[srcOff];
    } else {
      array.setRange(off + len, off + need, src, srcOff);
    }
    return Slice<T>._(array, off, need, cap, false);
  }

  /// Writes 1 element at an absolute index of the backing array. The caller
  /// holds the bounds check.
  void $put(int index, T value) {
    array[index] = value;
  }

  /// Moves a range of src into the backing array, at an absolute index.
  ///
  /// The 2 lists can share 1 buffer. setRange holds the result of Go for such
  /// a move, exactly as memmove does.
  void $fill(int start, int end, List<T> src, int srcOff) {
    array.setRange(start, end, src, srcOff);
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
    int max, [
    bool three = false,
    bool byLen = false,
    bool lowUns = false,
    bool highUns = false,
    bool maxUns = false,
  ]) {
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
    // Go gives a nil slice for each valid slice expression of a nil slice.
    // Such a slice holds a capacity of 0, thus each bound is 0 here.
    if (isNil) return this;
    return $view(off + low, high - low, max - low);
  }

  /// A header on the same storage, with an absolute offset. The class of the
  /// result is the class of this header.
  Slice<T> $view(int start, int n, int room) =>
      Slice<T>._(array, start, n, room, false);
}

/// A Go slice with a backing array of bytes. `[]byte` is the common case,
/// because strconv, strings.Builder, fmt, and encoding/json each write into 1.
///
/// The field `bytes` holds the same list as `array`, with the exact class.
/// Dart thus resolves each read and each write of this class to 1 target, and
/// no megamorphic dispatch stands in a byte loop.
///
/// The element type is `GoInt`, which erases to `int` at run time. The 2
/// casts of the class thus hold no test after the erasure, and a value of
/// `Slice<int>` accepts this class too.
final class $ByteSlice extends Slice<GoInt> {
  final Uint8List bytes;

  $ByteSlice(this.bytes, int off, int len, int cap)
    : super._(bytes as List<GoInt>, off, len, cap, false);

  @override
  GoInt operator [](int i) {
    if (i < 0 || i >= len) $panicIndex(i, len);
    return $gi(bytes[off + i]);
  }

  @override
  void operator []=(int i, GoInt value) {
    if (i < 0 || i >= len) $panicIndex(i, len);
    bytes[off + i] = $ix(value);
  }

  @override
  GoInt $at(int i, [bool uns = false]) {
    if (i < 0 || i >= len) $panicIndex(i, len, uns);
    return $gi(bytes[off + i]);
  }

  @override
  void $set(int i, GoInt value, [bool uns = false]) {
    if (i < 0 || i >= len) $panicIndex(i, len, uns);
    bytes[off + i] = $ix(value);
  }

  @override
  Slice<GoInt> $grow(List<GoInt> src, int srcOff, int n) {
    final int need = len + n;
    if (n == 1) {
      bytes[off + len] = $ix(src[srcOff]);
    } else {
      // A GoInt erases to an int on the native target, thus the cast holds no
      // test there. The JS target builds no $ByteSlice, thus it never runs.
      bytes.setRange(off + len, off + need, src as List<int>, srcOff);
    }
    return $ByteSlice(bytes, off, need, cap);
  }

  @override
  Slice<GoInt> $view(int start, int n, int room) =>
      $ByteSlice(bytes, start, n, room);

  @override
  void $put(int index, GoInt value) {
    bytes[index] = $ix(value);
  }

  @override
  void $fill(int start, int end, List<GoInt> src, int srcOff) {
    bytes.setRange(start, end, src as List<int>, srcOff);
  }
}

/// The Go expression `make([]T, len, cap)`. The allocator returns a list of
/// the given length that holds a fresh zero value in each slot.
/// The largest count of elements that make accepts.
///
/// Go compares the length against maxAlloc divided by the size of the
/// element. maxAlloc holds 2 to the power 48 on a 64-bit machine, thus a
/// slice of an element of 8 bytes reaches this count. The allocator of Dart
/// takes no element size, thus 1 bound stands for each element type. A count
/// between this bound and the bound of Go needs at least 32 terabytes, which
/// no machine holds, and Go reports "out of memory" there.
///
/// The value stands as a decimal literal, because the operator << of the JS
/// target holds 32 bits. `1 << 45` gives 8,192 there, and each slice above
/// that length would panic.
const int $maxSliceLen = 35184372088832;

Slice<T> $makeSlice<T>(int len, int cap, List<T> Function(int) alloc) {
  if (len < 0 || len > $maxSliceLen) {
    $panic(Iface($typeGoError, const GoError('makeslice: len out of range')));
  }
  if (cap < len || cap > $maxSliceLen) {
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
  return $appendRange(s, add, 0, add.length, alloc, assign);
}

/// Adds n elements of src, from the index off. $append and $appendSlice each
/// read this body, thus 1 place holds the growth rule.
///
/// $fill moves a whole range. A list of typed data then copies the bytes
/// with 1 move, and no per element call stands in the loop.
Slice<T> $appendRange<T>(
  Slice<T> s,
  List<T> src,
  int off,
  int n,
  List<T> Function(int) alloc, [
  void Function(T, T)? assign,
]) {
  final int need = s.len + n;
  if (need <= s.cap) return s.$grow(src, off, n);
  int cap = s.cap == 0 ? need : s.cap;
  while (cap < need) {
    cap += cap < 256 ? cap : cap ~/ 4;
  }
  final List<T> array = alloc(cap);
  final Slice<T> out = Slice<T>(array, 0, need, cap);
  if (assign != null) {
    for (var i = 0; i < s.len; i++) {
      assign(array[i], s.array[s.off + i]);
    }
  } else {
    out.$fill(0, s.len, s.array, s.off);
  }
  if (n == 1) {
    out.$put(s.len, src[off]);
  } else {
    out.$fill(s.len, need, src, off);
  }
  return out;
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
  if (n == 0) return 0;
  if (assign != null) {
    for (var i = 0; i < n; i++) {
      assign(dst.array[dst.off + i], src.array[src.off + i]);
    }
    return n;
  }
  // $fill moves a whole range, and it holds the result of Go when the 2
  // ranges overlap.
  dst.$fill(dst.off, dst.off + n, src.array, src.off);
  return n;
}

/// The Go expression `append(s, other...)`.
Slice<T> $appendSlice<T>(
  Slice<T> s,
  Slice<T> other,
  List<T> Function(int) alloc, [
  void Function(T, T)? assign,
]) {
  if (other.len == 0) return s;
  if (assign != null) {
    // Go copies each element of the source. A plain append copies its argument
    // at the call site, and a spread has no such point, thus the copy stands
    // here.
    final List<T> fresh = alloc(other.len);
    for (var i = 0; i < other.len; i++) {
      assign(fresh[i], other.array[other.off + i]);
    }
    return $appendRange(s, fresh, 0, other.len, alloc, assign);
  }
  return $appendRange(s, other.array, other.off, other.len, alloc);
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

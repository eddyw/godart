// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:collection';
import 'dart:typed_data';

import 'array.dart' show $ArrayView;
import 'goint.dart';

import 'iface.dart';
import 'panic.dart';

/// A Go slice: a view on a backing array with a length and a capacity.
///
/// The header is a value, thus each field is final. An operation that changes
/// the header, for example append, returns a new header.
///
/// The real class of `array` varies: Uint8List, Int32List, $Float64List, a
/// growable list, and more. Dart thus cannot resolve `array[i]` to 1 target,
/// and each read of a byte takes a megamorphic dispatch. `$ByteSlice` holds
/// the storage of a byte slice in a field of the type Uint8List, thus a call
/// site of `[]` sees 2 targets at most and each read reaches 1 list class.
final class Slice<T> {
  final List<T> array;
  final int off;
  final int len;
  final int cap;

  /// A zero-size slice can exceed the exact JS integer range.
  GoInt get $length => $gi(len);
  GoInt get $capacity => $gi(cap);

  /// True for the zero value of a slice type. Go compares such a slice with
  /// nil, but it allows len, cap, and append on it.
  final bool isNil;

  /// Builds a header. A backing array of bytes gives a `$ByteSlice`.
  ///
  /// The JS target holds a pair for each Go integer, thus no byte slice holds
  /// a Uint8List there and the constant `$isJS` removes the test.
  factory Slice(List<T> array, int off, int len, int cap) {
    if (array is $ZeroList<T>) {
      return $ZeroSlice<T>(array, $gi(len), $gi(cap));
    }
    if (!$isJS && array is Uint8List) {
      return $ByteSlice(array as Uint8List, off, len, cap) as Slice<T>;
    }
    if ($isJS && array is List<GoInt>) {
      return _IntegerSlice(array as List<GoInt>, off, len, cap) as Slice<T>;
    }
    return Slice<T>._(array, off, len, cap, false);
  }

  const Slice._(this.array, this.off, this.len, this.cap, this.isNil);

  /// The zero value of a slice type. The backing array is 1 shared empty
  /// list, because a nil slice holds a length of 0 and a capacity of 0, thus
  /// no write ever reaches it. A List of Never fits each List of T.
  factory Slice.nil() => Slice<T>._(const <Never>[], 0, 0, 0, true);

  /// Wraps a full backing array.
  factory Slice.of(List<T> array) => array is $ZeroList<T>
      ? $ZeroSlice<T>(array, array.$length, array.$length)
      : Slice<T>(array, 0, array.length, array.length);

  // The 2 operators hold the body of $at and of $set. A call of a helper
  // would stand in each read of a byte, and a scanner reads 2 megabytes.
  T operator [](int i) {
    if (i < 0 || i >= len) $panicIndex($gi(i), len);
    return array[off + i];
  }

  void operator []=(int i, T value) {
    if (i < 0 || i >= len) $panicIndex($gi(i), len);
    array[off + i] = value;
  }

  /// The element at index i. uns names an index of an unsigned type, which
  /// reads as a negative Dart int above 2^63.
  T $at(GoInt index, [bool uns = false]) {
    final i = $ix(index);
    if (i < 0 || i >= len) $panicIndex(index, len, uns);
    return array[off + i];
  }

  void $set(GoInt index, T value, [bool uns = false]) {
    final i = $ix(index);
    if (i < 0 || i >= len) $panicIndex(index, len, uns);
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
    GoInt lowIndex,
    GoInt highIndex,
    GoInt maxIndex, [
    bool three = false,
    bool byLen = false,
    bool lowUns = false,
    bool highUns = false,
    bool maxUns = false,
  ]) {
    final low = lowIndex;
    final high = highIndex;
    final max = maxIndex;
    if (three) {
      if (max < GoInt.zero || max > $capacity) {
        $panicBound(
          byLen ? SliceBound.threeALen : SliceBound.threeACap,
          maxIndex,
          $capacity,
          maxUns,
        );
      }
      if (high < GoInt.zero || high > max) {
        $panicBound(SliceBound.threeB, highIndex, max, highUns);
      }
      if (low < GoInt.zero || low > high) {
        $panicBound(SliceBound.threeC, lowIndex, high, lowUns);
      }
    } else {
      if (high < GoInt.zero || high > $capacity) {
        $panicBound(
          byLen ? SliceBound.aLen : SliceBound.aCap,
          highIndex,
          $capacity,
          highUns,
        );
      }
      if (low < GoInt.zero || low > high) {
        $panicBound(SliceBound.b, lowIndex, high, lowUns);
      }
    }
    // Go gives a nil slice for each valid slice expression of a nil slice.
    // Such a slice holds a capacity of 0, thus each bound is 0 here.
    if (isNil) return this;
    return $viewExact(low, high - low, max - low);
  }

  /// A header on the same storage, with a relative offset.
  Slice<T> $viewExact(GoInt low, GoInt n, GoInt room) =>
      $view(off + $ix(low), $ix(n), $ix(room));

  /// A header on the same storage, with an absolute offset.
  Slice<T> $view(int start, int n, int room) =>
      Slice<T>._(array, start, n, room, false);
}

final class _IntegerSlice extends Slice<GoInt> {
  final List<GoInt> words;

  _IntegerSlice(this.words, int off, int len, int cap)
    : super._(words, off, len, cap, false);

  @override
  GoInt $at(GoInt index, [bool uns = false]) {
    final i = $ix(index);
    if (i < 0 || i >= len) $panicIndex(index, len, uns);
    return words[off + i];
  }

  @override
  void $set(GoInt index, GoInt value, [bool uns = false]) {
    final i = $ix(index);
    if (i < 0 || i >= len) $panicIndex(index, len, uns);
    words[off + i] = value;
  }

  @override
  Slice<GoInt> $view(int start, int n, int room) =>
      _IntegerSlice(words, start, n, room);
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
    if (i < 0 || i >= len) $panicIndex($gi(i), len);
    return $gi(bytes[off + i]);
  }

  @override
  void operator []=(int i, GoInt value) {
    if (i < 0 || i >= len) $panicIndex($gi(i), len);
    bytes[off + i] = $ix(value);
  }

  @override
  GoInt $at(GoInt index, [bool uns = false]) {
    final i = $ix(index);
    if (i < 0 || i >= len) $panicIndex(index, len, uns);
    return $gi(bytes[off + i]);
  }

  @override
  void $set(GoInt index, GoInt value, [bool uns = false]) {
    final i = $ix(index);
    if (i < 0 || i >= len) $panicIndex(index, len, uns);
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

/// The maximum element count for a slice with nonzero element size.
/// The decimal literal avoids 32-bit shift truncation on JS.
const int $maxSliceLen = 35184372088832;

Slice<T> $makeSlice<T>(GoInt len, GoInt cap, List<T> Function(int) alloc) {
  final empty = alloc(0);
  final zeroSize = empty is $ZeroList<T>;
  if (len < GoInt.zero || (!zeroSize && len > $gi($maxSliceLen))) {
    $panic($runtimeError('makeslice: len out of range'));
  }
  if (cap < len || (!zeroSize && cap > $gi($maxSliceLen))) {
    $panic($runtimeError('makeslice: cap out of range'));
  }
  if (empty is $ZeroList<T>) return $ZeroSlice<T>(empty, len, cap);
  return Slice(alloc($ix(cap)), 0, $ix(len), $ix(cap));
}

Slice<T> $makeSliceN<T>(GoInt len, List<T> Function(int) alloc) =>
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
  final count = s.$length + $gi(n);
  if (count < s.$length) {
    $panic($runtimeError('growslice: len out of range'));
  }
  if (s is $ZeroSlice<T>) return s.appendCount($gi(n));
  final int need = $ix(count);
  if (need <= s.cap) {
    if (assign == null) return s.$grow(src, off, n);
    for (int i = 0; i < n; i++) {
      assign(s.array[s.off + s.len + i], src[off + i]);
    }
    return Slice<T>(s.array, s.off, need, s.cap);
  }
  final empty = alloc(0);
  if (empty is $ZeroList<T>) return $ZeroSlice<T>(empty, count, count);
  final cap = $nextSliceCapacity(s.cap, need);
  final List<T> array = alloc(cap);
  final Slice<T> out = Slice<T>(array, 0, need, cap);
  if (assign != null) {
    for (var i = 0; i < s.len; i++) {
      assign(array[i], s.array[s.off + i]);
    }
  } else {
    out.$fill(0, s.len, s.array, s.off);
  }
  if (assign != null) {
    for (var i = 0; i < n; i++) {
      assign(array[s.len + i], src[off + i]);
    }
    return out;
  }
  if (n == 1) {
    out.$put(s.len, src[off]);
  } else {
    out.$fill(s.len, need, src, off);
  }
  return out;
}

/// The length of a slice of an unknown element type.
GoInt $sliceLen(Object? value) {
  final Object? unwrapped = value is Iface ? value.value : value;
  if (unwrapped is! Slice<Object?>) {
    $panic($runtimeError('sort: the value is not a slice'));
  }
  return unwrapped.$length;
}

/// The Go builtin `copy`. The assign function writes 1 element. A struct
/// element needs a field copy, and not a reference write.
GoInt $copySlice<T>(Slice<T> dst, Slice<T> src, [void Function(T, T)? assign]) {
  final count = dst.$length < src.$length ? dst.$length : src.$length;
  if (dst is $ZeroSlice<T> || src is $ZeroSlice<T>) return count;
  final n = $ix(count);
  if (n == 0) return GoInt.zero;
  if (assign == null &&
      dst.array is! $ArrayView<T> &&
      src.array is! $ArrayView<T>) {
    dst.$fill(dst.off, dst.off + n, src.array, src.off);
    return count;
  }
  final (destination, destinationOffset) = _sliceStorage(dst.array, dst.off);
  final (source, sourceOffset) = _sliceStorage(src.array, src.off);
  if (identical(destination, source) &&
      destinationOffset > sourceOffset &&
      destinationOffset < sourceOffset + n) {
    if (assign != null) {
      for (var i = n - 1; i >= 0; i--) {
        assign(dst.array[dst.off + i], src.array[src.off + i]);
      }
    } else {
      for (var i = n - 1; i >= 0; i--) {
        dst[i] = src[i];
      }
    }
    return count;
  }
  if (assign != null) {
    for (var i = 0; i < n; i++) {
      assign(dst.array[dst.off + i], src.array[src.off + i]);
    }
    return count;
  }
  // $fill moves a whole range, and it holds the result of Go when the 2
  // ranges overlap.
  dst.$fill(dst.off, dst.off + n, src.array, src.off);
  return count;
}

// Slice-to-array pointer views can hide shared storage.
(List<T>, int) _sliceStorage<T>(List<T> array, int offset) {
  while (array is $ArrayView<T>) {
    offset += array.off;
    array = array.array;
  }
  return (array, offset);
}

/// The Go expression `append(s, other...)`.
Slice<T> $appendSlice<T>(
  Slice<T> s,
  Slice<T> other,
  List<T> Function(int) alloc, [
  void Function(T, T)? assign,
]) {
  if (other.$length == GoInt.zero) return s;
  if (s is $ZeroSlice<T>) return s.appendCount(other.$length);
  if (other is $ZeroSlice<T>) {
    return $ZeroSlice<T>(
      other.array as $ZeroList<T>,
      s.$length,
      s.$capacity,
    ).appendCount(other.$length);
  }
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
void $clearSlice<T>(
  Slice<T> s,
  T Function() zero, [
  void Function(T, T)? assign,
]) {
  if (s is $ZeroSlice<T>) return;
  for (var i = 0; i < s.len; i++) {
    if (assign == null) {
      s[i] = zero();
    } else {
      assign(s[i], zero());
    }
  }
}

/// The Go byte slice of a plain int slice of the runtime. The native form
/// shares the storage, because the element type erases there, and the JS
/// form copies each element into a pair.
Slice<GoInt> $gils(Slice<int> v) {
  if (v.isNil) return Slice<GoInt>.nil();
  if ($isJS) {
    final List<GoInt> out = <GoInt>[
      for (var i = 0; i < v.len; i++) $gi(v.array[v.off + i]),
    ];
    return Slice.of(out);
  }
  return Slice<GoInt>(v.array as List<GoInt>, v.off, v.len, v.cap);
}

/// Reports whether 2 slices share a range of 1 backing array. The package
/// slices reads it, and Go reads the address of the elements there.
///
/// A slice of no element overlaps nothing, exactly as Go answers.
bool $sliceOverlap(Iface? a, Iface? b) {
  final Object? x = a?.value;
  final Object? y = b?.value;
  if (x is! Slice<Object?> || y is! Slice<Object?>) return false;
  if (x.len == 0 || y.len == 0) return false;
  if (!identical(x.array, y.array)) return false;
  return x.off <= y.off + y.len - 1 && y.off <= x.off + x.len - 1;
}

final class $ZeroList<T> extends ListBase<T> {
  final GoInt $length;
  final T Function() zero;
  late final T value = zero();

  $ZeroList(this.$length, this.zero);

  @override
  int get length => $ix($length);
  @override
  set length(int value) => throw UnsupportedError('fixed length');
  @override
  T operator [](int index) {
    if (index < 0 || index >= length) throw RangeError.index(index, this);
    return value;
  }

  @override
  void operator []=(int index, T value) {
    if (index < 0 || index >= length) throw RangeError.index(index, this);
  }
}

final class $ZeroSlice<T> extends Slice<T> {
  @override
  final GoInt $length;
  @override
  final GoInt $capacity;

  factory $ZeroSlice($ZeroList<T> array, GoInt length, GoInt capacity) {
    final storage = array.$length < capacity
        ? $ZeroList<T>(capacity, array.zero)
        : array;
    return $ZeroSlice<T>._(storage, length, capacity);
  }

  $ZeroSlice._($ZeroList<T> array, this.$length, this.$capacity)
    : super._(array, 0, $ix($length), $ix($capacity), false);

  @override
  T operator [](int i) => $at($gi(i));

  @override
  void operator []=(int i, T value) => $set($gi(i), value);

  @override
  T $at(GoInt index, [bool uns = false]) {
    if (index < GoInt.zero || index >= $length) {
      $panicIndex(index, $length, uns);
    }
    return array[0];
  }

  @override
  void $set(GoInt index, T value, [bool uns = false]) {
    $at(index, uns);
  }

  @override
  Slice<T> $viewExact(GoInt low, GoInt n, GoInt room) =>
      $ZeroSlice<T>(array as $ZeroList<T>, n, room);

  $ZeroSlice<T> appendCount(GoInt count) {
    final need = $length + count;
    if (need < $length) {
      $panic($runtimeError('growslice: len out of range'));
    }
    return $ZeroSlice<T>(
      array as $ZeroList<T>,
      need,
      need > $capacity ? need : $capacity,
    );
  }
}

int $nextSliceCapacity(int current, int need) {
  var capacity = current == 0 ? need : current;
  while (capacity < need) {
    capacity += capacity < 256 ? capacity : capacity ~/ 4;
  }
  return capacity;
}

// Integer elements cannot invoke Go equality methods or panic.
bool $equalIntegerSlices(Slice<GoInt> left, Slice<GoInt> right) {
  if (left.len != right.len) return false;
  for (var i = 0; i < left.len; i++) {
    if (left.array[left.off + i] != right.array[right.off + i]) return false;
  }
  return true;
}

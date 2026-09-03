import 'dart:collection';

import 'panic.dart';
import 'slice.dart';

/// A Go array is a fixed length list with value semantics. The compiler emits
/// a plain Dart list, and it inserts a copy at each copy point.
List<T> $makeArray<T>(int n, T Function() zero) =>
    List<T>.generate(n, (_) => zero(), growable: false);

/// Builds an array of the first n elements of a slice. Go panics when the
/// slice holds fewer than n elements.
List<T> $sliceToArray<T>(Slice<T> s, int n) {
  if (s.len < n) $panicConvert(s.len, n);
  return List<T>.generate(n, (i) => s[i], growable: false);
}

/// Builds a pointer to an array of the first n elements of a slice. The array
/// shares the elements of the slice, thus a write through the pointer reaches
/// the slice. Go panics when the slice holds fewer than n elements.
List<T>? $sliceToArrayPtr<T>(Slice<T> s, int n) {
  if (s.len < n) $panicConvert(s.len, n);
  // A nil slice gives a nil pointer, and an empty slice that is not nil gives
  // a pointer that is not nil. Only a length of 0 reaches both.
  if (s.isNil) return null;
  if (s.off == 0 && s.array.length == n) return s.array;
  return $ArrayView<T>(s.array, s.off, n);
}

/// A window of n elements of a backing list, from an offset. It holds the
/// array of a pointer that a conversion of a slice gives.
final class $ArrayView<T> extends ListBase<T> {
  final List<T> array;
  final int off;
  final int count;

  $ArrayView(this.array, this.off, this.count);

  @override
  int get length => count;

  @override
  set length(int value) => $panic('the length of an array is fixed');

  @override
  T operator [](int i) {
    if (i < 0 || i >= count) $panicIndex(i, count);
    return array[off + i];
  }

  @override
  void operator []=(int i, T value) {
    if (i < 0 || i >= count) $panicIndex(i, count);
    array[off + i] = value;
  }
}

/// Copies an array. The element copy function is not null for a struct
/// element and for an array element.
List<T> $copyArray<T>(List<T> a, [T Function(T)? copy]) {
  if (copy == null) return List<T>.of(a, growable: false);
  return List<T>.generate(a.length, (i) => copy(a[i]), growable: false);
}

/// Assigns an array in place. Go writes through the destination, thus a
/// pointer to an element observes the new value.
void $assignArray<T>(List<T> dst, List<T> src, [void Function(T, T)? assign]) {
  if (assign == null) {
    dst.setRange(0, dst.length, src);
    return;
  }
  for (var i = 0; i < dst.length; i++) {
    assign(dst[i], src[i]);
  }
}

bool $equalArray<T>(List<T> a, List<T> b, [bool Function(T, T)? equal]) {
  for (var i = 0; i < a.length; i++) {
    if (equal == null ? a[i] != b[i] : !equal(a[i], b[i])) return false;
  }
  return true;
}

/// The element at index i of an array. uns names an index of an unsigned
/// type, which reads as a negative Dart int above 2^63.
T $arrayIndex<T>(List<T> a, int i, [bool uns = false]) {
  if (i < 0 || i >= a.length) $panicIndex(i, a.length, uns);
  return a[i];
}

void $arraySet<T>(List<T> a, int i, T v, [bool uns = false]) {
  if (i < 0 || i >= a.length) $panicIndex(i, a.length, uns);
  a[i] = v;
}

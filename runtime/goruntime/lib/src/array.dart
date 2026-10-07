import 'dart:collection';

import 'goint.dart';
import 'panic.dart';
import 'slice.dart';

final Expando<Map<(int, GoInt), Object>> _arrayViews = Expando('array views');

/// A Go array is a fixed length list with value semantics. The compiler emits
/// a plain Dart list, and it inserts a copy at each copy point.
List<T> $makeArray<T>(int n, T Function() zero) => n == 0
    ? List<T>.empty(growable: false)
    : List<T>.generate(n, (_) => zero(), growable: false);

GoInt $arrayLength<T>(List<T> array) =>
    array is $ZeroList<T> ? array.$length : $gi(array.length);

/// Builds an array of the first n elements of a slice. Go panics when the
/// slice holds fewer than n elements.
List<T> $sliceToArray<T>(Slice<T> s, GoInt n) {
  if (s.$length < n) $panicConvert(s.$length, n);
  if (s.array is $ZeroList<T>) {
    return $ZeroList<T>(n, (s.array as $ZeroList<T>).zero);
  }
  if (n == GoInt.zero) return List<T>.empty(growable: false);
  return List<T>.generate($ix(n), (i) => s[i], growable: false);
}

/// Builds a pointer to an array of the first n elements of a slice. The array
/// shares the elements of the slice, thus a write through the pointer reaches
/// the slice. Go panics when the slice holds fewer than n elements.
List<T>? $sliceToArrayPtr<T>(Slice<T> s, GoInt n) {
  if (s.$length < n) $panicConvert(s.$length, n);
  // A nil slice gives a nil pointer, and an empty slice that is not nil gives
  // a pointer that is not nil. Only a length of 0 reaches both.
  if (s.isNil) return null;
  List<T> array = s.array;
  int offset = s.off;
  while (array is $ArrayView<T>) {
    offset += array.off;
    array = array.array;
  }
  if (offset == 0 && $arrayLength(array) == n) return array;
  final views = _arrayViews[array] ??= <(int, GoInt), Object>{};
  return views.putIfAbsent(
    (offset, n),
    () => array is $ZeroList<T>
        ? $ZeroList<T>(n, array.zero)
        : $ArrayView<T>(array, offset, $ix(n)),
  ) as List<T>;
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
    if (i < 0 || i >= count) $panicIndex($gi(i), count);
    return array[off + i];
  }

  @override
  void operator []=(int i, T value) {
    if (i < 0 || i >= count) $panicIndex($gi(i), count);
    array[off + i] = value;
  }
}

/// Copies an array. The element copy function is not null for a struct
/// element and for an array element.
List<T> $copyArray<T>(List<T> a, [T Function(T)? copy]) {
  if (a is $ZeroList<T>) return $ZeroList<T>(a.$length, a.zero);
  if (a.isEmpty) return List<T>.empty(growable: false);
  if (copy == null) return List<T>.of(a, growable: false);
  return List<T>.generate(a.length, (i) => copy(a[i]), growable: false);
}

/// Assigns an array in place. Go writes through the destination, thus a
/// pointer to an element observes the new value.
void $assignArray<T>(List<T> dst, List<T> src, [void Function(T, T)? assign]) {
  if (dst is $ZeroList<T> || dst.isEmpty) return;
  if (assign == null) {
    dst.setRange(0, dst.length, src);
    return;
  }
  for (var i = 0; i < dst.length; i++) {
    assign(dst[i], src[i]);
  }
}

bool $equalArray<T>(List<T> a, List<T> b, [bool Function(T, T)? equal]) {
  if (a is $ZeroList<T> || b is $ZeroList<T>) return true;
  for (var i = 0; i < a.length; i++) {
    if (equal == null ? a[i] != b[i] : !equal(a[i], b[i])) return false;
  }
  return true;
}

/// The element at index i of an array. uns names an index of an unsigned
/// type, which reads as a negative Dart int above 2^63.
T $arrayIndex<T>(List<T> a, GoInt index, [bool uns = false]) {
  if (a is $ZeroList<T>) {
    if (index < GoInt.zero || index >= a.$length) {
      $panicIndex(index, a.$length, uns);
    }
    return a.value;
  }
  final i = $ix(index);
  if (i < 0 || i >= a.length) $panicIndex(index, a.length, uns);
  return a[i];
}

void $arraySet<T>(List<T> a, GoInt index, T v, [bool uns = false]) {
  if (a is $ZeroList<T>) {
    if (index < GoInt.zero || index >= a.$length) {
      $panicIndex(index, a.$length, uns);
    }
    return;
  }
  final i = $ix(index);
  if (i < 0 || i >= a.length) $panicIndex(index, a.length, uns);
  a[i] = v;
}

T $sequenceIndex<T>(Object? value, GoInt i, [bool uns = false]) {
  if (value is Slice<T>) return value.$at(i, uns);
  return $arrayIndex<T>(value as List<T>, i, uns);
}

void $sequenceSet<T>(Object? value, GoInt i, T element, [bool uns = false]) {
  if (value is Slice<T>) {
    value.$set(i, element, uns);
    return;
  }
  $arraySet<T>(value as List<T>, i, element, uns);
}

// A loop avoids a large assignment chain in the AOT optimizer.
List<T> $arrayLiteral<T>(List<T> array, List<GoInt> indices, List<T> values) {
  for (var i = 0; i < indices.length; i++) {
    $arraySet(array, indices[i], values[i]);
  }
  return array;
}

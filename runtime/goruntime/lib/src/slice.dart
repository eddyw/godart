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

  T operator [](int i) {
    if (i < 0 || i >= len) $panicIndex(i, len);
    return array[off + i];
  }

  void operator []=(int i, T value) {
    if (i < 0 || i >= len) $panicIndex(i, len);
    array[off + i] = value;
  }

  /// The Go expression `s[low:high:max]`.
  Slice<T> range(int low, int high, int max) {
    if (low < 0 || high < low || max < high || max > cap) {
      $panicSlice(low, high, cap);
    }
    return Slice(array, off + low, high - low, max - low);
  }
}

/// The Go expression `make([]T, len, cap)`. The allocator returns a list of
/// the given length that holds a fresh zero value in each slot.
Slice<T> $makeSlice<T>(int len, int cap, List<T> Function(int) alloc) {
  if (len < 0 || cap < len) {
    $panic(const GoError('makeslice: len out of range'));
  }
  return Slice(alloc(cap), 0, len, cap);
}

/// The Go builtin `append`. The result shares the backing array when the
/// capacity is enough, exactly as in Go.
Slice<T> $append<T>(Slice<T> s, List<T> add, List<T> Function(int) alloc) {
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
  for (var i = 0; i < s.len; i++) {
    array[i] = s.array[s.off + i];
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
    $panic(const GoError('sort: the value is not a slice'));
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
    $panic(const GoError('sort: the value is not a slice'));
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
  List<T> Function(int) alloc,
) {
  final add = <T>[];
  for (var i = 0; i < other.len; i++) {
    add.add(other.array[other.off + i]);
  }
  return $append(s, add, alloc);
}

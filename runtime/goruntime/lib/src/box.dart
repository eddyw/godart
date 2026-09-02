import 'dict.dart';
import 'iface.dart';
import 'panic.dart';
import 'slice.dart';

/// A pointer to a value that is not a struct.
///
/// The Dart form of a pointer to a struct is the object of the struct. Another
/// pointer needs a cell, because a Dart int, double, bool, String, Slice,
/// GoMap, or Chan gives no reference to the location that holds it.
abstract interface class $Ptr<T> {
  T get v;
  set v(T value);
}

/// A box holds 1 value behind a reference.
///
/// The compiler writes a box for a variable whose address the code takes, thus
/// a program that takes no address holds no box.
final class Box<T> implements $Ptr<T> {
  @override
  T v;

  Box(this.v);
}

/// A pointer to a field of a struct, or to an element of a slice or an array.
///
/// The cell holds the object and the index of the location, thus a read and a
/// write reach the location itself. Go compares 2 pointers with ==, and the
/// class compares the object and the index for it.
final class $Ref<O, T> implements $Ptr<T> {
  final O owner;
  final int index;
  final T Function(O, int) _get;
  final void Function(O, int, T) _set;

  $Ref(this.owner, this.index, this._get, this._set);

  @override
  T get v => _get(owner, index);

  @override
  set v(T value) => _set(owner, index, value);

  @override
  bool operator ==(Object other) =>
      other is $Ref && identical(owner, other.owner) && index == other.index;

  @override
  int get hashCode => Object.hash(identityHashCode(owner), index);
}

/// A pointer to 1 element of a slice.
$Ref<List<T>, T> $elemPtr<T>(Slice<T> s, int i) {
  if (i < 0 || i >= s.len) $panicIndex(i, s.len);
  return $Ref<List<T>, T>(s.array, s.off + i, _listGet<T>, _listSet<T>);
}

/// A pointer to 1 element of an array.
$Ref<List<T>, T> $indexPtr<T>(List<T> a, int i) {
  if (i < 0 || i >= a.length) $panicIndex(i, a.length);
  return $Ref<List<T>, T>(a, i, _listGet<T>, _listSet<T>);
}

T _listGet<T>(List<T> owner, int i) => owner[i];

void _listSet<T>(List<T> owner, int i, T value) {
  owner[i] = value;
}

/// The value behind a pointer of a generic function.
///
/// The Dart form of a pointer to a type parameter follows the type argument: a
/// pointer to a struct is the object itself, and each other pointer holds a
/// cell. Thus the read holds both forms.
T $deref<T>(Object? p) {
  if (p == null) $panicNil();
  if (p is $Ptr<T>) return p.v;
  return p as T;
}

/// The concrete form of a pointer that a generic function gave.
///
/// A generic function writes a cell for the address of a value, because it
/// holds no type argument. A pointer to a struct or to an array is the object
/// itself at the call site, thus the call site reads the cell out. A nil
/// pointer stays nil.
T? $ptrTo<T>(Object? p) {
  if (p == null) return null;
  if (p is $Ptr<T>) return p.v;
  return p as T;
}

/// Writes a value behind such a pointer. A pointer to a struct is the object
/// itself, thus the write copies each field.
void $setDeref<T>(Object? p, T value, Dict<T> d) {
  if (p == null) $panicNil();
  if (p is $Ptr<T>) {
    p.v = value;
    return;
  }
  for (final FieldDesc f in d.desc.fields) {
    f.set(p, f.get(value));
  }
}

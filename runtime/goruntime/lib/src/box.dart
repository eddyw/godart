// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'array.dart';
import 'dict.dart';
import 'goint.dart';
import 'panic.dart';
import 'slice.dart';

abstract interface class $StructView {
  Object get $origin;
}

Object? $pointerOwner(Object? value) =>
    value is $StructView ? value.$origin : value;

final Expando<Object> _inlineOwners = Expando<Object>('inline owners');

// The value must represent an inline Go struct or array.
T $inlineValue<T>(T value, Object owner) {
  if (value is List && value.isEmpty) return value;
  _inlineOwners[$pointerOwner(value)!] = $pointerOwner(owner)!;
  return value;
}

T $inlinePtr<T>($Ptr<T> pointer) => $inlineValue(
  pointer.v,
  pointer is $Ref ? (pointer as $Ref).owner as Object : pointer,
);

Object $storageOwner(Object value) {
  while (true) {
    if (value is $Ref) {
      value = value.owner as Object;
    } else if (value is $StructView) {
      value = value.$origin;
    } else if (value is $ArrayView) {
      value = value.array;
    } else {
      if (value is String) return value;
      final parent = _inlineOwners[value];
      if (parent == null) return value;
      value = parent;
    }
  }
}

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
      other is $Ref &&
      identical($pointerOwner(owner), $pointerOwner(other.owner)) &&
      index == other.index;

  @override
  int get hashCode =>
      Object.hash(identityHashCode($pointerOwner(owner)), index);
}

/// A pointer to the cell at index i of a list.
///
/// A conversion of a slice to a pointer to an array gives an $ArrayView over
/// the storage of the slice. Go gives 1 address for the element of the slice
/// and for the element of the array, thus the pointer names the cell of the
/// storage and not the cell of the view. A view over a view holds the same
/// rule, thus the walk goes to the list that holds the storage.
$Ref<List<T>, T> _cellPtr<T>(List<T> a, int i) {
  List<T> list = a;
  int at = i;
  while (list is $ArrayView<T>) {
    at += list.off;
    list = list.array;
  }
  return $Ref<List<T>, T>(list, at, _listGet<T>, _listSet<T>);
}

/// A pointer to 1 element of a slice.
$Ref<List<T>, T> $elemPtr<T>(Slice<T> s, GoInt index, [bool uns = false]) {
  if (s is $ZeroSlice<T>) {
    s.$at(index, uns);
    return _cellPtr<T>(s.array, 0);
  }
  final i = $ix(index);
  if (i < 0 || i >= s.len) $panicIndex(index, s.len, uns);
  return _cellPtr<T>(s.array, s.off + i);
}

/// A pointer to 1 element of an array.
$Ref<List<T>, T> $indexPtr<T>(List<T> a, GoInt index, [bool uns = false]) {
  final i = $ix(index);
  if (i < 0 || i >= a.length) $panicIndex(index, a.length, uns);
  return _cellPtr<T>(a, i);
}

$Ref<List<T>, T> $sequencePtr<T>(Object? value, GoInt i, [bool uns = false]) {
  if (value is Slice<T>) return $elemPtr<T>(value, i, uns);
  return $indexPtr<T>(value as List<T>, i, uns);
}

// Bigmod supplies an element pointer with n available uint cells.
Slice<GoInt> $bigmodUintSlice($Ptr<GoInt>? pointer, int n) {
  final ref = pointer as $Ref<List<GoInt>, GoInt>;
  return Slice<GoInt>(ref.owner, ref.index, n, n);
}

// The pointers address n available words. Reads precede each destination write.
GoInt $bigMulAddVWW($Ptr<GoInt> z, $Ptr<GoInt> x, int n, GoInt m, GoInt a) {
  final destination = z as $Ref<List<GoInt>, GoInt>;
  final source = x as $Ref<List<GoInt>, GoInt>;
  var carry = a;
  for (var i = 0; i < n; i++) {
    final result = $mulAddWord64(
      source.owner[source.index + i],
      m,
      GoInt.zero,
      carry,
    );
    destination.owner[destination.index + i] = result.$2;
    carry = result.$1;
  }
  return carry;
}

// The pointers address n available words. Reads precede each destination write.
GoInt $bigAddMulVVWW(
  $Ptr<GoInt> z,
  $Ptr<GoInt> x,
  $Ptr<GoInt> y,
  int n,
  GoInt m,
  GoInt a,
) {
  final destination = z as $Ref<List<GoInt>, GoInt>;
  final left = x as $Ref<List<GoInt>, GoInt>;
  final right = y as $Ref<List<GoInt>, GoInt>;
  var carry = a;
  for (var i = 0; i < n; i++) {
    final result = $mulAddWord64(
      right.owner[right.index + i],
      m,
      left.owner[left.index + i],
      carry,
    );
    destination.owner[destination.index + i] = result.$2;
    carry = result.$1;
  }
  return carry;
}

T _listGet<T>(List<T> owner, int i) => owner[i];

void _listSet<T>(List<T> owner, int i, T value) {
  owner[i] = value;
}

Object? $allocPtr<T>(T value, Dict<T> dict) {
  final T copy = dict.copyOf(value);
  return dict.desc.kind == 'struct' || dict.desc.kind == 'array'
      ? copy
      : Box<T>(copy);
}

/// The value behind a pointer of a generic function.
///
/// The Dart form of a pointer to a type parameter follows the type argument: a
/// pointer to a struct or array is the object itself, and other pointers hold a
/// cell. Thus the read holds both forms.
T $deref<T>(Object? p) {
  if (p == null) $panicNil();
  if (p is $Ptr<T>) return p.v;
  return p as T;
}

/// The concrete form of a pointer that a generic function gave.
///
/// A generic local can use a cell. A pointer to a struct or array uses the
/// object itself at a concrete call site. A nil pointer stays nil.
T? $ptrTo<T>(Object? p) {
  if (p == null) return null;
  if (p is $Ptr<T>) return $inlinePtr(p);
  return p as T;
}

/// Writes a value behind such a pointer. A pointer to a struct is the object
/// itself, thus the write copies each field.
void $setDeref<T>(Object? p, T value, Dict<T> d) {
  if (p == null) $panicNil();
  if (p is $Ptr<T>) {
    p.v = d.assignOf(p.v, value);
    return;
  }
  d.assignOf(p as T, value);
}

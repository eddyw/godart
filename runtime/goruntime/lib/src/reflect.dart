// This file includes code derived from the Go reflect package.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of the Go-derived portions is governed by the BSD-style
// license in ../../LICENSE.

/// The intrinsics of the package reflect.
///
/// A Go value of an interface type is an Iface: a type descriptor and the value
/// itself. Each function below reads or writes such a pair, thus the package
/// reflect holds no memory layout.
library;

import 'dart:collection';

import 'array.dart';
import 'box.dart';
import 'chan.dart';
import 'complex.dart';
import 'dict.dart';
import 'funcpc.dart';
import 'goint.dart';
import 'iface.dart';
import 'int.dart';
import 'layout.dart';
import 'map.dart';
import 'panic.dart';
import 'sched.dart';
import 'select.dart';
import 'slice.dart';
import 'string.dart';

/// The type descriptors that the package reflect reads. A handle is an index of
/// the list plus 1, thus the handle 0 names no type.
///
/// A Go value carries no descriptor of a type that no value holds, for example
/// the element type of a pointer. The table gives the package reflect a name
/// for such a type.
final List<TypeDesc> _descTable = <TypeDesc>[];
final Map<String, int> _descHandles = <String, int>{};

/// The handles that a type constructor built. Such a descriptor holds an
/// untyped zero and an untyped make, thus a descriptor of the emitted code
/// replaces it when a value of the same type reaches the package. Read
/// $sliceOf.
final Set<int> _synthetic = <int>{};
final Map<String, TypeDesc Function()> _typeFactories = {};

/// A factory must depend only on type metadata, never on Go values.
void $registerTypeFactory(String name, TypeDesc Function() factory) {
  _typeFactories.putIfAbsent(name, () => factory);
}

final Map<String, TypeDesc Function(List<TypeDesc>)> _structStorage = {};

/// A factory depends only on type metadata and registers before Go initializers.
void $registerStructStorage(
  String shape,
  TypeDesc Function(List<TypeDesc>) factory,
) {
  _structStorage.putIfAbsent(shape, () => factory);
}

/// Storage uses only the zero, equality, hash, copy, and assignment operations.
R $withStorageType<R>(TypeDesc desc, R Function<T>(Dict<T>) f) {
  R apply<T>() => f<T>(
    Dict<T>(
      () => _zeroOf(desc) as T,
      (a, b) => $goEqual(a, b, desc),
      (v) => $goHash(v, desc),
      (n) => _zeros<T>(desc, $gi(n)),
      $unordered<T>,
      $identity<T>,
      desc,
      copy: _holdsValue(desc) ? (v) => _copyOf(v, desc) as T : null,
      assign: _holdsValue(desc)
          ? (dst, src) {
              _assignInto(dst, src, desc);
              return dst;
            }
          : null,
    ),
  );
  final X Function<X>(X Function<Y>())? make =
      desc.withStorageType ?? desc.withType;
  if (make != null) return make<R>(apply);
  return _byKind<R>(desc.kind, apply);
}

int _descHandle(TypeDesc desc) {
  final int? found = _descHandles[desc.name];
  if (found != null) {
    if (_synthetic.contains(found)) {
      _descTable[found - 1] = desc;
      _synthetic.remove(found);
      // The new descriptor can hold another text, thus the cache drops the
      // answer of the old one.
      if (found <= _descInfoTable.length) _descInfoTable[found - 1] = null;
    }
    return found;
  }
  _descTable.add(desc);
  _descHandles[desc.name] = _descTable.length;
  return _descTable.length;
}

/// _syntheticHandle registers a descriptor that a type constructor built. A
/// name that the table already holds keeps its own descriptor.
int _syntheticHandle(TypeDesc Function() build) {
  final TypeDesc desc = build();
  final int? found = _descHandles[desc.name];
  if (found != null) return found;
  final factory = _typeFactories[desc.name];
  if (factory != null) return _descHandle(factory());
  _descTable.add(desc);
  _descHandles[desc.name] = _descTable.length;
  _synthetic.add(_descTable.length);
  return _descTable.length;
}

TypeDesc? _descAt(int handle) {
  if (handle <= 0 || handle > _descTable.length) return null;
  return _descTable[handle - 1];
}

/// A handle of the type of an interface value. A nil value gives 0.
int $descOf(Iface? value) => value == null ? 0 : _descHandle(value.desc);

/// The answer of $descInfo for each handle. A GoString holds UTF-8 bytes, thus
/// each build of one encodes the text again. The package reflect reads the
/// kind of a type for each value, and 1 build stands for each later read.
final List<(GoString, GoString, bool)?> _descInfoTable =
    <(GoString, GoString, bool)?>[];

/// The text, the kind, and the comparable flag of a type handle.
(GoString, GoString, bool) $descInfo(int handle) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null) return (GoString.empty, GoString.empty, false);
  while (_descInfoTable.length < handle) {
    _descInfoTable.add(null);
  }
  final (GoString, GoString, bool)? found = _descInfoTable[handle - 1];
  if (found != null) return found;
  final (GoString, GoString, bool) info = (
    GoString.fromString(desc.display),
    GoString.fromString(desc.kind),
    desc.comparable,
  );
  _descInfoTable[handle - 1] = info;
  return info;
}

/// A handle of the element type of a type. A type that holds no element type
/// gives 0.
int $descElem(int handle) {
  final TypeDesc? desc = _descAt(handle);
  final TypeDesc Function()? elem = desc?.elem;
  if (elem == null) return 0;
  return _descHandle(elem());
}

/// The zero value of a type, inside an interface value. The zero value of an
/// interface type is a nil interface, thus it holds no descriptor.
Iface? $descZero(int handle) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null || desc.kind == 'interface') return null;
  final Object? Function()? zero = desc.zero;
  if (zero == null) {
    $panic($runtimeError('reflect: no zero value of ${desc.display}'));
  }
  return Iface(desc, zero());
}

/// The value behind a pointer, inside an interface value.
///
/// A pointer to a value that is not a struct holds a box, and a pointer to a
/// struct is the object itself.
Iface? $pointerElem(Iface? pointer) {
  final TypeDesc elem = _pointerElemDesc(pointer);
  final Object? target = pointer!.value;
  if (target is $Ptr<Object?>) {
    if (elem.kind == 'interface') return target.v as Iface?;
    return Iface(elem, target.v);
  }
  return Iface(elem, target);
}

/// Writes a value behind a pointer.
///
/// A pointer to a struct is the object itself, thus the write copies each
/// field into the target object and preserves the address of each field.
void $pointerSet(Iface? pointer, Iface? value) {
  final TypeDesc elem = _pointerElemDesc(pointer);
  final Object? target = pointer!.value;
  final Object? stored = elem.kind == 'interface' ? value : value?.value;
  if (target is $Ptr<Object?>) {
    target.v = stored;
    return;
  }
  if (_holdsValue(elem)) {
    _assignInto(target, stored, elem);
    return;
  }
  $panic($runtimeError('reflect: Set on a pointer to ${elem.display}'));
}

/// True for a kind whose Dart form is an object and whose Go form is a value. A
/// write of such a value copies each part.
bool _holdsValue(TypeDesc desc) =>
    desc.kind == 'struct' || desc.kind == 'array';

/// _assignInto writes a value into a location that already holds the object.
///
/// Go copies a struct and an array at an assignment, and a field of such a type
/// holds a value of its own. The Dart form of each 1 is an object, thus a write
/// of the reference alone joins the 2 values and a later write of the source
/// reaches the target. The walk below reaches each level.
///
/// The object of the target stays, because Go holds the address of a field of
/// the target and a fresh object would leave that address behind.
void _assignInto(Object? dst, Object? src, TypeDesc desc) {
  if (dst == null) return;
  final void Function(Object?, Object?)? assign = desc.assign;
  if (assign != null) {
    assign(dst, src);
    return;
  }
  if (desc.kind == 'array') {
    if (dst is $ZeroList<Object?>) return;
    final List<Object?> a = dst as List<Object?>;
    final List<Object?> b = src is List<Object?> ? src : const <Object?>[];
    final TypeDesc t = desc.elem!();
    final int n = a.length < b.length ? a.length : b.length;
    if (n == 0) return;
    if (!_holdsValue(t)) {
      a.setRange(0, n, b);
      return;
    }
    for (int i = 0; i < n; i++) {
      final Object? inner = a[i];
      if (inner == null) {
        a[i] = _copyOf(b[i], t);
        continue;
      }
      _assignInto(inner, b[i], t);
    }
    return;
  }
  for (final FieldDesc field in desc.fields) {
    final TypeDesc t = field.type();
    final Object? part = field.get(src);
    if (!_holdsValue(t)) {
      field.set(dst, part);
      continue;
    }
    final Object? inner = field.get(dst);
    if (inner == null) {
      field.set(dst, _copyOf(part, t));
      continue;
    }
    _assignInto(inner, part, t);
  }
}

/// Stops the program when the target of errors.As holds no pointer, holds a
/// nil pointer, or points to a type that is neither an interface type nor a
/// type of the error interface.
///
/// Go panics with a text of the kind string there, and not with a runtime
/// error, thus a recover of such a panic reads a string.
///
/// errors.As holds the case of a nil target itself, thus this check reads a
/// target that stands.
void $checkErrorTarget(Iface? target) {
  if (target == null) return;
  final TypeDesc desc = target.desc;
  if (desc.kind != 'ptr' || desc.elem == null || target.value == null) {
    $panic(
      Iface(
        $typeString,
        GoString.fromString('errors: target must be a non-nil pointer'),
      ),
    );
  }
  final TypeDesc elem = desc.elem!();
  if (elem.kind == 'interface') return;
  if (!elem.methods.containsKey('Error()string')) {
    $panic(
      Iface(
        $typeString,
        GoString.fromString(
          'errors: *target must be interface or implement error',
        ),
      ),
    );
  }
}

/// Writes a value into a location that holds the object of a struct or of an
/// array. The decoder of encoding/json writes a target of such a kind with it.
void $assignValue(Object? dst, Object? src, TypeDesc desc) =>
    _assignInto(dst, src, desc);

/// Writes a value into the target of errors.As.
///
/// The target holds a pointer. The Dart form of a pointer to a struct and of a
/// pointer to an array is the object itself, thus the write there copies each
/// part of the value. Each other pointer holds a cell.
///
/// A target of a pointer to an interface type takes the method set: the
/// description of such a type holds the key of each method of it.
bool $assignError(Iface? value, Iface? target) {
  if (value == null || target == null) return false;
  final TypeDesc desc = target.desc;
  final TypeDesc elem = desc.elem!();
  final Object? box = target.value;
  if (elem.kind == 'interface') {
    if (!$implements(value, elem.methods.keys.toList())) return false;
    if (box is $Ptr<Object?>) {
      box.v = value;
      return true;
    }
    return false;
  }
  if (elem.name != value.desc.name) return false;
  if (box is $Ptr<Object?>) {
    box.v = value.value;
    return true;
  }
  if (_holdsValue(elem) && box != null) {
    _assignInto(box, value.value, elem);
    return true;
  }
  return false;
}

/// _copyOf gives a fresh value of a struct or of an array, and it reaches each
/// level. Each other kind holds no object of its own, thus it gives the value
/// again.
///
/// A type that the program never names holds no zero, and the copy then gives
/// the value itself. Only reflect.StructOf builds such a type.
Object? _copyOf(Object? v, TypeDesc desc) {
  if (v == null || !_holdsValue(desc)) return v;
  final Object? Function()? zero = desc.zero;
  if (zero == null) return v;
  final Object? out = zero();
  if (out == null) return v;
  _assignInto(out, v, desc);
  return out;
}

Iface? $copyInterface(Iface? value) {
  if (value == null || !_holdsValue(value.desc)) return value;
  return Iface(value.desc, _copyOf(value.value, value.desc));
}

TypeDesc _pointerElemDesc(Iface? pointer) {
  if (pointer == null) {
    $panic($runtimeError('reflect: the value holds no address'));
  }
  final TypeDesc desc = pointer.desc;
  final TypeDesc Function()? elem = desc.elem;
  if (desc.kind != 'ptr' || elem == null) {
    $panic($runtimeError('reflect: ${desc.display} is not a pointer'));
  }
  return elem();
}

/// The count of the elements of an array type.
GoInt $descLength(GoInt handle) {
  final TypeDesc? desc = _descAt($ix(handle));
  if (desc == null) return GoInt.zero;
  return desc.length;
}

GoInt $descSize(GoInt handle) => $typeSize(_descAt($ix(handle))!);

int $descAlign(int handle) => $typeAlign(_descAt(handle)!);

GoInt $descOffset(GoInt handle, GoInt index) =>
    $fieldOffset(_descAt($ix(handle))!, $ix(index));

/// The count of the fields of a struct type.
int $fieldCount(int handle) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null) return 0;
  return desc.fields.length;
}

/// The name, the tag, the exported flag, and the type of 1 field.
(GoString, GoString, bool, int) $fieldInfo(int handle, int index) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null || index < 0 || index >= desc.fields.length) {
    $panic($runtimeError('reflect: the type holds no such field'));
  }
  final FieldDesc field = desc.fields[index];
  return (
    GoString.fromString(field.name),
    GoString.fromString(field.tag),
    field.exported,
    _descHandle(field.type()),
  );
}

/// The value of 1 field of a struct value.
Iface? $fieldRead(Iface? value, int index) {
  final FieldDesc field = _fieldOf(value, index);
  final TypeDesc type = field.type();
  final Object? out = field.get(value!.value);
  if (type.kind == 'interface') return out as Iface?;
  return Iface(type, out);
}

/// Writes 1 field of a struct value.
void $fieldWrite(Iface? value, int index, Iface? x) {
  final FieldDesc field = _fieldOf(value, index);
  final TypeDesc type = field.type();
  final Object? stored = type.kind == 'interface' ? x : x?.value;
  if (_holdsValue(type)) {
    // Go copies a struct and an array into the field, and the address of the
    // field stays. Read _assignInto.
    final Object? target = field.get(value!.value);
    if (target != null) {
      _assignInto(target, stored, type);
      return;
    }
  }
  field.set(value!.value, stored);
}

FieldDesc _fieldOf(Iface? value, int index) {
  if (value == null) $panicNil();
  final TypeDesc desc = value.desc;
  if (desc.kind != 'struct' || index < 0 || index >= desc.fields.length) {
    $panic($runtimeError('reflect: ${desc.display} holds no field $index'));
  }
  return desc.fields[index];
}

/// The length of a slice, an array, a string, or a map.
GoInt $valueLen(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is Slice<Object?>) return v.$length;
  if (v is List<Object?>) return value.desc.length;
  if (v is GoString) return $gi(v.length);
  if (v is GoMap<Object?, Object?>) return $gi(v.length);
  // The length of a channel is the count of the values that the buffer holds.
  if (v is Chan<Object?>) return $gi(v.length);
  $panic($runtimeError('reflect: Len of ${value.desc.display}'));
}

/// 1 element of a slice, an array, or a string.
Iface? $valueIndex(Iface? value, GoInt exactIndex) {
  final index = $ix(exactIndex);
  if (value == null) $panicNil();
  final TypeDesc desc = value.desc;
  final Object? v = value.value;
  if (v is GoString) {
    return Iface(_byteDesc, $gi(v[index]));
  }
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) {
    $panic($runtimeError('reflect: Index of ${desc.display}'));
  }
  final Object? out = v is Slice<Object?>
      ? v.$at(exactIndex)
      : _elemAt(v, exactIndex, desc);
  if (elem.kind == 'interface') return out as Iface?;
  return Iface(elem, out);
}

/// Writes 1 element of a slice or an array.
void $valueSetIndex(Iface? value, GoInt exactIndex, Iface? x) {
  final index = $ix(exactIndex);
  if (value == null) $panicNil();
  final TypeDesc desc = value.desc;
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) {
    $panic($runtimeError('reflect: Index of ${desc.display}'));
  }
  final Object? stored = elem.kind == 'interface' ? x : x?.value;
  final Object? v = value.value;
  if (v is Slice<Object?>) {
    final Object? held = v.$at(exactIndex);
    if (_holdsValue(elem) && held != null) {
      _assignInto(held, stored, elem);
      return;
    }
    v.$set(exactIndex, stored);
    return;
  }
  if (v is $ZeroList<Object?>) {
    _elemAt(v, exactIndex, desc);
    return;
  }
  if (v is List<Object?>) {
    if (index < 0 || index >= v.length) $panicIndex($gi(index), v.length);
    final Object? held = v[index];
    if (_holdsValue(elem) && held != null) {
      _assignInto(held, stored, elem);
      return;
    }
    v[index] = stored;
    return;
  }
  $panic($runtimeError('reflect: Set of an element of ${desc.display}'));
}

Object? _elemAt(Object? v, GoInt exactIndex, TypeDesc desc) {
  if (v is $ZeroList<Object?>) {
    if (exactIndex < GoInt.zero || exactIndex >= desc.length) {
      $panicIndex(exactIndex, desc.length);
    }
    return v.value;
  }
  final index = $ix(exactIndex);
  if (v is Slice<Object?>) {
    if (index < 0 || index >= v.len) $panicIndex($gi(index), v.len);
    return v.array[v.off + index];
  }
  if (v is List<Object?>) {
    if (index < 0 || index >= v.length) $panicIndex($gi(index), v.length);
    return v[index];
  }
  $panic($runtimeError('reflect: Index of ${desc.display}'));
}

final TypeDesc _byteDesc = TypeDesc(
  'uint8',
  'uint8',
  const <String, Function>{},
  text: 'uint8',
  zero: () => 0,
);

/// The whole number inside a value of an integer kind.
GoInt $asInt(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is GoInt) return v;
  $panic(
    $runtimeError('reflect: the value of ${value.desc.display} is no integer'),
  );
}

/// The float inside a value of a float kind.
GoFloat64 $asFloat(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is GoFloat32) return $f64(v.toDouble());
  if (v is GoFloat64) return v;
  if (v is GoInt) return $f64(v.toDouble());
  $panic(
    $runtimeError('reflect: the value of ${value.desc.display} is no float'),
  );
}

/// The flag inside a value of the kind bool.
bool $asBool(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is bool) return v;
  $panic(
    $runtimeError('reflect: the value of ${value.desc.display} is no bool'),
  );
}

/// The text inside a value of the kind string.
GoString $asString(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is GoString) return v;
  $panic(
    $runtimeError('reflect: the value of ${value.desc.display} is no string'),
  );
}

/// A value of the type, with the whole number inside it.
Iface? $withInt(int handle, GoInt x) {
  final desc = _typeAt(handle);
  return Iface(desc, _integerValue(x, desc.kind));
}

GoInt _integerValue(GoInt x, String kind) => switch (kind) {
  'int8' => $i8(x),
  'int16' => $i16(x),
  'int32' => $i32(x),
  'uint8' => $u8(x),
  'uint16' => $u16(x),
  'uint32' => $u32(x),
  _ => x,
};

/// A value of the type, with the float inside it.
Iface? $withFloat(int handle, GoFloat64 x) {
  final type = _typeAt(handle);
  return Iface(type, type.kind == 'float32' ? $f32(x.toDouble()) : x);
}

/// A value of the type, with the flag inside it.
Iface? $withBool(int handle, bool x) => Iface(_typeAt(handle), x);

/// A value of the type, with the text inside it.
Iface? $withString(int handle, GoString x) => Iface(_typeAt(handle), x);

TypeDesc _typeAt(int handle) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null) {
    $panic($runtimeError('reflect: the handle names no type'));
  }
  return desc;
}

/// A pointer to a fresh zero value of the type.
Iface? $newValue(int handle) {
  final TypeDesc desc = _typeAt(handle);
  final Object? Function()? zero = desc.zero;
  if (zero == null) {
    $panic($runtimeError('reflect: New of ${desc.display}'));
  }
  // The Dart form of a pointer to a struct and of a pointer to an array is
  // the value itself. Each other pointer holds a cell of the Dart type of the
  // value, thus a type assertion of the result reads.
  if (_isObjectPointer(desc)) return Iface($ptrDesc(desc), zero());
  return Iface(
    $ptrDesc(desc),
    _withType<$Ptr<Object?>>(desc, <T>() => Box<T>(zero() as T)),
  );
}

/// True when the Dart form of a pointer to the type is the value itself.
bool _isObjectPointer(TypeDesc desc) =>
    desc.kind == 'struct' || desc.kind == 'array';

/// Calls the function with the Dart type of a value of the type.
///
/// A slice, a map, a channel, a function, and a pointer each read the Dart
/// type of the element, and stage 5 writes the factory for those 5. The kind
/// alone names the Dart type of each other type.
R _withType<R>(TypeDesc desc, R Function<T>() f) {
  final X Function<X>(X Function<Y>())? make = desc.withType;
  if (make != null) return make<R>(f);
  return _byKind<R>(desc.kind, f);
}

/// Calls the function with the Dart type that the kind names.
///
/// The Dart type of a value lives in the emitted code, thus a cell of Object?
/// fits no typed location. Each kind below holds 1 Dart type, and the runtime
/// builds the typed cell for it.
R _byKind<R>(String kind, R Function<T>() f) {
  switch (kind) {
    case 'bool':
      return f<bool>();
    case 'string':
      return f<GoString>();
    case 'float32':
      return f<GoFloat32>();
    case 'float64':
      return f<GoFloat64>();
    case 'complex64' || 'complex128':
      return f<GoComplex>();
    case 'interface':
      return f<Iface?>();
    case 'int' ||
        'int8' ||
        'int16' ||
        'int32' ||
        'int64' ||
        'uint' ||
        'uint8' ||
        'uint16' ||
        'uint32' ||
        'uint64' ||
        'uintptr':
      return f<GoInt>();
  }
  return f<Object?>();
}

/// A pointer to a fresh cell that holds the value, with the Dart type of the
/// element.
///
/// A cell of Object? fits no typed location: the setter of a field of the type
/// `*float64` casts to `$Ptr<GoFloat64>?`, and Dart rejects a `Box<Object?>` there.
/// The description of the element names the Dart type, thus the cell fits. The
/// decoder of encoding/json reads this function for a pointer field.
$Ptr<Object?> $typedBox(TypeDesc elem, Object? value) =>
    _withType<$Ptr<Object?>>(elem, <T>() => Box<T>(value as T));

/// A pointer to a location that a getter and a setter name, with the Dart type
/// that the kind of the value names.
$Ptr<Object?> _typedCell(
  TypeDesc desc,
  Object? owner,
  int index,
  Object? Function(Object?, int) get,
  void Function(Object?, int, Object?) set,
) => _withType<$Ptr<Object?>>(
  desc,
  <T>() => $Ref<Object?, T>(
    owner,
    index,
    (Object? o, int i) => get(o, i) as T,
    (Object? o, int i, T v) => set(o, i, v),
  ),
);

/// A slice of the type, with the length and the capacity.
Iface? $makeValue(int handle, GoInt length, GoInt capacity) {
  final TypeDesc desc = _typeAt(handle);
  final Object? Function(int)? make = desc.make;
  if (make == null) {
    $panic($runtimeError('reflect: Make of ${desc.display}'));
  }
  final empty = make(0);
  if (empty is $ZeroSlice<Object?>) {
    return Iface(desc, empty.$viewExact(GoInt.zero, length, capacity));
  }
  final size = $typeSize(desc.elem!());
  // The logical Go heap uses 48 address bits. Divide before multiplication.
  if (capacity > $udiv($gi(281474976710656), size)) {
    $panic($runtimeError('runtime: allocation size out of range', plain: true));
  }
  final Object? out = make($ix(capacity));
  if (out is Slice<Object?> && capacity != length) {
    return Iface(desc, out.range(GoInt.zero, length, capacity));
  }
  return Iface(desc, out);
}

final Map<String, TypeDesc> _pointers = <String, TypeDesc>{};

/// The description of a pointer to the type.
///
/// A generic function holds no variable for the description of a pointer to
/// its type parameter. The emitted code calls this function with the
/// description of the type argument.
TypeDesc $ptrDesc(TypeDesc elem) {
  // The emitter writes the description of the pointer when that pointer holds
  // a method that the value type does not hold. That description carries the
  // method table, thus Implements of it answers as Go does.
  final TypeDesc? held = elem.ptr?.call();
  if (held != null) return held;
  final TypeDesc? found = _pointers[elem.name];
  if (found != null) return found;
  final methods = _pointerMethods(elem);
  final TypeDesc out = TypeDesc(
    '*${elem.name}',
    'ptr',
    methods,
    text: '*${elem.display}',
    assertionPkg: methods.isEmpty ? '' : elem.assertionPkg,
    methodPkg: _unnamedType(elem.display) ? '' : elem.methodPkg,
    elem: () => elem,
    methodTypes: elem.kind == 'ptr' || elem.kind == 'interface'
        ? const <String, TypeDesc Function()>{}
        : elem.methodTypes,
    // The Dart type of the pointer comes from the Dart type of the value. A
    // description with no such type gives a cell of Object?, and no typed
    // location takes it: the setter of a field of the type `**int` casts to
    // `$Ptr<$Ptr<GoInt>?>?`, and Dart rejects a `Box<Object?>` there.
    withType: <R>(R Function<T>() f) => _withType<R>(
      elem,
      <E>() => _isObjectPointer(elem) ? f<E?>() : f<$Ptr<E>?>(),
    ),
    zero: () => null,
  );
  _pointers[elem.name] = out;
  return out;
}

/// _pointerMethods gives the method table of a pointer to a type that the
/// emitter wrote no pointer description for.
///
/// Go gives *T each method of T, and the adapter of T takes the value as the
/// receiver. The Dart form of a pointer to a struct and of a pointer to an
/// array is the value itself, thus the adapter of T reads it with no change.
/// Each other pointer holds a cell, thus the adapter reads the value out of the
/// cell first.
Map<String, Function> _pointerMethods(TypeDesc elem) {
  if (elem.kind == 'ptr' || elem.kind == 'interface') {
    return const <String, Function>{};
  }
  if (elem.methods.isEmpty) return const <String, Function>{};
  if (_isObjectPointer(elem)) return elem.methods;
  final Map<String, Function> out = <String, Function>{};
  for (final MapEntry<String, Function> e in elem.methods.entries) {
    final TypeDesc Function()? type = elem.methodTypes[e.key];
    final Function fn = e.value;
    if (type == null) {
      // An unexported method carries no type, thus the result stays dynamic.
      // Only a call inside the package that declares the type reaches such a
      // method, and no such call reads this table today.
      out[e.key] = _looseFunc(
        (List<Object?> args) => <Object?>[
          Function.apply(fn, <Object?>[
            args.isEmpty ? null : _cellValue(args[0]),
            ...args.skip(1),
          ]),
        ],
        1,
      );
      continue;
    }
    final TypeDesc sig = type();
    out[e.key] = _typedResultFunc(
      sig,
      (List<Object?> args) => $resumeReflect<List<Object?>>(
        Function.apply(fn, <Object?>[
          args.isEmpty ? null : _cellValue(args[0]),
          ...args.skip(1),
        ]),
        (Object? out) => _resultList(sig, out),
        <Object?>[],
      ),
      sig.params.length + 1,
    );
  }
  return out;
}

/// _cellValue reads the value that a cell holds. A pointer that is not the
/// object itself holds such a cell.
Object? _cellValue(Object? p) => p is $Ptr<Object?> ? p.v : p;

/// The value of a pointer to a type parameter, in the form of the type
/// argument.
///
/// A generic function holds no type argument, thus the address of a value
/// there writes a cell. The Dart form of a pointer to a struct and of a
/// pointer to an array is the value itself. An interface value of such a
/// pointer thus reads the cell out.
Object? $ptrValue(TypeDesc elem, Object? pointer) {
  if (!_isObjectPointer(elem)) return pointer;
  return pointer is $Ptr<Object?> ? pointer.v : pointer;
}

/// The count of the parameters of a function type.
int $descNumIn(int handle) => _typeAt(handle).params.length;

/// The count of the results of a function type.
int $descNumOut(int handle) => _typeAt(handle).results.length;

/// A handle of the type of 1 parameter of a function type.
int $descIn(int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  if (index < 0 || index >= desc.params.length) {
    $panic($runtimeError('reflect: the type holds no such parameter'));
  }
  return _descHandle(desc.params[index]());
}

/// A handle of the type of 1 result of a function type.
int $descOut(int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  if (index < 0 || index >= desc.results.length) {
    $panic($runtimeError('reflect: the type holds no such result'));
  }
  return _descHandle(desc.results[index]());
}

/// Calls a function value with the arguments, and gives each result.
///
/// A suspended function retains its result conversion in the frame chain.
Slice<Iface?> $callValue(Iface? fn, Slice<Iface?> args) =>
    _callValue(fn, args, false);

/// Calls a variadic function value and takes the last argument as the slice of
/// the variadic parameter.
///
/// Go assigns that slice itself, thus the function reads the same backing array
/// as the caller. A pack of the tail would give a copy, and the 2 meanings
/// differ.
Slice<Iface?> $callSliceValue(Iface? fn, Slice<Iface?> args) =>
    _callValue(fn, args, true);

Slice<Iface?> _callValue(Iface? fn, Slice<Iface?> args, bool spread) {
  if (fn == null) $panicNil();
  final TypeDesc desc = fn.desc;
  if (desc.kind != 'func') {
    $panic($runtimeError('reflect: Call of ${desc.display}'));
  }
  if (fn.value == null) {
    $panic(
      $runtimeError('reflect.Value.Call: call of nil function', plain: true),
    );
  }
  if (spread && !desc.variadic) {
    $panic(
      $runtimeError(
        'reflect: CallSlice of ${desc.display}, a type that is not '
        'variadic',
      ),
    );
  }
  if (spread) {
    // The last argument already holds the slice of the variadic parameter.
    final List<Object?> raw = <Object?>[];
    for (int i = 0; i < args.len; i++) {
      final Iface? a = args[i];
      raw.add(desc.params[i]().kind == 'interface' ? a : a?.value);
    }
    return _invokeReflect(fn, raw);
  }
  final int fixed = desc.variadic ? desc.params.length - 1 : desc.params.length;
  final bool wrong = desc.variadic
      ? args.len < fixed
      : args.len != desc.params.length;
  if (wrong) {
    $panic(
      $runtimeError(
        'reflect: Call with ${args.len} input arguments, '
        'and the function takes ${desc.params.length}',
      ),
    );
  }
  final List<Object?> raw = <Object?>[];
  for (int i = 0; i < fixed; i++) {
    final Iface? a = args[i];
    raw.add(desc.params[i]().kind == 'interface' ? a : a?.value);
  }
  if (desc.variadic) {
    // The Dart form of a variadic parameter is a slice, thus the call packs
    // each argument after the fixed ones.
    final TypeDesc rest = desc.params[fixed]();
    final TypeDesc? elem = rest.elem?.call();
    final Object? Function(int)? make = rest.make;
    if (make == null) {
      $panic($runtimeError('reflect: Call of ${desc.display}'));
    }
    final Slice<Object?> tail = make(args.len - fixed) as Slice<Object?>;
    for (int i = fixed; i < args.len; i++) {
      final Iface? a = args[i];
      tail[i - fixed] = elem != null && elem.kind == 'interface' ? a : a?.value;
    }
    raw.add(tail);
  }
  return _invokeReflect(fn, raw);
}

// Weak keys let an unreachable function release its captured Go values.
final Expando<List<Object?> Function(List<Object?>)> _reflectFunctions =
    Expando<List<Object?> Function(List<Object?>)>();

Slice<Iface?> _invokeReflect(Iface fn, List<Object?> args) {
  final function = fn.value as Function;
  final body = _reflectFunctions[function];
  return $resumeReflect<Slice<Iface?>>(
    body == null ? Function.apply(function, args) : body(args),
    (Object? out) => _resultValues(
      fn.desc,
      body == null ? _resultList(fn.desc, out) : out as List<Object?>,
    ),
    Slice<Iface?>.nil(),
  );
}

/// Retains a result conversion across a suspended reflection call.
T $resumeReflect<T>(Object? out, T Function(Object?) convert, T zero) {
  final Frame? frame = $blocked;
  if (frame == null) return convert(out);
  $block(_ReflectResume(frame, convert));
  return zero;
}

final class _ReflectResume implements WaitFrame, DurableWait, NestedFrame {
  Frame inner;
  final Object? Function(Object?) convert;
  _ReflectResume(this.inner, this.convert);

  @override
  Frame get $inner => inner;
  @override
  set $inner(Frame frame) {
    inner = frame;
  }

  @override
  String get $reason => $waitReason(inner);
  @override
  bool $durableIn(Bubble b) => $durableWait(inner, b);

  @override
  Object? resume() {
    final Object? out = $resumeFrame(inner);
    final Frame? again = $blocked;
    if (again != null) {
      $block(_ReflectResume(again, convert));
      return null;
    }
    return convert(out);
  }
}

Slice<Iface?> _resultValues(TypeDesc desc, List<Object?> values) =>
    Slice.of(<Iface?>[
      for (int i = 0; i < desc.results.length; i++)
        _result(desc.results[i](), values[i]),
    ]);

Iface? _result(TypeDesc desc, Object? value) {
  if (desc.kind == 'interface') return value as Iface?;
  return Iface(desc, value);
}

/// Reads 1 field of the record of a call that gives many results. Dart holds no
/// index of a record, thus the count of the results selects the field.
Object? _field(Object? record, int index, int count) {
  final dynamic r = record;
  switch (index) {
    case 0:
      return r.$1;
    case 1:
      return r.$2;
    case 2:
      return r.$3;
    case 3:
      return r.$4;
    case 4:
      return r.$5;
    case 5:
      return r.$6;
    case 6:
      return r.$7;
    case 7:
      return r.$8;
  }
  $panic(
    $runtimeError('reflect: a call of $count results is outside the subset'),
  );
}

List<ReflectedMethod> _reflectedMethods(
  TypeDesc desc, {
  bool includePrivate = false,
}) {
  if (desc.reflectionMethods != null) return desc.reflectionMethods!;
  final keys = desc.methods.keys
      .where(
        (key) =>
            includePrivate ||
            desc.kind == 'interface' ||
            _methodPackage(key).isEmpty,
      )
      .toList();
  keys.sort((a, b) {
    final ap = _methodPackage(a), bp = _methodPackage(b);
    if (ap.isEmpty != bp.isEmpty) return ap.isEmpty ? -1 : 1;
    final order = $compare(
      GoString.fromString(_methodName(a)),
      GoString.fromString(_methodName(b)),
    );
    return order == 0
        ? $compare(GoString.fromString(ap), GoString.fromString(bp))
        : order;
  });
  return <ReflectedMethod>[
    for (final key in keys)
      ReflectedMethod(
        key,
        desc.assertionPkg == 'main' && _methodPackage(key) == desc.methodPkg
            ? 'main'
            : _methodPackage(key),
        desc.methodTypes[key],
        (Iface value, List<Object?> args) => $call(value, key, args),
      ),
  ];
}

String _methodPackage(String key) {
  final int cut = key.indexOf('(');
  final String name = cut < 0 ? key : key.substring(0, cut);
  final int dot = name.lastIndexOf('.');
  return dot < 0 ? '' : name.substring(0, dot);
}

String _methodName(String key) {
  final int cut = key.indexOf('(');
  final String name = cut < 0 ? key : key.substring(0, cut);
  return name.substring(name.lastIndexOf('.') + 1);
}

/// The method count for reflection.
int $descNumMethod(int handle) => _reflectedMethods(_typeAt(handle)).length;

/// A handle of the function type of 1 reflected method of a type. A method that
/// carries no type gives 0.
///
/// Go puts the receiver in front of each parameter of the type of a method of a
/// concrete type, and it writes no receiver for a method of an interface type.
int $descMethodType(int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  final methods = _reflectedMethods(desc);
  if (index < 0 || index >= methods.length) return 0;
  final TypeDesc Function()? type = methods[index].type;
  if (type == null) return 0;
  final TypeDesc sig = type();
  if (desc.kind == 'interface') return _descHandle(sig);
  return _descHandle(_withReceiver(desc, sig));
}

/// The func value of 1 exported method of a type, with the receiver in front of
/// each parameter. A type of an interface holds no such value, exactly as in Go.
Iface? $descMethodFunc(int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  if (desc.kind == 'interface') return null;
  final methods = _reflectedMethods(desc);
  if (index < 0 || index >= methods.length) {
    $panic($runtimeError('reflect: ${desc.display} holds no method $index'));
  }
  final method = methods[index];
  final String key = method.key;
  final TypeDesc Function()? type = method.type;
  if (type == null) {
    $panic(
      $runtimeError('reflect: no type of the method $key of ${desc.display}'),
    );
  }
  final TypeDesc full = _withReceiver(desc, type());
  return Iface(
    full,
    _typedFunc(
      full,
      (List<Object?> args) => $resumeReflect<List<Object?>>(
        method.invoke(Iface(desc, args[0]), args.sublist(1)),
        (Object? out) => _resultList(full, out),
        <Object?>[],
      ),
    ),
  );
}

TypeDesc _withReceiver(TypeDesc recv, TypeDesc sig) => TypeDesc(
  _funcText(recv.name, sig, (TypeDesc d) => d.name),
  'func',
  const <String, Function>{},
  text: _funcText(recv.display, sig, (TypeDesc d) => d.display),
  params: <TypeDesc Function()>[() => recv, ...sig.params],
  results: sig.results,
  unpack: sig.unpack,
  variadic: sig.variadic,
  comparable: false,
  zero: () => null,
);

/// _funcText writes a function type in the form of Go. recv stands in front of
/// each parameter, and it is empty for a function type that holds no receiver.
String _funcText(String recv, TypeDesc sig, String Function(TypeDesc) name) {
  final StringBuffer b = StringBuffer('func(')..write(recv);
  var first = recv.isEmpty;
  for (int i = 0; i < sig.params.length; i++) {
    if (first) {
      first = false;
    } else {
      b.write(', ');
    }
    final TypeDesc p = sig.params[i]();
    if (sig.variadic && i == sig.params.length - 1) {
      final TypeDesc? e = p.elem?.call();
      b.write('...${name(e ?? p)}');
      continue;
    }
    b.write(name(p));
  }
  b.write(')');
  if (sig.results.length == 1) {
    b.write(' ${name(sig.results[0]())}');
    return b.toString();
  }
  if (sig.results.isEmpty) return b.toString();
  b.write(' (');
  for (int i = 0; i < sig.results.length; i++) {
    if (i > 0) b.write(', ');
    b.write(name(sig.results[i]()));
  }
  b.write(')');
  return b.toString();
}

/// Drops each entry of a map, or writes the zero value into each element of a
/// slice.
void $valueClear(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is GoMap<Object?, Object?>) {
    // A read of each key would drop no NaN key, because a NaN never compares
    // equal to itself. clear walks the entries themselves.
    v.clear();
    return;
  }
  if (v is $ZeroSlice<Object?>) return;
  if (v is Slice<Object?>) {
    final elem = value.desc.elem!();
    $clearSlice<Object?>(
      v,
      () => _zeroOf(elem),
      _holdsValue(elem) ? (a, b) => _assignInto(a, b, elem) : null,
    );
    return;
  }
  $panic($runtimeError('reflect: Clear of ${value.desc.display}'));
}

/// The method name at index in reflection order.
GoString $descMethodName(int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  final methods = _reflectedMethods(desc);
  if (index < 0 || index >= methods.length) {
    $panic($runtimeError('reflect: ${desc.display} holds no method $index'));
  }
  return GoString.fromString(_methodName(methods[index].key));
}

GoString $descMethodPkgPath(int handle, int index) =>
    GoString.fromString(_reflectedMethods(_typeAt(handle))[index].pkg);

/// Binds the method at index in the static type to the dynamic value.
Iface? $methodValue(Iface? value, int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  final methods = _reflectedMethods(desc);
  if (index < 0 || index >= methods.length) {
    $panic($runtimeError('reflect: ${desc.display} holds no method $index'));
  }
  if (value == null) {
    $panic(
      Iface(
        $typeString,
        GoString.fromString('reflect: Method on nil interface value'),
      ),
    );
  }
  final method = methods[index];
  final String key = method.key;
  final TypeDesc Function()? type = method.type;
  if (type == null) {
    $panic(
      $runtimeError('reflect: no type of the method $key of ${desc.display}'),
    );
  }
  final TypeDesc sig = type();
  return Iface(
    sig,
    $functionValue(
      _bindMethod(sig, method, value),
      GoString.fromString('reflect.methodValueCall||0'),
    ),
  );
}

/// _bindMethod gives a Dart closure of a method and its receiver.
///
/// The closure carries the Dart type of each parameter and of the result,
/// because the signature description holds a factory for the exact Dart type.
/// A Go program that asserts the value to its function type then reads the
/// result.
Function _bindMethod(TypeDesc sig, ReflectedMethod method, Iface value) =>
    _typedFunc(
      sig,
      (List<Object?> args) => $resumeReflect<List<Object?>>(
        method.invoke(value, args),
        (Object? out) => _resultList(sig, out),
        <Object?>[],
      ),
    );

/// Adapts raw reflection arguments and results to a callable function.
Function _typedFunc(TypeDesc sig, List<Object?> Function(List<Object?>) body) {
  final Function Function(List<Object?> Function(List<Object?>))? bind =
      sig.bind ?? _typeFactories[sig.name]?.call().bind;
  if (bind != null) return bind(body);
  if (sig.params.length > 6 || sig.results.length > 4) {
    List<Object?> function(List<Object?> args) => body(args);
    _reflectFunctions[function] = body;
    return function;
  }
  return _typedResultFunc(sig, body, sig.params.length);
}

/// _typedResultFunc gives a closure whose result type is the Dart type of the
/// results of a signature.
///
/// A Dart function type reads a parameter against the other direction, thus a
/// closure of Object? parameters fits each function type of the same arity. A
/// result stands the same way as the function, thus a closure that gives
/// Object? fits no location of a result type. The report of such a cast names
/// the function type of the closure and the function type of the location.
///
/// The description of each result names its Dart type, thus the closure below
/// carries the exact type.
Function _typedResultFunc(
  TypeDesc sig,
  List<Object?> Function(List<Object?>) body,
  int arity,
) {
  final List<TypeDesc Function()> rs = sig.results;
  switch (rs.length) {
    case 0:
      // A Dart location of a void result accepts each other result type.
      return _looseFunc(body, arity);
    case 1:
      return _withType<Function>(
        rs[0](),
        <A>() => _arityFunc<A>(
          (List<Object?> a) => $resumeReflect<A>(
            body(a),
            (Object? out) => (out as List<Object?>)[0] as A,
            _zeroOf(rs[0]()) as A,
          ),
          arity,
        ),
      );
    case 2:
      return _withType<Function>(
        rs[0](),
        <A>() => _withType<Function>(
          rs[1](),
          <B>() => _arityFunc<(A, B)>(
            (List<Object?> a) => $resumeReflect<(A, B)>(body(a), (Object? out) {
              final List<Object?> o = out as List<Object?>;
              return (o[0] as A, o[1] as B);
            }, (_zeroOf(rs[0]()) as A, _zeroOf(rs[1]()) as B)),
            arity,
          ),
        ),
      );
    case 3:
      return _withType<Function>(
        rs[0](),
        <A>() => _withType<Function>(
          rs[1](),
          <B>() => _withType<Function>(
            rs[2](),
            <C>() => _arityFunc<(A, B, C)>(
              (List<Object?> a) => $resumeReflect<(A, B, C)>(
                body(a),
                (Object? out) {
                  final List<Object?> o = out as List<Object?>;
                  return (o[0] as A, o[1] as B, o[2] as C);
                },
                (
                  _zeroOf(rs[0]()) as A,
                  _zeroOf(rs[1]()) as B,
                  _zeroOf(rs[2]()) as C,
                ),
              ),
              arity,
            ),
          ),
        ),
      );
    default:
      return _withType<Function>(
        rs[0](),
        <A>() => _withType<Function>(
          rs[1](),
          <B>() => _withType<Function>(
            rs[2](),
            <C>() => _withType<Function>(
              rs[3](),
              <D>() => _arityFunc<(A, B, C, D)>(
                (List<Object?> a) => $resumeReflect<(A, B, C, D)>(
                  body(a),
                  (Object? out) {
                    final List<Object?> o = out as List<Object?>;
                    return (o[0] as A, o[1] as B, o[2] as C, o[3] as D);
                  },
                  (
                    _zeroOf(rs[0]()) as A,
                    _zeroOf(rs[1]()) as B,
                    _zeroOf(rs[2]()) as C,
                    _zeroOf(rs[3]()) as D,
                  ),
                ),
                arity,
              ),
            ),
          ),
        ),
      );
  }
}

/// _arityFunc gives a closure of the count of the parameters, and of the result
/// type R. Each parameter takes the type Object?, which fits each function type
/// of that arity.
Function _arityFunc<R>(R Function(List<Object?>) make, int arity) {
  switch (arity) {
    case 0:
      return () => make(const <Object?>[]);
    case 1:
      return (Object? a) => make(<Object?>[a]);
    case 2:
      return (Object? a, Object? b) => make(<Object?>[a, b]);
    case 3:
      return (Object? a, Object? b, Object? c) => make(<Object?>[a, b, c]);
    case 4:
      return (Object? a, Object? b, Object? c, Object? d) =>
          make(<Object?>[a, b, c, d]);
    case 5:
      return (Object? a, Object? b, Object? c, Object? d, Object? e) =>
          make(<Object?>[a, b, c, d, e]);
    default:
      return (
        Object? a,
        Object? b,
        Object? c,
        Object? d,
        Object? e,
        Object? f,
      ) => make(<Object?>[a, b, c, d, e, f]);
  }
}

/// _resultList gives the raw results of a call as a list. A call of more than 1
/// result gives a Dart record, thus the list reads each member of it.
List<Object?> _resultList(TypeDesc sig, Object? out) {
  if (sig.unpack != null) return sig.unpack!(out);
  final int n = sig.results.length;
  if (n == 0) return const <Object?>[];
  if (n == 1) return <Object?>[out];
  return <Object?>[for (int i = 0; i < n; i++) _field(out, i, n)];
}

/// _looseFunc gives a closure of Object? parameters. Dart reads a parameter
/// type against the other direction, thus such a closure fits each function
/// type of the same arity. The result stays dynamic.
Function _looseFunc(List<Object?> Function(List<Object?>) body, int arity) {
  switch (arity) {
    case 0:
      return () => _onlyResult(body(const <Object?>[]));
    case 1:
      return (Object? a) => _onlyResult(body(<Object?>[a]));
    case 2:
      return (Object? a, Object? b) => _onlyResult(body(<Object?>[a, b]));
    case 3:
      return (Object? a, Object? b, Object? c) =>
          _onlyResult(body(<Object?>[a, b, c]));
    case 4:
      return (Object? a, Object? b, Object? c, Object? d) =>
          _onlyResult(body(<Object?>[a, b, c, d]));
    case 5:
      return (Object? a, Object? b, Object? c, Object? d, Object? e) =>
          _onlyResult(body(<Object?>[a, b, c, d, e]));
    default:
      return (
        Object? a,
        Object? b,
        Object? c,
        Object? d,
        Object? e,
        Object? f,
      ) => _onlyResult(body(<Object?>[a, b, c, d, e, f]));
  }
}

/// _onlyResult gives the single result of a call, or the whole list.
Object? _onlyResult(List<Object?> out) => out.length == 1 ? out[0] : out;

/// A func value of a function type. Each call of it runs the body with the
/// arguments, and the results of the body become the results of the call.
///
/// The signature supplies a typed bind factory when the compiler emits one.
/// Runtime-created signatures use their result descriptors.
Iface? $makeFunc(int handle, Object? body) {
  final TypeDesc desc = _typeAt(handle);
  if (desc.kind != 'func') {
    $panic($runtimeError('reflect: MakeFunc of ${desc.display}'));
  }
  final Function run = body! as Function;
  return Iface(
    desc,
    _typedFunc(desc, (List<Object?> args) {
      final List<Iface?> input = <Iface?>[];
      for (int i = 0; i < args.length && i < desc.params.length; i++) {
        final TypeDesc p = desc.params[i]();
        input.add(
          p.kind == 'interface' ? args[i] as Iface? : Iface(p, args[i]),
        );
      }
      return $resumeReflect<List<Object?>>(run(Slice<Iface?>.of(input)), (
        Object? value,
      ) {
        final Slice<Iface?> out = value as Slice<Iface?>;
        final int n = desc.results.length;
        final List<Object?> raw = <Object?>[];
        for (int i = 0; i < n; i++) {
          final Iface? o = out[i];
          raw.add(desc.results[i]().kind == 'interface' ? o : o?.value);
        }
        return raw;
      }, <Object?>[]);
    }),
  );
}

/// True when the value of a kind that can hold nil holds nil. A pointer, a
/// func, an interface, and a channel each hold null. A slice and a map each
/// carry a nil flag, because Go reads the length of a nil one.
bool $valueIsNil(Iface? value) {
  if (value == null) return true;
  final Object? v = value.value;
  if (v == null) return true;
  if (v is Slice<Object?>) return v.isNil;
  if (v is GoMap<Object?, Object?>) return v.isNil;
  // The zero value of a channel type is an object of its own, and that object
  // carries the nil flag.
  if (v is Chan<Object?>) return v.isNil;
  return false;
}

/// The keys of a map value, in the order of the insertion.
Slice<Iface?> $mapKeys(Iface? value) {
  if (value == null) $panicNil();
  final Object? m = value.value;
  if (m is! GoMap<Object?, Object?>) {
    $panic($runtimeError('reflect: MapKeys of ${value.desc.display}'));
  }
  final TypeDesc? keyType = value.desc.key?.call();
  final List<Iface?> out = <Iface?>[];
  final Slice<Object?> keys = m.keys();
  for (int i = 0; i < keys.len; i++) {
    final Object? k = keys[i];
    out.add(
      keyType == null || keyType.kind == 'interface'
          ? k as Iface?
          : Iface(keyType, k),
    );
  }
  return Slice.of(out);
}

/// The map that a value holds. The name of the caller stands in the panic.
GoMap<Object?, Object?> _mapOf(Iface? value, String name) {
  if (value == null) $panicNil();
  final Object? m = value.value;
  if (m is! GoMap<Object?, Object?>) {
    $panic($runtimeError('reflect: $name of ${value.desc.display}'));
  }
  return m;
}

/// The value of 1 pair of a map, with the description of the element type.
Iface? _mapPart(Object? v, TypeDesc? desc) {
  if (desc == null || desc.kind == 'interface') return v as Iface?;
  return Iface(desc, v);
}

/// The description of the handle that MapRange gives. The handle crosses to
/// the overlay as a value of the Go type any, thus it needs a description.
final TypeDesc $typeMapIter = TypeDesc(
  'reflect.mapIter',
  'ptr',
  const <String, Function>{},
  text: '*reflect.mapIter',
  zero: () => null,
);

/// The iterator of the pairs of a map value. The iterator reads the map
/// itself, thus a clear inside the loop ends the loop, and Go holds the same
/// rule. A snapshot of the keys would walk each entry that the clear dropped,
/// and a NaN key gives no second read.
Iface $mapIterOf(Iface? value) =>
    Iface($typeMapIter, GoMapIter<Object?, Object?>(_mapOf(value, 'MapRange')));

GoMapIter<Object?, Object?> _iterOf(Iface? it) =>
    it!.value! as GoMapIter<Object?, Object?>;

/// Steps the iterator, and reports whether a pair stands there.
bool $mapIterNext(Iface? it) => _iterOf(it).next();

/// The key of the pair that the iterator holds.
Iface? $mapIterKey(Iface? value, Iface? it) =>
    _mapPart(_iterOf(it).key, value!.desc.key?.call());

/// The value of the pair that the iterator holds. The read reaches the entry,
/// thus a write of the loop body gives the new value.
Iface? $mapIterValue(Iface? value, Iface? it) =>
    _mapPart(_iterOf(it).value, value!.desc.elem?.call());

/// The value of 1 key of a map value. A key that the map does not hold gives
/// null, and the package reflect reads that as the zero Value.
Iface? $mapIndex(Iface? target, Iface? key) {
  if (target == null) $panicNil();
  final Object? m = target.value;
  if (m is! GoMap<Object?, Object?>) {
    $panic($runtimeError('reflect: MapIndex of ${target.desc.display}'));
  }
  final TypeDesc? keyType = target.desc.key?.call();
  final Object? k = keyType?.kind == 'interface' ? key : key?.value;
  if (!m.has(k)) return null;
  final TypeDesc? elem = target.desc.elem?.call();
  final Object? out = m.get(k);
  if (elem == null || elem.kind == 'interface') return out as Iface?;
  return Iface(elem, out);
}

/// Reports whether a map value holds the key.
///
/// An entry that holds a nil interface reads as nil, exactly as a key that the
/// map does not hold. reflect.Value.MapIndex needs the 2 answers apart, because
/// Go marks the first 1 valid and the second 1 the zero Value.
bool $mapHas(Iface? target, Iface? key) {
  if (target == null) $panicNil();
  final Object? m = target.value;
  if (m is! GoMap<Object?, Object?>) {
    $panic($runtimeError('reflect: MapIndex of ${target.desc.display}'));
  }
  final TypeDesc? keyType = target.desc.key?.call();
  return m.has(keyType?.kind == 'interface' ? key : key?.value);
}

/// A map of the type.
Iface? $makeMap(int handle) {
  final TypeDesc desc = _typeAt(handle);
  final Object? Function(int)? make = desc.make;
  if (make == null) {
    $panic($runtimeError('reflect: MakeMap of ${desc.display}'));
  }
  return Iface(desc, make(0));
}

/// A handle of the key type of a map type. Each other type gives 0.
int $descKey(int handle) {
  final TypeDesc? desc = _descAt(handle);
  final TypeDesc Function()? key = desc?.key;
  if (key == null) return 0;
  return _descHandle(key());
}

/// Writes 1 entry of a map. A nil value drops the entry.
void $mapSet(Iface? target, Iface? key, Iface? value, bool drop) {
  if (target == null) $panicNil();
  final Object? m = target.value;
  if (m is! GoMap<Object?, Object?>) {
    $panic($runtimeError('reflect: SetMapIndex of ${target.desc.display}'));
  }
  final TypeDesc? keyType = target.desc.key?.call();
  final TypeDesc? elem = target.desc.elem?.call();
  final Object? k = keyType?.kind == 'interface' ? key : key?.value;
  if (drop) {
    m.remove(k);
    return;
  }
  m.set(k, elem?.kind == 'interface' ? value : value?.value);
}

GoString $embeddedFieldError(int handle, int index, int count) {
  final TypeDesc type = _typeAt(handle);
  String error = '';
  final bool named = !_unnamedType(type.display) && type.methodPkg.isNotEmpty;
  if (type.kind == 'ptr' &&
      (type.elem!().kind == 'ptr' || type.elem!().kind == 'interface')) {
    error = 'reflect.StructOf: illegal embedded field type ${type.display}';
  } else if (type.kind != 'interface') {
    if (index > 0 && type.methods.isNotEmpty) {
      error = 'reflect: embedded type with methods not implemented if type is not first field';
    } else if (count > 1 && (named || type.methods.isNotEmpty)) {
      if (type.kind == 'ptr') {
        error = 'reflect: embedded type with methods not implemented if there is more than one field';
      } else if (_directInterface(type)) {
        error = 'reflect: embedded type with methods not implemented for non-pointer type';
      }
    }
  }
  if (error.isEmpty &&
      type.methods.keys.any(
        (key) =>
            _methodPackage(key).isNotEmpty &&
            _methodPackage(key) != type.methodPkg,
      )) {
    error =
        'reflect: embedded interface with unexported method(s) not implemented';
  }
  return GoString.fromString(error);
}

bool _directInterface(TypeDesc type) {
  switch (type.kind) {
    case 'ptr' || 'unsafe.Pointer' || 'map' || 'chan' || 'func':
      return true;
    case 'array':
      return type.length == $gi(1) && _directInterface(type.elem!());
    case 'struct':
      return $typeSize(type) == $gi(8) &&
          type.fields.any((field) => _directInterface(field.type()));
  }
  return false;
}

/// A handle of a struct type that a list of fields builds.
///
/// The generated code registers storage templates before Go initializers.
/// A shape with no source template uses a list of its fields.
int $structOf(
  Slice<GoString> names,
  Slice<GoString> packages,
  Slice<GoString> tags,
  Slice<GoString> quotedTags,
  Slice<GoInt> handles,
  Slice<bool> anon,
) {
  final int n = names.len;
  final List<TypeDesc> types = <TypeDesc>[
    for (int i = 0; i < n; i++) _typeAt($ix(handles[i])),
  ];
  final List<String> field = <String>[
    for (int i = 0; i < n; i++) names[i].toDartString(),
  ];
  final List<String> tag = <String>[
    for (int i = 0; i < n; i++) tags[i].toDartString(),
  ];
  final StringBuffer key = StringBuffer('struct{');
  final StringBuffer storageKey = StringBuffer('struct{');
  final StringBuffer show = StringBuffer(n == 0 ? 'struct {}' : 'struct { ');
  for (int i = 0; i < n; i++) {
    if (i > 0) {
      key.write('; ');
      storageKey.write('; ');
      show.write('; ');
    }
    if (anon[i]) {
      key.write('~');
      storageKey.write('~');
    }
    key.write(field[i]);
    storageKey.write(field[i]);
    if (packages[i].length != 0) {
      storageKey.write('@${packages[i].toDartString()}');
    }
    storageKey.write(' \$F$i');
    if (packages[i].length != 0) key.write('@${packages[i].toDartString()}');
    key.write(' ');
    if (!anon[i]) show.write('${field[i]} ');
    key.write(types[i].name);
    show.write(types[i].display);
    if (tag[i].isNotEmpty) {
      final quoted = quotedTags[i].toDartString();
      key.write(' $quoted');
      show.write(' $quoted');
    }
  }
  key.write('}');
  storageKey.write('}');
  final storage = _structStorage[storageKey.toString()]?.call(types);
  Object? getField(Object? value, int i) => storage == null
      ? (value as List<Object?>)[i]
      : storage.fields[i].get(value);
  void setField(Object? value, int i, Object? fieldValue) {
    if (storage == null) {
      (value as List<Object?>)[i] = fieldValue;
    } else {
      storage.fields[i].set(value, fieldValue);
    }
  }

  if (n > 0) show.write(' }');
  return _syntheticHandle(() {
    final methods = <String, Function>{};
    final methodTypes = <String, TypeDesc Function()>{};
    final forwards = <String, Iface? Function(Object?)>{};
    final reflected = <ReflectedMethod>[];
    for (int i = 0; i < n; i++) {
      if (!anon[i]) continue;
      final elem = types[i];
      Object? invoke(Iface value, ReflectedMethod method, List<Object?> args) {
        if (elem.kind == 'interface') {
          $panic(
            Iface(
              $typeString,
              GoString.fromString(
                'reflect: StructOf does not support methods of embedded interfaces',
              ),
            ),
          );
        }
        return method.invoke(Iface(elem, getField(value.value, i)), args);
      }

      for (final method in _reflectedMethods(elem, includePrivate: true)) {
        reflected.add(
          ReflectedMethod(
            method.key,
            '',
            method.type,
            (Iface value, List<Object?> args) => invoke(value, method, args),
          ),
        );
        if (_methodPackage(method.key).isNotEmpty) continue;
        methods.putIfAbsent(method.key, () => $ifaceMethod);
        methodTypes.putIfAbsent(method.key, () => method.type!);
        forwards.putIfAbsent(
          method.key,
          () => (Object? value) {
            if (elem.kind == 'interface') {
              $panic(
                Iface(
                  $typeString,
                  GoString.fromString(
                    'reflect: StructOf does not support methods of embedded interfaces',
                  ),
                ),
              );
            }
            return Iface(elem, getField(value, i));
          },
        );
      }
      if (elem.kind == 'ptr') {
        for (final method in _reflectedMethods(
          elem.elem!(),
          includePrivate: true,
        )) {
          reflected.add(
            ReflectedMethod(
              method.key,
              '',
              method.type,
              (Iface value, List<Object?> args) => $call(
                Iface(elem, getField(value.value, i)),
                method.key,
                args,
              ),
            ),
          );
        }
      }
    }
    late final TypeDesc desc;
    desc = TypeDesc(
      key.toString(),
      'struct',
      methods,
      methodTypes: methodTypes,
      forwardMethods: forwards,
      reflectionMethods: reflected,
      ptr: () => TypeDesc(
        '*${key.toString()}',
        'ptr',
        const <String, Function>{},
        text: '*${show.toString()}',
        elem: () => desc,
        zero: () => null,
      ),
      text: show.toString(),
      fields: <FieldDesc>[
        for (int i = 0; i < n; i++)
          FieldDesc(
            field[i],
            tag[i],
            packages[i].length == 0,
            () => types[i],
            (Object? r) => getField(r, i),
            (Object? r, Object? v) => setField(r, i, v),
            embedded: anon[i],
            pkg: packages[i].toDartString(),
          ),
      ],
      comparable: types.every((TypeDesc t) => t.comparable),
      zero:
          storage?.zero ??
          () => <Object?>[for (int i = 0; i < n; i++) types[i].zero?.call()],
      assign: storage?.assign,
      view: storage?.view,
      withType: storage?.withType,
    );
    return desc;
  });
}

/// A handle of the slice type of an element type.
int $sliceOf(int elem) {
  final TypeDesc e = _typeAt(elem);
  return _syntheticHandle(
    () => TypeDesc(
      '[]${e.name}',
      'slice',
      const <String, Function>{},
      text: '[]${e.display}',
      elem: () => e,
      comparable: false,
      zero: () => _withType<Object?>(e, <T>() => Slice<T>.nil()),
      make: (int n) =>
          _withType<Object?>(e, <T>() => Slice<T>.of(_zeros<T>(e, $gi(n)))),
      withType: <R>(R Function<T>() f) =>
          _withType<R>(e, <T>() => f<Slice<T>>()),
    ),
  );
}

/// A handle of the function type of a parameter list and a result list.
///
/// An existing signature retains its descriptor and typed bind factory.
int $funcOf(Slice<GoInt> params, Slice<GoInt> results, bool variadic) {
  final List<TypeDesc Function()> input = <TypeDesc Function()>[];
  for (int i = 0; i < params.len; i++) {
    final TypeDesc d = _typeAt($ix(params[i]));
    input.add(() => d);
  }
  final List<TypeDesc Function()> output = <TypeDesc Function()>[];
  for (int i = 0; i < results.len; i++) {
    final TypeDesc d = _typeAt($ix(results[i]));
    output.add(() => d);
  }
  // The name of the type reads the parts of the type, thus a shape carries
  // them before the description itself stands.
  final TypeDesc shape = TypeDesc(
    '',
    'func',
    const <String, Function>{},
    params: input,
    results: output,
    variadic: variadic,
  );
  return _syntheticHandle(
    () => TypeDesc(
      _funcText('', shape, (TypeDesc d) => d.name),
      'func',
      const <String, Function>{},
      text: _funcText('', shape, (TypeDesc d) => d.display),
      params: input,
      results: output,
      variadic: variadic,
      comparable: false,
      zero: () => null,
    ),
  );
}

/// A handle of the array type of a length and an element type.
GoInt $arrayOf(GoInt length, GoInt elem) {
  final TypeDesc e = _typeAt($ix(elem));
  final String name = '[$length]${e.name}';
  return $gi(
    _syntheticHandle(
      () => _withType<TypeDesc>(
        e,
        <T>() => TypeDesc(
          name,
          'array',
          const <String, Function>{},
          text: '[$length]${e.display}',
          elem: () => e,
          length: length,
          comparable: e.comparable,
          zero: () => _zeros<T>(e, length),
          withType: <R>(R Function<U>() f) => f<List<T>>(),
        ),
      ),
    ),
  );
}

/// A handle of the map type of a key type and an element type.
int $mapOf(int key, int elem) {
  final TypeDesc k = _typeAt(key);
  final TypeDesc e = _typeAt(elem);
  final String name = 'map[${k.name}]${e.name}';
  return _syntheticHandle(
    () => TypeDesc(
      name,
      'map',
      const <String, Function>{},
      text: 'map[${k.display}]${e.display}',
      key: () => k,
      elem: () => e,
      comparable: false,
      zero: () => _withType<Object?>(
        k,
        <K>() => _withType<Object?>(
          e,
          <V>() => GoMap<K, V>.nil(
            () => _zeroOf(e) as V,
            equals: _keyEquals<K>(k),
            hashCode: _keyHash<K>(k),
          ),
        ),
      ),
      make: (int _) => _withType<Object?>(
        k,
        <K>() => _withType<Object?>(
          e,
          <V>() => GoMap<K, V>(
            () => _zeroOf(e) as V,
            equals: _keyEquals<K>(k),
            hashCode: _keyHash<K>(k),
          ),
        ),
      ),
      withType: <R>(R Function<T>() f) =>
          _withType<R>(k, <K>() => _withType<R>(e, <V>() => f<GoMap<K, V>>())),
    ),
  );
}

/// _keyEquals gives the equality of a map key, or null for the equality of
/// Dart.
///
/// Arrays and runtime-created structs use Dart lists, which compare by identity.
/// Go compares their fields. Pointers to structs and arrays retain identity.
bool Function(K, K)? _keyEquals<K>(TypeDesc k) {
  if (k.kind == 'array' || k.kind == 'struct') {
    return (K a, K b) => $goEqual(a, b, k);
  }
  if (k.kind == 'ptr' && _isObjectPointer(k.elem!())) {
    return (K a, K b) => identical(a, b);
  }
  return null;
}

/// _keyHash gives the hash of a map key, and it pairs with _keyEquals.
int Function(K)? _keyHash<K>(TypeDesc k) {
  if (k.kind == 'array' || k.kind == 'struct') {
    return (K a) => $goHash(a, k);
  }
  if (k.kind == 'ptr' && _isObjectPointer(k.elem!())) {
    return (K a) => identityHashCode(a);
  }
  return null;
}

/// A handle of the pointer type of an element type.
int $pointerTo(int elem) {
  final TypeDesc e = _typeAt(elem);
  return _syntheticHandle(() => $ptrDesc(e));
}

/// A handle of the channel type of a direction and an element type. The
/// direction is 3 for both, 2 for send, and 1 for receive.
int $chanOf(int dir, int elem) {
  final TypeDesc e = _typeAt(elem);
  final String head = dir == 1 ? '<-chan ' : (dir == 2 ? 'chan<- ' : 'chan ');
  final bool parenthesize = dir == 3 && e.name.startsWith('<-chan ');
  return _syntheticHandle(
    () => TypeDesc(
      parenthesize ? '$head(${e.name})' : '$head${e.name}',
      'chan',
      const <String, Function>{},
      text: parenthesize ? '$head(${e.display})' : '$head${e.display}',
      elem: () => e,
      dir: dir,
      zero: () =>
          _withType<Object?>(e, <T>() => Chan<T>.nil(() => _zeroOf(e) as T)),
      withType: <R>(R Function<T>() f) =>
          _withType<R>(e, <T>() => f<Chan<T>>()),
    ),
  );
}

/// A channel of the type, with the size of the buffer.
Iface? $makeChan(int handle, int buffer) {
  final TypeDesc desc = _typeAt(handle);
  final TypeDesc? elem = desc.elem?.call();
  if (desc.kind != 'chan' || elem == null) {
    $panic($runtimeError('reflect: MakeChan of ${desc.display}'));
  }
  if (desc.dir != 0 && desc.dir != 3) {
    $panic($runtimeError('reflect: MakeChan of a channel of 1 direction'));
  }
  return Iface(
    desc,
    _withType<Object?>(
      elem,
      <T>() => Chan<T>($chanCapacity(buffer, elem), () => _zeroOf(elem) as T),
    ),
  );
}

/// _chanOf gives the channel of a value, and it panics for each other kind.
Chan<Object?> _chanOf(Iface? value, String call) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is! Chan<Object?>) {
    $panic($runtimeError('reflect: $call of ${value.desc.display}'));
  }
  return v;
}

/// _rawElem gives the raw Dart form of a value that a channel of the type
/// holds. A channel of an interface type keeps the pair itself.
Object? _rawElem(TypeDesc chan, Iface? x) {
  final TypeDesc? elem = chan.elem?.call();
  return elem == null || elem.kind == 'interface' ? x : x?.value;
}

/// _wrapElem puts the raw Dart form of an element back into an interface value.
Iface? _wrapElem(TypeDesc chan, Object? raw) {
  final TypeDesc? elem = chan.elem?.call();
  if (elem == null || elem.kind == 'interface') return raw as Iface?;
  return Iface(elem, raw);
}

/// A send on a channel that a Value holds. The call blocks the goroutine,
/// exactly as a send of Go does.
void $valueSend(Iface? value, Iface? x) {
  _chanOf(value, 'Send').send(_rawElem(value!.desc, x));
}

/// Closes a channel that a Value holds.
void $valueClose(Iface? value) {
  _chanOf(value, 'Close').close();
}

/// A receive on a channel that gives up no control.
///
/// The 3 results are the value, the flag of the receive, and the flag of a
/// valid Value. A receive that cannot end at this moment gives the zero Value,
/// thus the third flag is false.
(Iface?, bool, bool) $valueTryRecv(Iface? value) {
  final Chan<Object?> c = _chanOf(value, 'TryRecv');
  if (!c.readyToReceive) return (null, false, false);
  final (Object?, bool) got = c.receiveNow();
  return (_wrapElem(value!.desc, got.$1), got.$2, true);
}

/// A send that gives up no control. The flag is false when the send cannot end
/// at this moment.
bool $valueTrySend(Iface? value, Iface? x) {
  final Chan<Object?> c = _chanOf(value, 'TrySend');
  if (!c.readyToSend) return false;
  c.sendNow(_rawElem(value!.desc, x));
  return true;
}

/// Runs a select statement of the package reflect.
///
/// The direction of each case takes the numbers of reflect.SelectDir: 1 for a
/// send, 2 for a receive, and 3 for the default. A case with a nil Chan is
/// ignored, exactly as in Go. The 3 results are the number of the case, the
/// value of a receive, and the flag of that receive.
(int, Iface?, bool) $selectValue(
  Slice<GoInt> dirs,
  Slice<Iface?> chans,
  Slice<Iface?> sends,
) {
  final List<SelectCase> cases = <SelectCase>[];
  final List<int> order = <int>[];
  final List<TypeDesc?> descs = <TypeDesc?>[];
  int fallback = -1;
  for (int i = 0; i < dirs.len; i++) {
    final int dir = $ix(dirs[i]);
    if (dir == 3) {
      if (fallback >= 0) {
        $panic($runtimeError('reflect.Select: multiple default cases'));
      }
      fallback = i;
      continue;
    }
    final Iface? holder = chans[i];
    // Go ignores a case whose Chan holds the zero Value.
    if (holder == null) continue;
    final Chan<Object?> c = _chanOf(holder, 'Select');
    order.add(i);
    descs.add(holder.desc);
    if (dir == 2) {
      cases.add(RecvCase<Object?>(c));
      continue;
    }
    cases.add(SendCase<Object?>(c, _rawElem(holder.desc, sends[i])));
  }
  final int taken = $select(cases, fallback >= 0);
  if ($blocked != null) {
    $block(_SelectResume($blocked!, cases, order, descs));
    return (0, null, false);
  }
  if (taken < 0) return (fallback, null, false);
  return _selectResult(cases, order, descs, taken);
}

/// _selectResult reads the value of the case that a select statement took.
(int, Iface?, bool) _selectResult(
  List<SelectCase> cases,
  List<int> order,
  List<TypeDesc?> descs,
  int taken,
) {
  if (taken < 0 || taken >= cases.length) return (0, null, false);
  final SelectCase c = cases[taken];
  if (c is! RecvCase<Object?>) return (order[taken], null, false);
  final (Object?, bool) got = c.result ?? (null, false);
  final TypeDesc? desc = descs[taken];
  final Iface? out = desc == null ? got.$1 as Iface? : _wrapElem(desc, got.$1);
  return (order[taken], out, got.$2);
}

/// The frame of a goroutine that waits inside reflect.Select.
///
/// The frame of the runtime gives the number of the case, and the package
/// reflect needs the value of a receive as well. The conversion stands here,
/// because the caller reads the result of the resume.
final class _SelectResume implements WaitFrame, DurableWait, NestedFrame {
  Frame inner;
  final List<SelectCase> cases;
  final List<int> order;
  final List<TypeDesc?> descs;

  _SelectResume(this.inner, this.cases, this.order, this.descs);

  @override
  Frame get $inner => inner;
  @override
  set $inner(Frame frame) {
    inner = frame;
  }

  @override
  String get $reason => $waitReason(inner);

  /// The wait of the select statement of the runtime holds the answer.
  @override
  bool $durableIn(Bubble b) => $durableWait(inner, b);

  @override
  Object? resume() {
    final Object? out = $resumeFrame(inner);
    final Frame? again = $blocked;
    if (again != null) {
      $block(_SelectResume(again, cases, order, descs));
      return (0, null, false);
    }
    return _selectResult(cases, order, descs, out is int ? out : -1);
  }
}

/// _zeroOf gives the zero value of a type, or null for a type that holds no
/// zero builder.
Object? _zeroOf(TypeDesc desc) {
  if (desc.kind == 'interface') return null;
  return desc.zero?.call();
}

/// _zeros gives a list of the length that holds the zero value of the type.
///
/// The caller names the Dart type of the element through _withType, thus a
/// type assertion of the value reads. Each slot takes a fresh zero, because Go
/// gives 1 struct of its own to each element of an array.
List<T> _zeros<T>(TypeDesc elem, GoInt n) {
  if ($typeSize(elem) == GoInt.zero) {
    return $ZeroList<T>(n, () => _zeroOf(elem) as T);
  }
  return List<T>.generate(
    $ix(n),
    (int _) => _zeroOf(elem) as T,
    growable: true,
  );
}

/// The capacity of a slice, of an array, or of a channel.
GoInt $valueCap(Iface? value) {
  if (value == null) return GoInt.zero;
  final Object? v = value.value;
  if (v is Slice<Object?>) return v.$capacity;
  if (v is List<Object?>) return value.desc.length;
  // The capacity of a channel is the size of its buffer.
  if (v is Chan<Object?>) return $gi(v.capacity);
  return $valueLen(value);
}

/// _freshSlice builds a slice of the type with the length, through the make of
/// the descriptor. A type that the program never holds gives a panic.
Slice<Object?> _freshSlice(TypeDesc desc, int length) {
  final Object? Function(int)? make = desc.make;
  if (make == null) {
    $panic($runtimeError('reflect: Append of ${desc.display}'));
  }
  return make(length) as Slice<Object?>;
}

/// Appends each value to a slice and gives the new slice.
Iface? $appendValue(Iface? slice, Slice<Iface?> items) {
  if (slice == null) $panicNil();
  final TypeDesc desc = slice.desc;
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) {
    $panic($runtimeError('reflect: Append of ${desc.display}'));
  }
  return _withType<Iface>(elem, <T>() {
    final Slice<T> held = slice.value as Slice<T>;
    final List<T> add = <T>[
      for (int i = 0; i < items.len; i++)
        _copyOf(elem.kind == 'interface' ? items[i] : items[i]?.value, elem)
            as T,
    ];
    return Iface(
      desc,
      $append<T>(
        held,
        add,
        (int n) => (_freshSlice(desc, n) as Slice<T>).array,
        _holdsValue(elem) ? (T a, T b) => _assignInto(a, b, elem) : null,
      ),
    );
  });
}

/// Appends each element of the second slice to the first one.
Iface? $appendSliceValue(Iface? slice, Iface? add) {
  if (slice == null) $panicNil();
  final TypeDesc desc = slice.desc;
  if (desc.elem == null) {
    $panic($runtimeError('reflect: AppendSlice of ${desc.display}'));
  }
  final TypeDesc elem = desc.elem!();
  return _withType<Iface>(
    elem,
    <T>() => Iface(
      desc,
      $appendSlice<T>(
        slice.value as Slice<T>,
        add!.value as Slice<T>,
        (int n) => (_freshSlice(desc, n) as Slice<T>).array,
        _holdsValue(elem) ? (T a, T b) => _assignInto(a, b, elem) : null,
      ),
    ),
  );
}

/// Copies each element of the source into the destination, and gives the count
/// that it wrote. The source may be a string, and the destination is then a
/// byte slice.
GoInt $copyValue(Iface? dst, Iface? src) {
  final to = dst!.value;
  final from = src!.value;
  final target = to is Slice<Object?>
      ? to
      : Slice<Object?>.of(to as List<Object?>);
  if (from is GoString) {
    final count = target.len < from.length ? target.len : from.length;
    for (var i = 0; i < count; i++) {
      target[i] = $gi(from[i]);
    }
    return $gi(count);
  }
  final source = from is Slice<Object?>
      ? from
      : Slice<Object?>.of(from as List<Object?>);
  final elem = dst.desc.elem!();
  return $copySlice<Object?>(
    target,
    source,
    _holdsValue(elem) ? (a, b) => _assignInto(a, b, elem) : null,
  );
}

Object? _readElem(Object? v, int i) {
  if (v is Slice<Object?>) return v[i];
  return (v as List<Object?>)[i];
}

// The type relations that the package reflect reads.

/// Reports whether a type holds each method of an interface type.
bool $descImplements(int handle, int iface) {
  final TypeDesc a = _typeAt(handle);
  final TypeDesc b = _typeAt(iface);
  if (b.kind != 'interface') return false;
  for (final String key in b.methods.keys) {
    if (!a.methods.containsKey(key)) return false;
  }
  return true;
}

/// Reports whether a value of the first type fits a location of the second.
bool $descAssignable(int handle, int other) {
  if (handle == other) return true;
  final TypeDesc a = _typeAt(handle);
  final TypeDesc b = _typeAt(other);
  if (a.name == b.name) return true;
  if (b.kind == 'interface') return $descImplements(handle, other);
  // Go assigns a value of the first type to a location of the second one when
  // the 2 underlying types are identical and at least 1 of the 2 holds no name
  // of its own. `[]byte` fits a location of `type Raw []byte` for that reason.
  if (_unnamedType(a.text) || _unnamedType(b.text)) {
    if (a.kind == 'chan' &&
        b.kind == 'chan' &&
        a.dir == 3 &&
        _sameDesc(a.elem, b.elem)) {
      return true;
    }
    return _sameUnderlying(a, b);
  }
  return false;
}

/// The start of the text of a type that Go writes with no name of its own.
/// The package reflect holds the same list, because Type.Name gives an empty
/// text for such a type.
const List<String> _unnamedHeads = <String>[
  '[',
  '*',
  'map[',
  'chan ',
  'chan<- ',
  '<-chan ',
  'func(',
  'struct{',
  'struct {',
  'interface{',
  'interface {',
];

bool _unnamedType(String text) {
  for (final String head in _unnamedHeads) {
    if (text.startsWith(head)) return true;
  }
  return false;
}

bool _sameUnderlying(TypeDesc a, TypeDesc b, [bool ignoreTags = false]) {
  if (a.kind != b.kind) return false;
  switch (a.kind) {
    case 'struct':
      return _sameStruct(a, b, ignoreTags);
    case 'slice' || 'ptr':
      return _sameFieldType(a.elem!(), b.elem!(), ignoreTags);
    case 'chan':
      return a.dir == b.dir && _sameFieldType(a.elem!(), b.elem!(), ignoreTags);
    case 'array':
      return a.length == b.length &&
          _sameFieldType(a.elem!(), b.elem!(), ignoreTags);
    case 'map':
      return _sameFieldType(a.key!(), b.key!(), ignoreTags) &&
          _sameFieldType(a.elem!(), b.elem!(), ignoreTags);
    case 'func':
      if (a.variadic != b.variadic ||
          a.params.length != b.params.length ||
          a.results.length != b.results.length) {
        return false;
      }
      for (int i = 0; i < a.params.length; i++) {
        if (!_sameFieldType(a.params[i](), b.params[i](), ignoreTags)) {
          return false;
        }
      }
      for (int i = 0; i < a.results.length; i++) {
        if (!_sameFieldType(a.results[i](), b.results[i](), ignoreTags)) {
          return false;
        }
      }
      return true;
    case 'interface':
      return a.methods.isEmpty && b.methods.isEmpty;
    default:
      return true;
  }
}

const Set<String> _numberKinds = <String>{
  'int',
  'int8',
  'int16',
  'int32',
  'int64',
  'uint',
  'uint8',
  'uint16',
  'uint32',
  'uint64',
  'uintptr',
  'float32',
  'float64',
};

/// Reports whether a conversion of the first type to the second one holds.
bool $descConvertible(int handle, int other) {
  if ($descAssignable(handle, other)) return true;
  final TypeDesc a = _typeAt(handle);
  final TypeDesc b = _typeAt(other);
  if (a.kind == 'slice') {
    if (b.kind == 'array') return _sameDesc(a.elem, b.elem);
    if (b.kind == 'ptr' && b.elem!().kind == 'array') {
      return _sameDesc(a.elem, b.elem!().elem);
    }
  }
  if (_numberKinds.contains(a.kind) && _numberKinds.contains(b.kind)) {
    return true;
  }
  if ((a.kind == 'complex64' || a.kind == 'complex128') &&
      (b.kind == 'complex64' || b.kind == 'complex128')) {
    return true;
  }
  if (a.kind == 'string' && b.kind == 'slice') {
    final String elem = b.elem?.call().kind ?? '';
    return elem == 'uint8' || elem == 'int32';
  }
  if (b.kind == 'string') {
    if (_numberKinds.contains(a.kind) &&
        a.kind != 'float32' &&
        a.kind != 'float64') {
      return true;
    }
    if (a.kind == 'slice') {
      final String elem = a.elem?.call().kind ?? '';
      return elem == 'uint8' || elem == 'int32';
    }
  }
  if (_sameUnderlying(a, b, true)) return true;
  return a.kind == 'ptr' &&
      b.kind == 'ptr' &&
      _unnamedType(a.text) &&
      _unnamedType(b.text) &&
      _sameUnderlying(a.elem!(), b.elem!(), true);
}

bool _sameStruct(TypeDesc a, TypeDesc b, bool ignoreTags) {
  if (a.fields.length != b.fields.length) return false;
  for (int i = 0; i < a.fields.length; i++) {
    final FieldDesc x = a.fields[i], y = b.fields[i];
    if (x.name != y.name ||
        x.pkg != y.pkg ||
        x.embedded != y.embedded ||
        (!ignoreTags && x.tag != y.tag) ||
        !_sameFieldType(x.type(), y.type(), ignoreTags)) {
      return false;
    }
  }
  return true;
}

bool _sameFieldType(TypeDesc a, TypeDesc b, bool ignoreTags) {
  if (a.name == b.name) return true;
  if (!ignoreTags ||
      !_unnamedType(a.text) ||
      !_unnamedType(b.text) ||
      a.kind != b.kind) {
    return false;
  }
  return _sameUnderlying(a, b, true);
}

/// Reports whether 2 lazy descriptions name 1 type.
bool _sameDesc(TypeDesc Function()? a, TypeDesc Function()? b) {
  if (a == null || b == null) return false;
  return a().name == b().name;
}

/// The direction of a channel type: 3 for both, 2 for send, and 1 for receive.
int $descChanDir(int handle) {
  final TypeDesc desc = _typeAt(handle);
  if (desc.kind != 'chan') return 0;
  // A description that names no direction reads as a channel of both
  // directions.
  return desc.dir == 0 ? 3 : desc.dir;
}

// The value operations that text/template reads.

/// A pointer to the location that a Value names.
///
/// The loc names the kind of the location: 1 for a pointer, 2 for a field of a
/// struct, and 3 for an element of a slice or of an array.
Iface? $valueAddr(Iface? value, Iface? owner, GoInt exactIndex, int loc) {
  final index = $ix(exactIndex);
  if (loc == 1) return owner;
  if (owner == null) $panicNil();
  final TypeDesc ownerDesc = owner.desc;
  if (loc == 2) {
    if (index < 0 || index >= ownerDesc.fields.length) {
      $panic($runtimeError('reflect: Addr of a field'));
    }
    final FieldDesc f = ownerDesc.fields[index];
    final TypeDesc field = f.type();
    // The Dart form of a pointer to a struct and of a pointer to an array is
    // the value itself.
    if (_isObjectPointer(field)) {
      return Iface($ptrDesc(field), f.get(owner.value));
    }
    // The index of the field keeps 2 addresses of 2 fields apart, because a
    // cell compares the owner and the index.
    return Iface(
      $ptrDesc(field),
      _typedCell(
        field,
        owner.value,
        index,
        (Object? o, int _) => f.get(o),
        (Object? o, int _, Object? v) => f.set(o, v),
      ),
    );
  }
  final TypeDesc? elem = ownerDesc.elem?.call();
  if (elem == null) {
    $panic($runtimeError('reflect: Addr of an element'));
  }
  final Object? v = owner.value;
  if (v is $ZeroSlice<Object?>) {
    return Iface($ptrDesc(elem), v.$at(exactIndex));
  }
  if (v is $ZeroList<Object?>) {
    return Iface($ptrDesc(elem), _elemAt(v, exactIndex, ownerDesc));
  }
  if (_isObjectPointer(elem)) {
    return Iface($ptrDesc(elem), _readElem(v, index));
  }
  return Iface(
    $ptrDesc(elem),
    _withType<$Ptr<Object?>>(elem, <T>() => $sequencePtr<T>(v, exactIndex)),
  );
}

/// Reports whether 2 values hold the same type and the same value.
bool $valueEqual(Iface? a, Iface? b) {
  if (a == null || b == null) return a == null && b == null;
  if (a.desc.name != b.desc.name) return false;
  return $goEqual(a.value, b.value, a.desc);
}

/// The value of a type conversion.
Iface? $valueConvert(Iface? value, int handle) {
  final TypeDesc to = _typeAt(handle);
  // A conversion to an interface keeps the value and its own descriptor. The
  // dynamic type of a value inside an interface is the type of that value, and
  // the interface stands as the static type of the result alone. A result that
  // carried the descriptor of the interface would lose the concrete type, thus
  // a later type assertion to that type would fail.
  if (to.kind == 'interface') {
    if (value == null) return null;
    if (!$descConvertible(_descHandle(value.desc), handle)) {
      $panic(
        $runtimeError(
          'reflect: cannot convert ${value.desc.display} to ${to.display}',
        ),
      );
    }
    return $copyInterface(value);
  }
  if (value == null) return Iface(to, to.zero?.call());
  final TypeDesc from = value.desc;
  if (from.name == to.name) return $copyInterface(value);
  if (!$descConvertible(_descHandle(from), handle)) {
    $panic(
      $runtimeError('reflect: cannot convert ${from.display} to ${to.display}'),
    );
  }
  return Iface(to, _converted(from, to, value.value));
}

/// The compiler supplies types that Go permits in a conversion.
T $convert<T>(Object? value, TypeDesc from, TypeDesc to) =>
    _converted(from, to, value) as T;

Object? $structLiteral(TypeDesc desc, List<Object?> fields) {
  final Object? out = desc.zero!();
  for (int i = 0; i < fields.length; i++) {
    if (desc.fields[i].name != '_') desc.fields[i].set(out, fields[i]);
  }
  return out;
}

// The source owns its converted views. The global table must not retain it.
final Expando<(TypeDesc, Map<String, Object>)> _structViews = Expando(
  'struct views',
);

Object? _structPointer(TypeDesc from, TypeDesc to, Object? value) {
  if (value == null) return null;
  final Object origin = $pointerOwner(value)!;
  final entry = _structViews[origin] ??= (
    from,
    <String, Object>{from.name: origin},
  );
  return entry.$2.putIfAbsent(
    to.name,
    () => to.view?.call(entry.$1, origin) ?? _StructListView(entry.$1, origin),
  );
}

final class _StructListView extends ListBase<Object?> implements $StructView {
  final TypeDesc type;
  @override
  final Object $origin;
  _StructListView(this.type, this.$origin);
  @override
  int get length => type.fields.length;
  @override
  set length(int value) => throw UnsupportedError('fixed struct fields');
  @override
  Object? operator [](int index) => type.fields[index].get($origin);
  @override
  void operator []=(int index, Object? value) =>
      type.fields[index].set($origin, value);
}

Object? _converted(TypeDesc from, TypeDesc to, Object? v) {
  if (from.kind == 'ptr' && to.kind == 'ptr') {
    v = $ptrValue(from.elem!(), v);
    if (to.elem!().kind == 'struct') {
      return _structPointer(from.elem!(), to.elem!(), v);
    }
  }

  if (to.kind == 'struct') {
    final Object? out = to.zero!();
    for (int i = 0; i < to.fields.length; i++) {
      final FieldDesc src = from.fields[i];
      final FieldDesc dst = to.fields[i];
      if (dst.name != '_') {
        dst.set(out, _converted(src.type(), dst.type(), src.get(v)));
      }
    }
    return out;
  }
  if (from.kind == 'slice' && to.kind == 'ptr') {
    final TypeDesc array = to.elem!();
    final Slice<Object?> src = v as Slice<Object?>;
    if (src.$length < array.length) {
      $panic(
        Iface(
          $typeString,
          GoString.fromString(
            'reflect: cannot convert slice with length ${src.$length} to pointer to array with length ${array.length}',
          ),
        ),
      );
    }
    return _withType<Object?>(
      from.elem!(),
      <T>() => $sliceToArrayPtr<T>(v as Slice<T>, array.length),
    );
  }
  if (to.kind == 'array') {
    if (from.kind == 'slice') {
      final Slice<Object?> src = v as Slice<Object?>;
      if (src.$length < to.length) {
        $panic(
          Iface(
            $typeString,
            GoString.fromString(
              'reflect: cannot convert slice with length ${src.$length} to array with length ${to.length}',
            ),
          ),
        );
      }
      final List<Object?> out = to.zero!() as List<Object?>;
      if (out is $ZeroList<Object?>) return out;
      for (int i = 0; i < out.length; i++) {
        out[i] = _converted(from.elem!(), to.elem!(), src[i]);
      }
      return out;
    }
    final List<Object?> out = to.zero!() as List<Object?>;
    if (out is $ZeroList<Object?>) return out;
    final List<Object?> src = v as List<Object?>;
    for (int i = 0; i < out.length; i++) {
      out[i] = _converted(from.elem!(), to.elem!(), src[i]);
    }
    return out;
  }
  if (to.kind == 'string') {
    if (_numberKinds.contains(from.kind)) {
      return $runeToString(v as GoInt);
    }
    if (v is Slice<GoInt>) {
      return from.elem!().kind == 'int32'
          ? $runesToString(v)
          : $bytesToString(v);
    }
    return v;
  }
  if (from.kind == 'string' && to.kind == 'slice') {
    final GoString s = v as GoString;
    return to.elem!().kind == 'int32' ? $stringToRunes(s) : $stringToBytes(s);
  }
  if (to.kind == 'complex64') return $c64(v as GoComplex);
  if (to.kind == 'complex128') return $c128(v as GoComplex);
  if (_numberKinds.contains(to.kind)) {
    if (to.kind == 'float32' || to.kind == 'float64') {
      if (from.kind == to.kind) return v;
      final double x = v is GoInt
          ? (from.kind == 'uint64' ||
                    from.kind == 'uint' ||
                    from.kind == 'uintptr'
                ? $utodouble(v)
                : $todouble(v))
          : v is GoFloat32
          ? v.toDouble()
          : (v as GoFloat64).toDouble();
      return to.kind == 'float32' ? $f32(x) : $f64(x);
    }
    final x = v is GoInt
        ? v
        : (to.kind == 'uint64' || to.kind == 'uint' || to.kind == 'uintptr'
              ? $fromF64U(
                  v is GoFloat32 ? v.toDouble() : (v as GoFloat64).toDouble(),
                )
              : $fromF64(
                  v is GoFloat32 ? v.toDouble() : (v as GoFloat64).toDouble(),
                ));
    return _integerValue(x, to.kind);
  }
  return v;
}

/// The slice of a value, with the low, the high, and the maximum.
Iface? $valueSlice(Iface? value, GoInt low, GoInt high, GoInt max) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is GoString) {
    return Iface(value.desc, $substring(v, low, high));
  }
  if (v is Slice<Object?>) {
    if (v.isNil) return value;
    // A zero-capacity result keeps the source pointer within its storage.
    return Iface(
      value.desc,
      v.$viewExact(low < max ? low : GoInt.zero, high - low, max - low),
    );
  }
  final TypeDesc? elem = value.desc.elem?.call();
  if (value.desc.kind == 'array' && v is List<Object?> && elem != null) {
    // Go gives a slice that shares the storage of the array. The Dart form of
    // an array is 1 list, thus a slice over that list holds the same list and
    // a write through it reaches the array.
    final TypeDesc desc = _typeAt($sliceOf(_descHandle(elem)));
    return Iface(
      desc,
      _withType<Slice<Object?>>(
        elem,
        <T>() => Slice<T>.of(v as List<T>)
            .$viewExact(low < max ? low : GoInt.zero, high - low, max - low),
      ),
    );
  }
  $panic($runtimeError('reflect: Slice of ${value.desc.display}'));
}

/// The complex value inside an interface value.
GoComplex $asComplex(Iface? value) {
  final Object? v = value?.value;
  if (v is GoComplex) return $c128(v);
  if (v is GoFloat64) return GoComplex.fromParts(v, GoFloat64.zero);
  if (v is GoInt) return GoComplex($ix(v).toDouble(), 0);
  return GoComplex.zero;
}

/// A complex value of the type of a handle.
Iface? $withComplex(int handle, GoComplex x) {
  final type = _typeAt(handle);
  return Iface(type, type.kind == 'complex64' ? $c64(x) : x);
}

/// A synthetic identity. A slice uses its storage and element offset.
/// The result is not a machine address.
int $valuePointer(Iface? value) {
  if ($valueIsNil(value)) return 0;
  final Object? v = value?.value;
  if (v is Function) {
    final pc = $functionPC(v);
    if (pc != null) return pc;
  }
  if (v is $Ref) return v.hashCode;
  if (v is Slice<Object?>) {
    return Object.hash(identityHashCode(v.array), v.off);
  }
  return identityHashCode($pointerOwner(v));
}

/// A receive on a channel that a Value holds. The call blocks the goroutine,
/// exactly as a receive of Go does.
(Iface?, bool) $valueRecv(Iface? value) {
  final Chan<Object?> c = _chanOf(value, 'Recv');
  final (Object?, bool) got = c.receive();
  // The frame of the channel gives the raw value of the element, and the
  // package reflect needs an interface value. The wrapper converts it.
  if ($blocked != null) {
    $block(_RecvResume($blocked!, value!.desc));
    return (null, false);
  }
  return (_wrapElem(value!.desc, got.$1), got.$2);
}

/// The frame of a goroutine that waits inside reflect.Value.Recv.
final class _RecvResume implements WaitFrame, DurableWait, NestedFrame {
  Frame inner;
  final TypeDesc desc;

  _RecvResume(this.inner, this.desc);

  @override
  Frame get $inner => inner;
  @override
  set $inner(Frame frame) {
    inner = frame;
  }

  @override
  String get $reason => $waitReason(inner);

  /// The wait of the channel of the runtime holds the answer.
  @override
  bool $durableIn(Bubble b) => $durableWait(inner, b);

  @override
  Object? resume() {
    final Object? out = $resumeFrame(inner);
    final Frame? again = $blocked;
    if (again != null) {
      $block(_RecvResume(again, desc));
      return (null, false);
    }
    final (Object?, bool) got = out as (Object?, bool);
    return (_wrapElem(desc, got.$1), got.$2);
  }
}

/// Reports whether the last parameter of a function type is variadic.
bool $descVariadic(int handle) => _typeAt(handle).variadic;

/// The import path of the package that declares an unexported field.
GoString $fieldPkgPath(int handle, int index) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null || index < 0 || index >= desc.fields.length) {
    return GoString.empty;
  }
  return GoString.fromString(desc.fields[index].pkg);
}

/// The import path of the package of a type. A type that no package holds, for
/// example a slice type, gives an empty text.
GoString $descPkgPath(int handle) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null) return GoString.empty;
  final int cut = desc.name.lastIndexOf('.');
  if (cut <= 0) return GoString.empty;
  // The emitter names a type of package main with the path of the module, and
  // Go names it main. The short text holds the name of the package.
  if (desc.display.startsWith('main.')) {
    return GoString.fromString('main');
  }
  final String head = desc.name.substring(0, cut);
  // A composite type holds no package. Its name starts with a bracket, a star,
  // or the word map, chan, func, or interface. A function type also holds a
  // parenthesis, and the name of a parameter of it holds the import path.
  if (head.contains('[') ||
      head.contains('*') ||
      head.contains(' ') ||
      head.contains('(')) {
    return GoString.empty;
  }
  return GoString.fromString(head);
}

/// Reports whether a field of a struct type is embedded.
bool $fieldEmbedded(int handle, int index) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null || index < 0 || index >= desc.fields.length) return false;
  return desc.fields[index].embedded;
}

Iface $growValue(Iface value, GoInt count) {
  final desc = value.desc;
  final elem = desc.elem!();
  return _withType<Iface>(elem, <T>() {
    final held = value.value as Slice<T>;
    final need = held.$length + count;
    if (held is $ZeroSlice<T>) {
      return Iface(
        desc,
        $ZeroSlice<T>(held.array as $ZeroList<T>, held.$length, need),
      );
    }
    final capacity = $nextSliceCapacity(held.cap, $ix(need));
    final fresh = _freshSlice(desc, capacity) as Slice<T>;
    $copySlice<T>(
      fresh,
      held.$view(held.off, held.cap, held.cap),
      _holdsValue(elem) ? (T a, T b) => _assignInto(a, b, elem) : null,
    );
    return Iface(desc, fresh.$view(0, held.len, capacity));
  });
}

/// Exchanges 2 elements of a slice. The package sort uses it, because a Go
/// sort works on a slice of an unknown element type.
void $swap(Iface? value, GoInt first, GoInt second) {
  final Object? unwrapped = value?.value;
  if (unwrapped is! Slice<Object?>) {
    $panic($runtimeError('sort: the value is not a slice'));
  }
  final Slice<Object?> slice = unwrapped;
  if (first < GoInt.zero || first >= slice.$length) {
    $panicIndex(first, slice.$length);
  }
  if (second < GoInt.zero || second >= slice.$length) {
    $panicIndex(second, slice.$length);
  }
  if (slice is $ZeroSlice<Object?>) return;
  final i = $ix(first);
  final j = $ix(second);
  final elem = value!.desc.elem!();
  if (_holdsValue(elem)) {
    final a = slice.array[slice.off + i];
    final b = slice.array[slice.off + j];
    final temporary = _copyOf(a, elem);
    _assignInto(a, b, elem);
    _assignInto(b, temporary, elem);
    return;
  }
  final Object? a = slice.array[slice.off + i];
  slice.array[slice.off + i] = slice.array[slice.off + j];
  slice.array[slice.off + j] = a;
}

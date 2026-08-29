/// The intrinsics of the package reflect.
///
/// A Go value of an interface type is an Iface: a type descriptor and the value
/// itself. Each function below reads or writes such a pair, thus the package
/// reflect holds no memory layout.
library;

import 'box.dart';
import 'iface.dart';
import 'map.dart';
import 'panic.dart';
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

int _descHandle(TypeDesc desc) {
  final int? found = _descHandles[desc.name];
  if (found != null) return found;
  _descTable.add(desc);
  _descHandles[desc.name] = _descTable.length;
  return _descTable.length;
}

TypeDesc? _descAt(int handle) {
  if (handle <= 0 || handle > _descTable.length) return null;
  return _descTable[handle - 1];
}

/// A handle of the type of an interface value. A nil value gives 0.
int $descOf(Iface? value) => value == null ? 0 : _descHandle(value.desc);

/// The text, the kind, and the comparable flag of a type handle.
(GoString, GoString, bool) $descInfo(int handle) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null) return (GoString.empty, GoString.empty, false);
  return (
    GoString.fromString(desc.display),
    GoString.fromString(desc.kind),
    desc.comparable,
  );
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
    $panic(GoError('reflect: no zero value of ${desc.display}'));
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
/// field. A field of a struct type keeps the reference of the source, and
/// docs/parity.md holds that gap.
void $pointerSet(Iface? pointer, Iface? value) {
  final TypeDesc elem = _pointerElemDesc(pointer);
  final Object? target = pointer!.value;
  final Object? stored = elem.kind == 'interface' ? value : value?.value;
  if (target is $Ptr<Object?>) {
    target.v = stored;
    return;
  }
  if (elem.kind == 'struct') {
    for (final FieldDesc field in elem.fields) {
      field.set(target, field.get(stored));
    }
    return;
  }
  $panic(GoError('reflect: Set on a pointer to ${elem.display}'));
}

TypeDesc _pointerElemDesc(Iface? pointer) {
  if (pointer == null) {
    $panic(const GoError('reflect: the value holds no address'));
  }
  final TypeDesc desc = pointer.desc;
  final TypeDesc Function()? elem = desc.elem;
  if (desc.kind != 'ptr' || elem == null) {
    $panic(GoError('reflect: ${desc.display} is not a pointer'));
  }
  return elem();
}

/// The count of the elements of an array type.
int $descLength(int handle) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null) return 0;
  return desc.length;
}

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
    $panic(const GoError('reflect: the type holds no such field'));
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
  field.set(value!.value, type.kind == 'interface' ? x : x?.value);
}

FieldDesc _fieldOf(Iface? value, int index) {
  if (value == null) $panicNil();
  final TypeDesc desc = value.desc;
  if (desc.kind != 'struct' || index < 0 || index >= desc.fields.length) {
    $panic(GoError('reflect: ${desc.display} holds no field $index'));
  }
  return desc.fields[index];
}

/// The length of a slice, an array, a string, or a map.
int $valueLen(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is Slice<Object?>) return v.len;
  if (v is List<Object?>) return v.length;
  if (v is GoString) return v.length;
  if (v is GoMap<Object?, Object?>) return v.length;
  $panic(GoError('reflect: Len of ${value.desc.display}'));
}

/// 1 element of a slice, an array, or a string.
Iface? $valueIndex(Iface? value, int index) {
  if (value == null) $panicNil();
  final TypeDesc desc = value.desc;
  final Object? v = value.value;
  if (v is GoString) {
    return Iface(_byteDesc, v[index]);
  }
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) {
    $panic(GoError('reflect: Index of ${desc.display}'));
  }
  final Object? out = _elemAt(v, index, desc);
  if (elem.kind == 'interface') return out as Iface?;
  return Iface(elem, out);
}

/// Writes 1 element of a slice or an array.
void $valueSetIndex(Iface? value, int index, Iface? x) {
  if (value == null) $panicNil();
  final TypeDesc desc = value.desc;
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) {
    $panic(GoError('reflect: Index of ${desc.display}'));
  }
  final Object? stored = elem.kind == 'interface' ? x : x?.value;
  final Object? v = value.value;
  if (v is Slice<Object?>) {
    if (index < 0 || index >= v.len) $panicIndex(index, v.len);
    v.array[v.off + index] = stored;
    return;
  }
  if (v is List<Object?>) {
    if (index < 0 || index >= v.length) $panicIndex(index, v.length);
    v[index] = stored;
    return;
  }
  $panic(GoError('reflect: Set of an element of ${desc.display}'));
}

Object? _elemAt(Object? v, int index, TypeDesc desc) {
  if (v is Slice<Object?>) {
    if (index < 0 || index >= v.len) $panicIndex(index, v.len);
    return v.array[v.off + index];
  }
  if (v is List<Object?>) {
    if (index < 0 || index >= v.length) $panicIndex(index, v.length);
    return v[index];
  }
  $panic(GoError('reflect: Index of ${desc.display}'));
}

final TypeDesc _byteDesc = TypeDesc(
  'uint8',
  'uint8',
  const <String, Function>{},
  text: 'uint8',
  zero: () => 0,
);

/// The whole number inside a value of an integer kind.
int $asInt(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is int) return v;
  $panic(GoError('reflect: the value of ${value.desc.display} is no integer'));
}

/// The float inside a value of a float kind.
double $asFloat(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is double) return v;
  if (v is int) return v.toDouble();
  $panic(GoError('reflect: the value of ${value.desc.display} is no float'));
}

/// The flag inside a value of the kind bool.
bool $asBool(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is bool) return v;
  $panic(GoError('reflect: the value of ${value.desc.display} is no bool'));
}

/// The text inside a value of the kind string.
GoString $asString(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is GoString) return v;
  $panic(GoError('reflect: the value of ${value.desc.display} is no string'));
}

/// A value of the type, with the whole number inside it.
Iface? $withInt(int handle, int x) => Iface(_typeAt(handle), x);

/// A value of the type, with the float inside it.
Iface? $withFloat(int handle, double x) => Iface(_typeAt(handle), x);

/// A value of the type, with the flag inside it.
Iface? $withBool(int handle, bool x) => Iface(_typeAt(handle), x);

/// A value of the type, with the text inside it.
Iface? $withString(int handle, GoString x) => Iface(_typeAt(handle), x);

TypeDesc _typeAt(int handle) {
  final TypeDesc? desc = _descAt(handle);
  if (desc == null) {
    $panic(const GoError('reflect: the handle names no type'));
  }
  return desc;
}

/// A pointer to a fresh zero value of the type.
Iface? $newValue(int handle) {
  final TypeDesc desc = _typeAt(handle);
  final Object? Function()? zero = desc.zero;
  if (zero == null) {
    $panic(GoError('reflect: New of ${desc.display}'));
  }
  final Object? target = desc.kind == 'struct' ? zero() : Box<Object?>(zero());
  return Iface(_pointerTo(desc), target);
}

/// A slice of the type, with the length and the capacity.
Iface? $makeValue(int handle, int length) {
  final TypeDesc desc = _typeAt(handle);
  final Object? Function(int)? make = desc.make;
  if (make == null) {
    $panic(GoError('reflect: Make of ${desc.display}'));
  }
  return Iface(desc, make(length));
}

final Map<String, TypeDesc> _pointers = <String, TypeDesc>{};

TypeDesc _pointerTo(TypeDesc elem) {
  final TypeDesc? found = _pointers[elem.name];
  if (found != null) return found;
  final TypeDesc out = TypeDesc(
    '*${elem.name}',
    'ptr',
    const <String, Function>{},
    text: '*${elem.display}',
    elem: () => elem,
    zero: () => null,
  );
  _pointers[elem.name] = out;
  return out;
}

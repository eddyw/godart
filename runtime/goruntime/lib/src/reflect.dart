/// The intrinsics of the package reflect.
///
/// A Go value of an interface type is an Iface: a type descriptor and the value
/// itself. Each function below reads or writes such a pair, thus the package
/// reflect holds no memory layout.
library;

import 'box.dart';
import 'chan.dart';
import 'complex.dart';
import 'format.dart';
import 'goint.dart';
import 'iface.dart';
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

int _descHandle(TypeDesc desc) {
  final int? found = _descHandles[desc.name];
  if (found != null) {
    if (_synthetic.contains(found)) {
      _descTable[found - 1] = desc;
      _synthetic.remove(found);
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
    $panic(
      Iface($typeGoError, GoError('reflect: no zero value of ${desc.display}')),
    );
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
  $panic(
    Iface(
      $typeGoError,
      GoError('reflect: Set on a pointer to ${elem.display}'),
    ),
  );
}

TypeDesc _pointerElemDesc(Iface? pointer) {
  if (pointer == null) {
    $panic(
      Iface($typeGoError, const GoError('reflect: the value holds no address')),
    );
  }
  final TypeDesc desc = pointer.desc;
  final TypeDesc Function()? elem = desc.elem;
  if (desc.kind != 'ptr' || elem == null) {
    $panic(
      Iface($typeGoError, GoError('reflect: ${desc.display} is not a pointer')),
    );
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
    $panic(
      Iface(
        $typeGoError,
        const GoError('reflect: the type holds no such field'),
      ),
    );
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
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: ${desc.display} holds no field $index'),
      ),
    );
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
  // The length of a channel is the count of the values that the buffer holds.
  if (v is Chan<Object?>) return v.length;
  $panic(Iface($typeGoError, GoError('reflect: Len of ${value.desc.display}')));
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
    $panic(Iface($typeGoError, GoError('reflect: Index of ${desc.display}')));
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
    $panic(Iface($typeGoError, GoError('reflect: Index of ${desc.display}')));
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
  $panic(
    Iface(
      $typeGoError,
      GoError('reflect: Set of an element of ${desc.display}'),
    ),
  );
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
  $panic(Iface($typeGoError, GoError('reflect: Index of ${desc.display}')));
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
  if (v is GoInt) return $ix(v);
  $panic(
    Iface(
      $typeGoError,
      GoError('reflect: the value of ${value.desc.display} is no integer'),
    ),
  );
}

/// The float inside a value of a float kind.
double $asFloat(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is double) return v;
  if (v is GoInt) return v.toDouble();
  $panic(
    Iface(
      $typeGoError,
      GoError('reflect: the value of ${value.desc.display} is no float'),
    ),
  );
}

/// The flag inside a value of the kind bool.
bool $asBool(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is bool) return v;
  $panic(
    Iface(
      $typeGoError,
      GoError('reflect: the value of ${value.desc.display} is no bool'),
    ),
  );
}

/// The text inside a value of the kind string.
GoString $asString(Iface? value) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is GoString) return v;
  $panic(
    Iface(
      $typeGoError,
      GoError('reflect: the value of ${value.desc.display} is no string'),
    ),
  );
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
    $panic(
      Iface($typeGoError, const GoError('reflect: the handle names no type')),
    );
  }
  return desc;
}

/// A pointer to a fresh zero value of the type.
Iface? $newValue(int handle) {
  final TypeDesc desc = _typeAt(handle);
  final Object? Function()? zero = desc.zero;
  if (zero == null) {
    $panic(Iface($typeGoError, GoError('reflect: New of ${desc.display}')));
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
    case 'float32' || 'float64':
      return f<double>();
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
/// `*float64` casts to `$Ptr<double>?`, and Dart rejects a `Box<Object?>` there.
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
Iface? $makeValue(int handle, int length, int capacity) {
  final TypeDesc desc = _typeAt(handle);
  final Object? Function(int)? make = desc.make;
  if (make == null) {
    $panic(Iface($typeGoError, GoError('reflect: Make of ${desc.display}')));
  }
  final Object? out = make(capacity);
  if (out is Slice<Object?> && capacity != length) {
    return Iface(desc, out.range(0, length, capacity));
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
  final TypeDesc out = TypeDesc(
    '*${elem.name}',
    'ptr',
    _pointerMethods(elem),
    text: '*${elem.display}',
    elem: () => elem,
    methodTypes: elem.methodTypes,
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
  if (elem.methods.isEmpty) return const <String, Function>{};
  if (_isObjectPointer(elem)) return elem.methods;
  final Map<String, Function> out = <String, Function>{};
  for (final MapEntry<String, Function> e in elem.methods.entries) {
    final TypeDesc Function()? type = elem.methodTypes[e.key];
    final Function fn = e.value;
    out[e.key] = _looseFunc(
      (List<Object?> args) => <Object?>[
        Function.apply(fn, <Object?>[
          args.isEmpty ? null : _cellValue(args[0]),
          ...args.skip(1),
        ]),
      ],
      (type == null ? 0 : type().params.length) + 1,
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
    $panic(
      Iface(
        $typeGoError,
        const GoError('reflect: the type holds no such parameter'),
      ),
    );
  }
  return _descHandle(desc.params[index]());
}

/// A handle of the type of 1 result of a function type.
int $descOut(int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  if (index < 0 || index >= desc.results.length) {
    $panic(
      Iface(
        $typeGoError,
        const GoError('reflect: the type holds no such result'),
      ),
    );
  }
  return _descHandle(desc.results[index]());
}

/// Calls a function value with the arguments, and gives each result.
///
/// A function that gives up control is outside this call: the Dart form of such
/// a function gives a sentinel, and the call reads no scheduler.
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
    $panic(Iface($typeGoError, GoError('reflect: Call of ${desc.display}')));
  }
  if (spread && !desc.variadic) {
    $panic(
      Iface(
        $typeGoError,
        GoError(
          'reflect: CallSlice of ${desc.display}, a type that is not '
          'variadic',
        ),
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
    return _resultSlice(desc, Function.apply(fn.value as Function, raw));
  }
  final int fixed = desc.variadic ? desc.params.length - 1 : desc.params.length;
  final bool wrong = desc.variadic
      ? args.len < fixed
      : args.len != desc.params.length;
  if (wrong) {
    $panic(
      Iface(
        $typeGoError,
        GoError(
          'reflect: Call with ${args.len} input arguments, '
          'and the function takes ${desc.params.length}',
        ),
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
      $panic(Iface($typeGoError, GoError('reflect: Call of ${desc.display}')));
    }
    final Slice<Object?> tail = make(args.len - fixed) as Slice<Object?>;
    for (int i = fixed; i < args.len; i++) {
      final Iface? a = args[i];
      tail[i - fixed] = elem != null && elem.kind == 'interface' ? a : a?.value;
    }
    raw.add(tail);
  }
  return _resultSlice(desc, Function.apply(fn.value as Function, raw));
}

/// _resultSlice gives the results of a call as a slice of interface values.
Slice<Iface?> _resultSlice(TypeDesc desc, Object? out) {
  final int n = desc.results.length;
  final List<Iface?> results = <Iface?>[];
  if (n == 1) {
    results.add(_result(desc.results[0](), out));
  } else if (n > 1) {
    for (int i = 0; i < n; i++) {
      results.add(_result(desc.results[i](), _field(out, i, n)));
    }
  }
  return Slice.of(results);
}

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
    Iface(
      $typeGoError,
      GoError('reflect: a call of $count results is outside the subset'),
    ),
  );
}

/// The keys of the exported methods of a type, in the order of the name.
///
/// A key holds the name and the signature, and a parenthesis follows the name.
/// The parenthesis stands below each letter and each digit, thus the order of
/// the keys equals the order of the names.
List<String> _exportedMethodKeys(TypeDesc desc) {
  final List<String> keys = <String>[];
  for (final String key in desc.methods.keys) {
    final int cut = key.indexOf('(');
    final String name = cut < 0 ? key : key.substring(0, cut);
    if (name.isEmpty) continue;
    final String first = name[0];
    if (first.toUpperCase() == first && first.toLowerCase() != first) {
      keys.add(key);
    }
  }
  keys.sort();
  return keys;
}

/// The count of the exported methods of a type.
int $descNumMethod(int handle) => _exportedMethodKeys(_typeAt(handle)).length;

/// A handle of the function type of 1 exported method of a type. A method that
/// carries no type gives 0.
///
/// Go puts the receiver in front of each parameter of the type of a method of a
/// concrete type, and it writes no receiver for a method of an interface type.
int $descMethodType(int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  final List<String> keys = _exportedMethodKeys(desc);
  if (index < 0 || index >= keys.length) return 0;
  final TypeDesc Function()? type = desc.methodTypes[keys[index]];
  if (type == null) return 0;
  final TypeDesc sig = type();
  if (desc.kind == 'interface') return _descHandle(sig);
  return _descHandle(_withReceiver(desc, sig));
}

/// The func value of 1 exported method of a type, with the receiver in front of
/// each parameter. A type of an interface holds no such value, exactly as in Go.
///
/// The closure reads the adapter of the method table, and that adapter already
/// takes the receiver first. The signature carries no factory of a Dart type,
/// thus Call of the value works and an assertion of it to its function type
/// fails. Read decision 325.
Iface? $descMethodFunc(int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  if (desc.kind == 'interface') return null;
  final List<String> keys = _exportedMethodKeys(desc);
  if (index < 0 || index >= keys.length) {
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: ${desc.display} holds no method $index'),
      ),
    );
  }
  final String key = keys[index];
  final Function? fn = desc.methods[key];
  final TypeDesc Function()? type = desc.methodTypes[key];
  if (fn == null || type == null) {
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: no type of the method $key of ${desc.display}'),
      ),
    );
  }
  final TypeDesc full = _withReceiver(desc, type());
  return Iface(
    full,
    _typedFunc(
      full,
      (List<Object?> args) => _resultList(full, Function.apply(fn, args)),
    ),
  );
}

/// _withReceiver gives the function type of a method of a concrete type.
///
/// The type carries no factory of its Dart type, because no call reads it. Go
/// holds such a call under Method.Func, and that name stays out.
TypeDesc _withReceiver(TypeDesc recv, TypeDesc sig) => TypeDesc(
  _funcText(recv.name, sig, (TypeDesc d) => d.name),
  'func',
  const <String, Function>{},
  text: _funcText(recv.display, sig, (TypeDesc d) => d.display),
  params: <TypeDesc Function()>[() => recv, ...sig.params],
  results: sig.results,
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
  if (v is Slice<Object?>) {
    final TypeDesc? elem = value.desc.elem?.call();
    for (int i = 0; i < v.len; i++) {
      v[i] = elem == null ? null : _zeroOf(elem);
    }
    return;
  }
  $panic(
    Iface($typeGoError, GoError('reflect: Clear of ${value.desc.display}')),
  );
}

/// The name of 1 exported method of a type.
GoString $descMethodName(int handle, int index) {
  final TypeDesc desc = _typeAt(handle);
  final List<String> keys = _exportedMethodKeys(desc);
  if (index < 0 || index >= keys.length) {
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: ${desc.display} holds no method $index'),
      ),
    );
  }
  final String key = keys[index];
  final int cut = key.indexOf('(');
  return GoString.fromString(cut < 0 ? key : key.substring(0, cut));
}

/// A func value that binds 1 exported method of the type to the value.
Iface? $methodValue(Iface? value, int index) {
  if (value == null) $panicNil();
  final TypeDesc desc = value.desc;
  final List<String> keys = _exportedMethodKeys(desc);
  if (index < 0 || index >= keys.length) {
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: ${desc.display} holds no method $index'),
      ),
    );
  }
  final String key = keys[index];
  final Function? fn = desc.methods[key];
  final TypeDesc Function()? type = desc.methodTypes[key];
  if (fn == null || type == null) {
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: no type of the method $key of ${desc.display}'),
      ),
    );
  }
  final TypeDesc sig = type();
  return Iface(sig, _bindMethod(sig, fn, value.value));
}

/// _bindMethod gives a Dart closure of a method and its receiver.
///
/// The closure carries the Dart type of each parameter and of the result,
/// because the description of the signature holds the factory of decision 254.
/// A Go program that asserts the value to its function type then reads the
/// result. Read decision 228 for the earlier limit.
Function _bindMethod(TypeDesc sig, Function fn, Object? recv) => _typedFunc(
  sig,
  (List<Object?> args) =>
      _resultList(sig, Function.apply(fn, <Object?>[recv, ...args])),
);

/// _typedFunc builds a closure of the Dart type of a function type.
///
/// The body reads a list of the raw arguments and gives a list of the raw
/// results. A signature that carries no factory gives a closure of Object?
/// parameters, and an assertion of that value to a function type with a result
/// fails.
Function _typedFunc(TypeDesc sig, List<Object?> Function(List<Object?>) body) {
  final Function Function(List<Object?> Function(List<Object?>))? bind =
      sig.bind;
  if (bind != null) return bind(body);
  return _looseFunc(body, sig.params.length);
}

/// _resultList gives the raw results of a call as a list. A call of more than 1
/// result gives a Dart record, thus the list reads each member of it.
List<Object?> _resultList(TypeDesc sig, Object? out) {
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
    case 6:
      return (
        Object? a,
        Object? b,
        Object? c,
        Object? d,
        Object? e,
        Object? f,
      ) => _onlyResult(body(<Object?>[a, b, c, d, e, f]));
  }
  $panic(
    Iface(
      $typeGoError,
      GoError('reflect: a function of $arity parameters is outside the subset'),
    ),
  );
}

/// _onlyResult gives the single result of a call, or the whole list.
Object? _onlyResult(List<Object?> out) => out.length == 1 ? out[0] : out;

/// A func value of a function type. Each call of it runs the body with the
/// arguments, and the results of the body become the results of the call.
///
/// The description of the signature carries the factory of the Dart type, thus
/// the value fits a location of its Go function type. A function type that the
/// program never names holds no such factory, and MakeFunc of it panics.
Iface? $makeFunc(int handle, Object? body) {
  final TypeDesc desc = _typeAt(handle);
  if (desc.kind != 'func') {
    $panic(
      Iface($typeGoError, GoError('reflect: MakeFunc of ${desc.display}')),
    );
  }
  if (desc.bind == null) {
    $panic(
      Iface(
        $typeGoError,
        GoError(
          'reflect: MakeFunc of ${desc.display}, a type that the '
          'program never names',
        ),
      ),
    );
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
      final Slice<Iface?> out = run(Slice<Iface?>.of(input)) as Slice<Iface?>;
      // A function that gives up control is outside this call, exactly as it
      // is outside Value.Call. The sentinel of the scheduler names the cause.
      if ($blocked != null) {
        $panic(
          Iface(
            $typeGoError,
            GoError(
              'reflect: MakeFunc of ${desc.display} ran a body that gives up '
              'control',
            ),
          ),
        );
      }
      final int n = desc.results.length;
      if (out.len != n) {
        $panic(
          Iface(
            $typeGoError,
            GoError(
              'reflect: MakeFunc of ${desc.display} gave ${out.len} '
              'results, and the type holds $n',
            ),
          ),
        );
      }
      final List<Object?> raw = <Object?>[];
      for (int i = 0; i < n; i++) {
        final Iface? o = out[i];
        raw.add(desc.results[i]().kind == 'interface' ? o : o?.value);
      }
      return raw;
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
    $panic(
      Iface($typeGoError, GoError('reflect: MapKeys of ${value.desc.display}')),
    );
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
    $panic(
      Iface($typeGoError, GoError('reflect: $name of ${value.desc.display}')),
    );
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
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: MapIndex of ${target.desc.display}'),
      ),
    );
  }
  final TypeDesc? keyType = target.desc.key?.call();
  final Object? k = keyType?.kind == 'interface' ? key : key?.value;
  if (!m.has(k)) return null;
  final TypeDesc? elem = target.desc.elem?.call();
  final Object? out = m.get(k);
  if (elem == null || elem.kind == 'interface') return out as Iface?;
  return Iface(elem, out);
}

/// A map of the type.
Iface? $makeMap(int handle) {
  final TypeDesc desc = _typeAt(handle);
  final Object? Function(int)? make = desc.make;
  if (make == null) {
    $panic(Iface($typeGoError, GoError('reflect: MakeMap of ${desc.display}')));
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
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: SetMapIndex of ${target.desc.display}'),
      ),
    );
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

// The type constructors and the slice operations of the package reflect.
//
// A constructor gives a descriptor that no value of the program carries. Each
// name goes through the handle table first, thus a type that the program
// already holds keeps its own descriptor, with its zero and its make. A type
// that the program never holds gives a descriptor for the inspection alone,
// and MakeSlice, MakeMap, New, and Zero of it each panic with a clear text.
// Read docs/parity.md.

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
          _withType<Object?>(e, <T>() => Slice<T>.of(_zeros<T>(e, n))),
      withType: <R>(R Function<T>() f) =>
          _withType<R>(e, <T>() => f<Slice<T>>()),
    ),
  );
}

/// A handle of the function type of a parameter list and a result list.
///
/// The Dart type of a function holds the type of each parameter, and that
/// type lives only in the emitted code. A signature that the program already
/// names thus gives its own descriptor, with its bind factory. A signature
/// that no code names gives a descriptor with no bind, and reflect.MakeFunc
/// of it panics with the cause.
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
int $arrayOf(int length, int elem) {
  final TypeDesc e = _typeAt(elem);
  final String name = '[$length]${e.name}';
  return _syntheticHandle(
    () => TypeDesc(
      name,
      'array',
      const <String, Function>{},
      text: '[$length]${e.display}',
      elem: () => e,
      length: length,
      comparable: e.comparable,
      zero: () => _withType<Object?>(e, <T>() => _zeros<T>(e, length)),
      withType: <R>(R Function<T>() f) =>
          _withType<R>(e, <T>() => f<List<T>>()),
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
          <V>() => GoMap<K, V>.nil(() => _zeroOf(e) as V),
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
/// A key of an array type compares by value in Go, and a Dart list compares by
/// identity. A key of a pointer to a struct compares by identity in Go, and the
/// Dart class of a struct compares each field.
bool Function(K, K)? _keyEquals<K>(TypeDesc k) {
  if (k.kind == 'array') return (K a, K b) => $goEqual(a, b, k);
  if (k.kind == 'ptr' && _isObjectPointer(k.elem!())) {
    return (K a, K b) => identical(a, b);
  }
  return null;
}

/// _keyHash gives the hash of a map key, and it pairs with _keyEquals.
int Function(K)? _keyHash<K>(TypeDesc k) {
  if (k.kind == 'array') return (K a) => $goHash(a, k);
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
  return _syntheticHandle(
    () => TypeDesc(
      '$head${e.name}',
      'chan',
      const <String, Function>{},
      text: '$head${e.display}',
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
    $panic(
      Iface($typeGoError, GoError('reflect: MakeChan of ${desc.display}')),
    );
  }
  if (desc.dir != 0 && desc.dir != 3) {
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: MakeChan of a channel of 1 direction'),
      ),
    );
  }
  return Iface(
    desc,
    _withType<Object?>(
      elem,
      <T>() => Chan<T>(buffer, () => _zeroOf(elem) as T),
    ),
  );
}

/// _chanOf gives the channel of a value, and it panics for each other kind.
Chan<Object?> _chanOf(Iface? value, String call) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is! Chan<Object?>) {
    $panic(
      Iface($typeGoError, GoError('reflect: $call of ${value.desc.display}')),
    );
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
        $panic(
          Iface(
            $typeGoError,
            const GoError('reflect.Select: multiple default cases'),
          ),
        );
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
final class _SelectResume implements WaitFrame {
  final Frame inner;
  final List<SelectCase> cases;
  final List<int> order;
  final List<TypeDesc?> descs;

  _SelectResume(this.inner, this.cases, this.order, this.descs);

  @override
  String get $reason => 'select';

  @override
  Object? resume() {
    final Object? out = inner.resume();
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
List<T> _zeros<T>(TypeDesc elem, int n) =>
    List<T>.generate(n, (int _) => _zeroOf(elem) as T, growable: true);

/// The capacity of a slice, of an array, or of a channel.
int $valueCap(Iface? value) {
  if (value == null) return 0;
  final Object? v = value.value;
  if (v is Slice<Object?>) return v.cap;
  if (v is List<Object?>) return v.length;
  // The capacity of a channel is the size of its buffer.
  if (v is Chan<Object?>) return v.capacity;
  return $valueLen(value);
}

/// _sliceParts gives the header of a slice value, or null.
Slice<Object?>? _sliceOfValue(Iface? value) {
  final Object? v = value?.value;
  return v is Slice<Object?> ? v : null;
}

/// _freshSlice builds a slice of the type with the length, through the make of
/// the descriptor. A type that the program never holds gives a panic.
Slice<Object?> _freshSlice(TypeDesc desc, int length) {
  final Object? Function(int)? make = desc.make;
  if (make == null) {
    $panic(Iface($typeGoError, GoError('reflect: Append of ${desc.display}')));
  }
  return make(length) as Slice<Object?>;
}

/// Appends each value to a slice and gives the new slice.
Iface? $appendValue(Iface? slice, Slice<Iface?> items) {
  if (slice == null) $panicNil();
  final TypeDesc desc = slice.desc;
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) {
    $panic(Iface($typeGoError, GoError('reflect: Append of ${desc.display}')));
  }
  final Slice<Object?>? held = _sliceOfValue(slice);
  final int n = held == null ? 0 : held.len;
  final Slice<Object?> out = _freshSlice(desc, n + items.len);
  for (int i = 0; i < n; i++) {
    out[i] = held![i];
  }
  for (int i = 0; i < items.len; i++) {
    final Iface? x = items[i];
    out[n + i] = elem.kind == 'interface' ? x : x?.value;
  }
  return Iface(desc, out);
}

/// Appends each element of the second slice to the first one.
Iface? $appendSliceValue(Iface? slice, Iface? add) {
  if (slice == null) $panicNil();
  final TypeDesc desc = slice.desc;
  if (desc.elem == null) {
    $panic(
      Iface($typeGoError, GoError('reflect: AppendSlice of ${desc.display}')),
    );
  }
  final Slice<Object?>? held = _sliceOfValue(slice);
  final Slice<Object?>? more = _sliceOfValue(add);
  final int n = held == null ? 0 : held.len;
  final int m = more == null ? 0 : more.len;
  final Slice<Object?> out = _freshSlice(desc, n + m);
  for (int i = 0; i < n; i++) {
    out[i] = held![i];
  }
  for (int i = 0; i < m; i++) {
    out[n + i] = more![i];
  }
  return Iface(desc, out);
}

/// Copies each element of the source into the destination, and gives the count
/// that it wrote. The source may be a string, and the destination is then a
/// byte slice.
int $copyValue(Iface? dst, Iface? src) {
  if (dst == null) $panicNil();
  final Object? to = dst.value;
  if (to is! Slice<Object?> && to is! List<Object?>) return 0;
  final Object? from = src?.value;
  final int room = to is Slice<Object?> ? to.len : (to as List<Object?>).length;
  if (from is GoString) {
    final int n = room < from.length ? room : from.length;
    for (int i = 0; i < n; i++) {
      _writeElem(to, i, $gi(from[i]));
    }
    return n;
  }
  final int have = from is Slice<Object?>
      ? from.len
      : (from is List<Object?> ? from.length : 0);
  final int n = room < have ? room : have;
  for (int i = 0; i < n; i++) {
    _writeElem(to, i, _readElem(from, i));
  }
  return n;
}

Object? _readElem(Object? v, int i) {
  if (v is Slice<Object?>) return v[i];
  return (v as List<Object?>)[i];
}

void _writeElem(Object? v, int i, Object? x) {
  if (v is Slice<Object?>) {
    v[i] = x;
    return;
  }
  (v as List<Object?>)[i] = x;
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
  return false;
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
  if (_numberKinds.contains(a.kind) && _numberKinds.contains(b.kind)) {
    return true;
  }
  if (a.kind == 'string' && b.kind == 'slice') {
    final String elem = b.elem?.call().kind ?? '';
    return elem == 'uint8' || elem == 'int32';
  }
  if (b.kind == 'string') {
    if (_numberKinds.contains(a.kind)) return true;
    if (a.kind == 'slice') {
      final String elem = a.elem?.call().kind ?? '';
      return elem == 'uint8' || elem == 'int32';
    }
  }
  if (a.kind == b.kind && a.kind == 'struct') {
    return a.name == b.name;
  }
  // Go converts between 2 types whose underlying types are identical. A
  // defined slice type and the slice itself hold 1 Dart list, thus the
  // conversion performs no operation. `type Raw []byte` holds that shape.
  if (a.kind == b.kind) {
    switch (a.kind) {
      case 'slice' || 'chan':
        return _sameDesc(a.elem, b.elem);
      case 'array':
        return a.length == b.length && _sameDesc(a.elem, b.elem);
      case 'map':
        return _sameDesc(a.key, b.key) && _sameDesc(a.elem, b.elem);
    }
  }
  return false;
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

/// The index of a field of a struct type, by name. A name that no field holds
/// gives -1.
int $descFieldByName(int handle, GoString name) {
  final TypeDesc desc = _typeAt(handle);
  final String want = name.toDartString();
  for (int i = 0; i < desc.fields.length; i++) {
    if (desc.fields[i].name == want) return i;
  }
  return -1;
}

// The value operations that text/template reads.

/// A pointer to the location that a Value names.
///
/// The loc names the kind of the location: 1 for a pointer, 2 for a field of a
/// struct, and 3 for an element of a slice or of an array.
Iface? $valueAddr(Iface? value, Iface? owner, int index, int loc) {
  if (loc == 1) return owner;
  if (owner == null) $panicNil();
  final TypeDesc ownerDesc = owner.desc;
  if (loc == 2) {
    if (index < 0 || index >= ownerDesc.fields.length) {
      $panic(Iface($typeGoError, GoError('reflect: Addr of a field')));
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
    $panic(Iface($typeGoError, GoError('reflect: Addr of an element')));
  }
  final Object? v = owner.value;
  if (_isObjectPointer(elem)) {
    return Iface($ptrDesc(elem), _readElem(v, index));
  }
  return Iface(
    $ptrDesc(elem),
    _typedCell(elem, v, index, _readElem, _writeElem),
  );
}

/// A bound method of a value, by name. A name that no method holds gives null.
Iface? $methodByName(Iface? value, GoString name) {
  if (value == null) return null;
  final String want = name.toDartString();
  for (final String key in value.desc.methods.keys) {
    final int cut = key.indexOf('(');
    if ((cut < 0 ? key : key.substring(0, cut)) != want) continue;
    final TypeDesc Function()? type = value.desc.methodTypes[key];
    if (type == null) return null;
    final TypeDesc sig = type();
    return Iface(sig, _bindMethod(sig, value.desc.methods[key]!, value.value));
  }
  return null;
}

/// Reports whether 2 values hold the same type and the same value.
bool $valueEqual(Iface? a, Iface? b) {
  if (a == null || b == null) return a == null && b == null;
  if (a.desc.name != b.desc.name) return false;
  return $deepEqual(a, b);
}

/// The value of a type conversion.
Iface? $valueConvert(Iface? value, int handle) {
  final TypeDesc to = _typeAt(handle);
  if (value == null) return Iface(to, to.zero?.call());
  final TypeDesc from = value.desc;
  if (from.name == to.name) return Iface(to, value.value);
  if (!$descConvertible(_descHandle(from), handle)) {
    $panic(
      Iface(
        $typeGoError,
        GoError('reflect: cannot convert ${from.display} to ${to.display}'),
      ),
    );
  }
  return Iface(to, _converted(from, to, value.value));
}

Object? _converted(TypeDesc from, TypeDesc to, Object? v) {
  if (to.kind == 'string') {
    if (_numberKinds.contains(from.kind)) {
      return GoString.fromString(String.fromCharCode(($ix(v as GoInt))));
    }
    if (v is Slice<Object?>) {
      final List<int> bytes = <int>[
        for (int i = 0; i < v.len; i++) $ix(v[i] as GoInt),
      ];
      final String elem = from.elem?.call().kind ?? '';
      if (elem == 'int32') {
        return GoString.fromString(String.fromCharCodes(bytes));
      }
      return GoString.fromBytes(bytes);
    }
    return v;
  }
  if (from.kind == 'string' && to.kind == 'slice') {
    final GoString s = v as GoString;
    final String elem = to.elem?.call().kind ?? '';
    final List<Object?> out = <Object?>[];
    if (elem == 'int32') {
      for (final int r in s.toDartString().runes) {
        out.add($gi(r));
      }
    } else {
      for (int i = 0; i < s.length; i++) {
        out.add($gi(s[i]));
      }
    }
    return Slice<Object?>.of(out);
  }
  if (_numberKinds.contains(to.kind)) {
    final double x = v is GoInt ? $ix(v).toDouble() : (v as double);
    if (to.kind == 'float32' || to.kind == 'float64') return x;
    return $gi(x.toInt());
  }
  return v;
}

/// The slice of a value, with the low, the high, and the maximum.
Iface? $valueSlice(Iface? value, int low, int high, int max) {
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is GoString) {
    return Iface(value.desc, $substring(v, low, high));
  }
  if (v is Slice<Object?>) {
    return Iface(value.desc, v.range(low, high, max));
  }
  $panic(
    Iface($typeGoError, GoError('reflect: Slice of ${value.desc.display}')),
  );
}

/// The complex value inside an interface value.
GoComplex $asComplex(Iface? value) {
  final Object? v = value?.value;
  if (v is GoComplex) return v;
  if (v is double) return GoComplex(v, 0);
  if (v is GoInt) return GoComplex($ix(v).toDouble(), 0);
  return GoComplex.zero;
}

/// A complex value of the type of a handle.
Iface? $withComplex(int handle, GoComplex x) => Iface(_typeAt(handle), x);

/// A number that names the object of a value. Dart gives no machine address,
/// thus the number is the identity hash of the object. Read docs/parity.md.
int $valuePointer(Iface? value) {
  final Object? v = value?.value;
  if (v == null) return 0;
  return identityHashCode(v);
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
final class _RecvResume implements WaitFrame {
  final Frame inner;
  final TypeDesc desc;

  _RecvResume(this.inner, this.desc);

  @override
  String get $reason => 'chan receive';

  @override
  Object? resume() {
    final Object? out = inner.resume();
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

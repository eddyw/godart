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
  final Object? target = desc.kind == 'struct' ? zero() : Box<Object?>(zero());
  return Iface(_pointerTo(desc), target);
}

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

TypeDesc _pointerTo(TypeDesc elem) {
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
    const <String, Function>{},
    text: '*${elem.display}',
    elem: () => elem,
    zero: () => null,
  );
  _pointers[elem.name] = out;
  return out;
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
Slice<Iface?> $callValue(Iface? fn, Slice<Iface?> args) {
  if (fn == null) $panicNil();
  final TypeDesc desc = fn.desc;
  if (desc.kind != 'func') {
    $panic(Iface($typeGoError, GoError('reflect: Call of ${desc.display}')));
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
  final Object? out = Function.apply(fn.value as Function, raw);
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
  return Iface(sig, _bindMethod(fn, value.value, sig.params.length));
}

/// _bindMethod gives a plain Dart closure of a method and its receiver.
///
/// A BoundMethod holds the pair, and $callValue reads it. A Go program that
/// asserts the value to its function type needs a closure, and a closure of
/// Object? parameters fits each such type: Dart reads a parameter type
/// against the other direction, thus Object? accepts each argument.
Function _bindMethod(Function fn, Object? recv, int arity) {
  switch (arity) {
    case 0:
      return () => fn(recv);
    case 1:
      return (Object? a) => fn(recv, a);
    case 2:
      return (Object? a, Object? b) => fn(recv, a, b);
    case 3:
      return (Object? a, Object? b, Object? c) => fn(recv, a, b, c);
    case 4:
      return (Object? a, Object? b, Object? c, Object? d) =>
          fn(recv, a, b, c, d);
    case 5:
      return (Object? a, Object? b, Object? c, Object? d, Object? e) =>
          fn(recv, a, b, c, d, e);
    case 6:
      return (
        Object? a,
        Object? b,
        Object? c,
        Object? d,
        Object? e,
        Object? f,
      ) => fn(recv, a, b, c, d, e, f);
  }
  $panic(
    Iface(
      $typeGoError,
      GoError('reflect: a method of $arity parameters is outside the subset'),
    ),
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
      zero: () => Slice<Object?>.nil(),
      make: (int n) => Slice<Object?>.of(_zeros(e, n)),
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
      zero: () => _zeros(e, length),
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
      zero: () => GoMap<Object?, Object?>.nil(() => _zeroOf(e)),
      make: (int _) => GoMap<Object?, Object?>(() => _zeroOf(e)),
    ),
  );
}

/// A handle of the pointer type of an element type.
int $pointerTo(int elem) {
  final TypeDesc e = _typeAt(elem);
  return _syntheticHandle(() => _pointerTo(e));
}

/// _zeroOf gives the zero value of a type, or null for a type that holds no
/// zero builder.
Object? _zeroOf(TypeDesc desc) {
  if (desc.kind == 'interface') return null;
  return desc.zero?.call();
}

/// _zeros gives a list of the length that holds the zero value of the type.
///
/// The list holds Object?, and not the Dart type of the element, because the
/// Dart type of a Go type lives only in the emitted code. Read docs/parity.md.
List<Object?> _zeros(TypeDesc elem, int n) =>
    List<Object?>.filled(n, _zeroOf(elem), growable: true);

/// The capacity of a slice, of an array, or of a channel.
int $valueCap(Iface? value) {
  if (value == null) return 0;
  final Object? v = value.value;
  if (v is Slice<Object?>) return v.cap;
  if (v is List<Object?>) return v.length;
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
  return false;
}

/// The direction of a channel type: 3 for both, 2 for receive, 1 for send.
int $descChanDir(int handle) {
  final TypeDesc desc = _typeAt(handle);
  if (desc.kind != 'chan') return 0;
  // The descriptor of the runtime holds no direction, thus each channel type
  // reads as a channel of both directions. Read docs/parity.md.
  return 3;
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
    // The Dart form of a pointer to a struct is the struct itself.
    if (field.kind == 'struct') {
      return Iface(_pointerTo(field), f.get(owner.value));
    }
    return Iface(
      _pointerTo(field),
      $Ref<Object?, Object?>(
        owner.value,
        0,
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
  if (elem.kind == 'struct') {
    return Iface(_pointerTo(elem), _readElem(v, index));
  }
  return Iface(
    _pointerTo(elem),
    $Ref<Object?, Object?>(
      v,
      index,
      (Object? o, int i) => _readElem(o, i),
      (Object? o, int i, Object? x) => _writeElem(o, i, x),
    ),
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
    return Iface(
      sig,
      _bindMethod(value.desc.methods[key]!, value.value, sig.params.length),
    );
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
  if (value == null) $panicNil();
  final Object? v = value.value;
  if (v is! Chan<Object?>) {
    $panic(
      Iface($typeGoError, GoError('reflect: Recv of ${value.desc.display}')),
    );
  }
  final TypeDesc? elem = value.desc.elem?.call();
  final (Object?, bool) got = v.receive();
  if (elem == null || elem.kind == 'interface') {
    return (got.$1 as Iface?, got.$2);
  }
  return (Iface(elem, got.$1), got.$2);
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
  // or the word map, chan, func, or interface.
  if (head.contains('[') || head.contains('*') || head.contains(' ')) {
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

import 'dict.dart';
import 'panic.dart';
import 'string.dart';

/// The description of 1 struct field. It gives the `reflect` subset the name,
/// the tag, and access to the value.
final class FieldDesc {
  final String name;
  final String tag;
  final bool exported;
  final TypeDesc Function() type;
  final Object? Function(Object?) get;
  final void Function(Object?, Object?) set;

  const FieldDesc(
    this.name,
    this.tag,
    this.exported,
    this.type,
    this.get,
    this.set,
  );
}

/// $noRead stands in place of the getter of a field that the emitted Dart
/// cannot name. A Dart library holds each name that starts with an underscore,
/// thus another library cannot read an unexported field of a type of that
/// library.
Object? $noRead(Object? r) => null;

/// $noWrite stands in place of the setter of such a field.
void $noWrite(Object? r, Object? v) {}

/// The run time description of a Go type.
///
/// The name holds the full Go name of the type, for example
/// `example.com/m.point`. A type assertion and a type switch compare the name,
/// thus the test needs no identity of the object.
final class TypeDesc {
  /// The full name of the type, with the import path of its package. A type
  /// assertion compares this name, thus it must be unique.
  final String name;

  /// The name that Go writes, with the package name and not the import path.
  final String text;

  /// The kind, in the words of the Go package reflect: `struct`, `int`,
  /// `string`, `slice`, `map`, `ptr`, and the other kinds.
  final String kind;
  final Map<String, Function> methods;
  final List<FieldDesc> fields;

  /// The element type of a slice, an array, a pointer, a channel, or a map.
  final TypeDesc Function()? elem;

  /// The key type of a map.
  final TypeDesc Function()? key;

  /// The count of the elements of an array type. Each other kind holds 0.
  final int length;

  /// The parameter types and the result types of a function type.
  final List<TypeDesc Function()> params;
  final List<TypeDesc Function()> results;

  /// zero builds the zero value of the type. A decoder writes a fresh value of
  /// a field of a struct through it.
  final Object? Function()? zero;

  /// make builds a slice of the length, or an empty map. A decoder reads it,
  /// because the Dart type of the element lives only in the emitted code.
  final Object? Function(int)? make;

  /// True when Go compares 2 values of the type. A slice, a map, and a
  /// function each give false, and so does a struct that holds one.
  final bool comparable;

  const TypeDesc(
    this.name,
    this.kind,
    this.methods, {
    this.text = '',
    this.fields = const <FieldDesc>[],
    this.elem,
    this.key,
    this.length = 0,
    this.params = const <TypeDesc Function()>[],
    this.results = const <TypeDesc Function()>[],
    this.zero,
    this.make,
    this.comparable = true,
  });

  /// The name for a message. Go writes the package name, and not the path.
  String get display => text.isEmpty ? name : text;

  /// The text and the comparable flag, for the package internal/reflectlite.
  (GoString, bool) get info => (GoString.fromString(display), comparable);

  @override
  String toString() => display;
}

/// A Go interface value: the type descriptor and the value.
final class Iface {
  final TypeDesc desc;
  final Object? value;

  const Iface(this.desc, this.value);

  @override
  bool operator ==(Object other) =>
      other is Iface && desc.name == other.desc.name && value == other.value;

  @override
  int get hashCode => Object.hash(desc.name, value);

  @override
  String toString() => '$value';
}

/// The zero value of a predeclared type. A tear-off of a top level function is
/// a constant expression of Dart, thus each descriptor below stays const.
Object? $zeroInt() => 0;
Object? $zeroFloat() => 0.0;
Object? $zeroBool() => false;
Object? $zeroString() => GoString.empty;

const TypeDesc $typeBool = TypeDesc(
  'bool',
  'bool',
  <String, Function>{},
  zero: $zeroBool,
);
const TypeDesc $typeInt = TypeDesc(
  'int',
  'int',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeInt8 = TypeDesc(
  'int8',
  'int8',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeInt16 = TypeDesc(
  'int16',
  'int16',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeInt32 = TypeDesc(
  'int32',
  'int32',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeInt64 = TypeDesc(
  'int64',
  'int64',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeUint = TypeDesc(
  'uint',
  'uint',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeUint8 = TypeDesc(
  'uint8',
  'uint8',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeUint16 = TypeDesc(
  'uint16',
  'uint16',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeUint32 = TypeDesc(
  'uint32',
  'uint32',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeUint64 = TypeDesc(
  'uint64',
  'uint64',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeUintptr = TypeDesc(
  'uintptr',
  'uintptr',
  <String, Function>{},
  zero: $zeroInt,
);
const TypeDesc $typeFloat32 = TypeDesc(
  'float32',
  'float32',
  <String, Function>{},
  zero: $zeroFloat,
);
const TypeDesc $typeFloat64 = TypeDesc(
  'float64',
  'float64',
  <String, Function>{},
  zero: $zeroFloat,
);
const TypeDesc $typeString = TypeDesc(
  'string',
  'string',
  <String, Function>{},
  zero: $zeroString,
);

/// True when the type holds each method of an interface.
bool $implements(Iface? x, List<String> methods) {
  if (x == null) return false;
  for (final m in methods) {
    if (!x.desc.methods.containsKey(m)) return false;
  }
  return true;
}

bool $isType(Iface? x, String name) => x != null && x.desc.name == name;

/// The value of a type assertion that gives 1 value. A wrong type panics.
Object? $assertType(Iface? x, String name) {
  if (x == null) {
    $panic(GoError('interface conversion: interface is nil, not $name'));
  }
  if (x.desc.name != name) {
    $panic(
      GoError('interface conversion: interface is ${x.desc.name}, not $name'),
    );
  }
  return x.value;
}

/// The value of an assertion to an interface type.
Iface $assertIface(Iface? x, List<String> methods, String name) {
  if (!$implements(x, methods)) {
    final String got = x == null ? 'nil' : x.desc.name;
    $panic(GoError('interface conversion: $got does not implement $name'));
  }
  return x!;
}

/// The method of an interface value.
Function $method(Iface? x, String name) {
  if (x == null) $panicNil();
  final Function? m = x.desc.methods[name];
  if (m == null) $panicNil();
  return m;
}

/// Calls a method of an interface value when the receiver is not a simple
/// name. The call reads the receiver 1 time, thus a side effect runs 1 time.
Object? $call(Iface? x, String name, List<Object?> args) {
  final Function m = $method(x, name);
  return Function.apply(m, <Object?>[x!.value, ...args]);
}

/// $noMethod stands in place of an unexported method of a type of another
/// package. A Dart library holds each name that starts with an underscore, thus
/// another library reaches no such method.
Never $noMethod(String name) =>
    $panic(GoError('the method $name belongs to another package'));

/// $ifaceMethod stands in the method table of the description of an interface
/// type. Such a table holds the key of each method that the interface needs,
/// and no call reaches the value.
Never $ifaceMethod() =>
    $panic(const GoError('the method belongs to an interface type'));

/// Reports whether the value holds the type of a dictionary. An interface type
/// argument reads the method set, and each other type argument reads the name.
bool $isDictType(Iface? x, TypeDesc d) {
  if (d.kind == 'interface') return $implements(x, d.methods.keys.toList());
  return $isType(x, d.name);
}

/// The value of a type assertion to a type parameter. An interface type
/// argument keeps the pair, and each other type argument gives the value.
T $dictValue<T>(Iface? x, Dict<T> d) {
  if (d.desc.kind == 'interface') return x as T;
  return x!.value as T;
}

/// The value of a type assertion to a type parameter with no second result. A
/// value of another type stops the program.
T $assertDict<T>(Iface? x, Dict<T> d) {
  if (!$isDictType(x, d.desc)) {
    $panic(
      GoError(
        'interface conversion: interface {} is '
        '${x == null ? 'nil' : x.desc.display}, not ${d.desc.display}',
      ),
    );
  }
  return $dictValue<T>(x, d);
}

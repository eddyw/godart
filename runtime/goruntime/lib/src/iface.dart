import 'panic.dart';

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

  const TypeDesc(
    this.name,
    this.kind,
    this.methods, {
    this.text = '',
    this.fields = const <FieldDesc>[],
    this.elem,
    this.key,
  });

  /// The name for a message. Go writes the package name, and not the path.
  String get display => text.isEmpty ? name : text;

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

const TypeDesc $typeBool = TypeDesc('bool', 'bool', <String, Function>{});
const TypeDesc $typeInt = TypeDesc('int', 'int', <String, Function>{});
const TypeDesc $typeInt8 = TypeDesc('int8', 'int8', <String, Function>{});
const TypeDesc $typeInt16 = TypeDesc('int16', 'int16', <String, Function>{});
const TypeDesc $typeInt32 = TypeDesc('int32', 'int32', <String, Function>{});
const TypeDesc $typeInt64 = TypeDesc('int64', 'int64', <String, Function>{});
const TypeDesc $typeUint = TypeDesc('uint', 'uint', <String, Function>{});
const TypeDesc $typeUint8 = TypeDesc('uint8', 'uint8', <String, Function>{});
const TypeDesc $typeUint16 = TypeDesc('uint16', 'uint16', <String, Function>{});
const TypeDesc $typeUint32 = TypeDesc('uint32', 'uint32', <String, Function>{});
const TypeDesc $typeUint64 = TypeDesc('uint64', 'uint64', <String, Function>{});
const TypeDesc $typeUintptr = TypeDesc(
  'uintptr',
  'uintptr',
  <String, Function>{},
);
const TypeDesc $typeFloat32 = TypeDesc(
  'float32',
  'float32',
  <String, Function>{},
);
const TypeDesc $typeFloat64 = TypeDesc(
  'float64',
  'float64',
  <String, Function>{},
);
const TypeDesc $typeString = TypeDesc('string', 'string', <String, Function>{});

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

import 'panic.dart';

/// The run time description of a Go type.
///
/// The name holds the full Go name of the type, for example
/// `example.com/m.point`. A type assertion and a type switch compare the name,
/// thus the test needs no identity of the object.
final class TypeDesc {
  final String name;
  final Map<String, Function> methods;

  const TypeDesc(this.name, this.methods);

  @override
  String toString() => name;
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

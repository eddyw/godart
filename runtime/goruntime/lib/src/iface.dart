// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'box.dart';
import 'complex.dart';
import 'dict.dart';
import 'goint.dart';
import 'int.dart';
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

  /// True for an embedded field. The package reflect reads it as Anonymous.
  final bool embedded;

  /// The import path of the package that declares an unexported field. An
  /// exported field holds an empty text.
  ///
  /// The package reflect reads it as PkgPath, and an unnamed struct type holds
  /// no package of its own.
  final String pkg;

  const FieldDesc(
    this.name,
    this.tag,
    this.exported,
    this.type,
    this.get,
    this.set, {
    this.embedded = false,
    this.pkg = '',
  });
}

final class ReflectedMethod {
  final String key;
  final String pkg;
  final TypeDesc Function()? type;
  final Object? Function(Iface, List<Object?>) invoke;

  const ReflectedMethod(this.key, this.pkg, this.type, this.invoke);
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

  /// Package identity for assertion errors, including unnamed private fields.
  final String assertionPkg;

  /// The kind, in the words of the Go package reflect: `struct`, `int`,
  /// `string`, `slice`, `map`, `ptr`, and the other kinds.
  final String kind;
  final Map<String, Function> methods;

  /// Receiver access for promoted methods of embedded interfaces.
  final Map<String, Iface? Function(Object?)> forwardMethods;

  /// The function type of each method, under its method key.
  /// The package reflect reads it, and each other path reads only [methods].
  final Map<String, TypeDesc Function()> methodTypes;
  final String methodPkg;
  final List<FieldDesc> fields;

  /// Runtime-created structs can expose duplicate method entries.
  final List<ReflectedMethod>? reflectionMethods;

  /// The element type of a slice, an array, a pointer, a channel, or a map.
  final TypeDesc Function()? elem;

  /// The key type of a map.
  final TypeDesc Function()? key;

  /// The count of the elements of an array type. Each other kind holds 0.
  final GoInt length;

  /// The direction of a channel type: 3 for both, 2 for send, and 1 for
  /// receive. Each other kind holds 0, and so does a channel description that
  /// an older build wrote.
  final int dir;

  /// The parameter types and the result types of a function type.
  final List<TypeDesc Function()> params;
  final List<TypeDesc Function()> results;

  /// True for a function type whose last parameter is variadic. The package
  /// reflect reads it.
  final bool variadic;

  /// zero builds the zero value of the type. A decoder writes a fresh value of
  /// a field of a struct through it.
  final Object? Function()? zero;

  /// Copies all struct fields, even private fields of generic instances.
  final void Function(Object?, Object?)? assign;

  /// make builds a slice of the length, or an empty map. A decoder reads it,
  /// because the Dart type of the element lives only in the emitted code.
  final Object? Function(int)? make;

  /// True when Go compares 2 values of the type. A slice, a map, and a
  /// function each give false, and so does a struct that holds one.
  final bool comparable;

  /// The description of the pointer to the type, when that pointer holds a
  /// method that the type does not hold. A method with a pointer receiver
  /// stays out of the value method set, and json reads such a method, because
  /// Unmarshal writes through a pointer.
  final TypeDesc Function()? ptr;

  /// Builds a pointer to the zero value of the element type. The description of
  /// a pointer type holds it under the same condition as [ptr].
  final Object? Function()? newPtr;

  /// Calls the function with the Dart type of a value of the type.
  ///
  /// The Dart type of a slice, a map, a channel, a function, and a pointer
  /// holds the name of its element type, and that name lives only in the
  /// emitted code. The package reflect builds a typed cell through this
  /// field, thus Value.Addr and reflect.New each give a pointer that a type
  /// assertion reads. The kind alone names the Dart type of each other type,
  /// thus only these 5 carry the field.
  final R Function<R>(R Function<T>())? withType;

  /// Names a finite field representation for recursive local structs.
  final R Function<R>(R Function<T>())? withStorageType;

  /// Builds a closure that carries the Dart type of each parameter and of the
  /// result of a function type.
  ///
  /// The body reads a list of the raw arguments and gives a list of the raw
  /// results. reflect.Value.Method, reflect.Value.MethodByName, and
  /// reflect.MakeFunc each build a func value through the field, thus a type
  /// assertion of such a value to its Go function type reads. A plain closure
  /// of Object? parameters gives a dynamic result, and Dart rejects that value
  /// at a location of a function type with a result. Only a function type
  /// carries the field.
  final Function Function(List<Object?> Function(List<Object?>))? bind;

  /// Source metadata names result fields that Dart cannot index at runtime.
  final List<Object?> Function(Object?)? unpack;

  /// Builds a pointer view over the original storage.
  final Object Function(TypeDesc, Object)? view;

  const TypeDesc(
    this.name,
    this.kind,
    this.methods, {
    this.text = '',
    this.assertionPkg = '',
    this.methodTypes = const <String, TypeDesc Function()>{},
    this.methodPkg = '',
    this.forwardMethods = const <String, Iface? Function(Object?)>{},
    this.fields = const <FieldDesc>[],
    this.reflectionMethods,
    this.elem,
    this.key,
    this.length = GoInt.zero,
    this.dir = 0,
    this.params = const <TypeDesc Function()>[],
    this.results = const <TypeDesc Function()>[],
    this.variadic = false,
    this.zero,
    this.assign,
    this.make,
    this.comparable = true,
    this.ptr,
    this.newPtr,
    this.withType,
    this.withStorageType,
    this.bind,
    this.unpack,
    this.view,
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
      other is Iface &&
      desc.name == other.desc.name &&
      $goEqual(value, other.value, desc);

  @override
  int get hashCode => Object.hash(desc.name, $goHash(value, desc));

  @override
  String toString() => '$value';
}

/// The Go equality of 2 values of 1 type. A comparison of a value whose type
/// holds no identity panics, and the panic names the type of the comparison.
///
/// root carries that type through the walk of the fields. Go names the type
/// that the code compares, and not the field that holds no identity.
bool $goEqual(Object? a, Object? b, TypeDesc d, [TypeDesc? root]) {
  final TypeDesc top = root ?? d;
  switch (d.kind) {
    case 'slice' || 'map' || 'func':
      $panic($runtimeError('comparing uncomparable type ${top.display}'));
    case 'array':
      final List<Object?> ea = a! as List<Object?>;
      final List<Object?> eb = b! as List<Object?>;
      final TypeDesc? elem = d.elem?.call();
      for (var i = 0; i < ea.length; i++) {
        if (!$goEqual(ea[i], eb[i], elem!, top)) return false;
      }
      return true;
    case 'struct':
      for (final FieldDesc f in d.fields) {
        if (!$goEqual(f.get(a), f.get(b), f.type.call(), top)) return false;
      }
      return true;
    case 'interface':
      final ia = a as Iface?;
      final ib = b as Iface?;
      if (ia == null || ib == null) return ia == null && ib == null;
      if (ia.desc.name != ib.desc.name) return false;
      // The dynamic type of the interface value takes over, exactly as Go
      // reports it.
      return $goEqual(ia.value, ib.value, ia.desc);
    case 'ptr':
      final String? elem = d.elem?.call().kind;
      if (elem == 'struct' || elem == 'array') {
        if (a is $Ptr<Object?>) a = a.v;
        if (b is $Ptr<Object?>) b = b.v;
      }
      // Go compares 2 pointers by the location that each 1 holds. A cell
      // holds that location, and the == operator of the cell compares it.
      // Each other pointer is the object at the location itself, thus
      // identical of Dart compares the 2.
      if (a is $Ptr<Object?>) return a == b;
      return identical(a, b);
  }
  return a == b;
}

/// True while a map write reads the entry of its key. Go writes a text of its
/// own for the hash of a key that holds no identity in a write. All Go code
/// runs in 1 isolate, and no goroutine switch happens inside a hash, thus 1
/// flag carries the form.
bool $hashInWrite = false;

/// The hash of a Go value of 1 type. It holds the rule of [$goEqual]: 2 values
/// that Go reports as equal give an equal number.
///
/// A slice, a map, and a function holds no identity, thus a key of such a type
/// panics. Go hashes the key of a map before it compares, thus the hash names
/// the cause and the comparison never runs.
///
/// root carries the type of the key through the walk of the fields. Go names
/// that type, and not the field that holds no identity.
int $goHash(Object? v, TypeDesc d, [TypeDesc? root]) {
  final TypeDesc top = root ?? d;
  switch (d.kind) {
    case 'slice' || 'map' || 'func':
      $panicHash(top.display, $hashInWrite);
    case 'array':
      final List<Object?> e = v! as List<Object?>;
      final TypeDesc? elem = d.elem?.call();
      if (elem == null) return e.length;
      final List<int> parts = <int>[];
      for (final Object? x in e) {
        parts.add($goHash(x, elem, top));
      }
      return Object.hashAll(parts);
    case 'struct':
      final List<int> parts = <int>[];
      for (final FieldDesc f in d.fields) {
        parts.add($goHash(f.get(v), f.type.call(), top));
      }
      return Object.hashAll(parts);
    case 'interface':
      // The hash reads the parts of the pair, and not the getter hashCode of
      // Iface. That getter calls this function again, and the type flow
      // analysis of dart2wasm stops on such a cycle through a getter. The 2
      // forms give the same number.
      final Iface? iv = v as Iface?;
      if (iv == null) return 0;
      return Object.hash(iv.desc.name, $goHash(iv.value, iv.desc));
    case 'ptr':
      final String? elem = d.elem?.call().kind;
      if ((elem == 'struct' || elem == 'array') && v is $Ptr<Object?>) v = v.v;
      if (v is $Ptr<Object?>) return v.hashCode;
      return identityHashCode(v);
  }
  return v.hashCode;
}

/// The zero value of a predeclared type. A tear-off of a top level function is
/// a constant expression of Dart, thus each descriptor below stays const.
Object? $zeroInt() => GoInt.zero;
Object? $zeroFloat() => GoFloat64.zero;
Object? $zeroBool() => false;
Object? $zeroString() => GoString.empty;
Object? $zeroComplex() => GoComplex.zero;

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
  zero: $float32Zero,
);
const TypeDesc $typeFloat64 = TypeDesc(
  'float64',
  'float64',
  <String, Function>{},
  zero: $zeroFloat,
);
const TypeDesc $typeComplex64 = TypeDesc(
  'complex64',
  'complex64',
  <String, Function>{},
  zero: $complex32Zero,
);
const TypeDesc $typeComplex128 = TypeDesc(
  'complex128',
  'complex128',
  <String, Function>{},
  zero: $zeroComplex,
);
const TypeDesc $typeString = TypeDesc(
  'string',
  'string',
  <String, Function>{},
  zero: $zeroString,
);

/// The name of the first method of the list that the type does not hold, or an
/// empty text. Go names that method in the report of a failed assertion.
String $missingMethod(Iface? x, List<String> methods) {
  if (x == null) return '';
  for (final m in methods) {
    if (!x.desc.methods.containsKey(m)) return _methodName(m);
  }
  return '';
}

/// The Go name of a method key. A key holds the name, the parameters, and the
/// results, for example `String()string`.
String _methodName(String key) {
  final int i = key.indexOf('(');
  final String name = i < 0 ? key : key.substring(0, i);
  return name.substring(name.lastIndexOf('.') + 1);
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
///
/// static names the interface type of the operand, and the report of Go writes
/// it in front of the value: "interface {} is int, not string".
Object? $assertType(
  Iface? x,
  String name, [
  String static = 'interface',
  String want = '',
  String wantPkg = '',
]) {
  // name holds the identity of the type, and the report needs the text that
  // reflect.Type.String gives. The 2 differ for a type of package main.
  final String target = want.isEmpty ? name : want;
  if (x == null) {
    $panic(
      $typeAssertionError('interface conversion: $static is nil, not $target'),
    );
  }
  if (x.desc.name != name) {
    $panic($typeAssertionError(_assertText(static, x.desc, target, wantPkg)));
  }
  return x.value;
}

String _assertText(String static, TypeDesc got, String want, String wantPkg) {
  final String tail = got.display == want
      ? (got.assertionPkg == wantPkg
            ? ' (types from different scopes)'
            : ' (types from different packages)')
      : '';
  return 'interface conversion: $static is ${got.display}, not $want$tail';
}

/// The value of an assertion to an interface type.
Iface $assertIface(Iface? x, List<String> methods, String name) {
  if (!$implements(x, methods)) {
    if (x == null) {
      $panic(
        $typeAssertionError(
          'interface conversion: interface is nil, not $name',
        ),
      );
    }
    $panic(
      $typeAssertionError(
        'interface conversion: ${x.desc.display} is not $name: '
        'missing method ${$missingMethod(x, methods)}',
      ),
    );
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
  while (true) {
    if (x == null) $panicNil();
    final forward = x.desc.forwardMethods[name];
    if (forward != null) {
      x = forward(x.value);
      continue;
    }
    final Function m = $method(x, name);
    if (identical(m, $ifaceMethod)) {
      if (x.value is Iface) {
        x = x.value as Iface;
        continue;
      }
      $panicNil();
    }
    return $invoke(m, x.value, args);
  }
}

/// Calls a method with a receiver and a list of arguments.
///
/// Function.apply builds a list and reads the signature at run time, thus it
/// costs about 10 times a plain call. A small count of arguments takes the
/// plain form, and each larger count falls back.
Object? $invoke(Function m, Object? recv, List<Object?> args) {
  switch (args.length) {
    case 0:
      return m(recv);
    case 1:
      return m(recv, args[0]);
    case 2:
      return m(recv, args[0], args[1]);
    case 3:
      return m(recv, args[0], args[1], args[2]);
    case 4:
      return m(recv, args[0], args[1], args[2], args[3]);
  }
  return Function.apply(m, <Object?>[recv, ...args]);
}

/// $boxDesc puts a value into an interface value with the given description.
/// A type argument can name an interface type, and a value of such a type
/// parameter already holds an interface value. That value passes through.
Iface? $boxDesc(TypeDesc d, Object? v) {
  if (d.kind == 'interface') return v as Iface?;
  return Iface(d, v);
}

/// $ifaceMethod stands in the method table of the description of an interface
/// type. Such a table holds the key of each method that the interface needs,
/// and no call reaches the value.
Never $ifaceMethod() =>
    $panic($runtimeError('the method belongs to an interface type'));

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
T $assertDict<T>(Iface? x, Dict<T> d, String static) {
  if (d.desc.kind == 'interface') {
    return $assertIface(x, d.desc.methods.keys.toList(), d.desc.display) as T;
  }
  if (!$isDictType(x, d.desc)) {
    $panic(
      $typeAssertionError(
        x == null
            ? 'interface conversion: $static is nil, '
                  'not ${d.desc.display}'
            : _assertText(static, x.desc, d.desc.display, d.desc.assertionPkg),
      ),
    );
  }
  return $dictValue<T>(x, d);
}

import 'dart:convert';

import 'box.dart';
import 'goint.dart';
import 'iface.dart';
import 'jsonparse.dart';
import 'map.dart';
import 'reflect.dart';
import 'slice.dart';
import 'string.dart';

/// The key of the method of the interface json.Marshaler in a method table.
const String _marshalKey = 'MarshalJSON()[]uint8,error';

/// The key of the method of the interface json.Unmarshaler.
const String _unmarshalKey = 'UnmarshalJSON([]uint8)error';

/// The key of the method of the interface encoding.TextMarshaler. Go writes the
/// text of such a value as a JSON string, and MarshalJSON takes precedence.
const String _textMarshalKey = 'MarshalText()[]uint8,error';

/// The key of the method of the interface encoding.TextUnmarshaler.
const String _textUnmarshalKey = 'UnmarshalText([]uint8)error';

/// The full name of the type encoding/json.Number. A field of that type holds
/// the text of a JSON number, thus the walk holds it apart from a string.
const String _numberType = 'encoding/json.Number';

/// Writes the JSON form of a Go value.
///
/// The walk reads the type descriptor of the value. A struct gives an object,
/// and the name of each member comes from the json tag of the field.
///
/// The second result holds the cause of a failure, and it is empty after a
/// success. A cause that starts with a null byte holds the code and the parts
/// of a typed error of the package encoding/json.
(GoString, GoString) $jsonMarshal(Iface? v) {
  try {
    return (GoString.fromString(_text(_tree(v))), GoString.empty);
  } on _JsonCause catch (e) {
    return (GoString.empty, GoString.fromString(e.text));
  }
}

/// _JsonCause names a value that holds no JSON form.
class _JsonCause implements Exception {
  final String text;

  _JsonCause(this.text);
}

/// _coded builds the cause of a typed error of the package encoding/json.
_JsonCause _coded(String code, List<String> parts) =>
    _JsonCause('\u0000$code\u0000${parts.join('\u0000')}');

/// _tree maps a Go value to the tree that jsonEncode accepts.
Object? _tree(Iface? v) {
  if (v == null) return null;
  return _value(v.desc, v.value);
}

Object? _value(TypeDesc desc, Object? value) {
  final Function? marshal = desc.methods[_marshalKey];
  if (marshal != null) return _callMarshal(desc, marshal, value);
  final Function? text = desc.methods[_textMarshalKey];
  if (text != null) return _callTextMarshal(desc, text, value);
  switch (desc.kind) {
    case 'bool':
      return value as bool;
    case 'string':
      if (desc.name == _numberType) return _numberOf(value as GoString);
      return (value as GoString).toDartString();
    case 'float32':
    case 'float64':
      return _number(value as double);
    case 'ptr':
      if (value == null) return null;
      final TypeDesc? elem = desc.elem?.call();
      if (elem == null) {
        throw _JsonCause('json: a pointer holds no element type');
      }
      // The Dart form of a pointer to a struct and of a pointer to an array is
      // the value itself. Each other pointer holds a cell, thus the walk reads
      // the cell out first.
      return _value(elem, value is $Ptr<Object?> ? value.v : value);
    case 'struct':
      return _object(desc, value);
    case 'slice':
      return _slice(desc, value as Slice<Object?>?);
    case 'array':
      return _array(desc, value as List<Object?>);
    case 'map':
      return _map(desc, value as GoMap<Object?, Object?>);
    case 'interface':
      return _tree(value as Iface?);
  }
  if (value is GoInt) return $ix(value);
  throw _coded('unsupportedType', <String>[desc.display]);
}

/// _numberOf writes a value of the type encoding/json.Number. Go writes the
/// text of the number, and not a JSON string.
Object _numberOf(GoString text) {
  if (text.length == 0) return 0;
  final String literal = text.toDartString();
  if (num.tryParse(literal) == null) {
    throw _coded('unsupportedValue', <String>[
      'invalid number literal $literal',
    ]);
  }
  return JsonNum(literal);
}

/// _callMarshal writes a value through the method MarshalJSON of its type.
///
/// Go writes the bytes of the method into the output. This encoder reads the
/// bytes back into a tree, thus the output holds the compact form of them.
Object? _callMarshal(TypeDesc desc, Function marshal, Object? value) {
  if (desc.kind == 'ptr' && value == null) return null;
  final (Slice<GoInt>, Iface?) out =
      Function.apply(marshal, <Object?>[value]) as (Slice<GoInt>, Iface?);
  final Iface? cause = out.$2;
  if (cause != null) {
    throw _coded('marshaler', <String>[desc.display, _errorText(cause)]);
  }
  try {
    return jsonParse($bytesToString(out.$1).toDartString());
  } on FormatException catch (e) {
    throw _coded('marshaler', <String>[desc.display, e.message]);
  }
}

/// _callTextMarshal writes a value through the method MarshalText of its type.
/// Go writes the text as a JSON string.
Object? _callTextMarshal(TypeDesc desc, Function marshal, Object? value) {
  if (desc.kind == 'ptr' && value == null) return null;
  final (Slice<GoInt>, Iface?) out =
      Function.apply(marshal, <Object?>[value]) as (Slice<GoInt>, Iface?);
  final Iface? cause = out.$2;
  if (cause != null) {
    throw _coded('marshaler', <String>[desc.display, _errorText(cause)]);
  }
  return $bytesToString(out.$1).toDartString();
}

/// _text writes a tree in the compact JSON form.
///
/// jsonEncode of dart:convert cannot write a JsonNum, and a literal above the
/// range of a Dart int loses digits through a double. The writer below puts
/// the text of each literal in the output, thus a Number and a RawMessage keep
/// each digit of the input.
String _text(Object? tree) {
  final StringBuffer b = StringBuffer();
  _writeTree(b, tree);
  return b.toString();
}

void _writeTree(StringBuffer b, Object? tree) {
  if (tree is JsonNum) {
    b.write(tree.text);
    return;
  }
  if (tree is List) {
    b.write('[');
    for (int i = 0; i < tree.length; i++) {
      if (i > 0) b.write(',');
      _writeTree(b, tree[i]);
    }
    b.write(']');
    return;
  }
  if (tree is Map) {
    b.write('{');
    bool first = true;
    for (final MapEntry<Object?, Object?> e in tree.entries) {
      if (!first) b.write(',');
      first = false;
      b.write(jsonEncode(e.key as String));
      b.write(':');
      _writeTree(b, e.value);
    }
    b.write('}');
    return;
  }
  b.write(jsonEncode(tree));
}

String _errorText(Iface cause) =>
    ($call(cause, 'Error()string', <Object?>[]) as GoString).toDartString();

/// _slice writes a slice. Go writes base64 text for a byte slice, and it holds
/// the type of the element apart when that type writes a form of its own.
Object? _slice(TypeDesc desc, Slice<Object?>? s) {
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) throw _JsonCause('json: a slice holds no element type');
  if (elem.kind == 'uint8' && !_marshaler(elem)) return _base64(s);
  return _list(elem, s);
}

/// _marshaler reports whether a pointer to the type holds MarshalJSON. Go reads
/// the method set of the pointer, because it holds each method of the type.
bool _marshaler(TypeDesc desc) =>
    _hasMethod(desc, _marshalKey) || _hasMethod(desc, _textMarshalKey);

bool _hasMethod(TypeDesc desc, String key) =>
    desc.methods.containsKey(key) ||
    (desc.ptr?.call().methods.containsKey(key) ?? false);

String? _base64(Slice<Object?>? s) {
  if (s == null || s.isNil) return null;
  final List<int> out = List<int>.filled(s.len, 0);
  for (int i = 0; i < s.len; i++) {
    out[i] = $ix(s.array[s.off + i] as GoInt);
  }
  return base64.encode(out);
}

Map<String, Object?> _object(TypeDesc desc, Object? value) {
  if (value == null) return <String, Object?>{};
  final Map<String, Object?> out = <String, Object?>{};
  for (final FieldDesc f in desc.fields) {
    if (!f.exported) continue;
    final _Tag tag = _readTag(f.tag, f.name);
    if (tag.skip) continue;
    final Object? raw = f.get(value);
    if (tag.omitEmpty && _empty(f.type(), raw)) continue;
    out[tag.name] = _value(f.type(), raw);
  }
  return out;
}

/// _number writes a float in the form of Go. Go writes the shortest text that
/// reads back as the same value, thus a whole value holds no fraction.
Object _number(double v) {
  if (v.isNaN || v.isInfinite) {
    throw _coded('unsupportedValue', <String>['$v']);
  }
  if (v == v.roundToDouble() && v.abs() < 1e21) return v.toInt();
  return v;
}

/// _list writes the elements of a slice. Go writes null for a nil slice.
List<Object?>? _list(TypeDesc elem, Slice<Object?>? s) {
  if (s == null || s.isNil) return null;
  final List<Object?> out = <Object?>[];
  for (int i = 0; i < s.len; i++) {
    out.add(_value(elem, s.array[s.off + i]));
  }
  return out;
}

List<Object?> _array(TypeDesc desc, List<Object?> a) {
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) throw _JsonCause('json: an array holds no element type');
  final List<Object?> out = <Object?>[];
  for (final Object? v in a) {
    out.add(_value(elem, v));
  }
  return out;
}

Map<String, Object?> _map(TypeDesc desc, GoMap<Object?, Object?> m) {
  final TypeDesc? key = desc.key?.call();
  final TypeDesc? elem = desc.elem?.call();
  if (key == null || elem == null) {
    throw _JsonCause('json: a map holds no key type or no element type');
  }
  final Slice<Object?> keys = m.keys();
  final List<String> names = <String>[];
  final Map<String, Object?> byName = <String, Object?>{};
  for (int i = 0; i < keys.len; i++) {
    final Object? k = keys.array[keys.off + i];
    final String text = _keyText(key, k);
    names.add(text);
    byName[text] = _value(elem, m.get(k));
  }
  names.sort();
  final Map<String, Object?> out = <String, Object?>{};
  for (final String n in names) {
    out[n] = byName[n];
  }
  return out;
}

String _keyText(TypeDesc key, Object? k) {
  final Function? text = key.methods[_textMarshalKey];
  if (text != null) {
    final Object? out = _callTextMarshal(key, text, k);
    if (out is String) return out;
  }
  if (key.kind == 'string') return (k as GoString).toDartString();
  if (k is GoInt) return k.toString();
  throw _coded('unsupportedType', <String>['map key ${key.display}']);
}

/// _Tag holds the parts of a json tag of a field.
class _Tag {
  final String name;
  final bool omitEmpty;
  final bool skip;

  _Tag(this.name, this.omitEmpty, this.skip);
}

_Tag _readTag(String tag, String fieldName) {
  final String value = _tagValue(tag, 'json');
  if (value == '-') return _Tag(fieldName, false, true);
  final List<String> parts = value.split(',');
  final String name = parts.isNotEmpty && parts[0].isNotEmpty
      ? parts[0]
      : fieldName;
  bool omit = false;
  for (int i = 1; i < parts.length; i++) {
    if (parts[i] == 'omitempty') omit = true;
  }
  return _Tag(name, omit, false);
}

/// _tagValue reads 1 key of a Go struct tag. The tag holds
/// `key:"value" key2:"value2"`.
String _tagValue(String tag, String want) {
  int i = 0;
  while (i < tag.length) {
    while (i < tag.length && tag[i] == ' ') {
      i++;
    }
    final int colon = tag.indexOf(':', i);
    if (colon < 0) return '';
    final String key = tag.substring(i, colon);
    if (colon + 1 >= tag.length || tag[colon + 1] != '"') return '';
    final int end = tag.indexOf('"', colon + 2);
    if (end < 0) return '';
    if (key == want) return tag.substring(colon + 2, end);
    i = end + 1;
  }
  return '';
}

bool _empty(TypeDesc desc, Object? v) {
  switch (desc.kind) {
    case 'bool':
      return v == false;
    case 'string':
      return (v as GoString).length == 0;
    case 'float32':
    case 'float64':
      return v == 0.0;
    case 'ptr':
    case 'interface':
      return v == null;
    case 'slice':
      final Slice<Object?>? s = v as Slice<Object?>?;
      return s == null || s.len == 0;
    case 'map':
      return (v as GoMap<Object?, Object?>).length == 0;
  }
  return v is GoInt && v == GoInt.zero;
}

/// Reads the JSON form into a Go value.
///
/// The target holds a pointer, and the walk reads the type descriptor of the
/// element. The result holds the cause of a failure, and it is empty after a
/// success.
GoString $jsonUnmarshal(
  Iface? target,
  GoString data,
  bool strict,
  bool useNumber,
) {
  _strict = strict;
  _useNumber = useNumber;
  _structName = '';
  _fieldName = '';
  try {
    if (target == null) {
      throw _coded('invalid', <String>['nil']);
    }
    if (target.desc.kind != 'ptr') {
      throw _coded('invalid', <String>['non-pointer ${target.desc.display}']);
    }
    final TypeDesc? elem = target.desc.elem?.call();
    if (elem == null || target.value == null) {
      throw _coded('invalid', <String>['nil ${target.desc.display}']);
    }
    _store(elem, target.value, jsonParse(data.toDartString()));
    return GoString.empty;
  } on _JsonCause catch (e) {
    return GoString.fromString(e.text);
  } on FormatException catch (e) {
    // The Go side scans the text before this call. A form that only the Dart
    // parser rejects reaches this line.
    return GoString.fromString('json: ${e.message}');
  }
}

/// The state of 1 walk of the decoder. Go names the struct and the field of a
/// value that does not fit, and it reports a member that matches no field.
/// The walk is synchronous, thus 1 set of variables holds that state.
bool _strict = false;
bool _useNumber = false;
String _structName = '';
String _fieldName = '';

/// _store writes a value through a pointer.
///
/// The Dart form of a pointer to a struct is the object of the struct, and the
/// Dart form of a pointer to an array is the list. Each other pointer holds a
/// box.
void _store(TypeDesc elem, Object? pointer, Object? tree) {
  final Function? unmarshal = _unmarshaler(elem);
  if (unmarshal != null) {
    _callUnmarshal(unmarshal, pointer, tree);
    return;
  }
  final Function? text = _textUnmarshaler(elem);
  if (text != null) {
    _callTextUnmarshal(elem, text, pointer, tree);
    return;
  }
  switch (elem.kind) {
    case 'struct':
      _fillStruct(elem, pointer, tree);
      return;
    case 'array':
      _fillArray(elem, pointer as List<Object?>, tree);
      return;
  }
  final $Ptr<dynamic> cell = pointer as $Ptr<dynamic>;
  cell.v = _into(elem, cell.v, tree);
}

/// _unmarshaler gives the method UnmarshalJSON of the pointer to a type.
///
/// A method with a value receiver cannot write the value, thus this walk reads
/// only the method set of the pointer. docs/parity.md holds that gap.
Function? _unmarshaler(TypeDesc desc) {
  if (desc.kind == 'ptr') return null;
  return desc.ptr?.call().methods[_unmarshalKey];
}

/// _textUnmarshaler gives the method UnmarshalText of the pointer to a type.
Function? _textUnmarshaler(TypeDesc desc) {
  if (desc.kind == 'ptr') return null;
  return desc.ptr?.call().methods[_textUnmarshalKey];
}

/// _callTextUnmarshal reads a JSON string through the method UnmarshalText.
///
/// Go gives the method the text of the string, and it rejects each other JSON
/// value. A null leaves the target as it stands.
void _callTextUnmarshal(
  TypeDesc desc,
  Function unmarshal,
  Object? pointer,
  Object? tree,
) {
  if (tree == null) return;
  if (tree is! String) throw _wrong(tree, desc);
  final Iface? cause = Function.apply(unmarshal, <Object?>[
    pointer,
    $stringToBytes(GoString.fromString(tree)),
  ]) as Iface?;
  if (cause != null) throw _JsonCause(_errorText(cause));
}

/// _callUnmarshal reads a tree through the method UnmarshalJSON of a type.
///
/// Go gives the method the bytes of the input. This decoder reads the whole
/// input first, thus the method reads the compact form of them.
void _callUnmarshal(Function unmarshal, Object? pointer, Object? tree) {
  final Iface? cause = Function.apply(unmarshal, <Object?>[
    pointer,
    $stringToBytes(GoString.fromString(_text(tree))),
  ]) as Iface?;
  if (cause != null) throw _JsonCause(_errorText(cause));
}

/// _into reads a tree into a value that already stands.
///
/// Go descends into a non-nil pointer that a non-nil interface holds, and it
/// writes the value that the pointer names. net/rpc/jsonrpc reads its
/// parameter that way. A null keeps the rule of Go too: the interface takes
/// the value nil, and the pointer stays untouched.
Object? _into(TypeDesc desc, Object? current, Object? tree) {
  if (desc.kind != 'interface' || tree == null || current is! Iface) {
    return _decode(desc, tree);
  }
  if (current.desc.kind != 'ptr' || current.value == null) {
    return _decode(desc, tree);
  }
  final TypeDesc? elem = current.desc.elem?.call();
  if (elem == null) return _decode(desc, tree);
  _store(elem, current.value, tree);
  return current;
}

Object? _decode(TypeDesc desc, Object? tree) {
  final Function? unmarshal = _unmarshaler(desc);
  if (unmarshal != null) {
    final Object? pointer = desc.ptr!.call().newPtr!();
    _callUnmarshal(unmarshal, pointer, tree);
    return pointer is $Ptr<Object?> ? pointer.v : pointer;
  }
  final Function? text = _textUnmarshaler(desc);
  if (text != null) {
    final Object? pointer = desc.ptr!.call().newPtr!();
    _callTextUnmarshal(desc, text, pointer, tree);
    return pointer is $Ptr<Object?> ? pointer.v : pointer;
  }
  switch (desc.kind) {
    case 'interface':
      return _any(tree);
    case 'bool':
      if (tree is! bool) throw _wrong(tree, desc);
      return tree;
    case 'string':
      if (desc.name == _numberType) {
        if (tree is! JsonNum) throw _wrong(tree, desc);
        return GoString.fromString(tree.text);
      }
      if (tree is! String) throw _wrong(tree, desc);
      return GoString.fromString(tree);
    case 'float32':
    case 'float64':
      if (tree is! JsonNum) throw _wrong(tree, desc);
      return tree.asDouble;
    case 'ptr':
      return _decodePointer(desc, tree);
    case 'struct':
      final Object? out = _fresh(desc);
      _fillStruct(desc, out, tree);
      return out;
    case 'slice':
      return _decodeSlice(desc, tree);
    case 'array':
      final Object? out = _fresh(desc);
      _fillArray(desc, out as List<Object?>, tree);
      return out;
    case 'map':
      return _decodeMap(desc, tree);
  }
  if (tree is JsonNum) return _integer(tree, desc);
  throw _wrong(tree, desc);
}

Object? _decodePointer(TypeDesc desc, Object? tree) {
  if (tree == null) return null;
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) throw _JsonCause('json: a pointer holds no element type');
  if (elem.kind == 'struct' || elem.kind == 'array') {
    final Object? out = _fresh(elem);
    _store(elem, out, tree);
    return out;
  }
  // The cell must hold the Dart type of the element, because the setter of a
  // field of a pointer type casts to that exact type. Read $typedBox.
  return $typedBox(elem, _decode(elem, tree));
}

Object? _decodeSlice(TypeDesc desc, Object? tree) {
  if (tree == null) return _fresh(desc);
  final TypeDesc? elem = desc.elem?.call();
  final Object? Function(int)? make = desc.make;
  if (elem == null || make == null) {
    throw _JsonCause('json: a slice holds no element type');
  }
  if (tree is String) {
    if (elem.kind != 'uint8') throw _wrong(tree, desc);
    return _decodeBase64(make, tree);
  }
  if (tree is! List) throw _wrong(tree, desc);
  final Slice<Object?> out = make(tree.length) as Slice<Object?>;
  for (int i = 0; i < tree.length; i++) {
    out.array[out.off + i] = _decode(elem, tree[i]);
  }
  return out;
}

/// _decodeBase64 reads the base64 text of a byte slice.
Object? _decodeBase64(Object? Function(int) make, String text) {
  final List<int> bytes;
  try {
    bytes = base64.decode(text);
  } on FormatException {
    throw _JsonCause('json: illegal base64 data');
  }
  final Slice<Object?> out = make(bytes.length) as Slice<Object?>;
  for (int i = 0; i < bytes.length; i++) {
    out.array[out.off + i] = $gi(bytes[i]);
  }
  return out;
}

Object? _decodeMap(TypeDesc desc, Object? tree) {
  if (tree == null) return _fresh(desc);
  if (tree is! Map) throw _wrong(tree, desc);
  final TypeDesc? key = desc.key?.call();
  final TypeDesc? elem = desc.elem?.call();
  final Object? Function(int)? make = desc.make;
  if (key == null || elem == null || make == null) {
    throw _JsonCause('json: a map holds no key type or no element type');
  }
  final GoMap<Object?, Object?> out = make(0) as GoMap<Object?, Object?>;
  for (final MapEntry<Object?, Object?> entry in tree.entries) {
    out.set(_key(key, entry.key as String), _decode(elem, entry.value));
  }
  return out;
}

Object? _key(TypeDesc desc, String text) {
  final Function? unmarshal = _textUnmarshaler(desc);
  if (unmarshal != null) {
    final Object? pointer = desc.ptr!.call().newPtr!();
    _callTextUnmarshal(desc, unmarshal, pointer, text);
    return pointer is $Ptr<Object?> ? pointer.v : pointer;
  }
  if (desc.kind == 'string') return GoString.fromString(text);
  final int? n = int.tryParse(text);
  if (n == null) {
    throw _JsonCause('json: unsupported map key type: ${desc.display}');
  }
  return n;
}

void _fillStruct(TypeDesc desc, Object? target, Object? tree) {
  if (tree == null) return;
  if (tree is! Map) throw _wrong(tree, desc);
  final String outerStruct = _structName;
  final String outerField = _fieldName;
  for (final MapEntry<Object?, Object?> entry in tree.entries) {
    final String name = entry.key as String;
    final FieldDesc? field = _field(desc, name);
    if (field == null) {
      if (_strict) throw _coded('unknown', <String>[_path(outerField, name)]);
      continue;
    }
    _structName = _bareName(desc.display);
    _fieldName = _path(outerField, _readTag(field.tag, field.name).name);
    final TypeDesc ft = field.type();
    // The read of the field costs a call, thus only an interface field takes
    // it. _into descends into a pointer that such a field already holds.
    final Object? current = ft.kind == 'interface' ? field.get(target) : null;
    field.set(target, _into(ft, current, entry.value));
  }
  _structName = outerStruct;
  _fieldName = outerField;
}

/// _path names a field under the field that holds it.
String _path(String outer, String name) =>
    outer.isEmpty ? name : '$outer.$name';

/// _bareName drops the package of a type name. Go names the struct of an
/// UnmarshalTypeError without its package.
String _bareName(String display) {
  final int bracket = display.indexOf('[');
  final String head = bracket < 0 ? display : display.substring(0, bracket);
  final int dot = head.lastIndexOf('.');
  final String name = dot < 0 ? head : head.substring(dot + 1);
  return bracket < 0 ? name : name + display.substring(bracket);
}

/// _field finds the field of a name. Go takes an equal name first, and a name
/// that differs only in case after it.
FieldDesc? _field(TypeDesc desc, String name) {
  FieldDesc? loose;
  for (final FieldDesc f in desc.fields) {
    if (!f.exported) continue;
    final _Tag tag = _readTag(f.tag, f.name);
    if (tag.skip) continue;
    if (tag.name == name) return f;
    if (loose == null && tag.name.toLowerCase() == name.toLowerCase()) {
      loose = f;
    }
  }
  return loose;
}

void _fillArray(TypeDesc desc, List<Object?> target, Object? tree) {
  if (tree == null) return;
  if (tree is! List) throw _wrong(tree, desc);
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) throw _JsonCause('json: an array holds no element type');
  for (int i = 0; i < target.length; i++) {
    target[i] = i < tree.length
        ? _into(elem, target[i], tree[i])
        : elem.zero?.call();
  }
}

Object? _fresh(TypeDesc desc) {
  final Object? Function()? zero = desc.zero;
  if (zero == null) {
    throw _JsonCause('json: the type ${desc.display} builds no value');
  }
  return zero();
}

/// _any reads a value into an empty interface. Go gives a number the type
/// float64, and an object the type map[string]interface{}.
Object? _any(Object? tree) {
  if (tree == null) return null;
  if (tree is bool) return Iface($typeBool, tree);
  if (tree is JsonNum) {
    if (_useNumber) {
      return Iface(_numberDesc, GoString.fromString(tree.text));
    }
    return Iface($typeFloat64, tree.asDouble);
  }
  if (tree is String) return Iface($typeString, GoString.fromString(tree));
  if (tree is List) {
    final List<Iface?> out = <Iface?>[];
    for (final Object? v in tree) {
      out.add(_any(v) as Iface?);
    }
    return Iface(_anySliceDesc, Slice<Iface?>.of(out));
  }
  if (tree is Map) {
    final GoMap<GoString, Iface?> out = GoMap<GoString, Iface?>(() => null);
    for (final MapEntry<Object?, Object?> entry in tree.entries) {
      out.set(
        GoString.fromString(entry.key as String),
        _any(entry.value) as Iface?,
      );
    }
    return Iface(_anyMapDesc, out);
  }
  throw _JsonCause('json: unsupported value');
}

// The first argument of TypeDesc is the identity of the type, and the compiler
// writes it with the text of `go/types`. That text holds no space inside
// `interface{}`. reflect.Type.String of Go writes a space, thus the display
// text differs from the identity.
const TypeDesc _anyDesc = TypeDesc(
  'interface{}',
  'interface',
  <String, Function>{},
  text: 'interface {}',
);

final TypeDesc _anySliceDesc = TypeDesc(
  '[]interface{}',
  'slice',
  const <String, Function>{},
  text: '[]interface {}',
  elem: () => _anyDesc,
);

final TypeDesc _anyMapDesc = TypeDesc(
  'map[string]interface{}',
  'map',
  const <String, Function>{},
  text: 'map[string]interface {}',
  key: () => $typeString,
  elem: () => _anyDesc,
);

_JsonCause _wrong(Object? tree, TypeDesc desc) => _coded('type', <String>[
  _treeKind(tree),
  desc.display,
  _structName,
  _fieldName,
]);

String _treeKind(Object? tree) {
  if (tree == null) return 'null';
  if (tree is bool) return 'bool';
  if (tree is JsonNum) return 'number';
  if (tree is String) return 'string';
  if (tree is List) return 'array';
  return 'object';
}

/// _integer reads a number literal into an integer field.
///
/// A uint64 above the largest int64 does not fit a Dart int as an unsigned
/// value, thus the text goes through BigInt and lands as the 2 complement
/// form. Go holds each unsigned integer in that form as well.
Object _integer(JsonNum tree, TypeDesc desc) {
  final String text = tree.text;
  final int? plain = int.tryParse(text);
  if (plain != null) return plain;
  if (!text.contains('.') && !text.contains('e') && !text.contains('E')) {
    final BigInt? big = BigInt.tryParse(text);
    if (big != null) return big.toSigned(64).toInt();
  }
  final double v = tree.asDouble;
  if (v != v.roundToDouble()) throw _wrong(tree, desc);
  return v.toInt();
}

/// The type descriptor of encoding/json.Number. UseNumber writes it into an
/// empty interface.
const TypeDesc _numberDesc = TypeDesc(
  _numberType,
  'string',
  <String, Function>{},
  text: 'json.Number',
);

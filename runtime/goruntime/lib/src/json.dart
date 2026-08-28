import 'dart:convert';

import 'box.dart';
import 'iface.dart';
import 'map.dart';
import 'slice.dart';
import 'string.dart';

/// Writes the JSON form of a Go value.
///
/// The walk reads the type descriptor of the value. A struct gives an object,
/// and the name of each member comes from the json tag of the field.
///
/// The second result holds the cause of a failure, and it is empty after a
/// success.
(GoString, GoString) $jsonMarshal(Iface? v, GoString indent) {
  try {
    final Object? tree = _tree(v);
    final String text = indent.length == 0
        ? jsonEncode(tree)
        : JsonEncoder.withIndent(indent.toDartString()).convert(tree);
    return (GoString.fromString(text), GoString.empty);
  } on _JsonCause catch (e) {
    return (GoString.empty, GoString.fromString(e.text));
  }
}

/// _JsonCause names a value that holds no JSON form.
class _JsonCause implements Exception {
  final String text;

  _JsonCause(this.text);
}

/// _tree maps a Go value to the tree that jsonEncode accepts.
///
/// Go writes a byte slice as base64 text. This encoder writes a list of numbers
/// instead, thus a byte slice does not match Go.
Object? _tree(Iface? v) {
  if (v == null) return null;
  return _value(v.desc, v.value);
}

Object? _value(TypeDesc desc, Object? value) {
  switch (desc.kind) {
    case 'bool':
      return value as bool;
    case 'string':
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
      return _value(elem, value);
    case 'struct':
      return _object(desc, value);
    case 'slice':
      return _list(desc, value as Slice<Object?>?);
    case 'array':
      return _array(desc, value as List<Object?>);
    case 'map':
      return _map(desc, value as GoMap<Object?, Object?>);
    case 'interface':
      return _tree(value as Iface?);
  }
  if (value is int) return value;
  throw _JsonCause('json: unsupported type: ${desc.display}');
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
    throw _JsonCause('json: unsupported value: $v');
  }
  if (v == v.roundToDouble() && v.abs() < 1e21) return v.toInt();
  return v;
}

/// _list writes a slice. Go writes null for a nil slice.
List<Object?>? _list(TypeDesc desc, Slice<Object?>? s) {
  final TypeDesc? elem = desc.elem?.call();
  if (elem == null) throw _JsonCause('json: a slice holds no element type');
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
  if (key.kind == 'string') return (k as GoString).toDartString();
  if (k is int) return k.toString();
  throw _JsonCause('json: unsupported map key type: ${key.display}');
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
  return v is int && v == 0;
}

/// Reads the JSON form into a Go value.
///
/// The target holds a pointer, and the walk reads the type descriptor of the
/// element. The result holds the cause of a failure, and it is empty after a
/// success.
GoString $jsonUnmarshal(Iface? target, GoString data) {
  try {
    if (target == null) {
      throw _JsonCause('json: Unmarshal(nil)');
    }
    if (target.desc.kind != 'ptr') {
      throw _JsonCause('json: Unmarshal(non-pointer ${target.desc.display})');
    }
    final TypeDesc? elem = target.desc.elem?.call();
    if (elem == null || target.value == null) {
      throw _JsonCause('json: Unmarshal(nil ${target.desc.display})');
    }
    _store(elem, target.value, jsonDecode(data.toDartString()));
    return GoString.empty;
  } on _JsonCause catch (e) {
    return GoString.fromString(e.text);
  } on FormatException catch (e) {
    return GoString.fromString('json: ${e.message}');
  }
}

/// _store writes a value through a pointer.
///
/// The Dart form of a pointer to a struct is the object of the struct, and the
/// Dart form of a pointer to an array is the list. Each other pointer holds a
/// box.
void _store(TypeDesc elem, Object? pointer, Object? tree) {
  switch (elem.kind) {
    case 'struct':
      _fillStruct(elem, pointer, tree);
      return;
    case 'array':
      _fillArray(elem, pointer as List<Object?>, tree);
      return;
  }
  (pointer as Box<dynamic>).v = _decode(elem, tree);
}

Object? _decode(TypeDesc desc, Object? tree) {
  switch (desc.kind) {
    case 'interface':
      return _any(tree);
    case 'bool':
      if (tree is! bool) throw _wrong(tree, desc);
      return tree;
    case 'string':
      if (tree is! String) throw _wrong(tree, desc);
      return GoString.fromString(tree);
    case 'float32':
    case 'float64':
      if (tree is! num) throw _wrong(tree, desc);
      return tree.toDouble();
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
  if (tree is num && tree == tree.roundToDouble()) return tree.toInt();
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
  return Box<Object?>(_decode(elem, tree));
}

Object? _decodeSlice(TypeDesc desc, Object? tree) {
  if (tree == null) return _fresh(desc);
  if (tree is! List) throw _wrong(tree, desc);
  final TypeDesc? elem = desc.elem?.call();
  final Object? Function(int)? make = desc.make;
  if (elem == null || make == null) {
    throw _JsonCause('json: a slice holds no element type');
  }
  final Slice<Object?> out = make(tree.length) as Slice<Object?>;
  for (int i = 0; i < tree.length; i++) {
    out.array[out.off + i] = _decode(elem, tree[i]);
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
  for (final MapEntry<Object?, Object?> entry in tree.entries) {
    final FieldDesc? field = _field(desc, entry.key as String);
    if (field == null) continue;
    field.set(target, _decode(field.type(), entry.value));
  }
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
    target[i] = i < tree.length ? _decode(elem, tree[i]) : elem.zero?.call();
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
  if (tree is num) return Iface($typeFloat64, tree.toDouble());
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

const TypeDesc _anyDesc = TypeDesc(
  'interface {}',
  'interface',
  <String, Function>{},
  text: 'interface {}',
);

final TypeDesc _anySliceDesc = TypeDesc(
  '[]interface {}',
  'slice',
  const <String, Function>{},
  text: '[]interface {}',
  elem: () => _anyDesc,
);

final TypeDesc _anyMapDesc = TypeDesc(
  'map[string]interface {}',
  'map',
  const <String, Function>{},
  text: 'map[string]interface {}',
  key: () => $typeString,
  elem: () => _anyDesc,
);

_JsonCause _wrong(Object? tree, TypeDesc desc) => _JsonCause(
  'json: cannot unmarshal ${_treeKind(tree)} into Go value of type ${desc.display}',
);

String _treeKind(Object? tree) {
  if (tree == null) return 'null';
  if (tree is bool) return 'bool';
  if (tree is num) return 'number';
  if (tree is String) return 'string';
  if (tree is List) return 'array';
  return 'object';
}

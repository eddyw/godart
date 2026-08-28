import 'dart:convert';

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

import 'package:goruntime/goruntime.dart';

/// GoNode reads 1 node of the Go tree through its type descriptor.
///
/// The host reads no generated class. It reads the name of the descriptor and
/// the field table that the descriptor carries. Thus 1 host serves each node
/// set, and the generated set of milestone 3 needs no change here.
final class GoNode {
  final Iface value;
  final Map<String, FieldDesc> _fields;

  GoNode(this.value) : _fields = _index(value.desc);

  /// kind gives the full Go name of the type, for example `gart.dev/ui.Text`.
  String get kind => value.desc.name;

  static final Map<String, Map<String, FieldDesc>> _cache =
      <String, Map<String, FieldDesc>>{};

  static Map<String, FieldDesc> _index(TypeDesc desc) {
    final Map<String, FieldDesc>? held = _cache[desc.name];
    if (held != null) return held;
    final Map<String, FieldDesc> out = <String, FieldDesc>{};
    for (final FieldDesc f in desc.fields) {
      out[f.name] = f;
    }
    _cache[desc.name] = out;
    return out;
  }

  Object? raw(String name) {
    final FieldDesc? f = _fields[name];
    if (f == null) return null;
    return f.get(value.value);
  }

  /// text gives a string field.
  String text(String name) {
    final Object? v = raw(name);
    return v is GoString ? v.toDartString() : '';
  }

  /// size gives a float64 field.
  double size(String name) {
    final Object? v = raw(name);
    return v is double ? v : 0;
  }

  /// count gives an integer field.
  int count(String name) {
    final Object? v = raw(name);
    return v is int ? v : 0;
  }

  /// flag gives a bool field.
  bool flag(String name) => raw(name) == true;

  /// argb gives a color field. A value of 0 means no color.
  int? argb(String name) {
    final int v = count(name);
    return v == 0 ? null : v;
  }

  /// child gives a Node field.
  Iface? child(String name) {
    final Object? v = raw(name);
    return v is Iface ? v : null;
  }

  /// children gives a slice of Node.
  List<Iface?> children(String name) {
    final Object? v = raw(name);
    if (v is! Slice<Iface?>) return const <Iface?>[];
    return <Iface?>[for (int i = 0; i < v.len; i++) v[i]];
  }

  /// action gives a `func()` field.
  void Function()? action(String name) {
    final Object? v = raw(name);
    return v is void Function() ? v : null;
  }

  /// onText gives a `func(string)` field.
  void Function(GoString)? onText(String name) {
    final Object? v = raw(name);
    return v is void Function(GoString) ? v : null;
  }

  /// onFlag gives a `func(bool)` field.
  void Function(bool)? onFlag(String name) {
    final Object? v = raw(name);
    return v is void Function(bool) ? v : null;
  }

  /// item gives a `func(int) Node` field. GoInt erases to int on each target
  /// that Flutter builds, thus the runtime type names int.
  Iface? Function(int)? item(String name) {
    final Object? v = raw(name);
    return v is Iface? Function(int) ? v : null;
  }
}

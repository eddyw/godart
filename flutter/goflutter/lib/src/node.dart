import 'package:goruntime/goruntime.dart';

/// GoNode reads 1 node of the Go tree through its type descriptor.
///
/// The host reads no generated class. It reads the name of the descriptor and
/// the field table that the descriptor carries. Thus 1 host serves each node
/// set, and the generated set of milestone 3 needs no change here.
final class GoNode {
  final Object? value;
  final TypeDesc desc;
  final Map<String, FieldDesc> _fields;

  GoNode(Iface node)
      : value = node.value,
        desc = node.desc,
        _fields = _index(node.desc);

  /// GoNode.struct reads a struct that stands inside a node, for example the
  /// Insets of a padding field.
  GoNode.struct(this.value, this.desc) : _fields = _index(desc);

  /// kind gives the full Go name of the type, for example `gart.dev/ui.Text`.
  String get kind => desc.name;

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
    return f.get(value);
  }

  /// point reads a pointer field. A nil pointer means "the default of
  /// Flutter", thus the host leaves the Dart parameter out.
  T? point<T>(String name) {
    final Object? v = raw(name);
    return v is $Ptr<T> ? v.v : null;
  }

  /// nested reads a struct field, or a pointer to a struct. It gives null when
  /// the pointer is nil.
  GoNode? nested(String name) {
    final FieldDesc? f = _fields[name];
    if (f == null) return null;
    final Object? v = f.get(value);
    if (v == null) return null;
    final TypeDesc t = f.type();
    if (t.kind == 'ptr') {
      final TypeDesc Function()? elem = t.elem;
      if (elem == null) return null;
      return GoNode.struct(v, elem());
    }
    return GoNode.struct(v, t);
  }

  /// textOpt gives an optional string field.
  String? textOpt(String name) => point<GoString>(name)?.toDartString();

  /// sizeOpt gives an optional float64 field.
  double? sizeOpt(String name) => point<double>(name);

  /// countOpt gives an optional integer field.
  int? countOpt(String name) => point<int>(name);

  /// flagOpt gives an optional bool field.
  bool? flagOpt(String name) => point<bool>(name);

  /// texts gives a slice of string.
  List<String> texts(String name) {
    final Object? v = raw(name);
    if (v is! Slice<GoString>) return const <String>[];
    return <String>[for (int i = 0; i < v.len; i++) v[i].toDartString()];
  }

  /// onSize gives a `func(float64)` field.
  void Function(double)? onSize(String name) {
    final Object? v = raw(name);
    return v is void Function(double) ? v : null;
  }

  /// onCount gives a `func(int)` field.
  void Function(int)? onCount(String name) {
    final Object? v = raw(name);
    return v is void Function(int) ? v : null;
  }

  /// build0 gives a `func() Node` field.
  Iface? Function()? build0(String name) {
    final Object? v = raw(name);
    return v is Iface? Function() ? v : null;
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

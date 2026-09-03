import 'package:goruntime/goruntime.dart';

/// GoNode reads 1 node of the Go tree through its type descriptor.
///
/// The host reads no generated class. It reads the name of the descriptor and
/// the field table that the descriptor carries. Thus 1 host serves each node
/// set, and the generated set of milestone 3 needs no change here.
///
/// Each reader takes the name of the field and the position of it in the Go
/// struct. The position is the fast path: the read then costs 1 index of a
/// list and 1 compare of 2 pointers. The name stays in the call, and it holds
/// the meaning for a reader of the generated host. It also guards the
/// position, thus a Go struct of another order still reads right through the
/// scan of [_find]. A caller that names no position takes that scan too.
///
/// The measurement of `flutter/bench` of 2026-09-03 drove the shape. The map
/// of the name took 7 nanoseconds of the 13 that 1 optional field cost, and
/// the whole tree paid 1 map lookup for each node on top of that.
final class GoNode {
  final Object? value;
  final TypeDesc desc;
  final List<FieldDesc> _fields;

  GoNode(Iface node)
      : value = node.value,
        desc = node.desc,
        _fields = node.desc.fields;

  /// GoNode.struct reads a struct that stands inside a node, for example the
  /// Insets of a padding field.
  GoNode.struct(this.value, this.desc) : _fields = desc.fields;

  /// kind gives the full Go name of the type, for example `gart.dev/ui.Text`.
  String get kind => desc.name;

  /// noAt stands for "the caller names no position". No field list reaches
  /// this length, thus the test of the fast path fails and the scan runs.
  static const int noAt = 0x7fffffff;

  /// _find gives the descriptor of 1 field by its name alone.
  ///
  /// The name of the descriptor and the name of the call site are both string
  /// literals of 1 program, thus the compare of the 2 pointers answers. The
  /// second pass holds a name that another path built.
  FieldDesc? _find(String name) {
    final List<FieldDesc> f = _fields;
    for (int i = 0; i < f.length; i++) {
      if (identical(f[i].name, name)) return f[i];
    }
    for (int i = 0; i < f.length; i++) {
      if (f[i].name == name) return f[i];
    }
    return null;
  }

  /// _field gives the descriptor of 1 field. The readers that need the type of
  /// the field read it, and [raw] holds the same 2 lines of the fast path.
  FieldDesc? _field(String name, int at) {
    final List<FieldDesc> f = _fields;
    if (at < f.length) {
      final FieldDesc d = f[at];
      if (identical(d.name, name)) return d;
    }
    return _find(name);
  }

  /// raw gives the value of 1 field, and null when the struct holds no field
  /// of that name.
  Object? raw(String name, [int at = noAt]) {
    final List<FieldDesc> f = _fields;
    if (at < f.length) {
      final FieldDesc d = f[at];
      if (identical(d.name, name)) return d.get(value);
    }
    return _find(name)?.get(value);
  }

  /// point reads a pointer field. A nil pointer means "the default of
  /// Flutter", thus the host leaves the Dart parameter out.
  ///
  /// The 4 readers below name their own element type. A test against
  /// `$Ptr<T>` of a type parameter costs more than a test against a written
  /// type, and each read of the host runs 1 of the 4.
  T? point<T>(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<T> ? v.v : null;
  }

  /// nested reads a struct field, or a pointer to a struct. It gives null when
  /// the pointer is nil.
  GoNode? nested(String name, [int at = noAt]) {
    final FieldDesc? f = _field(name, at);
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
  String? textOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<GoString> ? v.v.toDartString() : null;
  }

  /// sizeOpt gives an optional float64 field.
  double? sizeOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<double> ? v.v : null;
  }

  /// countOpt gives an optional integer field.
  int? countOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<int> ? v.v : null;
  }

  /// flagOpt gives an optional bool field.
  bool? flagOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<bool> ? v.v : null;
  }

  /// structs reads a slice of structs, for example the shadows of a
  /// decoration.
  List<GoNode> structs(String name, [int at = noAt]) {
    final FieldDesc? f = _field(name, at);
    if (f == null) return const <GoNode>[];
    final Object? v = f.get(value);
    if (v is! Slice<Object?>) return const <GoNode>[];
    final TypeDesc Function()? elem = f.type().elem;
    if (elem == null) return const <GoNode>[];
    final TypeDesc t = elem();
    return <GoNode>[for (int i = 0; i < v.len; i++) GoNode.struct(v[i], t)];
  }

  /// counts gives a slice of integers. A slice of byte reads it too.
  List<int> counts(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    if (v is! Slice<Object?>) return const <int>[];
    final List<int> out = <int>[];
    for (int i = 0; i < v.len; i++) {
      final Object? e = v[i];
      if (e is int) out.add(e);
    }
    return out;
  }

  /// sizes gives a slice of float64.
  List<double> sizes(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    if (v is! Slice<Object?>) return const <double>[];
    final List<double> out = <double>[];
    for (int i = 0; i < v.len; i++) {
      final Object? e = v[i];
      if (e is double) out.add(e);
    }
    return out;
  }

  /// texts gives a slice of string.
  List<String> texts(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    if (v is! Slice<GoString>) return const <String>[];
    return <String>[for (int i = 0; i < v.len; i++) v[i].toDartString()];
  }

  /// onSize gives a `func(float64)` field.
  void Function(double)? onSize(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function(double) ? v : null;
  }

  /// onCount gives a `func(int)` field.
  void Function(int)? onCount(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function(int) ? v : null;
  }

  /// build0 gives a `func() Node` field.
  Iface? Function()? build0(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is Iface? Function() ? v : null;
  }

  /// text gives a string field.
  String text(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is GoString ? v.toDartString() : '';
  }

  /// size gives a float64 field.
  double size(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is double ? v : 0;
  }

  /// count gives an integer field.
  int count(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is int ? v : 0;
  }

  /// flag gives a bool field.
  bool flag(String name, [int at = noAt]) => raw(name, at) == true;

  /// argb gives a color field. A value of 0 means no color.
  int? argb(String name, [int at = noAt]) {
    final int v = count(name, at);
    return v == 0 ? null : v;
  }

  /// child gives a Node field.
  Iface? child(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is Iface ? v : null;
  }

  /// children gives a slice of Node.
  List<Iface?> children(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    if (v is! Slice<Iface?>) return const <Iface?>[];
    return <Iface?>[for (int i = 0; i < v.len; i++) v[i]];
  }

  /// action gives a `func()` field.
  void Function()? action(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function() ? v : null;
  }

  /// onText gives a `func(string)` field.
  void Function(GoString)? onText(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function(GoString) ? v : null;
  }

  /// onTexts gives a `func([]string)` field.
  void Function(Slice<GoString>)? onTexts(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function(Slice<GoString>) ? v : null;
  }

  /// onFlag gives a `func(bool)` field.
  void Function(bool)? onFlag(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function(bool) ? v : null;
  }

  /// item gives a `func(int) Node` field. GoInt erases to int on each target
  /// that Flutter builds, thus the runtime type names int.
  Iface? Function(int)? item(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is Iface? Function(int) ? v : null;
  }

  /// flags gives a slice of bool.
  List<bool> flags(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    if (v is! Slice<Object?>) return const <bool>[];
    return <bool>[for (int i = 0; i < v.len; i++) v[i] == true];
  }

  /// onPair gives a `func(float64, float64)` field.
  void Function(double, double)? onPair(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function(double, double) ? v : null;
  }

  /// onSort gives a `func(int, bool)` field.
  void Function(int, bool)? onSort(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function(int, bool) ? v : null;
  }

  /// task gives a `func(func())` field. The Go side calls the argument when
  /// the work ends, thus the host can hold a Future. Read the Task type of
  /// flutter/ui/node.go.
  void Function(void Function())? task(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function(void Function()) ? v : null;
  }

  /// ask gives a `func(func(bool))` field.
  void Function(void Function(bool))? ask(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is void Function(void Function(bool)) ? v : null;
  }

  /// buildText gives a `func(string) Node` field.
  Iface? Function(GoString)? buildText(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is Iface? Function(GoString) ? v : null;
  }

  /// buildBox gives a `func(float64, float64, float64, float64) Node` field.
  Iface? Function(double, double, double, double)? buildBox(
    String name, [
    int at = noAt,
  ]) {
    final Object? v = raw(name, at);
    return v is Iface? Function(double, double, double, double) ? v : null;
  }

  /// buildSlivers gives a `func(bool) []Node` field.
  Slice<Iface?> Function(bool)? buildSlivers(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is Slice<Iface?> Function(bool) ? v : null;
  }
}

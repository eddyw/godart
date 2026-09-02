import 'package:goruntime/goruntime.dart';

/// This file mirrors the read side of the real host. GoNode of
/// `flutter/goflutter/lib/src/node.dart` and the generated host of an
/// application hold the same field reads, and this copy builds no Flutter
/// widget, because the widget cost stands outside the boundary question.
///
/// A copy stands here for 2 reasons. The measurement must run under
/// `dart compile exe`, and `package:flutter` needs the Flutter engine. The
/// widget construction is also the same work that a Dart application does, thus
/// it hides the price of the boundary.

/// sink holds the result of each walk. A write to a top level variable keeps
/// the Dart compiler away from a drop of the work.
int sink = 0;

final Map<String, Map<String, FieldDesc>> _cache = <String, Map<String, FieldDesc>>{};

Map<String, FieldDesc> _index(TypeDesc desc) {
  final Map<String, FieldDesc>? held = _cache[desc.name];
  if (held != null) return held;
  final Map<String, FieldDesc> out = <String, FieldDesc>{};
  for (final FieldDesc f in desc.fields) {
    out[f.name] = f;
  }
  _cache[desc.name] = out;
  return out;
}

/// GoNodeView reads 1 node of the Go tree through its type descriptor. The
/// method set holds only the readers that this measurement needs.
final class GoNodeView {
  final Object? value;
  final TypeDesc desc;
  final Map<String, FieldDesc> _fields;

  GoNodeView(Iface node)
      : value = node.value,
        desc = node.desc,
        _fields = _index(node.desc);

  String get kind => desc.name;

  Object? raw(String name) {
    final FieldDesc? f = _fields[name];
    if (f == null) return null;
    return f.get(value);
  }

  T? point<T>(String name) {
    final Object? v = raw(name);
    return v is $Ptr<T> ? v.v : null;
  }

  String text(String name) {
    final Object? v = raw(name);
    return v is GoString ? v.toDartString() : '';
  }

  /// bytes gives the byte count of a string field, and it runs no decode. The
  /// difference against text is the price of GoString.toDartString.
  int bytes(String name) {
    final Object? v = raw(name);
    return v is GoString ? v.length : 0;
  }

  String? textOpt(String name) => point<GoString>(name)?.toDartString();

  /// goText gives a string field with no decode. The diff reads it, because a
  /// compare of 2 GoString values reads the bytes and allocates nothing.
  GoString? goText(String name) {
    final Object? v = raw(name);
    return v is GoString ? v : null;
  }

  /// goTextOpt gives an optional string field with no decode.
  GoString? goTextOpt(String name) => point<GoString>(name);

  /// bytesOpt reads the same pointer as textOpt, and it runs no decode.
  int bytesOpt(String name) => point<GoString>(name)?.length ?? 0;

  double? sizeOpt(String name) => point<double>(name);

  int? countOpt(String name) => point<int>(name);

  bool? flagOpt(String name) => point<bool>(name);

  Object? nested(String name) {
    final FieldDesc? f = _fields[name];
    if (f == null) return null;
    return f.get(value);
  }

  List<Iface?> children(String name) {
    final Object? v = raw(name);
    if (v is! Slice<Iface?>) return const <Iface?>[];
    return <Iface?>[for (int i = 0; i < v.len; i++) v[i]];
  }
}

/// walkText mirrors 1 pass of the real host over the tree, with the decode of
/// each text property.
void walkText(Iface? node) {
  if (node == null) return;
  final GoNodeView n = GoNodeView(node);
  switch (n.kind) {
    case 'gart.dev/ui.Text':
      sink += n.text('Data').length;
      if (n.nested('Style') != null) sink++;
      if (n.textOpt('TextAlign') != null) sink++;
      if (n.textOpt('TextDirection') != null) sink++;
      if (n.textOpt('Locale') != null) sink++;
      if (n.flagOpt('SoftWrap') != null) sink++;
      if (n.textOpt('Overflow') != null) sink++;
      if (n.sizeOpt('TextScaleFactor') != null) sink++;
      if (n.countOpt('MaxLines') != null) sink++;
      if (n.textOpt('SemanticsLabel') != null) sink++;
      if (n.textOpt('SemanticsIdentifier') != null) sink++;
      if (n.textOpt('TextWidthBasis') != null) sink++;
      if (n.countOpt('SelectionColor') != null) sink++;
      return;
    case 'gart.dev/ui.Column':
    case 'gart.dev/ui.Row':
      if (n.textOpt('MainAxisAlignment') != null) sink++;
      if (n.textOpt('MainAxisSize') != null) sink++;
      if (n.textOpt('CrossAxisAlignment') != null) sink++;
      if (n.textOpt('TextDirection') != null) sink++;
      if (n.textOpt('VerticalDirection') != null) sink++;
      if (n.textOpt('TextBaseline') != null) sink++;
      if (n.sizeOpt('Spacing') != null) sink++;
      for (final Iface? kid in n.children('Children')) {
        walkText(kid);
      }
      return;
  }
  throw StateError('the walk holds no case for the node ${n.kind}');
}

/// walkRaw reads the same fields as walkText, and it decodes no text. The
/// difference between the 2 is the price of the GoString conversions of 1
/// pass.
void walkRaw(Iface? node) {
  if (node == null) return;
  final GoNodeView n = GoNodeView(node);
  switch (n.kind) {
    case 'gart.dev/ui.Text':
      sink += n.bytes('Data');
      if (n.nested('Style') != null) sink++;
      sink += n.bytesOpt('TextAlign');
      sink += n.bytesOpt('TextDirection');
      sink += n.bytesOpt('Locale');
      if (n.flagOpt('SoftWrap') != null) sink++;
      sink += n.bytesOpt('Overflow');
      if (n.sizeOpt('TextScaleFactor') != null) sink++;
      if (n.countOpt('MaxLines') != null) sink++;
      sink += n.bytesOpt('SemanticsLabel');
      sink += n.bytesOpt('SemanticsIdentifier');
      sink += n.bytesOpt('TextWidthBasis');
      if (n.countOpt('SelectionColor') != null) sink++;
      return;
    case 'gart.dev/ui.Column':
    case 'gart.dev/ui.Row':
      sink += n.bytesOpt('MainAxisAlignment');
      sink += n.bytesOpt('MainAxisSize');
      sink += n.bytesOpt('CrossAxisAlignment');
      sink += n.bytesOpt('TextDirection');
      sink += n.bytesOpt('VerticalDirection');
      sink += n.bytesOpt('TextBaseline');
      if (n.sizeOpt('Spacing') != null) sink++;
      for (final Iface? kid in n.children('Children')) {
        walkRaw(kid);
      }
      return;
  }
  throw StateError('the walk holds no case for the node ${n.kind}');
}

/// buildOf gives the Build field of a Watch node. The host reads the same
/// field at the mount of a boundary.
Object? buildOf(Iface node) => GoNodeView(node).raw('Build');

/// kidsOf gives the children of a Column node.
List<Iface?> kidsOf(Iface node) => GoNodeView(node).children('Children');

/// walkData reads the text property alone, and it reads no optional field. The
/// difference against walkText is the price of the 12 optional fields that the
/// generated host of a Text reads.
void walkData(Iface? node) {
  if (node == null) return;
  final GoNodeView n = GoNodeView(node);
  switch (n.kind) {
    case 'gart.dev/ui.Text':
      sink += n.text('Data').length;
      return;
    case 'gart.dev/ui.Column':
    case 'gart.dev/ui.Row':
      for (final Iface? kid in n.children('Children')) {
        walkData(kid);
      }
      return;
  }
  throw StateError('the walk holds no case for the node ${n.kind}');
}

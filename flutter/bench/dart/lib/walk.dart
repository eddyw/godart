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
///
/// Each read names the position of the field and the name of it, exactly as
/// GoNode reads them. The position is the fast path, and the name guards it.

/// sink holds the result of each walk. A write to a top level variable keeps
/// the Dart compiler away from a drop of the work.
int sink = 0;

/// The position of each field of a ui.Text, in the order of the Go struct.
const int textData = 0;
const int textStyle = 1;
const int textAlign = 2;
const int textDirection = 3;
const int textLocale = 4;
const int textSoftWrap = 5;
const int textOverflow = 6;
const int textScaleFactor = 7;
const int textMaxLines = 8;
const int textSemanticsLabel = 9;
const int textSemanticsIdentifier = 10;
const int textWidthBasis = 11;
const int textSelectionColor = 12;

/// The position of each field of a ui.Column and of a ui.Row.
const int flexMainAxisAlignment = 0;
const int flexMainAxisSize = 1;
const int flexCrossAxisAlignment = 2;
const int flexTextDirection = 3;
const int flexVerticalDirection = 4;
const int flexTextBaseline = 5;
const int flexSpacing = 6;
const int flexChildren = 7;

/// GoNodeView reads 1 node of the Go tree through its type descriptor. The
/// method set holds only the readers that this measurement needs.
final class GoNodeView {
  final Object? value;
  final TypeDesc desc;
  final List<FieldDesc> _fields;

  GoNodeView(Iface node)
      : value = node.value,
        desc = node.desc,
        _fields = node.desc.fields;

  String get kind => desc.name;

  /// noAt stands for "the caller names no position", as in GoNode.
  static const int noAt = 0x7fffffff;

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

  Object? raw(String name, [int at = noAt]) {
    final List<FieldDesc> f = _fields;
    if (at < f.length) {
      final FieldDesc d = f[at];
      if (identical(d.name, name)) return d.get(value);
    }
    return _find(name)?.get(value);
  }

  T? point<T>(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<T> ? v.v : null;
  }

  String text(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is GoString ? v.toDartString() : '';
  }

  /// bytes gives the byte count of a string field, and it runs no decode. The
  /// difference against text is the price of GoString.toDartString.
  int bytes(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is GoString ? v.length : 0;
  }

  String? textOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<GoString> ? v.v.toDartString() : null;
  }

  /// goText gives a string field with no decode. The diff reads it, because a
  /// compare of 2 GoString values reads the bytes and allocates nothing.
  GoString? goText(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is GoString ? v : null;
  }

  /// goTextOpt gives an optional string field with no decode.
  GoString? goTextOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<GoString> ? v.v : null;
  }

  /// bytesOpt reads the same pointer as textOpt, and it runs no decode.
  int bytesOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<GoString> ? v.v.length : 0;
  }

  double? sizeOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<double> ? v.v : null;
  }

  int? countOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<int> ? v.v : null;
  }

  bool? flagOpt(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
    return v is $Ptr<bool> ? v.v : null;
  }

  Object? nested(String name, [int at = noAt]) => raw(name, at);

  List<Iface?> children(String name, [int at = noAt]) {
    final Object? v = raw(name, at);
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
      sink += n.text('Data', textData).length;
      if (n.nested('Style', textStyle) != null) sink++;
      if (n.textOpt('TextAlign', textAlign) != null) sink++;
      if (n.textOpt('TextDirection', textDirection) != null) sink++;
      if (n.textOpt('Locale', textLocale) != null) sink++;
      if (n.flagOpt('SoftWrap', textSoftWrap) != null) sink++;
      if (n.textOpt('Overflow', textOverflow) != null) sink++;
      if (n.sizeOpt('TextScaleFactor', textScaleFactor) != null) sink++;
      if (n.countOpt('MaxLines', textMaxLines) != null) sink++;
      if (n.textOpt('SemanticsLabel', textSemanticsLabel) != null) sink++;
      if (n.textOpt('SemanticsIdentifier', textSemanticsIdentifier) != null) {
        sink++;
      }
      if (n.textOpt('TextWidthBasis', textWidthBasis) != null) sink++;
      if (n.countOpt('SelectionColor', textSelectionColor) != null) sink++;
      return;
    case 'gart.dev/ui.Column':
    case 'gart.dev/ui.Row':
      if (n.textOpt('MainAxisAlignment', flexMainAxisAlignment) != null) sink++;
      if (n.textOpt('MainAxisSize', flexMainAxisSize) != null) sink++;
      if (n.textOpt('CrossAxisAlignment', flexCrossAxisAlignment) != null) {
        sink++;
      }
      if (n.textOpt('TextDirection', flexTextDirection) != null) sink++;
      if (n.textOpt('VerticalDirection', flexVerticalDirection) != null) sink++;
      if (n.textOpt('TextBaseline', flexTextBaseline) != null) sink++;
      if (n.sizeOpt('Spacing', flexSpacing) != null) sink++;
      for (final Iface? kid in n.children('Children', flexChildren)) {
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
      sink += n.bytes('Data', textData);
      if (n.nested('Style', textStyle) != null) sink++;
      sink += n.bytesOpt('TextAlign', textAlign);
      sink += n.bytesOpt('TextDirection', textDirection);
      sink += n.bytesOpt('Locale', textLocale);
      if (n.flagOpt('SoftWrap', textSoftWrap) != null) sink++;
      sink += n.bytesOpt('Overflow', textOverflow);
      if (n.sizeOpt('TextScaleFactor', textScaleFactor) != null) sink++;
      if (n.countOpt('MaxLines', textMaxLines) != null) sink++;
      sink += n.bytesOpt('SemanticsLabel', textSemanticsLabel);
      sink += n.bytesOpt('SemanticsIdentifier', textSemanticsIdentifier);
      sink += n.bytesOpt('TextWidthBasis', textWidthBasis);
      if (n.countOpt('SelectionColor', textSelectionColor) != null) sink++;
      return;
    case 'gart.dev/ui.Column':
    case 'gart.dev/ui.Row':
      sink += n.bytesOpt('MainAxisAlignment', flexMainAxisAlignment);
      sink += n.bytesOpt('MainAxisSize', flexMainAxisSize);
      sink += n.bytesOpt('CrossAxisAlignment', flexCrossAxisAlignment);
      sink += n.bytesOpt('TextDirection', flexTextDirection);
      sink += n.bytesOpt('VerticalDirection', flexVerticalDirection);
      sink += n.bytesOpt('TextBaseline', flexTextBaseline);
      if (n.sizeOpt('Spacing', flexSpacing) != null) sink++;
      for (final Iface? kid in n.children('Children', flexChildren)) {
        walkRaw(kid);
      }
      return;
  }
  throw StateError('the walk holds no case for the node ${n.kind}');
}

/// buildOf gives the Build field of a Watch node. The host reads the same
/// field at the mount of a boundary.
Object? buildOf(Iface node) => GoNodeView(node).raw('Build', 0);

/// kidsOf gives the children of a Column node.
List<Iface?> kidsOf(Iface node) =>
    GoNodeView(node).children('Children', flexChildren);

/// walkData reads the text property alone, and it reads no optional field. The
/// difference against walkText is the price of the 12 optional fields that the
/// generated host of a Text reads.
void walkData(Iface? node) {
  if (node == null) return;
  final GoNodeView n = GoNodeView(node);
  switch (n.kind) {
    case 'gart.dev/ui.Text':
      sink += n.text('Data', textData).length;
      return;
    case 'gart.dev/ui.Column':
    case 'gart.dev/ui.Row':
      for (final Iface? kid in n.children('Children', flexChildren)) {
        walkData(kid);
      }
      return;
  }
  throw StateError('the walk holds no case for the node ${n.kind}');
}

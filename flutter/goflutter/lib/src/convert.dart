import 'package:flutter/material.dart';
import 'package:goruntime/goruntime.dart';

import 'host.dart';
import 'node.dart';
import 'runtime.dart';

/// This file holds the converters that the generated host reads. Each one
/// turns 1 Go value into the Flutter value of the same meaning.
///
/// A null result always means "the default of Flutter". The generated host
/// then writes the default of the constructor, or it throws when the parameter
/// is required.

/// goColorOpt gives the color of an optional field.
Color? goColorOpt(int? argb) => argb == null ? null : Color(argb);

/// goMillis gives the duration of an optional field.
Duration? goMillis(int? ms) =>
    ms == null ? null : Duration(milliseconds: ms);

/// goIcon reads the name of an icon. An application adds a name to goIcons.
IconData? goIcon(String? name) =>
    name == null || name.isEmpty ? null : goIcons[name];

/// goChildOpt builds 1 child, and it gives null when the Go field holds no
/// node. A required child then reports its own name.
Widget? goChildOpt(Iface? node, GoRuntime rt) =>
    node == null ? null : goRender(node, rt);

/// goChildren builds each child of a slice.
List<Widget> goChildren(List<Iface?> nodes, GoRuntime rt) =>
    <Widget>[for (final Iface? n in nodes) goRender(n, rt)];

/// goAction wraps a callback with no result. The Go function may block, thus
/// the host runs it as a goroutine.
VoidCallback? goAction(void Function()? f, GoRuntime rt) =>
    f == null ? null : () => rt.run(f);

void Function(String?)? goOnStr(void Function(GoString)? f, GoRuntime rt) =>
    f == null
        ? null
        : (String? v) => rt.run(() => f(GoString.fromString(v ?? '')));

void Function(double?)? goOnF64(void Function(double)? f, GoRuntime rt) =>
    f == null ? null : (double? v) => rt.run(() => f(v ?? 0));

void Function(int?)? goOnI64(void Function(int)? f, GoRuntime rt) =>
    f == null ? null : (int? v) => rt.run(() => f(v ?? 0));

void Function(bool?)? goOnBool(void Function(bool)? f, GoRuntime rt) =>
    f == null ? null : (bool? v) => rt.run(() => f(v ?? false));

void Function(Color?)? goOnColor(void Function(int)? f, GoRuntime rt) =>
    f == null ? null : (Color? v) => rt.run(() => f(v == null ? 0 : v.toARGB32()));

void Function(Duration?)? goOnDuration(
        void Function(int)? f, GoRuntime rt) =>
    f == null
        ? null
        : (Duration? v) => rt.run(() => f(v == null ? 0 : v.inMilliseconds));

/// goOnEnum wraps a callback that takes 1 enum value. The Go side reads the
/// name of the value, and name writes it.
void Function(T?)? goOnEnum<T>(
  void Function(GoString)? f,
  GoRuntime rt,
  String Function(T) name,
) =>
    f == null
        ? null
        : (T? v) => rt.run(
            () => f(GoString.fromString(v == null ? '' : name(v))));

/// goBuild wraps a builder that takes no index. The compiler proves that the
/// function never blocks, because its Go type carries //gart:sync.
Widget Function(BuildContext)? goBuild(Iface? Function()? f, GoRuntime rt) =>
    f == null ? null : (BuildContext _) => goRender(f(), rt);

/// goItem wraps a builder of 1 row.
Widget Function(BuildContext, int)? goItem(
        Iface? Function(int)? f, GoRuntime rt) =>
    f == null ? null : (BuildContext _, int i) => goRender(f(i), rt);

/// goInsets reads the Insets struct of the Go side.
EdgeInsets? goInsets(GoNode? n) => n == null
    ? null
    : EdgeInsets.fromLTRB(
        n.size('Left'), n.size('Top'), n.size('Right'), n.size('Bottom'));

/// goOffset reads the Offset struct of the Go side.
Offset? goOffset(GoNode? n) =>
    n == null ? null : Offset(n.size('Dx'), n.size('Dy'));

/// goConstraints reads the Constraints struct of the Go side. A maximum of 0
/// means no limit.
BoxConstraints? goConstraints(GoNode? n) {
  if (n == null) return null;
  final double maxW = n.size('MaxWidth');
  final double maxH = n.size('MaxHeight');
  return BoxConstraints(
    minWidth: n.size('MinWidth'),
    maxWidth: maxW == 0 ? double.infinity : maxW,
    minHeight: n.size('MinHeight'),
    maxHeight: maxH == 0 ? double.infinity : maxH,
  );
}

const Map<String, FontWeight> _weights = <String, FontWeight>{
  'thin': FontWeight.w100,
  'light': FontWeight.w300,
  'normal': FontWeight.w400,
  'medium': FontWeight.w500,
  'semibold': FontWeight.w600,
  'bold': FontWeight.w700,
  'black': FontWeight.w900,
};

const Map<String, TextDecoration> _decorations = <String, TextDecoration>{
  'underline': TextDecoration.underline,
  'overline': TextDecoration.overline,
  'through': TextDecoration.lineThrough,
};

/// goTextStyle reads the TextStyle struct of the Go side.
TextStyle? goTextStyle(GoNode? n) {
  if (n == null) return null;
  final String family = n.text('Family');
  return TextStyle(
    color: goColorOpt(n.countOpt('Color')),
    fontSize: n.sizeOpt('Size'),
    fontWeight: _weights[n.text('Weight')],
    fontStyle: n.flag('Italic') ? FontStyle.italic : null,
    letterSpacing: n.sizeOpt('LetterSpacing'),
    wordSpacing: n.sizeOpt('WordSpacing'),
    height: n.sizeOpt('Height'),
    decoration: _decorations[n.text('Decoration')],
    fontFamily: family.isEmpty ? null : family,
  );
}

const Map<String, Alignment> _alignments = <String, Alignment>{
  'topLeft': Alignment.topLeft,
  'topCenter': Alignment.topCenter,
  'topRight': Alignment.topRight,
  'centerLeft': Alignment.centerLeft,
  'center': Alignment.center,
  'centerRight': Alignment.centerRight,
  'bottomLeft': Alignment.bottomLeft,
  'bottomCenter': Alignment.bottomCenter,
  'bottomRight': Alignment.bottomRight,
};

/// goAlignment reads the name of an alignment.
AlignmentGeometry? goAlignment(String? s) => s == null ? null : _alignments[s];

/// goAlignmentXY reads the name of an alignment that holds no direction.
Alignment? goAlignmentXY(String? s) => s == null ? null : _alignments[s];

const Map<String, Curve> _curves = <String, Curve>{
  'linear': Curves.linear,
  'ease': Curves.ease,
  'easeIn': Curves.easeIn,
  'easeOut': Curves.easeOut,
  'easeInOut': Curves.easeInOut,
  'bounceIn': Curves.bounceIn,
  'bounceOut': Curves.bounceOut,
  'elasticIn': Curves.elasticIn,
  'elasticOut': Curves.elasticOut,
  'fastOutSlowIn': Curves.fastOutSlowIn,
  'decelerate': Curves.decelerate,
};

/// goCurve reads the name of a curve.
Curve? goCurve(String? s) => s == null ? null : _curves[s];

/// goCast builds 1 child and gives it the subtype that the parameter needs.
///
/// The appBar of a Scaffold needs a PreferredSizeWidget, and a Go value carries
/// no Dart type, thus the cast stands here. A widget that gives no preferred
/// size takes the height of a toolbar.
T? goCast<T>(Iface? node, GoRuntime rt, String field) {
  if (node == null) return null;
  final Widget w = goRender(node, rt);
  if (w is T) return w as T;
  if (T == PreferredSizeWidget) {
    return PreferredSize(
          preferredSize: const Size.fromHeight(kToolbarHeight),
          child: w,
        )
        as T;
  }
  throw StateError('the field $field holds a ${w.runtimeType}, and the '
      'parameter needs a $T');
}

const Map<String, MouseCursor> _cursors = <String, MouseCursor>{
  'none': SystemMouseCursors.none,
  'basic': SystemMouseCursors.basic,
  'click': SystemMouseCursors.click,
  'forbidden': SystemMouseCursors.forbidden,
  'wait': SystemMouseCursors.wait,
  'progress': SystemMouseCursors.progress,
  'contextMenu': SystemMouseCursors.contextMenu,
  'help': SystemMouseCursors.help,
  'text': SystemMouseCursors.text,
  'verticalText': SystemMouseCursors.verticalText,
  'cell': SystemMouseCursors.cell,
  'precise': SystemMouseCursors.precise,
  'move': SystemMouseCursors.move,
  'grab': SystemMouseCursors.grab,
  'grabbing': SystemMouseCursors.grabbing,
  'noDrop': SystemMouseCursors.noDrop,
  'alias': SystemMouseCursors.alias,
  'copy': SystemMouseCursors.copy,
  'allScroll': SystemMouseCursors.allScroll,
  'resizeLeftRight': SystemMouseCursors.resizeLeftRight,
  'resizeUpDown': SystemMouseCursors.resizeUpDown,
  'zoomIn': SystemMouseCursors.zoomIn,
  'zoomOut': SystemMouseCursors.zoomOut,
  'defer': MouseCursor.defer,
  'uncontrolled': MouseCursor.uncontrolled,
};

/// goCursor reads the name of a mouse cursor.
MouseCursor? goCursor(String? s) => s == null ? null : _cursors[s];

/// goPhysics reads the name of a scroll physics.
///
/// Each one takes the physics of the platform as its parent, thus a list keeps
/// the feel of the platform under the named rule.
ScrollPhysics? goPhysics(String? s) {
  switch (s) {
    case 'bouncing':
      return const BouncingScrollPhysics();
    case 'clamping':
      return const ClampingScrollPhysics();
    case 'never':
      return const NeverScrollableScrollPhysics();
    case 'always':
      return const AlwaysScrollableScrollPhysics();
    case 'page':
      return const PageScrollPhysics();
    case 'range':
      return const RangeMaintainingScrollPhysics();
    case 'fixedExtent':
      return const FixedExtentScrollPhysics();
  }
  return null;
}

/// goLocale reads a language tag, for example `en` or `en_US`.
Locale? goLocale(String? s) {
  if (s == null || s.isEmpty) return null;
  final List<String> parts = s.split(RegExp(r'[_-]'));
  if (parts.length == 1) return Locale(parts[0]);
  return Locale(parts[0], parts[1]);
}

/// goBorderSide reads the BorderSide struct of the Go side.
BorderSide? goBorderSide(GoNode? n) {
  if (n == null) return null;
  return BorderSide(
    color: goColorOpt(n.countOpt('Color')) ?? const BorderSide().color,
    width: n.sizeOpt('Width') ?? 1,
    style: n.text('Style') == 'none' ? BorderStyle.none : BorderStyle.solid,
  );
}

/// goRadius reads the Radius struct of the Go side.
BorderRadius? goRadius(GoNode? n) {
  if (n == null) return null;
  return BorderRadius.only(
    topLeft: Radius.circular(n.size('TopLeft')),
    topRight: Radius.circular(n.size('TopRight')),
    bottomLeft: Radius.circular(n.size('BottomLeft')),
    bottomRight: Radius.circular(n.size('BottomRight')),
  );
}

/// goShape reads the Shape struct of the Go side.
///
/// The result is an OutlinedBorder, thus a parameter of that type and a
/// parameter of ShapeBorder each take it.
OutlinedBorder? goShape(GoNode? n) {
  if (n == null) return null;
  final BorderSide side = goBorderSide(n.nested('Side')) ?? BorderSide.none;
  final BorderRadius radius = goRadius(n.nested('Radius')) ?? BorderRadius.zero;
  switch (n.text('Kind')) {
    case 'rounded':
      return RoundedRectangleBorder(borderRadius: radius, side: side);
    case 'stadium':
      return StadiumBorder(side: side);
    case 'circle':
      return CircleBorder(side: side);
    case 'beveled':
      return BeveledRectangleBorder(borderRadius: radius, side: side);
  }
  return RoundedRectangleBorder(side: side);
}

/// goDensity reads the Density struct of the Go side.
VisualDensity? goDensity(GoNode? n) => n == null
    ? null
    : VisualDensity(
        horizontal: n.size('Horizontal'),
        vertical: n.size('Vertical'),
      );

/// goIconStyle reads the IconStyle struct of the Go side.
IconThemeData? goIconStyle(GoNode? n) => n == null
    ? null
    : IconThemeData(
        color: goColorOpt(n.countOpt('Color')),
        size: n.sizeOpt('Size'),
        opacity: n.sizeOpt('Opacity'),
        fill: n.sizeOpt('Fill'),
        weight: n.sizeOpt('Weight'),
      );

/// goStateColor reads the StateColor struct of the Go side.
///
/// A state with no color of its own takes the color of Default. A nil Default
/// gives null for that state, thus Flutter reads its own default.
WidgetStateProperty<Color?>? goStateColor(GoNode? n) {
  if (n == null) return null;
  final Color? base = goColorOpt(n.countOpt('Default'));
  final Color? hovered = goColorOpt(n.countOpt('Hovered'));
  final Color? focused = goColorOpt(n.countOpt('Focused'));
  final Color? pressed = goColorOpt(n.countOpt('Pressed'));
  final Color? selected = goColorOpt(n.countOpt('Selected'));
  final Color? disabled = goColorOpt(n.countOpt('Disabled'));
  return WidgetStateProperty.resolveWith((Set<WidgetState> states) {
    if (states.contains(WidgetState.disabled)) return disabled ?? base;
    if (states.contains(WidgetState.pressed)) return pressed ?? base;
    if (states.contains(WidgetState.hovered)) return hovered ?? base;
    if (states.contains(WidgetState.focused)) return focused ?? base;
    if (states.contains(WidgetState.selected)) return selected ?? base;
    return base;
  });
}

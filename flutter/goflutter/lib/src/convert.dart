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
        n.size('Left', 0), n.size('Top', 1), n.size('Right', 2),
        n.size('Bottom', 3));

/// goOffset reads the Offset struct of the Go side.
Offset? goOffset(GoNode? n) =>
    n == null ? null : Offset(n.size('Dx', 0), n.size('Dy', 1));

/// goConstraints reads the Constraints struct of the Go side. A maximum of 0
/// means no limit.
BoxConstraints? goConstraints(GoNode? n) {
  if (n == null) return null;
  final double maxW = n.size('MaxWidth', 1);
  final double maxH = n.size('MaxHeight', 3);
  return BoxConstraints(
    minWidth: n.size('MinWidth', 0),
    maxWidth: maxW == 0 ? double.infinity : maxW,
    minHeight: n.size('MinHeight', 2),
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
  final String family = n.text('Family', 8);
  return TextStyle(
    color: goColorOpt(n.countOpt('Color', 0)),
    fontSize: n.sizeOpt('Size', 1),
    fontWeight: _weights[n.text('Weight', 2)],
    fontStyle: n.flag('Italic', 3) ? FontStyle.italic : null,
    letterSpacing: n.sizeOpt('LetterSpacing', 4),
    wordSpacing: n.sizeOpt('WordSpacing', 5),
    height: n.sizeOpt('Height', 6),
    decoration: _decorations[n.text('Decoration', 7)],
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
    color: goColorOpt(n.countOpt('Color', 0)) ?? const BorderSide().color,
    width: n.sizeOpt('Width', 1) ?? 1,
    style: n.text('Style', 2) == 'none' ? BorderStyle.none : BorderStyle.solid,
  );
}

/// goRadius reads the Radius struct of the Go side.
BorderRadius? goRadius(GoNode? n) {
  if (n == null) return null;
  return BorderRadius.only(
    topLeft: Radius.circular(n.size('TopLeft', 0)),
    topRight: Radius.circular(n.size('TopRight', 1)),
    bottomLeft: Radius.circular(n.size('BottomLeft', 2)),
    bottomRight: Radius.circular(n.size('BottomRight', 3)),
  );
}

/// goShape reads the Shape struct of the Go side.
///
/// The result is an OutlinedBorder, thus a parameter of that type and a
/// parameter of ShapeBorder each take it.
OutlinedBorder? goShape(GoNode? n) {
  if (n == null) return null;
  final BorderSide side = goBorderSide(n.nested('Side', 2)) ?? BorderSide.none;
  final BorderRadius radius =
      goRadius(n.nested('Radius', 1)) ?? BorderRadius.zero;
  switch (n.text('Kind', 0)) {
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
        horizontal: n.size('Horizontal', 0),
        vertical: n.size('Vertical', 1),
      );

/// goIconStyle reads the IconStyle struct of the Go side.
IconThemeData? goIconStyle(GoNode? n) => n == null
    ? null
    : IconThemeData(
        color: goColorOpt(n.countOpt('Color', 0)),
        size: n.sizeOpt('Size', 1),
        opacity: n.sizeOpt('Opacity', 2),
        fill: n.sizeOpt('Fill', 3),
        weight: n.sizeOpt('Weight', 4),
      );

/// goStateColor reads the StateColor struct of the Go side.
///
/// A state with no color of its own takes the color of Default. A nil Default
/// gives null for that state, thus Flutter reads its own default.
WidgetStateProperty<Color?>? goStateColor(GoNode? n) {
  if (n == null) return null;
  final Color? base = goColorOpt(n.countOpt('Default', 0));
  final Color? hovered = goColorOpt(n.countOpt('Hovered', 1));
  final Color? focused = goColorOpt(n.countOpt('Focused', 2));
  final Color? pressed = goColorOpt(n.countOpt('Pressed', 3));
  final Color? selected = goColorOpt(n.countOpt('Selected', 4));
  final Color? disabled = goColorOpt(n.countOpt('Disabled', 5));
  return WidgetStateProperty.resolveWith((Set<WidgetState> states) {
    if (states.contains(WidgetState.disabled)) return disabled ?? base;
    if (states.contains(WidgetState.pressed)) return pressed ?? base;
    if (states.contains(WidgetState.hovered)) return hovered ?? base;
    if (states.contains(WidgetState.focused)) return focused ?? base;
    if (states.contains(WidgetState.selected)) return selected ?? base;
    return base;
  });
}

/// goSize reads the Size struct of the Go side.
Size? goSize(GoNode? n) =>
    n == null ? null : Size(n.size('Width', 0), n.size('Height', 1));

/// goImage reads the ImageSource struct of the Go side.
///
/// No kind names a file of the disk. FileImage reads dart:io, and the host
/// also builds for the web, thus a Go program gives the bytes of the file.
ImageProvider<Object>? goImage(GoNode? n) {
  if (n == null) return null;
  final double scale = n.sizeOpt('Scale', 3) ?? 1;
  final String src = n.text('Src', 1);
  switch (n.text('Kind', 0)) {
    case 'network':
      if (src.isEmpty) return null;
      return NetworkImage(src, scale: scale);
    case 'asset':
      if (src.isEmpty) return null;
      final String pkg = n.text('Package', 4);
      return AssetImage(src, package: pkg.isEmpty ? null : pkg);
    case 'memory':
      final List<int> bytes = n.counts('Bytes', 2);
      if (bytes.isEmpty) return null;
      return MemoryImage(Uint8List.fromList(bytes), scale: scale);
  }
  return null;
}

final Map<String, BoxFit> _fits = BoxFit.values.asNameMap();

/// goShadows reads the Shadow structs of a decoration.
List<BoxShadow>? goShadows(List<GoNode> list) {
  if (list.isEmpty) return null;
  return <BoxShadow>[
    for (final GoNode n in list)
      BoxShadow(
        color: goColorOpt(n.argb('Color', 0)) ?? const BoxShadow().color,
        offset: goOffset(n.nested('Offset', 1)) ?? Offset.zero,
        blurRadius: n.size('BlurRadius', 2),
        spreadRadius: n.size('SpreadRadius', 3),
      ),
  ];
}

/// goGradient reads the Gradient struct of the Go side.
///
/// Flutter demands 2 colors at least, and a shorter list gives no gradient.
/// The count of the stops must match the count of the colors, thus a list of
/// another length spreads the colors evenly.
Gradient? goGradient(GoNode? n) {
  if (n == null) return null;
  final List<Color> colors = <Color>[
    for (final int v in n.counts('Colors', 1)) Color(v),
  ];
  if (colors.length < 2) return null;
  final List<double> stops = n.sizes('Stops', 2);
  final List<double>? at = stops.length == colors.length ? stops : null;
  if (n.text('Kind', 0) == 'radial') {
    final double radius = n.size('Radius', 6);
    return RadialGradient(
      colors: colors,
      stops: at,
      center: goAlignment(n.text('Center', 5)) ?? Alignment.center,
      radius: radius == 0 ? 0.5 : radius,
    );
  }
  return LinearGradient(
    colors: colors,
    stops: at,
    begin: goAlignment(n.text('Begin', 3)) ?? Alignment.centerLeft,
    end: goAlignment(n.text('End', 4)) ?? Alignment.centerRight,
  );
}

/// goDecoration reads the Decoration struct of the Go side.
///
/// A circle carries no border radius, because Flutter rejects the pair.
BoxDecoration? goDecoration(GoNode? n) {
  if (n == null) return null;
  final BorderSide? side = goBorderSide(n.nested('Border', 3));
  final ImageProvider<Object>? image = goImage(n.nested('Image', 1));
  final bool circle = n.text('Shape', 7) == 'circle';
  return BoxDecoration(
    color: goColorOpt(n.countOpt('Color', 0)),
    image: image == null
        ? null
        : DecorationImage(image: image, fit: _fits[n.text('Fit', 2)]),
    border: side == null ? null : Border.fromBorderSide(side),
    borderRadius: circle ? null : goRadius(n.nested('Radius', 4)),
    boxShadow: goShadows(n.structs('Shadows', 5)),
    gradient: goGradient(n.nested('Gradient', 6)),
    shape: circle ? BoxShape.circle : BoxShape.rectangle,
  );
}

const Map<String, MaterialTapTargetSize> _tapTargets =
    <String, MaterialTapTargetSize>{
  'padded': MaterialTapTargetSize.padded,
  'shrinkWrap': MaterialTapTargetSize.shrinkWrap,
};

/// _all gives 1 value for each state of a Material widget.
WidgetStateProperty<T>? _all<T>(T? v) =>
    v == null ? null : WidgetStatePropertyAll<T>(v);

/// goButtonStyle reads the ButtonStyle struct of the Go side.
///
/// A color carries 1 value for each state, and each other field carries 1
/// value for the whole button.
ButtonStyle? goButtonStyle(GoNode? n) {
  if (n == null) return null;
  return ButtonStyle(
    backgroundColor: goStateColor(n.nested('BackgroundColor', 0)),
    foregroundColor: goStateColor(n.nested('ForegroundColor', 1)),
    overlayColor: goStateColor(n.nested('OverlayColor', 2)),
    shadowColor: goStateColor(n.nested('ShadowColor', 3)),
    surfaceTintColor: goStateColor(n.nested('SurfaceTintColor', 4)),
    iconColor: goStateColor(n.nested('IconColor', 5)),
    textStyle: _all<TextStyle>(goTextStyle(n.nested('TextStyle', 6))),
    elevation: _all<double>(n.sizeOpt('Elevation', 7)),
    iconSize: _all<double>(n.sizeOpt('IconSize', 8)),
    padding: _all<EdgeInsetsGeometry>(goInsets(n.nested('Padding', 9))),
    minimumSize: _all<Size>(goSize(n.nested('MinimumSize', 10))),
    maximumSize: _all<Size>(goSize(n.nested('MaximumSize', 11))),
    fixedSize: _all<Size>(goSize(n.nested('FixedSize', 12))),
    side: _all<BorderSide>(goBorderSide(n.nested('Side', 13))),
    shape: _all<OutlinedBorder>(goShape(n.nested('Shape', 14))),
    alignment: goAlignment(n.text('Alignment', 15)),
    visualDensity: goDensity(n.nested('VisualDensity', 16)),
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 17))),
    tapTargetSize: _tapTargets[n.text('TapTargetSize', 18)],
    animationDuration: goMillis(n.countOpt('AnimationMillis', 19)),
    enableFeedback: n.flagOpt('EnableFeedback', 20),
  );
}

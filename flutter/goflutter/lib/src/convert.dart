import 'package:cupertino_ui/cupertino_ui.dart';
import 'package:material_ui/material_ui.dart';
// The 8 names below stand outside material.dart and cupertino.dart. A show
// clause holds them, thus no other name of rendering.dart reaches this file.
import 'package:flutter/rendering.dart'
    show
        AccessibilityFocusBlockType,
        ItemExtentBuilder,
        SelectionGeometry,
        SelectionStatus,
        SemanticsProperties,
        SemanticsRole,
        SemanticsValidationResult,
        SliverLayoutDimensions;
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
Duration? goMillis(int? ms) => ms == null ? null : Duration(milliseconds: ms);

/// goIcon reads the name of an icon.
///
/// nodegen writes each icon that a string constant of the Go source names, and
/// goGenInstall puts that set into goIcons. A name that the Go source builds at
/// run time reaches no such constant, thus the application adds it to goIcons
/// itself. An unknown name reports its own name, because a silent empty icon
/// reads as a defect of the layout.
IconData? goIcon(String? name) {
  if (name == null || name.isEmpty) return null;
  final IconData? out = goIcons[name];
  if (out == null) {
    throw StateError(
      'the host holds no icon of the name $name. '
      'Add it to goIcons, or name it with a string constant of the Go '
      'source.',
    );
  }
  return out;
}

/// goValueKey reads an optional key field. An empty text and a nil pointer
/// each give no key, thus the widget takes the key that its place gives.
Key? goValueKey(String? name) =>
    name == null || name.isEmpty ? null : ValueKey<String>(name);

/// goChildOpt builds 1 child, and it gives null when the Go field holds no
/// node. A required child then reports its own name.
Widget? goChildOpt(Iface? node, GoRuntime rt) =>
    node == null ? null : goRender(node, rt);

/// goChildren builds each child of a slice.
List<Widget> goChildren(List<Iface?> nodes, GoRuntime rt) => <Widget>[
  for (final Iface? n in nodes) goRender(n, rt),
];

List<Widget>? goChildrenOpt(List<Iface?>? nodes, GoRuntime rt) =>
    nodes == null ? null : goChildren(nodes, rt);

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
    f == null
    ? null
    : (Color? v) => rt.run(() => f(v == null ? 0 : v.toARGB32()));

void Function(Duration?)? goOnDuration(void Function(int)? f, GoRuntime rt) =>
    f == null
    ? null
    : (Duration? v) => rt.run(() => f(v == null ? 0 : v.inMilliseconds));

/// goOnEnum wraps a callback that takes 1 enum value. The Go side reads the
/// name of the value, and name writes it.
void Function(T?)? goOnEnum<T>(
  void Function(GoString)? f,
  GoRuntime rt,
  String Function(T) name,
) => f == null
    ? null
    : (T? v) => rt.run(() => f(GoString.fromString(v == null ? '' : name(v))));

/// goBuild wraps a builder that takes no index. The compiler proves that the
/// function never blocks, because its Go type carries //gart:sync.
Widget Function(BuildContext)? goBuild(Iface? Function()? f, GoRuntime rt) =>
    f == null ? null : (BuildContext _) => goRender(f(), rt);

/// goItem wraps a builder of 1 row.
Widget Function(BuildContext, int)? goItem(
  Iface? Function(int)? f,
  GoRuntime rt,
) => f == null ? null : (BuildContext _, int i) => goRender(f(i), rt);

/// goInsets reads the Insets struct of the Go side.
EdgeInsets? goInsets(GoNode? n) => n == null
    ? null
    : EdgeInsets.fromLTRB(
        n.size('Left', 0),
        n.size('Top', 1),
        n.size('Right', 2),
        n.size('Bottom', 3),
      );

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
  'extralight': FontWeight.w200,
  'light': FontWeight.w300,
  'normal': FontWeight.w400,
  'medium': FontWeight.w500,
  'semibold': FontWeight.w600,
  'bold': FontWeight.w700,
  'extrabold': FontWeight.w800,
  'black': FontWeight.w900,
  // Flutter names each weight w100 to w900. A Go program that reads the
  // Flutter documentation writes such a name, thus the table holds each 1.
  'w100': FontWeight.w100,
  'w200': FontWeight.w200,
  'w300': FontWeight.w300,
  'w400': FontWeight.w400,
  'w500': FontWeight.w500,
  'w600': FontWeight.w600,
  'w700': FontWeight.w700,
  'w800': FontWeight.w800,
  'w900': FontWeight.w900,
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
    shadows: goShadows(n.structs('Shadows', 9)),
    fontVariations: goVariations(n.structs('Variations', 10)),
  );
}

/// goVariations reads the FontVariation structs of a text style.
List<FontVariation>? goVariations(List<GoNode> list) {
  if (list.isEmpty) return null;
  return <FontVariation>[
    for (final GoNode n in list)
      FontVariation(n.text('Axis', 0), n.size('Value', 1)),
  ];
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
    ) as T;
  }
  throw StateError(
    'the field $field holds a ${w.runtimeType}, and the '
    'parameter needs a $T',
  );
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

/// goDate reads a date. The Go side holds the milliseconds of the Unix epoch,
/// and a value of 0 means no date.
DateTime? goDate(int? ms) =>
    ms == null || ms == 0 ? null : DateTime.fromMillisecondsSinceEpoch(ms);

/// goOnDate wraps a callback that takes 1 date.
void Function(DateTime)? goOnDate(void Function(int)? f, GoRuntime rt) =>
    f == null
    ? null
    : (DateTime v) => rt.run(() => f(v.millisecondsSinceEpoch));

/// goClock reads a time of the day. The Go side holds the minutes after
/// midnight.
TimeOfDay? goClock(int? m) =>
    m == null ? null : TimeOfDay(hour: (m ~/ 60) % 24, minute: m % 60);

/// goOnClock wraps a callback that takes 1 time of the day.
void Function(TimeOfDay)? goOnClock(void Function(int)? f, GoRuntime rt) =>
    f == null ? null : (TimeOfDay v) => rt.run(() => f(v.hour * 60 + v.minute));

/// goBytes reads a byte slice. An empty slice gives null, thus a required
/// parameter reports the name of the field that holds no bytes.
Uint8List? goBytes(List<int> bytes) =>
    bytes.isEmpty ? null : Uint8List.fromList(bytes);

/// goRect reads the Rect struct of the Go side.
Rect? goRect(GoNode? n) => n == null
    ? null
    : Rect.fromLTRB(
        n.size('Left', 0),
        n.size('Top', 1),
        n.size('Right', 2),
        n.size('Bottom', 3),
      );

/// goRelRect reads the Rect struct of the Go side as a RelativeRect. Each edge
/// gives the distance from that edge of the parent.
RelativeRect? goRelRect(GoNode? n) => n == null
    ? null
    : RelativeRect.fromLTRB(
        n.size('Left', 0),
        n.size('Top', 1),
        n.size('Right', 2),
        n.size('Bottom', 3),
      );

/// goRange reads the Range struct of the Go side.
RangeValues? goRange(GoNode? n) =>
    n == null ? null : RangeValues(n.size('Start', 0), n.size('End', 1));

/// goOnRange wraps a callback that takes a range. The Go function takes the 2
/// ends, because the host cannot build a Go struct.
void Function(RangeValues)? goOnRange(
  void Function(double, double)? f,
  GoRuntime rt,
) => f == null ? null : (RangeValues v) => rt.run(() => f(v.start, v.end));

/// goSpan reads the Span struct of the Go side.
///
/// The result is a TextSpan, thus a parameter of that type and a parameter of
/// InlineSpan each take it.
TextSpan? goSpan(GoNode? n) {
  if (n == null) return null;
  final List<GoNode> kids = n.structs('Children', 2);
  return TextSpan(
    text: n.text('Text', 0),
    style: goTextStyle(n.nested('Style', 1)),
    children: kids.isEmpty
        ? null
        : <InlineSpan>[for (final GoNode k in kids) goSpan(k)!],
  );
}

/// goSliverChildren reads the SliverChildren struct of the Go side.
SliverChildDelegate? goSliverChildren(GoNode? n, GoRuntime rt) {
  if (n == null) return null;
  final Iface? Function(int)? make = n.item('Builder', 1);
  if (make != null) {
    final int count = n.count('Count', 2);
    return SliverChildBuilderDelegate(
      (BuildContext _, int i) => goRender(make(i), rt),
      childCount: count == 0 ? null : count,
    );
  }
  return SliverChildListDelegate(goChildren(n.children('Children', 0), rt));
}

/// goGridSpec reads the GridSpec struct of the Go side.
SliverGridDelegate? goGridSpec(GoNode? n) {
  if (n == null) return null;
  final double ratio = n.size('AspectRatio', 5);
  final double main = n.size('MainSpacing', 3);
  final double cross = n.size('CrossSpacing', 4);
  final double? extent = n.sizeOpt('MainExtent', 6);
  if (n.text('Kind', 0) == 'extent') {
    final double max = n.size('MaxExtent', 2);
    return SliverGridDelegateWithMaxCrossAxisExtent(
      maxCrossAxisExtent: max <= 0 ? 100 : max,
      mainAxisSpacing: main,
      crossAxisSpacing: cross,
      childAspectRatio: ratio == 0 ? 1 : ratio,
      mainAxisExtent: extent,
    );
  }
  final int count = n.count('Count', 1);
  return SliverGridDelegateWithFixedCrossAxisCount(
    crossAxisCount: count < 1 ? 1 : count,
    mainAxisSpacing: main,
    crossAxisSpacing: cross,
    childAspectRatio: ratio == 0 ? 1 : ratio,
    mainAxisExtent: extent,
  );
}

/// goBarItem reads the BarItem struct of the Go side.
BottomNavigationBarItem? goBarItem(GoNode? n, GoRuntime rt) {
  if (n == null) return null;
  final Widget icon = goChildOpt(n.child('Icon', 0), rt) ?? const SizedBox();
  final String tip = n.text('Tooltip', 3);
  return BottomNavigationBarItem(
    icon: icon,
    label: n.text('Label', 1),
    activeIcon: goChildOpt(n.child('ActiveIcon', 2), rt) ?? icon,
    tooltip: tip.isEmpty ? null : tip,
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 4)),
  );
}

/// goBarItems reads a slice of the BarItem struct.
List<BottomNavigationBarItem> goBarItems(List<GoNode> list, GoRuntime rt) =>
    <BottomNavigationBarItem>[for (final GoNode n in list) goBarItem(n, rt)!];

/// goRailItem reads the RailItem struct of the Go side.
NavigationRailDestination? goRailItem(GoNode? n, GoRuntime rt) {
  if (n == null) return null;
  return NavigationRailDestination(
    icon: goChildOpt(n.child('Icon', 0), rt) ?? const SizedBox(),
    label: goChildOpt(n.child('Label', 1), rt) ?? const SizedBox(),
    selectedIcon: goChildOpt(n.child('SelectedIcon', 2), rt),
    padding: goInsets(n.nested('Padding', 3)),
    disabled: n.flag('Disabled', 4),
  );
}

/// goRailItems reads a slice of the RailItem struct.
List<NavigationRailDestination> goRailItems(List<GoNode> list, GoRuntime rt) =>
    <NavigationRailDestination>[
      for (final GoNode n in list) goRailItem(n, rt)!,
    ];

const Map<String, StepState> _stepStates = <String, StepState>{
  'indexed': StepState.indexed,
  'editing': StepState.editing,
  'complete': StepState.complete,
  'disabled': StepState.disabled,
  'error': StepState.error,
};

/// goStep reads the Step struct of the Go side.
Step? goStep(GoNode? n, GoRuntime rt) {
  if (n == null) return null;
  return Step(
    title: goChildOpt(n.child('Title', 0), rt) ?? const SizedBox(),
    content: goChildOpt(n.child('Content', 1), rt) ?? const SizedBox(),
    subtitle: goChildOpt(n.child('Subtitle', 2), rt),
    state: _stepStates[n.text('State', 3)] ?? StepState.indexed,
    isActive: n.flag('Active', 4),
  );
}

/// goSteps reads a slice of the Step struct.
List<Step> goSteps(List<GoNode> list, GoRuntime rt) => <Step>[
  for (final GoNode n in list) goStep(n, rt)!,
];

/// goColorScheme reads the ColorScheme struct of the Go side.
///
/// The Go side resolves each color that Flutter derives, thus a nil field
/// reaches this call as the derived color and no field stays null.
ColorScheme? goColorScheme(GoNode? n) {
  if (n == null) return null;
  Color at(String name, int i) => Color(n.count(name, i));
  Color? opt(String name, int i) => goColorOpt(n.countOpt(name, i));
  return ColorScheme(
    brightness: n.text('Brightness', 0) == 'dark'
        ? Brightness.dark
        : Brightness.light,
    primary: at('Primary', 1),
    onPrimary: at('OnPrimary', 2),
    secondary: at('Secondary', 3),
    onSecondary: at('OnSecondary', 4),
    error: at('Error', 5),
    onError: at('OnError', 6),
    surface: at('Surface', 7),
    onSurface: at('OnSurface', 8),
    primaryContainer: opt('PrimaryContainer', 9),
    onPrimaryContainer: opt('OnPrimaryContainer', 10),
    primaryFixed: opt('PrimaryFixed', 11),
    primaryFixedDim: opt('PrimaryFixedDim', 12),
    onPrimaryFixed: opt('OnPrimaryFixed', 13),
    onPrimaryFixedVariant: opt('OnPrimaryFixedVariant', 14),
    secondaryContainer: opt('SecondaryContainer', 15),
    onSecondaryContainer: opt('OnSecondaryContainer', 16),
    secondaryFixed: opt('SecondaryFixed', 17),
    secondaryFixedDim: opt('SecondaryFixedDim', 18),
    onSecondaryFixed: opt('OnSecondaryFixed', 19),
    onSecondaryFixedVariant: opt('OnSecondaryFixedVariant', 20),
    tertiary: opt('Tertiary', 21),
    onTertiary: opt('OnTertiary', 22),
    tertiaryContainer: opt('TertiaryContainer', 23),
    onTertiaryContainer: opt('OnTertiaryContainer', 24),
    tertiaryFixed: opt('TertiaryFixed', 25),
    tertiaryFixedDim: opt('TertiaryFixedDim', 26),
    onTertiaryFixed: opt('OnTertiaryFixed', 27),
    onTertiaryFixedVariant: opt('OnTertiaryFixedVariant', 28),
    errorContainer: opt('ErrorContainer', 29),
    onErrorContainer: opt('OnErrorContainer', 30),
    surfaceDim: opt('SurfaceDim', 31),
    surfaceBright: opt('SurfaceBright', 32),
    surfaceContainerLowest: opt('SurfaceContainerLowest', 33),
    surfaceContainerLow: opt('SurfaceContainerLow', 34),
    surfaceContainer: opt('SurfaceContainer', 35),
    surfaceContainerHigh: opt('SurfaceContainerHigh', 36),
    surfaceContainerHighest: opt('SurfaceContainerHighest', 37),
    onSurfaceVariant: opt('OnSurfaceVariant', 38),
    outline: opt('Outline', 39),
    outlineVariant: opt('OutlineVariant', 40),
    shadow: opt('Shadow', 41),
    scrim: opt('Scrim', 42),
    inverseSurface: opt('InverseSurface', 43),
    onInverseSurface: opt('OnInverseSurface', 44),
    inversePrimary: opt('InversePrimary', 45),
    surfaceTint: opt('SurfaceTint', 46),
  );
}

/// goTextTheme reads the TextTheme struct of the Go side.
TextTheme? goTextTheme(GoNode? n) {
  if (n == null) return null;
  return TextTheme(
    displayLarge: goTextStyle(n.nested('DisplayLarge', 0)),
    displayMedium: goTextStyle(n.nested('DisplayMedium', 1)),
    displaySmall: goTextStyle(n.nested('DisplaySmall', 2)),
    headlineLarge: goTextStyle(n.nested('HeadlineLarge', 3)),
    headlineMedium: goTextStyle(n.nested('HeadlineMedium', 4)),
    headlineSmall: goTextStyle(n.nested('HeadlineSmall', 5)),
    titleLarge: goTextStyle(n.nested('TitleLarge', 6)),
    titleMedium: goTextStyle(n.nested('TitleMedium', 7)),
    titleSmall: goTextStyle(n.nested('TitleSmall', 8)),
    bodyLarge: goTextStyle(n.nested('BodyLarge', 9)),
    bodyMedium: goTextStyle(n.nested('BodyMedium', 10)),
    bodySmall: goTextStyle(n.nested('BodySmall', 11)),
    labelLarge: goTextStyle(n.nested('LabelLarge', 12)),
    labelMedium: goTextStyle(n.nested('LabelMedium', 13)),
    labelSmall: goTextStyle(n.nested('LabelSmall', 14)),
  );
}

/// goThemeStyle reads the ThemeStyle struct of the Go side.
ThemeData? goThemeStyle(GoNode? n) {
  if (n == null) return null;
  final String family = n.text('FontFamily', 4);
  final ColorScheme? scheme = goColorScheme(n.nested('ColorScheme', 7));
  return ThemeData(
    brightness: scheme != null
        ? null
        : (n.text('Brightness', 0) == 'dark'
              ? Brightness.dark
              : (n.text('Brightness', 0) == 'light' ? Brightness.light : null)),
    colorScheme: scheme,
    colorSchemeSeed: scheme != null ? null : goColorOpt(n.countOpt('Seed', 1)),
    primaryColor: goColorOpt(n.countOpt('PrimaryColor', 2)),
    scaffoldBackgroundColor: goColorOpt(n.countOpt('ScaffoldBackground', 3)),
    fontFamily: family.isEmpty ? null : family,
    visualDensity: goDensity(n.nested('Density', 5)),
    iconTheme: goIconStyle(n.nested('IconStyle', 6)),
    textTheme: goTextTheme(n.nested('TextTheme', 8)),
    inputDecorationTheme: goInputDecorationStyle(
      n.nested('InputDecoration', 9),
    ),
  );
}

/// goTask wraps a Go function that the host waits for.
///
/// The Go side runs as a goroutine, thus it may block. The handle of that
/// goroutine ends the Future, thus the Go side reports nothing itself. Read
/// the Task type of flutter/ui/node.go.
Future<void> Function()? goTask(void Function()? f, GoRuntime rt) {
  if (f == null) return null;
  return () {
    final Completer<void> done = Completer<void>();
    rt.run(f).whenEnd(done.complete);
    return done.future;
  };
}

/// goAskEnum wraps a Go function that the host waits for, and that takes the
/// name of a value of a Flutter enum.
///
/// The confirmDismiss of a Dismissible is such a function. The result of the
/// goroutine holds the answer, and a panic gives false.
Future<bool?> Function(T)? goAskEnum<T extends Enum>(
  bool Function(GoString)? f,
  GoRuntime rt,
) {
  if (f == null) return null;
  return (T value) {
    final Completer<bool> done = Completer<bool>();
    final GoString name = GoString.fromString(value.name);
    final Goroutine g = rt.run(() => f(name));
    g.whenEnd(() => done.complete(g.result == true));
    return done.future;
  };
}

/// goAsk wraps a Go function that the host waits for, and that gives an
/// answer.
///
/// The result of the goroutine holds the answer. A panic and runtime.Goexit
/// each leave no result, and the answer is then false.
Future<bool> Function()? goAsk(bool Function()? f, GoRuntime rt) {
  if (f == null) return null;
  return () {
    final Completer<bool> done = Completer<bool>();
    final Goroutine g = rt.run(f);
    g.whenEnd(() => done.complete(g.result == true));
    return done.future;
  };
}

/// goTurnBuilder wraps a builder that takes the orientation of the parent.
Widget Function(BuildContext, Orientation)? goTurnBuilder(
  Iface? Function(GoString)? f,
  GoRuntime rt,
) => f == null
    ? null
    : (BuildContext _, Orientation o) =>
          goRender(f(GoString.fromString(o.name)), rt);

/// goBoxBuilder wraps a builder that takes the limits of the box. A maximum
/// with no limit gives 0 to the Go side, as in ui.Constraints.
Widget Function(BuildContext, BoxConstraints)? goBoxBuilder(
  Iface? Function(double, double, double, double)? f,
  GoRuntime rt,
) => f == null
    ? null
    : (BuildContext _, BoxConstraints c) => goRender(
        f(
          c.minWidth,
          c.maxWidth.isFinite ? c.maxWidth : 0,
          c.minHeight,
          c.maxHeight.isFinite ? c.maxHeight : 0,
        ),
        rt,
      );

/// goSliverBuilder wraps a builder that gives the slivers of a header.
List<Widget> Function(BuildContext, bool)? goSliverBuilder(
  Slice<Iface?> Function(bool)? f,
  GoRuntime rt,
) => f == null
    ? null
    : (BuildContext _, bool scrolled) {
        final Slice<Iface?> out = f(scrolled);
        return <Widget>[for (int i = 0; i < out.len; i++) goRender(out[i], rt)];
      };

/// goStateColorOn reads the StateColor struct for a parameter that demands 1
/// color for each state.
///
/// A state with no color of its own and no Default gives a transparent color,
/// because such a parameter takes no null.
WidgetStateProperty<Color>? goStateColorOn(GoNode? n) {
  final WidgetStateProperty<Color?>? p = goStateColor(n);
  if (p == null) return null;
  return WidgetStateProperty.resolveWith<Color>(
    (Set<WidgetState> states) => p.resolve(states) ?? const Color(0x00000000),
  );
}

const Map<String, ContextMenuButtonType> _buttonTypes =
    <String, ContextMenuButtonType>{
      'cut': ContextMenuButtonType.cut,
      'copy': ContextMenuButtonType.copy,
      'paste': ContextMenuButtonType.paste,
      'selectAll': ContextMenuButtonType.selectAll,
      'delete': ContextMenuButtonType.delete,
      'lookUp': ContextMenuButtonType.lookUp,
      'searchWeb': ContextMenuButtonType.searchWeb,
      'share': ContextMenuButtonType.share,
      'liveTextInput': ContextMenuButtonType.liveTextInput,
      'custom': ContextMenuButtonType.custom,
    };

/// goMenuItem reads the MenuButtonItem struct of the Go side.
ContextMenuButtonItem? goMenuItem(GoNode? n, GoRuntime rt) {
  if (n == null) return null;
  final String label = n.text('Label', 0);
  return ContextMenuButtonItem(
    label: label.isEmpty ? null : label,
    type: _buttonTypes[n.text('Kind', 1)] ?? ContextMenuButtonType.custom,
    onPressed: goAction(n.action('OnPressed', 2), rt),
  );
}

/// goMenuItems reads a slice of the MenuButtonItem struct.
List<ContextMenuButtonItem> goMenuItems(List<GoNode> list, GoRuntime rt) =>
    <ContextMenuButtonItem>[for (final GoNode n in list) goMenuItem(n, rt)!];

/// goMatrix reads a matrix of 16 numbers, in the order of the columns.
Matrix4? goMatrix(List<double> v) =>
    v.length == 16 ? Matrix4.fromList(v) : null;

/// goColumns reads a slice of the DataColumn struct of the Go side.
List<DataColumn> goColumns(List<GoNode> list, GoRuntime rt) => <DataColumn>[
  for (final GoNode n in list)
    DataColumn(
      label: goChildOpt(n.child('Label', 0), rt) ?? const SizedBox(),
      tooltip: n.text('Tooltip', 1).isEmpty ? null : n.text('Tooltip', 1),
      numeric: n.flag('Numeric', 2),
      onSort: goOnSort(n.onSort('OnSort', 3), rt),
    ),
];

/// goOnSort wraps the callback of the header of a column.
void Function(int, bool)? goOnSort(void Function(int, bool)? f, GoRuntime rt) =>
    f == null ? null : (int i, bool up) => rt.run(() => f(i, up));

/// goRows reads a slice of the DataRow struct of the Go side.
List<DataRow> goRows(List<GoNode> list, GoRuntime rt) => <DataRow>[
  for (final GoNode n in list)
    DataRow(
      cells: <DataCell>[
        for (final GoNode c in n.structs('Cells', 0))
          DataCell(
            goChildOpt(c.child('Child', 0), rt) ?? const SizedBox(),
            placeholder: c.flag('Placeholder', 1),
            showEditIcon: c.flag('ShowEditIcon', 2),
            onTap: goAction(c.action('OnTap', 3), rt),
          ),
      ],
      selected: n.flag('Selected', 1),
      onSelectChanged: goOnBool(n.onFlag('OnSelect', 2), rt),
      color: goStateColor(n.nested('Color', 3)),
    ),
];

// The converters below build the theme of 1 widget family. Each 1 reads the
// value type of flutter/ui/themes.go, and it gives the Flutter theme of the
// same meaning.

/// goEnumValue gives the value of an enum by its name. The Go side holds the
/// name of the Dart value, and an empty text gives null.
T? goEnumValue<T extends Enum>(List<T> values, String s) {
  for (final T v in values) {
    if (v.name == s) return v;
  }
  return null;
}

/// goCircular gives a circular radius. An elliptical radius stays out.
Radius? goCircular(double? v) => v == null ? null : Radius.circular(v);

/// goActionIconStyle reads the ActionIconStyle struct of the Go side.
ActionIconThemeData? goActionIconStyle(GoNode? n, GoRuntime rt) {
  if (n == null) return null;
  return ActionIconThemeData(
    backButtonIconBuilder: goBuild(n.build0('BackButtonIconBuilder', 0), rt),
    closeButtonIconBuilder: goBuild(n.build0('CloseButtonIconBuilder', 1), rt),
    drawerButtonIconBuilder: goBuild(
      n.build0('DrawerButtonIconBuilder', 2),
      rt,
    ),
    endDrawerButtonIconBuilder: goBuild(
      n.build0('EndDrawerButtonIconBuilder', 3),
      rt,
    ),
  );
}

/// goBadgeStyle reads the BadgeStyle struct of the Go side.
BadgeThemeData? goBadgeStyle(GoNode? n) {
  if (n == null) return null;
  return BadgeThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    textColor: goColorOpt(n.countOpt('TextColor', 1)),
    smallSize: n.sizeOpt('SmallSize', 2),
    largeSize: n.sizeOpt('LargeSize', 3),
    textStyle: goTextStyle(n.nested('TextStyle', 4)),
    padding: goInsets(n.nested('Padding', 5)),
    alignment: goAlignment(n.text('Alignment', 6)),
    offset: goOffset(n.nested('Offset', 7)),
  );
}

/// goBottomNavigationBarStyle reads the BottomNavigationBarStyle struct of the
/// Go side.
BottomNavigationBarThemeData? goBottomNavigationBarStyle(GoNode? n) {
  if (n == null) return null;
  return BottomNavigationBarThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    elevation: n.sizeOpt('Elevation', 1),
    selectedIconTheme: goIconStyle(n.nested('SelectedIconTheme', 2)),
    unselectedIconTheme: goIconStyle(n.nested('UnselectedIconTheme', 3)),
    selectedItemColor: goColorOpt(n.countOpt('SelectedItemColor', 4)),
    unselectedItemColor: goColorOpt(n.countOpt('UnselectedItemColor', 5)),
    selectedLabelStyle: goTextStyle(n.nested('SelectedLabelStyle', 6)),
    unselectedLabelStyle: goTextStyle(n.nested('UnselectedLabelStyle', 7)),
    showSelectedLabels: n.flagOpt('ShowSelectedLabels', 8),
    showUnselectedLabels: n.flagOpt('ShowUnselectedLabels', 9),
    type: goEnumValue<BottomNavigationBarType>(
      BottomNavigationBarType.values,
      n.text('Type', 10),
    ),
    enableFeedback: n.flagOpt('EnableFeedback', 11),
    landscapeLayout: goEnumValue<BottomNavigationBarLandscapeLayout>(
      BottomNavigationBarLandscapeLayout.values,
      n.text('LandscapeLayout', 12),
    ),
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 13))),
  );
}

/// goButtonBarStyle reads the ButtonBarStyle struct of the Go side.
///
/// Flutter holds ButtonBar as deprecated, thus the 2 lines below stand under
/// an ignore.
// ignore: deprecated_member_use
ButtonBarThemeData? goButtonBarStyle(GoNode? n) {
  if (n == null) return null;
  // ignore: deprecated_member_use
  return ButtonBarThemeData(
    alignment: goEnumValue<MainAxisAlignment>(
      MainAxisAlignment.values,
      n.text('Alignment', 0),
    ),
    mainAxisSize: goEnumValue<MainAxisSize>(
      MainAxisSize.values,
      n.text('MainAxisSize', 1),
    ),
    buttonTextTheme: goEnumValue<ButtonTextTheme>(
      ButtonTextTheme.values,
      n.text('ButtonTextTheme', 2),
    ),
    buttonMinWidth: n.sizeOpt('ButtonMinWidth', 3),
    buttonHeight: n.sizeOpt('ButtonHeight', 4),
    buttonPadding: goInsets(n.nested('ButtonPadding', 5)),
    buttonAlignedDropdown: n.flagOpt('ButtonAlignedDropdown', 6),
    layoutBehavior: goEnumValue<ButtonBarLayoutBehavior>(
      ButtonBarLayoutBehavior.values,
      n.text('LayoutBehavior', 7),
    ),
    overflowDirection: goEnumValue<VerticalDirection>(
      VerticalDirection.values,
      n.text('OverflowDirection', 8),
    ),
  );
}

/// goButtonThemeStyle reads the ButtonThemeStyle struct of the Go side.
ButtonThemeData? goButtonThemeStyle(GoNode? n) {
  if (n == null) return null;
  return const ButtonThemeData().copyWith(
    textTheme: goEnumValue<ButtonTextTheme>(
      ButtonTextTheme.values,
      n.text('TextTheme', 0),
    ),
    minWidth: n.sizeOpt('MinWidth', 1),
    height: n.sizeOpt('Height', 2),
    padding: goInsets(n.nested('Padding', 3)),
    shape: goShape(n.nested('Shape', 4)),
    layoutBehavior: goEnumValue<ButtonBarLayoutBehavior>(
      ButtonBarLayoutBehavior.values,
      n.text('LayoutBehavior', 5),
    ),
    alignedDropdown: n.flagOpt('AlignedDropdown', 6),
    buttonColor: goColorOpt(n.countOpt('ButtonColor', 7)),
    disabledColor: goColorOpt(n.countOpt('DisabledColor', 8)),
    focusColor: goColorOpt(n.countOpt('FocusColor', 9)),
    hoverColor: goColorOpt(n.countOpt('HoverColor', 10)),
    highlightColor: goColorOpt(n.countOpt('HighlightColor', 11)),
    splashColor: goColorOpt(n.countOpt('SplashColor', 12)),
    materialTapTargetSize: goEnumValue<MaterialTapTargetSize>(
      MaterialTapTargetSize.values,
      n.text('MaterialTapTargetSize', 13),
    ),
  );
}

/// goCarouselViewStyle reads the CarouselViewStyle struct of the Go side.
CarouselViewThemeData? goCarouselViewStyle(GoNode? n) {
  if (n == null) return null;
  return CarouselViewThemeData(
    elevation: n.sizeOpt('Elevation', 0),
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 1)),
    overlayColor: goStateColor(n.nested('OverlayColor', 2)),
    shape: goShape(n.nested('Shape', 3)),
    padding: goInsets(n.nested('Padding', 4)),
    itemClipBehavior: goEnumValue<Clip>(
      Clip.values,
      n.text('ItemClipBehavior', 5),
    ),
  );
}

/// goCheckboxStyle reads the CheckboxStyle struct of the Go side.
CheckboxThemeData? goCheckboxStyle(GoNode? n) {
  if (n == null) return null;
  return CheckboxThemeData(
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 0))),
    fillColor: goStateColor(n.nested('FillColor', 1)),
    checkColor: goStateColor(n.nested('CheckColor', 2)),
    overlayColor: goStateColor(n.nested('OverlayColor', 3)),
    splashRadius: n.sizeOpt('SplashRadius', 4),
    materialTapTargetSize: goEnumValue<MaterialTapTargetSize>(
      MaterialTapTargetSize.values,
      n.text('MaterialTapTargetSize', 5),
    ),
    visualDensity: goDensity(n.nested('VisualDensity', 6)),
    shape: goShape(n.nested('Shape', 7)),
    side: goBorderSide(n.nested('Side', 8)),
  );
}

/// goChipStyle reads the ChipStyle struct of the Go side.
ChipThemeData? goChipStyle(GoNode? n) {
  if (n == null) return null;
  return ChipThemeData(
    color: goStateColor(n.nested('Color', 0)),
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 1)),
    deleteIconColor: goColorOpt(n.countOpt('DeleteIconColor', 2)),
    disabledColor: goColorOpt(n.countOpt('DisabledColor', 3)),
    selectedColor: goColorOpt(n.countOpt('SelectedColor', 4)),
    secondarySelectedColor: goColorOpt(n.countOpt('SecondarySelectedColor', 5)),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 6)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 7)),
    selectedShadowColor: goColorOpt(n.countOpt('SelectedShadowColor', 8)),
    showCheckmark: n.flagOpt('ShowCheckmark', 9),
    checkmarkColor: goColorOpt(n.countOpt('CheckmarkColor', 10)),
    labelPadding: goInsets(n.nested('LabelPadding', 11)),
    padding: goInsets(n.nested('Padding', 12)),
    side: goBorderSide(n.nested('Side', 13)),
    shape: goShape(n.nested('Shape', 14)),
    labelStyle: goTextStyle(n.nested('LabelStyle', 15)),
    secondaryLabelStyle: goTextStyle(n.nested('SecondaryLabelStyle', 16)),
    brightness: goEnumValue<Brightness>(
      Brightness.values,
      n.text('Brightness', 17),
    ),
    elevation: n.sizeOpt('Elevation', 18),
    pressElevation: n.sizeOpt('PressElevation', 19),
    iconTheme: goIconStyle(n.nested('IconTheme', 20)),
    avatarBoxConstraints: goConstraints(n.nested('AvatarBoxConstraints', 21)),
    deleteIconBoxConstraints: goConstraints(
      n.nested('DeleteIconBoxConstraints', 22),
    ),
  );
}

/// goCupertinoStyle reads the CupertinoStyle struct of the Go side.
CupertinoThemeData? goCupertinoStyle(GoNode? n) {
  if (n == null) return null;
  return CupertinoThemeData(
    brightness: goEnumValue<Brightness>(
      Brightness.values,
      n.text('Brightness', 0),
    ),
    primaryColor: goColorOpt(n.countOpt('PrimaryColor', 1)),
    primaryContrastingColor: goColorOpt(
      n.countOpt('PrimaryContrastingColor', 2),
    ),
    barBackgroundColor: goColorOpt(n.countOpt('BarBackgroundColor', 3)),
    scaffoldBackgroundColor: goColorOpt(
      n.countOpt('ScaffoldBackgroundColor', 4),
    ),
    selectionHandleColor: goColorOpt(n.countOpt('SelectionHandleColor', 5)),
    applyThemeToAll: n.flagOpt('ApplyThemeToAll', 6),
  );
}

/// goDataTableStyle reads the DataTableStyle struct of the Go side.
DataTableThemeData? goDataTableStyle(GoNode? n) {
  if (n == null) return null;
  return DataTableThemeData(
    decoration: goDecoration(n.nested('Decoration', 0)),
    dataRowColor: goStateColor(n.nested('DataRowColor', 1)),
    dataRowMinHeight: n.sizeOpt('DataRowMinHeight', 2),
    dataRowMaxHeight: n.sizeOpt('DataRowMaxHeight', 3),
    dataTextStyle: goTextStyle(n.nested('DataTextStyle', 4)),
    headingRowColor: goStateColor(n.nested('HeadingRowColor', 5)),
    headingRowHeight: n.sizeOpt('HeadingRowHeight', 6),
    headingTextStyle: goTextStyle(n.nested('HeadingTextStyle', 7)),
    horizontalMargin: n.sizeOpt('HorizontalMargin', 8),
    columnSpacing: n.sizeOpt('ColumnSpacing', 9),
    dividerThickness: n.sizeOpt('DividerThickness', 10),
    checkboxHorizontalMargin: n.sizeOpt('CheckboxHorizontalMargin', 11),
    headingCellCursor: _all<MouseCursor>(
      goCursor(n.text('HeadingCellCursor', 12)),
    ),
    dataRowCursor: _all<MouseCursor>(goCursor(n.text('DataRowCursor', 13))),
    headingRowAlignment: goEnumValue<MainAxisAlignment>(
      MainAxisAlignment.values,
      n.text('HeadingRowAlignment', 14),
    ),
  );
}

/// goDatePickerStyle reads the DatePickerStyle struct of the Go side.
DatePickerThemeData? goDatePickerStyle(GoNode? n) {
  if (n == null) return null;
  return DatePickerThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    elevation: n.sizeOpt('Elevation', 1),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 2)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 3)),
    shape: goShape(n.nested('Shape', 4)),
    headerBackgroundColor: goColorOpt(n.countOpt('HeaderBackgroundColor', 5)),
    headerForegroundColor: goColorOpt(n.countOpt('HeaderForegroundColor', 6)),
    headerHeadlineStyle: goTextStyle(n.nested('HeaderHeadlineStyle', 7)),
    headerHelpStyle: goTextStyle(n.nested('HeaderHelpStyle', 8)),
    weekdayStyle: goTextStyle(n.nested('WeekdayStyle', 9)),
    dayStyle: goTextStyle(n.nested('DayStyle', 10)),
    dayForegroundColor: goStateColor(n.nested('DayForegroundColor', 11)),
    dayBackgroundColor: goStateColor(n.nested('DayBackgroundColor', 12)),
    dayOverlayColor: goStateColor(n.nested('DayOverlayColor', 13)),
    dayShape: _all<OutlinedBorder>(goShape(n.nested('DayShape', 14))),
    todayForegroundColor: goStateColor(n.nested('TodayForegroundColor', 15)),
    todayBackgroundColor: goStateColor(n.nested('TodayBackgroundColor', 16)),
    todayBorder: goBorderSide(n.nested('TodayBorder', 17)),
    yearStyle: goTextStyle(n.nested('YearStyle', 18)),
    yearForegroundColor: goStateColor(n.nested('YearForegroundColor', 19)),
    yearBackgroundColor: goStateColor(n.nested('YearBackgroundColor', 20)),
    yearOverlayColor: goStateColor(n.nested('YearOverlayColor', 21)),
    yearShape: _all<OutlinedBorder>(goShape(n.nested('YearShape', 22))),
    rangePickerBackgroundColor: goColorOpt(
      n.countOpt('RangePickerBackgroundColor', 23),
    ),
    rangePickerElevation: n.sizeOpt('RangePickerElevation', 24),
    rangePickerShadowColor: goColorOpt(
      n.countOpt('RangePickerShadowColor', 25),
    ),
    rangePickerSurfaceTintColor: goColorOpt(
      n.countOpt('RangePickerSurfaceTintColor', 26),
    ),
    rangePickerShape: goShape(n.nested('RangePickerShape', 27)),
    rangePickerHeaderBackgroundColor: goColorOpt(
      n.countOpt('RangePickerHeaderBackgroundColor', 28),
    ),
    rangePickerHeaderForegroundColor: goColorOpt(
      n.countOpt('RangePickerHeaderForegroundColor', 29),
    ),
    rangePickerHeaderHeadlineStyle: goTextStyle(
      n.nested('RangePickerHeaderHeadlineStyle', 30),
    ),
    rangePickerHeaderHelpStyle: goTextStyle(
      n.nested('RangePickerHeaderHelpStyle', 31),
    ),
    rangeSelectionBackgroundColor: goColorOpt(
      n.countOpt('RangeSelectionBackgroundColor', 32),
    ),
    rangeSelectionOverlayColor: goStateColor(
      n.nested('RangeSelectionOverlayColor', 33),
    ),
    dividerColor: goColorOpt(n.countOpt('DividerColor', 34)),
    cancelButtonStyle: goButtonStyle(n.nested('CancelButtonStyle', 35)),
    confirmButtonStyle: goButtonStyle(n.nested('ConfirmButtonStyle', 36)),
    locale: goLocale(n.text('Locale', 37)),
    toggleButtonTextStyle: goTextStyle(n.nested('ToggleButtonTextStyle', 38)),
    subHeaderForegroundColor: goColorOpt(
      n.countOpt('SubHeaderForegroundColor', 39),
    ),
  );
}

/// goDividerStyle reads the DividerStyle struct of the Go side.
DividerThemeData? goDividerStyle(GoNode? n) {
  if (n == null) return null;
  return DividerThemeData(
    color: goColorOpt(n.countOpt('Color', 0)),
    space: n.sizeOpt('Space', 1),
    thickness: n.sizeOpt('Thickness', 2),
    indent: n.sizeOpt('Indent', 3),
    endIndent: n.sizeOpt('EndIndent', 4),
    radius: goRadius(n.nested('Radius', 5)),
  );
}

/// goDrawerStyle reads the DrawerStyle struct of the Go side.
DrawerThemeData? goDrawerStyle(GoNode? n) {
  if (n == null) return null;
  return DrawerThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    scrimColor: goColorOpt(n.countOpt('ScrimColor', 1)),
    elevation: n.sizeOpt('Elevation', 2),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 3)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 4)),
    shape: goShape(n.nested('Shape', 5)),
    endShape: goShape(n.nested('EndShape', 6)),
    width: n.sizeOpt('Width', 7),
    clipBehavior: goEnumValue<Clip>(Clip.values, n.text('ClipBehavior', 8)),
  );
}

/// goDropdownMenuStyle reads the DropdownMenuStyle struct of the Go side.
DropdownMenuThemeData? goDropdownMenuStyle(GoNode? n) {
  if (n == null) return null;
  return DropdownMenuThemeData(
    textStyle: goTextStyle(n.nested('TextStyle', 0)),
    menuStyle: goMenuStyle(n.nested('MenuStyle', 1)),
    disabledColor: goColorOpt(n.countOpt('DisabledColor', 2)),
  );
}

/// goExpansionTileStyle reads the ExpansionTileStyle struct of the Go side.
ExpansionTileThemeData? goExpansionTileStyle(GoNode? n) {
  if (n == null) return null;
  return ExpansionTileThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    collapsedBackgroundColor: goColorOpt(
      n.countOpt('CollapsedBackgroundColor', 1),
    ),
    tilePadding: goInsets(n.nested('TilePadding', 2)),
    expandedAlignment: goAlignment(n.text('ExpandedAlignment', 3)),
    childrenPadding: goInsets(n.nested('ChildrenPadding', 4)),
    iconColor: goColorOpt(n.countOpt('IconColor', 5)),
    collapsedIconColor: goColorOpt(n.countOpt('CollapsedIconColor', 6)),
    textColor: goColorOpt(n.countOpt('TextColor', 7)),
    collapsedTextColor: goColorOpt(n.countOpt('CollapsedTextColor', 8)),
    shape: goShape(n.nested('Shape', 9)),
    collapsedShape: goShape(n.nested('CollapsedShape', 10)),
    clipBehavior: goEnumValue<Clip>(Clip.values, n.text('ClipBehavior', 11)),
  );
}

/// goFloatingActionButtonStyle reads the FloatingActionButtonStyle struct of
/// the Go side.
FloatingActionButtonThemeData? goFloatingActionButtonStyle(GoNode? n) {
  if (n == null) return null;
  return FloatingActionButtonThemeData(
    foregroundColor: goColorOpt(n.countOpt('ForegroundColor', 0)),
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 1)),
    focusColor: goColorOpt(n.countOpt('FocusColor', 2)),
    hoverColor: goColorOpt(n.countOpt('HoverColor', 3)),
    splashColor: goColorOpt(n.countOpt('SplashColor', 4)),
    elevation: n.sizeOpt('Elevation', 5),
    focusElevation: n.sizeOpt('FocusElevation', 6),
    hoverElevation: n.sizeOpt('HoverElevation', 7),
    disabledElevation: n.sizeOpt('DisabledElevation', 8),
    highlightElevation: n.sizeOpt('HighlightElevation', 9),
    shape: goShape(n.nested('Shape', 10)),
    enableFeedback: n.flagOpt('EnableFeedback', 11),
    iconSize: n.sizeOpt('IconSize', 12),
    sizeConstraints: goConstraints(n.nested('SizeConstraints', 13)),
    smallSizeConstraints: goConstraints(n.nested('SmallSizeConstraints', 14)),
    largeSizeConstraints: goConstraints(n.nested('LargeSizeConstraints', 15)),
    extendedSizeConstraints: goConstraints(
      n.nested('ExtendedSizeConstraints', 16),
    ),
    extendedIconLabelSpacing: n.sizeOpt('ExtendedIconLabelSpacing', 17),
    extendedPadding: goInsets(n.nested('ExtendedPadding', 18)),
    extendedTextStyle: goTextStyle(n.nested('ExtendedTextStyle', 19)),
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 20))),
  );
}

/// goMenuStyle reads the MenuStyle struct of the Go side.
MenuStyle? goMenuStyle(GoNode? n) {
  if (n == null) return null;
  return MenuStyle(
    backgroundColor: goStateColor(n.nested('BackgroundColor', 0)),
    shadowColor: goStateColor(n.nested('ShadowColor', 1)),
    surfaceTintColor: goStateColor(n.nested('SurfaceTintColor', 2)),
    elevation: _all<double>(n.sizeOpt('Elevation', 3)),
    padding: _all<EdgeInsetsGeometry>(goInsets(n.nested('Padding', 4))),
    minimumSize: _all<Size>(goSize(n.nested('MinimumSize', 5))),
    fixedSize: _all<Size>(goSize(n.nested('FixedSize', 6))),
    maximumSize: _all<Size>(goSize(n.nested('MaximumSize', 7))),
    side: _all<BorderSide>(goBorderSide(n.nested('Side', 8))),
    shape: _all<OutlinedBorder>(goShape(n.nested('Shape', 9))),
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 10))),
    visualDensity: goDensity(n.nested('VisualDensity', 11)),
    alignment: goAlignment(n.text('Alignment', 12)),
  );
}

/// goNavigationBarStyle reads the NavigationBarStyle struct of the Go side.
NavigationBarThemeData? goNavigationBarStyle(GoNode? n) {
  if (n == null) return null;
  return NavigationBarThemeData(
    height: n.sizeOpt('Height', 0),
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 1)),
    elevation: n.sizeOpt('Elevation', 2),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 3)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 4)),
    indicatorColor: goColorOpt(n.countOpt('IndicatorColor', 5)),
    indicatorShape: goShape(n.nested('IndicatorShape', 6)),
    labelTextStyle: _all<TextStyle>(goTextStyle(n.nested('LabelTextStyle', 7))),
    iconTheme: _all<IconThemeData>(goIconStyle(n.nested('IconTheme', 8))),
    labelBehavior: goEnumValue<NavigationDestinationLabelBehavior>(
      NavigationDestinationLabelBehavior.values,
      n.text('LabelBehavior', 9),
    ),
    overlayColor: goStateColor(n.nested('OverlayColor', 10)),
    labelPadding: goInsets(n.nested('LabelPadding', 11)),
  );
}

/// goNavigationDrawerStyle reads the NavigationDrawerStyle struct of the Go
/// side.
NavigationDrawerThemeData? goNavigationDrawerStyle(GoNode? n) {
  if (n == null) return null;
  return NavigationDrawerThemeData(
    tileHeight: n.sizeOpt('TileHeight', 0),
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 1)),
    elevation: n.sizeOpt('Elevation', 2),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 3)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 4)),
    indicatorColor: goColorOpt(n.countOpt('IndicatorColor', 5)),
    indicatorShape: goShape(n.nested('IndicatorShape', 6)),
    indicatorSize: goSize(n.nested('IndicatorSize', 7)),
    labelTextStyle: _all<TextStyle>(goTextStyle(n.nested('LabelTextStyle', 8))),
    iconTheme: _all<IconThemeData>(goIconStyle(n.nested('IconTheme', 9))),
  );
}

/// goNavigationRailStyle reads the NavigationRailStyle struct of the Go side.
NavigationRailThemeData? goNavigationRailStyle(GoNode? n) {
  if (n == null) return null;
  return NavigationRailThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    elevation: n.sizeOpt('Elevation', 1),
    unselectedLabelTextStyle: goTextStyle(
      n.nested('UnselectedLabelTextStyle', 2),
    ),
    selectedLabelTextStyle: goTextStyle(n.nested('SelectedLabelTextStyle', 3)),
    unselectedIconTheme: goIconStyle(n.nested('UnselectedIconTheme', 4)),
    selectedIconTheme: goIconStyle(n.nested('SelectedIconTheme', 5)),
    groupAlignment: n.sizeOpt('GroupAlignment', 6),
    labelType: goEnumValue<NavigationRailLabelType>(
      NavigationRailLabelType.values,
      n.text('LabelType', 7),
    ),
    useIndicator: n.flagOpt('UseIndicator', 8),
    indicatorColor: goColorOpt(n.countOpt('IndicatorColor', 9)),
    indicatorShape: goShape(n.nested('IndicatorShape', 10)),
    minWidth: n.sizeOpt('MinWidth', 11),
    minExtendedWidth: n.sizeOpt('MinExtendedWidth', 12),
  );
}

/// goPopupMenuStyle reads the PopupMenuStyle struct of the Go side.
PopupMenuThemeData? goPopupMenuStyle(GoNode? n) {
  if (n == null) return null;
  return PopupMenuThemeData(
    color: goColorOpt(n.countOpt('Color', 0)),
    shape: goShape(n.nested('Shape', 1)),
    menuPadding: goInsets(n.nested('MenuPadding', 2)),
    elevation: n.sizeOpt('Elevation', 3),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 4)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 5)),
    textStyle: goTextStyle(n.nested('TextStyle', 6)),
    labelTextStyle: _all<TextStyle>(goTextStyle(n.nested('LabelTextStyle', 7))),
    enableFeedback: n.flagOpt('EnableFeedback', 8),
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 9))),
    position: goEnumValue<PopupMenuPosition>(
      PopupMenuPosition.values,
      n.text('Position', 10),
    ),
    iconColor: goColorOpt(n.countOpt('IconColor', 11)),
    iconSize: n.sizeOpt('IconSize', 12),
  );
}

/// goProgressIndicatorStyle reads the ProgressIndicatorStyle struct of the Go
/// side.
ProgressIndicatorThemeData? goProgressIndicatorStyle(GoNode? n) {
  if (n == null) return null;
  return ProgressIndicatorThemeData(
    color: goColorOpt(n.countOpt('Color', 0)),
    linearTrackColor: goColorOpt(n.countOpt('LinearTrackColor', 1)),
    linearMinHeight: n.sizeOpt('LinearMinHeight', 2),
    circularTrackColor: goColorOpt(n.countOpt('CircularTrackColor', 3)),
    refreshBackgroundColor: goColorOpt(n.countOpt('RefreshBackgroundColor', 4)),
    borderRadius: goRadius(n.nested('BorderRadius', 5)),
    stopIndicatorColor: goColorOpt(n.countOpt('StopIndicatorColor', 6)),
    stopIndicatorRadius: n.sizeOpt('StopIndicatorRadius', 7),
    strokeWidth: n.sizeOpt('StrokeWidth', 8),
    strokeAlign: n.sizeOpt('StrokeAlign', 9),
    strokeCap: goEnumValue<StrokeCap>(
      StrokeCap.values,
      n.text('StrokeCap', 10),
    ),
    constraints: goConstraints(n.nested('Constraints', 11)),
    trackGap: n.sizeOpt('TrackGap', 12),
    circularTrackPadding: goInsets(n.nested('CircularTrackPadding', 13)),
  );
}

/// goRadioStyle reads the RadioStyle struct of the Go side.
RadioThemeData? goRadioStyle(GoNode? n) {
  if (n == null) return null;
  return RadioThemeData(
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 0))),
    fillColor: goStateColor(n.nested('FillColor', 1)),
    overlayColor: goStateColor(n.nested('OverlayColor', 2)),
    splashRadius: n.sizeOpt('SplashRadius', 3),
    materialTapTargetSize: goEnumValue<MaterialTapTargetSize>(
      MaterialTapTargetSize.values,
      n.text('MaterialTapTargetSize', 4),
    ),
    visualDensity: goDensity(n.nested('VisualDensity', 5)),
    backgroundColor: goStateColor(n.nested('BackgroundColor', 6)),
    side: goBorderSide(n.nested('Side', 7)),
    innerRadius: _all<double>(n.sizeOpt('InnerRadius', 8)),
  );
}

/// goScrollbarStyle reads the ScrollbarStyle struct of the Go side.
ScrollbarThemeData? goScrollbarStyle(GoNode? n) {
  if (n == null) return null;
  return ScrollbarThemeData(
    thumbVisibility: _all<bool>(n.flagOpt('ThumbVisibility', 0)),
    thickness: _all<double>(n.sizeOpt('Thickness', 1)),
    trackVisibility: _all<bool>(n.flagOpt('TrackVisibility', 2)),
    radius: goCircular(n.sizeOpt('Radius', 3)),
    thumbColor: goStateColor(n.nested('ThumbColor', 4)),
    trackColor: goStateColor(n.nested('TrackColor', 5)),
    trackBorderColor: goStateColor(n.nested('TrackBorderColor', 6)),
    crossAxisMargin: n.sizeOpt('CrossAxisMargin', 7),
    mainAxisMargin: n.sizeOpt('MainAxisMargin', 8),
    minThumbLength: n.sizeOpt('MinThumbLength', 9),
    interactive: n.flagOpt('Interactive', 10),
  );
}

/// goSearchBarStyle reads the SearchBarStyle struct of the Go side.
SearchBarThemeData? goSearchBarStyle(GoNode? n) {
  if (n == null) return null;
  return SearchBarThemeData(
    elevation: _all<double>(n.sizeOpt('Elevation', 0)),
    backgroundColor: goStateColor(n.nested('BackgroundColor', 1)),
    shadowColor: goStateColor(n.nested('ShadowColor', 2)),
    surfaceTintColor: goStateColor(n.nested('SurfaceTintColor', 3)),
    overlayColor: goStateColor(n.nested('OverlayColor', 4)),
    side: _all<BorderSide>(goBorderSide(n.nested('Side', 5))),
    shape: _all<OutlinedBorder>(goShape(n.nested('Shape', 6))),
    padding: _all<EdgeInsetsGeometry>(goInsets(n.nested('Padding', 7))),
    textStyle: _all<TextStyle>(goTextStyle(n.nested('TextStyle', 8))),
    hintStyle: _all<TextStyle>(goTextStyle(n.nested('HintStyle', 9))),
    constraints: goConstraints(n.nested('Constraints', 10)),
    textCapitalization: goEnumValue<TextCapitalization>(
      TextCapitalization.values,
      n.text('TextCapitalization', 11),
    ),
  );
}

/// goSearchViewStyle reads the SearchViewStyle struct of the Go side.
SearchViewThemeData? goSearchViewStyle(GoNode? n) {
  if (n == null) return null;
  return SearchViewThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    elevation: n.sizeOpt('Elevation', 1),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 2)),
    constraints: goConstraints(n.nested('Constraints', 3)),
    padding: goInsets(n.nested('Padding', 4)),
    barPadding: goInsets(n.nested('BarPadding', 5)),
    shrinkWrap: n.flagOpt('ShrinkWrap', 6),
    side: goBorderSide(n.nested('Side', 7)),
    shape: goShape(n.nested('Shape', 8)),
    headerHeight: n.sizeOpt('HeaderHeight', 9),
    headerTextStyle: goTextStyle(n.nested('HeaderTextStyle', 10)),
    headerHintStyle: goTextStyle(n.nested('HeaderHintStyle', 11)),
    dividerColor: goColorOpt(n.countOpt('DividerColor', 12)),
  );
}

/// goSegmentedButtonStyle reads the SegmentedButtonStyle struct of the Go side.
SegmentedButtonThemeData? goSegmentedButtonStyle(GoNode? n, GoRuntime rt) {
  if (n == null) return null;
  return SegmentedButtonThemeData(
    style: goButtonStyle(n.nested('Style', 0)),
    selectedIcon: goChildOpt(n.child('SelectedIcon', 1), rt),
  );
}

/// goSliderStyle reads the SliderStyle struct of the Go side.
SliderThemeData? goSliderStyle(GoNode? n) {
  if (n == null) return null;
  return SliderThemeData(
    trackHeight: n.sizeOpt('TrackHeight', 0),
    activeTrackColor: goColorOpt(n.countOpt('ActiveTrackColor', 1)),
    inactiveTrackColor: goColorOpt(n.countOpt('InactiveTrackColor', 2)),
    secondaryActiveTrackColor: goColorOpt(
      n.countOpt('SecondaryActiveTrackColor', 3),
    ),
    disabledActiveTrackColor: goColorOpt(
      n.countOpt('DisabledActiveTrackColor', 4),
    ),
    disabledInactiveTrackColor: goColorOpt(
      n.countOpt('DisabledInactiveTrackColor', 5),
    ),
    disabledSecondaryActiveTrackColor: goColorOpt(
      n.countOpt('DisabledSecondaryActiveTrackColor', 6),
    ),
    activeTickMarkColor: goColorOpt(n.countOpt('ActiveTickMarkColor', 7)),
    inactiveTickMarkColor: goColorOpt(n.countOpt('InactiveTickMarkColor', 8)),
    disabledActiveTickMarkColor: goColorOpt(
      n.countOpt('DisabledActiveTickMarkColor', 9),
    ),
    disabledInactiveTickMarkColor: goColorOpt(
      n.countOpt('DisabledInactiveTickMarkColor', 10),
    ),
    thumbColor: goColorOpt(n.countOpt('ThumbColor', 11)),
    overlappingShapeStrokeColor: goColorOpt(
      n.countOpt('OverlappingShapeStrokeColor', 12),
    ),
    disabledThumbColor: goColorOpt(n.countOpt('DisabledThumbColor', 13)),
    overlayColor: goColorOpt(n.countOpt('OverlayColor', 14)),
    valueIndicatorColor: goColorOpt(n.countOpt('ValueIndicatorColor', 15)),
    valueIndicatorStrokeColor: goColorOpt(
      n.countOpt('ValueIndicatorStrokeColor', 16),
    ),
    showValueIndicator: goEnumValue<ShowValueIndicator>(
      ShowValueIndicator.values,
      n.text('ShowValueIndicator', 17),
    ),
    valueIndicatorTextStyle: goTextStyle(
      n.nested('ValueIndicatorTextStyle', 18),
    ),
    minThumbSeparation: n.sizeOpt('MinThumbSeparation', 19),
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 20))),
    allowedInteraction: goEnumValue<SliderInteraction>(
      SliderInteraction.values,
      n.text('AllowedInteraction', 21),
    ),
    padding: goInsets(n.nested('Padding', 22)),
    thumbSize: _all<Size>(goSize(n.nested('ThumbSize', 23))),
    trackGap: n.sizeOpt('TrackGap', 24),
  );
}

/// goSnackBarStyle reads the SnackBarStyle struct of the Go side.
SnackBarThemeData? goSnackBarStyle(GoNode? n) {
  if (n == null) return null;
  return SnackBarThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    actionTextColor: goColorOpt(n.countOpt('ActionTextColor', 1)),
    disabledActionTextColor: goColorOpt(
      n.countOpt('DisabledActionTextColor', 2),
    ),
    contentTextStyle: goTextStyle(n.nested('ContentTextStyle', 3)),
    elevation: n.sizeOpt('Elevation', 4),
    shape: goShape(n.nested('Shape', 5)),
    behavior: goEnumValue<SnackBarBehavior>(
      SnackBarBehavior.values,
      n.text('Behavior', 6),
    ),
    width: n.sizeOpt('Width', 7),
    insetPadding: goInsets(n.nested('InsetPadding', 8)),
    showCloseIcon: n.flagOpt('ShowCloseIcon', 9),
    closeIconColor: goColorOpt(n.countOpt('CloseIconColor', 10)),
    actionOverflowThreshold: n.sizeOpt('ActionOverflowThreshold', 11),
    actionBackgroundColor: goColorOpt(n.countOpt('ActionBackgroundColor', 12)),
    disabledActionBackgroundColor: goColorOpt(
      n.countOpt('DisabledActionBackgroundColor', 13),
    ),
    dismissDirection: goEnumValue<DismissDirection>(
      DismissDirection.values,
      n.text('DismissDirection', 14),
    ),
  );
}

/// goSwitchStyle reads the SwitchStyle struct of the Go side.
SwitchThemeData? goSwitchStyle(GoNode? n) {
  if (n == null) return null;
  return SwitchThemeData(
    thumbColor: goStateColor(n.nested('ThumbColor', 0)),
    trackColor: goStateColor(n.nested('TrackColor', 1)),
    trackOutlineColor: goStateColor(n.nested('TrackOutlineColor', 2)),
    trackOutlineWidth: _all<double>(n.sizeOpt('TrackOutlineWidth', 3)),
    materialTapTargetSize: goEnumValue<MaterialTapTargetSize>(
      MaterialTapTargetSize.values,
      n.text('MaterialTapTargetSize', 4),
    ),
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 5))),
    overlayColor: goStateColor(n.nested('OverlayColor', 6)),
    splashRadius: n.sizeOpt('SplashRadius', 7),
    padding: goInsets(n.nested('Padding', 8)),
  );
}

/// goTextSelectionStyle reads the TextSelectionStyle struct of the Go side.
TextSelectionThemeData? goTextSelectionStyle(GoNode? n) {
  if (n == null) return null;
  return TextSelectionThemeData(
    cursorColor: goColorOpt(n.countOpt('CursorColor', 0)),
    selectionColor: goColorOpt(n.countOpt('SelectionColor', 1)),
    selectionHandleColor: goColorOpt(n.countOpt('SelectionHandleColor', 2)),
  );
}

/// goTimePickerStyle reads the TimePickerStyle struct of the Go side.
TimePickerThemeData? goTimePickerStyle(GoNode? n) {
  if (n == null) return null;
  return TimePickerThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    cancelButtonStyle: goButtonStyle(n.nested('CancelButtonStyle', 1)),
    confirmButtonStyle: goButtonStyle(n.nested('ConfirmButtonStyle', 2)),
    dayPeriodBorderSide: goBorderSide(n.nested('DayPeriodBorderSide', 3)),
    dayPeriodColor: goColorOpt(n.countOpt('DayPeriodColor', 4)),
    dayPeriodShape: goShape(n.nested('DayPeriodShape', 5)),
    dayPeriodTextColor: goColorOpt(n.countOpt('DayPeriodTextColor', 6)),
    dayPeriodTextStyle: goTextStyle(n.nested('DayPeriodTextStyle', 7)),
    dialBackgroundColor: goColorOpt(n.countOpt('DialBackgroundColor', 8)),
    dialHandColor: goColorOpt(n.countOpt('DialHandColor', 9)),
    dialTextColor: goColorOpt(n.countOpt('DialTextColor', 10)),
    dialTextStyle: goTextStyle(n.nested('DialTextStyle', 11)),
    elevation: n.sizeOpt('Elevation', 12),
    entryModeIconColor: goColorOpt(n.countOpt('EntryModeIconColor', 13)),
    helpTextStyle: goTextStyle(n.nested('HelpTextStyle', 14)),
    hourMinuteColor: goColorOpt(n.countOpt('HourMinuteColor', 15)),
    hourMinuteShape: goShape(n.nested('HourMinuteShape', 16)),
    hourMinuteTextColor: goColorOpt(n.countOpt('HourMinuteTextColor', 17)),
    hourMinuteTextStyle: goTextStyle(n.nested('HourMinuteTextStyle', 18)),
    padding: goInsets(n.nested('Padding', 19)),
    shape: goShape(n.nested('Shape', 20)),
    timeSelectorSeparatorColor: goStateColor(
      n.nested('TimeSelectorSeparatorColor', 21),
    ),
    timeSelectorSeparatorTextStyle: _all<TextStyle>(
      goTextStyle(n.nested('TimeSelectorSeparatorTextStyle', 22)),
    ),
  );
}

/// goToggleButtonsStyle reads the ToggleButtonsStyle struct of the Go side.
ToggleButtonsThemeData? goToggleButtonsStyle(GoNode? n) {
  if (n == null) return null;
  return ToggleButtonsThemeData(
    textStyle: goTextStyle(n.nested('TextStyle', 0)),
    constraints: goConstraints(n.nested('Constraints', 1)),
    color: goColorOpt(n.countOpt('Color', 2)),
    selectedColor: goColorOpt(n.countOpt('SelectedColor', 3)),
    disabledColor: goColorOpt(n.countOpt('DisabledColor', 4)),
    fillColor: goColorOpt(n.countOpt('FillColor', 5)),
    focusColor: goColorOpt(n.countOpt('FocusColor', 6)),
    highlightColor: goColorOpt(n.countOpt('HighlightColor', 7)),
    hoverColor: goColorOpt(n.countOpt('HoverColor', 8)),
    splashColor: goColorOpt(n.countOpt('SplashColor', 9)),
    borderColor: goColorOpt(n.countOpt('BorderColor', 10)),
    selectedBorderColor: goColorOpt(n.countOpt('SelectedBorderColor', 11)),
    disabledBorderColor: goColorOpt(n.countOpt('DisabledBorderColor', 12)),
    borderRadius: goRadius(n.nested('BorderRadius', 13)),
    borderWidth: n.sizeOpt('BorderWidth', 14),
  );
}

/// goTooltipStyle reads the TooltipStyle struct of the Go side.
TooltipThemeData? goTooltipStyle(GoNode? n) {
  if (n == null) return null;
  return TooltipThemeData(
    constraints: goConstraints(n.nested('Constraints', 0)),
    padding: goInsets(n.nested('Padding', 1)),
    margin: goInsets(n.nested('Margin', 2)),
    verticalOffset: n.sizeOpt('VerticalOffset', 3),
    preferBelow: n.flagOpt('PreferBelow', 4),
    excludeFromSemantics: n.flagOpt('ExcludeFromSemantics', 5),
    decoration: goDecoration(n.nested('Decoration', 6)),
    textStyle: goTextStyle(n.nested('TextStyle', 7)),
    textAlign: goEnumValue<TextAlign>(TextAlign.values, n.text('TextAlign', 8)),
    waitDuration: goMillis(n.countOpt('WaitDuration', 9)),
    showDuration: goMillis(n.countOpt('ShowDuration', 10)),
    exitDuration: goMillis(n.countOpt('ExitDuration', 11)),
    triggerMode: goEnumValue<TooltipTriggerMode>(
      TooltipTriggerMode.values,
      n.text('TriggerMode', 12),
    ),
    enableFeedback: n.flagOpt('EnableFeedback', 13),
  );
}

// A theme of 1 button family holds 1 ButtonStyle alone, thus each converter
// below reads the ButtonStyle struct of the Go side.

/// goElevatedButtonTheme gives the ElevatedButtonThemeData of 1 ButtonStyle.
ElevatedButtonThemeData? goElevatedButtonTheme(GoNode? n) =>
    n == null ? null : ElevatedButtonThemeData(style: goButtonStyle(n));

/// goFilledButtonTheme gives the FilledButtonThemeData of 1 ButtonStyle.
FilledButtonThemeData? goFilledButtonTheme(GoNode? n) =>
    n == null ? null : FilledButtonThemeData(style: goButtonStyle(n));

/// goIconButtonTheme gives the IconButtonThemeData of 1 ButtonStyle.
IconButtonThemeData? goIconButtonTheme(GoNode? n) =>
    n == null ? null : IconButtonThemeData(style: goButtonStyle(n));

/// goMenuButtonTheme gives the MenuButtonThemeData of 1 ButtonStyle.
MenuButtonThemeData? goMenuButtonTheme(GoNode? n) =>
    n == null ? null : MenuButtonThemeData(style: goButtonStyle(n));

/// goOutlinedButtonTheme gives the OutlinedButtonThemeData of 1 ButtonStyle.
OutlinedButtonThemeData? goOutlinedButtonTheme(GoNode? n) =>
    n == null ? null : OutlinedButtonThemeData(style: goButtonStyle(n));

/// goTextButtonTheme gives the TextButtonThemeData of 1 ButtonStyle.
TextButtonThemeData? goTextButtonTheme(GoNode? n) =>
    n == null ? null : TextButtonThemeData(style: goButtonStyle(n));

/// goMenuTheme gives the MenuThemeData of 1 MenuStyle.
MenuThemeData? goMenuTheme(GoNode? n) =>
    n == null ? null : MenuThemeData(style: goMenuStyle(n));

/// goMenuBarTheme gives the MenuBarThemeData of 1 MenuStyle.
MenuBarThemeData? goMenuBarTheme(GoNode? n) =>
    n == null ? null : MenuBarThemeData(style: goMenuStyle(n));

/// _txt gives a text, and null for an empty 1. An empty Go text means "the
/// default of Flutter".
String? _txt(String s) => s.isEmpty ? null : s;

/// goAppBarStyle reads the AppBarStyle struct of the Go side.
AppBarThemeData? goAppBarStyle(GoNode? n) {
  if (n == null) return null;
  return AppBarThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    foregroundColor: goColorOpt(n.countOpt('ForegroundColor', 1)),
    elevation: n.sizeOpt('Elevation', 2),
    scrolledUnderElevation: n.sizeOpt('ScrolledUnderElevation', 3),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 4)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 5)),
    shape: goShape(n.nested('Shape', 6)),
    iconTheme: goIconStyle(n.nested('IconTheme', 7)),
    actionsIconTheme: goIconStyle(n.nested('ActionsIconTheme', 8)),
    centerTitle: n.flagOpt('CenterTitle', 9),
    titleSpacing: n.sizeOpt('TitleSpacing', 10),
    leadingWidth: n.sizeOpt('LeadingWidth', 11),
    toolbarHeight: n.sizeOpt('ToolbarHeight', 12),
    toolbarTextStyle: goTextStyle(n.nested('ToolbarTextStyle', 13)),
    titleTextStyle: goTextStyle(n.nested('TitleTextStyle', 14)),
    actionsPadding: goInsets(n.nested('ActionsPadding', 15)),
  );
}

/// goBottomAppBarStyle reads the BottomAppBarStyle struct of the Go side.
BottomAppBarThemeData? goBottomAppBarStyle(GoNode? n) {
  if (n == null) return null;
  return BottomAppBarThemeData(
    color: goColorOpt(n.countOpt('Color', 0)),
    elevation: n.sizeOpt('Elevation', 1),
    height: n.sizeOpt('Height', 2),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 3)),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 4)),
    padding: goInsets(n.nested('Padding', 5)),
  );
}

/// goCardStyle reads the CardStyle struct of the Go side.
CardThemeData? goCardStyle(GoNode? n) {
  if (n == null) return null;
  return CardThemeData(
    clipBehavior: goEnumValue<Clip>(Clip.values, n.text('ClipBehavior', 0)),
    color: goColorOpt(n.countOpt('Color', 1)),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 2)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 3)),
    elevation: n.sizeOpt('Elevation', 4),
    margin: goInsets(n.nested('Margin', 5)),
    shape: goShape(n.nested('Shape', 6)),
  );
}

/// goDialogStyle reads the DialogStyle struct of the Go side.
DialogThemeData? goDialogStyle(GoNode? n) {
  if (n == null) return null;
  return DialogThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    elevation: n.sizeOpt('Elevation', 1),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 2)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 3)),
    shape: goShape(n.nested('Shape', 4)),
    alignment: goAlignment(n.text('Alignment', 5)),
    iconColor: goColorOpt(n.countOpt('IconColor', 6)),
    titleTextStyle: goTextStyle(n.nested('TitleTextStyle', 7)),
    contentTextStyle: goTextStyle(n.nested('ContentTextStyle', 8)),
    actionsPadding: goInsets(n.nested('ActionsPadding', 9)),
    barrierColor: goColorOpt(n.countOpt('BarrierColor', 10)),
    insetPadding: goInsets(n.nested('InsetPadding', 11)),
    clipBehavior: goEnumValue<Clip>(Clip.values, n.text('ClipBehavior', 12)),
    constraints: goConstraints(n.nested('Constraints', 13)),
  );
}

/// goInputDecorationStyle reads the InputDecorationStyle struct of the Go
/// side.
///
/// 5 parameters of the Flutter type take no null, thus each 1 reads the
/// default of Flutter when the Go field holds none.
InputDecorationThemeData? goInputDecorationStyle(GoNode? n) {
  if (n == null) return null;
  return InputDecorationThemeData(
    labelStyle: goTextStyle(n.nested('LabelStyle', 0)),
    floatingLabelStyle: goTextStyle(n.nested('FloatingLabelStyle', 1)),
    helperStyle: goTextStyle(n.nested('HelperStyle', 2)),
    helperMaxLines: n.countOpt('HelperMaxLines', 3),
    hintStyle: goTextStyle(n.nested('HintStyle', 4)),
    hintFadeDuration: goMillis(n.countOpt('HintFadeDuration', 5)),
    hintMaxLines: n.countOpt('HintMaxLines', 6),
    errorStyle: goTextStyle(n.nested('ErrorStyle', 7)),
    errorMaxLines: n.countOpt('ErrorMaxLines', 8),
    floatingLabelBehavior:
        goEnumValue<FloatingLabelBehavior>(
          FloatingLabelBehavior.values,
          n.text('FloatingLabelBehavior', 9),
        ) ??
        FloatingLabelBehavior.auto,
    isDense: n.flagOpt('IsDense', 10) ?? false,
    contentPadding: goInsets(n.nested('ContentPadding', 11)),
    isCollapsed: n.flagOpt('IsCollapsed', 12) ?? false,
    iconColor: goColorOpt(n.countOpt('IconColor', 13)),
    prefixStyle: goTextStyle(n.nested('PrefixStyle', 14)),
    prefixIconColor: goColorOpt(n.countOpt('PrefixIconColor', 15)),
    prefixIconConstraints: goConstraints(n.nested('PrefixIconConstraints', 16)),
    suffixStyle: goTextStyle(n.nested('SuffixStyle', 17)),
    suffixIconColor: goColorOpt(n.countOpt('SuffixIconColor', 18)),
    suffixIconConstraints: goConstraints(n.nested('SuffixIconConstraints', 19)),
    counterStyle: goTextStyle(n.nested('CounterStyle', 20)),
    filled: n.flagOpt('Filled', 21) ?? false,
    fillColor: goColorOpt(n.countOpt('FillColor', 22)),
    activeIndicatorBorder: goBorderSide(n.nested('ActiveIndicatorBorder', 23)),
    outlineBorder: goBorderSide(n.nested('OutlineBorder', 24)),
    focusColor: goColorOpt(n.countOpt('FocusColor', 25)),
    hoverColor: goColorOpt(n.countOpt('HoverColor', 26)),
    alignLabelWithHint: n.flagOpt('AlignLabelWithHint', 27) ?? false,
    constraints: goConstraints(n.nested('Constraints', 28)),
    visualDensity: goDensity(n.nested('VisualDensity', 29)),
  );
}

/// goListTileStyle reads the ListTileStyle struct of the Go side.
ListTileThemeData? goListTileStyle(GoNode? n) {
  if (n == null) return null;
  return ListTileThemeData(
    dense: n.flagOpt('Dense', 0),
    shape: goShape(n.nested('Shape', 1)),
    style: goEnumValue<ListTileStyle>(ListTileStyle.values, n.text('Style', 2)),
    selectedColor: goColorOpt(n.countOpt('SelectedColor', 3)),
    iconColor: goColorOpt(n.countOpt('IconColor', 4)),
    textColor: goColorOpt(n.countOpt('TextColor', 5)),
    titleTextStyle: goTextStyle(n.nested('TitleTextStyle', 6)),
    subtitleTextStyle: goTextStyle(n.nested('SubtitleTextStyle', 7)),
    leadingAndTrailingTextStyle: goTextStyle(
      n.nested('LeadingAndTrailingTextStyle', 8),
    ),
    contentPadding: goInsets(n.nested('ContentPadding', 9)),
    tileColor: goColorOpt(n.countOpt('TileColor', 10)),
    selectedTileColor: goColorOpt(n.countOpt('SelectedTileColor', 11)),
    horizontalTitleGap: n.sizeOpt('HorizontalTitleGap', 12),
    minVerticalPadding: n.sizeOpt('MinVerticalPadding', 13),
    minLeadingWidth: n.sizeOpt('MinLeadingWidth', 14),
    minTileHeight: n.sizeOpt('MinTileHeight', 15),
    enableFeedback: n.flagOpt('EnableFeedback', 16),
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 17))),
    visualDensity: goDensity(n.nested('VisualDensity', 18)),
    titleAlignment: goEnumValue<ListTileTitleAlignment>(
      ListTileTitleAlignment.values,
      n.text('TitleAlignment', 19),
    ),
    controlAffinity: goEnumValue<ListTileControlAffinity>(
      ListTileControlAffinity.values,
      n.text('ControlAffinity', 20),
    ),
    isThreeLine: n.flagOpt('IsThreeLine', 21),
  );
}

/// goMaterialBannerStyle reads the MaterialBannerStyle struct of the Go side.
MaterialBannerThemeData? goMaterialBannerStyle(GoNode? n) {
  if (n == null) return null;
  return MaterialBannerThemeData(
    backgroundColor: goColorOpt(n.countOpt('BackgroundColor', 0)),
    surfaceTintColor: goColorOpt(n.countOpt('SurfaceTintColor', 1)),
    shadowColor: goColorOpt(n.countOpt('ShadowColor', 2)),
    dividerColor: goColorOpt(n.countOpt('DividerColor', 3)),
    contentTextStyle: goTextStyle(n.nested('ContentTextStyle', 4)),
    elevation: n.sizeOpt('Elevation', 5),
    padding: goInsets(n.nested('Padding', 6)),
    leadingPadding: goInsets(n.nested('LeadingPadding', 7)),
  );
}

/// goTabBarStyle reads the TabBarStyle struct of the Go side.
TabBarThemeData? goTabBarStyle(GoNode? n) {
  if (n == null) return null;
  return TabBarThemeData(
    indicator: goDecoration(n.nested('Indicator', 0)),
    indicatorColor: goColorOpt(n.countOpt('IndicatorColor', 1)),
    indicatorSize: goEnumValue<TabBarIndicatorSize>(
      TabBarIndicatorSize.values,
      n.text('IndicatorSize', 2),
    ),
    dividerColor: goColorOpt(n.countOpt('DividerColor', 3)),
    dividerHeight: n.sizeOpt('DividerHeight', 4),
    labelColor: goColorOpt(n.countOpt('LabelColor', 5)),
    labelPadding: goInsets(n.nested('LabelPadding', 6)),
    labelStyle: goTextStyle(n.nested('LabelStyle', 7)),
    unselectedLabelColor: goColorOpt(n.countOpt('UnselectedLabelColor', 8)),
    unselectedLabelStyle: goTextStyle(n.nested('UnselectedLabelStyle', 9)),
    overlayColor: goStateColor(n.nested('OverlayColor', 10)),
    mouseCursor: _all<MouseCursor>(goCursor(n.text('MouseCursor', 11))),
    tabAlignment: goEnumValue<TabAlignment>(
      TabAlignment.values,
      n.text('TabAlignment', 12),
    ),
    indicatorAnimation: goEnumValue<TabIndicatorAnimation>(
      TabIndicatorAnimation.values,
      n.text('IndicatorAnimation', 13),
    ),
    splashBorderRadius: goRadius(n.nested('SplashBorderRadius', 14)),
  );
}

/// goToolbarAnchors reads the ToolbarAnchors struct of the Go side.
TextSelectionToolbarAnchors? goToolbarAnchors(GoNode? n) {
  if (n == null) return null;
  return TextSelectionToolbarAnchors(
    primaryAnchor: goOffset(n.nested('PrimaryAnchor', 0)) ?? Offset.zero,
    secondaryAnchor: goOffset(n.nested('SecondaryAnchor', 1)),
  );
}

/// goSelectionGeometry reads the SelectionGeometry struct of the Go side. An
/// empty Status gives "none", which means no selection.
SelectionGeometry? goSelectionGeometry(GoNode? n) {
  if (n == null) return null;
  return SelectionGeometry(
    status:
        goEnumValue<SelectionStatus>(
          SelectionStatus.values,
          n.text('Status', 0),
        ) ??
        SelectionStatus.none,
    hasContent: n.flag('HasContent', 1),
  );
}

/// goSemanticsProps reads the SemanticsProperties struct of the Go side. It
/// carries the same subset as the node ui.Semantics.
SemanticsProperties? goSemanticsProps(GoNode? n, GoRuntime rt) {
  if (n == null) return null;
  return SemanticsProperties(
    enabled: n.flagOpt('Enabled', 0),
    checked: n.flagOpt('Checked', 1),
    mixed: n.flagOpt('Mixed', 2),
    selected: n.flagOpt('Selected', 3),
    toggled: n.flagOpt('Toggled', 4),
    button: n.flagOpt('Button', 5),
    slider: n.flagOpt('Slider', 6),
    keyboardKey: n.flagOpt('KeyboardKey', 7),
    link: n.flagOpt('Link', 8),
    header: n.flagOpt('Header', 9),
    headingLevel: n.countOpt('HeadingLevel', 10),
    textField: n.flagOpt('TextField', 11),
    readOnly: n.flagOpt('ReadOnly', 12),
    // Flutter holds focusable as deprecated, and the node ui.Semantics carries
    // it too.
    // ignore: deprecated_member_use
    focusable: n.flagOpt('Focusable', 13),
    focused: n.flagOpt('Focused', 14),
    accessibilityFocusBlockType: goEnumValue<AccessibilityFocusBlockType>(
      AccessibilityFocusBlockType.values,
      n.text('AccessibilityFocusBlockType', 15),
    ),
    inMutuallyExclusiveGroup: n.flagOpt('InMutuallyExclusiveGroup', 16),
    obscured: n.flagOpt('Obscured', 17),
    multiline: n.flagOpt('Multiline', 18),
    scopesRoute: n.flagOpt('ScopesRoute', 19),
    namesRoute: n.flagOpt('NamesRoute', 20),
    hidden: n.flagOpt('Hidden', 21),
    image: n.flagOpt('Image', 22),
    liveRegion: n.flagOpt('LiveRegion', 23),
    expanded: n.flagOpt('Expanded', 24),
    isRequired: n.flagOpt('IsRequired', 25),
    maxValueLength: n.countOpt('MaxValueLength', 26),
    currentValueLength: n.countOpt('CurrentValueLength', 27),
    identifier: _txt(n.text('Identifier', 28)),
    traversalParentIdentifier: _txt(n.text('TraversalParentIdentifier', 29)),
    traversalChildIdentifier: _txt(n.text('TraversalChildIdentifier', 30)),
    label: _txt(n.text('Label', 31)),
    value: _txt(n.text('Value', 32)),
    increasedValue: _txt(n.text('IncreasedValue', 33)),
    decreasedValue: _txt(n.text('DecreasedValue', 34)),
    hint: _txt(n.text('Hint', 35)),
    tooltip: _txt(n.text('Tooltip', 36)),
    textDirection: goEnumValue<TextDirection>(
      TextDirection.values,
      n.text('TextDirection', 37),
    ),
    role: goEnumValue<SemanticsRole>(SemanticsRole.values, n.text('Role', 38)),
    validationResult:
        goEnumValue<SemanticsValidationResult>(
          SemanticsValidationResult.values,
          n.text('ValidationResult', 39),
        ) ??
        SemanticsValidationResult.none,
    minValue: _txt(n.text('MinValue', 40)),
    maxValue: _txt(n.text('MaxValue', 41)),
    onTap: goAction(n.action('OnTap', 42), rt),
    onLongPress: goAction(n.action('OnLongPress', 43), rt),
    onScrollLeft: goAction(n.action('OnScrollLeft', 44), rt),
    onScrollRight: goAction(n.action('OnScrollRight', 45), rt),
    onScrollUp: goAction(n.action('OnScrollUp', 46), rt),
    onScrollDown: goAction(n.action('OnScrollDown', 47), rt),
    onIncrease: goAction(n.action('OnIncrease', 48), rt),
    onDecrease: goAction(n.action('OnDecrease', 49), rt),
    onCopy: goAction(n.action('OnCopy', 50), rt),
    onCut: goAction(n.action('OnCut', 51), rt),
    onPaste: goAction(n.action('OnPaste', 52), rt),
    onDismiss: goAction(n.action('OnDismiss', 53), rt),
    onMoveCursorForwardByCharacter: goOnBool(
      n.onFlag('OnMoveCursorForwardByCharacter', 54),
      rt,
    ),
    onMoveCursorBackwardByCharacter: goOnBool(
      n.onFlag('OnMoveCursorBackwardByCharacter', 55),
      rt,
    ),
    onMoveCursorForwardByWord: goOnBool(
      n.onFlag('OnMoveCursorForwardByWord', 56),
      rt,
    ),
    onMoveCursorBackwardByWord: goOnBool(
      n.onFlag('OnMoveCursorBackwardByWord', 57),
      rt,
    ),
    onSetText: goOnStr(n.onText('OnSetText', 58), rt),
    onDidGainAccessibilityFocus: goAction(
      n.action('OnDidGainAccessibilityFocus', 59),
      rt,
    ),
    onDidLoseAccessibilityFocus: goAction(
      n.action('OnDidLoseAccessibilityFocus', 60),
      rt,
    ),
    onFocus: goAction(n.action('OnFocus', 61), rt),
    onExpand: goAction(n.action('OnExpand', 62), rt),
    onCollapse: goAction(n.action('OnCollapse', 63), rt),
  );
}

/// goExtentBuilder wraps a builder that gives the size of 1 item of a list.
/// The host gives the 4 fields of the SliverLayoutDimensions as plain numbers,
/// and a result below 0 gives the null of Dart, which holds the end of the
/// list.
ItemExtentBuilder? goExtentBuilder(
  double Function(int, double, double, double, double)? f,
) => f == null
    ? null
    : (int index, SliverLayoutDimensions d) {
        final double v = f(
          index,
          d.scrollOffset,
          d.precedingScrollExtent,
          d.viewportMainAxisExtent,
          d.crossAxisExtent,
        );
        return v < 0 ? null : v;
      };

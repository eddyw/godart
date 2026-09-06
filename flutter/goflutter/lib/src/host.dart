import 'dart:ui' show ImageFilter;

import 'package:material_ui/material_ui.dart';
import 'package:goruntime/goruntime.dart';

import 'convert.dart';
import 'identity.dart';
import 'node.dart';
import 'runtime.dart';

/// goIcons maps the name of a Go Icon node to the Material set. An application
/// adds a name that it needs.
final Map<String, IconData> goIcons = <String, IconData>{
  'add': Icons.add,
  'remove': Icons.remove,
  'delete': Icons.delete,
  'refresh': Icons.refresh,
  'search': Icons.search,
  'close': Icons.close,
  'check': Icons.check,
  'edit': Icons.edit,
  'menu': Icons.menu,
  'settings': Icons.settings,
  'home': Icons.home,
  'person': Icons.person,
  'favorite': Icons.favorite,
  'star': Icons.star,
  'share': Icons.share,
  'arrow_back': Icons.arrow_back,
  'arrow_forward': Icons.arrow_forward,
  'more_vert': Icons.more_vert,
  'error': Icons.error,
  'info': Icons.info,
};

/// GoWidgets builds the Flutter widget of 1 generated node, or gives null when
/// the node stands outside the set that the application holds.
typedef GoWidgets = Widget? Function(GoNode n, GoRuntime rt);

GoWidgets? _widgets;

/// goAnchorContext holds the BuildContext of the Anchor node of the tree.
///
/// showDateRangePicker, a dialog, and a snack bar each need a context under
/// the application. Read the Anchor node of flutter/ui/node.go.
BuildContext? goAnchorContext;

/// GoAnchor keeps its own context while it stands on the screen.
final class GoAnchor extends StatefulWidget {
  final Widget child;

  const GoAnchor({super.key, required this.child});

  @override
  State<GoAnchor> createState() => _GoAnchorState();
}

class _GoAnchorState extends State<GoAnchor> {
  @override
  void dispose() {
    if (identical(goAnchorContext, context)) goAnchorContext = null;
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    goAnchorContext = context;
    return widget.child;
  }
}

/// goSetWidgets installs the generated node set of the application.
///
/// nodegen writes 1 such function for each application, and it holds a case
/// only for the node types that the Go source of that application names. A
/// shared function would name each Flutter widget, and the tree shaker of Dart
/// would then keep the whole framework. The measurement of 2026-09-02 puts that
/// cost at 7.2 megabytes of the ahead of time snapshot.
void goSetWidgets(GoWidgets f) {
  _widgets = f;
}

/// goRender builds 1 Flutter widget from 1 node of the Go tree.
///
/// The 3 cases below hold a node that no Flutter widget carries. Each other
/// node reaches the generated set that goSetWidgets installed.
Widget goRender(Iface? node, GoRuntime rt) {
  if (node == null) return const SizedBox.shrink();
  final GoNode n = GoNode(node);
  switch (n.kind) {
    case 'gart.dev/ui.Watch':
      return GoWatch(runtime: rt, build: n.raw('Build', 0));

    case 'gart.dev/ui.Keyed':
      return KeyedSubtree(
        key: ValueKey<String>(n.text('Key', 0)),
        child: goRender(n.child('Child', 1), rt),
      );

    case 'gart.dev/ui.Pages':
      final List<String> keys = n.texts('Keys', 0);
      final List<Iface?> kids = n.children('Children', 1);
      final void Function(int)? onPop = n.onCount('OnPop', 2);
      return Navigator(
        pages: <Page<Object?>>[
          for (int i = 0; i < kids.length; i++)
            MaterialPage<Object?>(
              key: ValueKey<String>(i < keys.length ? keys[i] : '$i'),
              child: goRender(kids[i], rt),
            ),
        ],
        onDidRemovePage: (Page<Object?> page) {
          final Object? key = page.key;
          if (onPop == null || key is! ValueKey<String>) return;
          final int i = keys.indexOf(key.value);
          if (i >= 0) onPop(i);
        },
      );

    case 'gart.dev/ui.Anchor':
      return GoAnchor(child: goRender(n.child('Child', 0), rt));

    case 'gart.dev/ui.ShaderMask':
      final Widget child = goRender(n.child('Child', 2), rt);
      final Gradient? paint = goGradient(n.nested('Gradient', 0));
      if (paint == null) return child;
      return ShaderMask(
        blendMode:
            goEnumValue<BlendMode>(BlendMode.values, n.text('BlendMode', 1)) ??
            BlendMode.modulate,
        shaderCallback: (Rect bounds) => paint.createShader(bounds),
        child: child,
      );

    case 'gart.dev/ui.Blur':
      return BackdropFilter(
        filter: ImageFilter.blur(
          sigmaX: n.size('SigmaX', 0),
          sigmaY: n.size('SigmaY', 1),
        ),
        blendMode:
            goEnumValue<BlendMode>(BlendMode.values, n.text('BlendMode', 2)) ??
            BlendMode.srcOver,
        child: goRender(n.child('Child', 3), rt),
      );

    case 'gart.dev/ui.Picture':
      final ImageProvider<Object>? source = goImage(n.nested('Image', 0));
      if (source == null) return const SizedBox.shrink();
      final Iface? loading = n.child('Loading', 6);
      final Iface? failed = n.child('Error', 7);
      return Image(
        image: source,
        width: n.sizeOpt('Width', 1),
        height: n.sizeOpt('Height', 2),
        fit: goEnumValue<BoxFit>(BoxFit.values, n.text('Fit', 3)),
        color: goColorOpt(n.countOpt('Color', 4)),
        colorBlendMode: goEnumValue<BlendMode>(
          BlendMode.values,
          n.text('BlendMode', 5),
        ),
        errorBuilder: failed == null
            ? null
            : (BuildContext c, Object e, StackTrace? t) =>
                  goRender(failed, rt),
        frameBuilder: loading == null
            ? null
            : (BuildContext c, Widget child, int? frame, bool sync) =>
                  frame == null && !sync ? goRender(loading, rt) : child,
      );

    case 'gart.dev/ui.TextInput':
      return GoTextField(
        value: n.text('Value', 0),
        hint: n.text('Hint', 1),
        label: n.text('Label', 2),
        obscure: n.flag('Obscure', 3),
        onChanged: n.onText('OnChanged', 4),
        onSubmit: n.onText('OnSubmit', 5),
        runtime: rt,
      );
  }
  // The build of a generated node may read a key, for example a ui.FocusKey.
  // The host then wraps the widget, and the element of the wrapper holds the
  // Flutter object until the widget leaves the tree.
  final GoIdentities table = rt.identities;
  final Widget? built;
  final Set<String> keys;
  table.open();
  try {
    built = _widgets?.call(n, rt);
  } finally {
    keys = table.close();
  }
  if (built == null) {
    throw StateError('the host holds no case for the node ${n.kind}. '
        'Run nodegen again, or call goSetWidgets.');
  }
  if (keys.isEmpty) return built;
  return GoKeep(ids: keys, table: table, child: built);
}


/// GoWatch is 1 rebuild boundary. It holds a scope of the Go graph, and it
/// rebuilds alone when a signal that its build read changes.
final class GoWatch extends StatefulWidget {
  final GoRuntime runtime;
  final Object? build;

  const GoWatch({super.key, required this.runtime, required this.build});

  @override
  State<GoWatch> createState() => _GoWatchState();
}

class _GoWatchState extends State<GoWatch> {
  late final int _id;

  @override
  void initState() {
    super.initState();
    _id = widget.runtime.bridge.newWatch();
    widget.runtime.bind(_id, _mark);
  }

  void _mark() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    widget.runtime.unbind(_id);
    widget.runtime.bridge.disposeWatch(_id);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return goRender(
      widget.runtime.bridge.runWatch(_id, widget.build),
      widget.runtime,
    );
  }
}

/// GoTextField keeps the controller and the focus of a Flutter TextField across
/// a rebuild that Go drives.
final class GoTextField extends StatefulWidget {
  final String value;
  final String hint;
  final String label;
  final bool obscure;
  final void Function(GoString)? onChanged;
  final void Function(GoString)? onSubmit;
  final GoRuntime runtime;

  const GoTextField({
    super.key,
    required this.value,
    required this.hint,
    required this.label,
    required this.obscure,
    required this.onChanged,
    required this.onSubmit,
    required this.runtime,
  });

  @override
  State<GoTextField> createState() => _GoTextFieldState();
}

class _GoTextFieldState extends State<GoTextField> {
  late final TextEditingController _c = TextEditingController(
    text: widget.value,
  );

  @override
  void didUpdateWidget(covariant GoTextField old) {
    super.didUpdateWidget(old);
    if (widget.value != _c.text) {
      _c.value = TextEditingValue(
        text: widget.value,
        selection: TextSelection.collapsed(offset: widget.value.length),
      );
    }
  }

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final void Function(GoString)? changed = widget.onChanged;
    final void Function(GoString)? submit = widget.onSubmit;
    return TextField(
      controller: _c,
      obscureText: widget.obscure,
      decoration: InputDecoration(
        hintText: widget.hint == '' ? null : widget.hint,
        labelText: widget.label == '' ? null : widget.label,
      ),
      onChanged: changed == null
          ? null
          : (String v) =>
                widget.runtime.run(() => changed(GoString.fromString(v))),
      onSubmitted: submit == null
          ? null
          : (String v) =>
                widget.runtime.run(() => submit(GoString.fromString(v))),
    );
  }
}

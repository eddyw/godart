import 'package:flutter/material.dart';
import 'package:goruntime/goruntime.dart';

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
      return GoWatch(runtime: rt, build: n.raw('Build'));

    case 'gart.dev/ui.Keyed':
      return KeyedSubtree(
        key: ValueKey<String>(n.text('Key')),
        child: goRender(n.child('Child'), rt),
      );

    case 'gart.dev/ui.TextInput':
      return GoTextField(
        value: n.text('Value'),
        hint: n.text('Hint'),
        label: n.text('Label'),
        obscure: n.flag('Obscure'),
        onChanged: n.onText('OnChanged'),
        onSubmit: n.onText('OnSubmit'),
        runtime: rt,
      );
  }
  final Widget? built = _widgets?.call(n, rt);
  if (built != null) return built;
  throw StateError('the host holds no case for the node ${n.kind}. '
      'Run nodegen again, or call goSetWidgets.');
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

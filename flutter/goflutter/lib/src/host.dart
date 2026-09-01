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

/// goRender builds 1 Flutter widget from 1 node of the Go tree.
Widget goRender(Iface? node, GoRuntime rt) {
  if (node == null) return const SizedBox.shrink();
  final GoNode n = GoNode(node);
  switch (n.kind) {
    case 'gart.dev/ui.Watch':
      return GoWatch(runtime: rt, build: n.raw('Build'));

    case 'gart.dev/ui.Text':
      final int? argb = n.argb('Color');
      final double size = n.size('Size');
      return Text(
        n.text('Data'),
        textAlign: _align(n.text('Align')),
        style: TextStyle(
          fontSize: size == 0 ? null : size,
          fontWeight: n.flag('Bold') ? FontWeight.bold : null,
          color: argb == null ? null : Color(argb),
        ),
      );

    case 'gart.dev/ui.Column':
      return Column(
        mainAxisAlignment: _main(n.text('Main')),
        crossAxisAlignment: _cross(n.text('Cross')),
        children: _kids(n.children('Children'), rt),
      );

    case 'gart.dev/ui.Row':
      return Row(
        mainAxisAlignment: _main(n.text('Main')),
        crossAxisAlignment: _cross(n.text('Cross')),
        children: _kids(n.children('Children'), rt),
      );

    case 'gart.dev/ui.Stack':
      return Stack(children: _kids(n.children('Children'), rt));

    case 'gart.dev/ui.Center':
      return Center(child: goRender(n.child('Child'), rt));

    case 'gart.dev/ui.Padding':
      final double all = n.size('All');
      return Padding(
        padding: all != 0
            ? EdgeInsets.all(all)
            : EdgeInsets.fromLTRB(
                n.size('Left'),
                n.size('Top'),
                n.size('Right'),
                n.size('Bottom'),
              ),
        child: goRender(n.child('Child'), rt),
      );

    case 'gart.dev/ui.SizedBox':
      final double w = n.size('Width');
      final double h = n.size('Height');
      return SizedBox(
        width: w == 0 ? null : w,
        height: h == 0 ? null : h,
        child: goRender(n.child('Child'), rt),
      );

    case 'gart.dev/ui.Expanded':
      final int flex = n.count('Flex');
      return Expanded(
        flex: flex == 0 ? 1 : flex,
        child: goRender(n.child('Child'), rt),
      );

    case 'gart.dev/ui.Spacer':
      final int flex = n.count('Flex');
      return Spacer(flex: flex == 0 ? 1 : flex);

    case 'gart.dev/ui.Container':
      final int? argb = n.argb('Color');
      final double pad = n.size('Padding');
      final double w = n.size('Width');
      final double h = n.size('Height');
      final double radius = n.size('Radius');
      return Container(
        width: w == 0 ? null : w,
        height: h == 0 ? null : h,
        padding: pad == 0 ? null : EdgeInsets.all(pad),
        decoration: argb == null && radius == 0
            ? null
            : BoxDecoration(
                color: argb == null ? null : Color(argb),
                borderRadius: radius == 0
                    ? null
                    : BorderRadius.circular(radius),
              ),
        child: goRender(n.child('Child'), rt),
      );

    case 'gart.dev/ui.Divider':
      final double h = n.size('Height');
      return Divider(height: h == 0 ? null : h);

    case 'gart.dev/ui.Icon':
      final int? argb = n.argb('Color');
      final double size = n.size('Size');
      return Icon(
        goIcons[n.text('Name')] ?? Icons.help_outline,
        size: size == 0 ? null : size,
        color: argb == null ? null : Color(argb),
      );

    case 'gart.dev/ui.Image':
      final String url = n.text('URL');
      final double w = n.size('Width');
      final double h = n.size('Height');
      final BoxFit fit = _fit(n.text('Fit'));
      if (url != '') {
        return Image.network(
          url,
          width: w == 0 ? null : w,
          height: h == 0 ? null : h,
          fit: fit,
        );
      }
      return Image.asset(
        n.text('Asset'),
        width: w == 0 ? null : w,
        height: h == 0 ? null : h,
        fit: fit,
      );

    case 'gart.dev/ui.Progress':
      final double size = n.size('Size');
      if (size == 0) return const CircularProgressIndicator();
      return SizedBox(
        width: size,
        height: size,
        child: const CircularProgressIndicator(),
      );

    case 'gart.dev/ui.ElevatedButton':
      return ElevatedButton(
        onPressed: _tap(n.action('OnPressed'), rt),
        child: goRender(n.child('Child'), rt),
      );

    case 'gart.dev/ui.TextButton':
      return TextButton(
        onPressed: _tap(n.action('OnPressed'), rt),
        child: goRender(n.child('Child'), rt),
      );

    case 'gart.dev/ui.IconButton':
      final String tip = n.text('Tooltip');
      return IconButton(
        icon: Icon(goIcons[n.text('Icon')] ?? Icons.help_outline),
        tooltip: tip == '' ? null : tip,
        onPressed: _tap(n.action('OnPressed'), rt),
      );

    case 'gart.dev/ui.TextField':
      return GoTextField(
        value: n.text('Value'),
        hint: n.text('Hint'),
        label: n.text('Label'),
        obscure: n.flag('Obscure'),
        onChanged: n.onText('OnChanged'),
        onSubmit: n.onText('OnSubmit'),
        runtime: rt,
      );

    case 'gart.dev/ui.Switch':
      final void Function(bool)? changed = n.onFlag('OnChanged');
      return Switch(
        value: n.flag('Value'),
        onChanged: changed == null
            ? null
            : (bool v) => rt.run(() => changed(v)),
      );

    case 'gart.dev/ui.Checkbox':
      final void Function(bool)? changed = n.onFlag('OnChanged');
      return Checkbox(
        value: n.flag('Value'),
        onChanged: changed == null
            ? null
            : (bool? v) => rt.run(() => changed(v ?? false)),
      );

    case 'gart.dev/ui.InkWell':
      return InkWell(
        onTap: _tap(n.action('OnTap'), rt),
        child: goRender(n.child('Child'), rt),
      );

    case 'gart.dev/ui.Scaffold':
      final Iface? bar = n.child('AppBar');
      final Iface? fab = n.child('Fab');
      return Scaffold(
        appBar: bar == null ? null : goRender(bar, rt) as PreferredSizeWidget,
        body: goRender(n.child('Body'), rt),
        floatingActionButton: fab == null ? null : goRender(fab, rt),
      );

    case 'gart.dev/ui.AppBar':
      return AppBar(
        title: goRender(n.child('Title'), rt),
        actions: _kids(n.children('Actions'), rt),
      );

    case 'gart.dev/ui.ListView':
      final double pad = n.size('Padding');
      return ListView(
        padding: pad == 0 ? null : EdgeInsets.all(pad),
        children: _kids(n.children('Children'), rt),
      );

    case 'gart.dev/ui.ListViewBuilder':
      final Iface? Function(int)? item = n.item('Item');
      final double pad = n.size('Padding');
      return ListView.builder(
        padding: pad == 0 ? null : EdgeInsets.all(pad),
        itemCount: n.count('Count'),
        itemBuilder: (BuildContext _, int i) =>
            goRender(item == null ? null : item(i), rt),
      );

    case 'gart.dev/ui.Card':
      final double pad = n.size('Padding');
      final Widget child = goRender(n.child('Child'), rt);
      return Card(
        child: pad == 0
            ? child
            : Padding(padding: EdgeInsets.all(pad), child: child),
      );

    case 'gart.dev/ui.Keyed':
      return KeyedSubtree(
        key: ValueKey<String>(n.text('Key')),
        child: goRender(n.child('Child'), rt),
      );
  }
  throw StateError('the host holds no case for the node ${n.kind}');
}

List<Widget> _kids(List<Iface?> list, GoRuntime rt) {
  return <Widget>[for (final Iface? n in list) goRender(n, rt)];
}

VoidCallback? _tap(void Function()? fn, GoRuntime rt) {
  if (fn == null) return null;
  return () => rt.run(fn);
}

MainAxisAlignment _main(String s) {
  switch (s) {
    case 'center':
      return MainAxisAlignment.center;
    case 'end':
      return MainAxisAlignment.end;
    case 'between':
      return MainAxisAlignment.spaceBetween;
    case 'around':
      return MainAxisAlignment.spaceAround;
    case 'evenly':
      return MainAxisAlignment.spaceEvenly;
  }
  return MainAxisAlignment.start;
}

CrossAxisAlignment _cross(String s) {
  switch (s) {
    case 'start':
      return CrossAxisAlignment.start;
    case 'end':
      return CrossAxisAlignment.end;
    case 'stretch':
      return CrossAxisAlignment.stretch;
  }
  return CrossAxisAlignment.center;
}

TextAlign? _align(String s) {
  switch (s) {
    case 'center':
      return TextAlign.center;
    case 'end':
      return TextAlign.end;
    case 'start':
      return TextAlign.start;
  }
  return null;
}

BoxFit _fit(String s) {
  switch (s) {
    case 'contain':
      return BoxFit.contain;
    case 'fill':
      return BoxFit.fill;
    case 'fitWidth':
      return BoxFit.fitWidth;
    case 'fitHeight':
      return BoxFit.fitHeight;
  }
  return BoxFit.cover;
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

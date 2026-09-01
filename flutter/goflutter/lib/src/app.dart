import 'package:flutter/material.dart';

import 'bridge.dart';
import 'host.dart';
import 'runtime.dart';

/// GoApp puts the tree of 1 Go library on the screen.
///
///     void main() {
///       runApp(MaterialApp(home: GoApp(bridge: CounterBridge())));
///     }
///
/// It runs the init of the Go library, it gives the theme and the media query
/// to the Go build, and it rebuilds each boundary that the Go graph marks.
final class GoApp extends StatefulWidget {
  final GoBridge bridge;

  const GoApp({super.key, required this.bridge});

  @override
  State<GoApp> createState() => _GoAppState();
}

class _GoAppState extends State<GoApp> {
  late final GoRuntime _rt;

  @override
  void initState() {
    super.initState();
    _rt = GoRuntime(widget.bridge);
    _rt.onRoot = () {
      if (mounted) setState(() {});
    };
    _rt.start();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final ThemeData t = Theme.of(context);
    final MediaQueryData m = MediaQuery.of(context);
    widget.bridge.setTheme(
      dark: t.brightness == Brightness.dark,
      primary: t.colorScheme.primary.toARGB32(),
      surface: t.colorScheme.surface.toARGB32(),
      onSurface: t.colorScheme.onSurface.toARGB32(),
      textSize: t.textTheme.bodyMedium?.fontSize ?? 14,
    );
    widget.bridge.setMedia(
      width: m.size.width,
      height: m.size.height,
      textScale: m.textScaler.scale(1),
      topInset: m.padding.top,
    );
  }

  @override
  void dispose() {
    _rt.stop();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return goRender(widget.bridge.build(), _rt);
  }
}

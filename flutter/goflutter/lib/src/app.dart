import 'package:material_ui/material_ui.dart';

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
///
/// The Go tree can also hold the whole application:
///
///     void main() {
///       runApp(GoApp(bridge: CompassBridge()));
///     }
///
/// The root node of Go is then material.MaterialApp, and the Go program names
/// its own theme with ui.UseTheme. This widget reads the brightness of the
/// platform, because it stands above the MaterialApp of the tree.
final class GoApp extends StatefulWidget {
  final GoBridge bridge;

  const GoApp({super.key, required this.bridge});

  @override
  State<GoApp> createState() => _GoAppState();
}

class _GoAppState extends State<GoApp> with TickerProviderStateMixin {
  late final GoRuntime _rt;

  @override
  void initState() {
    super.initState();
    _rt = GoRuntime(widget.bridge);
    // A ui.AnimationKey names an AnimationController, and such a controller
    // needs a ticker. The ticker of this State stops when the host leaves the
    // screen.
    _rt.identities.vsync = this;
    _rt.onRoot = () {
      if (mounted) setState(() {});
    };
    _rt.start();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    // The Go tree can hold the MaterialApp itself, thus this widget can stand
    // above each inherited widget of Flutter. The 2 reads below then take the
    // value of the view, and the theme takes the default of Material.
    final ThemeData t = Theme.of(context);
    final MediaQueryData? inherited = MediaQuery.maybeOf(context);
    final MediaQueryData m =
        inherited ?? MediaQueryData.fromView(View.of(context));
    // No MediaQuery stands above this widget when the Go tree holds the
    // MaterialApp. Theme.of then gives the default of Material, thus the
    // brightness comes from the platform.
    final bool dark = inherited != null
        ? t.brightness == Brightness.dark
        : m.platformBrightness == Brightness.dark;
    widget.bridge.setTheme(
      dark: dark,
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
      bottomInset: m.viewInsets.bottom,
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

import 'dart:ui' show PointerDeviceKind;

import 'package:flutter_localizations/flutter_localizations.dart'
    show GlobalWidgetsLocalizations, GlobalCupertinoLocalizations;
import 'package:goflutter/goflutter.dart';
import 'package:material_ui/material_ui.dart';
import 'package:share_plus/share_plus.dart';

import 'bridge.dart';
import 'host.dart';
import 'theme/theme.dart';
import 'widgets_gen.dart';

void main() {
  installCompass();
  runApp(const CompassApp());
}

void installCompass() {
  goGenInstall();
  goSetWidgets(
    (node, runtime) =>
        compassWidget(node, runtime) ?? goGenWidget(node, runtime),
  );
  goSetCall(compassCall);
}

final shareButtonKey = GlobalKey();

Future<String> compassCall(String name, String argument) async {
  if (name == 'share') {
    final box = shareButtonKey.currentContext!.findRenderObject()! as RenderBox;
    await SharePlus.instance.share(
      ShareParams(
        text: argument,
        sharePositionOrigin: box.localToGlobal(Offset.zero) & box.size,
      ),
    );
    return '';
  }
  return goHostCall(name, argument);
}

class CompassApp extends StatelessWidget {
  const CompassApp({super.key});

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'Compass v2',
    localizationsDelegates: const [
      GlobalWidgetsLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    scrollBehavior: const CompassScrollBehavior(),
    theme: AppTheme.lightTheme,
    darkTheme: AppTheme.darkTheme,
    themeMode: ThemeMode.system,
    home: const GoApp(bridge: CompassBridge()),
  );
}

class CompassScrollBehavior extends MaterialScrollBehavior {
  const CompassScrollBehavior();

  @override
  Set<PointerDeviceKind> get dragDevices => {
    PointerDeviceKind.touch,
    PointerDeviceKind.mouse,
  };
}

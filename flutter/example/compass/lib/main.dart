import 'package:flutter/material.dart';
import 'package:goflutter/goflutter.dart';

import 'bridge.dart';
import 'widgets_gen.dart';

void main() {
  goSetWidgets(goGenWidget);
  runApp(const CompassApp());
}

/// CompassApp holds the Material theme. Each widget under it comes from Go.
class CompassApp extends StatelessWidget {
  const CompassApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'compass',
      theme: ThemeData(colorSchemeSeed: const Color(0xFF3F51B5), useMaterial3: true),
      darkTheme: ThemeData(
        colorSchemeSeed: const Color(0xFF3F51B5),
        brightness: Brightness.dark,
        useMaterial3: true,
      ),
      home: const GoApp(bridge: CompassBridge()),
    );
  }
}

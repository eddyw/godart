import 'package:flutter/material.dart';
import 'package:goflutter/goflutter.dart';

import 'bridge.dart';
import 'widgets_gen.dart';

void main() {
  goSetWidgets(goGenWidget);
  runApp(const FormApp());
}

/// FormApp holds the Material theme. Each widget under it comes from Go.
class FormApp extends StatelessWidget {
  const FormApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Go validates a form',
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF00ADD8),
        useMaterial3: true,
      ),
      darkTheme: ThemeData(
        colorSchemeSeed: const Color(0xFF00ADD8),
        brightness: Brightness.dark,
        useMaterial3: true,
      ),
      home: const GoApp(bridge: FormBridge()),
    );
  }
}

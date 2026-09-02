import 'package:flutter/material.dart';
import 'package:goflutter/goflutter.dart';

import 'bridge.dart';
import 'widgets_gen.dart';

void main() {
  goSetWidgets(goGenWidget);
  runApp(const ApiListApp());
}

/// ApiListApp holds the Material theme. Each widget under it comes from Go.
class ApiListApp extends StatelessWidget {
  const ApiListApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'golang/go issues',
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF00ADD8),
        useMaterial3: true,
      ),
      darkTheme: ThemeData(
        colorSchemeSeed: const Color(0xFF00ADD8),
        brightness: Brightness.dark,
        useMaterial3: true,
      ),
      home: const GoApp(bridge: ApiListBridge()),
    );
  }
}

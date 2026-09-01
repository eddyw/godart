import 'package:flutter/material.dart';
import 'package:goflutter/goflutter.dart';

import 'bridge.dart';

void main() {
  runApp(const CounterApp());
}

/// CounterApp holds the Material theme. Each widget under it comes from Go.
class CounterApp extends StatelessWidget {
  const CounterApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Go drives Flutter',
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF00ADD8),
        useMaterial3: true,
      ),
      darkTheme: ThemeData(
        colorSchemeSeed: const Color(0xFF00ADD8),
        brightness: Brightness.dark,
        useMaterial3: true,
      ),
      home: const GoApp(bridge: CounterBridge()),
    );
  }
}

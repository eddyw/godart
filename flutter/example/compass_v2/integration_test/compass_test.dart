import 'dart:io';
import 'dart:ui' show ImageByteFormat;

import 'package:compass_v2/main.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:material_ui/material_ui.dart';

import '../test/flow.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  for (final brightness in Brightness.values) {
    testWidgets('native trip in ${brightness.name} theme', (tester) async {
      tester.binding.platformDispatcher.platformBrightnessTestValue =
          brightness;
      addTearDown(
        tester.binding.platformDispatcher.clearPlatformBrightnessTestValue,
      );
      installCompass();
      final key = GlobalKey();
      await tester.pumpWidget(
        RepaintBoundary(key: key, child: const CompassApp()),
      );
      final directory = Directory(
        '${Directory.systemTemp.path}/compass-v2-native',
      );
      await directory.create(recursive: true);
      await tripFlow(
        tester,
        capture: (screen) async {
          await tester.runAsync(
            () => Future<void>.delayed(const Duration(milliseconds: 300)),
          );
          await tester.pumpAndSettle();
          final boundary =
              key.currentContext!.findRenderObject()! as RenderRepaintBoundary;
          final image = await boundary.toImage();
          final bytes = await image.toByteData(format: ImageByteFormat.png);
          await File('${directory.path}/${brightness.name}-$screen.png')
              .writeAsBytes(bytes!.buffer.asUint8List());
          image.dispose();
        },
      );
      // The native app writes inside its platform sandbox.
      // ignore: avoid_print
      print('Native captures: ${directory.path}');
    });
  }
}

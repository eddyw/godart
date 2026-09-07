import 'dart:io';
import 'dart:ui' show ImageByteFormat;

import 'package:compass_v2/main.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../test/flow.dart';
import '../test/fonts.dart';
import '../test/images.dart';

void main() {
  for (final size in [const Size(430, 932), const Size(800, 900)]) {
    for (final brightness in Brightness.values) {
      final name = '${size.width.toInt()}-${brightness.name}';
      testWidgets('capture $name', (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = size;
        tester.binding.platformDispatcher.platformBrightnessTestValue =
            brightness;
        addTearDown(tester.view.reset);
        addTearDown(
          tester.binding.platformDispatcher.clearPlatformBrightnessTestValue,
        );
        await loadCompassFonts();
        installCompass();
        installImageFixture();
        final key = GlobalKey();
        await tester.pumpWidget(
          RepaintBoundary(key: key, child: const CompassApp()),
        );
        await tripFlow(
          tester,
          capture: (screen) async {
            for (var i = 0; i < 5; i++) {
              await tester.runAsync(
                () => Future<void>.delayed(const Duration(milliseconds: 20)),
              );
              await tester.pump();
            }
            await tester.pumpAndSettle();
            await tester.runAsync(() async {
              final boundary =
                  key.currentContext!.findRenderObject()!
                      as RenderRepaintBoundary;
              final image = await boundary.toImage();
              final bytes = await image.toByteData(format: ImageByteFormat.png);
              final file = File('doc/capture/$name-$screen.png');
              file.parent.createSync(recursive: true);
              file.writeAsBytesSync(bytes!.buffer.asUint8List());
              image.dispose();
            });
          },
        );
        await tester.pumpWidget(const SizedBox());
        await tester.pumpAndSettle();
      });
    }
  }
}

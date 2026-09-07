import 'dart:ui' show PointerDeviceKind;

import 'package:compass_v2/main.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import 'flow.dart';
import 'fonts.dart';
import 'images.dart';

void main() {
  testWidgets('mouse carousel, live theme, and share request', (tester) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(800, 900);
    tester.binding.platformDispatcher.platformBrightnessTestValue =
        Brightness.light;
    addTearDown(tester.view.reset);
    addTearDown(
      tester.binding.platformDispatcher.clearPlatformBrightnessTestValue,
    );
    await loadCompassFonts();
    installCompass();
    installImageFixture();
    await tester.pumpWidget(const CompassApp());
    await tapAndSettle(tester, find.text('Login'));
    await tapAndSettle(tester, find.byKey(const ValueKey('booking-button')));
    await tester.drag(
      find.byType(ListView),
      const Offset(-800, 0),
      kind: PointerDeviceKind.mouse,
    );
    await tester.pumpAndSettle();
    expect(
      find.byKey(const ValueKey('Australia')).hitTestable(),
      findsOneWidget,
    );
    await tapAndSettle(tester, find.byKey(const ValueKey('Australia')));
    tester.binding.platformDispatcher.platformBrightnessTestValue =
        Brightness.dark;
    await tester.pumpAndSettle();
    final context = tester.element(find.text('Who'));
    expect(Theme.of(context).brightness, Brightness.dark);
    expect(find.text('Australia'), findsOneWidget);
    await tapAndSettle(tester, find.byKey(const ValueKey('home-button')));
    await tapAndSettle(tester, find.text('Alaska, North America'));
    MethodCall? request;
    const channel = MethodChannel('dev.fluttercommunity.plus/share');
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (
      call,
    ) async {
      request = call;
      return 'success';
    });
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        channel,
        null,
      ),
    );
    await tapAndSettle(tester, find.byKey(const ValueKey('share-button')));
    expect(request!.method, 'share');
    final arguments = request!.arguments as Map;
    expect(
      arguments['text'],
      startsWith('Trip to Alaska\non 1 Jan - 1 Feb\nActivities:\n'),
    );
    expect(arguments['originWidth'], greaterThan(0));
    expect(arguments['originHeight'], greaterThan(0));
    expect(find.text('Error while sharing booking'), findsNothing);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  for (final brightness in Brightness.values) {
    testWidgets('complete trip in ${brightness.name} theme', (tester) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(430, 932);
      tester.binding.platformDispatcher.platformBrightnessTestValue =
          brightness;
      addTearDown(tester.view.reset);
      addTearDown(
        tester.binding.platformDispatcher.clearPlatformBrightnessTestValue,
      );
      await loadCompassFonts();
      installCompass();
      installImageFixture();
      await tester.pumpWidget(const CompassApp());
      await tripFlow(tester);
      final context = tester.element(find.byType(TextField).first);
      expect(Theme.of(context).brightness, brightness);
      expect(Theme.of(context).textTheme.bodyMedium!.fontSize, 16);
      expect(
        Theme.of(context).colorScheme.primary.toARGB32(),
        brightness == Brightness.dark ? 0xFFFFF7FA : 0xFF101010,
      );
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }
}

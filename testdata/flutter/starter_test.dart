import 'package:flutter_test/flutter_test.dart';
import 'package:goruntime/goruntime.dart';
import 'package:gostarter/dep/gart.dev/ui.dart' as ui;
import 'package:material_ui/material_ui.dart';
import 'package:starter_app/bridge.dart';
import 'package:starter_app/main.dart' as starter;

void main() {
  testWidgets('starter counter and host values reach Go', (tester) async {
    starter.main();
    await tester.pumpAndSettle();
    expect(find.text('This is Go'), findsOneWidget);
    expect(find.text('0'), findsOneWidget);
    expect(tester.widget<Text>(find.text('0')).style?.fontSize, 48);

    await tester.tap(find.byType(FloatingActionButton));
    await tester.pumpAndSettle();
    expect(find.text('1'), findsOneWidget);

    const bridge = StarterBridge();
    bridge.setTheme(
      dark: true,
      primary: 0xff112233,
      surface: 0xff445566,
      onSurface: 0xff778899,
      textSize: 17.5,
    );
    final theme = ui.Theme();
    expect(theme.Dark, true);
    expect($ix(theme.Colors.Primary), 0xff112233);
    expect($ix(theme.Colors.Surface), 0xff445566);
    expect($ix(theme.Colors.OnSurface), 0xff778899);
    expect(theme.TextSize.toDouble(), 17.5);

    bridge.setMedia(
      width: 480.5,
      height: 640.5,
      textScale: 1.25,
      topInset: 24.5,
      bottomInset: 120.5,
    );
    final media = ui.Media();
    expect(media.Width.toDouble(), 480.5);
    expect(media.Height.toDouble(), 640.5);
    expect(media.TextScale.toDouble(), 1.25);
    expect(media.TopInset.toDouble(), 24.5);
    expect(media.BottomInset.toDouble(), 120.5);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}

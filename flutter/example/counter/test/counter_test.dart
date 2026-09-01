import 'package:counter_app/bridge.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gocounter/app.dart' as app;
import 'package:goflutter/goflutter.dart';

Widget harness() {
  return const MaterialApp(home: GoApp(bridge: CounterBridge()));
}

void main() {
  setUp(() {
    app.$init();
    app.Restart();
  });

  testWidgets('the Go tree builds real Flutter widgets', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    expect(find.text('Go drives Flutter'), findsOneWidget);
    expect(find.text('Hello'), findsOneWidget);
    expect(find.text('count 0'), findsOneWidget);
    expect(find.text('press refresh to load 100,000 rows'), findsOneWidget);
    expect(find.byType(Scaffold), findsOneWidget);
    expect(find.byType(AppBar), findsOneWidget);
    expect(find.byType(TextField), findsOneWidget);
    expect(find.byType(Switch), findsOneWidget);
  });

  testWidgets('a Go closure drives the count', (WidgetTester tester) async {
    await tester.pumpWidget(harness());

    await tester.tap(find.byTooltip('count 1 up'));
    await tester.pump();
    expect(find.text('count 1'), findsOneWidget);

    await tester.tap(find.byTooltip('count 1 up'));
    await tester.tap(find.byTooltip('count 1 up'));
    await tester.pump();
    expect(find.text('count 3'), findsOneWidget);

    await tester.tap(find.text('reset'));
    await tester.pump();
    expect(find.text('count 0'), findsOneWidget);
  });

  testWidgets('a text field keeps its focus across a Go rebuild', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    await tester.tap(find.byType(TextField));
    await tester.pump();
    expect(
      FocusManager.instance.primaryFocus?.hasFocus,
      isTrue,
      reason: 'the field must take the focus',
    );

    await tester.enterText(find.byType(TextField), 'Ada');
    await tester.pump();
    expect(find.text('Hello Ada'), findsOneWidget);
    expect(
      FocusManager.instance.primaryFocus?.hasFocus,
      isTrue,
      reason: 'the field must keep the focus after the Go rebuild',
    );

    final EditableText field = tester.widget<EditableText>(
      find.byType(EditableText),
    );
    expect(field.controller.text, 'Ada');
  });

  testWidgets('a goroutine fills a lazy list of 100,000 rows', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    await tester.tap(find.byTooltip('load the list'));
    await tester.pump();
    expect(find.byType(CircularProgressIndicator), findsOneWidget);

    await tester.pump(const Duration(milliseconds: 250));
    await tester.pump();

    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(find.text('row 0'), findsOneWidget);
    expect(find.byType(ListView), findsOneWidget);

    // The list is lazy. Flutter builds the visible rows only, thus a row far
    // down the list holds no widget.
    expect(find.text('row 900'), findsNothing);

    await tester.drag(find.byType(ListView), const Offset(0, -400));
    await tester.pump();
    expect(find.text('row 0'), findsNothing);
    expect(find.byType(Card), findsWidgets);
  });

  testWidgets('a Go error reaches the screen', (WidgetTester tester) async {
    await tester.pumpWidget(harness());

    await tester.tap(find.byType(Switch));
    await tester.pump();

    await tester.tap(find.byTooltip('load the list'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 250));
    await tester.pump();

    expect(find.text('error: the server gave 503'), findsOneWidget);
  });

  testWidgets('one signal rebuilds one boundary', (WidgetTester tester) async {
    await tester.pumpWidget(harness());

    final Element greeting = tester.element(find.text('Hello'));
    final Element count = tester.element(find.text('count 0'));

    await tester.tap(find.byTooltip('count 1 up'));
    await tester.pump();

    // The count boundary rebuilt, and the greeting boundary kept its element.
    expect(find.text('count 1'), findsOneWidget);
    expect(tester.element(find.text('Hello')), same(greeting));
    expect(count.mounted, isTrue);
  });
}

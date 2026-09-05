import 'dart:convert';
import 'dart:io';

import 'package:material_ui/material_ui.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:form_app/bridge.dart';
import 'package:form_app/widgets_gen.dart';
import 'package:goflutter/goflutter.dart';
import 'package:goform/app.dart' as app;
import 'package:goruntime/goruntime.dart';

Widget harness() {
  return const MaterialApp(home: GoApp(bridge: FormBridge()));
}

void main() {
  late HttpServer server;
  late List<Map<String, Object?>> sent;
  late int status;
  late String body;

  setUp(() async {
    sent = <Map<String, Object?>>[];
    status = HttpStatus.created;
    body = '{"id":42,"received":"2026-09-02T10:00:00Z"}';
    server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    server.listen((HttpRequest request) async {
      final String text = await utf8.decoder.bind(request).join();
      sent.add(jsonDecode(text) as Map<String, Object?>);
      request.response
        ..statusCode = status
        ..headers.contentType = ContentType.json
        ..write(body);
      await request.response.close();
    });

    goSetWidgets(goGenWidget);
    app.$init();
    app.Restart();
    app.SetBaseURL(GoString.fromString('http://127.0.0.1:${server.port}'));
  });

  tearDown(() async {
    await server.close(force: true);
  });

  /// settle runs the real socket work of the Go goroutine. A widget test holds
  /// a fake clock, thus the loop gives time to the real zone and to the fake
  /// zone in turn.
  Future<void> settle(WidgetTester tester) async {
    for (int i = 0; i < 200 && !app.Sent(); i++) {
      await tester.runAsync(
        () => Future<void>.delayed(const Duration(milliseconds: 10)),
      );
      await tester.pump();
    }
    await tester.pump();
  }

  Future<void> fill(WidgetTester tester, String name, String email,
      String message) async {
    final Finder fields = find.byType(TextField);
    await tester.enterText(fields.at(0), name);
    await tester.pump();
    await tester.enterText(fields.at(1), email);
    await tester.pump();
    await tester.enterText(fields.at(2), message);
    await tester.pump();
  }

  testWidgets('the first screen holds 3 fields and no report', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    expect(find.text('Go validates a form'), findsOneWidget);
    expect(find.byType(TextField), findsNWidgets(3));
    expect(find.text('send'), findsOneWidget);
    expect(find.textContaining('the name is empty'), findsNothing);
    expect(find.text('the form holds an error'), findsNothing);
  });

  testWidgets('an invalid form reports each field and reaches no server', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    await tester.tap(find.text('send'));
    await tester.pump();

    expect(find.text('the name is empty'), findsOneWidget);
    expect(find.text('the address is empty'), findsOneWidget);
    expect(find.text('the message is empty'), findsOneWidget);
    expect(find.text('the form holds an error'), findsOneWidget);
    expect(sent, isEmpty);
  });

  testWidgets('a keystroke clears the report of its own field', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());
    await tester.tap(find.text('send'));
    await tester.pump();

    await tester.enterText(find.byType(TextField).at(0), 'Ada');
    await tester.pump();

    expect(find.text('the name is empty'), findsNothing);
    expect(find.text('the address is empty'), findsOneWidget);

    await tester.enterText(find.byType(TextField).at(1), 'ada@example');
    await tester.pump();
    expect(find.text('the address has no valid form'), findsOneWidget);
  });

  testWidgets('a valid form sends the JSON and shows the answer', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());
    await fill(tester, 'Ada', 'ada@example.com', 'the machine weaves');

    await tester.tap(find.text('send'));
    await tester.pump();
    expect(find.byType(CircularProgressIndicator), findsOneWidget);

    await settle(tester);

    expect(sent, <Map<String, Object?>>[
      <String, Object?>{
        'name': 'Ada',
        'email': 'ada@example.com',
        'message': 'the machine weaves',
      },
    ]);
    expect(find.text('the server took the message'), findsOneWidget);
    expect(find.text('number 42, 2026-09-02'), findsOneWidget);
    expect(find.byType(TextField), findsNothing);
  });

  testWidgets('a status error reaches the screen and keeps the form', (
    WidgetTester tester,
  ) async {
    status = HttpStatus.internalServerError;
    body = '';
    await tester.pumpWidget(harness());
    await fill(tester, 'Ada', 'ada@example.com', 'the machine weaves');

    await tester.tap(find.text('send'));
    await tester.pump();
    await settle(tester);

    expect(
      find.text('error: the server gave 500 Internal Server Error'),
      findsOneWidget,
    );
    expect(find.byType(TextField), findsNWidgets(3));
  });

  testWidgets('a Go ButtonStyle reaches the Flutter button', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    final FilledButton button = tester.widget<FilledButton>(
      find.byType(FilledButton),
    );
    final ButtonStyle style = button.style!;
    const Set<WidgetState> idle = <WidgetState>{};
    const Set<WidgetState> off = <WidgetState>{WidgetState.disabled};

    // ui.ButtonStyle, ui.StateColor, ui.Size, and ui.Shape each reach the
    // ButtonStyle of Flutter.
    expect(style.backgroundColor!.resolve(idle), const Color(0xff1a73e8));
    expect(style.backgroundColor!.resolve(off), const Color(0xffbdbdbd));
    expect(style.foregroundColor!.resolve(idle), const Color(0xffffffff));
    expect(
      style.padding!.resolve(idle),
      const EdgeInsets.fromLTRB(24, 14, 24, 14),
    );
    expect(style.minimumSize!.resolve(idle), const Size(0, 48));
    expect(style.textStyle!.resolve(idle)!.fontWeight, FontWeight.w600);
    final RoundedRectangleBorder shape =
        style.shape!.resolve(idle)! as RoundedRectangleBorder;
    expect(shape.borderRadius, BorderRadius.circular(8));
  });

  testWidgets('a text field keeps its focus and its text across a rebuild', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    await tester.tap(find.byType(TextField).at(1));
    await tester.pump();
    await tester.enterText(find.byType(TextField).at(1), 'ada@example.com');
    await tester.pump();

    expect(
      FocusManager.instance.primaryFocus?.hasFocus,
      isTrue,
      reason: 'the field must keep the focus after the Go rebuild',
    );
    final EditableText field = tester.widget<EditableText>(
      find.byType(EditableText).at(1),
    );
    expect(field.controller.text, 'ada@example.com');
  });
}

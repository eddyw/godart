import 'dart:io';

import 'package:apilist_app/bridge.dart';
import 'package:apilist_app/widgets_gen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:goapilist/app.dart' as app;
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';

const String twoIssues = '''
[
  {"number":71234,"title":"spec: clarify the range over a function",
   "state":"open","created_at":"2026-03-04T09:15:00Z",
   "user":{"login":"griesemer"}},
  {"number":71987,"title":"net/http: the client drops a trailer",
   "state":"closed","created_at":"2026-05-21T17:40:00Z",
   "user":{"login":"neild"}}
]''';

Widget harness() {
  return const MaterialApp(home: GoApp(bridge: ApiListBridge()));
}

void main() {
  late HttpServer server;
  late List<String> seen;
  late int status;
  late String body;

  setUp(() async {
    seen = <String>[];
    status = HttpStatus.ok;
    body = twoIssues;
    server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    server.listen((HttpRequest request) async {
      seen.add(request.uri.toString());
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

  /// settle runs the real socket work of the Go goroutine.
  ///
  /// A widget test holds a fake clock. The socket of dart:io completes in the
  /// real zone, and the Go scheduler resumes in the fake zone, thus the loop
  /// gives time to each one.
  Future<void> settle(WidgetTester tester) async {
    for (int i = 0; i < 200 && !app.Loaded(); i++) {
      await tester.runAsync(
        () => Future<void>.delayed(const Duration(milliseconds: 10)),
      );
      await tester.pump();
    }
    await tester.pump();
  }

  testWidgets('the first screen asks for a refresh', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    expect(find.text('golang/go issues'), findsOneWidget);
    expect(find.text('press refresh to read the API'), findsOneWidget);
    expect(find.byType(Scaffold), findsOneWidget);
    expect(find.byType(AppBar), findsOneWidget);
    expect(seen, isEmpty);
  });

  testWidgets('a real HTTP GET fills the list', (WidgetTester tester) async {
    await tester.pumpWidget(harness());

    await tester.tap(find.byTooltip('read the API'));
    await tester.pump();
    expect(find.byType(CircularProgressIndicator), findsOneWidget);

    await settle(tester);

    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(find.byType(ListView), findsOneWidget);
    expect(seen, <String>['/repos/golang/go/issues?per_page=20']);
    expect(
      find.text('spec: clarify the range over a function'),
      findsOneWidget,
    );
    expect(find.text('net/http: the client drops a trailer'), findsOneWidget);
    expect(find.byType(Card), findsNWidgets(2));
  });

  testWidgets('the row shows each decoded field', (WidgetTester tester) async {
    await tester.pumpWidget(harness());
    await tester.tap(find.byTooltip('read the API'));
    await tester.pump();
    await settle(tester);

    // The number, the state, the author, and the date of the JSON answer. The
    // date proves that encoding/json read the time.Time field.
    expect(find.text('#71234  open  griesemer  2026-03-04'), findsOneWidget);
    expect(find.text('#71987  closed  neild  2026-05-21'), findsOneWidget);
  });

  testWidgets('a status error reaches the screen, and a retry loads', (
    WidgetTester tester,
  ) async {
    status = HttpStatus.serviceUnavailable;
    body = '';
    await tester.pumpWidget(harness());

    await tester.tap(find.byTooltip('read the API'));
    await tester.pump();
    await settle(tester);

    expect(
      find.text('error: the server gave 503 Service Unavailable'),
      findsOneWidget,
    );

    status = HttpStatus.ok;
    body = twoIssues;
    await tester.tap(find.text('try again'));
    await tester.pump();
    await settle(tester);

    expect(find.byType(Card), findsNWidgets(2));
    expect(seen.length, 2);
  });

  testWidgets('the load rebuilds the body boundary alone', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());
    final Element bar = tester.element(find.text('golang/go issues'));

    await tester.tap(find.byTooltip('read the API'));
    await tester.pump();
    await settle(tester);

    expect(find.byType(Card), findsNWidgets(2));
    expect(tester.element(find.text('golang/go issues')), same(bar));
  });
}

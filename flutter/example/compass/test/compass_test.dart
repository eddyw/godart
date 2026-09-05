import 'dart:convert';
import 'dart:io';

import 'package:compass_app/bridge.dart';
import 'package:compass_app/widgets_gen.dart';
import 'package:material_ui/material_ui.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gocompass/app.dart' as app;
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';

/// The whole application runs in a real Flutter tree. Go holds each screen,
/// each read of the server, and the stack of the navigator.

const String token =
    ' e1c37dfd973353b78bb71df050e2c6e72d53034e148920383968ae49b96f1fd2';

/// _pixel holds a transparent PNG of 1 pixel. A destination carries the URL
/// of a picture, and the HTTP client of a widget test answers 400. The
/// override below gives this pixel to each such request, thus no image error
/// reaches the test. The Go runtime library opens a socket of its own, thus
/// the override does not reach the application.
final Uint8List _pixel = base64Decode(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk'
  'YPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
);

class _ImageOverrides extends HttpOverrides {
  @override
  HttpClient createHttpClient(SecurityContext? context) => _ImageClient();
}

class _ImageClient implements HttpClient {
  @override
  bool autoUncompress = true;
  @override
  Duration idleTimeout = const Duration(seconds: 15);
  @override
  Duration? connectionTimeout;
  @override
  int? maxConnectionsPerHost;
  @override
  String? userAgent;

  @override
  Future<HttpClientRequest> getUrl(Uri url) async => _ImageRequest(url);

  @override
  void close({bool force = false}) {}

  @override
  dynamic noSuchMethod(Invocation invocation) =>
      super.noSuchMethod(invocation);
}

class _ImageRequest implements HttpClientRequest {
  _ImageRequest(this.uri);

  @override
  final Uri uri;
  @override
  final HttpHeaders headers = _ImageHeaders();

  @override
  Future<HttpClientResponse> close() async => _ImageResponse();

  @override
  dynamic noSuchMethod(Invocation invocation) =>
      super.noSuchMethod(invocation);
}

class _ImageHeaders implements HttpHeaders {
  @override
  dynamic noSuchMethod(Invocation invocation) =>
      super.noSuchMethod(invocation);
}

class _ImageResponse implements HttpClientResponse {
  @override
  int get statusCode => HttpStatus.ok;
  @override
  int get contentLength => _pixel.length;
  @override
  HttpClientResponseCompressionState get compressionState =>
      HttpClientResponseCompressionState.notCompressed;

  @override
  StreamSubscription<List<int>> listen(
    void Function(List<int> event)? onData, {
    Function? onError,
    void Function()? onDone,
    bool? cancelOnError,
  }) {
    return Stream<List<int>>.value(_pixel).listen(
      onData,
      onError: onError,
      onDone: onDone,
      cancelOnError: cancelOnError,
    );
  }

  @override
  dynamic noSuchMethod(Invocation invocation) =>
      super.noSuchMethod(invocation);
}

Widget harness() {
  return const MaterialApp(home: GoApp(bridge: CompassBridge()));
}

void main() {
  late HttpServer server;
  late List<String> seen;
  late List<Map<String, Object?>> posted;
  late bool deleted;

  setUp(() async {
    HttpOverrides.global = _ImageOverrides();
    seen = <String>[];
    posted = <Map<String, Object?>>[];
    deleted = false;
    server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    server.listen((HttpRequest request) async {
      final String path = request.uri.path;
      seen.add('${request.method} $path');
      Object? body;
      int status = HttpStatus.ok;
      if (path == '/login') {
        body = <String, Object?>{'token': token, 'userId': '123'};
      } else if (path == '/user') {
        body = <String, Object?>{
          'id': '123',
          'name': 'Sofie',
          'email': 'email@example.com',
          'picture': 'assets/user.jpg',
        };
      } else if (path == '/continent') {
        body = <Object>[
          <String, Object?>{'name': 'Europe', 'imageUrl': 'http://x/e.png'},
          <String, Object?>{'name': 'Asia', 'imageUrl': 'http://x/a.png'},
        ];
      } else if (path == '/destination') {
        body = <Object>[
          <String, Object?>{
            'ref': 'alps',
            'name': 'Alps',
            'country': 'Switzerland',
            'continent': 'Europe',
            'knownFor': 'snow',
            'tags': <String>['Mountain'],
            'imageUrl': 'http://x/alps.png',
          },
          <String, Object?>{
            'ref': 'kyoto',
            'name': 'Kyoto',
            'country': 'Japan',
            'continent': 'Asia',
            'knownFor': 'temples',
            'tags': <String>['City'],
            'imageUrl': 'http://x/kyoto.png',
          },
        ];
      } else if (path == '/destination/alps/activity') {
        body = <Object>[
          <String, Object?>{
            'ref': 'ski',
            'destinationRef': 'alps',
            'name': 'Ski',
            'description': 'snow',
            'locationName': 'Zermatt',
            'duration': 6,
            'timeOfDay': 'morning',
            'familyFriendly': true,
            'price': 3,
            'imageUrl': 'http://x/ski.png',
          },
          <String, Object?>{
            'ref': 'fondue',
            'destinationRef': 'alps',
            'name': 'Fondue',
            'description': 'cheese',
            'locationName': 'Zermatt',
            'duration': 2,
            'timeOfDay': 'evening',
            'familyFriendly': true,
            'price': 2,
            'imageUrl': 'http://x/fondue.png',
          },
        ];
      } else if (path == '/booking' && request.method == 'GET') {
        body = deleted
            ? <Object>[]
            : <Object>[
          <String, Object?>{
            'id': 0,
            'name': 'Alps, Europe',
            'startDate': '2024-07-20T00:00:00Z',
            'endDate': '2024-07-25T00:00:00Z',
            'destinationRef': 'alps',
            'activitiesRef': <String>['ski'],
          },
        ];
      } else if (request.method == 'DELETE') {
        deleted = true;
        status = HttpStatus.noContent;
        body = null;
      } else if (path == '/booking' && request.method == 'POST') {
        final String text = await utf8.decoder.bind(request).join();
        posted.add(jsonDecode(text) as Map<String, Object?>);
        status = HttpStatus.created;
        body = jsonDecode(text);
        (body as Map<String, Object?>)['id'] = 7;
      } else {
        body = <Object>[];
      }
      request.response
        ..statusCode = status
        ..headers.contentType = ContentType.json
        ..write(jsonEncode(body));
      await request.response.close();
    });

    goSetWidgets(goGenWidget);
    app.$init();
    app.SetBaseURL(
      GoString.fromString('http://127.0.0.1:${server.port}'),
    );
    app.Reset();
  });

  tearDown(() async {
    await server.close(force: true);
  });

  /// settle runs the real socket work of the Go goroutine.
  ///
  /// A widget test holds a fake clock. The socket of dart:io completes in the
  /// real zone, and the Go scheduler resumes in the fake zone, thus the loop
  /// gives time to each one.
  Future<void> settle(WidgetTester tester, String screen) async {
    for (int i = 0; i < 300; i++) {
      if (app.Top().toDartString() == screen && app.Settled()) break;
      await tester.runAsync(
        () => Future<void>.delayed(const Duration(milliseconds: 10)),
      );
      await tester.pump();
    }
    await tester.pump();
    // The navigator ignores a pointer while a page transition runs, thus the
    // test waits for the end of that animation.
    await tester.pumpAndSettle();
  }

  Future<void> logIn(WidgetTester tester) async {
    await tester.pumpWidget(harness());
    await tester.tap(find.text('Login'));
    await tester.pump();
    await settle(tester, 'home');
  }

  testWidgets('the first screen holds the login form', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    expect(find.text('compass'), findsOneWidget);
    expect(find.byType(TextField), findsNWidgets(2));
    expect(find.text('Login'), findsOneWidget);
    expect(seen, isEmpty);
  });

  testWidgets('the login opens the home with the bookings', (
    WidgetTester tester,
  ) async {
    await logIn(tester);

    expect(seen, contains('POST /login'));
    expect(seen, contains('GET /booking'));
    expect(seen, contains('GET /user'));
    expect(find.text("Sofie's Trips"), findsOneWidget);
    expect(find.text('Alps, Europe'), findsOneWidget);
    expect(find.text('20 - 25 Jul'), findsOneWidget);
    expect(find.text('Book New Trip'), findsOneWidget);
  });

  testWidgets('the whole flow reaches a new booking', (
    WidgetTester tester,
  ) async {
    await logIn(tester);

    await tester.tap(find.text('Book New Trip'));
    await tester.pump();
    await settle(tester, 'search');
    expect(find.text('Europe'), findsOneWidget);
    expect(find.text('Add Dates'), findsOneWidget);

    await tester.tap(find.text('Europe'));
    await tester.pump();

    // The calendar of Flutter stands on a screen of its own, and Go holds the
    // 2 dates. The first tap names the first day, and the second one names the
    // last day and pops the screen.
    await tester.tap(find.text('Add Dates'));
    await tester.pump();
    await settle(tester, 'dates');
    expect(find.byType(CalendarDatePicker), findsOneWidget);
    expect(find.text('Choose the first day'), findsOneWidget);
    // The calendar opens on the month of today, and a day before today takes
    // no tap. The next month lies whole in the future, thus each day of it
    // takes one.
    await tester.tap(find.byTooltip('Next month'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('10').first);
    await tester.pump();
    expect(find.text('Choose the last day'), findsOneWidget);
    await tester.tap(find.text('11').first);
    await tester.pump();
    await settle(tester, 'search');
    expect(find.text('Add Dates'), findsNothing);

    await tester.tap(find.byTooltip('1 guest more'));
    await tester.pump();
    expect(find.text('1 guest'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Search'));
    await tester.pump();
    await settle(tester, 'results');
    expect(find.text('Alps, Switzerland'), findsNothing);
    expect(find.text('Alps, Europe'), findsOneWidget);
    expect(find.text('Kyoto, Asia'), findsNothing);

    await tester.tap(find.text('Alps, Europe'));
    await tester.pump();
    await settle(tester, 'activities');
    expect(find.text('Daytime'), findsOneWidget);
    expect(find.text('Ski'), findsOneWidget);
    expect(find.text('Evening'), findsOneWidget);
    expect(find.text('Fondue'), findsOneWidget);

    await tester.tap(find.text('Ski'));
    await tester.pump();
    expect(find.text('1 selected'), findsOneWidget);

    await tester.tap(find.text('Confirm'));
    await tester.pump();
    await settle(tester, 'booking');

    expect(posted.length, 1);
    expect(posted.first['destinationRef'], 'alps');
    expect(posted.first['activitiesRef'], <String>['ski']);
    expect(find.text('Your chosen activities'), findsOneWidget);
    expect(app.Top().toDartString(), 'booking');
  });

  testWidgets('the back button of the platform pops 1 screen', (
    WidgetTester tester,
  ) async {
    await logIn(tester);

    await tester.tap(find.text('Book New Trip'));
    await tester.pump();
    await settle(tester, 'search');
    expect(app.Top().toDartString(), 'search');

    final NavigatorState nav = tester.state(find.byType(Navigator).last);
    nav.pop();
    await tester.pumpAndSettle();

    expect(app.Top().toDartString(), 'home');
    expect(find.text("Sofie's Trips"), findsOneWidget);
  });

  testWidgets('the delete removes a booking', (WidgetTester tester) async {
    await logIn(tester);

    await tester.tap(find.byTooltip('delete the trip'));
    await tester.pump();
    // The tap starts a goroutine, and the command reads the list again after
    // the delete. The loop below gives time to the 2 requests.
    for (int i = 0;
        i < 300 &&
            seen.where((String s) => s == 'GET /booking').length < 2;
        i++) {
      await tester.runAsync(
        () => Future<void>.delayed(const Duration(milliseconds: 10)),
      );
      await tester.pump();
    }
    await tester.pumpAndSettle();

    expect(seen, contains('DELETE /booking/0'));
    expect(find.text('Alps, Europe'), findsNothing);
  });
}

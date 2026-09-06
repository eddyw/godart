import 'dart:convert';

import 'package:flutter/services.dart' show FontLoader;
import 'dart:typed_data';
import 'dart:io';

import 'package:compass_app/bridge.dart';
import 'package:compass_app/widgets_gen.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:material_ui/material_ui.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gocompass/app.dart' as app;
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';

/// shot.dart writes 1 picture of each screen into doc/.
///
///     flutter test tool/shot.dart --update-goldens
///
/// It stands outside test/, thus `flutter test` does not run it. A golden file
/// reads the font of the machine, and a second machine would give another
/// picture.
///
/// The test loads the 2 fonts of the application, because a widget test draws
/// each glyph as a black box. The HTTP client of the test gives 1 small
/// picture to each request of an image.

const String token =
    ' e1c37dfd973353b78bb71df050e2c6e72d53034e148920383968ae49b96f1fd2';

/// _pixel holds a transparent PNG of 1 pixel. A destination carries the URL
/// of a picture, and the HTTP client of a widget test answers 400. The
/// override below gives this pixel to each such request, thus no image error
/// reaches the test. The Go runtime library opens a socket of its own, thus
/// the override does not reach the application.
final Uint8List _pixel = base64Decode(
  'iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAIAAABLbSncAAAAEUlEQVR4nGOI'
  'alqFFTEMLQkABQJhgR3TWfAAAAAASUVORK5CYII=',
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
  // The Go tree holds the MaterialApp itself, thus this widget stands above
  // it. Read the Build method of go/app.go.
  return const GoApp(bridge: CompassBridge());
}

/// _nextMonth gives the header of the month after this 1.
String _nextMonth() {
  final DateTime now = DateTime.now();
  final DateTime next = DateTime(now.year, now.month + 1);
  const List<String> months = <String>[
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December',
  ];
  return '${months[next.month - 1]} ${next.year}';
}

void main() {
  late HttpServer server;
  late bool deleted;

  setUp(() async {
    HttpOverrides.global = _ImageOverrides();
    deleted = false;
    server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    server.listen((HttpRequest request) async {
      final String path = request.uri.path;
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
          <String, Object?>{'name': 'Africa', 'imageUrl': 'http://x/f.png'},
        ];
      } else if (path == '/destination') {
        body = <Object>[
          <String, Object?>{
            'ref': 'alps', 'name': 'Alps', 'country': 'Switzerland',
            'continent': 'Europe', 'knownFor': 'The Alps hold snow, a lake, and a long walk over each pass.',
            'tags': <String>['Mountain', 'Hiking', 'Romantic'],
            'imageUrl': 'http://x/alps.png',
          },
          <String, Object?>{
            'ref': 'porto', 'name': 'Porto', 'country': 'Portugal',
            'continent': 'Europe', 'knownFor': 'wine',
            'tags': <String>['City', 'Foodie', 'Wine tasting'],
            'imageUrl': 'http://x/porto.png',
          },
        ];
      } else if (path == '/destination/alps/activity') {
        body = <Object>[
          <String, Object?>{
            'ref': 'ski', 'destinationRef': 'alps', 'name': 'Ski over the glacier of Zermatt',
            'description': 'A long run over the snow.', 'locationName': 'Zermatt',
            'duration': 6, 'timeOfDay': 'morning', 'familyFriendly': true,
            'price': 3, 'imageUrl': 'http://x/ski.png',
          },
          <String, Object?>{
            'ref': 'fondue', 'destinationRef': 'alps', 'name': 'Fondue at the lake',
            'description': 'Cheese and bread.', 'locationName': 'Zermatt',
            'duration': 2, 'timeOfDay': 'evening', 'familyFriendly': true,
            'price': 2, 'imageUrl': 'http://x/fondue.png',
          },
        ];
      } else if (path == '/booking' && request.method == 'GET') {
        body = deleted ? <Object>[] : <Object>[
          <String, Object?>{
            'id': 0, 'name': 'Alps, Europe',
            'startDate': '2024-07-20T00:00:00Z', 'endDate': '2024-07-25T00:00:00Z',
            'destinationRef': 'alps', 'activitiesRef': <String>['ski'],
          },
        ];
      } else if (request.method == 'DELETE') {
        deleted = true;
        status = HttpStatus.noContent;
        body = null;
      } else if (path == '/booking' && request.method == 'POST') {
        final String text = await utf8.decoder.bind(request).join();
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

    goGenInstall();
    app.$init();
    app.SetBaseURL(GoString.fromString('http://127.0.0.1:${server.port}'));
    app.Reset();
  });

  tearDown(() async {
    await server.close(force: true);
  });

  Future<void> settle(WidgetTester tester, String screen) async {
    for (int i = 0; i < 300; i++) {
      if (app.Top().toDartString() == screen && app.Settled()) break;
      await tester.runAsync(
        () => Future<void>.delayed(const Duration(milliseconds: 10)),
      );
      await tester.pump();
    }
    await tester.pump();
    await tester.pumpAndSettle();
  }

  setUpAll(() async {
    // A widget test draws each glyph as a black box, because it holds no real
    // font. The 2 faces of the application take the place of the default, thus
    // the shot shows the text of each screen.
    for (final String name in <String>['Roboto', 'Rubik']) {
      final FontLoader loader = FontLoader(name);
      loader.addFont(
        File('assets/fonts/Rubik.ttf').readAsBytes().then(
          (Uint8List b) => ByteData.view(b.buffer),
        ),
      );
      await loader.load();
    }
    final FontLoader sans = FontLoader('OpenSans');
    sans.addFont(
      File('assets/fonts/OpenSans.ttf').readAsBytes().then(
        (Uint8List b) => ByteData.view(b.buffer),
      ),
    );
    await sans.load();
  });

  /// shot gives each image time to decode, and it then writes the file.
  Future<void> shot(WidgetTester tester, String name) async {
    for (int i = 0; i < 20; i++) {
      await tester.runAsync(
        () => Future<void>.delayed(const Duration(milliseconds: 20)),
      );
      await tester.pump();
    }
    await tester.pumpAndSettle();
    await expectLater(find.byType(GoApp), matchesGoldenFile('../doc/$name.png'));
  }

  testWidgets('the screens of the sample', (WidgetTester tester) async {
    tester.view.physicalSize = const Size(1290, 2796);
    tester.view.devicePixelRatio = 3;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(const GoApp(bridge: CompassBridge()));
    await tester.pumpAndSettle();
    await shot(tester, '1-login');

    await tester.tap(find.text('Login'));
    await tester.pump();
    await settle(tester, 'home');
    await shot(tester, '2-home');

    await tester.tap(find.text('Book New Trip'));
    await tester.pump();
    await settle(tester, 'search');
    await shot(tester, '3-search');

    await tester.tap(find.text('Europe'), warnIfMissed: false);
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey<String>('add-guests')));
    await tester.pump();
    await tester.pumpAndSettle();
    await shot(tester, '4-search-filled');

    await tester.tap(find.text('Add Dates'));
    await tester.pumpAndSettle();
    await shot(tester, '4b-dates');
    await tester.scrollUntilVisible(find.text(_nextMonth()), 200,
        scrollable: find.byType(Scrollable).last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('10').last);
    await tester.pump();
    await tester.tap(find.text('12').last);
    await tester.pump();
    await tester.tap(find.text('Save'));
    await settle(tester, 'search');
    await tester.tap(find.widgetWithText(FilledButton, 'Search'));
    await tester.pump();
    await settle(tester, 'results');
    await shot(tester, '5-results');

    await tester.tap(find.text('ALPS'), warnIfMissed: false);
    await tester.pump();
    await settle(tester, 'activities');
    await shot(tester, '6-activities');

    await tester.tap(find.byKey(const ValueKey<String>('ski-checkbox')));
    await tester.pump();
    await tester.tap(find.text('Confirm'));
    await tester.pump();
    await settle(tester, 'booking');
    await shot(tester, '7-booking');
  });
}

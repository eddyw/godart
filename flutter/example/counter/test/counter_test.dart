import 'package:counter_app/bridge.dart';
import 'package:counter_app/widgets_gen.dart';
import 'package:material_ui/material_ui.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gocounter/app.dart' as app;
import 'package:goflutter/goflutter.dart';

Widget harness() {
  return const MaterialApp(home: GoApp(bridge: CounterBridge()));
}

void main() {
  setUp(() {
    goSetWidgets(goGenWidget);
    goSetCall((String name, String arg) async => '$name: $arg');
    app.$init();
    app.Restart();
  });

  testWidgets('a Go call reaches the host and the answer reaches the screen', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());
    await tester.tap(find.byTooltip('share the greeting'));
    await tester.pumpAndSettle();

    // The Go side parked a goroutine on a channel, the host ran the handler,
    // and the answer woke the goroutine.
    expect(find.text('share: Hello'), findsOneWidget);
  });

  testWidgets('an error of the host reaches the Go side', (
    WidgetTester tester,
  ) async {
    goSetCall((String name, String arg) async => throw StateError('no sheet'));
    await tester.pumpWidget(harness());
    await tester.tap(find.byTooltip('share the greeting'));
    await tester.pumpAndSettle();

    expect(find.textContaining('no sheet'), findsOneWidget);
  });

  testWidgets('the Go tree builds real Flutter widgets', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    expect(find.text('This is Go!'), findsOneWidget);
    expect(find.text('Hello'), findsOneWidget);
    expect(find.text('count 0'), findsOneWidget);
    expect(find.text('press refresh to load 100,000 rows'), findsOneWidget);
    expect(find.byType(Scaffold), findsOneWidget);
    expect(find.byType(AppBar), findsOneWidget);
    expect(find.byType(TextField), findsNWidgets(2));
    expect(find.byType(Switch), findsOneWidget);
    expect(find.text('note idle, button none'), findsOneWidget);
  });

  testWidgets('a Go Decoration reaches the Flutter box', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    final Container box = tester.widget<Container>(
      find
          .ancestor(of: find.text('Hello'), matching: find.byType(Container))
          .first,
    );
    final BoxDecoration paint = box.decoration! as BoxDecoration;

    // ui.Decoration, ui.Gradient, ui.Shadow, and ui.ImageSource each reach the
    // BoxDecoration of Flutter.
    expect(paint.borderRadius, BorderRadius.circular(12));
    expect(paint.border, const Border.fromBorderSide(
      BorderSide(color: Color(0xff90a4ae)),
    ));
    final LinearGradient fade = paint.gradient! as LinearGradient;
    expect(fade.colors, const <Color>[Color(0xffe3f2fd), Color(0xffbbdefb)]);
    expect(fade.begin, Alignment.centerLeft);
    expect(paint.boxShadow!.single.offset, const Offset(0, 2));
    expect(paint.boxShadow!.single.blurRadius, 4);
    expect(paint.image!.image, isA<MemoryImage>());
    expect(paint.image!.fit, BoxFit.cover);
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

    await tester.tap(find.byType(TextField).first);
    await tester.pump();
    expect(
      FocusManager.instance.primaryFocus?.hasFocus,
      isTrue,
      reason: 'the field must take the focus',
    );

    await tester.enterText(find.byType(TextField).first, 'Ada');
    await tester.pump();
    expect(find.text('Hello Ada'), findsOneWidget);
    expect(
      FocusManager.instance.primaryFocus?.hasFocus,
      isTrue,
      reason: 'the field must keep the focus after the Go rebuild',
    );

    final EditableText field = tester.widget<EditableText>(
      find.byType(EditableText).first,
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

  testWidgets('a Go key keeps the focus node across a Go rebuild', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    final FocusNode kept = tester
        .widget<TextField>(find.byType(TextField).at(1))
        .focusNode!;
    expect(find.textContaining('note idle'), findsOneWidget);

    await tester.tap(find.byType(TextField).at(1));
    await tester.pump();
    await tester.pump();

    // The host reported the focus to Go, and the Go rebuild of the boundary
    // gave the field the same node again.
    expect(find.textContaining('note focused'), findsOneWidget);
    expect(kept.hasFocus, isTrue);
    final TextField again = tester.widget<TextField>(
      find.byType(TextField).at(1),
    );
    expect(identical(again.focusNode, kept), isTrue);
  });

  testWidgets('a Go key keeps the scroll controller across a Go rebuild', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());
    await tester.tap(find.byTooltip('load the list'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 250));
    await tester.pump();

    final ScrollController kept = tester
        .widget<ListView>(find.byType(ListView))
        .controller!;
    expect(find.text('offset 0'), findsOneWidget);

    await tester.drag(find.byType(ListView), const Offset(0, -300));
    await tester.pump();
    await tester.pump();

    // The offset reached Go, and the Go rebuild replaced the list widget. The
    // list kept the controller, thus it kept the offset too.
    final double at = kept.offset;
    expect(at, greaterThan(0));
    expect(find.text('offset ${at.round()}'), findsOneWidget);
    final ListView again = tester.widget<ListView>(find.byType(ListView));
    expect(identical(again.controller, kept), isTrue);
    expect(again.controller!.offset, at);
  });

  testWidgets('a Go key keeps the states controller across a Go rebuild', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    final WidgetStatesController kept = tester
        .widget<ElevatedButton>(find.byType(ElevatedButton))
        .statesController!;
    expect(find.textContaining('button none'), findsOneWidget);

    final TestGesture press = await tester.startGesture(
      tester.getCenter(find.byType(ElevatedButton)),
    );
    await tester.pump(const Duration(milliseconds: 200));
    await tester.pump();

    // The press reached Go, and the Go rebuild gave the button the same
    // controller, thus the pressed state holds.
    expect(find.textContaining('pressed'), findsOneWidget);
    expect(kept.value, contains(WidgetState.pressed));
    final ElevatedButton again = tester.widget<ElevatedButton>(
      find.byType(ElevatedButton),
    );
    expect(identical(again.statesController, kept), isTrue);

    await press.up();
    await tester.pumpAndSettle();
    expect(find.text('the note field left the tree'), findsOneWidget);
  });

  testWidgets('the host disposes an object when its key leaves the tree', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(harness());

    final FocusNode kept = tester
        .widget<TextField>(find.byType(TextField).at(1))
        .focusNode!;

    await tester.tap(find.text('hide the note'));
    await tester.pump();
    await tester.pump();

    expect(find.byType(TextField), findsOneWidget);
    expect(find.text('the note field left the tree'), findsOneWidget);

    // The sweep of the host runs after the frame. A disposed ChangeNotifier
    // rejects a listener, thus the call below proves the dispose.
    expect(
      () => kept.addListener(() {}),
      throwsA(isA<FlutterError>()),
      reason: 'the host must dispose the focus node of a key that left',
    );

    // The name comes back with a new node, and the old node stays gone.
    await tester.tap(find.text('show the note'));
    await tester.pump();
    final FocusNode fresh = tester
        .widget<TextField>(find.byType(TextField).at(1))
        .focusNode!;
    expect(identical(fresh, kept), isFalse);
  });
}

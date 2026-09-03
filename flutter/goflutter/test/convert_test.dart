import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';

/// These cases drive the converters of the generated host with a Go node that
/// the test builds by hand. A widget test of an example proves the whole path
/// from Go source, and this file proves each rule of 1 converter.
///
/// The Dart output of a Go module holds the real node classes, and this
/// package imports no such output. A node is a value and a type descriptor,
/// thus a list of the fields and a descriptor of the same order give the same
/// read.

/// _any stands for the type of a field that no converter reads through the
/// descriptor.
const TypeDesc _any = TypeDesc('any', 'string', <String, Function>{});

/// desc gives the type descriptor of a Go struct. The order of the names is
/// the order of the fields, which is the order that the host reads.
TypeDesc desc(String name, List<String> names, [Map<String, TypeDesc>? kinds]) {
  return TypeDesc(
    name,
    'struct',
    const <String, Function>{},
    fields: <FieldDesc>[
      for (int i = 0; i < names.length; i++)
        FieldDesc(
          names[i],
          '',
          true,
          () => kinds?[names[i]] ?? _any,
          (Object? r) => (r! as List<Object?>)[i],
          (Object? r, Object? v) {},
        ),
    ],
  );
}

/// node gives 1 Go node of a descriptor and its values.
GoNode node(TypeDesc d, List<Object?> values) => GoNode.struct(values, d);

/// slice gives a Go slice of structs, and the descriptor of the field.
TypeDesc sliceOf(TypeDesc elem) => TypeDesc(
  'slice',
  'slice',
  const <String, Function>{},
  elem: () => elem,
);

/// _Bridge holds no Go library. GoRuntime.run needs no bridge, because it
/// starts a goroutine and runs the scheduler.
final class _Bridge implements GoBridge {
  @override
  void init() {}

  @override
  Iface? build() => null;

  @override
  int newWatch() => 0;

  @override
  Iface? runWatch(int id, Object? build) => null;

  @override
  void disposeWatch(int id) {}

  @override
  List<int> takeDirty() => const <int>[];

  @override
  int pump() => 0;

  @override
  void setTheme({
    required bool dark,
    required int primary,
    required int surface,
    required int onSurface,
    required double textSize,
  }) {}

  @override
  void setMedia({
    required double width,
    required double height,
    required double textScale,
    required double topInset,
  }) {}
}

void main() {
  final GoRuntime rt = GoRuntime(_Bridge());

  test('a date and a time of the day cross as a number', () {
    expect(goDate(0), isNull, reason: '0 means no date');
    expect(goDate(null), isNull);
    expect(
      goDate(1700000000000),
      DateTime.fromMillisecondsSinceEpoch(1700000000000),
    );
    expect(goClock(9 * 60 + 30), const TimeOfDay(hour: 9, minute: 30));
    expect(goClock(null), isNull);
  });

  test('a Rect gives 4 edges, and a RelativeRect gives 4 distances', () {
    final TypeDesc d = desc('gart.dev/ui.Rect', <String>[
      'Left',
      'Top',
      'Right',
      'Bottom',
    ]);
    final GoNode n = node(d, <Object?>[1.0, 2.0, 3.0, 4.0]);
    expect(goRect(n), const Rect.fromLTRB(1, 2, 3, 4));
    expect(goRelRect(n), const RelativeRect.fromLTRB(1, 2, 3, 4));
    expect(goRect(null), isNull);
  });

  test('a Span holds the text of each child', () {
    final TypeDesc d = desc('gart.dev/ui.Span', <String>[
      'Text',
      'Style',
      'Children',
    ]);
    final TypeDesc kids = sliceOf(d);
    final TypeDesc top = desc(
      'gart.dev/ui.Span',
      <String>['Text', 'Style', 'Children'],
      <String, TypeDesc>{'Children': kids},
    );
    final List<Object?> child = <Object?>[
      GoString.fromString(' and Dart'),
      null,
      Slice<Object?>.of(const <Object?>[]),
    ];
    final GoNode n = node(top, <Object?>[
      GoString.fromString('Go'),
      null,
      Slice<Object?>.of(<Object?>[child]),
    ]);

    final TextSpan? span = goSpan(n);
    expect(span!.text, 'Go');
    expect(span.children!.length, 1);
    expect((span.children!.first as TextSpan).text, ' and Dart');
    expect(span.toPlainText(), 'Go and Dart');
  });

  test('SliverChildren gives a list delegate or a builder delegate', () {
    final TypeDesc d = desc('gart.dev/ui.SliverChildren', <String>[
      'Children',
      'Builder',
      'Count',
    ]);
    final GoNode list = node(d, <Object?>[
      Slice<Iface?>.of(const <Iface?>[null, null]),
      null,
      0,
    ]);
    final SliverChildDelegate? one = goSliverChildren(list, rt);
    expect(one, isA<SliverChildListDelegate>());
    expect((one! as SliverChildListDelegate).children.length, 2);

    Iface? make(int i) => null;
    final GoNode lazy = node(d, <Object?>[
      Slice<Iface?>.of(const <Iface?>[]),
      make,
      7,
    ]);
    final SliverChildDelegate? two = goSliverChildren(lazy, rt);
    expect(two, isA<SliverChildBuilderDelegate>());
    expect((two! as SliverChildBuilderDelegate).childCount, 7);
  });

  test('a GridSpec gives the delegate of its kind', () {
    final TypeDesc d = desc('gart.dev/ui.GridSpec', <String>[
      'Kind',
      'Count',
      'MaxExtent',
      'MainSpacing',
      'CrossSpacing',
      'AspectRatio',
      'MainExtent',
    ]);
    final GoNode count = node(d, <Object?>[
      GoString.fromString('count'),
      3,
      0.0,
      4.0,
      5.0,
      0.0,
      null,
    ]);
    final SliverGridDelegate? a = goGridSpec(count);
    expect(a, isA<SliverGridDelegateWithFixedCrossAxisCount>());
    final SliverGridDelegateWithFixedCrossAxisCount fixed =
        a! as SliverGridDelegateWithFixedCrossAxisCount;
    expect(fixed.crossAxisCount, 3);
    expect(fixed.mainAxisSpacing, 4);
    expect(fixed.childAspectRatio, 1, reason: '0 gives 1');

    final GoNode extent = node(d, <Object?>[
      GoString.fromString('extent'),
      0,
      120.0,
      0.0,
      0.0,
      2.0,
      null,
    ]);
    final SliverGridDelegate? b = goGridSpec(extent);
    expect(b, isA<SliverGridDelegateWithMaxCrossAxisExtent>());
    expect(
      (b! as SliverGridDelegateWithMaxCrossAxisExtent).maxCrossAxisExtent,
      120,
    );
  });

  test('a builder of a box gives 0 for a maximum with no limit', () {
    final List<List<double>> seen = <List<double>>[];
    Iface? build(double a, double b, double c, double d) {
      seen.add(<double>[a, b, c, d]);
      return null;
    }

    final Widget Function(BuildContext, BoxConstraints)? f = goBoxBuilder(
      build,
      rt,
    );
    f!(_FakeContext(), const BoxConstraints(minWidth: 10, maxWidth: 20));
    expect(seen.single, <double>[10, 20, 0, 0]);
  });

  test('a Task gives a Future that the Go side completes', () async {
    void Function()? held;
    void work(void Function() done) => held = done;

    final Future<void> Function()? f = goTask(work, rt);
    bool ended = false;
    final Future<void> waiting = f!()..then((void _) => ended = true);
    await Future<void>.delayed(Duration.zero);
    expect(ended, isFalse, reason: 'the work still runs');

    held!();
    await waiting;
    expect(ended, isTrue);
  });

  test('an Ask gives the answer of the Go side', () async {
    void Function(bool)? held;
    void work(void Function(bool) answer) => held = answer;

    final Future<bool> Function()? f = goAsk(work, rt);
    final Future<bool> waiting = f!();
    held!(true);
    expect(await waiting, isTrue);
  });

  testWidgets('an animation runs to its target', (WidgetTester tester) async {
    final GoIdentities table = GoIdentities();
    table.vsync = const TestVSync();
    final TypeDesc d = desc('gart.dev/ui.AnimationKey', <String>[
      'Name',
      'Millis',
      'Target',
      'Curve',
      'Repeat',
      'Reverse',
      'OnEnd',
    ]);
    final GoNode key = node(d, <Object?>[
      GoString.fromString('card'),
      100,
      1.0,
      GoString.fromString('linear'),
      false,
      false,
      null,
    ]);

    final GoRuntime host = GoRuntime(_Bridge());
    host.identities.vsync = table.vsync;
    // goRender holds the keys of 1 node in a GoKeep widget. A test that reads
    // a key with no such widget loses the object to the sweep of the frame.
    host.identities.open();
    final Animation<double>? a = host.identities.animation(key, host);
    host.identities.retain(host.identities.close());
    expect(a!.value, 0);

    // The host gives the command after the frame, because an animation that
    // starts during a build marks a widget dirty in that build. A real build
    // stands inside a frame, and this test holds no widget, thus it asks for
    // the frame itself.
    tester.binding.scheduleFrame();
    await tester.pump();
    // The first tick of a Flutter ticker holds the start time, thus the run
    // needs a second frame. pumpAndSettle gives each frame that the animation
    // asks for.
    await tester.pumpAndSettle();
    expect(a.value, 1);

    // A second read of the same key gives the same object.
    expect(identical(host.identities.animation(key, host), a), isTrue);
    host.identities.dropAll();
  });

  testWidgets('a link and an overlap handle hold their identity', (
    WidgetTester tester,
  ) async {
    final GoRuntime host = GoRuntime(_Bridge());
    final TypeDesc d = desc('gart.dev/ui.LinkKey', <String>['Name']);
    final GoNode a = node(d, <Object?>[GoString.fromString('menu')]);
    final GoNode b = node(d, <Object?>[GoString.fromString('other')]);
    expect(
      identical(goLayerLink(a, host), goLayerLink(a, host)),
      isTrue,
    );
    expect(identical(goLayerLink(a, host), goLayerLink(b, host)), isFalse);
    expect(
      identical(goOverlapHandle(a, host), goOverlapHandle(a, host)),
      isTrue,
    );
    host.identities.dropAll();
  });
}

/// _FakeContext stands for the BuildContext that a builder of the host drops.
/// The Go side reads no context, thus no converter touches it.
final class _FakeContext extends StatelessWidget implements BuildContext {
  @override
  Widget build(BuildContext context) => const SizedBox();

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

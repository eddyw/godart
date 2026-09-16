import 'package:compass_v2/bridge.dart';
import 'package:compass_v2/widgets_gen.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';
import 'package:material_ui/material_ui.dart';

Iface node(String kind, Map<String, Object?> fields) => Iface(
  TypeDesc(
    kind,
    'struct',
    const <String, Function>{},
    fields: [
      for (final name in fields.keys)
        FieldDesc(
          name,
          '',
          true,
          () => const TypeDesc('any', 'interface', <String, Function>{}),
          (value) => (value as Map<String, Object?>)[name],
          (value, field) => (value as Map<String, Object?>)[name] = field,
        ),
    ],
  ),
  fields,
);

void main() {
  final runtime = GoRuntime(const CompassBridge());
  setUp(goGenInstall);
  tearDown(runtime.stop);

  testWidgets('fill uses default insets and explicit insets', (tester) async {
    Widget fill(Map<String, Object?> insets) => goGenWidget(
      GoNode(
        node('gart.dev/ui.PositionedFill', {
          ...insets,
          'Child': node('gart.dev/ui.SizedBox', {}),
        }),
      ),
      runtime,
    )!;
    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: Center(
          child: SizedBox(
            width: 200,
            height: 100,
            child: Stack(children: [fill({})]),
          ),
        ),
      ),
    );
    expect(tester.getSize(find.byType(SizedBox).last), const Size(200, 100));
    final custom =
        fill({'Left': Box<double>(12), 'Bottom': Box<double>(8)}) as Positioned;
    expect(
      [custom.left, custom.top, custom.right, custom.bottom],
      [12, 0, 0, 8],
    );
  });

  testWidgets('scaffold preserves nil and empty footers', (tester) async {
    Scaffold scaffold(Slice<Iface?> footer) =>
        goGenWidget(
              GoNode(
                node('gart.dev/ui/material.Scaffold', {
                  'PersistentFooterButtons': footer,
                  'Body': node('gart.dev/ui.SizedBox', {}),
                }),
              ),
              runtime,
            )!
            as Scaffold;
    final withoutFooter = scaffold(Slice<Iface?>.nil());
    expect(withoutFooter.persistentFooterButtons, isNull);
    await tester.pumpWidget(MaterialApp(home: withoutFooter));
    expect(find.byType(OverflowBar), findsNothing);
    final emptyFooter = scaffold(Slice<Iface?>.of([]));
    expect(emptyFooter.persistentFooterButtons, isEmpty);
    await tester.pumpWidget(MaterialApp(home: emptyFooter));
    expect(find.byType(OverflowBar), findsOneWidget);
    final populated = scaffold(
      Slice<Iface?>.of([node('gart.dev/ui.SizedBox', {})]),
    );
    expect(populated.persistentFooterButtons, hasLength(1));
  });
}

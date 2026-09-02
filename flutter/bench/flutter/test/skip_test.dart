import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:skipbench/skip.dart';

/// The counts of the sweep. The pair gives a slope, thus the fixed cost of 1
/// build scope leaves the number.
const List<int> counts = <int>[50, 200, 500];

/// The build scopes of the warm up and of the measurement.
const int warm = 200;
const int runs = 2000;

/// measure gives the mean time of 1 build scope, in nanoseconds.
///
/// The loop drives the build owner and not the pump of the tester. A profile
/// run holds a real binding, thus a pump waits for the next frame of the
/// screen and the wait hides the work. The build scope holds each part that an
/// unchanged node skips: the widget of the child, the update of its element,
/// and the write of its render object.
Future<double> measure(WidgetTester tester, int nodes, bool reuse) async {
  final GlobalKey<SkipTreeState> key = GlobalKey<SkipTreeState>();
  await tester.pumpWidget(
    Directionality(
      textDirection: TextDirection.ltr,
      child: SingleChildScrollView(
        child: SkipTree(key: key, nodes: nodes, reuse: reuse),
      ),
    ),
  );
  final SkipTreeState state = key.currentState!;
  final BuildOwner owner = tester.binding.buildOwner!;
  final Element root = tester.binding.rootElement!;
  for (int i = 0; i < warm; i++) {
    state.again();
    owner.buildScope(root);
  }
  final Stopwatch clock = Stopwatch()..start();
  for (int i = 0; i < runs; i++) {
    state.again();
    owner.buildScope(root);
  }
  clock.stop();
  return clock.elapsedMicroseconds * 1000 / runs;
}

void main() {
  testWidgets('the work that 1 unchanged node skips', (
    WidgetTester tester,
  ) async {
    final StringBuffer out = StringBuffer();
    out.writeln('nodes,fresh_ns,reuse_ns,skipped_ns_per_node');
    final Map<int, double> fresh = <int, double>{};
    final Map<int, double> reuse = <int, double>{};
    for (final int nodes in counts) {
      fresh[nodes] = await measure(tester, nodes, false);
      reuse[nodes] = await measure(tester, nodes, true);
      final double each = (fresh[nodes]! - reuse[nodes]!) / nodes;
      out.writeln(
        '$nodes,${fresh[nodes]!.round()},${reuse[nodes]!.round()},'
        '${each.toStringAsFixed(1)}',
      );
    }
    final int low = counts.first;
    final int high = counts.last;
    final double slope =
        ((fresh[high]! - reuse[high]!) - (fresh[low]! - reuse[low]!)) /
        (high - low);
    out.writeln(
      'reuse_ns_per_node,,,'
      '${((reuse[high]! - reuse[low]!) / (high - low)).toStringAsFixed(1)}',
    );
    out.writeln('slope,,,${slope.toStringAsFixed(1)}');
    // The table reaches standard output. flutter test and flutter drive each
    // show it.
    debugPrint(out.toString());
    expect(fresh[high]!, greaterThan(reuse[high]!));
  });
}

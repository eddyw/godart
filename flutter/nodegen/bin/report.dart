// A report of the yield of the reader. It names each widget that maps, each
// widget that the reader dropped, and the Dart types that no Go type carries.
import 'dart:io';

import 'package:nodegen/read.dart';

Future<void> main(List<String> args) async {
  final clock = Stopwatch()..start();
  final r = await read(args.isEmpty ? '.' : args.first);
  stdout.writeln('the read took ${clock.elapsedMilliseconds} ms');
  stdout.writeln('${r.widgets.length} node types map, '
      '${r.skipped.length} do not');

  var fields = 0;
  for (final w in r.widgets) {
    fields += w.params.length;
  }
  stdout.writeln('$fields fields, '
      '${(fields / r.widgets.length).toStringAsFixed(1)} for each node');

  final byLib = <String, int>{};
  for (final w in r.widgets) {
    byLib[w.library] = (byLib[w.library] ?? 0) + 1;
  }
  stdout.writeln('by library: $byLib');

  stdout.writeln('\nthe 25 Dart types that no Go type carries:');
  final pairs = r.unmapped.entries.toList()
    ..sort((a, b) => b.value.compareTo(a.value));
  for (final e in pairs.take(25)) {
    stdout.writeln('  ${e.value.toString().padLeft(4)}  ${e.key}');
  }

  stdout.writeln('\nthe first 20 widgets that the reader dropped:');
  for (final s in r.skipped.take(20)) {
    stdout.writeln('  ${s.name}: ${s.cause}');
  }

  stdout.writeln('\nthe first 10 node types:');
  for (final w in r.widgets.take(10)) {
    final f = w.params
        .map((p) => p.goDecl)
        .join(', ');
    stdout.writeln('  ${w.goName} = ${w.dartCall}($f)');
  }
}

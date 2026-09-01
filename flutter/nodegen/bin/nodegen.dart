// nodegen reads the public API of Flutter and writes the node set of
// milestone 3. Read flutter/nodegen/README.md.
//
// The 2 modes:
//
//   nodegen go   <flutter package dir> <go file>
//   nodegen host <flutter package dir> <dart file> <go source dir>
//
// The Go set is shared, and 1 run writes it for the whole project. The host is
// per application, because a host that names each Flutter widget stops the
// tree shaker of Dart from dropping the framework.
import 'dart:io';

import 'package:nodegen/read.dart';
import 'package:nodegen/write_dart.dart';
import 'package:nodegen/write_go.dart';

Future<void> main(List<String> args) async {
  final mode = args.isEmpty ? '' : args.first;
  if (mode != 'go' && mode != 'host' ||
      mode == 'go' && args.length != 3 ||
      mode == 'host' && args.length != 4) {
    stderr.writeln('usage: nodegen go   <flutter package dir> <go file>');
    stderr.writeln('       nodegen host <flutter package dir> <dart file> '
        '<go source dir>');
    exitCode = 2;
    return;
  }
  final clock = Stopwatch()..start();
  final r = await read(args[1]);
  var fields = 0;
  for (final w in r.widgets) {
    fields += w.params.length;
  }
  stdout.writeln('the read of the Flutter API took '
      '${clock.elapsedMilliseconds} ms: ${r.widgets.length} node types, '
      '$fields fields, ${r.enums.length} enums');

  if (mode == 'go') {
    File(args[2]).writeAsStringSync(writeGo(r));
    // The emitter writes 1 space between the name and the type of a field, and
    // gofmt aligns the block.
    final fmt = Process.runSync('gofmt', <String>['-w', args[2]]);
    if (fmt.exitCode != 0) {
      stderr.writeln('gofmt failed: ${fmt.stderr}');
      exitCode = 1;
      return;
    }
    stdout.writeln('${args[2]} holds the Go node set');
    return;
  }
  final only = usedNodes(args[3]);
  File(args[2]).writeAsStringSync(writeDart(r, only: only));
  final kept = r.widgets.where((w) => only.contains(w.goName)).length;
  stdout.writeln('${args[2]} holds $kept cases of the ${only.length} names '
      'that ${args[3]} writes');
}

/// usedNodes reads each name that a Go source file writes after the qualifier
/// of the node package.
///
/// The set is a superset of the node types that the application builds, thus
/// the host holds a case for each 1 that it needs. A name that the scan misses
/// gives a clear error at run time, and never a wrong widget.
Set<String> usedNodes(String dir) {
  final out = <String>{};
  final pattern = RegExp(r'\bui\.([A-Z][A-Za-z0-9]*)');
  for (final f in Directory(dir).listSync(recursive: true)) {
    if (f is! File || !f.path.endsWith('.go')) continue;
    for (final m in pattern.allMatches(f.readAsStringSync())) {
      out.add(m.group(1)!);
    }
  }
  return out;
}

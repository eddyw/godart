// nodegen reads the public API of Flutter and writes the node set of
// milestone 3. Read flutter/nodegen/README.md.
//
// The 3 modes:
//
//   nodegen go     <flutter package dir> <go file>
//   nodegen host   <flutter package dir> <dart file> <name list>
//   nodegen report <flutter package dir> [-all]
//
// The Go set is shared, and 1 run writes it for the whole project. The host is
// per application, because a host that names each Flutter widget stops the
// tree shaker of Dart from dropping the framework.
//
// The name list of the host mode holds 1 node type for each line. The Go
// command of flutter/nodescan writes it from the type information of the
// application. A single dash reads the list from the standard input, and
// -all writes a case for each node type of the set.
import 'dart:convert';
import 'dart:io';

import 'package:nodegen/read.dart';
import 'package:nodegen/report.dart';
import 'package:nodegen/write_dart.dart';
import 'package:nodegen/write_go.dart';

const _usage = 'usage: nodegen go     <flutter package dir> <go file>\n'
    '       nodegen host   <flutter package dir> <dart file> <name list>\n'
    '       nodegen report <flutter package dir> [-all]';

Future<void> main(List<String> args) async {
  final mode = args.isEmpty ? '' : args.first;
  const counts = <String, int>{'go': 3, 'host': 4, 'report': 2};
  final want = counts[mode];
  if (want == null ||
      args.length != want && !(mode == 'report' && args.length == 3)) {
    stderr.writeln(_usage);
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

  switch (mode) {
    case 'go':
      File(args[2]).writeAsStringSync(writeGo(r));
      // The emitter writes 1 space between the name and the type of a field,
      // and gofmt aligns the block.
      final fmt = Process.runSync('gofmt', <String>['-w', args[2]]);
      if (fmt.exitCode != 0) {
        stderr.writeln('gofmt failed: ${fmt.stderr}');
        exitCode = 1;
        return;
      }
      stdout.writeln('${args[2]} holds the Go node set');

    case 'report':
      final all = args.length == 3 && args[2] == '-all';
      if (args.length == 3 && !all) {
        stderr.writeln(_usage);
        exitCode = 2;
        return;
      }
      stdout.write(report(r, all: all));

    case 'host':
      final only = await readNames(args[3]);
      File(args[2]).writeAsStringSync(writeDart(r, only: only));
      if (only == null) {
        stdout.writeln('${args[2]} holds a case for each of the '
            '${r.widgets.length} node types');
        return;
      }
      final kept = r.widgets.where((w) => only.contains(w.goName)).length;
      stdout.writeln('${args[2]} holds $kept cases of the '
          '${only.length} names that ${args[3]} gives');
  }
}

/// readNames gives the node types that the host must build.
///
/// The source holds 1 name for each line. A dash reads the standard input, and
/// -all gives no set, thus the host holds a case for each node type.
Future<Set<String>?> readNames(String source) async {
  if (source == '-all') return null;
  final text = source == '-'
      ? await stdin.transform(utf8.decoder).join()
      : File(source).readAsStringSync();
  final out = <String>{};
  for (final line in const LineSplitter().convert(text)) {
    final name = line.trim();
    if (name.isNotEmpty) out.add(name);
  }
  return out;
}

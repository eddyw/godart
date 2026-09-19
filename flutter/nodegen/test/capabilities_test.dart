import 'dart:io';

import 'package:nodegen/read.dart';
import 'package:nodegen/write_go.dart';

Future<void> main() async {
  final result = await read('../goflutter');
  var fields = 0;
  final missing = <String>[];
  for (final pack in packs) {
    final suffix = pack.goPath.substring(corePack.goPath.length);
    final source = File('../ui$suffix/widgets_gen.go').readAsStringSync();
    final structs = RegExp(r'type (\w+) struct \{([^}]*)\}', multiLine: true);
    for (final match in structs.allMatches(source)) {
      final name = match[1]!;
      final widgets = result.widgets.where(
        (w) => w.pack == pack && w.goName == name,
      );
      if (widgets.length != 1) throw StateError('Missing ${pack.goPath}.$name');
      final generated = {
        for (final p in widgets.single.params)
          goDecl(p, pack).replaceAll(RegExp(r'\s+'), ' '),
      };
      for (final line in match[2]!.split('\n')) {
        final field = line.trim().replaceAll(RegExp(r'\s+'), ' ');
        if (field.isEmpty || field.startsWith('//')) continue;
        if (!generated.contains(field))
          missing.add('${pack.goPath}.$name.$field');
        fields++;
      }
    }
  }
  final paint = result.widgets.singleWhere(
    (w) => w.kind == 'gart.dev/ui.CustomPaint',
  );
  for (final name in ['Painter', 'ForegroundPainter']) {
    if (!paint.params.any((p) => p.goName == name && p.type.kind == 'painter'))
      throw StateError('Missing CustomPaint.$name');
  }
  if (missing.isNotEmpty)
    throw StateError('Missing fields:\n${missing.join("\n")}');
  stdout.writeln(
    'Preserved $fields generated fields. CustomPainter callbacks are mapped.',
  );
}

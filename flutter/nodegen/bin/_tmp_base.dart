import 'dart:io';
import 'package:nodegen/read.dart';

Future<void> main(List<String> args) async {
  final r = await read(args.isEmpty ? '.' : args.first);
  var fields = 0;
  for (final w in r.widgets) { fields += w.params.length; }
  stdout.writeln('${r.widgets.length} nodes, $fields fields, ${r.skipped.length} dropped');
  final causes = <String,int>{};
  for (final s in r.skipped) {
    final c = s.cause.replaceAll(RegExp(r'the parameter \w+ holds '), '');
    causes[c] = (causes[c] ?? 0) + 1;
  }
  final pairs = causes.entries.toList()..sort((a,b)=>b.value.compareTo(a.value));
  stdout.writeln('--- dropped by cause ---');
  for (final e in pairs) { stdout.writeln('  ${e.value.toString().padLeft(3)}  ${e.key}'); }
  stdout.writeln('--- no parameter maps ---');
  for (final s in r.skipped) {
    if (s.cause == 'no parameter maps') stdout.writeln('  ${s.name}');
  }
  stdout.writeln('--- dropped naming ButtonStyle/ImageProvider/Decoration ---');
  for (final s in r.skipped) {
    if (s.cause.contains('ButtonStyle') || s.cause.contains('ImageProvider') || s.cause.contains('Decoration')) {
      stdout.writeln('  ${s.name}: ${s.cause}');
    }
  }
}

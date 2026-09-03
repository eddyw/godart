import 'read.dart';

/// report gives the yield of the reader in a text form.
///
/// It names the count of the node types and of the fields, each Dart type that
/// no Go type carries, and each widget constructor that the reader dropped.
/// The cause of a drop is the first parameter that the constructor demands and
/// that mapType cannot map.
///
/// `all` gives the whole list of the dropped constructors, and not the head of
/// it.
String report(Reading r, {bool all = false}) {
  final b = StringBuffer();
  var fields = 0;
  for (final w in r.widgets) {
    fields += w.params.length;
  }
  b.writeln('${r.widgets.length} node types map, ${r.skipped.length} do not');
  b.writeln('$fields fields, '
      '${(fields / r.widgets.length).toStringAsFixed(1)} for each node');

  final byLib = <String, int>{};
  for (final w in r.widgets) {
    byLib[w.library] = (byLib[w.library] ?? 0) + 1;
  }
  b.writeln('by library: $byLib');

  b.writeln();
  b.writeln('the cause of each drop, by count:');
  final byCause = <String, int>{};
  for (final s in r.skipped) {
    byCause[s.shape] = (byCause[s.shape] ?? 0) + 1;
  }
  for (final e in sortByCount(byCause)) {
    b.writeln('  ${e.value.toString().padLeft(4)}  ${e.key}');
  }

  b.writeln();
  b.writeln('the Dart types that no Go type carries, by count:');
  final types = sortByCount(r.unmapped);
  for (final e in all ? types : types.take(25)) {
    b.writeln('  ${e.value.toString().padLeft(4)}  ${e.key}');
  }
  if (!all && types.length > 25) {
    b.writeln('  ... ${types.length - 25} more. Run the mode with -all.');
  }

  b.writeln();
  b.writeln('each constructor that the reader dropped:');
  final drops = all ? r.skipped : r.skipped.take(25);
  for (final s in drops) {
    b.writeln('  ${s.name}: ${s.cause}');
  }
  if (!all && r.skipped.length > 25) {
    b.writeln('  ... ${r.skipped.length - 25} more. Run the mode with -all.');
  }
  return b.toString();
}

/// sortByCount gives the entries of a count table, the largest first.
List<MapEntry<String, int>> sortByCount(Map<String, int> m) {
  final out = m.entries.toList()
    ..sort((a, b) {
      final n = b.value.compareTo(a.value);
      return n != 0 ? n : a.key.compareTo(b.key);
    });
  return out;
}

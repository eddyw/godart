import 'dart:convert';
import 'dart:io';

import 'package:path/path.dart' as p;

String sourceNotices(String project, Iterable<String> sources) {
  final config = File(p.join(project, '.dart_tool', 'package_config.json'))
      .absolute;
  final data = jsonDecode(config.readAsStringSync()) as Map<String, dynamic>;
  final packages = <String, String>{};
  for (final entry in data['packages'] as List<dynamic>) {
    final root = config.uri.resolve(entry['rootUri'] as String).toFilePath();
    packages[p.normalize(root)] = entry['name'] as String;
  }
  final roots = packages.keys.toList()
    ..sort((a, b) => b.length.compareTo(a.length));
  final notices = <String, String>{};
  final visited = <String>{};
  final ordered = sources.toSet().toList()..sort();
  for (final source in ordered) {
    final root = roots.firstWhere(
      (root) => p.isWithin(root, source),
      orElse: () => throw StateError('No package owns $source'),
    );
    final component = packages[root]!;
    var dir = p.dirname(source);
    var licensed = false;
    while (true) {
      for (final entity in Directory(dir).listSync()) {
        if (entity is! File) continue;
        final name = p.basename(entity.path);
        if (!RegExp(
          r'^(LICENSE|LICENCE|COPYING|NOTICE|PATENTS)([.-].*)?$',
          caseSensitive: false,
        ).hasMatch(name)) {
          continue;
        }
        if (RegExp(
          r'^(LICENSE|LICENCE|COPYING)',
          caseSensitive: false,
        ).hasMatch(name)) {
          licensed = true;
        }
        if (visited.add(entity.path)) {
          notices['$component / ${p.relative(entity.path, from: root)}'] =
              entity.readAsStringSync();
        }
      }
      if ((dir == root || !p.isWithin(root, dir)) && licensed) break;
      final parent = p.dirname(dir);
      if (parent == dir) {
        throw StateError('No upstream license for $component');
      }
      dir = parent;
    }
  }
  final keys = notices.keys.toList()..sort();
  return keys
      .map((key) => '$key\n\n${notices[key]}')
      .join(
        '\n--------------------------------------------------------------------------------\n\n',
      );
}

String noticeHeader(String notices) =>
    '${notices.split('\n').map((line) => '// $line').join('\n')}\n';

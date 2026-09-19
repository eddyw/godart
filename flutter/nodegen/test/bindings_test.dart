import 'dart:convert';
import 'dart:io';

import 'package:nodegen/bindings.dart';
import 'package:path/path.dart' as p;

Future<void> main() async {
  final temp = Directory.systemTemp.createTempSync('nodegen-bindings-');
  try {
    final original = File('../testdata/core/.dart_tool/package_config.json')
        .absolute;
    final config =
        jsonDecode(original.readAsStringSync()) as Map<String, dynamic>;
    for (final package in config['packages'] as List) {
      package['rootUri'] = package['name'] == 'core_fixture'
          ? temp.uri.toString()
          : original.uri.resolve(package['rootUri'] as String).toString();
    }
    void write(String name, String value) {
      final file = File(p.join(temp.path, name));
      file.parent.createSync(recursive: true);
      file.writeAsStringSync(value);
    }

    write('.dart_tool/package_config.json', jsonEncode(config));
    write('pubspec.yaml', 'name: core_fixture\nenvironment:\n  sdk: ^3.13.0\n');
    write('LICENSE', 'Fixture license\n');
    final manifest = {
      'version': 1,
      'libraries': [
        {
          'uri': 'package:core_fixture/brand.dart',
          'go_import': 'example.dev/probe/brand',
          'directory': 'go/brand',
          'symbols': ['Badge'],
        },
        {
          'uri': 'package:core_fixture/other.dart',
          'go_import': 'example.dev/probe/other',
          'directory': 'go/other',
          'symbols': ['Badge'],
        },
      ],
    };
    write('glutter.bindings.json', jsonEncode(manifest));
    String source(String field) =>
        '''
import 'package:flutter/widgets.dart';
class Badge extends StatelessWidget {
 const Badge({super.key, required this.label, $field});
 final String label;
 @override
 Widget build(BuildContext context) => Text(label);
}
''';
    write('lib/brand.dart', source(''));
    write('lib/other.dart', source(''));
    await generateBindings(temp.path);
    final first = File(p.join(temp.path, 'go/brand/widgets_gen.go'));
    if (!first.readAsStringSync().contains('Fixture license') ||
        !first.readAsStringSync().contains(
          'Copyright 2014 The Flutter Authors',
        ))
      throw StateError('Missing notices');
    if (!File(p.join(temp.path, 'go/other/widgets_gen.go'))
        .readAsStringSync()
        .contains('type Badge struct'))
      throw StateError('Equal class names lost a binding');
    write('lib/brand.dart', source('int count = 0'));
    await generateBindings(temp.path);
    if (!RegExp(r'Count\s+\*int').hasMatch(first.readAsStringSync()))
      throw StateError('Additive API update lost a field');
    final before = first.readAsStringSync();
    write('lib/brand.dart', source('required Map<String, String> values'));
    var rejected = false;
    try {
      await generateBindings(temp.path);
    } on StateError catch (error) {
      rejected =
          error.toString().contains('values') &&
          error.toString().contains('Map');
    }
    if (!rejected || first.readAsStringSync() != before)
      throw StateError('Incompatible API update replaced valid output');
    write('lib/brand.dart', source(''));
    first.writeAsStringSync('package brand\n// handwritten file\n');
    rejected = false;
    try {
      await generateBindings(temp.path);
    } on StateError catch (error) {
      rejected = error.toString().contains('handwritten');
    }
    if (!rejected || !first.readAsStringSync().contains('handwritten'))
      throw StateError('Replaced a handwritten file');
    first.writeAsStringSync(before);
    (manifest['libraries'] as List).removeLast();
    write('glutter.bindings.json', jsonEncode(manifest));
    await generateBindings(temp.path);
    if (File(p.join(temp.path, 'go/other/widgets_gen.go')).existsSync())
      throw StateError('Retained an obsolete generated package');
    for (final directory in ['../escape', '/tmp/escape']) {
      final entry = (manifest['libraries'] as List).first as Map;
      entry['directory'] = directory;
      write('glutter.bindings.json', jsonEncode(manifest));
      rejected = false;
      try {
        bindings(temp.path);
      } on FormatException {
        rejected = true;
      }
      if (!rejected) throw StateError('Accepted an output escape');
    }
    stdout.writeln(
      'Binding API changes, duplicate names, notices, and output boundaries pass.',
    );
  } finally {
    temp.deleteSync(recursive: true);
  }
}

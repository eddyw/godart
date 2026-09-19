import 'dart:convert';
import 'dart:io';

import 'package:nodegen/notices.dart';
import 'package:path/path.dart' as p;

void main() {
  final temp = Directory.systemTemp.createTempSync('nodegen-notices-');
  try {
    void write(String name, String text) {
      final file = File(p.join(temp.path, name));
      file.parent.createSync(recursive: true);
      file.writeAsStringSync(text);
    }

    write(
      'app/.dart_tool/package_config.json',
      jsonEncode({
        'configVersion': 2,
        'packages': [
          {'name': 'widget', 'rootUri': '../../sdk/packages/widget'},
          {'name': 'extra', 'rootUri': '../../extra'},
        ],
      }),
    );
    write('sdk/LICENSE', 'SDK license\n');
    write('sdk/packages/widget/lib/NOTICE', 'Widget attribution\n');
    write('sdk/packages/widget/lib/a.dart', '');
    write('extra/LICENSE', 'Extra license\n');
    write('extra/PATENTS', 'Extra patent grant\n');
    write('extra/lib/b.dart', '');
    final sources = [
      p.join(temp.path, 'sdk/packages/widget/lib/a.dart'),
      p.join(temp.path, 'extra/lib/b.dart'),
    ];
    final app = p.join(temp.path, 'app');
    final text = sourceNotices(app, sources);
    for (final value in [
      'SDK license\n',
      'Widget attribution\n',
      'Extra license\n',
      'Extra patent grant\n',
    ]) {
      if (!text.contains(value)) throw StateError('Missing $value');
    }
    if (text.contains(temp.path)) throw StateError('Absolute path in notices');
    if (sourceNotices(app, sources.reversed) != text) {
      throw StateError('Source order changed notices');
    }
    if (!noticeHeader(text).contains('// SDK license')) {
      throw StateError('Source header omits the license');
    }
    File(p.join(temp.path, 'sdk/LICENSE')).deleteSync();
    var rejected = false;
    try {
      sourceNotices(app, sources);
    } on StateError {
      rejected = true;
    }
    if (!rejected) throw StateError('Missing license was accepted');
    stdout.writeln('Nodegen notice tests pass.');
  } finally {
    temp.deleteSync(recursive: true);
  }
}

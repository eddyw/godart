import 'dart:io';

import 'package:goruntime/goruntime.dart';

void main() {
  final root = Directory.systemTemp.createTempSync('gart-open-errors-');
  try {
    final missing = GoString.fromString('${root.path}/missing');
    final first = $syscallOpen(missing, 0, 0);
    if (first != (-1, 2)) throw StateError('missing file: $first');
    final directory = $syscallOpen(GoString.fromString(root.path), 1, 0);
    if (directory != (-1, 21)) {
      throw StateError('directory write: $directory');
    }
    final file = File('${root.path}/file')..writeAsStringSync('keep');
    final exclusive = Platform.isMacOS ? 0x200 | 0x800 : 0x40 | 0x80;
    final exists = $syscallOpen(
      GoString.fromString(file.path),
      exclusive,
      0x180,
    );
    if (exists != (-1, 17)) throw StateError('exclusive open: $exists');
    if (file.readAsStringSync() != 'keep') throw StateError('file changed');
    for (var i = 0; i < 100; i++) {
      final repeated = $syscallOpen(missing, 0, 0);
      if (repeated != (-1, 2)) throw StateError('repeated open: $repeated');
    }
  } finally {
    root.deleteSync(recursive: true);
  }
}

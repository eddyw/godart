import 'dart:io';

import 'package:goruntime/goruntime.dart';

void main() {
  final root = Directory.systemTemp.createTempSync('gart-unlink-errno-');
  try {
    final path = GoString.fromString(root.path);
    final directoryError = Platform.isMacOS ? 1 : 21;
    final first = $syscallUnlink(path, false);
    if (first != directoryError) throw StateError('directory errno $first');
    final invalid = $syscallUnlinkat(-1, GoString.fromString('file'));
    if (invalid != 9) throw StateError('descriptor errno $invalid');
    final nul = $syscallUnlinkat(-1, GoString.fromString('bad\x00name'));
    if (nul != 22) throw StateError('NUL errno $nul');
    for (var i = 0; i < 32; i++) {
      final file = File('${root.path}/file')..writeAsStringSync('remove');
      final name = GoString.fromString(file.path);
      final removed = $syscallUnlink(name, false);
      if (removed != 0 || file.existsSync()) {
        throw StateError('file removal $removed');
      }
      final missing = $syscallUnlink(name, false);
      if (missing != 2) throw StateError('missing errno $missing');
      final retained = $syscallUnlink(path, false);
      if (retained != directoryError || !root.existsSync()) {
        throw StateError('directory errno $retained');
      }
    }
  } finally {
    root.deleteSync();
  }
}

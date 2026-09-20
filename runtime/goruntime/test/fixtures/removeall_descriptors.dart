import 'dart:ffi';
import 'dart:io';

import 'package:goruntime/goruntime.dart';

void main() {
  final root = Directory.systemTemp.createTempSync('gart-removeall-');
  try {
    final libc = DynamicLibrary.process();
    final dup = libc.lookupFunction<Int Function(Int), int Function(int)>(
      'dup',
    );
    final close = libc.lookupFunction<Int Function(Int), int Function(int)>(
      'close',
    );
    int nextDescriptor() {
      final fd = dup(1);
      if (fd < 0) throw StateError('dup failed');
      close(fd);
      return fd;
    }

    final before = nextDescriptor();
    for (var i = 0; i < 32; i++) {
      final tree = Directory('${root.path}/tree/a/b/c')
        ..createSync(recursive: true);
      File('${tree.path}/file').writeAsStringSync('remove');
      final result = $removeAll(GoString.fromString('${root.path}/tree'));
      if (result.$3.length != 0) throw StateError(result.toString());
    }
    final after = nextDescriptor();
    if (before != after) throw StateError('$before descriptors became $after');
  } finally {
    if (root.existsSync()) root.deleteSync(recursive: true);
  }
}

import 'dart:convert';
import 'dart:ffi';
import 'dart:io';

import 'package:goruntime/goruntime.dart';
import 'package:goruntime/src/native_errno.dart';
import 'package:test/test.dart';

void main() {
  test('native openat retains errors and recovers', () {
    final libc = DynamicLibrary.process();
    final malloc = libc
        .lookupFunction<
          Pointer<Uint8> Function(Size),
          Pointer<Uint8> Function(int)
        >('malloc');
    final free = libc
        .lookupFunction<
          Void Function(Pointer<Uint8>),
          void Function(Pointer<Uint8>)
        >('free');
    final root = Directory.systemTemp.createTempSync('gart-openat-errno-');
    final file = File('${root.path}/file')..writeAsStringSync('keep');
    final cwd = Platform.isMacOS ? -2 : -100;
    final exclusive = Platform.isMacOS ? 0x200 | 0x800 : 0x40 | 0x80;
    (int, int) open(String path, int flags) {
      final bytes = utf8.encode(path);
      final name = malloc(bytes.length + 1);
      try {
        name.asTypedList(bytes.length + 1)
          ..setAll(0, bytes)
          ..[bytes.length] = 0;
        final result = nativeOpenat(cwd, name, flags, 0x180);
        return (result.value, result.error);
      } finally {
        free(name);
      }
    }

    try {
      for (var attempt = 0; attempt < 32; attempt++) {
        expect(open('${root.path}/missing', 0), (-1, 2));
        expect(open('${file.path}/child', 0), (-1, 20));
        expect(open(file.path, exclusive), (-1, 17));
        final (fd, error) = open(file.path, 0);
        expect(fd, greaterThanOrEqualTo(0));
        expect(error, 0);
        expect($syscallClose(fd), 0);
      }
      expect(file.readAsStringSync(), 'keep');
    } finally {
      root.deleteSync(recursive: true);
    }
  }, skip: !Platform.isLinux && !Platform.isMacOS);
}

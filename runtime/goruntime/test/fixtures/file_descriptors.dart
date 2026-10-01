import 'dart:ffi';
import 'dart:io';

import 'package:goruntime/goruntime.dart';

void require(bool value, String message) {
  if (!value) throw StateError(message);
}

void main() {
  final libc = DynamicLibrary.process();
  final flags = libc
      .lookupFunction<Int Function(Int, Int), int Function(int, int)>('fcntl');
  final seek = libc
      .lookupFunction<
        Int64 Function(Int, Int64, Int),
        int Function(int, int, int)
      >('lseek');
  final chdir = libc.lookupFunction<Int Function(Int), int Function(int)>(
    'fchdir',
  );
  final root = Directory.systemTemp.createTempSync('gart-file-descriptors-');
  final cwd = Directory.current.path;
  try {
    require($fileFd(0, 3) == 0, 'stdin');
    require($fileFd(0, 1) == 1, 'stdout');
    require($fileFd(0, 2) == 2, 'stderr');
    final name = GoString.fromString('${root.path}/file');
    final (handle, error) = $fileOpen(name, $openWrite);
    require(error.length == 0, 'open: $error');
    try {
      final (count, cause, _) = $fileWrite(handle, Slice.of([97, 98, 99]));
      require(count == 3 && cause.length == 0, 'write');
      final fd = $fileFd(handle, 0);
      for (var i = 0; i < 100; i++) {
        require($fileFd(handle, 0) == fd, 'descriptor identity');
      }
      require(flags(fd, 1) & 1 != 0, 'close-on-exec');
      require(seek(fd, 1, 0) == 1, 'native seek');
      final bytes = Slice.of([0]);
      final read = $fileRead(handle, bytes);
      require(
        read.$1 == 1 && read.$2.length == 0 && bytes.array[0] == 98,
        'shared position',
      );
      require($fileClose(handle).$1 == 0, 'close');
      require(flags(fd, 1) == -1, 'native descriptor closes');
      require($fileFd(handle, 0) == -1, 'closed registry slot');
    } finally {
      if ($fileFd(handle, 0) >= 0) $fileClose(handle);
    }
    final (adoptedFD, openCode) = $syscallOpen(name, 0, 0);
    require(openCode == 0, 'adopted descriptor open');
    final originalFlags = flags(adoptedFD, 1);
    require(originalFlags & 1 == 0, 'adopted descriptor lacks close-on-exec');
    final (adopted, directoryFD, _) = $fileFromFD(adoptedFD);
    try {
      require(directoryFD == -1, 'adopted regular file');
      require($fileFd(adopted, 0) == adoptedFD, 'adopted descriptor identity');
      require(flags(adoptedFD, 1) == originalFlags, 'adoption preserves flags');
    } finally {
      require($fileClose(adopted).$1 == 0, 'adopted close');
    }
    require(flags(adoptedFD, 1) == -1, 'adoption transfers ownership');
    final original = Directory('${root.path}/original')..createSync();
    final moved = '${root.path}/moved';
    final (fd, errno) = $directoryOpen(GoString.fromString(original.path));
    require(errno == 0, 'directory open: $errno');
    try {
      require(flags(fd, 1) & 1 != 0, 'directory close-on-exec');
      original.renameSync(moved);
      original.createSync();
      require(
        chdir($directoryFd(fd)) == 0,
        'directory descriptor after rename',
      );
      require(
        Directory.current.resolveSymbolicLinksSync() ==
            Directory(moved).resolveSymbolicLinksSync(),
        'original directory identity',
      );
      Directory.current = cwd;
      Directory(moved).deleteSync();
      require(
        $directoryFd(fd) == fd && flags(fd, 1) >= 0,
        'descriptor after unlink',
      );
    } finally {
      Directory.current = cwd;
      require($directoryClose(fd) == 0, 'directory close');
    }
    require(flags(fd, 1) == -1, 'directory descriptor closes');
    require(
      $directoryOpen(GoString.fromString('${root.path}/absent')).$2 == 2,
      'absent directory',
    );
    require($directoryOpen(name).$2 == 20, 'regular file is not a directory');
    require(
      $directoryOpen(GoString.fromString('${root.path}\x00suffix')).$2 == 22,
      'NUL path',
    );
    for (var i = 0; i < 100; i++) {
      final (fd, error) = $directoryOpen(GoString.fromString(root.path));
      require(error == 0, 'repeat open');
      require($directoryClose(fd) == 0 && flags(fd, 1) == -1, 'repeat close');
      final listing = $readDir(GoString.fromString(root.path));
      require(listing.$2.length == 0 && listing.$1.len == 2, 'directory names');
      require($readDir(name).$2.length != 0, 'regular file listing fails');
      final (next, nextError) = $directoryOpen(GoString.fromString(root.path));
      require(
        nextError == 0 && next == fd,
        'directory listing closes its descriptor',
      );
      require($directoryClose(next) == 0, 'listing probe close');
    }
    print('file descriptor contracts pass');
  } finally {
    Directory.current = cwd;
    root.deleteSync(recursive: true);
  }
}

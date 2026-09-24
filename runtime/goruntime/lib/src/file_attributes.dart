// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2017 The Go Authors. All rights reserved.
// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

import 'string.dart';

final _libc = DynamicLibrary.process();
final _fchmod = Platform.isMacOS
    ? _libc.lookupFunction<Int Function(Int, Uint16), int Function(int, int)>(
        'fchmod',
      )
    : _libc.lookupFunction<Int Function(Int, Uint32), int Function(int, int)>(
        'fchmod',
      );
final _fchown = _libc
    .lookupFunction<
      Int Function(Int, Uint32, Uint32),
      int Function(int, int, int)
    >('fchown');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

int $fileChmod(int fd, int mode) {
  int error;
  do {
    error = _fchmod(fd, mode) == 0 ? 0 : _errno().value;
  } while (error == 4);
  return error;
}

int $fileChown(int fd, int uid, int gid) {
  int error;
  do {
    error = _fchown(fd, uid, gid) == 0 ? 0 : _errno().value;
  } while (error == 4);
  return error;
}

final _truncate = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Int64),
      int Function(Pointer<Uint8>, int)
    >('truncate');

int $nativeTruncate(GoString path, int size) {
  final truncate = _truncate;
  return _pathAttribute(path, (name) => truncate(name, size));
}

final _ftruncate = _libc
    .lookupFunction<Int Function(Int, Int64), int Function(int, int)>(
      'ftruncate',
    );
final _fsync = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'fsync',
);
final _fcntl = _libc
    .lookupFunction<
      Int Function(Int, Int, VarArgs<(Int,)>),
      int Function(int, int, int)
    >('fcntl');

int $nativeFtruncate(int fd, int size) {
  final truncate = _ftruncate;
  final errno = _errno;
  int code;
  do {
    code = truncate(fd, size) == 0 ? 0 : errno().value;
  } while (code == 4);
  return code;
}

int $nativeFsync(int fd) {
  final sync = _fsync;
  final errno = _errno;
  if (Platform.isMacOS) {
    // Darwin uses F_FULLFSYNC to flush the device cache.
    final fullSync = _fcntl;
    int code;
    do {
      code = fullSync(fd, 0x33, 0) == 0 ? 0 : errno().value;
      if (code == 45) code = sync(fd) == 0 ? 0 : errno().value;
    } while (code == 4);
    return code;
  }
  int code;
  do {
    code = sync(fd) == 0 ? 0 : errno().value;
  } while (code == 4);
  return code;
}

final _utimensat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Pointer<Int64>, Int),
      int Function(int, Pointer<Uint8>, Pointer<Int64>, int)
    >('utimensat');
final _malloc = _libc
    .lookupFunction<
      Pointer<Uint8> Function(Size),
      Pointer<Uint8> Function(int)
    >('malloc');
final _free = _libc
    .lookupFunction<
      Void Function(Pointer<Uint8>),
      void Function(Pointer<Uint8>)
    >('free');

final _chmod = Platform.isMacOS
    ? _libc.lookupFunction<
        Int Function(Pointer<Uint8>, Uint16),
        int Function(Pointer<Uint8>, int)
      >('chmod')
    : _libc.lookupFunction<
        Int Function(Pointer<Uint8>, Uint32),
        int Function(Pointer<Uint8>, int)
      >('chmod');
final _chown = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Uint32, Uint32),
      int Function(Pointer<Uint8>, int, int)
    >('chown');
final _lchown = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Uint32, Uint32),
      int Function(Pointer<Uint8>, int, int)
    >('lchown');

int _pathAttribute(GoString path, int Function(Pointer<Uint8>) apply) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return 22;
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final errno = _errno();
    int error;
    do {
      error = apply(name) == 0 ? 0 : errno.value;
    } while (error == 4);
    return error;
  } finally {
    _free(name);
  }
}

int $nativeChmod(GoString path, int mode) =>
    _pathAttribute(path, (name) => _chmod(name, mode));

int $nativeChown(GoString path, int uid, int gid, bool follow) =>
    _pathAttribute(
      path,
      (name) => follow ? _chown(name, uid, gid) : _lchown(name, uid, gid),
    );

int $nativeChtimes(
  GoString path,
  int atime,
  bool setAtime,
  int mtime,
  bool setMtime,
) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return 22;
  final buffer = _malloc(32 + bytes.length + 1);
  if (buffer == nullptr) return 12;
  try {
    final times = buffer.cast<Int64>();
    final omitted = Platform.isMacOS ? -2 : 1073741822;
    void store(int index, int value, bool set) {
      if (!set) {
        times[index] = omitted;
        times[index + 1] = omitted;
        return;
      }
      final nsec = value % 1000000000;
      var sec = value ~/ 1000000000;
      if (value < 0 && nsec != 0) sec--;
      times[index] = sec;
      times[index + 1] = nsec;
    }

    store(0, atime, setAtime);
    store(2, mtime, setMtime);
    final name = buffer + 32;
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final errno = _errno();
    int error;
    do {
      error = _utimensat(Platform.isMacOS ? -2 : -100, name, times, 0) == 0
          ? 0
          : errno.value;
    } while (error == 4);
    return error;
  } finally {
    _free(buffer);
  }
}

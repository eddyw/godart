// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

import 'slice.dart';
import 'string.dart';

final _libc = DynamicLibrary.process();
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
final _open = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Int, VarArgs<(UnsignedInt,)>),
      int Function(Pointer<Uint8>, int, int)
    >('open');
final _close = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'close',
);
final _dup = _libc.lookupFunction<Int Function(Int), int Function(int)>('dup');
final _dup2 = _libc
    .lookupFunction<Int Function(Int, Int), int Function(int, int)>('dup2');
final _dup3 = _libc
    .lookupFunction<Int Function(Int, Int, Int), int Function(int, int, int)>(
      'dup3',
    );
final _fcntl = _libc
    .lookupFunction<
      Int Function(Int, Int, VarArgs<(Int,)>),
      int Function(int, int, int)
    >('fcntl');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

(int, int) $syscallOpen(GoString path, int flags, int mode) {
  if (path.bytes.contains(0)) return (0, 22);
  final name = _malloc(path.length + 1);
  if (name == nullptr) return (-1, 12);
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    // Resolve native bindings before open can set errno.
    final open = _open;
    final errorAddress = _errno();
    final fd = open(name, flags, mode);
    return (fd, fd < 0 ? errorAddress.value : 0);
  } finally {
    _free(name);
  }
}

int $syscallClose(int fd) => _close(fd) == 0 ? 0 : _errno().value;

(int, int) $syscallDup(int oldfd) {
  final dup = _dup;
  final errorAddress = _errno();
  final fd = dup(oldfd);
  return (fd, fd < 0 ? errorAddress.value : 0);
}

int $syscallDup2(int oldfd, int newfd) {
  final dup2 = _dup2;
  final errorAddress = _errno();
  return dup2(oldfd, newfd) < 0 ? errorAddress.value : 0;
}

int $syscallDup3(int oldfd, int newfd, int flags) {
  final dup3 = _dup3;
  final errorAddress = _errno();
  return dup3(oldfd, newfd, flags) < 0 ? errorAddress.value : 0;
}

(int, int) $syscallFcntl(int fd, int cmd, int arg) {
  final fcntl = _fcntl;
  final errorAddress = _errno();
  final value = fcntl(fd, cmd, arg);
  return (value, value < 0 ? errorAddress.value : 0);
}

final _getdents = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Uint8>, Size),
      int Function(int, Pointer<Uint8>, int)
    >('getdents64');

(int, int) $syscallReadDirent(int fd, Slice<int> data) {
  if (Platform.isMacOS) return _readDarwin(fd, data);
  final buffer = data.len == 0 ? nullptr.cast<Uint8>() : _malloc(data.len);
  if (data.len != 0 && buffer == nullptr) return (-1, 12);
  try {
    final n = _getdents(fd, buffer, data.len);
    if (n < 0) return (n, _errno().value);
    if (n != 0) {
      data.array.setRange(data.off, data.off + n, buffer.asTypedList(n));
    }
    return (n, 0);
  } finally {
    _free(buffer);
  }
}

final _seek = _libc
    .lookupFunction<
      Int64 Function(Int, Int64, Int),
      int Function(int, int, int)
    >('lseek');
final _openat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Int, VarArgs<()>),
      int Function(int, Pointer<Uint8>, int)
    >('openat');
final _fdopendir = _libc
    .lookupFunction<Pointer<Void> Function(Int), Pointer<Void> Function(int)>(
      Abi.current() == Abi.macosX64 ? r'fdopendir$INODE64' : 'fdopendir',
    );
final _closedir = _libc
    .lookupFunction<Int Function(Pointer<Void>), int Function(Pointer<Void>)>(
      'closedir',
    );
final _readdir = _libc
    .lookupFunction<
      Pointer<Uint8> Function(Pointer<Void>),
      Pointer<Uint8> Function(Pointer<Void>)
    >(Abi.current() == Abi.macosX64 ? r'readdir$INODE64' : 'readdir');

// Darwin stores an entry count in the descriptor offset, as Go does.
// openat creates independent directory state. dup would share the offset.
(int, int) _readDarwin(int fd, Slice<int> data) {
  var skip = _seek(fd, 0, 1);
  if (skip < 0) return (0, _errno().value);
  final dot = _malloc(2);
  if (dot == nullptr) return (0, 12);
  final int copy;
  try {
    dot[0] = 46;
    dot[1] = 0;
    copy = _openat(fd, dot, 0);
    if (copy < 0) return (0, _errno().value);
  } finally {
    _free(dot);
  }
  final directory = _fdopendir(copy);
  if (directory == nullptr) {
    final error = _errno().value;
    _close(copy);
    return (0, error);
  }
  try {
    var count = 0;
    var n = 0;
    while (true) {
      _errno().value = 0;
      final entry = _readdir(directory);
      if (entry == nullptr) {
        final error = _errno().value;
        if (error != 0) return (n, error);
        break;
      }
      if (skip > 0) {
        skip--;
        count++;
        continue;
      }
      final length = (entry + 16).cast<Uint16>().value;
      if (length > data.len - n) break;
      data.array.setRange(
        data.off + n,
        data.off + n + length,
        entry.asTypedList(length),
      );
      n += length;
      count++;
    }
    final position = _seek(fd, count, 0);
    return (n, position < 0 ? _errno().value : 0);
  } finally {
    _closedir(directory);
  }
}

(int, int) $syscallSeek(int fd, int offset, int whence) {
  final seek = _seek;
  final errno = _errno;
  // Darwin device files can return a negative offset without an error.
  final position = seek(fd, offset, whence);
  return (position, position == -1 ? errno().value : 0);
}

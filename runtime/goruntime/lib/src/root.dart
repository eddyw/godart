// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

import 'file_stat.dart';
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
final _openat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Int, VarArgs<(UnsignedInt,)>),
      int Function(int, Pointer<Uint8>, int, int)
    >('openat');
final _mkdirat = Platform.isMacOS
    ? _libc.lookupFunction<
        Int Function(Int, Pointer<Uint8>, Uint16),
        int Function(int, Pointer<Uint8>, int)
      >('mkdirat')
    : _libc.lookupFunction<
        Int Function(Int, Pointer<Uint8>, Uint32),
        int Function(int, Pointer<Uint8>, int)
      >('mkdirat');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );
final _cwd = Platform.isMacOS ? -2 : -100;
final _cloexec = Platform.isMacOS ? 0x1000000 : 0x80000;
final _nofollow = Platform.isMacOS ? 0x100 : 0x20000;
final _directory = Platform.isMacOS ? 0x100000 : 0x10000;

(int, int) _open(int parent, GoString path, int flags, int mode) {
  if (path.bytes.contains(0)) return (-1, 22);
  final name = _malloc(path.length + 1);
  if (name == nullptr) return (-1, 12);
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final open = _openat;
    final errno = _errno();
    int fd;
    int code;
    do {
      fd = open(parent, name, flags, mode);
      code = fd < 0 ? errno.value : 0;
    } while (code == 4);
    return (fd, code);
  } finally {
    _free(name);
  }
}

(int, int) $rootOpen(GoString path) =>
    $hasNativeStat ? _open(_cwd, path, _cloexec, 0) : (-1, 38);

(int, int) $rootOpenAt(int parent, GoString path) =>
    _open(parent, path, _cloexec | _nofollow, 0);

(int, int) $rootOpenDir(int parent, GoString path) =>
    _open(parent, path, _cloexec | _nofollow | _directory, 0);

(int, int) rootOpenFileDescriptor(
  int parent,
  GoString path,
  int flags,
  int mode,
) => _open(parent, path, flags | _cloexec | _nofollow, mode);

int $rootMkdir(int parent, GoString path, int mode) {
  if (path.bytes.contains(0)) return 22;
  final name = _malloc(path.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final mkdir = _mkdirat;
    final errno = _errno();
    int code;
    do {
      code = mkdir(parent, name, mode) == 0 ? 0 : errno.value;
    } while (code == 4);
    return code;
  } finally {
    _free(name);
  }
}

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

import 'string.dart';

final _supported = Platform.isLinux || Platform.isMacOS;
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
      Int Function(Pointer<Uint8>, Int),
      int Function(Pointer<Uint8>, int)
    >('open');
final _mkdir = Platform.isMacOS
    ? _libc.lookupFunction<
        Int Function(Pointer<Uint8>, Uint16),
        int Function(Pointer<Uint8>, int)
      >('mkdir')
    : _libc.lookupFunction<
        Int Function(Pointer<Uint8>, Uint32),
        int Function(Pointer<Uint8>, int)
      >('mkdir');
final _close = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'close',
);
final _fchdir = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'fchdir',
);
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );
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
final _strerror = _libc
    .lookupFunction<Pointer<Uint8> Function(Int), Pointer<Uint8> Function(int)>(
      'strerror',
    );

Never _readError(GoString path, int error) {
  final message = _strerror(error);
  final bytes = <int>[];
  for (var i = 0; message[i] != 0; i++) {
    bytes.add(message[i]);
  }
  throw FileSystemException(
    'Directory listing failed',
    path.toDartString(),
    OSError(GoString.fromBytes(bytes).toDartString(), error),
  );
}

List<GoString> readDirectoryNames(GoString path) {
  final (fd, error) = $directoryOpen(path);
  if (error != 0) _readError(path, error);
  final directory = _fdopendir(fd);
  if (directory == nullptr) {
    final error = _errno().value;
    _close(fd);
    _readError(path, error);
  }
  final names = <GoString>[];
  try {
    while (true) {
      _errno().value = 0;
      final entry = _readdir(directory);
      if (entry == nullptr) {
        final error = _errno().value;
        if (error == 4) continue;
        if (error != 0) _readError(path, error);
        return names;
      }
      // Darwin can retain a deleted entry with a zero inode.
      if (Platform.isMacOS && entry.cast<Uint64>().value == 0) continue;
      final recordLength = (entry + 16).cast<Uint16>().value;
      final nameOffset = Platform.isMacOS ? 21 : 19;
      if (recordLength <= nameOffset) _readError(path, 5);
      final bytes = (entry + nameOffset).asTypedList(recordLength - nameOffset);
      final end = bytes.indexOf(0);
      if (end < 0) _readError(path, 5);
      if (end == 0 ||
          end == 1 && bytes[0] == 46 ||
          end == 2 && bytes[0] == 46 && bytes[1] == 46) {
        continue;
      }
      names.add(GoString.fromBytes(bytes.sublist(0, end)));
    }
  } finally {
    // fdopendir transfers descriptor ownership to closedir.
    _closedir(directory);
  }
}

(int, int) $directoryOpen(GoString path) {
  // Other hosts retain path-based directory access without a native descriptor.
  if (!_supported) return (-1, 0);
  final bytes = path.bytes;
  if (bytes.contains(0)) return (-1, 22);
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return (-1, 12);
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final flags = Platform.isMacOS ? 0x1000000 | 0x100000 : 0x80000 | 0x10000;
    while (true) {
      final fd = _open(name, flags);
      final error = fd < 0 ? _errno().value : 0;
      if (error != 4) return (fd, error);
    }
  } finally {
    _free(name);
  }
}

int $directoryFd(int fd) {
  if (!_supported) throw UnsupportedError('native directory descriptors');
  return fd;
}

int $directoryClose(int fd) {
  if (!_supported) return 0;
  return _close(fd) == 0 ? 0 : _errno().value;
}

int $directoryChdir(int fd) => _fchdir(fd) == 0 ? 0 : _errno().value;

int $directoryCreate(GoString path, int mode) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return 22;
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    int error;
    do {
      error = _mkdir(name, mode) == 0 ? 0 : _errno().value;
    } while (error == 4);
    return error;
  } finally {
    _free(name);
  }
}

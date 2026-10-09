// Copyright 2018 The Go Authors. All rights reserved.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'native_errno.dart';
import 'string.dart';

final bool hasDescriptorRemoval = switch (Abi.current()) {
  Abi.linuxX64 || Abi.linuxArm64 || Abi.macosX64 || Abi.macosArm64 => true,
  _ => false,
};

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
final _close = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'close',
);
final _fdopendir = _libc
    .lookupFunction<Pointer<Void> Function(Int), Pointer<Void> Function(int)>(
      Abi.current() == Abi.macosX64 ? r'fdopendir$INODE64' : 'fdopendir',
    );
final _closedir = _libc
    .lookupFunction<Int Function(Pointer<Void>), int Function(Pointer<Void>)>(
      'closedir',
    );
final _dirfd = _libc
    .lookupFunction<Int Function(Pointer<Void>), int Function(Pointer<Void>)>(
      'dirfd',
    );
final _readdir = _libc
    .lookupFunction<
      Pointer<Uint8> Function(Pointer<Void>),
      Pointer<Uint8> Function(Pointer<Void>)
    >(Abi.current() == Abi.macosX64 ? r'readdir$INODE64' : 'readdir');
final _errnoLocation = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );
final _cwd = Platform.isMacOS ? -2 : -100;
final _closeOnExec = Platform.isMacOS ? 0x1000000 : 0x80000;
final _directoryFlags =
    _closeOnExec |
    (Platform.isMacOS
        ? 0x100000
        : (Abi.current() == Abi.linuxArm64 ? 0x4000 : 0x10000));
final _noFollow = Platform.isMacOS
    ? 0x100
    : (Abi.current() == Abi.linuxArm64 ? 0x8000 : 0x20000);
final _removeDirectory = Platform.isMacOS ? 0x80 : 0x200;

final class _RemovalError implements Exception {
  final String op;
  final Uint8List path;
  final int errno;

  _RemovalError(this.op, this.path, this.errno);
}

Pointer<Uint8> _name(Uint8List bytes) {
  if (bytes.contains(0)) throw _RemovalError('removeall', bytes, 22);
  final p = _malloc(bytes.length + 1);
  if (p == nullptr) throw _RemovalError('removeall', bytes, 12);
  p.asTypedList(bytes.length + 1)
    ..setRange(0, bytes.length, bytes)
    ..[bytes.length] = 0;
  return p;
}

int _unlink(int parent, Uint8List name, int flags) {
  try {
    final p = _name(name);
    try {
      return nativeUnlinkat(parent, p, flags).error;
    } finally {
      _free(p);
    }
  } on _RemovalError catch (error) {
    return error.errno;
  }
}

int $syscallUnlink(GoString path, bool directory) =>
    _unlink(_cwd, path.bytes, directory ? _removeDirectory : 0);

int removeDescriptors(GoString path) => $rootRemove(_cwd, path);

int $rootRemove(int parent, GoString path) {
  int fileError;
  do {
    fileError = _unlink(parent, path.bytes, 0);
  } while (fileError == 4);
  if (fileError == 0) return 0;
  int directoryError;
  do {
    directoryError = _unlink(parent, path.bytes, _removeDirectory);
  } while (directoryError == 4);
  if (directoryError == 0) return 0;
  // ENOTDIR identifies the file error on both Linux and macOS.
  return directoryError == 20 ? fileError : directoryError;
}

(int, int) _open(int parent, Uint8List name, int flags) {
  try {
    final p = _name(name);
    try {
      final result = nativeOpenat(parent, p, flags, 0);
      return (result.value, result.error);
    } finally {
      _free(p);
    }
  } on _RemovalError catch (error) {
    return (-1, error.errno);
  }
}

Uint8List _join(Uint8List parent, Uint8List child) =>
    Uint8List.fromList([...parent, 47, ...child]);

// Each child operation uses a directory descriptor. A deep child does not
// require an absolute system path.
_RemovalError? _removeFrom(int parent, Uint8List base) {
  final first = _unlink(parent, base, 0);
  if (first == 0 || first == 2) return null;
  if (first != 21 && first != 1 && first != 13) {
    return _RemovalError('unlinkat', base, first);
  }
  _RemovalError? recurseError;
  while (true) {
    final (fd, openError) = _open(parent, base, _directoryFlags | _noFollow);
    if (fd < 0) {
      if (openError == 2) return null;
      if (openError == 20) return _RemovalError('unlinkat', base, first);
      final loopError = Platform.isMacOS ? 62 : 40;
      recurseError = _RemovalError(
        'openfdat',
        base,
        openError == loopError ? first : openError,
      );
      break;
    }
    final directory = _fdopendir(fd);
    if (directory == nullptr) {
      final error = _errnoLocation().value;
      _close(fd);
      recurseError = _RemovalError('openfdat', base, error);
      break;
    }
    var count = 0;
    try {
      while (true) {
        final names = <Uint8List>[];
        while (names.length < 1024) {
          _errnoLocation().value = 0;
          final entry = _readdir(directory);
          if (entry == nullptr) {
            final error = _errnoLocation().value;
            if (error == 2) return null;
            if (error != 0) return _RemovalError('readdirnames', base, error);
            break;
          }
          final recordLength = (entry + 16).cast<Uint16>().value;
          final nameOffset = Platform.isMacOS ? 21 : 19;
          if (recordLength <= nameOffset) {
            return _RemovalError('readdirnames', base, 5);
          }
          final bytes = (entry + nameOffset).asTypedList(
            recordLength - nameOffset,
          );
          final end = bytes.indexOf(0);
          if (end < 0) return _RemovalError('readdirnames', base, 5);
          if (end == 0 ||
              end == 1 && bytes[0] == 46 ||
              end == 2 && bytes[0] == 46 && bytes[1] == 46) {
            continue;
          }
          names.add(Uint8List.fromList(bytes.sublist(0, end)));
        }
        count = names.length;
        var errors = 0;
        for (final name in names) {
          final error = _removeFrom(_dirfd(directory), name);
          if (error != null) {
            errors++;
            recurseError ??= _RemovalError(
              error.op,
              _join(base, error.path),
              error.errno,
            );
          }
        }
        if (errors != 1024) break;
      }
    } finally {
      // fdopendir owns fd after success. closedir also closes that descriptor.
      _closedir(directory);
    }
    // Deletions can shuffle directory entries. Reopen after a partial batch
    // of successful deletions, but advance when all entries fail.
    if (count < 1024) break;
  }
  final error = _unlink(parent, base, _removeDirectory);
  if (error == 0 || error == 2) return null;
  return recurseError ?? _RemovalError('unlinkat', base, error);
}

(GoString, GoString, int) removeAllDescriptors(GoString path) {
  final direct = removeDescriptors(path);
  if (direct == 0 || direct == 2) {
    return (GoString.empty, GoString.empty, 0);
  }
  final bytes = path.bytes;
  final slash = bytes.lastIndexOf(47);
  final parent = slash < 0
      ? Uint8List.fromList([46])
      : Uint8List.sublistView(bytes, 0, slash == 0 ? 1 : slash);
  final base = slash < 0 ? bytes : Uint8List.sublistView(bytes, slash + 1);
  final (fd, openError) = _open(_cwd, parent, _closeOnExec);
  _RemovalError? error;
  if (fd < 0) {
    if (openError == 2) {
      return (GoString.empty, GoString.empty, 0);
    }
    error = _RemovalError('open', parent, openError);
  } else {
    try {
      final childError = _removeFrom(fd, base);
      if (childError != null) {
        error = _RemovalError(
          childError.op,
          _join(parent, childError.path),
          childError.errno,
        );
      }
    } finally {
      _close(fd);
    }
  }
  if (error == null) return (GoString.empty, GoString.empty, 0);
  return (
    GoString.fromString(error.op),
    GoString.fromBytes(error.path),
    error.errno,
  );
}

int $rootRemoveAll(int parent, GoString path) =>
    _removeFrom(parent, path.bytes)?.errno ?? 0;

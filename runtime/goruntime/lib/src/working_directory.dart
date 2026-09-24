// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

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
final _chdir = _libc
    .lookupFunction<Int Function(Pointer<Uint8>), int Function(Pointer<Uint8>)>(
      'chdir',
    );
final _getcwd = _libc
    .lookupFunction<
      Pointer<Uint8> Function(Pointer<Uint8>, Size),
      Pointer<Uint8> Function(Pointer<Uint8>, int)
    >('getcwd');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

int $nativeChdir(GoString path) {
  if (path.bytes.contains(0)) return 22;
  final name = _malloc(path.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    return _chdir(name) == 0 ? 0 : _errno().value;
  } finally {
    _free(name);
  }
}

(GoString, int) $nativeGetwd() {
  for (var size = 256; ; size *= 2) {
    final result = _getwd(size);
    if (result.$2 != 34) return result;
  }
}

(GoString, int) $syscallGetwd() {
  if (!Platform.isMacOS) return _linuxGetwd();
  final buffer = _malloc(1024);
  if (buffer == nullptr) return (GoString.empty, 12);
  try {
    final bytes = buffer.asTypedList(1024)..fillRange(0, 1024, 0);
    // Go's Darwin wrapper inspects the buffer even when getcwd returns null.
    _getcwd(buffer, 1024);
    final end = bytes.indexOf(0);
    final size = end < 0 ? bytes.length : end;
    if (size == 0) return (GoString.empty, 22);
    return (GoString.fromBytes(bytes.sublist(0, size)), 0);
  } finally {
    _free(buffer);
  }
}

(GoString, int) _getwd(int size) {
  final buffer = _malloc(size);
  if (buffer == nullptr) return (GoString.empty, 12);
  try {
    final errno = _errno;
    Pointer<Uint8> result;
    int error;
    do {
      result = _getcwd(buffer, size);
      error = result == nullptr ? errno().value : 0;
    } while (error == 4);
    if (error != 0) return (GoString.empty, error);
    final bytes = buffer.asTypedList(size);
    return (GoString.fromBytes(bytes.sublist(0, bytes.indexOf(0))), 0);
  } finally {
    _free(buffer);
  }
}

final _syscallGetcwd = _libc
    .lookupFunction<
      IntPtr Function(IntPtr, VarArgs<(Pointer<Uint8>, Size)>),
      int Function(int, Pointer<Uint8>, int)
    >('syscall');

(GoString, int) _linuxGetwd() {
  final buffer = _malloc(4096);
  if (buffer == nullptr) return (GoString.empty, 12);
  try {
    final errno = _errno;
    final number = Abi.current() == Abi.linuxArm64 ? 17 : 79;
    int count;
    int error;
    do {
      count = _syscallGetcwd(number, buffer, 4096);
      error = count < 0 ? errno().value : 0;
    } while (error == 4);
    if (error != 0) return (GoString.empty, error);
    if (count < 1 || count > 4096 || buffer[count - 1] != 0) {
      return (GoString.empty, 22);
    }
    if (buffer[0] != 47) return (GoString.empty, 2);
    return (GoString.fromBytes(buffer.asTypedList(count - 1)), 0);
  } finally {
    _free(buffer);
  }
}

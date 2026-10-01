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
final _readlink = _libc
    .lookupFunction<
      IntPtr Function(Pointer<Uint8>, Pointer<Uint8>, Size),
      int Function(Pointer<Uint8>, Pointer<Uint8>, int)
    >('readlink');
final _readlinkat = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Uint8>, Pointer<Uint8>, Size),
      int Function(int, Pointer<Uint8>, Pointer<Uint8>, int)
    >('readlinkat');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

(GoString, int) $nativeReadlink(GoString path) => _readLink(path, _readlink);

(GoString, int) $rootReadlink(int parent, GoString path) => _readLink(
  path,
  (name, buffer, size) => _readlinkat(parent, name, buffer, size),
);

(GoString, int) _readLink(
  GoString path,
  int Function(Pointer<Uint8>, Pointer<Uint8>, int) readlink,
) {
  if (path.bytes.contains(0)) return (GoString.empty, 22);
  final name = _malloc(path.length + 1);
  if (name == nullptr) return (GoString.empty, 12);
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    for (var size = 128; ; size *= 2) {
      final buffer = _malloc(size);
      if (buffer == nullptr) return (GoString.empty, 12);
      try {
        int count;
        int error;
        do {
          count = readlink(name, buffer, size);
          error = count < 0 ? _errno().value : 0;
        } while (error == 4);
        if (error != 0) return (GoString.empty, error);
        if (count < size) {
          return (GoString.fromBytes(buffer.asTypedList(count)), 0);
        }
      } finally {
        _free(buffer);
      }
    }
  } finally {
    _free(name);
  }
}

(int, int) $syscallReadlink(GoString path, Slice<int> bytes) {
  if (path.bytes.contains(0)) return (0, 22);
  final readlink = _readlink;
  final errno = _errno;
  final name = _malloc(path.length + 1 + bytes.len);
  if (name == nullptr) return (-1, 12);
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final buffer = name + path.length + 1;
    final count = readlink(name, buffer, bytes.len);
    if (count < 0) return (count, errno().value);
    if (count > 0) {
      bytes.array.setRange(
        bytes.off,
        bytes.off + count,
        buffer.asTypedList(count),
      );
    }
    return (count, 0);
  } finally {
    _free(name);
  }
}

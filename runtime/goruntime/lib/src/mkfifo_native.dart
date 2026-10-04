// Copyright 2009,2010 The Go Authors. All rights reserved.
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
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isLinux ? '__errno_location' : '__error',
    );
final _mkfifo = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Uint16),
      int Function(Pointer<Uint8>, int)
    >('mkfifo');

final _mknod = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Uint32, Uint64),
      int Function(Pointer<Uint8>, int, int)
    >('mknod');

final _darwinMknod = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Uint16, Int32),
      int Function(Pointer<Uint8>, int, int)
    >('mknod');

int $syscallDarwinMknod(GoString path, int mode, int dev) {
  if (path.bytes.contains(0)) return 22;
  final text = _malloc(path.length + 1);
  if (text == nullptr) return 12;
  try {
    text.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final create = _darwinMknod;
    final errorAddress = _errno();
    return create(text, mode, dev) == 0 ? 0 : errorAddress.value;
  } finally {
    _free(text);
  }
}

int $syscallMkfifo(GoString path, int mode) {
  if (path.bytes.contains(0)) return 22;
  final text = _malloc(path.length + 1);
  if (text == nullptr) return 12;
  try {
    text.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    if (Platform.isLinux) {
      final create = _mknod;
      final errorAddress = _errno();
      return create(text, mode | 0x1000, 0) == 0 ? 0 : errorAddress.value;
    }
    final create = _mkfifo;
    final errorAddress = _errno();
    return create(text, mode) == 0 ? 0 : errorAddress.value;
  } finally {
    _free(text);
  }
}

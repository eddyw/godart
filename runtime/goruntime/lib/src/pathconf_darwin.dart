// Copyright 2009,2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';

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
      '__error',
    );
final _pathconf = _libc
    .lookupFunction<
      Long Function(Pointer<Uint8>, Int),
      int Function(Pointer<Uint8>, int)
    >('pathconf');
final _fpathconf = _libc
    .lookupFunction<Long Function(Int, Int), int Function(int, int)>(
      'fpathconf',
    );

// Go's Darwin syscall adapter detects errors from the low 32 result bits.
(int, int) $syscallPathconf(GoString path, int name) {
  if (path.bytes.contains(0)) return (0, 22);
  final text = _malloc(path.length + 1);
  if (text == nullptr) return (0, 12);
  try {
    text.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final query = _pathconf;
    final errorAddress = _errno();
    final value = query(text, name);
    return (value, value.toSigned(32) == -1 ? errorAddress.value : 0);
  } finally {
    _free(text);
  }
}

(int, int) $syscallFpathconf(int fd, int name) {
  final query = _fpathconf;
  final errorAddress = _errno();
  final value = query(fd, name);
  return (value, value.toSigned(32) == -1 ? errorAddress.value : 0);
}

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
final _mkfifo = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Uint16),
      int Function(Pointer<Uint8>, int)
    >('mkfifo');

int $syscallMkfifo(GoString path, int mode) {
  if (path.bytes.contains(0)) return 22;
  final text = _malloc(path.length + 1);
  if (text == nullptr) return 12;
  try {
    text.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final create = _mkfifo;
    final errorAddress = _errno();
    return create(text, mode) == 0 ? 0 : errorAddress.value;
  } finally {
    _free(text);
  }
}

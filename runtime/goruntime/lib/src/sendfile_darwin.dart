// Copyright 2009,2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';

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
final _sendfile = _libc
    .lookupFunction<
      Int Function(Int, Int, Int64, Pointer<Int64>, Pointer<Void>, Int),
      int Function(int, int, int, Pointer<Int64>, Pointer<Void>, int)
    >('sendfile');

(int, int) $syscallDarwinSendfile(int outfd, int infd, int offset, int count) {
  final length = _malloc(sizeOf<Int64>()).cast<Int64>();
  if (length == nullptr) return (0, 12);
  try {
    length.value = count;
    final sendfile = _sendfile;
    final errorAddress = _errno();
    final result = sendfile(infd, outfd, offset, length, nullptr, 0);
    final code = result == 0 ? 0 : errorAddress.value;
    return (length.value, code);
  } finally {
    _free(length.cast<Uint8>());
  }
}

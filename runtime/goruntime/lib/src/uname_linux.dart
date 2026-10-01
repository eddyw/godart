// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';

import 'slice.dart';

final _libc = DynamicLibrary.process();
final _malloc = _libc
    .lookupFunction<Pointer<Int8> Function(Size), Pointer<Int8> Function(int)>(
      'malloc',
    );
final _free = _libc
    .lookupFunction<Void Function(Pointer<Int8>), void Function(Pointer<Int8>)>(
      'free',
    );
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      '__errno_location',
    );
final _uname = _libc
    .lookupFunction<Int Function(Pointer<Int8>), int Function(Pointer<Int8>)>(
      'uname',
    );

(Slice<int>, int) $syscallUname(bool output) {
  // Linux Utsname has 6 arrays of 65 signed bytes on AMD64 and ARM64.
  final buffer = output ? _malloc(390) : nullptr.cast<Int8>();
  if (output && buffer == nullptr) return (Slice<int>.nil(), 12);
  try {
    final query = _uname;
    final errorAddress = _errno();
    final result = query(buffer);
    final code = result == 0 ? 0 : errorAddress.value;
    if (code != 0) return (Slice<int>.nil(), code);
    return (Slice.of(buffer.asTypedList(390).toList()), 0);
  } finally {
    _free(buffer);
  }
}

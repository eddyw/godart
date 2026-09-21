// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

final _libc = DynamicLibrary.process();
final _fchmod = Platform.isMacOS
    ? _libc.lookupFunction<Int Function(Int, Uint16), int Function(int, int)>(
        'fchmod',
      )
    : _libc.lookupFunction<Int Function(Int, Uint32), int Function(int, int)>(
        'fchmod',
      );
final _fchown = _libc
    .lookupFunction<
      Int Function(Int, Uint32, Uint32),
      int Function(int, int, int)
    >('fchown');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

int $fileChmod(int fd, int mode) {
  int error;
  do {
    error = _fchmod(fd, mode) == 0 ? 0 : _errno().value;
  } while (error == 4);
  return error;
}

int $fileChown(int fd, int uid, int gid) {
  int error;
  do {
    error = _fchown(fd, uid, gid) == 0 ? 0 : _errno().value;
  } while (error == 4);
  return error;
}

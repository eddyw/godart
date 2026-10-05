// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

import 'slice.dart';

final _libc = DynamicLibrary.process();
final _mmap = _libc
    .lookupFunction<
      Pointer<Uint8> Function(Pointer<Void>, Size, Int, Int, Int, Int64),
      Pointer<Uint8> Function(Pointer<Void>, int, int, int, int, int)
    >('mmap');
final _munmap = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Size),
      int Function(Pointer<Uint8>, int)
    >('munmap');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

final _mapFailed = Pointer<Uint8>.fromAddress(-1);
final _active = Map<List<int>, (Pointer<Uint8>, int)>.identity();

(Slice<int>, int) $syscallMmap(
  int fd,
  int offset,
  int length,
  int prot,
  int flags,
) {
  final mmap = _mmap;
  final failed = _mapFailed;
  final errorAddress = _errno();
  final address = mmap(nullptr, length, prot, flags, fd, offset);
  final code = address == failed ? errorAddress.value : 0;
  if (code != 0) return (Slice<int>.nil(), code);
  final bytes = address.asTypedList(length);
  _active[bytes] = (address, length);
  return (Slice<int>.of(bytes), 0);
}

int $syscallMunmap(Slice<int> data) {
  final mapping = _active[data.array];
  if (mapping == null || data.off != 0 || data.len != mapping.$2) return 22;
  final munmap = _munmap;
  final errorAddress = _errno();
  final result = munmap(mapping.$1, mapping.$2);
  final code = result < 0 ? errorAddress.value : 0;
  if (code == 0) _active.remove(data.array);
  return code;
}

final _mprotectDarwin = _libc.lookupFunction<
  Int Function(Pointer<Uint8>, Size, Int),
  int Function(Pointer<Uint8>, int, int)
>('mprotect');
final _mprotectLinux = _libc.lookupFunction<
  Long Function(Long, VarArgs<(Pointer<Uint8>, Size, UnsignedLong)>),
  int Function(int, Pointer<Uint8>, int, int)
>('syscall');

int $syscallMprotect(Slice<int> data, int prot) {
  final mapping = _active[data.array];
  if (data.len == 0 || mapping == null) return -1;
  final address = mapping.$1 + data.off;
  if (Platform.isMacOS) {
    final protect = _mprotectDarwin;
    final errorAddress = _errno();
    final result = protect(address, data.len, prot);
    return result < 0 ? errorAddress.value : 0;
  }
  final protect = _mprotectLinux;
  final number = Abi.current() == Abi.linuxArm64 ? 226 : 10;
  final errorAddress = _errno();
  final result = protect(number, address, data.len, prot);
  return result < 0 ? errorAddress.value : 0;
}

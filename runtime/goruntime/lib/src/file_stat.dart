// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'string.dart';

final bool $hasNativeStat = switch (Abi.current()) {
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
final _stat = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>),
      int Function(Pointer<Uint8>, Pointer<Uint8>)
    >(Abi.current() == Abi.macosX64 ? 'stat64' : 'stat');
final _lstat = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>),
      int Function(Pointer<Uint8>, Pointer<Uint8>)
    >(Abi.current() == Abi.macosX64 ? 'lstat64' : 'lstat');
final _fstat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>),
      int Function(int, Pointer<Uint8>)
    >(Abi.current() == Abi.macosX64 ? 'fstat64' : 'fstat');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

(int, int, int, int, int, int, int) _readStat(
  int Function(Pointer<Uint8>) read,
) {
  final buffer = _malloc(144);
  if (buffer == nullptr) return (0, 0, 0, 0, 0, 0, 12);
  try {
    int error;
    do {
      error = read(buffer) == 0 ? 0 : _errno().value;
    } while (error == 4);
    if (error != 0) return (0, 0, 0, 0, 0, 0, error);
    final data = ByteData.sublistView(buffer.asTypedList(144));
    if (Platform.isMacOS) {
      return (
        data.getInt64(96, Endian.host),
        data.getUint16(4, Endian.host),
        data.getInt64(48, Endian.host),
        data.getInt64(56, Endian.host),
        data.getUint32(0, Endian.host),
        data.getUint64(8, Endian.host),
        0,
      );
    }
    return (
      data.getInt64(48, Endian.host),
      data.getUint32(Abi.current() == Abi.linuxX64 ? 24 : 16, Endian.host),
      data.getInt64(88, Endian.host),
      data.getInt64(96, Endian.host),
      data.getUint64(0, Endian.host),
      data.getUint64(8, Endian.host),
      0,
    );
  } finally {
    _free(buffer);
  }
}

(int, int, int, int, int, int, int) $nativeStat(GoString path, bool follow) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return (0, 0, 0, 0, 0, 0, 22);
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return (0, 0, 0, 0, 0, 0, 12);
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    return _readStat(
      (buffer) => follow ? _stat(name, buffer) : _lstat(name, buffer),
    );
  } finally {
    _free(name);
  }
}

(int, int, int, int, int, int, int) $nativeFstat(int fd) =>
    _readStat((buffer) => _fstat(fd, buffer));

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'slice.dart';
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

(Slice<int>, int) _readStat(int Function(Pointer<Uint8>) read) {
  final buffer = _malloc(144);
  if (buffer == nullptr) return (Slice<int>.nil(), 12);
  try {
    int error;
    do {
      error = read(buffer) == 0 ? 0 : _errno().value;
    } while (error == 4);
    if (error != 0) return (Slice<int>.nil(), error);
    return _decodeStat(buffer);
  } finally {
    _free(buffer);
  }
}

(Slice<int>, int) _decodeStat(Pointer<Uint8> buffer) {
  final data = ByteData.sublistView(buffer.asTypedList(144));
  if (Platform.isMacOS) {
    return (
      Slice.of([
        data.getInt64(96, Endian.host),
        data.getUint16(4, Endian.host),
        data.getInt64(48, Endian.host),
        data.getInt64(56, Endian.host),
        data.getInt32(0, Endian.host),
        data.getUint64(8, Endian.host),
        data.getUint16(6, Endian.host),
        data.getUint32(16, Endian.host),
        data.getUint32(20, Endian.host),
        data.getInt32(24, Endian.host),
        data.getInt32(112, Endian.host),
        data.getInt64(104, Endian.host),
        data.getInt64(32, Endian.host),
        data.getInt64(40, Endian.host),
        data.getInt64(64, Endian.host),
        data.getInt64(72, Endian.host),
        data.getInt64(80, Endian.host),
        data.getInt64(88, Endian.host),
        data.getUint32(116, Endian.host),
        data.getUint32(120, Endian.host),
      ]),
      0,
    );
  }
  final x64 = Abi.current() == Abi.linuxX64;
  return (
    Slice.of([
      data.getInt64(48, Endian.host),
      data.getUint32(x64 ? 24 : 16, Endian.host),
      data.getInt64(88, Endian.host),
      data.getInt64(96, Endian.host),
      data.getUint64(0, Endian.host),
      data.getUint64(8, Endian.host),
      x64 ? data.getUint64(16, Endian.host) : data.getUint32(20, Endian.host),
      data.getUint32(x64 ? 28 : 24, Endian.host),
      data.getUint32(x64 ? 32 : 28, Endian.host),
      data.getUint64(x64 ? 40 : 32, Endian.host),
      x64 ? data.getInt64(56, Endian.host) : data.getInt32(56, Endian.host),
      data.getInt64(64, Endian.host),
      data.getInt64(72, Endian.host),
      data.getInt64(80, Endian.host),
      data.getInt64(104, Endian.host),
      data.getInt64(112, Endian.host),
      0,
      0,
      0,
      0,
    ]),
    0,
  );
}

(Slice<int>, int) $nativeStat(GoString path, bool follow) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return (Slice<int>.nil(), 22);
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return (Slice<int>.nil(), 12);
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

(Slice<int>, int) $nativeFstat(int fd) =>
    _readStat((buffer) => _fstat(fd, buffer));

final _fstatat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Pointer<Uint8>, Int),
      int Function(int, Pointer<Uint8>, Pointer<Uint8>, int)
    >(Abi.current() == Abi.macosX64 ? 'fstatat64' : 'fstatat');

(Slice<int>, int) $nativeLstatat(int fd, GoString path) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return (Slice<int>.nil(), 22);
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return (Slice<int>.nil(), 12);
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    return _readStat(
      (buffer) => _fstatat(fd, name, buffer, Platform.isMacOS ? 0x20 : 0x100),
    );
  } finally {
    _free(name);
  }
}

(int, int, int, int) $syscallFaccessatStat(int fd, GoString path, int flags) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return (0, 0, 0, 22);
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return (0, 0, 0, 12);
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final buffer = _malloc(144);
    if (buffer == nullptr) return (0, 0, 0, 12);
    try {
      final stat = _fstatat;
      final errorAddress = _errno();
      if (stat(fd, name, buffer, flags) != 0) {
        return (0, 0, 0, errorAddress.value);
      }
      final data = ByteData.sublistView(buffer.asTypedList(144));
      final x64 = Abi.current() == Abi.linuxX64;
      return (
        data.getUint32(x64 ? 24 : 16, Endian.host),
        data.getUint32(x64 ? 28 : 24, Endian.host),
        data.getUint32(x64 ? 32 : 28, Endian.host),
        0,
      );
    } finally {
      _free(buffer);
    }
  } finally {
    _free(name);
  }
}

(Slice<int>, int) $nativeFstatat(
  int fd,
  GoString path,
  bool output,
  int flags,
) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return (Slice<int>.nil(), 22);
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return (Slice<int>.nil(), 12);
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final buffer = output ? _malloc(144) : nullptr.cast<Uint8>();
    if (output && buffer == nullptr) return (Slice<int>.nil(), 12);
    try {
      final stat = _fstatat;
      final errorAddress = _errno();
      if (stat(fd, name, buffer, flags) != 0) {
        return (Slice<int>.nil(), errorAddress.value);
      }
      return _decodeStat(buffer);
    } finally {
      _free(buffer);
    }
  } finally {
    _free(name);
  }
}

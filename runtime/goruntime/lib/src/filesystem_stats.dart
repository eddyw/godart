// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

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
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isLinux ? '__errno_location' : '__error',
    );
final _getfsstat = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Int, Int),
      int Function(Pointer<Uint8>, int, int)
    >('getfsstat');

(int, Slice<int>, int) $syscallGetfsstat(int count, int flags) {
  // Darwin Statfs_t occupies 2168 bytes on AMD64 and ARM64.
  final size = count * 2168;
  final buffer = count == 0 ? nullptr.cast<Uint8>() : _malloc(size);
  if (count != 0 && buffer == nullptr) return (-1, Slice<int>.nil(), 12);
  try {
    final get = _getfsstat;
    final errorAddress = _errno();
    final n = get(buffer, size, flags);
    final code = n < 0 ? errorAddress.value : 0;
    if (code != 0 || count == 0) return (n, Slice<int>.nil(), code);
    return (n, _statfsData(buffer, n), 0);
  } finally {
    _free(buffer);
  }
}

Slice<int> _statfsData(Pointer<Uint8> buffer, int n) {
  final data = ByteData.sublistView(buffer.asTypedList(n * 2168));
  return Slice.of(<int>[
    for (var i = 0; i < n; i++) ...[
      data.getUint32(i * 2168, Endian.host),
      data.getInt32(i * 2168 + 4, Endian.host),
      for (var offset = 8; offset < 48; offset += 8)
        data.getUint64(i * 2168 + offset, Endian.host),
      data.getInt32(i * 2168 + 48, Endian.host),
      data.getInt32(i * 2168 + 52, Endian.host),
      for (var offset = 56; offset < 72; offset += 4)
        data.getUint32(i * 2168 + offset, Endian.host),
      for (var offset = 72; offset < 2136; offset++)
        data.getInt8(i * 2168 + offset),
      for (var offset = 2136; offset < 2168; offset += 4)
        data.getUint32(i * 2168 + offset, Endian.host),
    ],
  ]);
}

Slice<int> _linuxStatfsData(Pointer<Uint8> buffer) {
  // Linux Statfs_t occupies 120 bytes on AMD64 and ARM64.
  final data = ByteData.sublistView(buffer.asTypedList(120));
  return Slice.of(<int>[
    for (var offset = 0; offset < 56; offset += 8)
      data.getInt64(offset, Endian.host),
    data.getInt32(56, Endian.host),
    data.getInt32(60, Endian.host),
    for (var offset = 64; offset < 120; offset += 8)
      data.getInt64(offset, Endian.host),
  ]);
}

final _statfs = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>),
      int Function(Pointer<Uint8>, Pointer<Uint8>)
    >('statfs');
final _fstatfs = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>),
      int Function(int, Pointer<Uint8>)
    >('fstatfs');

(Slice<int>, int) $syscallStatfs(GoString path, bool output) {
  if (path.bytes.contains(0)) return (Slice<int>.nil(), 22);
  final name = _malloc(path.length + 1);
  if (name == nullptr) return (Slice<int>.nil(), 12);
  final buffer = output
      ? _malloc(Platform.isLinux ? 120 : 2168)
      : nullptr.cast<Uint8>();
  try {
    if (output && buffer == nullptr) return (Slice<int>.nil(), 12);
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final get = _statfs;
    final errorAddress = _errno();
    final code = get(name, buffer) == 0 ? 0 : errorAddress.value;
    if (code != 0) return (Slice<int>.nil(), code);
    return (
      Platform.isLinux ? _linuxStatfsData(buffer) : _statfsData(buffer, 1),
      0,
    );
  } finally {
    _free(buffer);
    _free(name);
  }
}

(Slice<int>, int) $syscallFstatfs(int fd, bool output) {
  final buffer = output
      ? _malloc(Platform.isLinux ? 120 : 2168)
      : nullptr.cast<Uint8>();
  if (output && buffer == nullptr) return (Slice<int>.nil(), 12);
  try {
    final get = _fstatfs;
    final errorAddress = _errno();
    final code = get(fd, buffer) == 0 ? 0 : errorAddress.value;
    if (code != 0) return (Slice<int>.nil(), code);
    return (
      Platform.isLinux ? _linuxStatfsData(buffer) : _statfsData(buffer, 1),
      0,
    );
  } finally {
    _free(buffer);
  }
}

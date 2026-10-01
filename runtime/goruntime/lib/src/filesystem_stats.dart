// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:typed_data';

import 'slice.dart';

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
    final data = ByteData.sublistView(buffer.asTypedList(n * 2168));
    return (
      n,
      Slice.of(<int>[
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
      ]),
      0,
    );
  } finally {
    _free(buffer);
  }
}

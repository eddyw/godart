// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';
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
      Platform.isMacOS ? '__error' : '__errno_location',
    );
final _getrusage = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>),
      int Function(int, Pointer<Uint8>)
    >('getrusage');

(Slice<int>, int) $syscallGetrusage(int who, bool output) {
  final buffer = output ? _malloc(144) : nullptr;
  if (output && buffer == nullptr) return (Slice<int>.nil(), 12);
  try {
    final get = _getrusage;
    final errorAddress = _errno();
    final code = get(who, buffer) == 0 ? 0 : errorAddress.value;
    if (code != 0 || !output) return (Slice<int>.nil(), code);
    final data = ByteData.sublistView(buffer.asTypedList(144));
    return (
      Slice.of([
        data.getInt64(0, Endian.host),
        Platform.isMacOS
            ? data.getInt32(8, Endian.host)
            : data.getInt64(8, Endian.host),
        data.getInt64(16, Endian.host),
        Platform.isMacOS
            ? data.getInt32(24, Endian.host)
            : data.getInt64(24, Endian.host),
        for (var offset = 32; offset < 144; offset += 8)
          data.getInt64(offset, Endian.host),
      ]),
      0,
    );
  } finally {
    _free(buffer);
  }
}

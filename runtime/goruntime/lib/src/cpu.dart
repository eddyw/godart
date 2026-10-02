// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

final _libc = DynamicLibrary.process();
final _malloc = _libc
    .lookupFunction<Pointer<Void> Function(Size), Pointer<Void> Function(int)>(
      'malloc',
    );
final _free = _libc
    .lookupFunction<Void Function(Pointer<Void>), void Function(Pointer<Void>)>(
      'free',
    );
final _getAffinity = _libc
    .lookupFunction<
      Int Function(Int, Size, Pointer<Uint8>),
      int Function(int, int, Pointer<Uint8>)
    >('sched_getaffinity');
final _sysctl = _libc
    .lookupFunction<
      Int Function(
        Pointer<Int>,
        UnsignedInt,
        Pointer<Void>,
        Pointer<Size>,
        Pointer<Void>,
        Size,
      ),
      int Function(
        Pointer<Int>,
        int,
        Pointer<Void>,
        Pointer<Size>,
        Pointer<Void>,
        int,
      )
    >('sysctl');

int $numCPU() {
  if (Platform.isLinux) {
    // Go uses an 8192-byte mask to include CPU IDs up to 65535.
    const maskSize = 8192;
    final mask = _malloc(maskSize).cast<Uint8>();
    if (mask == nullptr) return 1;
    try {
      final bytes = mask.asTypedList(maskSize)..fillRange(0, maskSize, 0);
      if (_getAffinity(0, maskSize, mask) != 0) return 1;
      var count = 0;
      for (var bits in bytes) {
        while (bits != 0) {
          count += bits & 1;
          bits >>= 1;
        }
      }
      return count > 0 ? count : 1;
    } finally {
      _free(mask.cast<Void>());
    }
  }
  if (!Platform.isMacOS) return Platform.numberOfProcessors;
  final storage = _malloc(16 + sizeOf<Size>()).cast<Uint8>();
  if (storage == nullptr) return 1;
  try {
    final mib = storage.cast<Int>();
    mib[0] = 6;
    mib[1] = 3;
    final count = (storage + 8).cast<Int>()..value = 0;
    final length = (storage + 16).cast<Size>()..value = sizeOf<Int>();
    final code = _sysctl(mib, 2, count.cast<Void>(), length, nullptr, 0);
    return code == 0 && count.value > 0 ? count.value : 1;
  } finally {
    _free(storage.cast<Void>());
  }
}

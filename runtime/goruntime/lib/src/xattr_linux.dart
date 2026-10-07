// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';

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
      '__errno_location',
    );
final _getxattr = _libc
    .lookupFunction<
      IntPtr Function(Pointer<Uint8>, Pointer<Uint8>, Pointer<Uint8>, Size),
      int Function(Pointer<Uint8>, Pointer<Uint8>, Pointer<Uint8>, int)
    >('getxattr');
final _listxattr = _libc
    .lookupFunction<
      IntPtr Function(Pointer<Uint8>, Pointer<Uint8>, Size),
      int Function(Pointer<Uint8>, Pointer<Uint8>, int)
    >('listxattr');
final _setxattr = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>, Pointer<Uint8>, Size, Int),
      int Function(Pointer<Uint8>, Pointer<Uint8>, Pointer<Uint8>, int, int)
    >('setxattr');
final _removexattr = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>),
      int Function(Pointer<Uint8>, Pointer<Uint8>)
    >('removexattr');

(int, int) $syscallGetxattr(GoString path, GoString attr, Slice<int> dest) {
  final get = _getxattr;
  return _xattr(path, attr, dest, get, true);
}

(int, int) $syscallListxattr(GoString path, Slice<int> dest) {
  final list = _listxattr;
  return _xattr(
    path,
    null,
    dest,
    (name, attr, data, size) => list(name, data, size),
    true,
  );
}

int $syscallSetxattr(GoString path, GoString attr, Slice<int> data, int flags) {
  final set = _setxattr;
  return _xattr(
    path,
    attr,
    data,
    (name, attr, data, size) => set(name, attr, data, size, flags),
    false,
  ).$2;
}

int $syscallRemovexattr(GoString path, GoString attr) {
  final remove = _removexattr;
  return _xattr(
    path,
    attr,
    Slice<int>.nil(),
    (name, attr, data, size) => remove(name, attr),
    false,
  ).$2;
}

(int, int) _xattr(
  GoString path,
  GoString? attr,
  Slice<int> bytes,
  int Function(Pointer<Uint8>, Pointer<Uint8>, Pointer<Uint8>, int) call,
  bool copyBack,
) {
  if (path.bytes.contains(0) || (attr?.bytes.contains(0) ?? false)) {
    return (0, 22);
  }
  final nameSize = path.length + 1;
  final attrSize = attr == null ? 0 : attr.length + 1;
  final memory = _malloc(nameSize + attrSize + bytes.len + 1);
  if (memory == nullptr) return (-1, 12);
  try {
    memory.asTypedList(nameSize)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final attribute = attr == null ? nullptr.cast<Uint8>() : memory + nameSize;
    if (attr != null) {
      attribute.asTypedList(attrSize)
        ..setRange(0, attr.length, attr.bytes)
        ..[attr.length] = 0;
    }
    final buffer = memory + nameSize + attrSize;
    buffer
        .asTypedList(bytes.len)
        .setRange(0, bytes.len, bytes.array, bytes.off);
    final errorAddress = _errno();
    final result = call(memory, attribute, buffer, bytes.len);
    final code = result < 0 ? errorAddress.value : 0;
    // Preserve all native buffer writes, including writes before an error.
    if (copyBack && bytes.len > 0) {
      bytes.array.setRange(
        bytes.off,
        bytes.off + bytes.len,
        buffer.asTypedList(bytes.len),
      );
    }
    return (result, code);
  } finally {
    _free(memory);
  }
}

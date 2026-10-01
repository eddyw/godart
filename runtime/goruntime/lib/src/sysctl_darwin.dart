// Copyright 2009,2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';

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
      '__error',
    );
final _sysctl = _libc
    .lookupFunction<
      Int Function(
        Pointer<Int32>,
        UnsignedInt,
        Pointer<Void>,
        Pointer<Size>,
        Pointer<Void>,
        Size,
      ),
      int Function(
        Pointer<Int32>,
        int,
        Pointer<Void>,
        Pointer<Size>,
        Pointer<Void>,
        int,
      )
    >('sysctl');

(List<int>, int) _sysctlName(GoString name) {
  if (name.bytes.contains(0)) return (const <int>[], 22);
  const maxName = 12;
  final query = _malloc(2 * sizeOf<Int32>()).cast<Int32>();
  // Go reserves 2 spare words because Darwin's native name conversion reserves them.
  final mib = _malloc((maxName + 2) * sizeOf<Int32>()).cast<Int32>();
  final length = _malloc(sizeOf<Size>()).cast<Size>();
  final text = _malloc(name.length + 1);
  try {
    if (query == nullptr ||
        mib == nullptr ||
        length == nullptr ||
        text == nullptr) {
      return (const <int>[], 12);
    }
    query[0] = 0;
    query[1] = 3;
    length.value = maxName * sizeOf<Int32>();
    text.asTypedList(name.length + 1)
      ..setRange(0, name.length, name.bytes)
      ..[name.length] = 0;
    final get = _sysctl;
    final errorAddress = _errno();
    final code =
        get(
              query,
              2,
              mib.cast<Void>(),
              length,
              text.cast<Void>(),
              name.length,
            ) ==
            0
        ? 0
        : errorAddress.value;
    if (code != 0) return (const <int>[], code);
    return (
      mib.asTypedList(length.value ~/ sizeOf<Int32>()).toList(growable: false),
      0,
    );
  } finally {
    _free(text);
    _free(length.cast<Uint8>());
    _free(mib.cast<Uint8>());
    _free(query.cast<Uint8>());
  }
}

(GoString, int) $syscallSysctl(GoString name) {
  final (values, nameCode) = _sysctlName(name);
  if (nameCode != 0) return (GoString.empty, nameCode);
  final mib = _malloc(values.length * sizeOf<Int32>()).cast<Int32>();
  final length = _malloc(sizeOf<Size>()).cast<Size>();
  var buffer = nullptr.cast<Uint8>();
  try {
    if (mib == nullptr || length == nullptr) return (GoString.empty, 12);
    mib.asTypedList(values.length).setAll(0, values);
    length.value = 0;
    final get = _sysctl;
    final errorAddress = _errno();
    var code = get(mib, values.length, nullptr, length, nullptr, 0) == 0
        ? 0
        : errorAddress.value;
    if (code != 0) return (GoString.empty, code);
    if (length.value == 0) return (GoString.empty, 0);
    buffer = _malloc(length.value);
    if (buffer == nullptr) return (GoString.empty, 12);
    code = get(mib, values.length, buffer.cast<Void>(), length, nullptr, 0) == 0
        ? 0
        : errorAddress.value;
    if (code != 0) return (GoString.empty, code);
    var n = length.value;
    if (n > 0 && buffer[n - 1] == 0) n--;
    return (GoString.fromBytes(buffer.asTypedList(n)), 0);
  } finally {
    _free(buffer);
    _free(length.cast<Uint8>());
    _free(mib.cast<Uint8>());
  }
}

(int, int) $syscallSysctlUint32(GoString name) {
  final (values, nameCode) = _sysctlName(name);
  if (nameCode != 0) return (0, nameCode);
  final mib = _malloc(values.length * sizeOf<Int32>()).cast<Int32>();
  final length = _malloc(sizeOf<Size>()).cast<Size>();
  final buffer = _malloc(4).cast<Uint32>();
  try {
    if (mib == nullptr || length == nullptr || buffer == nullptr) {
      return (0, 12);
    }
    mib.asTypedList(values.length).setAll(0, values);
    length.value = 4;
    final get = _sysctl;
    final errorAddress = _errno();
    final code =
        get(mib, values.length, buffer.cast<Void>(), length, nullptr, 0) == 0
        ? 0
        : errorAddress.value;
    if (code != 0) return (0, code);
    if (length.value != 4) return (0, 5);
    return (buffer.value, 0);
  } finally {
    _free(buffer.cast<Uint8>());
    _free(length.cast<Uint8>());
    _free(mib.cast<Uint8>());
  }
}

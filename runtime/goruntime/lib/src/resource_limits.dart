// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

final class _Rlimit extends Struct {
  @Uint64()
  external int cur;

  @Uint64()
  external int max;
}

final _libc = DynamicLibrary.process();
final _malloc = _libc
    .lookupFunction<Pointer<Void> Function(Size), Pointer<Void> Function(int)>(
      'malloc',
    );
final _free = _libc
    .lookupFunction<Void Function(Pointer<Void>), void Function(Pointer<Void>)>(
      'free',
    );
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );
final _getrlimit = _libc
    .lookupFunction<
      Int Function(Int, Pointer<_Rlimit>),
      int Function(int, Pointer<_Rlimit>)
    >('getrlimit');
final _setrlimit = _libc
    .lookupFunction<
      Int Function(Int, Pointer<_Rlimit>),
      int Function(int, Pointer<_Rlimit>)
    >('setrlimit');
final _prlimit = _libc
    .lookupFunction<
      Int Function(Int, Int, Pointer<_Rlimit>, Pointer<_Rlimit>),
      int Function(int, int, Pointer<_Rlimit>, Pointer<_Rlimit>)
    >('prlimit');

int _getLimit(int resource, Pointer<_Rlimit> limit) {
  if (Platform.isLinux) {
    final get = _prlimit;
    final errorAddress = _errno();
    return get(0, resource, nullptr, limit) == 0 ? 0 : errorAddress.value;
  }
  final get = _getrlimit;
  final errorAddress = _errno();
  return get(resource, limit) == 0 ? 0 : errorAddress.value;
}

int _setLimit(int resource, Pointer<_Rlimit> limit) {
  if (Platform.isLinux) {
    final set = _prlimit;
    final errorAddress = _errno();
    return set(0, resource, limit, nullptr) == 0 ? 0 : errorAddress.value;
  }
  final set = _setrlimit;
  final errorAddress = _errno();
  return set(resource, limit) == 0 ? 0 : errorAddress.value;
}

(int, int, int) $syscallGetrlimit(int resource, bool output) {
  final limit = output ? _malloc(sizeOf<_Rlimit>()).cast<_Rlimit>() : nullptr;
  if (output && limit == nullptr) return (0, 0, 12);
  try {
    final code = _getLimit(resource, limit);
    if (code != 0 || !output) return (0, 0, code);
    return (limit.ref.cur, limit.ref.max, 0);
  } finally {
    _free(limit.cast<Void>());
  }
}

int $syscallSetrlimit(int resource, bool input, int cur, int max) {
  final limit = input ? _malloc(sizeOf<_Rlimit>()).cast<_Rlimit>() : nullptr;
  if (input && limit == nullptr) return 12;
  try {
    if (input) {
      limit.ref
        ..cur = cur
        ..max = max;
    }
    return _setLimit(resource, limit);
  } finally {
    _free(limit.cast<Void>());
  }
}

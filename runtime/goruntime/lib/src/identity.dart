// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

import 'slice.dart';

final _libc = DynamicLibrary.process();
final _posix = Platform.isLinux || Platform.isMacOS;

final _getuid = _libc.lookupFunction<UnsignedInt Function(), int Function()>(
  'getuid',
);
int $getuid() => _posix ? _getuid() : -1;

final _geteuid = _libc.lookupFunction<UnsignedInt Function(), int Function()>(
  'geteuid',
);
int $geteuid() => _posix ? _geteuid() : -1;

final _getgid = _libc.lookupFunction<UnsignedInt Function(), int Function()>(
  'getgid',
);
int $getgid() => _posix ? _getgid() : -1;

final _getegid = _libc.lookupFunction<UnsignedInt Function(), int Function()>(
  'getegid',
);
int $getegid() => _posix ? _getegid() : -1;

final _getppid = _libc.lookupFunction<Int Function(), int Function()>(
  'getppid',
);
int $getppid() => _posix ? _getppid() : 0;

final _getpgrp = _libc.lookupFunction<Int Function(), int Function()>(
  'getpgrp',
);
int $getpgrp() => _posix ? _getpgrp() : -1;

final _getpgid = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'getpgid',
);
(int, int) $getpgid(int pid) {
  if (!_posix) return (-1, -1);
  final group = _getpgid(pid);
  return (group, group < 0 ? _errno().value : 0);
}

final _getsid = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'getsid',
);
(int, int) $getsid(int pid) {
  if (!_posix) return (-1, -1);
  final session = _getsid(pid);
  return (session, session < 0 ? _errno().value : 0);
}

final _setpgid = _libc
    .lookupFunction<Int Function(Int, Int), int Function(int, int)>('setpgid');

int $setpgid(int pid, int pgid) {
  if (!_posix) return -1;
  final set = _setpgid;
  final errno = _errno();
  return set(pid, pgid) == 0 ? 0 : errno.value;
}

final _setsid = _libc.lookupFunction<Int Function(), int Function()>('setsid');

(int, int) $setsid() {
  if (!_posix) return (-1, -1);
  final set = _setsid;
  final errno = _errno();
  final pid = set();
  return (pid, pid == -1 ? errno.value : 0);
}

final _linuxGetpriority = _libc
    .lookupFunction<
      Long Function(Long, VarArgs<(Int, UnsignedInt)>),
      int Function(int, int, int)
    >('syscall');
final _darwinGetpriority = _libc
    .lookupFunction<Int Function(Int, UnsignedInt), int Function(int, int)>(
      'getpriority',
    );

(int, int) $getpriority(int which, int who) {
  if (!_posix) return (-1, -1);
  if (Platform.isLinux) {
    final get = _linuxGetpriority;
    final errno = _errno();
    final value = get(Abi.current() == Abi.linuxArm64 ? 141 : 140, which, who);
    return (value, value == -1 ? errno.value : 0);
  }
  final get = _darwinGetpriority;
  final errno = _errno();
  // A native priority of -1 is valid when errno remains zero.
  errno.value = 0;
  final value = get(which, who);
  return (value, value == -1 ? errno.value : 0);
}

final _setpriority = _libc
    .lookupFunction<
      Int Function(Int, UnsignedInt, Int),
      int Function(int, int, int)
    >('setpriority');

int $setpriority(int which, int who, int priority) {
  if (!_posix) return -1;
  final set = _setpriority;
  final errno = _errno();
  return set(which, who, priority) == 0 ? 0 : errno.value;
}

final _getpagesize = _libc.lookupFunction<Int Function(), int Function()>(
  'getpagesize',
);
int $getpagesize() => _posix ? _getpagesize() : 4096;

final _getgroups = _libc
    .lookupFunction<
      Int Function(Int, Pointer<UnsignedInt>),
      int Function(int, Pointer<UnsignedInt>)
    >('getgroups');
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

(Slice<int>, int) $getgroups() {
  if (!_posix) return (Slice<int>.nil(), -1);
  final count = _getgroups(0, nullptr);
  if (count < 0) return (Slice<int>.nil(), _errno().value);
  if (count == 0) return (Slice<int>.nil(), 0);
  if (count > (Platform.isMacOS ? 1000 : 1 << 20)) {
    return (Slice<int>.nil(), 22);
  }
  final buffer = _malloc(count * sizeOf<UnsignedInt>()).cast<UnsignedInt>();
  if (buffer == nullptr) return (Slice<int>.nil(), 12);
  try {
    final actual = _getgroups(count, buffer);
    if (actual < 0) return (Slice<int>.nil(), _errno().value);
    return (
      Slice<int>.of(
        List<int>.generate(actual, (i) => buffer[i], growable: false),
      ),
      0,
    );
  } finally {
    _free(buffer.cast());
  }
}

final _kill = _libc
    .lookupFunction<Int Function(Int, Int), int Function(int, int)>('kill');

int $sendSignal(int pid, int signal) {
  if (!_posix) return -1;
  return _kill(pid, signal) == 0 ? 0 : _errno().value;
}

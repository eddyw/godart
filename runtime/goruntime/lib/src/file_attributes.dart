// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2017 The Go Authors. All rights reserved.
// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

import 'string.dart';

final _libc = DynamicLibrary.process();
final _fchmod = Platform.isMacOS
    ? _libc.lookupFunction<Int Function(Int, Uint16), int Function(int, int)>(
        'fchmod',
      )
    : _libc.lookupFunction<Int Function(Int, Uint32), int Function(int, int)>(
        'fchmod',
      );
final _fchown = _libc
    .lookupFunction<
      Int Function(Int, Uint32, Uint32),
      int Function(int, int, int)
    >('fchown');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

int $fileChmod(int fd, int mode) {
  int error;
  do {
    error = _fchmod(fd, mode) == 0 ? 0 : _errno().value;
  } while (error == 4);
  return error;
}

int $fileChown(int fd, int uid, int gid) {
  int error;
  do {
    error = _fchown(fd, uid, gid) == 0 ? 0 : _errno().value;
  } while (error == 4);
  return error;
}

final _truncate = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Int64),
      int Function(Pointer<Uint8>, int)
    >('truncate');

int $nativeTruncate(GoString path, int size) {
  final truncate = _truncate;
  return _pathAttribute(path, (name) => truncate(name, size));
}

final _ftruncate = _libc
    .lookupFunction<Int Function(Int, Int64), int Function(int, int)>(
      'ftruncate',
    );
final _fsync = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'fsync',
);
final _fdatasync = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'fdatasync',
);
final _fcntl = _libc
    .lookupFunction<
      Int Function(Int, Int, VarArgs<(Int,)>),
      int Function(int, int, int)
    >('fcntl');

int $nativeFtruncate(int fd, int size) {
  final truncate = _ftruncate;
  final errno = _errno;
  int code;
  do {
    code = truncate(fd, size) == 0 ? 0 : errno().value;
  } while (code == 4);
  return code;
}

int $nativeFsync(int fd) {
  final sync = _fsync;
  final errno = _errno;
  if (Platform.isMacOS) {
    // Darwin uses F_FULLFSYNC to flush the device cache.
    final fullSync = _fcntl;
    int code;
    do {
      code = fullSync(fd, 0x33, 0) == 0 ? 0 : errno().value;
      if (code == 45) code = sync(fd) == 0 ? 0 : errno().value;
    } while (code == 4);
    return code;
  }
  int code;
  do {
    code = sync(fd) == 0 ? 0 : errno().value;
  } while (code == 4);
  return code;
}

int $syscallFsync(int fd) {
  final sync = _fsync;
  final errorAddress = _errno();
  return sync(fd) == 0 ? 0 : errorAddress.value;
}

int $syscallFdatasync(int fd) {
  final sync = _fdatasync;
  final errorAddress = _errno();
  return sync(fd) == 0 ? 0 : errorAddress.value;
}

final _utimensat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Pointer<Int64>, Int),
      int Function(int, Pointer<Uint8>, Pointer<Int64>, int)
    >('utimensat');
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

final _chmod = Platform.isMacOS
    ? _libc.lookupFunction<
        Int Function(Pointer<Uint8>, Uint16),
        int Function(Pointer<Uint8>, int)
      >('chmod')
    : _libc.lookupFunction<
        Int Function(Pointer<Uint8>, Uint32),
        int Function(Pointer<Uint8>, int)
      >('chmod');
final _fchmodat = Platform.isMacOS
    ? _libc.lookupFunction<
        Int Function(Int, Pointer<Uint8>, Uint16, Int),
        int Function(int, Pointer<Uint8>, int, int)
      >('fchmodat')
    : _libc.lookupFunction<
        Int Function(Int, Pointer<Uint8>, Uint32, Int),
        int Function(int, Pointer<Uint8>, int, int)
      >('fchmodat');
final _fchownat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Uint32, Uint32, Int),
      int Function(int, Pointer<Uint8>, int, int, int)
    >('fchownat');

final _fchmodat2 = _libc
    .lookupFunction<
      Long Function(Long, VarArgs<(Int, Pointer<Uint8>, Uint32, Int)>),
      int Function(int, int, Pointer<Uint8>, int, int)
    >('syscall');

int $syscallFchmodat(int fd, GoString path, int mode, int flags) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return 22;
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    if (flags == 0) {
      final chmod = _fchmodat;
      final errorAddress = _errno();
      return chmod(fd, name, mode, 0) == 0 ? 0 : errorAddress.value;
    }
    final chmod = _fchmodat2;
    final errorAddress = _errno();
    // Linux amd64 and arm64 assign syscall 452 to fchmodat2.
    return chmod(452, fd, name, mode, flags) == 0 ? 0 : errorAddress.value;
  } finally {
    _free(name);
  }
}

final _atNoFollow = Platform.isMacOS ? 0x20 : 0x100;

int $rootChmod(int parent, GoString path, int mode) =>
    _pathAttribute(path, (name) => _fchmodat(parent, name, mode, _atNoFollow));

int $rootChown(int parent, GoString path, int uid, int gid) => _pathAttribute(
  path,
  (name) => _fchownat(parent, name, uid, gid, _atNoFollow),
);

final _chown = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Uint32, Uint32),
      int Function(Pointer<Uint8>, int, int)
    >('chown');
final _lchown = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Uint32, Uint32),
      int Function(Pointer<Uint8>, int, int)
    >('lchown');

int _pathAttribute(GoString path, int Function(Pointer<Uint8>) apply) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return 22;
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final errno = _errno();
    int error;
    do {
      error = apply(name) == 0 ? 0 : errno.value;
    } while (error == 4);
    return error;
  } finally {
    _free(name);
  }
}

int $nativeChmod(GoString path, int mode) =>
    _pathAttribute(path, (name) => _chmod(name, mode));

final _faccessat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Int, Int),
      int Function(int, Pointer<Uint8>, int, int)
    >('faccessat');

int $nativeExecutableAccess(GoString path) {
  final access = _faccessat;
  final directory = Platform.isMacOS ? -2 : -100;
  final effective = Platform.isMacOS ? 0x10 : 0x200;
  return _pathAttribute(path, (name) => access(directory, name, 1, effective));
}

final _access = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Uint32),
      int Function(Pointer<Uint8>, int)
    >('access');

int $syscallAccess(GoString path, int mode) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return 22;
  final access = _access;
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final errorAddress = _errno();
    return access(name, mode) == 0 ? 0 : errorAddress.value;
  } finally {
    _free(name);
  }
}

int $nativeChown(GoString path, int uid, int gid, bool follow) =>
    _pathAttribute(
      path,
      (name) => follow ? _chown(name, uid, gid) : _lchown(name, uid, gid),
    );

int $nativeChtimes(
  GoString path,
  int atime,
  bool setAtime,
  int mtime,
  bool setMtime,
) => _chtimes(
  Platform.isMacOS ? -2 : -100,
  path,
  0,
  atime,
  setAtime,
  mtime,
  setMtime,
);

int $rootChtimes(
  int parent,
  GoString path,
  int atime,
  bool setAtime,
  int mtime,
  bool setMtime,
) => _chtimes(parent, path, _atNoFollow, atime, setAtime, mtime, setMtime);

int _chtimes(
  int parent,
  GoString path,
  int flags,
  int atime,
  bool setAtime,
  int mtime,
  bool setMtime,
) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return 22;
  final buffer = _malloc(32 + bytes.length + 1);
  if (buffer == nullptr) return 12;
  try {
    final times = buffer.cast<Int64>();
    final omitted = Platform.isMacOS ? -2 : 1073741822;
    void store(int index, int value, bool set) {
      if (!set) {
        times[index] = omitted;
        times[index + 1] = omitted;
        return;
      }
      final nsec = value % 1000000000;
      var sec = value ~/ 1000000000;
      if (value < 0 && nsec != 0) sec--;
      times[index] = sec;
      times[index + 1] = nsec;
    }

    store(0, atime, setAtime);
    store(2, mtime, setMtime);
    final name = buffer + 32;
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final errno = _errno();
    int error;
    do {
      error = _utimensat(parent, name, times, flags) == 0 ? 0 : errno.value;
    } while (error == 4);
    return error;
  } finally {
    _free(buffer);
  }
}

final _umask = Platform.isMacOS
    ? _libc.lookupFunction<Uint16 Function(Uint16), int Function(int)>('umask')
    : _libc.lookupFunction<Uint32 Function(Uint32), int Function(int)>('umask');

int $syscallUmask(int mask) => _umask(mask);

final _syscallUtimes = _libc
    .lookupFunction<
      Long Function(Long, VarArgs<(Pointer<Uint8>, Pointer<Int64>)>),
      int Function(int, Pointer<Uint8>, Pointer<Int64>)
    >('syscall');
final _syscallFutimesat = _libc
    .lookupFunction<
      Long Function(Long, VarArgs<(Int, Pointer<Uint8>, Pointer<Int64>)>),
      int Function(int, int, Pointer<Uint8>, Pointer<Int64>)
    >('syscall');

int $syscallUtimensat(
  int fd,
  GoString path,
  int atimeSec,
  int atimeNsec,
  int mtimeSec,
  int mtimeNsec,
) {
  final utimens = _utimensat;
  return _withSyscallTimes(
    path,
    atimeSec,
    atimeNsec,
    mtimeSec,
    mtimeNsec,
    (name, times) => utimens(fd, name, times, 0),
  );
}

int $syscallUtimes(
  GoString path,
  int atimeSec,
  int atimeUsec,
  int mtimeSec,
  int mtimeUsec,
) {
  final utimes = _syscallUtimes;
  return _withSyscallTimes(
    path,
    atimeSec,
    atimeUsec,
    mtimeSec,
    mtimeUsec,
    (name, times) => utimes(235, name, times),
  );
}

int $syscallFutimesat(
  int fd,
  GoString path,
  int atimeSec,
  int atimeUsec,
  int mtimeSec,
  int mtimeUsec,
) {
  final futimes = _syscallFutimesat;
  return _withSyscallTimes(
    path,
    atimeSec,
    atimeUsec,
    mtimeSec,
    mtimeUsec,
    (name, times) => futimes(261, fd, name, times),
  );
}

int _withSyscallTimes(
  GoString path,
  int atimeSec,
  int atimeFraction,
  int mtimeSec,
  int mtimeFraction,
  int Function(Pointer<Uint8>, Pointer<Int64>) apply,
) => _withSyscallPath(path, (name) {
  final memory = _malloc(32);
  if (memory == nullptr) return 12;
  try {
    final times = memory.cast<Int64>();
    times[0] = atimeSec;
    times[1] = atimeFraction;
    times[2] = mtimeSec;
    times[3] = mtimeFraction;
    final errorAddress = _errno();
    return apply(name, times) == 0 ? 0 : errorAddress.value;
  } finally {
    _free(memory);
  }
});

final _faccessatSyscall = _libc
    .lookupFunction<
      Long Function(Long, VarArgs<(Int, Pointer<Uint8>, Uint32, Int)>),
      int Function(int, int, Pointer<Uint8>, int, int)
    >('syscall');

int $syscallFaccessat(int fd, GoString path, int mode, int flags) {
  final access = _faccessatSyscall;
  final number = flags == 0
      ? (Abi.current() == Abi.linuxArm64 ? 48 : 269)
      : 439;
  final bytes = path.bytes;
  if (bytes.contains(0)) return 22;
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final errorAddress = _errno();
    return access(number, fd, name, mode, flags) == 0 ? 0 : errorAddress.value;
  } finally {
    _free(name);
  }
}

final _capgetSyscall = _libc
    .lookupFunction<
      Long Function(Long, VarArgs<(Pointer<Uint32>, Pointer<Uint32>)>),
      int Function(int, Pointer<Uint32>, Pointer<Uint32>)
    >('syscall');

bool $syscallCapDacOverride() {
  final memory = _malloc(32);
  if (memory == nullptr) return false;
  try {
    memory.asTypedList(32).fillRange(0, 32, 0);
    final header = memory.cast<Uint32>();
    header.value = 0x20080522;
    final data = (memory + 8).cast<Uint32>();
    final capget = _capgetSyscall;
    final number = Abi.current() == Abi.linuxArm64 ? 90 : 125;
    return capget(number, header, data) == 0 && data.value & 2 != 0;
  } finally {
    _free(memory);
  }
}

int _withSyscallPath(GoString path, int Function(Pointer<Uint8>) apply) {
  final bytes = path.bytes;
  if (bytes.contains(0)) return 22;
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    return apply(name);
  } finally {
    _free(name);
  }
}

final _mkdirat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Uint32),
      int Function(int, Pointer<Uint8>, int)
    >('mkdirat');
final _unlinkat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Int),
      int Function(int, Pointer<Uint8>, int)
    >('unlinkat');
final _renameat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Int, Pointer<Uint8>),
      int Function(int, Pointer<Uint8>, int, Pointer<Uint8>)
    >('renameat');

int $syscallMkdirat(int fd, GoString path, int mode) {
  final mkdir = _mkdirat;
  final errorAddress = _errno();
  return _withSyscallPath(
    path,
    (name) => mkdir(fd, name, mode) == 0 ? 0 : errorAddress.value,
  );
}

int $syscallFchownat(int fd, GoString path, int uid, int gid, int flags) {
  final chown = _fchownat;
  final errorAddress = _errno();
  return _withSyscallPath(
    path,
    (name) => chown(fd, name, uid, gid, flags) == 0 ? 0 : errorAddress.value,
  );
}

int $syscallUnlinkat(int fd, GoString path) {
  final unlink = _unlinkat;
  final errorAddress = _errno();
  return _withSyscallPath(
    path,
    (name) => unlink(fd, name, 0) == 0 ? 0 : errorAddress.value,
  );
}

int $syscallRenameat(int oldFD, GoString oldPath, int newFD, GoString newPath) {
  final rename = _renameat;
  final errorAddress = _errno();
  return _withSyscallPath(
    oldPath,
    (oldName) => _withSyscallPath(
      newPath,
      (newName) =>
          rename(oldFD, oldName, newFD, newName) == 0 ? 0 : errorAddress.value,
    ),
  );
}

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

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
final _mknodat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, UnsignedInt, Uint64),
      int Function(int, Pointer<Uint8>, int, int)
    >('mknodat');
final _nanosleep = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>),
      int Function(Pointer<Uint8>, Pointer<Uint8>)
    >('nanosleep');
final _exchangedata = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>, UnsignedInt),
      int Function(Pointer<Uint8>, Pointer<Uint8>, int)
    >('exchangedata');
final _revoke = _libc
    .lookupFunction<Int Function(Pointer<Uint8>), int Function(Pointer<Uint8>)>(
      'revoke',
    );

int $syscallRevoke(GoString path) {
  final text = _malloc(path.length + 1);
  if (text == nullptr) return 12;
  try {
    text.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final control = _revoke;
    final errorAddress = _errno();
    return control(text) < 0 ? errorAddress.value : 0;
  } finally {
    _free(text);
  }
}

final _undelete = _libc
    .lookupFunction<Int Function(Pointer<Uint8>), int Function(Pointer<Uint8>)>(
      'undelete',
    );

int $syscallUndelete(GoString path) {
  final text = _malloc(path.length + 1);
  if (text == nullptr) return 12;
  try {
    text.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final control = _undelete;
    final errorAddress = _errno();
    return control(text) < 0 ? errorAddress.value : 0;
  } finally {
    _free(text);
  }
}

final _kevent = _libc
    .lookupFunction<
      Int Function(
        Int,
        Pointer<Uint8>,
        Int,
        Pointer<Uint8>,
        Int,
        Pointer<Uint8>,
      ),
      int Function(
        int,
        Pointer<Uint8>,
        int,
        Pointer<Uint8>,
        int,
        Pointer<Uint8>,
      )
    >('kevent');

(int, int) $syscallKevent(
  int kq,
  Slice<int> changes,
  Slice<int> events,
  Slice<int> timeout,
) {
  if (!Platform.isMacOS) return (-1, 78);
  final buffers = Map<List<int>, Pointer<Uint8>>.identity();
  try {
    for (final data in [changes, events, timeout]) {
      if (data.len == 0 || buffers.containsKey(data.array)) continue;
      final buffer = _malloc(data.array.length);
      if (buffer == nullptr) return (-1, 12);
      buffers[data.array] = buffer;
      buffer.asTypedList(data.array.length).setAll(0, data.array);
    }
    final change = changes.len == 0
        ? nullptr.cast<Uint8>()
        : buffers[changes.array]! + changes.off;
    final event = events.len == 0
        ? nullptr.cast<Uint8>()
        : buffers[events.array]! + events.off;
    final limit = timeout.len == 0
        ? nullptr.cast<Uint8>()
        : buffers[timeout.array]! + timeout.off;
    final call = _kevent;
    final errorAddress = _errno();
    final n = call(
      kq,
      change,
      changes.len ~/ 32,
      event,
      events.len ~/ 32,
      limit,
    );
    final code = n < 0 ? errorAddress.value : 0;
    if (events.len != 0) {
      events.array.setRange(
        events.off,
        events.off + events.len,
        event.asTypedList(events.len),
      );
    }
    return (n, code);
  } finally {
    for (final buffer in buffers.values) {
      _free(buffer);
    }
  }
}

final _bpfIoctl = _libc
    .lookupFunction<
      Int Function(Int, UnsignedLong, VarArgs<(Pointer<Uint8>,)>),
      int Function(int, int, Pointer<Uint8>)
    >('ioctl');

int $syscallBpfIoctl(
  int fd,
  int request,
  Slice<int> data,
  Slice<int> instructions,
) {
  if (!Platform.isMacOS) return 78;
  switch (request) {
    case 0x40044266:
    case 0xc0044266:
    case 0x4004426a:
    case 0x80044278:
    case 0x20004269:
    case 0x20004268:
    case 0x4020426b:
    case 0x8020426c:
    case 0x4010426e:
    case 0x8010426d:
    case 0x4008426f:
    case 0x80044270:
    case 0x80104267:
    case 0x40044271:
    case 0x40044274:
    case 0x80044275:
      break;
    default:
      return 78;
  }
  final size = data.len + instructions.len;
  final buffer = size == 0 ? nullptr.cast<Uint8>() : _malloc(size);
  if (size != 0 && buffer == nullptr) return 12;
  try {
    if (data.len != 0) {
      buffer.asTypedList(data.len).setRange(0, data.len, data.array, data.off);
    }
    if (request == 0x80104267) {
      final filters = buffer + data.len;
      filters
          .asTypedList(instructions.len)
          .setRange(0, instructions.len, instructions.array, instructions.off);
      (buffer + 8).cast<Pointer<Uint8>>().value = filters;
    }
    final call = _bpfIoctl;
    final errorAddress = _errno();
    final code = call(fd, request, buffer) < 0 ? errorAddress.value : 0;
    // A native filter pointer must never enter a Go byte object.
    if (data.len != 0 && request != 0x80104267) {
      data.array.setRange(
        data.off,
        data.off + data.len,
        buffer.asTypedList(data.len),
      );
    }
    return code;
  } finally {
    if (buffer != nullptr) _free(buffer);
  }
}

final _kqueue = _libc.lookupFunction<Int Function(), int Function()>('kqueue');

(int, int) $syscallKqueue() {
  final create = _kqueue;
  final errorAddress = _errno();
  final fd = create();
  return (fd, fd < 0 ? errorAddress.value : 0);
}

final _getdtablesize = _libc.lookupFunction<Int Function(), int Function()>(
  'getdtablesize',
);
final _issetugid = _libc.lookupFunction<Int Function(), int Function()>(
  'issetugid',
);

int $syscallGetdtablesize() => _getdtablesize();
bool $syscallIssetugid() => _issetugid() != 0;

final _sync = _libc.lookupFunction<Void Function(), void Function()>('sync');

void $syscallSync() => _sync();

final _open = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Int, VarArgs<(UnsignedInt,)>),
      int Function(Pointer<Uint8>, int, int)
    >('open');
final _socket = _libc
    .lookupFunction<Int Function(Int, Int, Int), int Function(int, int, int)>(
      'socket',
    );
final _socketpair = _libc
    .lookupFunction<
      Int Function(Int, Int, Int, Pointer<Int32>),
      int Function(int, int, int, Pointer<Int32>)
    >('socketpair');
final _accept = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Pointer<Uint32>),
      int Function(int, Pointer<Uint8>, Pointer<Uint32>)
    >('accept');
final _accept4 = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Pointer<Uint32>, Int),
      int Function(int, Pointer<Uint8>, Pointer<Uint32>, int)
    >('accept4');
final _recvfrom = _libc
    .lookupFunction<
      IntPtr Function(
        Int,
        Pointer<Uint8>,
        Size,
        Int,
        Pointer<Uint8>,
        Pointer<Uint32>,
      ),
      int Function(
        int,
        Pointer<Uint8>,
        int,
        int,
        Pointer<Uint8>,
        Pointer<Uint32>,
      )
    >('recvfrom');
final _getsockname = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Pointer<Uint32>),
      int Function(int, Pointer<Uint8>, Pointer<Uint32>)
    >('getsockname');
final _getpeername = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Pointer<Uint32>),
      int Function(int, Pointer<Uint8>, Pointer<Uint32>)
    >('getpeername');
final _socketBind = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Uint32),
      int Function(int, Pointer<Uint8>, int)
    >('bind');
final _socketConnect = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Uint32),
      int Function(int, Pointer<Uint8>, int)
    >('connect');
final _socketSendto = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Uint8>, Size, Int, Pointer<Uint8>, Uint32),
      int Function(int, Pointer<Uint8>, int, int, Pointer<Uint8>, int)
    >('sendto');
final _select = _libc
    .lookupFunction<
      Int Function(
        Int,
        Pointer<Uint8>,
        Pointer<Uint8>,
        Pointer<Uint8>,
        Pointer<Uint8>,
      ),
      int Function(
        int,
        Pointer<Uint8>,
        Pointer<Uint8>,
        Pointer<Uint8>,
        Pointer<Uint8>,
      )
    >('select');
final _sysinfo = _libc
    .lookupFunction<Int Function(Pointer<Uint8>), int Function(Pointer<Uint8>)>(
      'sysinfo',
    );
final _gettid = _libc.lookupFunction<Int Function(), int Function()>('gettid');
final _times = _libc
    .lookupFunction<
      Int64 Function(Pointer<Uint8>),
      int Function(Pointer<Uint8>)
    >('times');
final _inotifyInit = _libc.lookupFunction<Int Function(), int Function()>(
  'inotify_init',
);
final _inotifyInit1 = _libc
    .lookupFunction<Int Function(Int), int Function(int)>('inotify_init1');
final _inotifyAddWatch = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Uint32),
      int Function(int, Pointer<Uint8>, int)
    >('inotify_add_watch');
final _inotifyRmWatch = _libc
    .lookupFunction<Int Function(Int, Uint32), int Function(int, int)>(
      'inotify_rm_watch',
    );
final _epollCreate1 = _libc
    .lookupFunction<Int Function(Int), int Function(int)>('epoll_create1');
final _epollCtl = _libc
    .lookupFunction<
      Int Function(Int, Int, Int, Pointer<Uint8>),
      int Function(int, int, int, Pointer<Uint8>)
    >('epoll_ctl');
final _epollWait = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Int, Int),
      int Function(int, Pointer<Uint8>, int, int)
    >('epoll_wait');
final _rawPipe = _libc
    .lookupFunction<Int Function(Pointer<Int32>), int Function(Pointer<Int32>)>(
      'pipe',
    );
final _pipe2 = _libc
    .lookupFunction<
      Int Function(Pointer<Int32>, Int),
      int Function(Pointer<Int32>, int)
    >('pipe2');
final _splice = _libc
    .lookupFunction<
      IntPtr Function(
        Int,
        Pointer<Int64>,
        Int,
        Pointer<Int64>,
        Size,
        UnsignedInt,
      ),
      int Function(int, Pointer<Int64>, int, Pointer<Int64>, int, int)
    >('splice');
final _tee = _libc
    .lookupFunction<
      IntPtr Function(Int, Int, Size, UnsignedInt),
      int Function(int, int, int, int)
    >('tee');
final _sendfile = _libc
    .lookupFunction<
      IntPtr Function(Int, Int, Pointer<Int64>, Size),
      int Function(int, int, Pointer<Int64>, int)
    >('sendfile');
final _syncFileRange = _libc
    .lookupFunction<
      Int Function(Int, Int64, Int64, UnsignedInt),
      int Function(int, int, int, int)
    >('sync_file_range');
final _fallocate = _libc
    .lookupFunction<
      Int Function(Int, Int, Int64, Int64),
      int Function(int, int, int, int)
    >('fallocate');
final _flock = _libc
    .lookupFunction<Int Function(Int, Int), int Function(int, int)>('flock');
final _listen = _libc
    .lookupFunction<Int Function(Int, Int), int Function(int, int)>('listen');
final _shutdown = _libc
    .lookupFunction<Int Function(Int, Int), int Function(int, int)>('shutdown');
final _getsockopt = _libc
    .lookupFunction<
      Int Function(Int, Int, Int, Pointer<Uint8>, Pointer<Uint32>),
      int Function(int, int, int, Pointer<Uint8>, Pointer<Uint32>)
    >('getsockopt');
final _setsockopt = _libc
    .lookupFunction<
      Int Function(Int, Int, Int, Pointer<Uint8>, Uint32),
      int Function(int, int, int, Pointer<Uint8>, int)
    >('setsockopt');
final _close = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'close',
);
final _dup = _libc.lookupFunction<Int Function(Int), int Function(int)>('dup');
final _dup2 = _libc
    .lookupFunction<Int Function(Int, Int), int Function(int, int)>('dup2');
final _dup3 = _libc
    .lookupFunction<Int Function(Int, Int, Int), int Function(int, int, int)>(
      'dup3',
    );
final _fcntl = _libc
    .lookupFunction<
      Int Function(Int, Int, VarArgs<(Int,)>),
      int Function(int, int, int)
    >('fcntl');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

(int, int) $syscallOpen(GoString path, int flags, int mode) {
  if (path.bytes.contains(0)) return (0, 22);
  final name = _malloc(path.length + 1);
  if (name == nullptr) return (-1, 12);
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    // Resolve native bindings before open can set errno.
    final open = _open;
    final errorAddress = _errno();
    final fd = open(name, flags, mode);
    return (fd, fd < 0 ? errorAddress.value : 0);
  } finally {
    _free(name);
  }
}

int $syscallMknodat(int parent, GoString path, int mode, int dev) {
  if (path.bytes.contains(0)) return 22;
  final name = _malloc(path.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final mknodat = _mknodat;
    final errorAddress = _errno();
    // The Linux syscall consumes the low 32 bits of the device argument.
    final result = mknodat(parent, name, mode, dev & 0xffffffff);
    return result < 0 ? errorAddress.value : 0;
  } finally {
    _free(name);
  }
}

(int, int) $syscallOpenat(int parent, GoString path, int flags, int mode) {
  if (path.bytes.contains(0)) return (0, 22);
  final name = _malloc(path.length + 1);
  if (name == nullptr) return (-1, 12);
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final openat = _openat;
    final errorAddress = _errno();
    final fd = openat(parent, name, flags, mode);
    return (fd, fd < 0 ? errorAddress.value : 0);
  } finally {
    _free(name);
  }
}

(int, int) $syscallSocket(int domain, int type, int protocol) {
  final socket = _socket;
  final errorAddress = _errno();
  final fd = socket(domain, type, protocol);
  return (fd, fd < 0 ? errorAddress.value : 0);
}

int $syscallClose(int fd) => _close(fd) == 0 ? 0 : _errno().value;

(int, int) $syscallDup(int oldfd) {
  final dup = _dup;
  final errorAddress = _errno();
  final fd = dup(oldfd);
  return (fd, fd < 0 ? errorAddress.value : 0);
}

int $syscallDup2(int oldfd, int newfd) {
  final dup2 = _dup2;
  final errorAddress = _errno();
  return dup2(oldfd, newfd) < 0 ? errorAddress.value : 0;
}

int $syscallDup3(int oldfd, int newfd, int flags) {
  final dup3 = _dup3;
  final errorAddress = _errno();
  return dup3(oldfd, newfd, flags) < 0 ? errorAddress.value : 0;
}

(int, int) $syscallFcntl(int fd, int cmd, int arg) {
  final fcntl = _fcntl;
  final errorAddress = _errno();
  final value = fcntl(fd, cmd, arg);
  return (value, value < 0 ? errorAddress.value : 0);
}

final _getdents = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Uint8>, Size),
      int Function(int, Pointer<Uint8>, int)
    >('getdents64');

(int, int) $syscallReadDirent(int fd, Slice<int> data) {
  if (Platform.isMacOS) return _readDarwin(fd, data);
  final buffer = data.len == 0 ? nullptr.cast<Uint8>() : _malloc(data.len);
  if (data.len != 0 && buffer == nullptr) return (-1, 12);
  try {
    // getdents64 leaves record padding bytes unchanged.
    if (data.len != 0) {
      buffer.asTypedList(data.len).setRange(0, data.len, data.array, data.off);
    }
    final n = _getdents(fd, buffer, data.len);
    if (n < 0) return (n, _errno().value);
    if (n != 0) {
      data.array.setRange(data.off, data.off + n, buffer.asTypedList(n));
    }
    return (n, 0);
  } finally {
    _free(buffer);
  }
}

final _seek = _libc
    .lookupFunction<
      Int64 Function(Int, Int64, Int),
      int Function(int, int, int)
    >('lseek');
final _openat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Int, VarArgs<(UnsignedInt,)>),
      int Function(int, Pointer<Uint8>, int, int)
    >('openat');
final _fdopendir = _libc
    .lookupFunction<Pointer<Void> Function(Int), Pointer<Void> Function(int)>(
      Abi.current() == Abi.macosX64 ? r'fdopendir$INODE64' : 'fdopendir',
    );
final _closedir = _libc
    .lookupFunction<Int Function(Pointer<Void>), int Function(Pointer<Void>)>(
      'closedir',
    );
final _readdir = _libc
    .lookupFunction<
      Pointer<Uint8> Function(Pointer<Void>),
      Pointer<Uint8> Function(Pointer<Void>)
    >(Abi.current() == Abi.macosX64 ? r'readdir$INODE64' : 'readdir');

// Darwin stores an entry count in the descriptor offset, as Go does.
// openat creates independent directory state. dup would share the offset.
(int, int) _readDarwin(int fd, Slice<int> data) {
  var skip = _seek(fd, 0, 1);
  if (skip < 0) return (0, _errno().value);
  final dot = _malloc(2);
  if (dot == nullptr) return (0, 12);
  final int copy;
  try {
    dot[0] = 46;
    dot[1] = 0;
    copy = _openat(fd, dot, 0, 0);
    if (copy < 0) return (0, _errno().value);
  } finally {
    _free(dot);
  }
  final directory = _fdopendir(copy);
  if (directory == nullptr) {
    final error = _errno().value;
    _close(copy);
    return (0, error);
  }
  try {
    var count = 0;
    var n = 0;
    while (true) {
      _errno().value = 0;
      final entry = _readdir(directory);
      if (entry == nullptr) {
        final error = _errno().value;
        if (error != 0) return (n, error);
        break;
      }
      if (skip > 0) {
        skip--;
        count++;
        continue;
      }
      final length = (entry + 16).cast<Uint16>().value;
      if (length > data.len - n) break;
      data.array.setRange(
        data.off + n,
        data.off + n + length,
        entry.asTypedList(length),
      );
      n += length;
      count++;
    }
    final position = _seek(fd, count, 0);
    return (n, position < 0 ? _errno().value : 0);
  } finally {
    _closedir(directory);
  }
}

(int, int) $syscallSeek(int fd, int offset, int whence) {
  final seek = _seek;
  final errno = _errno;
  // Darwin device files can return a negative offset without an error.
  final position = seek(fd, offset, whence);
  return (position, position == -1 ? errno().value : 0);
}

int $syscallAttachLsf(int fd, Slice<int> data) {
  final storage = _malloc(16 + data.len);
  if (storage == nullptr) return 12;
  try {
    storage.asTypedList(16).fillRange(0, 16, 0);
    final filters = storage + 16;
    filters.asTypedList(data.len).setRange(0, data.len, data.array, data.off);
    storage.cast<Uint16>().value = (data.len ~/ 8) & 0xffff;
    (storage + 8).cast<Pointer<Uint8>>().value = filters;
    final setsockopt = _setsockopt;
    final errorAddress = _errno();
    return setsockopt(fd, 1, 26, storage, 16) < 0 ? errorAddress.value : 0;
  } finally {
    _free(storage);
  }
}

int $syscallSetsockoptString(int fd, int level, int option, GoString value) {
  final buffer = value.length == 0
      ? nullptr.cast<Uint8>()
      : _malloc(value.length);
  if (value.length != 0 && buffer == nullptr) return 12;
  try {
    if (value.length != 0) {
      buffer.asTypedList(value.length).setRange(0, value.length, value.bytes);
    }
    final setsockopt = _setsockopt;
    final errorAddress = _errno();
    return setsockopt(fd, level, option, buffer, value.length) < 0
        ? errorAddress.value
        : 0;
  } finally {
    _free(buffer);
  }
}

(int, int) $syscallGetsockoptInt(int fd, int level, int option) {
  final buffer = _malloc(8);
  if (buffer == nullptr) return (0, 12);
  try {
    final value = buffer.cast<Int32>();
    final length = (buffer + 4).cast<Uint32>();
    value.value = 0;
    length.value = 4;
    final getsockopt = _getsockopt;
    final errorAddress = _errno();
    final result = getsockopt(fd, level, option, buffer, length);
    final code = result < 0 ? errorAddress.value : 0;
    return (value.value, code);
  } finally {
    _free(buffer);
  }
}

int $syscallSetsockoptInt(int fd, int level, int option, int value) {
  final buffer = _malloc(4);
  if (buffer == nullptr) return 12;
  try {
    buffer.cast<Int32>().value = value;
    final setsockopt = _setsockopt;
    final errorAddress = _errno();
    return setsockopt(fd, level, option, buffer, 4) < 0
        ? errorAddress.value
        : 0;
  } finally {
    _free(buffer);
  }
}

(int, int, int) $syscallSocketpair(int domain, int type, int protocol) {
  final buffer = _malloc(8);
  if (buffer == nullptr) return (0, 0, 12);
  try {
    final descriptors = buffer.cast<Int32>();
    descriptors[0] = 0;
    descriptors[1] = 0;
    final socketpair = _socketpair;
    final errorAddress = _errno();
    final result = socketpair(domain, type, protocol, descriptors);
    final code = result < 0 ? errorAddress.value : 0;
    return (descriptors[0], descriptors[1], code);
  } finally {
    _free(buffer);
  }
}

(int, int) $syscallGetsockoptByte(int fd, int level, int option) {
  final buffer = _malloc(8);
  if (buffer == nullptr) return (0, 12);
  try {
    buffer[0] = 0;
    final length = (buffer + 4).cast<Uint32>();
    length.value = 1;
    final getsockopt = _getsockopt;
    final errorAddress = _errno();
    final result = getsockopt(fd, level, option, buffer, length);
    final code = result < 0 ? errorAddress.value : 0;
    return (buffer[0], code);
  } finally {
    _free(buffer);
  }
}

(int, int, int, int) $syscallGetsockoptUcred(int fd, int level, int option) {
  final buffer = _malloc(16);
  if (buffer == nullptr) return (0, 0, 0, 12);
  try {
    buffer.asTypedList(12).fillRange(0, 12, 0);
    final length = (buffer + 12).cast<Uint32>();
    length.value = 12;
    final getsockopt = _getsockopt;
    final errorAddress = _errno();
    final result = getsockopt(fd, level, option, buffer, length);
    final code = result < 0 ? errorAddress.value : 0;
    return (
      buffer.cast<Int32>().value,
      (buffer + 4).cast<Uint32>().value,
      (buffer + 8).cast<Uint32>().value,
      code,
    );
  } finally {
    _free(buffer);
  }
}

int $syscallGetsockoptIPMreq(int fd, int level, int option, Slice<int> data) {
  final buffer = _malloc(12);
  if (buffer == nullptr) return 12;
  try {
    buffer.asTypedList(8).fillRange(0, 8, 0);
    final length = (buffer + 8).cast<Uint32>();
    length.value = 8;
    final getsockopt = _getsockopt;
    final errorAddress = _errno();
    final result = getsockopt(fd, level, option, buffer, length);
    final code = result < 0 ? errorAddress.value : 0;
    data.array.setRange(data.off, data.off + 8, buffer.asTypedList(8));
    return code;
  } finally {
    _free(buffer);
  }
}

int $syscallSetsockoptIPMreq(int fd, int level, int option, Slice<int> data) {
  final buffer = data.isNil ? nullptr.cast<Uint8>() : _malloc(8);
  if (!data.isNil && buffer == nullptr) return 12;
  try {
    if (!data.isNil) {
      buffer.asTypedList(8).setRange(0, 8, data.array, data.off);
    }
    final setsockopt = _setsockopt;
    final errorAddress = _errno();
    return setsockopt(fd, level, option, buffer, 8) < 0
        ? errorAddress.value
        : 0;
  } finally {
    _free(buffer);
  }
}

int $syscallGetsockoptIPMreqn(int fd, int level, int option, Slice<int> data) {
  final buffer = _malloc(16);
  if (buffer == nullptr) return 12;
  try {
    buffer.asTypedList(12).fillRange(0, 12, 0);
    final length = (buffer + 12).cast<Uint32>();
    length.value = 12;
    final getsockopt = _getsockopt;
    final errorAddress = _errno();
    final result = getsockopt(fd, level, option, buffer, length);
    final code = result < 0 ? errorAddress.value : 0;
    data.array.setRange(data.off, data.off + 12, buffer.asTypedList(12));
    return code;
  } finally {
    _free(buffer);
  }
}

int $syscallSetsockoptIPMreqn(int fd, int level, int option, Slice<int> data) {
  final buffer = data.isNil ? nullptr.cast<Uint8>() : _malloc(12);
  if (!data.isNil && buffer == nullptr) return 12;
  try {
    if (!data.isNil) {
      buffer.asTypedList(12).setRange(0, 12, data.array, data.off);
    }
    final setsockopt = _setsockopt;
    final errorAddress = _errno();
    return setsockopt(fd, level, option, buffer, 12) < 0
        ? errorAddress.value
        : 0;
  } finally {
    _free(buffer);
  }
}

int $syscallGetsockoptIPv6Mreq(int fd, int level, int option, Slice<int> data) {
  final buffer = _malloc(24);
  if (buffer == nullptr) return 12;
  try {
    buffer.asTypedList(20).fillRange(0, 20, 0);
    final length = (buffer + 20).cast<Uint32>();
    length.value = 20;
    final getsockopt = _getsockopt;
    final errorAddress = _errno();
    final result = getsockopt(fd, level, option, buffer, length);
    final code = result < 0 ? errorAddress.value : 0;
    data.array.setRange(data.off, data.off + 20, buffer.asTypedList(20));
    return code;
  } finally {
    _free(buffer);
  }
}

int $syscallSetsockoptIPv6Mreq(int fd, int level, int option, Slice<int> data) {
  final buffer = data.isNil ? nullptr.cast<Uint8>() : _malloc(20);
  if (!data.isNil && buffer == nullptr) return 12;
  try {
    if (!data.isNil) {
      buffer.asTypedList(20).setRange(0, 20, data.array, data.off);
    }
    final setsockopt = _setsockopt;
    final errorAddress = _errno();
    return setsockopt(fd, level, option, buffer, 20) < 0
        ? errorAddress.value
        : 0;
  } finally {
    _free(buffer);
  }
}

int $syscallSetsockoptTimeval(int fd, int level, int option, Slice<int> data) {
  final buffer = data.isNil ? nullptr.cast<Uint8>() : _malloc(16);
  if (!data.isNil && buffer == nullptr) return 12;
  try {
    if (!data.isNil) {
      buffer.asTypedList(16).setRange(0, 16, data.array, data.off);
    }
    final setsockopt = _setsockopt;
    final errorAddress = _errno();
    return setsockopt(fd, level, option, buffer, 16) < 0
        ? errorAddress.value
        : 0;
  } finally {
    _free(buffer);
  }
}

int $syscallGetsockoptICMPv6Filter(
  int fd,
  int level,
  int option,
  Slice<int> data,
) {
  final buffer = _malloc(36);
  if (buffer == nullptr) return 12;
  try {
    buffer.asTypedList(32).fillRange(0, 32, 0);
    final length = (buffer + 32).cast<Uint32>();
    length.value = 32;
    final getsockopt = _getsockopt;
    final errorAddress = _errno();
    final result = getsockopt(fd, level, option, buffer, length);
    final code = result < 0 ? errorAddress.value : 0;
    data.array.setRange(data.off, data.off + 32, buffer.asTypedList(32));
    return code;
  } finally {
    _free(buffer);
  }
}

int $syscallSetsockoptICMPv6Filter(
  int fd,
  int level,
  int option,
  Slice<int> data,
) {
  final buffer = data.isNil ? nullptr.cast<Uint8>() : _malloc(32);
  if (!data.isNil && buffer == nullptr) return 12;
  try {
    if (!data.isNil) {
      buffer.asTypedList(32).setRange(0, 32, data.array, data.off);
    }
    final setsockopt = _setsockopt;
    final errorAddress = _errno();
    return setsockopt(fd, level, option, buffer, 32) < 0
        ? errorAddress.value
        : 0;
  } finally {
    _free(buffer);
  }
}

int $syscallShutdown(int fd, int how) {
  final shutdown = _shutdown;
  final errorAddress = _errno();
  return shutdown(fd, how) < 0 ? errorAddress.value : 0;
}

int $syscallFlock(int fd, int how) {
  final flock = _flock;
  final errorAddress = _errno();
  return flock(fd, how) < 0 ? errorAddress.value : 0;
}

int $syscallListen(int fd, int how) {
  final listen = _listen;
  final errorAddress = _errno();
  return listen(fd, how) < 0 ? errorAddress.value : 0;
}

int $syscallSysinfo(Slice<int> data) {
  final buffer = data.isNil ? nullptr.cast<Uint8>() : _malloc(112);
  if (!data.isNil && buffer == nullptr) return 12;
  try {
    if (!data.isNil) {
      buffer.asTypedList(112).setRange(0, 112, data.array, data.off);
    }
    final sysinfo = _sysinfo;
    final errorAddress = _errno();
    final result = sysinfo(buffer);
    final code = result < 0 ? errorAddress.value : 0;
    if (!data.isNil) {
      data.array.setRange(data.off, data.off + 112, buffer.asTypedList(112));
    }
    return code;
  } finally {
    if (buffer != nullptr) _free(buffer);
  }
}

int $syscallGettid() => _gettid();

(int, int) $syscallTimes(Slice<int> data) {
  final buffer = data.isNil ? nullptr.cast<Uint8>() : _malloc(32);
  if (!data.isNil && buffer == nullptr) return (-1, 12);
  try {
    if (!data.isNil) {
      buffer.asTypedList(32).setRange(0, 32, data.array, data.off);
    }
    final times = _times;
    final errorAddress = _errno();
    final ticks = times(buffer);
    final code = ticks == -1 ? errorAddress.value : 0;
    if (!data.isNil) {
      data.array.setRange(data.off, data.off + 32, buffer.asTypedList(32));
    }
    return (ticks, code);
  } finally {
    if (buffer != nullptr) _free(buffer);
  }
}

(int, int) $syscallInotifyInit() {
  final init = _inotifyInit;
  final errorAddress = _errno();
  final fd = init();
  return (fd, fd < 0 ? errorAddress.value : 0);
}

(int, int) $syscallInotifyInit1(int flags) {
  final init = _inotifyInit1;
  final errorAddress = _errno();
  final fd = init(flags);
  return (fd, fd < 0 ? errorAddress.value : 0);
}

(int, int) $syscallInotifyAddWatch(int fd, GoString path, int mask) {
  final name = _malloc(path.length + 1);
  if (name == nullptr) return (-1, 12);
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    final add = _inotifyAddWatch;
    final errorAddress = _errno();
    final wd = add(fd, name, mask);
    return (wd, wd < 0 ? errorAddress.value : 0);
  } finally {
    _free(name);
  }
}

(int, int) $syscallInotifyRmWatch(int fd, int watchdesc) {
  final remove = _inotifyRmWatch;
  final errorAddress = _errno();
  final success = remove(fd, watchdesc);
  return (success, success < 0 ? errorAddress.value : 0);
}

(int, int) $syscallEpollCreate1(int flag) {
  final create = _epollCreate1;
  final errorAddress = _errno();
  final fd = create(flag);
  return (fd, fd < 0 ? errorAddress.value : 0);
}

int $syscallEpollCtl(int epfd, int op, int fd, Slice<int> data) {
  final buffer = data.isNil ? nullptr.cast<Uint8>() : _malloc(data.len);
  if (!data.isNil && buffer == nullptr) return 12;
  try {
    if (!data.isNil) {
      buffer.asTypedList(data.len).setRange(0, data.len, data.array, data.off);
    }
    final control = _epollCtl;
    final errorAddress = _errno();
    return control(epfd, op, fd, buffer) < 0 ? errorAddress.value : 0;
  } finally {
    if (buffer != nullptr) _free(buffer);
  }
}

(int, int) $syscallEpollWait(int epfd, Slice<int> data, int count, int msec) {
  final buffer = _malloc(data.len == 0 ? 1 : data.len);
  if (buffer == nullptr) return (-1, 12);
  try {
    buffer.asTypedList(data.len).setRange(0, data.len, data.array, data.off);
    final wait = _epollWait;
    final errorAddress = _errno();
    final n = wait(epfd, buffer, count, msec);
    final code = n < 0 ? errorAddress.value : 0;
    data.array.setRange(
      data.off,
      data.off + data.len,
      buffer.asTypedList(data.len),
    );
    return (n, code);
  } finally {
    _free(buffer);
  }
}

(int, int, int) $syscallPipe() {
  final buffer = _malloc(8);
  if (buffer == nullptr) return (0, 0, 12);
  try {
    final pipe = _rawPipe;
    final errorAddress = _errno();
    final result = pipe(buffer.cast<Int32>());
    if (result < 0) return (0, 0, errorAddress.value);
    final pair = buffer.cast<Int32>();
    return (pair[0], pair[1], 0);
  } finally {
    _free(buffer);
  }
}

(int, int, int) $syscallPipe2(int flags) {
  final buffer = _malloc(8);
  if (buffer == nullptr) return (0, 0, 12);
  try {
    final pipe2 = _pipe2;
    final errorAddress = _errno();
    final result = pipe2(buffer.cast<Int32>(), flags);
    if (result < 0) return (0, 0, errorAddress.value);
    final pair = buffer.cast<Int32>();
    return (pair[0], pair[1], 0);
  } finally {
    _free(buffer);
  }
}

(int, int) $syscallSplice(
  int rfd,
  Slice<int> read,
  int wfd,
  Slice<int> write,
  int length,
  int flags,
) {
  final splice = _splice;
  Pointer<Uint8> storage = nullptr;
  Pointer<Int64> readPointer = nullptr;
  Pointer<Int64> writePointer = nullptr;
  if (!read.isNil || !write.isNil) {
    storage = _malloc(16);
    if (storage == nullptr) return (-1, 12);
    if (!read.isNil) {
      readPointer = storage.cast<Int64>();
      for (var i = 0; i < 8; i++) {
        storage[i] = read[i];
      }
    }
    if (!write.isNil) {
      writePointer =
          !read.isNil &&
              identical(read.array, write.array) &&
              read.off == write.off
          ? readPointer
          : (storage + 8).cast<Int64>();
      final bytes = writePointer.cast<Uint8>();
      for (var i = 0; i < 8; i++) {
        bytes[i] = write[i];
      }
    }
  }
  try {
    final errorAddress = _errno();
    final n = splice(rfd, readPointer, wfd, writePointer, length, flags);
    final code = n < 0 ? errorAddress.value : 0;
    if (!read.isNil) {
      final bytes = readPointer.cast<Uint8>();
      for (var i = 0; i < 8; i++) {
        read[i] = bytes[i];
      }
    }
    if (!write.isNil) {
      final bytes = writePointer.cast<Uint8>();
      for (var i = 0; i < 8; i++) {
        write[i] = bytes[i];
      }
    }
    return (n, code);
  } finally {
    if (storage != nullptr) _free(storage);
  }
}

(int, int) $syscallTee(int rfd, int wfd, int length, int flags) {
  final tee = _tee;
  final errorAddress = _errno();
  final n = tee(rfd, wfd, length, flags);
  return (n, n < 0 ? errorAddress.value : 0);
}

(int, int) $syscallSendfile(int outfd, int infd, Slice<int> offset, int count) {
  final sendfile = _sendfile;
  Pointer<Uint8> storage = nullptr;
  if (!offset.isNil) {
    storage = _malloc(8);
    if (storage == nullptr) return (-1, 12);
    for (var i = 0; i < 8; i++) {
      storage[i] = offset[i];
    }
  }
  try {
    final errorAddress = _errno();
    final written = sendfile(outfd, infd, storage.cast<Int64>(), count);
    final code = written < 0 ? errorAddress.value : 0;
    if (!offset.isNil) {
      for (var i = 0; i < 8; i++) {
        offset[i] = storage[i];
      }
    }
    return (written, code);
  } finally {
    if (storage != nullptr) _free(storage);
  }
}

int $syscallExchangedata(GoString first, GoString second, int options) {
  if (first.bytes.contains(0) || second.bytes.contains(0)) return 22;
  final length = first.length + second.length + 2;
  final storage = _malloc(length);
  if (storage == nullptr) return 12;
  try {
    final bytes = storage.asTypedList(length);
    bytes.setRange(0, first.length, first.bytes);
    bytes[first.length] = 0;
    bytes.setRange(first.length + 1, length - 1, second.bytes);
    bytes[length - 1] = 0;
    final other = Pointer<Uint8>.fromAddress(
      storage.address + first.length + 1,
    );
    final exchangedata = _exchangedata;
    final errorAddress = _errno();
    return exchangedata(storage, other, options) < 0 ? errorAddress.value : 0;
  } finally {
    _free(storage);
  }
}

int $syscallNanosleep(Slice<int> request, Slice<int> leftover) {
  final buffers = Map<List<int>, Pointer<Uint8>>.identity();
  try {
    for (final data in [request, leftover]) {
      if (data.isNil || buffers.containsKey(data.array)) continue;
      final buffer = _malloc(data.len);
      if (buffer == nullptr) return 12;
      buffers[data.array] = buffer;
      buffer.asTypedList(data.len).setRange(0, data.len, data.array);
    }
    final nanosleep = _nanosleep;
    final errorAddress = _errno();
    final result = nanosleep(
      buffers[request.array] ?? nullptr,
      buffers[leftover.array] ?? nullptr,
    );
    final code = result < 0 ? errorAddress.value : 0;
    if (!leftover.isNil) {
      leftover.array.setRange(
        0,
        leftover.len,
        buffers[leftover.array]!.asTypedList(leftover.len),
      );
    }
    return code;
  } finally {
    for (final buffer in buffers.values) {
      _free(buffer);
    }
  }
}

int $syscallSyncFileRange(int fd, int offset, int length, int flags) {
  final syncFileRange = _syncFileRange;
  final errorAddress = _errno();
  return syncFileRange(fd, offset, length, flags) < 0 ? errorAddress.value : 0;
}

int $syscallFallocate(int fd, int mode, int offset, int length) {
  final fallocate = _fallocate;
  final errorAddress = _errno();
  return fallocate(fd, mode, offset, length) < 0 ? errorAddress.value : 0;
}

(int, int) $syscallSelect(
  int nfd,
  Slice<int> read,
  Slice<int> write,
  Slice<int> except,
  Slice<int> timeout,
) {
  final buffers = Map<List<int>, Pointer<Uint8>>.identity();
  try {
    for (final data in [read, write, except, timeout]) {
      if (data.isNil || buffers.containsKey(data.array)) continue;
      final buffer = _malloc(data.len);
      if (buffer == nullptr) return (-1, 12);
      buffers[data.array] = buffer;
      buffer.asTypedList(data.len).setRange(0, data.len, data.array);
    }
    final select = _select;
    final errorAddress = _errno();
    final count = select(
      nfd,
      buffers[read.array] ?? nullptr,
      buffers[write.array] ?? nullptr,
      buffers[except.array] ?? nullptr,
      buffers[timeout.array] ?? nullptr,
    );
    final code = count < 0 ? errorAddress.value : 0;
    for (final entry in buffers.entries) {
      final data = entry.key;
      data.setRange(0, data.length, entry.value.asTypedList(data.length));
    }
    return (count, code);
  } finally {
    for (final buffer in buffers.values) {
      _free(buffer);
    }
  }
}

int _socketAddressCall(
  int Function(int, Pointer<Uint8>, int) call,
  int fd,
  Slice<int> address,
) {
  final buffer = _malloc(address.len);
  if (buffer == nullptr) return 12;
  try {
    buffer
        .asTypedList(address.len)
        .setRange(0, address.len, address.array, address.off);
    final errorAddress = _errno();
    return call(fd, buffer, address.len) < 0 ? errorAddress.value : 0;
  } finally {
    _free(buffer);
  }
}

int $syscallBind(int fd, Slice<int> address) =>
    _socketAddressCall(_socketBind, fd, address);

int $syscallConnect(int fd, Slice<int> address) =>
    _socketAddressCall(_socketConnect, fd, address);

int $syscallSendto(int fd, Slice<int> data, int flags, Slice<int> address) {
  final destination = address.isNil
      ? nullptr.cast<Uint8>()
      : _malloc(address.len);
  if (!address.isNil && destination == nullptr) return 12;
  try {
    final payload = data.len == 0 ? nullptr.cast<Uint8>() : _malloc(data.len);
    if (data.len != 0 && payload == nullptr) return 12;
    try {
      if (!address.isNil) {
        destination
            .asTypedList(address.len)
            .setRange(0, address.len, address.array, address.off);
      }
      if (data.len != 0) {
        payload
            .asTypedList(data.len)
            .setRange(0, data.len, data.array, data.off);
      }
      final sendto = _socketSendto;
      final errorAddress = _errno();
      return sendto(fd, payload, data.len, flags, destination, address.len) < 0
          ? errorAddress.value
          : 0;
    } finally {
      if (payload != nullptr) _free(payload);
    }
  } finally {
    if (destination != nullptr) _free(destination);
  }
}

int _socketName(
  int Function(int, Pointer<Uint8>, Pointer<Uint32>) call,
  int fd,
  Slice<int> data,
) {
  final address = _malloc(data.len + 4);
  if (address == nullptr) return 12;
  try {
    final bytes = address.asTypedList(data.len);
    bytes.fillRange(0, bytes.length, 0);
    final length = Pointer<Uint32>.fromAddress(address.address + data.len);
    length.value = data.len;
    final errorAddress = _errno();
    if (call(fd, address, length) < 0) return errorAddress.value;
    data.array.setRange(data.off, data.off + data.len, bytes);
    return 0;
  } finally {
    _free(address);
  }
}

int $syscallGetsockname(int fd, Slice<int> data) =>
    _socketName(_getsockname, fd, data);

int $syscallGetpeername(int fd, Slice<int> data) =>
    _socketName(_getpeername, fd, data);

(int, int) $syscallRecvfrom(
  int fd,
  Slice<int> data,
  int flags,
  Slice<int> address,
) {
  final nativeAddress = _malloc(address.len + 4);
  if (nativeAddress == nullptr) return (-1, 12);
  try {
    final bytes = nativeAddress.asTypedList(address.len);
    bytes.fillRange(0, bytes.length, 0);
    final length = Pointer<Uint32>.fromAddress(
      nativeAddress.address + address.len,
    );
    length.value = address.len;
    final payload = _malloc(data.len == 0 ? 1 : data.len);
    if (payload == nullptr) return (-1, 12);
    try {
      final recvfrom = _recvfrom;
      final errorAddress = _errno();
      final count = recvfrom(
        fd,
        payload,
        data.len,
        flags,
        nativeAddress,
        length,
      );
      if (count < 0) return (count, errorAddress.value);
      final copied = count < data.len ? count : data.len;
      if (copied != 0) {
        data.array.setRange(
          data.off,
          data.off + copied,
          payload.asTypedList(copied),
        );
      }
      address.array.setRange(address.off, address.off + address.len, bytes);
      return (count, 0);
    } finally {
      _free(payload);
    }
  } finally {
    _free(nativeAddress);
  }
}

(int, int, int) $syscallAccept(int fd, int flags, Slice<int> address) {
  final nativeAddress = _malloc(address.len + 4);
  if (nativeAddress == nullptr) return (-1, 0, 12);
  try {
    final bytes = nativeAddress.asTypedList(address.len);
    bytes.fillRange(0, bytes.length, 0);
    final length = Pointer<Uint32>.fromAddress(
      nativeAddress.address + address.len,
    );
    length.value = address.len;
    final linuxAccept = Platform.isLinux ? _accept4 : null;
    final darwinAccept = Platform.isLinux ? null : _accept;
    final errorAddress = _errno();
    final accepted = linuxAccept != null
        ? linuxAccept(fd, nativeAddress, length, flags)
        : darwinAccept!(fd, nativeAddress, length);
    final code = accepted < 0 ? errorAddress.value : 0;
    if (accepted >= 0) {
      address.array.setRange(address.off, address.off + address.len, bytes);
    }
    return (accepted, length.value, code);
  } finally {
    _free(nativeAddress);
  }
}

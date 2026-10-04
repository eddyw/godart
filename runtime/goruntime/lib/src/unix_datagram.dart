// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'slice.dart';
import 'string.dart';
import 'syscall_directory.dart';

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
final _bind = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Void>, Uint32),
      int Function(int, Pointer<Void>, int)
    >('bind');
final _connect = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Void>, Uint32),
      int Function(int, Pointer<Void>, int)
    >('connect');
final _getsockname = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Void>, Pointer<Uint32>),
      int Function(int, Pointer<Void>, Pointer<Uint32>)
    >('getsockname');
final _getpeername = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Void>, Pointer<Uint32>),
      int Function(int, Pointer<Void>, Pointer<Uint32>)
    >('getpeername');
final _recvmsg = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Void>, Int),
      int Function(int, Pointer<Void>, int)
    >('recvmsg');
final _sendmsg = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Void>, Int),
      int Function(int, Pointer<Void>, int)
    >('sendmsg');
final _shutdown = _libc
    .lookupFunction<Int Function(Int, Int), int Function(int, int)>('shutdown');
final _setsockopt = _libc
    .lookupFunction<
      Int Function(Int, Int, Int, Pointer<Int>, Uint32),
      int Function(int, int, int, Pointer<Int>, int)
    >('setsockopt');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );
final _clock = Stopwatch()..start();

final class _Vector extends Struct {
  external Pointer<Uint8> data;
  @Size()
  external int length;
}

final class _LinuxMessage extends Struct {
  external Pointer<Void> name;
  @Uint32()
  external int nameLength;
  external Pointer<_Vector> vectors;
  @Size()
  external int vectorCount;
  external Pointer<Uint8> control;
  @Size()
  external int controlLength;
  @Int()
  external int flags;
}

final class _DarwinMessage extends Struct {
  external Pointer<Void> name;
  @Uint32()
  external int nameLength;
  external Pointer<_Vector> vectors;
  @Int()
  external int vectorCount;
  external Pointer<Uint8> control;
  @Uint32()
  external int controlLength;
  @Int()
  external int flags;
}

SocketException _error(int code) => SocketException(
  'Unix datagram operation failed',
  osError: OSError('native socket error', code),
);

int _address(Pointer<Uint8> memory, GoString name) {
  final path = name.bytes;
  final limit = Platform.isMacOS ? 104 : 108;
  final abstract =
      Platform.isLinux && path.isNotEmpty && (path[0] == 64 || path[0] == 0);
  if (path.length > limit ||
      path.length == limit && (!abstract || Platform.isMacOS)) {
    throw _error(22);
  }
  final length = path.isEmpty ? 2 : 2 + path.length + (abstract ? 0 : 1);
  final bytes = memory.asTypedList(128)..fillRange(0, 128, 0);
  if (Platform.isMacOS) {
    bytes[0] = length;
    bytes[1] = 1;
  } else {
    ByteData.sublistView(bytes).setUint16(0, 1, Endian.host);
  }
  bytes.setRange(2, 2 + path.length, path);
  if (abstract) bytes[2] = 0;
  return length;
}

GoString _name(Pointer<Uint8> memory, int length, {bool local = false}) {
  if (length == 0) return GoString.empty;
  final bytes = memory.asTypedList(128);
  final path = <int>[];
  var start = 2;
  if (Platform.isLinux && bytes[2] == 0) {
    // The SDK displays an unnamed local Unix socket as @. A recvmsg with no
    // source address has length 0 and remains distinct from that local name.
    if (length == 2 && !local) return GoString.empty;
    path.add(64);
    start++;
  }
  for (var i = start; i < length && bytes[i] != 0; i++) {
    path.add(bytes[i]);
  }
  return GoString.fromBytes(path);
}

final class _MessageBuffer {
  // Both supported 64-bit ABIs fit msghdr in 64 bytes and iovec in 16 bytes.
  // The following sockaddr buffer and control buffer retain 8-byte alignment.
  final Pointer<Uint8> memory;
  final int dataLength;
  final int controlLength;
  _MessageBuffer(this.dataLength, this.controlLength)
    : memory = _malloc(
        208 + controlLength + (dataLength == 0 ? 1 : dataLength),
      ) {
    if (memory == nullptr) throw _error(12);
    memory
        .asTypedList(208 + controlLength + (dataLength == 0 ? 1 : dataLength))
        .fillRange(
          0,
          208 + controlLength + (dataLength == 0 ? 1 : dataLength),
          0,
        );
    final vector = (memory + 64).cast<_Vector>();
    vector.ref
      ..data = data
      ..length = dataLength;
    if (Platform.isMacOS) {
      memory.cast<_DarwinMessage>().ref
        ..name = name.cast<Void>()
        ..nameLength = 128
        ..vectors = vector
        ..vectorCount = 1
        ..control = controlLength == 0 ? nullptr : control
        ..controlLength = controlLength;
    } else {
      memory.cast<_LinuxMessage>().ref
        ..name = name.cast<Void>()
        ..nameLength = 128
        ..vectors = vector
        ..vectorCount = 1
        ..control = controlLength == 0 ? nullptr : control
        ..controlLength = controlLength;
    }
  }
  Pointer<Uint8> get name => memory + 80;
  Pointer<Uint8> get control => memory + 208;
  Pointer<Uint8> get data => memory + 208 + controlLength;
  int get nameLength => Platform.isMacOS
      ? memory.cast<_DarwinMessage>().ref.nameLength
      : memory.cast<_LinuxMessage>().ref.nameLength;
  int get receivedControl => Platform.isMacOS
      ? memory.cast<_DarwinMessage>().ref.controlLength
      : memory.cast<_LinuxMessage>().ref.controlLength;
  int get flags => Platform.isMacOS
      ? memory.cast<_DarwinMessage>().ref.flags
      : memory.cast<_LinuxMessage>().ref.flags;
  void destination(GoString? address) {
    final length = address == null ? 0 : _address(name, address);
    if (Platform.isMacOS) {
      memory.cast<_DarwinMessage>().ref
        ..name = address == null ? nullptr : name.cast<Void>()
        ..nameLength = length;
    } else {
      memory.cast<_LinuxMessage>().ref
        ..name = address == null ? nullptr : name.cast<Void>()
        ..nameLength = length;
    }
  }

  void release() => _free(memory);
}

final class UnixDatagramClosed implements Exception {}

final class UnixDatagramExpired implements Exception {}

final class UnixDatagram {
  final int descriptor;
  final GoString local;
  final GoString? remote;
  bool closed = false;
  int? _readDeadline;
  int? _writeDeadline;
  Future<void> _reads = Future.value();
  Future<void> _writes = Future.value();
  final _waiters = <Completer<void>, Timer>{};

  UnixDatagram._(this.descriptor, this.local, this.remote);

  static UnixDatagram open(GoString? local, GoString? remote) {
    final (fd, code) = $syscallSocket(
      1,
      2 | (Platform.isLinux ? 0x80800 : 0),
      0,
    );
    if (code != 0) throw _error(code);
    try {
      if (Platform.isMacOS) {
        for (final (command, value) in [(2, 1), (4, 4)]) {
          final (_, error) = $syscallFcntl(fd, command, value);
          if (error != 0) throw _error(error);
        }
      }
      final buffer = _malloc(132);
      if (buffer == nullptr) throw _error(12);
      try {
        final bind = _bind;
        final connect = _connect;
        final getsockname = _getsockname;
        final getpeername = _getpeername;
        final error = _errno();
        if (local != null) {
          final length = _address(buffer, local);
          if (bind(fd, buffer.cast<Void>(), length) != 0) {
            throw _error(error.value);
          }
        }
        if (remote != null) {
          final length = _address(buffer, remote);
          if (connect(fd, buffer.cast<Void>(), length) != 0) {
            throw _error(error.value);
          }
        }
        buffer.asTypedList(132).fillRange(0, 132, 0);
        final length = (buffer + 128).cast<Uint32>()..value = 128;
        if (getsockname(fd, buffer.cast<Void>(), length) != 0) {
          throw _error(error.value);
        }
        final actualLocal = _name(buffer, length.value, local: true);
        GoString? actualRemote;
        if (remote != null) {
          buffer.asTypedList(128).fillRange(0, 128, 0);
          length.value = 128;
          if (getpeername(fd, buffer.cast<Void>(), length) != 0) {
            throw _error(error.value);
          }
          actualRemote = _name(buffer, length.value);
        }
        return UnixDatagram._(fd, actualLocal, actualRemote);
      } finally {
        _free(buffer);
      }
    } catch (_) {
      $syscallClose(fd);
      rethrow;
    }
  }

  void setDeadline(int timeout, int mode, bool clear) {
    if (closed) throw UnixDatagramClosed();
    final micros = timeout <= 0
        ? 0
        : timeout ~/ 1000 + (timeout % 1000 == 0 ? 0 : 1);
    final deadline = clear ? null : _clock.elapsedMicroseconds + micros;
    if (mode & 1 != 0) _readDeadline = deadline;
    if (mode & 2 != 0) _writeDeadline = deadline;
    _wake();
  }

  void _prepare(int? deadline) {
    if (closed) throw UnixDatagramClosed();
    if (deadline != null && deadline <= _clock.elapsedMicroseconds) {
      throw UnixDatagramExpired();
    }
  }

  Future<void> _retry() {
    final waiter = Completer<void>();
    _waiters[waiter] = Timer(const Duration(milliseconds: 1), () {
      _waiters.remove(waiter);
      waiter.complete();
    });
    return waiter.future;
  }

  void _wake() {
    for (final entry in _waiters.entries) {
      entry.value.cancel();
      entry.key.complete();
    }
    _waiters.clear();
  }

  Future<(int, int, int, GoString)> read(
    List<int> bytes,
    int start,
    int end,
    List<int> control,
    int controlStart,
    int controlEnd,
  ) {
    final result = _reads.then(
      (_) => _read(bytes, start, end, control, controlStart, controlEnd),
    );
    _reads = result.then<void>(
      (_) {},
      onError: (Object error, StackTrace stack) {},
    );
    return result;
  }

  Future<(int, int, int, GoString)> _read(
    List<int> bytes,
    int start,
    int end,
    List<int> control,
    int controlStart,
    int controlEnd,
  ) async {
    final dummy =
        Platform.isMacOS && end == start && controlEnd != controlStart;
    final buffer = _MessageBuffer(
      dummy ? 1 : end - start,
      controlEnd - controlStart,
    );
    try {
      final receive = _recvmsg;
      while (true) {
        _prepare(_readDeadline);
        if (Platform.isMacOS) {
          buffer.memory.cast<_DarwinMessage>().ref
            ..nameLength = 128
            ..controlLength = controlEnd - controlStart
            ..flags = 0;
        } else {
          buffer.memory.cast<_LinuxMessage>().ref
            ..nameLength = 128
            ..controlLength = controlEnd - controlStart
            ..flags = 0;
        }
        final error = _errno();
        final n = receive(
          descriptor,
          buffer.memory.cast<Void>(),
          Platform.isLinux ? 0x40000000 : 0,
        );
        if (n >= 0) {
          for (var i = 0; !dummy && i < n; i++) {
            bytes[start + i] = buffer.data[i];
          }
          final oob = buffer.receivedControl;
          if (Platform.isMacOS && oob != 0) {
            final ancillary = ByteData.sublistView(
              buffer.control.asTypedList(oob),
            );
            for (var offset = 0; offset + 12 <= oob;) {
              final length = ancillary.getUint32(offset, Endian.host);
              if (ancillary.getInt32(offset + 4, Endian.host) == 0xffff &&
                  ancillary.getInt32(offset + 8, Endian.host) == 1) {
                for (
                  var item = offset + 12;
                  item + 4 <= offset + length;
                  item += 4
                ) {
                  $syscallFcntl(ancillary.getInt32(item, Endian.host), 2, 1);
                }
              }
              offset += (length + 3) & ~3;
            }
          }
          for (var i = 0; i < oob; i++) {
            control[controlStart + i] = buffer.control[i];
          }
          return (n, oob, buffer.flags, _name(buffer.name, buffer.nameLength));
        }
        final code = error.value;
        if (code == 4) continue;
        if (code != (Platform.isMacOS ? 35 : 11)) throw _error(code);
        await _retry();
      }
    } finally {
      buffer.release();
    }
  }

  Future<int> write(
    List<int> bytes,
    int start,
    int end,
    List<int> control,
    int controlStart,
    int controlEnd,
    GoString? destination,
  ) {
    final result = _writes.then(
      (_) => _write(
        bytes,
        start,
        end,
        control,
        controlStart,
        controlEnd,
        destination,
      ),
    );
    _writes = result.then<void>(
      (_) {},
      onError: (Object error, StackTrace stack) {},
    );
    return result;
  }

  Future<int> _write(
    List<int> bytes,
    int start,
    int end,
    List<int> control,
    int controlStart,
    int controlEnd,
    GoString? destination,
  ) async {
    final dummy =
        Platform.isMacOS && end == start && controlEnd != controlStart;
    final buffer = _MessageBuffer(
      dummy ? 1 : end - start,
      controlEnd - controlStart,
    );
    try {
      buffer.data
          .asTypedList(end - start)
          .setRange(0, end - start, bytes, start);
      buffer.control
          .asTypedList(controlEnd - controlStart)
          .setRange(0, controlEnd - controlStart, control, controlStart);
      buffer.destination(destination);
      final send = _sendmsg;
      while (true) {
        _prepare(_writeDeadline);
        final error = _errno();
        final n = send(descriptor, buffer.memory.cast<Void>(), 0);
        if (n >= 0) return dummy ? 0 : n;
        final code = error.value;
        if (code == 4) continue;
        if (code != (Platform.isMacOS ? 35 : 11)) throw _error(code);
        await _retry();
      }
    } finally {
      buffer.release();
    }
  }

  void setBuffer(bool read, int value) {
    if (closed) throw UnixDatagramClosed();
    final memory = _malloc(sizeOf<Int>()).cast<Int>();
    if (memory == nullptr) throw _error(12);
    try {
      memory.value = value;
      final set = _setsockopt;
      final error = _errno();
      if (set(
            descriptor,
            Platform.isMacOS ? 0xffff : 1,
            Platform.isMacOS ? (read ? 0x1002 : 0x1001) : (read ? 8 : 7),
            memory,
            sizeOf<Int>(),
          ) !=
          0) {
        throw _error(error.value);
      }
    } finally {
      _free(memory.cast<Uint8>());
    }
  }

  void shutdown(int direction) {
    if (closed) throw UnixDatagramClosed();
    final shutdown = _shutdown;
    final error = _errno();
    if (shutdown(descriptor, direction) != 0) throw _error(error.value);
  }

  void close() {
    if (closed) throw UnixDatagramClosed();
    closed = true;
    _wake();
    final code = $syscallClose(descriptor);
    if (code != 0) throw _error(code);
  }
}

(int, int) $syscallSendmsgN(
  int fd,
  Slice<int> data,
  Slice<int> control,
  int flags,
  Slice<int> address,
) {
  late _MessageBuffer buffer;
  try {
    buffer = _MessageBuffer(data.len, control.len);
  } on SocketException catch (error) {
    return (0, error.osError!.errorCode);
  }
  try {
    if (data.len == 0) {
      (buffer.memory + 64).cast<_Vector>().ref.data = nullptr;
    } else {
      buffer.data.asTypedList(data.len).setRange(0, data.len, data.array, data.off);
    }
    buffer.control.asTypedList(control.len).setRange(0, control.len, control.array, control.off);
    if (!address.isNil) {
      buffer.name.asTypedList(address.len).setRange(0, address.len, address.array, address.off);
    }
    if (Platform.isMacOS) {
      buffer.memory.cast<_DarwinMessage>().ref
        ..name = address.isNil ? nullptr : buffer.name.cast<Void>()
        ..nameLength = address.len;
    } else {
      buffer.memory.cast<_LinuxMessage>().ref
        ..name = address.isNil ? nullptr : buffer.name.cast<Void>()
        ..nameLength = address.len;
    }
    final send = _sendmsg;
    final errorAddress = _errno();
    final count = send(fd, buffer.memory.cast<Void>(), flags);
    final code = count < 0 ? errorAddress.value : 0;
    return (count, code);
  } finally {
    buffer.release();
  }
}

(int, int, int, int) $syscallRecvmsg(
  int fd,
  Slice<int> data,
  Slice<int> control,
  int flags,
  Slice<int> address,
) {
  late _MessageBuffer buffer;
  try {
    buffer = _MessageBuffer(data.len, control.len);
  } on SocketException catch (error) {
    return (0, 0, 0, error.osError!.errorCode);
  }
  try {
    if (data.len == 0) {
      (buffer.memory + 64).cast<_Vector>().ref.data = nullptr;
    }
    buffer.control.asTypedList(control.len).setRange(0, control.len, control.array, control.off);
    if (Platform.isMacOS) {
      buffer.memory.cast<_DarwinMessage>().ref.nameLength = address.len;
    } else {
      buffer.memory.cast<_LinuxMessage>().ref.nameLength = address.len;
    }
    final receive = _recvmsg;
    final errorAddress = _errno();
    final count = receive(fd, buffer.memory.cast<Void>(), flags);
    if (count < 0) return (count, 0, 0, errorAddress.value);
    final receivedControl = buffer.receivedControl;
    final receivedFlags = buffer.flags;
    final copied = count < data.len ? count : data.len;
    if (copied != 0) {
      data.array.setRange(data.off, data.off + copied, buffer.data.asTypedList(copied));
    }
    final copiedControl = receivedControl < control.len ? receivedControl : control.len;
    if (copiedControl != 0) {
      control.array.setRange(control.off, control.off + copiedControl, buffer.control.asTypedList(copiedControl));
    }
    address.array.setRange(address.off, address.off + address.len, buffer.name.asTypedList(address.len));
    return (count, receivedControl, receivedFlags, 0);
  } finally {
    buffer.release();
  }
}

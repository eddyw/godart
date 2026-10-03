// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'file_attributes.dart';
import 'file_stat.dart';
import 'slice.dart';
import 'string.dart';
import 'syscall_directory.dart';

abstract interface class FileHandle {
  int get descriptor;
  int readIntoSync(List<int> bytes, int start, int end);
  int writeFromSync(List<int> bytes, int start, int end);
  void setPositionSync(int position);
  int positionSync();
  int lengthSync();
  void truncateSync(int size);
  void flushSync();
  void closeSync();
}

final class DartFileHandle implements FileHandle {
  final RandomAccessFile file;
  DartFileHandle(this.file);
  @override
  int get descriptor {
    // SCM_RIGHTS data borrows the descriptor. Only the file owns its lifetime.
    final control = SocketControlMessage.fromHandles([
      ResourceHandle.fromFile(file),
    ]);
    return ByteData.sublistView(control.data).getInt32(0, Endian.host);
  }

  @override
  int readIntoSync(List<int> bytes, int start, int end) =>
      file.readIntoSync(bytes, start, end);
  @override
  int writeFromSync(List<int> bytes, int start, int end) {
    file.writeFromSync(bytes, start, end);
    return end - start;
  }

  @override
  void setPositionSync(int position) => file.setPositionSync(position);
  @override
  int positionSync() => file.positionSync();
  @override
  int lengthSync() => file.lengthSync();
  @override
  void truncateSync(int size) => file.truncateSync(size);
  @override
  void flushSync() => file.flushSync();
  @override
  void closeSync() => file.closeSync();
}

final _libc = DynamicLibrary.process();
final _fcntl = _libc
    .lookupFunction<Int Function(Int, Int), int Function(int, int)>('fcntl');

int nativeFileFlags(int fd) {
  final flags = _fcntl(fd, 3);
  return flags < 0 ? 0 : flags;
}

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
final _read = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Uint8>, Size),
      int Function(int, Pointer<Uint8>, int)
    >('read');
final _write = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Uint8>, Size),
      int Function(int, Pointer<Uint8>, int)
    >('write');
final _pread = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Uint8>, Size, Int64),
      int Function(int, Pointer<Uint8>, int, int)
    >('pread');
final _pwrite = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Uint8>, Size, Int64),
      int Function(int, Pointer<Uint8>, int, int)
    >('pwrite');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );
final _strerror = _libc
    .lookupFunction<Pointer<Uint8> Function(Int), Pointer<Uint8> Function(int)>(
      'strerror',
    );

String _message(int code) {
  final text = _strerror(code);
  var size = 0;
  while (text[size] != 0) {
    size++;
  }
  return String.fromCharCodes(text.asTypedList(size));
}

FileSystemException _failure(int code) =>
    FileSystemException('', '', OSError(_message(code), code));

(int, int) $nativePread(int fd, Slice<int> bytes, int offset) {
  final read = _pread;
  final errno = _errno;
  final buffer = _malloc(bytes.len);
  if (buffer == nullptr) return (0, 12);
  try {
    int count;
    int code;
    do {
      count = read(fd, buffer, bytes.len, offset);
      code = count < 0 ? errno().value : 0;
    } while (code == 4);
    if (code != 0) return (0, code);
    if (count > 0) {
      bytes.array.setRange(
        bytes.off,
        bytes.off + count,
        buffer.asTypedList(count),
      );
    }
    return (count, 0);
  } finally {
    _free(buffer);
  }
}

(int, int) $nativePwrite(int fd, Slice<int> bytes, int offset) {
  final write = _pwrite;
  final errno = _errno;
  final buffer = _malloc(bytes.len);
  if (buffer == nullptr) return (0, 12);
  try {
    buffer
        .asTypedList(bytes.len)
        .setRange(0, bytes.len, bytes.array, bytes.off);
    int count;
    int code;
    do {
      count = write(fd, buffer, bytes.len, offset);
      code = count < 0 ? errno().value : 0;
    } while (code == 4);
    return (code == 0 ? count : 0, code);
  } finally {
    _free(buffer);
  }
}

final class PartialFileWrite extends FileSystemException {
  final int count;
  PartialFileWrite(this.count, int code)
    : super('', '', OSError(_message(code), code));
}

final class NativeFileHandle implements FileHandle {
  @override
  final int descriptor;
  NativeFileHandle(this.descriptor);

  static (NativeFileHandle?, int) open(GoString path, int flags, int mode) {
    final cloexec = Platform.isMacOS ? 0x1000000 : 0x80000;
    while (true) {
      final (fd, code) = $syscallOpen(path, flags | cloexec, mode);
      if (code == 4) continue;
      return (code == 0 ? NativeFileHandle(fd) : null, code);
    }
  }

  @override
  int readIntoSync(List<int> bytes, int start, int end) {
    final size = end - start;
    final buffer = _malloc(size);
    if (buffer == nullptr) throw _failure(12);
    try {
      final errno = _errno;
      int count;
      int code;
      do {
        count = _read(descriptor, buffer, size);
        code = count < 0 ? errno().value : 0;
      } while (code == 4);
      if (code != 0) throw _failure(code);
      bytes.setRange(start, start + count, buffer.asTypedList(count));
      return count;
    } finally {
      _free(buffer);
    }
  }

  @override
  int writeFromSync(List<int> bytes, int start, int end) {
    final size = end - start;
    final buffer = size == 0 ? nullptr.cast<Uint8>() : _malloc(size);
    if (size != 0 && buffer == nullptr) throw _failure(12);
    try {
      if (size != 0) buffer.asTypedList(size).setRange(0, size, bytes, start);
      final errno = _errno;
      var offset = 0;
      while (true) {
        final count = _write(descriptor, buffer + offset, size - offset);
        if (count < 0) {
          final code = errno().value;
          if (code == 4) continue;
          throw PartialFileWrite(offset, code);
        }
        offset += count;
        if (count == 0 || offset == size) return offset;
      }
    } finally {
      _free(buffer);
    }
  }

  @override
  void setPositionSync(int position) {
    final (_, code) = $syscallSeek(descriptor, position, 0);
    if (code != 0) throw _failure(code);
  }

  @override
  int positionSync() {
    final (position, code) = $syscallSeek(descriptor, 0, 1);
    if (code != 0) throw _failure(code);
    return position;
  }

  @override
  int lengthSync() {
    final (data, code) = $nativeFstat(descriptor);
    if (code != 0) throw _failure(code);
    return data.array[data.off];
  }

  @override
  void truncateSync(int size) {
    final code = $nativeFtruncate(descriptor, size);
    if (code != 0) throw _failure(code);
  }

  @override
  void flushSync() {
    final code = $nativeFsync(descriptor);
    if (code != 0) throw _failure(code);
  }

  @override
  void closeSync() {
    final code = $syscallClose(descriptor);
    if (code != 0) throw _failure(code);
  }
}

final _pipe = _libc
    .lookupFunction<Int Function(Pointer<Int>), int Function(Pointer<Int>)>(
      'pipe',
    );

final _pipe2 = _libc
    .lookupFunction<
      Int Function(Pointer<Int>, Int),
      int Function(Pointer<Int>, int)
    >('pipe2');

(NativePipeHandle?, NativePipeHandle?, int) $nativePipe() {
  final pipe = Platform.isMacOS
      ? _pipe
      : (Pointer<Int> pair) => _pipe2(pair, 0x80000 | 0x800);
  final errno = _errno;
  final descriptors = _malloc(2 * sizeOf<Int>());
  if (descriptors == nullptr) return (null, null, 12);
  try {
    final pair = descriptors.cast<Int>();
    int result;
    int code;
    do {
      result = pipe(pair);
      code = result < 0 ? errno().value : 0;
    } while (code == 4);
    if (code != 0) return (null, null, code);
    final reader = NativePipeHandle(pair[0]);
    final writer = NativePipeHandle(pair[1]);
    if (Platform.isMacOS) {
      for (final fd in [reader.descriptor, writer.descriptor]) {
        for (final command in [(2, 1), (4, 4)]) {
          final (_, error) = $syscallFcntl(fd, command.$1, command.$2);
          if (error != 0) {
            $syscallClose(reader.descriptor);
            $syscallClose(writer.descriptor);
            return (null, null, error);
          }
        }
      }
    }
    return (reader, writer, 0);
  } finally {
    _free(descriptors);
  }
}

final class ClosedPipe implements Exception {
  final int count;
  ClosedPipe(this.count);
}

final class ExpiredPipe implements Exception {
  final int count;
  ExpiredPipe(this.count);
}

final _pipeClock = Stopwatch()..start();

final class NativePipeHandle extends NativeFileHandle {
  bool _closed = false;
  int? _readDeadline;
  int? _writeDeadline;
  final _waiters = <Completer<void>, Timer>{};
  Future<void> _reads = Future.value();
  Future<void> _writes = Future.value();
  NativePipeHandle(super.descriptor);

  // Each retry reaches the actual nonblocking syscall. The timer releases
  // the scheduler without registering or replacing the descriptor in Dart.
  Future<void> _retry() {
    final waiter = Completer<void>();
    _waiters[waiter] = Timer(const Duration(milliseconds: 1), () {
      _waiters.remove(waiter);
      waiter.complete();
    });
    return waiter.future;
  }

  void setDeadline(int timeout, int mode, bool clear) {
    // Relative deadlines use the host monotonic clock. Microseconds avoid
    // overflow when Go saturates a distant future timeout at MaxInt64.
    final micros = timeout <= 0
        ? 0
        : timeout ~/ 1000 + (timeout % 1000 == 0 ? 0 : 1);
    final deadline = clear ? null : _pipeClock.elapsedMicroseconds + micros;
    if (mode & 1 != 0) _readDeadline = deadline;
    if (mode & 2 != 0) _writeDeadline = deadline;
    _wakeRetries();
  }

  bool _expired(int? micros) =>
      micros != null && micros <= _pipeClock.elapsedMicroseconds;

  void _wakeRetries() {
    for (final waiter in _waiters.entries) {
      waiter.value.cancel();
      waiter.key.complete();
    }
    _waiters.clear();
  }

  bool _wouldBlock(FileSystemException error) =>
      error.osError?.errorCode == (Platform.isMacOS ? 35 : 11);

  Future<int> read(List<int> bytes, int start, int end) {
    final next = _reads.then((_) => _read(bytes, start, end));
    _reads = next.then<void>((_) {}, onError: (Object _) {});
    return next;
  }

  Future<int> _read(List<int> bytes, int start, int end) async {
    while (true) {
      if (_closed) throw ClosedPipe(0);
      if (_expired(_readDeadline)) throw ExpiredPipe(0);
      try {
        return readIntoSync(bytes, start, end);
      } on FileSystemException catch (error) {
        if (!_wouldBlock(error)) rethrow;
        await _retry();
      }
    }
  }

  Future<int> write(List<int> bytes, int start, int end) {
    final next = _writes.then((_) => _write(bytes, start, end));
    _writes = next.then<void>((_) {}, onError: (Object _) {});
    return next;
  }

  Future<int> _write(List<int> bytes, int start, int end) async {
    var count = 0;
    while (true) {
      if (_closed) throw ClosedPipe(count);
      if (_expired(_writeDeadline)) throw ExpiredPipe(count);
      try {
        return count + writeFromSync(bytes, start + count, end);
      } on PartialFileWrite catch (error) {
        count += error.count;
        if (!_wouldBlock(error)) {
          throw PartialFileWrite(count, error.osError!.errorCode);
        }
        await _retry();
      }
    }
  }

  @override
  void closeSync() {
    _closed = true;
    _wakeRetries();
    super.closeSync();
  }
}

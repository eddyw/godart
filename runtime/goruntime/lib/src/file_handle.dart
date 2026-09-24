// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
    bytes.array.setRange(
      bytes.off,
      bytes.off + count,
      buffer.asTypedList(count),
    );
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

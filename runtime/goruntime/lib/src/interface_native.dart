// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'slice.dart';
import 'string.dart';

final class _Ifaddrs extends Struct {
  external Pointer<_Ifaddrs> next;
  external Pointer<Uint8> name;
  @Uint32()
  external int flags;
  external Pointer<Uint8> address;
  external Pointer<Uint8> netmask;
  external Pointer<Uint8> broadcast;
  external Pointer<Void> data;
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
final _getifaddrs = _libc
    .lookupFunction<
      Int Function(Pointer<Pointer<_Ifaddrs>>),
      int Function(Pointer<Pointer<_Ifaddrs>>)
    >('getifaddrs');
final _freeifaddrs = _libc
    .lookupFunction<
      Void Function(Pointer<_Ifaddrs>),
      void Function(Pointer<_Ifaddrs>)
    >('freeifaddrs');
final _index = _libc
    .lookupFunction<
      UnsignedInt Function(Pointer<Uint8>),
      int Function(Pointer<Uint8>)
    >('if_nametoindex');
final _socket = _libc
    .lookupFunction<Int Function(Int, Int, Int), int Function(int, int, int)>(
      'socket',
    );
final _close = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'close',
);
final _fcntl = _libc
    .lookupFunction<
      Int Function(Int, Int, VarArgs<(Int,)>),
      int Function(int, int, int)
    >('fcntl');
final _ioctl = _libc
    .lookupFunction<
      Int Function(Int, UnsignedLong, VarArgs<(Pointer<Uint8>,)>),
      int Function(int, int, Pointer<Uint8>)
    >('ioctl');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );
final _strerror = _libc
    .lookupFunction<Pointer<Uint8> Function(Int), Pointer<Uint8> Function(int)>(
      'strerror',
    );

Never _failure(String operation) {
  final error = _errno().value;
  final message = _strerror(error);
  final bytes = <int>[];
  for (var i = 0; message[i] != 0; i++) {
    bytes.add(message[i]);
  }
  throw OSError(
    '$operation: ${GoString.fromBytes(bytes).toDartString()}',
    error,
  );
}

Pointer<Uint8> _allocate(int size) {
  final result = _malloc(size);
  if (result == nullptr) throw const OSError('not enough memory', 12);
  return result;
}

int _flags(int raw) {
  var flags = 0;
  if (raw & 1 != 0) flags |= 1;
  if (raw & 2 != 0) flags |= 2;
  if (raw & 8 != 0) flags |= 4;
  if (raw & 16 != 0) flags |= 8;
  if (raw & (Platform.isMacOS ? 0x8000 : 0x1000) != 0) flags |= 16;
  if (raw & 64 != 0) flags |= 32;
  return flags;
}

Uint8List _hardware(Pointer<Uint8> address) {
  if (address == nullptr) return Uint8List(0);
  if (Platform.isMacOS) {
    final offset = 8 + address[5], length = address[6];
    if (offset + length > address[0]) {
      throw const FormatException('invalid link address length');
    }
    return Uint8List.fromList((address + offset).asTypedList(length));
  }
  final length = address[11], type = (address + 8).cast<Uint16>().value;
  if (length > 24) {
    throw const FormatException('unsupported link address length');
  }
  if (length == 4 && (type == 768 || type == 778 || type == 776) ||
      length == 16 && (type == 769 || type == 823)) {
    return Uint8List(0);
  }
  final bytes = (address + 12).asTypedList(length);
  return bytes.any((b) => b != 0) ? Uint8List.fromList(bytes) : Uint8List(0);
}

(Slice<GoString>, GoString) $interfaceTable() {
  if (!(Platform.isLinux || Platform.isMacOS)) {
    return (
      Slice<GoString>.nil(),
      GoString.fromString('native interface metadata is not supported'),
    );
  }
  try {
    return (Slice.of(_table()), GoString.empty);
  } catch (e) {
    return (
      Slice<GoString>.nil(),
      GoString.fromString(e is OSError ? e.message : e.toString()),
    );
  }
}

List<GoString> _table() {
  final head = _allocate(sizeOf<Pointer<_Ifaddrs>>()).cast<Pointer<_Ifaddrs>>();
  head.value = nullptr;
  Pointer<Uint8> request = nullptr;
  var fd = -1;
  try {
    request = _allocate(40);
    fd = _socket(2, 2 | (Platform.isLinux ? 0x80000 : 0), 0);
    if (fd < 0) _failure('socket');
    if (Platform.isMacOS && _fcntl(fd, 2, 1) < 0) _failure('fcntl');
    if (_getifaddrs(head) != 0) _failure('getifaddrs');
    final rows = <int, GoString>{};
    for (var cursor = head.value; cursor != nullptr; cursor = cursor.ref.next) {
      final entry = cursor.ref;
      final address = entry.address;
      if (address != nullptr) {
        final family = Platform.isMacOS
            ? address[1]
            : address.cast<Uint16>().value;
        if (family != (Platform.isMacOS ? 18 : 17)) continue;
      }
      final name = <int>[];
      for (var i = 0; i < 16 && entry.name[i] != 0; i++) {
        name.add(entry.name[i]);
      }
      if (name.isEmpty || name.length == 16) {
        throw const FormatException('invalid interface name');
      }
      final index = _index(entry.name);
      if (index == 0) _failure('if_nametoindex');
      request.asTypedList(40)
        ..fillRange(0, 40, 0)
        ..setRange(0, name.length, name);
      if (_ioctl(fd, Platform.isMacOS ? 0xc0206933 : 0x8921, request) < 0) {
        _failure('ioctl');
      }
      final mtu = (request + 16).cast<Int32>().value;
      final hardware = _hardware(address)
          .map((b) => b.toRadixString(16).padLeft(2, '0'))
          .join();
      final prefix = GoString.fromString(
        '$index|$mtu|${_flags(entry.flags)}|$hardware|',
      );
      rows[index] = GoString.fromBytes([...prefix.bytes, ...name]);
    }
    final indexes = rows.keys.toList()..sort();
    return [for (final index in indexes) rows[index]!];
  } finally {
    if (head.value != nullptr) _freeifaddrs(head.value);
    _free(head.cast());
    if (request != nullptr) _free(request);
    if (fd >= 0) _close(fd);
  }
}

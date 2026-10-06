// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'native_errno.dart';

final _libc = DynamicLibrary.process();
final _allocate = _libc
    .lookupFunction<Pointer<Void> Function(Size), Pointer<Void> Function(int)>(
      'malloc',
    );
final _release = _libc
    .lookupFunction<Void Function(Pointer<Void>), void Function(Pointer<Void>)>(
      'free',
    );
final _strerror = _libc
    .lookupFunction<Pointer<Uint8> Function(Int), Pointer<Uint8> Function(int)>(
      'strerror',
    );
final _ifindex = _libc
    .lookupFunction<
      UnsignedInt Function(Pointer<Uint8>),
      int Function(Pointer<Uint8>)
    >('if_nametoindex');

SocketException datagramError(int code) {
  final message = _strerror(code);
  var length = 0;
  while (message[length] != 0) {
    length++;
  }
  return SocketException(
    'UDP operation failed',
    osError: OSError(utf8.decode(message.asTypedList(length)), code),
  );
}

(int, String, String) connectDatagram(
  RawDatagramSocket socket,
  InternetAddress remote,
  int port,
) {
  final fd = _datagramFD(socket);
  final ipv6 = remote.type == InternetAddressType.IPv6;
  final length = ipv6 ? 28 : 16;
  final memory = _allocate(length + 4);
  if (memory == nullptr) throw datagramError(12);
  try {
    _writeAddress(memory, remote, port);
    final bytes = memory.cast<Uint8>().asTypedList(length);
    final connected = nativeConnect(fd, memory, length);
    if (connected.value != 0) throw datagramError(connected.error);
    final size = (memory.cast<Uint8>() + length).cast<Uint32>()..value = length;
    final localName = nativeGetsockname(fd, memory, size);
    if (localName.value != 0) {
      throw datagramError(localName.error);
    }
    final local = _readAddress(bytes, ipv6);
    size.value = length;
    // An empty peer lets the overlay retain the original Go address when getpeername fails.
    final peer = nativeGetpeername(fd, memory, size).value == 0
        ? _readAddress(bytes, ipv6)
        : '';
    return (fd, local, peer);
  } finally {
    _release(memory);
  }
}

String _readAddress(Uint8List bytes, bool ipv6) {
  final address = InternetAddress.fromRawAddress(
    Uint8List.fromList(bytes.sublist(ipv6 ? 8 : 4, ipv6 ? 24 : 8)),
  );
  final scope = ipv6
      ? ByteData.sublistView(bytes).getUint32(24, Endian.host)
      : 0;
  return scope == 0 ? address.address : '${address.address}%$scope';
}

int sendUnconnectedDatagram(RawDatagramSocket socket, Uint8List bytes) {
  return sendDatagram(_datagramFD(socket), bytes);
}

int sendDatagram(int fd, Uint8List bytes) {
  final memory = _allocate(bytes.isEmpty ? 1 : bytes.length);
  if (memory == nullptr) throw datagramError(12);
  try {
    memory.cast<Uint8>().asTypedList(bytes.length).setAll(0, bytes);
    final result = nativeSend(fd, memory, bytes.length, 0);
    if (result.value < 0) throw datagramError(result.error);
    return result.value;
  } finally {
    _release(memory);
  }
}

int datagramPendingError(RawDatagramSocket socket) {
  final option = RawSocketOption.fromInt(
    RawSocketOption.levelSocket,
    Platform.isMacOS ? 0x1007 : 4,
    0,
  );
  return ByteData.sublistView(socket.getRawOption(option))
      .getInt32(0, Endian.host);
}

int _datagramFD(RawDatagramSocket socket) {
  final control = SocketControlMessage.fromHandles([
    ResourceHandle.fromRawDatagramSocket(socket),
  ]);
  // SCM_RIGHTS stores native C int descriptors on Linux and macOS.
  return ByteData.sublistView(control.data).getInt32(0, Endian.host);
}

void _writeAddress(Pointer<Void> memory, InternetAddress remote, int port) {
  final ipv6 = remote.type == InternetAddressType.IPv6;
  final length = ipv6 ? 28 : 16;
  final bytes = memory.cast<Uint8>().asTypedList(length)
    ..fillRange(0, length, 0);
  final data = ByteData.sublistView(bytes);
  final family = ipv6 ? (Platform.isMacOS ? 30 : 10) : 2;
  if (Platform.isMacOS) {
    bytes[0] = length;
    bytes[1] = family;
  } else {
    data.setUint16(0, family, Endian.host);
  }
  data.setUint16(2, port, Endian.big);
  bytes.setRange(
    ipv6 ? 8 : 4,
    (ipv6 ? 8 : 4) + remote.rawAddress.length,
    remote.rawAddress,
  );
  if (ipv6 && remote.address.contains('%')) {
    final zone = remote.address.substring(remote.address.indexOf('%') + 1);
    var index = int.tryParse(zone);
    if (index == null) {
      final name = utf8.encode(zone);
      final buffer = _allocate(name.length + 1).cast<Uint8>();
      if (buffer == nullptr) throw datagramError(12);
      try {
        buffer.asTypedList(name.length + 1)
          ..setRange(0, name.length, name)
          ..[name.length] = 0;
        index = _ifindex(buffer);
      } finally {
        _release(buffer.cast());
      }
    }
    data.setUint32(24, index, Endian.host);
  }
}

int sendDatagramTo(
  RawDatagramSocket socket,
  Uint8List bytes,
  InternetAddress remote,
  int port,
) {
  final fd = _datagramFD(socket);
  final length = remote.type == InternetAddressType.IPv6 ? 28 : 16;
  final memory = _allocate(length + bytes.length);
  if (memory == nullptr) throw datagramError(12);
  try {
    _writeAddress(memory, remote, port);
    final payload = memory.cast<Uint8>() + length;
    payload.asTypedList(bytes.length).setAll(0, bytes);
    final result = nativeSendto(
      fd,
      payload.cast(),
      bytes.length,
      0,
      memory,
      length,
    );
    if (result.value < 0) throw datagramError(result.error);
    return result.value;
  } finally {
    _release(memory);
  }
}

Datagram? receiveEmptyDatagram(RawDatagramSocket socket) {
  final fd = _datagramFD(socket);
  final memory = _allocate(33);
  if (memory == nullptr) throw datagramError(12);
  try {
    final bytes = memory.cast<Uint8>();
    final size = (bytes + 28).cast<Uint32>()..value = 28;
    final result = nativeRecvfrom(fd, (bytes + 32).cast(), 1, 2, memory, size);
    if (result.value > 0) return null;
    if (result.value < 0) {
      final code = result.error;
      if (code == (Platform.isMacOS ? 35 : 11)) return null;
      throw datagramError(code);
    }
    final data = ByteData.sublistView(bytes.asTypedList(28));
    final family = Platform.isMacOS ? bytes[1] : data.getUint16(0, Endian.host);
    final ipv6 = family == (Platform.isMacOS ? 30 : 10);
    final address = InternetAddress.fromRawAddress(
      Uint8List.fromList(
        bytes.asTypedList(28).sublist(ipv6 ? 8 : 4, ipv6 ? 24 : 8),
      ),
    );
    final port = data.getUint16(2, Endian.big);
    // Dart discards empty datagrams. Its receive call also refreshes read readiness.
    socket.receive();
    return Datagram(Uint8List(0), address, port);
  } finally {
    _release(memory);
  }
}

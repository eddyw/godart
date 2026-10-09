// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:async';
import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'file_handle.dart';
import 'native_errno.dart';
import 'syscall_directory.dart';

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

SocketException tcpError(
  int code, [
  String operation = 'TCP operation failed',
]) {
  final text = _strerror(code);
  var length = 0;
  while (text[length] != 0) {
    length++;
  }
  return SocketException(
    operation,
    osError: OSError(utf8.decode(text.asTypedList(length)), code),
  );
}

bool _wouldBlock(int code) => code == (Platform.isMacOS ? 35 : 11);

(InternetAddress, int) _name(int fd, {bool peer = false}) {
  final memory = _allocate(32);
  if (memory == nullptr) throw tcpError(12);
  try {
    final bytes = memory.cast<Uint8>();
    final length = (bytes + 28).cast<Uint32>()..value = 28;
    final result = peer
        ? nativeGetpeername(fd, memory, length)
        : nativeGetsockname(fd, memory, length);
    if (result.value < 0) {
      throw tcpError(result.error, peer ? 'getpeername' : 'getsockname');
    }
    final data = ByteData.sublistView(bytes.asTypedList(28));
    final family = Platform.isMacOS ? bytes[1] : data.getUint16(0, Endian.host);
    final ipv6 = family == (Platform.isMacOS ? 30 : 10);
    if (family != 2 && !ipv6) {
      throw UnsupportedError('FileListener supports TCP descriptors only');
    }
    final raw = Uint8List.fromList(
      bytes.asTypedList(28).sublist(ipv6 ? 8 : 4, ipv6 ? 24 : 8),
    );
    var address = InternetAddress.fromRawAddress(raw);
    final scope = ipv6 ? data.getUint32(24, Endian.host) : 0;
    if (scope != 0) address = InternetAddress('${address.address}%$scope');
    return (address, data.getUint16(2, Endian.big));
  } finally {
    _release(memory);
  }
}

final class NativeTcpServer {
  final int descriptor;
  final InternetAddress address;
  final int port;
  bool _closed = false;

  NativeTcpServer._(this.descriptor, this.address, this.port);

  static NativeTcpServer bind(
    InternetAddress address,
    int port,
    bool ipv6Only,
  ) {
    final ipv6 = address.type == InternetAddressType.IPv6;
    final length = ipv6 ? 28 : 16;
    final memory = _allocate(length);
    if (memory == nullptr) throw tcpError(12);
    try {
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
      bytes.setRange(ipv6 ? 8 : 4, ipv6 ? 24 : 8, address.rawAddress);
      if (ipv6 && address.address.contains('%')) {
        final zone = address.address.substring(
          address.address.indexOf('%') + 1,
        );
        var index = int.tryParse(zone);
        if (index == null) {
          final text = utf8.encode(zone);
          final name = _allocate(text.length + 1).cast<Uint8>();
          if (name == nullptr) throw tcpError(12);
          try {
            name.asTypedList(text.length + 1)
              ..setRange(0, text.length, text)
              ..[text.length] = 0;
            index = _ifindex(name);
          } finally {
            _release(name.cast());
          }
        }
        data.setUint32(24, index, Endian.host);
      }
      final result = nativeTcpListen(memory, length, ipv6Only ? 1 : 0);
      if (result.value < 0) throw tcpError(result.error);
      final fd = result.value;
      try {
        final (bound, port) = _name(fd);
        return NativeTcpServer._(fd, bound, port);
      } catch (_) {
        NativeFileHandle(fd).closeSync();
        rethrow;
      }
    } finally {
      _release(memory);
    }
  }

  static NativeTcpServer fromFile(int descriptor) {
    final result = nativeTcpAdopt(descriptor);
    if (result.value < 0) {
      throw tcpError(result.error, switch (result.operation) {
        1 => 'fcntl',
        2 => 'getsockopt',
        _ => '',
      });
    }
    final fd = result.value;
    try {
      final (address, port) = _name(fd);
      final (value, code) = $syscallFcntl(
        fd,
        4,
        nativeFileFlags(fd) | (Platform.isMacOS ? 4 : 0x800),
      );
      if (value < 0) throw tcpError(code, 'fcntl');
      return NativeTcpServer._(fd, address, port);
    } catch (_) {
      NativeFileHandle(fd).closeSync();
      rethrow;
    }
  }

  int duplicate() {
    final result = nativeTcpDuplicate(descriptor);
    if (result.value < 0) throw tcpError(result.error);
    return result.value;
  }

  Stream<NativeTcpSocket> get incoming async* {
    while (!_closed) {
      final result = nativeTcpAccept(descriptor);
      if (result.value < 0) {
        if (result.error == 4) continue;
        if (!_wouldBlock(result.error)) throw tcpError(result.error);
        await Future<void>.delayed(const Duration(milliseconds: 1));
        continue;
      }
      final fd = result.value;
      NativeTcpSocket socket;
      try {
        socket = NativeTcpSocket(fd);
      } catch (_) {
        NativeFileHandle(fd).closeSync();
        rethrow;
      }
      yield socket;
    }
  }

  Future<void> close() async {
    if (_closed) return;
    _closed = true;
    NativeFileHandle(descriptor).closeSync();
  }
}

final class NativeTcpSocket {
  final int descriptor;
  late final (InternetAddress, int) local;
  late final (InternetAddress, int) remote;
  bool closed = false;

  NativeTcpSocket(this.descriptor) {
    local = _name(descriptor);
    remote = _name(descriptor, peer: true);
  }

  Future<Uint8List> read() async {
    final memory = _allocate(32768);
    if (memory == nullptr) throw tcpError(12);
    try {
      while (!closed) {
        final result = nativeRead(descriptor, memory, 32768);
        if (result.value >= 0) {
          return Uint8List.fromList(
            memory.cast<Uint8>().asTypedList(result.value),
          );
        }
        if (result.error == 4) continue;
        if (!_wouldBlock(result.error)) throw tcpError(result.error);
        await Future<void>.delayed(const Duration(milliseconds: 1));
      }
      return Uint8List(0);
    } finally {
      _release(memory);
    }
  }

  int write(Uint8List bytes, int start) {
    final count = bytes.length - start;
    final memory = _allocate(count == 0 ? 1 : count);
    if (memory == nullptr) throw tcpError(12);
    try {
      memory.cast<Uint8>().asTypedList(count).setRange(0, count, bytes, start);
      while (true) {
        final result = nativeSend(
          descriptor,
          memory,
          count,
          Platform.isLinux ? 0x4000 : 0,
        );
        if (result.value >= 0) return result.value;
        if (result.error == 4) continue;
        if (_wouldBlock(result.error)) return 0;
        throw tcpError(result.error);
      }
    } finally {
      _release(memory);
    }
  }

  void shutdown(int direction) {
    final result = nativeShutdown(descriptor, direction);
    if (result.value < 0) throw tcpError(result.error);
  }

  void setOption(RawSocketOption option) {
    final bytes = option.value;
    final memory = _allocate(bytes.length);
    if (memory == nullptr) throw tcpError(12);
    try {
      memory.cast<Uint8>().asTypedList(bytes.length).setAll(0, bytes);
      final result = nativeSetsockopt(
        descriptor,
        option.level,
        option.option,
        memory,
        bytes.length,
      );
      if (result.value < 0) throw tcpError(result.error);
    } finally {
      _release(memory);
    }
  }

  void close() {
    if (closed) return;
    closed = true;
    NativeFileHandle(descriptor).closeSync();
  }
}

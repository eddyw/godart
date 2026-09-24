import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

final _libc = DynamicLibrary.process();
final _allocate = _libc
    .lookupFunction<Pointer<Void> Function(Size), Pointer<Void> Function(int)>(
      'malloc',
    );
final _release = _libc
    .lookupFunction<Void Function(Pointer<Void>), void Function(Pointer<Void>)>(
      'free',
    );
final _connect = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Void>, Uint32),
      int Function(int, Pointer<Void>, int)
    >('connect');
final _send = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Void>, Size, Int),
      int Function(int, Pointer<Void>, int, int)
    >('send');
final _sendto = _libc
    .lookupFunction<
      IntPtr Function(Int, Pointer<Void>, Size, Int, Pointer<Void>, Uint32),
      int Function(int, Pointer<Void>, int, int, Pointer<Void>, int)
    >('sendto');
final _recvfrom = _libc
    .lookupFunction<
      IntPtr Function(
        Int,
        Pointer<Void>,
        Size,
        Int,
        Pointer<Void>,
        Pointer<Uint32>,
      ),
      int Function(int, Pointer<Void>, int, int, Pointer<Void>, Pointer<Uint32>)
    >('recvfrom');
final _getsockname = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Void>, Pointer<Uint32>),
      int Function(int, Pointer<Void>, Pointer<Uint32>)
    >('getsockname');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
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

(int, String) connectDatagram(
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
    final data = ByteData.sublistView(bytes);
    if (_connect(fd, memory, length) != 0) throw datagramError(_errno().value);
    final size = (memory.cast<Uint8>() + length).cast<Uint32>()..value = length;
    if (_getsockname(fd, memory, size) != 0) {
      throw datagramError(_errno().value);
    }
    final local = InternetAddress.fromRawAddress(
      Uint8List.fromList(bytes.sublist(ipv6 ? 8 : 4, ipv6 ? 24 : 8)),
    );
    final scope = ipv6 ? data.getUint32(24, Endian.host) : 0;
    return (fd, scope == 0 ? local.address : '${local.address}%$scope');
  } finally {
    _release(memory);
  }
}

int sendDatagram(int fd, Uint8List bytes) {
  final memory = _allocate(bytes.isEmpty ? 1 : bytes.length);
  if (memory == nullptr) throw datagramError(12);
  try {
    memory.cast<Uint8>().asTypedList(bytes.length).setAll(0, bytes);
    final count = _send(fd, memory, bytes.length, 0);
    if (count < 0) throw datagramError(_errno().value);
    return count;
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
    final count = _sendto(fd, payload.cast(), bytes.length, 0, memory, length);
    if (count < 0) throw datagramError(_errno().value);
    return count;
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
    final count = _recvfrom(fd, (bytes + 32).cast(), 1, 2, memory, size);
    if (count > 0) return null;
    if (count < 0) {
      final code = _errno().value;
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

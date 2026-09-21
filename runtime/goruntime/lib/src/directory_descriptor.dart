import 'dart:ffi';
import 'dart:io';

import 'string.dart';

final _supported = Platform.isLinux || Platform.isMacOS;
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
final _open = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Int),
      int Function(Pointer<Uint8>, int)
    >('open');
final _close = _libc.lookupFunction<Int Function(Int), int Function(int)>(
  'close',
);
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

(int, int) $directoryOpen(GoString path) {
  // Other hosts retain path-based directory access without a native descriptor.
  if (!_supported) return (-1, 0);
  final bytes = path.bytes;
  if (bytes.contains(0)) return (-1, 22);
  final name = _malloc(bytes.length + 1);
  if (name == nullptr) return (-1, 12);
  try {
    name.asTypedList(bytes.length + 1)
      ..setRange(0, bytes.length, bytes)
      ..[bytes.length] = 0;
    final flags = Platform.isMacOS ? 0x1000000 | 0x100000 : 0x80000 | 0x10000;
    while (true) {
      final fd = _open(name, flags);
      final error = fd < 0 ? _errno().value : 0;
      if (error != 4) return (fd, error);
    }
  } finally {
    _free(name);
  }
}

int $directoryFd(int fd) {
  if (!_supported) throw UnsupportedError('native directory descriptors');
  return fd;
}

int $directoryClose(int fd) {
  if (!_supported) return 0;
  return _close(fd) == 0 ? 0 : _errno().value;
}

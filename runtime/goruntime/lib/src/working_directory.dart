import 'dart:ffi';
import 'dart:io';

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
final _chdir = _libc
    .lookupFunction<Int Function(Pointer<Uint8>), int Function(Pointer<Uint8>)>(
      'chdir',
    );
final _getcwd = _libc
    .lookupFunction<
      Pointer<Uint8> Function(Pointer<Uint8>, Size),
      Pointer<Uint8> Function(Pointer<Uint8>, int)
    >('getcwd');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

int $nativeChdir(GoString path) {
  if (path.bytes.contains(0)) return 22;
  final name = _malloc(path.length + 1);
  if (name == nullptr) return 12;
  try {
    name.asTypedList(path.length + 1)
      ..setRange(0, path.length, path.bytes)
      ..[path.length] = 0;
    return _chdir(name) == 0 ? 0 : _errno().value;
  } finally {
    _free(name);
  }
}

(GoString, int) $nativeGetwd() {
  for (var size = 256; ; size *= 2) {
    final buffer = _malloc(size);
    if (buffer == nullptr) return (GoString.empty, 12);
    try {
      Pointer<Uint8> result;
      int error;
      do {
        result = _getcwd(buffer, size);
        error = result == nullptr ? _errno().value : 0;
      } while (error == 4);
      if (error == 34) continue;
      if (error != 0) return (GoString.empty, error);
      final bytes = buffer.asTypedList(size);
      return (GoString.fromBytes(bytes.sublist(0, bytes.indexOf(0))), 0);
    } finally {
      _free(buffer);
    }
  }
}

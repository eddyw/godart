import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'slice.dart';

final _libc = DynamicLibrary.process();
final _fcntl = _libc
    .lookupFunction<
      Int Function(Int, Int, VarArgs<(Pointer<Uint8>,)>),
      int Function(int, int, Pointer<Uint8>)
    >('fcntl');
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
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

(Slice<int>, int) $syscallFcntlFlock(
  int fd,
  int cmd,
  bool input,
  int start,
  int length,
  int pid,
  int kind,
  int whence,
  int pad0,
  int pad1,
) {
  final command = cmd.toSigned(32);
  final first = Platform.isMacOS ? 7 : 5;
  // Other fcntl pointer commands require different native buffer sizes.
  if (command >= 0 && (command < first || command > first + 2)) {
    return (Slice<int>.nil(), -1);
  }
  final original = [kind, whence, start, length, pid, pad0, pad1];
  final size = Platform.isMacOS ? 24 : 32;
  final buffer = input ? _malloc(size) : nullptr.cast<Uint8>();
  if (input && buffer == nullptr) return (Slice.of(original), 12);
  try {
    if (input) {
      final data = ByteData.sublistView(buffer.asTypedList(size));
      if (Platform.isMacOS) {
        data.setInt64(0, start, Endian.host);
        data.setInt64(8, length, Endian.host);
        data.setInt32(16, pid, Endian.host);
        data.setInt16(20, kind, Endian.host);
        data.setInt16(22, whence, Endian.host);
      } else {
        data.setInt16(0, kind, Endian.host);
        data.setInt16(2, whence, Endian.host);
        data.setUint32(4, pad0, Endian.host);
        data.setInt64(8, start, Endian.host);
        data.setInt64(16, length, Endian.host);
        data.setInt32(24, pid, Endian.host);
        data.setUint32(28, pad1, Endian.host);
      }
    }
    final fcntl = _fcntl;
    final errorAddress = _errno();
    final value = fcntl(fd, command, buffer);
    final error = value < 0 ? errorAddress.value : 0;
    if (!input) return (Slice.of(original), error);
    final data = ByteData.sublistView(buffer.asTypedList(size));
    return (
      Slice.of(
        Platform.isMacOS
            ? [
                data.getInt16(20, Endian.host),
                data.getInt16(22, Endian.host),
                data.getInt64(0, Endian.host),
                data.getInt64(8, Endian.host),
                data.getInt32(16, Endian.host),
                0,
                0,
              ]
            : [
                data.getInt16(0, Endian.host),
                data.getInt16(2, Endian.host),
                data.getInt64(8, Endian.host),
                data.getInt64(16, Endian.host),
                data.getInt32(24, Endian.host),
                data.getUint32(4, Endian.host),
                data.getUint32(28, Endian.host),
              ],
      ),
      error,
    );
  } finally {
    _free(buffer);
  }
}

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
final _rename = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>),
      int Function(Pointer<Uint8>, Pointer<Uint8>)
    >('rename');
final _renameat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Int, Pointer<Uint8>),
      int Function(int, Pointer<Uint8>, int, Pointer<Uint8>)
    >('renameat');
final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

int $nativeRename(GoString from, GoString to) =>
    _renamePaths(from, to, _rename);

int $rootRename(int oldParent, GoString from, int newParent, GoString to) =>
    _renamePaths(
      from,
      to,
      (oldName, newName) => _renameat(oldParent, oldName, newParent, newName),
    );

int _renamePaths(
  GoString from,
  GoString to,
  int Function(Pointer<Uint8>, Pointer<Uint8>) rename,
) {
  if (from.bytes.contains(0) || to.bytes.contains(0)) return 22;
  final oldName = _malloc(from.length + 1);
  if (oldName == nullptr) return 12;
  try {
    final newName = _malloc(to.length + 1);
    if (newName == nullptr) return 12;
    try {
      oldName.asTypedList(from.length + 1)
        ..setRange(0, from.length, from.bytes)
        ..[from.length] = 0;
      newName.asTypedList(to.length + 1)
        ..setRange(0, to.length, to.bytes)
        ..[to.length] = 0;
      // Resolve the symbol before rename because lookup can change errno.
      final errno = _errno;
      int error;
      do {
        error = rename(oldName, newName) == 0 ? 0 : errno().value;
      } while (error == 4);
      return error;
    } finally {
      _free(newName);
    }
  } finally {
    _free(oldName);
  }
}

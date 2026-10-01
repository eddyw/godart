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
final _link = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>),
      int Function(Pointer<Uint8>, Pointer<Uint8>)
    >('link');
final _linkat = _libc
    .lookupFunction<
      Int Function(Int, Pointer<Uint8>, Int, Pointer<Uint8>, Int),
      int Function(int, Pointer<Uint8>, int, Pointer<Uint8>, int)
    >('linkat');
final _symlink = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Pointer<Uint8>),
      int Function(Pointer<Uint8>, Pointer<Uint8>)
    >('symlink');
final _symlinkat = _libc
    .lookupFunction<
      Int Function(Pointer<Uint8>, Int, Pointer<Uint8>),
      int Function(Pointer<Uint8>, int, Pointer<Uint8>)
    >('symlinkat');

final _errno = _libc
    .lookupFunction<Pointer<Int> Function(), Pointer<Int> Function()>(
      Platform.isMacOS ? '__error' : '__errno_location',
    );

int $nativeLink(GoString from, GoString to) => _createLink(from, to, _link);

int $nativeSymlink(GoString from, GoString to) =>
    _createLink(from, to, _symlink);

int $rootLink(int oldParent, GoString from, int newParent, GoString to) =>
    _createLink(
      from,
      to,
      (oldName, newName) => _linkat(oldParent, oldName, newParent, newName, 0),
    );

int $rootSymlink(GoString target, int parent, GoString path) => _createLink(
  target,
  path,
  (oldName, newName) => _symlinkat(oldName, parent, newName),
);

int _createLink(
  GoString from,
  GoString to,
  int Function(Pointer<Uint8>, Pointer<Uint8>) create,
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
      int error;
      do {
        error = create(oldName, newName) == 0 ? 0 : _errno().value;
      } while (error == 4);
      return error;
    } finally {
      _free(newName);
    }
  } finally {
    _free(oldName);
  }
}

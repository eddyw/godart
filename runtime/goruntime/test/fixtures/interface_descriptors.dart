import 'dart:ffi';

import 'package:goruntime/goruntime.dart';

void main() {
  final libc = DynamicLibrary.process();
  final dup = libc.lookupFunction<Int Function(Int), int Function(int)>('dup');
  final close = libc.lookupFunction<Int Function(Int), int Function(int)>(
    'close',
  );
  int nextDescriptor() {
    final fd = dup(1);
    if (fd < 0 || close(fd) != 0) throw StateError('descriptor probe failed');
    return fd;
  }

  final before = nextDescriptor();
  for (var i = 0; i < 100; i++) {
    final result = $interfaceTable();
    if (result.$2.length != 0) throw StateError(result.$2.toDartString());
  }
  final after = nextDescriptor();
  if (before != after) {
    throw StateError('next descriptor changed: $before -> $after');
  }
}

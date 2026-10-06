import 'dart:ffi';

import 'package:goruntime/src/native_errno.dart';

void main() {
  final operations = [
    () => nativeConnect(-1, nullptr, 0),
    () => nativeSend(-1, nullptr, 0, 0),
    () => nativeSendto(-1, nullptr, 0, 0, nullptr, 0),
    () => nativeRecvfrom(-1, nullptr, 0, 0, nullptr, nullptr),
    () => nativeGetsockname(-1, nullptr, nullptr),
    () => nativeGetpeername(-1, nullptr, nullptr),
  ];
  for (final operation in operations) {
    final result = operation();
    if (result.value != -1 || result.error != 9) {
      throw StateError('invalid descriptor: ${result.value} ${result.error}');
    }
  }
  print('atomic UDP errno passes');
}

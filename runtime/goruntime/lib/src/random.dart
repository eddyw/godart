import 'dart:math';
import 'dart:typed_data';

import 'slice.dart';

/// The secure source of the platform. Random.secure reads the generator of the
/// operating system, and the web build reads window.crypto.
final Random _secure = Random.secure();

/// Gives n bytes of the secure source of the platform.
Slice<int> $randomBytes(int n) {
  final Uint8List out = Uint8List(n);
  for (int i = 0; i < n; i++) {
    out[i] = _secure.nextInt(256);
  }
  return Slice.of(out);
}

/// The runtime library of the godart compiler.
///
/// The compiler emits Dart code that calls this library. All state in this
/// library is per isolate.
library;

export 'dart:typed_data';

export 'src/array.dart';
export 'src/iface.dart';
export 'src/int.dart';
export 'src/map.dart';
export 'src/panic.dart';
export 'src/print.dart';
export 'src/slice.dart';
export 'src/string.dart';

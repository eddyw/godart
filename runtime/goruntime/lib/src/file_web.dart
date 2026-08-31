/// The web form of file.dart. A web target holds no file access, thus this
/// file holds only the names that each program reads. A program that reaches
/// a file operation names a function that this file does not hold, and the
/// Dart compile of the web target fails there, per the web decision of
/// 2026-09-01.
library;

import 'string.dart';

/// The name of the operating system. The wasm target runs under a JS host,
/// and Go names that system js.
GoString $goos() => GoString.fromString('js');

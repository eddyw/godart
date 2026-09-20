/// The process access of the scheduler and of the print helpers, on the Dart
/// VM. A web build takes platform_web.dart in its place.
library;

import 'dart:io';

String $platformGODEBUG() => Platform.environment['GODEBUG'] ?? '';

/// Writes bytes to standard output.
void $platformStdout(List<int> bytes) => stdout.add(bytes);

/// Writes bytes to standard error.
void $platformStderr(List<int> bytes) => stderr.add(bytes);

/// Writes text to standard error.
void $platformStderrText(String text) => stderr.write(text);

/// Ends the program with a status code.
Never $platformExit(int code) => exit(code);

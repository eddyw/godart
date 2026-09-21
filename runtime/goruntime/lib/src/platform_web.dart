/// The process access of the scheduler and of the print helpers, on a web
/// target. The target holds no dart:io, thus each write goes through the
/// print function of Dart, which reaches the console of the host.
///
/// The print function writes 1 line for each call, thus the writes gather in
/// a buffer, and each full line leaves the buffer at once.
library;

import 'dart:convert';
import 'dart:js_interop';

@JS('globalThis.process')
external _Process? get _process;

extension type _Process._(JSObject _) implements JSObject {
  external void exit(int code);
}

String $platformGODEBUG() => '';

String _outBuffer = '';
String _errBuffer = '';

/// Writes bytes to standard output.
void $platformStdout(List<int> bytes) {
  _outBuffer = _flush(_outBuffer + utf8.decode(bytes, allowMalformed: true));
}

/// Writes bytes to standard error.
void $platformStderr(List<int> bytes) {
  _errBuffer = _flush(_errBuffer + utf8.decode(bytes, allowMalformed: true));
}

/// Writes text to standard error.
void $platformStderrText(String text) {
  _errBuffer = _flush(_errBuffer + text);
}

/// Prints each full line of the text and gives the rest back.
String _flush(String text) {
  var at = text.indexOf('\n');
  while (at >= 0) {
    print(text.substring(0, at));
    text = text.substring(at + 1);
    at = text.indexOf('\n');
  }
  return text;
}

/// Ends a Node process with the requested status. A browser has no process
/// control, thus it reports an exception after the output buffers flush.
Never $platformExit(int code) {
  if (_outBuffer.isNotEmpty) print(_outBuffer);
  if (_errBuffer.isNotEmpty) print(_errBuffer);
  _outBuffer = '';
  _errBuffer = '';
  _process?.exit(code);
  throw StateError('exit $code');
}

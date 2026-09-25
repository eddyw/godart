/// Node exposes byte streams. Browsers receive complete console lines.
library;

import 'dart:convert';
import 'dart:js_interop';
import 'dart:typed_data';

@JS('globalThis.process')
external _Process? get _process;

extension type _Process._(JSObject _) implements JSObject {
  external void exit(int code);
  external _Output get stdout;
  external _Output get stderr;
}

extension type _Output._(JSObject _) implements JSObject {
  external bool write(JSAny value);
}

String $platformGODEBUG() => '';

String _outBuffer = '';
String _errBuffer = '';

/// Writes bytes to standard output.
void $platformStdout(List<int> bytes) {
  final process = _process;
  if (process != null) {
    process.stdout.write(Uint8List.fromList(bytes).toJS);
    return;
  }
  _outBuffer = _flush(_outBuffer + utf8.decode(bytes, allowMalformed: true));
}

/// Writes bytes to standard error.
void $platformStderr(List<int> bytes) {
  final process = _process;
  if (process != null) {
    process.stderr.write(Uint8List.fromList(bytes).toJS);
    return;
  }
  _errBuffer = _flush(_errBuffer + utf8.decode(bytes, allowMalformed: true));
}

/// Writes text to standard error.
void $platformStderrText(String text) {
  final process = _process;
  if (process != null) {
    process.stderr.write(text.toJS);
    return;
  }
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

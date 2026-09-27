/// Node exposes byte streams. Browsers receive complete console lines.
library;

import 'dart:convert';
import 'dart:js_interop';
import 'dart:js_interop_unsafe';
import 'dart:typed_data';

import 'process_exit.dart';

@JS('globalThis.process')
external _Process? get _process;

extension type _Process._(JSObject _) implements JSObject {
  external JSObject get env;
  external void exit(int code);
  external _Output get stdout;
  external _Output get stderr;
}

extension type _Output._(JSObject _) implements JSObject {
  external bool write(JSAny value, [JSFunction? callback]);
}

@JS('Object.keys')
external JSArray<JSString> _keys(JSObject object);

final Map<String, String> $platformEnvironment = _environment();

Map<String, String> _environment() {
  final process = _process;
  if (process == null) return const {};
  final env = process.env;
  return Map<String, String>.unmodifiable({
    for (final key in _keys(env).toDart)
      key.toDart: env.getProperty<JSString>(key).toDart,
  });
}

String $platformGODEBUG() => $platformEnvironment['GODEBUG'] ?? '';

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

bool _exiting = false;
bool get $platformExiting => _exiting;

/// Ends a Node process with the requested status. A browser has no process
/// control, thus it reports an exception after the output buffers flush.
Never $platformExit(int code) {
  final process = _process;
  if (process != null) {
    if (!_exiting) {
      _exiting = true;
      var pending = 2;
      void complete(JSAny? error) {
        if (--pending == 0) process.exit(code);
      }

      // Empty writes complete after each stream drains its prior output.
      process.stdout.write(''.toJS, complete.toJS);
      process.stderr.write(''.toJS, complete.toJS);
    }
    throw const ProcessExit();
  }
  if (_outBuffer.isNotEmpty) print(_outBuffer);
  if (_errBuffer.isNotEmpty) print(_errBuffer);
  _outBuffer = '';
  _errBuffer = '';
  throw StateError('exit $code');
}

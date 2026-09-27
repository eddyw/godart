/// The fetch client of the web target.
///
/// A browser holds no socket, thus net/http reaches the network through the
/// fetch API of the host. The Go side picks this path when runtime.GOOS gives
/// "js".
///
/// The body of an answer rides a ReadableStream, thus a read takes 1 chunk at
/// a time and the program holds no whole body in memory. A host that gives no
/// such stream falls back to the whole array, exactly as Go does under
/// GOOS=js.
library;

import 'dart:js_interop';
import 'dart:js_interop_unsafe';
import 'dart:typed_data';

import 'chan.dart';
import 'goint.dart';
import 'sched.dart';
import 'slice.dart';
import 'string.dart';

@JS('fetch')
external JSPromise<_Response> _fetch(String url, _Init init);

extension type _Init._(JSObject _) implements JSObject {
  external factory _Init({
    String method,
    String credentials,
    _Headers headers,
    JSAny? body,
    JSAny? signal,
  });
}

@JS('Headers')
extension type _Headers._(JSObject _) implements JSObject {
  external factory _Headers();
  external void append(String name, String value);
  external void forEach(JSFunction each);
}

extension type _Response._(JSObject _) implements JSObject {
  external int get status;
  external String get statusText;
  external bool get redirected;
  external String get url;
  external _Headers get headers;
  external _Stream? get body;
  external JSPromise<JSArrayBuffer> arrayBuffer();
}

extension type _Stream._(JSObject _) implements JSObject {
  external _Reader getReader();
}

extension type _Reader._(JSObject _) implements JSObject {
  external JSPromise<_Chunk> read();
  external JSPromise<JSAny?> cancel();
}

extension type _Chunk._(JSObject _) implements JSObject {
  external bool get done;
  external JSUint8Array? get value;
}

extension type _Abort._(JSObject _) implements JSObject {
  external JSAny get signal;
  external void abort();
}

/// The body of 1 answer.
///
/// reader holds the stream of the host. A host that gives no stream leaves it
/// null, and rest then holds the whole body.
final class _Body {
  final _Reader? reader;

  /// The bytes that a read did not take. A Go read takes no more bytes than
  /// its slice holds.
  Uint8List rest;

  bool ended = false;

  /// The deadline of the request, in nanoseconds after the start of the Unix
  /// epoch. A value of 0 holds no limit.
  final int deadline;

  /// The controller of the request. An abort ends the body at a deadline.
  final _Abort? stop;

  _Body(this.reader, this.rest, this.deadline, this.stop);
}

final class _Request {
  final _Abort? stop = _controller();
  _Body? body;
}

final Map<int, _Request> _requests = <int, _Request>{};
// Cancellation can arrive after close. A new request must not reuse that handle.
int _nextRequest = 0;

int $fetchOpen() {
  final int handle = _nextRequest++;
  _requests[handle] = _Request();
  return handle;
}

void $fetchCancel(int handle) {
  _requests[handle]?.stop?.abort();
}

final GoString _empty = GoString.empty;

final GoString _noFetch = GoString.fromString('operation not supported');

GoString _closed() => GoString.fromString('use of closed response body');

GoString _timeout() => GoString.fromString('i/o timeout');

/// _cause gives the text of a failure of the host. The message of a fetch
/// failure names no reason on purpose, because a browser hides the reason of a
/// blocked request.
GoString _cause(Object cause) => GoString.fromString(cause.toString());

/// _limit ends a wait at a deadline and gives null at that end. A deadline of
/// 0 holds no limit, and a deadline that passed already gives null at once.
///
/// The source future takes a then that widens its type. A timeout of a future
/// of an extension type reads the type of the object itself, thus a plain
/// widening at the call gives a type error on dart2wasm.
Future<T?> _limit<T>(Future<T> f, int deadline) {
  if (deadline == 0) return f;
  final int left = deadline - $ix($nowNanos());
  if (left <= 0) return Future<T?>.value(null);
  return f
      .then<T?>((T value) => value)
      .timeout(Duration(microseconds: left ~/ 1000), onTimeout: () => null);
}

/// True when the host holds the fetch API. A host without it reports the same
/// cause as the Dart VM.
final bool _has = globalContext.has('fetch');

_Abort? _controller() {
  final JSAny? make = globalContext['AbortController'];
  if (make == null) return null;
  return (make as JSFunction).callAsConstructor<_Abort>();
}

Uint8List _bytes(Slice<GoInt> p) {
  final Uint8List out = Uint8List(p.len);
  for (int i = 0; i < p.len; i++) {
    out[i] = $ix(p.array[p.off + i]) & 0xFF;
  }
  return out;
}

/// _take moves the held bytes of a body into the slice and gives the count.
int _take(_Body b, Slice<GoInt> p) {
  final int n = b.rest.length < p.len ? b.rest.length : p.len;
  for (int i = 0; i < n; i++) {
    p[i] = $gi(b.rest[i]);
  }
  b.rest = n == b.rest.length ? Uint8List(0) : Uint8List.sublistView(b.rest, n);
  return n;
}

/// _head reads the status, the header, and the first state of the body.
///
/// The result rides the frame of the wait, thus each integer carries the Go
/// form. A plain int reaches the caller of $fetchStart, and that result is
/// dead. Read the note on $await.
Future<(GoInt, GoString, Slice<GoString>, GoString, GoString)> _head(
  _Request request,
  _Response r,
  int deadline,
) async {
  final List<GoString> fields = <GoString>[];
  r.headers.forEach(
    ((JSString value, JSString name, JSAny parent) {
      fields.add(GoString.fromString(name.toDart));
      // Fetch represents each header byte as 1 string code unit.
      fields.add(GoString.fromBytes(value.toDart.codeUnits));
    }).toJS,
  );
  final _Stream? stream = r.body;
  final _Body b = stream == null
      ? _Body(
          null,
          (await r.arrayBuffer().toDart).toDart.asUint8List(),
          deadline,
          request.stop,
        )
      : _Body(stream.getReader(), Uint8List(0), deadline, request.stop);
  request.body = b;
  return (
    $gi(r.status),
    GoString.fromBytes(r.statusText.codeUnits),
    Slice<GoString>.of(fields),
    r.redirected ? GoString.fromString(r.url) : _empty,
    _empty,
  );
}

/// Sends a request and waits for the head of the answer.
///
/// The call gives up control, thus the result of the call itself is dead. The
/// frame of the wait gives the value to the caller.
(int, GoString, Slice<GoString>, GoString, GoString) $fetchStart(
  int handle,
  GoString method,
  GoString url,
  Slice<GoString> header,
  Slice<GoInt> body,
  int deadline,
) {
  if (!_has) {
    return (0, _empty, Slice<GoString>.nil(), _empty, _noFetch);
  }
  final _Headers fields = _Headers();
  for (int i = 0; i + 1 < header.len; i += 2) {
    fields.append(
      header[i].toDartString(),
      String.fromCharCodes(header[i + 1].bytes),
    );
  }
  final _Request request = _requests[handle]!;
  final _Abort? stop = request.stop;
  final _Init init = _Init(
    method: method.toDartString(),
    credentials: 'same-origin',
    headers: fields,
    body: body.len == 0 ? null : _bytes(body).toJS,
    signal: stop?.signal,
  );
  $await(
    _limit<_Response>(_fetch(url.toDartString(), init).toDart, deadline).then((
      _Response? r,
    ) async {
      if (r == null) {
        stop?.abort();
        return ($gi(0), _empty, Slice<GoString>.nil(), _empty, _timeout());
      }
      return await _head(request, r, deadline);
    }),
    (Object cause) =>
        ($gi(0), _empty, Slice<GoString>.nil(), _empty, _cause(cause)),
  );
  return (0, _empty, Slice<GoString>.nil(), _empty, _empty);
}

/// Reads bytes of the body of an answer into the slice and gives the count. A
/// count of 0 with no cause means the end of the body.
(int, GoString) $fetchRead(int handle, Slice<GoInt> p) {
  final _Body? b = _requests[handle]?.body;
  if (b == null) return (0, _closed());
  if (p.len == 0) return (0, _empty);
  if (b.rest.isNotEmpty) return (_take(b, p), _empty);
  final _Reader? reader = b.reader;
  if (b.ended || reader == null) return (0, _empty);
  $await(
    _limit<_Chunk>(reader.read().toDart, b.deadline).then((_Chunk? chunk) {
      if (chunk == null) {
        b.stop?.abort();
        return ($gi(0), _timeout());
      }
      final JSUint8Array? value = chunk.value;
      if (chunk.done || value == null) {
        b.ended = true;
        return ($gi(0), _empty);
      }
      b.rest = value.toDart;
      return ($gi(_take(b, p)), _empty);
    }),
    (Object cause) => ($gi(0), _cause(cause)),
  );
  return (0, _empty);
}

/// Frees a request handle and cancels its response reader.
GoString $fetchClose(int handle) {
  final _Request? request = _requests.remove(handle);
  if (request == null) return _closed();
  // Cancel can reject after an abort. Read already reports the request cause.
  request.body?.reader?.cancel().toDart.ignore();
  return _empty;
}

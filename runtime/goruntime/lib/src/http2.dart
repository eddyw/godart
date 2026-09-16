import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:http2/transport.dart' as h2;

import 'net.dart';
import 'sched.dart';
import 'slice.dart';
import 'string.dart';

/// One HTTP/2 session of a client.
final class _ClientSession {
  final h2.ClientTransportConnection conn;
  _ClientSession(this.conn);
}

/// One HTTP/2 session of a server. The iterator takes 1 stream of a client
/// at a time.
final class _ServerSession {
  final h2.ServerTransportConnection conn;
  final StreamIterator<h2.ServerTransportStream> incoming;
  _ServerSession(this.conn)
    : incoming = StreamIterator<h2.ServerTransportStream>(conn.incomingStreams);
}

/// The open sessions of the program, by handle.
final List<Object?> _sessions = <Object?>[];

/// One open stream. The iterator takes 1 message of the stream at a time,
/// and the rest holds the bytes of a data message that a read did not take.
final class _H2Stream {
  final h2.TransportStream stream;
  final StreamIterator<h2.StreamMessage> messages;
  Uint8List rest = Uint8List(0);
  bool ended = false;
  Future<int>? pending;

  _H2Stream(this.stream)
    : messages = StreamIterator<h2.StreamMessage>(stream.incomingMessages);
}

/// The open streams of the program, by handle. The streams of a client and
/// of a server share 1 table, thus 1 read call serves the 2 sides.
final List<_H2Stream?> _streams = <_H2Stream?>[];

/// True after the release of the sessions stands in the list of the
/// scheduler. An open session keeps the isolate alive, and Go ends the
/// process when package main returns, thus the release ends each one.
bool _released = false;

void _register() {
  if (_released) return;
  _released = true;
  $releases.add(_closeAll);
}

void _closeAll() {
  for (int i = 0; i < _sessions.length; i++) {
    final Object? held = _sessions[i];
    if (held == null) continue;
    _sessions[i] = null;
    if (held is _ClientSession) unawaited(held.conn.terminate());
    if (held is _ServerSession) unawaited(held.conn.terminate());
  }
}

int _sessionSlot(Object session) {
  _register();
  for (int i = 0; i < _sessions.length; i++) {
    if (_sessions[i] == null) {
      _sessions[i] = session;
      return i;
    }
  }
  _sessions.add(session);
  return _sessions.length - 1;
}

int _streamSlot(_H2Stream s) {
  for (int i = 0; i < _streams.length; i++) {
    if (_streams[i] == null) {
      _streams[i] = s;
      return i;
    }
  }
  _streams.add(s);
  return _streams.length - 1;
}

_H2Stream? _streamAt(int handle) {
  if (handle < 0 || handle >= _streams.length) return null;
  return _streams[handle];
}

final GoString _none = GoString.empty;

GoString _text(Object cause) => GoString.fromString(cause.toString());

GoString _gone() => GoString.fromString('use of closed network connection');

Slice<GoString> _noFields() => Slice.of(<GoString>[]);

/// The bytes of a slice, as a list of Dart.
///
/// A byte slice holds its storage in a Uint8List, thus setRange moves the whole
/// range in 1 call. A loop over the bytes reads and writes 1 element at a time,
/// and a write of 1 megabyte then costs a million bounds checks. A store into a
/// Uint8List holds the low 8 bits, exactly as the mask of the loop did.
Uint8List _h2Bytes(Slice<int> p) {
  final Uint8List out = Uint8List(p.len);
  if (p.len != 0) out.setRange(0, p.len, p.array, p.off);
  return out;
}

/// _fields builds the header list of the wire from the flat pairs, behind
/// the pseudo headers of the start.
List<h2.Header> _fields(Slice<GoString> flat, List<h2.Header> pseudo) {
  final List<h2.Header> out = <h2.Header>[...pseudo];
  for (int i = 0; i + 1 < flat.len; i += 2) {
    out.add(
      h2.Header.ascii(flat[i].toDartString(), flat[i + 1].toDartString()),
    );
  }
  return out;
}

/// _flat gives the flat pairs of a header list, without the pseudo headers.
Slice<GoString> _flat(List<h2.Header> headers) {
  final List<GoString> out = <GoString>[];
  for (final h2.Header h in headers) {
    final String name = String.fromCharCodes(h.name);
    if (name.startsWith(':')) continue;
    out.add(GoString.fromString(name));
    out.add(GoString.fromString(String.fromCharCodes(h.value)));
  }
  return Slice.of(out);
}

/// Wraps an open TLS connection whose ALPN took h2. The socket leaves the
/// table of the connections, and the session drives it.
(int, GoString) $h2Client(int conn) {
  final Socket? socket = $takeSocket(conn);
  if (socket == null) return (-1, _gone());
  try {
    final h2.ClientTransportConnection session =
        h2.ClientTransportConnection.viaSocket(socket);
    return (_sessionSlot(_ClientSession(session)), _none);
  } catch (cause) {
    return (-1, _text(cause));
  }
}

/// Runs 1 request on a session of a client. The results are the handle of
/// the stream of the body, the status, the flat response headers, and the
/// cause of a failure.
(int, int, Slice<GoString>, GoString) $h2Fetch(
  int session,
  GoString method,
  GoString path,
  GoString authority,
  Slice<GoString> headers,
  Slice<int> body,
  bool hasBody,
) {
  final Object? held = session >= 0 && session < _sessions.length
      ? _sessions[session]
      : null;
  if (held is! _ClientSession) return (-1, 0, _noFields(), _gone());
  try {
    final List<h2.Header> wire = _fields(headers, <h2.Header>[
      h2.Header.ascii(':method', method.toDartString()),
      h2.Header.ascii(':path', path.toDartString()),
      h2.Header.ascii(':scheme', 'https'),
      h2.Header.ascii(':authority', authority.toDartString()),
    ]);
    final h2.ClientTransportStream stream = held.conn.makeRequest(
      wire,
      endStream: !hasBody,
    );
    if (hasBody) stream.sendData(_h2Bytes(body), endStream: true);
    final _H2Stream entry = _H2Stream(stream);
    $await(
      entry.messages.moveNext().then((bool more) {
        if (!more) {
          return (
            -1,
            0,
            _noFields(),
            GoString.fromString('http2: the stream ended before the headers'),
          );
        }
        final h2.StreamMessage msg = entry.messages.current;
        if (msg is! h2.HeadersStreamMessage) {
          return (
            -1,
            0,
            _noFields(),
            GoString.fromString(
              'http2: the stream gave data before the headers',
            ),
          );
        }
        int status = 0;
        for (final h2.Header h in msg.headers) {
          if (String.fromCharCodes(h.name) == ':status') {
            status = int.tryParse(String.fromCharCodes(h.value)) ?? 0;
          }
        }
        if (msg.endStream) entry.ended = true;
        return (_streamSlot(entry), status, _flat(msg.headers), _none);
      }),
      (Object cause) => (-1, 0, _noFields(), _text(cause)),
    );
    return (-1, 0, _noFields(), _none);
  } catch (cause) {
    return (-1, 0, _noFields(), _text(cause));
  }
}

int _takeH2(_H2Stream s, Slice<int> p) {
  final int n = s.rest.length < p.len ? s.rest.length : p.len;
  if (n != 0) p.$fill(p.off, p.off + n, s.rest, 0);
  s.rest = n == s.rest.length ? Uint8List(0) : Uint8List.sublistView(s.rest, n);
  return n;
}

/// _fill reads the next data message into the rest. It gives 0 for data, 1
/// for the end of the stream, and 2 for a message that a read jumps over.
Future<int> _fill(_H2Stream s) {
  final Future<int>? open = s.pending;
  if (open != null) return open;
  final Future<int> next = s.messages.moveNext().then((bool more) {
    s.pending = null;
    if (!more) {
      s.ended = true;
      return 1;
    }
    final h2.StreamMessage msg = s.messages.current;
    if (msg is! h2.DataStreamMessage) return 2;
    final List<int> bytes = msg.bytes;
    s.rest = bytes is Uint8List ? bytes : Uint8List.fromList(bytes);
    if (s.rest.isEmpty) return 2;
    return 0;
  });
  s.pending = next;
  return next;
}

Future<(int, GoString)> _read(_H2Stream s, Slice<int> p) async {
  while (true) {
    if (s.rest.isNotEmpty) return (_takeH2(s, p), _none);
    if (s.ended) return (0, _none);
    await _fill(s);
  }
}

/// Reads bytes of the body of a stream. A count of 0 with no cause means the
/// end of the stream.
(int, GoString) $h2Read(int stream, Slice<int> p) {
  final _H2Stream? s = _streamAt(stream);
  if (s == null) return (0, _gone());
  if (p.len == 0) return (0, _none);
  if (s.rest.isNotEmpty) return (_takeH2(s, p), _none);
  if (s.ended) return (0, _none);
  $await(_read(s, p), (Object cause) => (0, _text(cause)));
  return (0, _none);
}

/// Wraps an accepted TLS connection whose ALPN took h2, on the server side.
(int, GoString) $h2Server(int conn) {
  final Socket? socket = $takeSocket(conn);
  if (socket == null) return (-1, _gone());
  try {
    final h2.ServerTransportConnection session =
        h2.ServerTransportConnection.viaSocket(socket);
    return (_sessionSlot(_ServerSession(session)), _none);
  } catch (cause) {
    return (-1, _text(cause));
  }
}

/// Waits for the next stream of a client. The results are the handle of the
/// stream, the method, the path, the authority, the flat headers, and the
/// cause. The end of the connection gives the cause EOF.
(int, GoString, GoString, GoString, Slice<GoString>, GoString) $h2Next(
  int session,
) {
  final Object? held = session >= 0 && session < _sessions.length
      ? _sessions[session]
      : null;
  if (held is! _ServerSession) {
    return (-1, _none, _none, _none, _noFields(), _gone());
  }
  $await(_next(held), (Object cause) {
    return (-1, _none, _none, _none, _noFields(), _text(cause));
  });
  return (-1, _none, _none, _none, _noFields(), _none);
}

Future<(int, GoString, GoString, GoString, Slice<GoString>, GoString)> _next(
  _ServerSession s,
) async {
  if (!await s.incoming.moveNext()) {
    return (-1, _none, _none, _none, _noFields(), GoString.fromString('EOF'));
  }
  final _H2Stream entry = _H2Stream(s.incoming.current);
  if (!await entry.messages.moveNext()) {
    return (
      -1,
      _none,
      _none,
      _none,
      _noFields(),
      GoString.fromString('http2: the stream ended before the headers'),
    );
  }
  final h2.StreamMessage msg = entry.messages.current;
  if (msg is! h2.HeadersStreamMessage) {
    return (
      -1,
      _none,
      _none,
      _none,
      _noFields(),
      GoString.fromString('http2: the stream gave data before the headers'),
    );
  }
  String method = '';
  String path = '';
  String authority = '';
  for (final h2.Header h in msg.headers) {
    final String name = String.fromCharCodes(h.name);
    final String value = String.fromCharCodes(h.value);
    switch (name) {
      case ':method':
        method = value;
      case ':path':
        path = value;
      case ':authority':
        authority = value;
    }
  }
  if (msg.endStream) entry.ended = true;
  return (
    _streamSlot(entry),
    GoString.fromString(method),
    GoString.fromString(path),
    GoString.fromString(authority),
    _flat(msg.headers),
    _none,
  );
}

/// Writes the status and the fields of an answer. endStream marks an answer
/// with no body.
GoString $h2WriteHead(
  int stream,
  int status,
  Slice<GoString> headers,
  bool endStream,
) {
  final _H2Stream? s = _streamAt(stream);
  if (s == null) return _gone();
  try {
    s.stream.sendHeaders(
      _fields(headers, <h2.Header>[h2.Header.ascii(':status', '$status')]),
      endStream: endStream,
    );
    return _none;
  } catch (cause) {
    return _text(cause);
  }
}

/// Writes bytes of a body onto a stream.
(int, GoString) $h2Write(int stream, Slice<int> p) {
  final _H2Stream? s = _streamAt(stream);
  if (s == null) return (0, _gone());
  try {
    s.stream.sendData(_h2Bytes(p));
    return (p.len, _none);
  } catch (cause) {
    return (0, _text(cause));
  }
}

/// Closes a stream and releases unread incoming data.
GoString $h2End(int stream) {
  final _H2Stream? s = _streamAt(stream);
  if (s == null) return _none;
  _streams[stream] = null;
  try {
    s.stream.outgoingMessages.close();
  } catch (_) {
    // The head with endStream closed the sink already.
  }
  unawaited(s.messages.cancel());
  return _none;
}

/// Ends a session and its connection.
GoString $h2Close(int session) {
  final Object? held = session >= 0 && session < _sessions.length
      ? _sessions[session]
      : null;
  if (held == null) return _none;
  _sessions[session] = null;
  if (held is _ClientSession) unawaited(held.conn.finish());
  if (held is _ServerSession) unawaited(held.conn.finish());
  return _none;
}

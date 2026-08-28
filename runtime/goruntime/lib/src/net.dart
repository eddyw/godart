import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'sched.dart';
import 'slice.dart';
import 'string.dart';

/// One open connection.
///
/// The iterator takes 1 packet of the stream at a time. The rest holds the
/// bytes of a packet that a read did not take, because a Go read gives only
/// the count that the slice holds.
final class _Conn {
  final Socket socket;
  final StreamIterator<Uint8List> packets;
  Uint8List rest = Uint8List(0);

  /// True after the peer closed the write side. Each read then gives 0.
  bool ended = false;

  _Conn(this.socket) : packets = StreamIterator<Uint8List>(socket);
}

/// The open connections of the program, by handle.
///
/// A handle is an index into this list. A closed slot holds null, and a new
/// dial takes the first free slot.
final List<_Conn?> _open = <_Conn?>[];

final GoString _empty = GoString.empty;

/// _cause writes the message of a failure. Go reads such a text as the message
/// of an error value.
GoString _cause(Object e) {
  if (e is SocketException) {
    final String os = e.osError?.message ?? '';
    if (os.isNotEmpty) return GoString.fromString('${e.message}: $os');
    return GoString.fromString(e.message);
  }
  return GoString.fromString(e.toString());
}

int _slot(_Conn c) {
  for (int i = 0; i < _open.length; i++) {
    if (_open[i] == null) {
      _open[i] = c;
      return i;
    }
  }
  _open.add(c);
  return _open.length - 1;
}

_Conn? _at(int handle) {
  if (handle < 0 || handle >= _open.length) return null;
  return _open[handle];
}

GoString _closed() => GoString.fromString('use of closed network connection');

Uint8List _bytes(Slice<int> p) {
  final Uint8List out = Uint8List(p.len);
  for (int i = 0; i < p.len; i++) {
    out[i] = p.array[p.off + i] & 0xFF;
  }
  return out;
}

/// _take moves the held bytes of a connection into the slice and gives the
/// count. A Go read takes no more bytes than the slice holds.
int _take(_Conn c, Slice<int> p) {
  final int n = c.rest.length < p.len ? c.rest.length : p.len;
  for (int i = 0; i < n; i++) {
    p.array[p.off + i] = c.rest[i];
  }
  c.rest = n == c.rest.length ? Uint8List(0) : Uint8List.sublistView(c.rest, n);
  return n;
}

/// Opens a TCP connection to a host and a port, and gives its handle. The
/// second result holds the cause of a failure, and it is empty after a
/// success.
///
/// The call gives up control, thus the result of the call itself is dead. The
/// frame of the wait gives the value to the caller.
(int, GoString) $dialTCP(GoString host, int port) {
  $await(
    Socket.connect(
      host.toDartString(),
      port,
    ).then((Socket s) => (_slot(_Conn(s)), _empty)),
    (Object cause) => (-1, _cause(cause)),
  );
  return (-1, _empty);
}

/// Reads bytes into the slice and gives the count. A count of 0 with no cause
/// means the end of the stream.
(int, GoString) $connRead(int handle, Slice<int> p) {
  final _Conn? c = _at(handle);
  if (c == null) return (0, _closed());
  if (p.len == 0) return (0, _empty);
  if (c.rest.isNotEmpty) return (_take(c, p), _empty);
  if (c.ended) return (0, _empty);
  $await(
    c.packets.moveNext().then((bool more) {
      if (!more) {
        c.ended = true;
        return (0, _empty);
      }
      c.rest = c.packets.current;
      return (_take(c, p), _empty);
    }),
    (Object cause) => (0, _cause(cause)),
  );
  return (0, _empty);
}

/// Writes the bytes of the slice and gives the count.
///
/// The call waits for the flush of the socket, exactly as a Go write waits for
/// the operating system.
(int, GoString) $connWrite(int handle, Slice<int> p) {
  final _Conn? c = _at(handle);
  if (c == null) return (0, _closed());
  if (p.len == 0) return (0, _empty);
  c.socket.add(_bytes(p));
  $await(
    c.socket.flush().then((_) => (p.len, _empty)),
    (Object cause) => (0, _cause(cause)),
  );
  return (0, _empty);
}

/// Closes a connection in both directions.
GoString $connClose(int handle) {
  final _Conn? c = _at(handle);
  if (c == null) return _closed();
  _open[handle] = null;
  unawaited(c.packets.cancel());
  c.socket.destroy();
  return _empty;
}

// Copyright 2011 The Go Authors. All rights reserved.
// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'chan.dart';
import 'datagram_posix.dart';
import 'goint.dart';
import 'sched.dart';
import 'slice.dart';
import 'string.dart';

export 'interface_native.dart';

/// One open connection.
///
/// The rest holds the bytes that a read did not take, because a Go read gives
/// only the count that the slice holds.
///
/// The class holds 2 forms. A plain TCP connection and a connection of the
/// Unix domain each ride a RawSocket, thus each 1 reaches the half close of
/// the system. A TLS connection rides a Socket, because the secure socket of
/// Dart owns the record layer and it drains the socket under it.
abstract class _Conn {
  Uint8List rest = Uint8List(0);

  /// True after the receive direction ended. Each read then gives 0.
  bool ended = false;

  /// True after this side ended the write side. Each write then gives a cause.
  bool writeEnded = false;

  /// True after this side ended the read side.
  bool readEnded = false;

  /// The read that stands open. A deadline ends the wait of the caller, and
  /// the read of the socket stays, thus the next read takes the same future.
  Future<int>? pending;
  Future<void> reads = Future<void>.value();

  /// The 2 deadlines, in nanoseconds after the start of the Unix epoch. A
  /// value of 0 holds no limit.
  final _Deadline readDeadline = _Deadline();
  final _Deadline writeDeadline = _Deadline();

  /// The local address, the local port, the address of the peer, and the port
  /// of the peer.
  /// Dart can discard these values after close. Go retains them.
  final InternetAddress local;
  final int localPort;
  final InternetAddress remote;
  final int remotePort;

  _Conn(this.local, this.localPort, this.remote, this.remotePort);

  /// fill reads the next bytes into the rest. It gives 0 for bytes and 1 for
  /// the end of the receive direction.
  Future<int> fill();

  /// send writes each byte and gives the count.
  Future<int> send(Uint8List bytes);

  /// endWrite ends the send direction. The peer then reads the end of the
  /// stream, and this side still reads.
  void endWrite();

  /// endRead ends the receive direction. Each read of this side then gives the
  /// end of the stream, and the system drops each byte that arrives after it.
  ///
  /// A raw socket holds the operation, and a Socket of Dart holds none, thus
  /// the result names the cause on a form that cannot reach it.
  String endRead();

  /// destroy ends the 2 directions and frees the socket.
  void destroy();

  /// option writes 1 socket option.
  void setFlag(SocketOption option, bool on);
  void setRaw(RawSocketOption option);
}

final class _WriteFailure implements Exception {
  final int count;
  final Object cause;
  const _WriteFailure(this.count, this.cause);
}

/// A connection over a Socket of dart:io. The TLS connections take this form.
///
/// The iterator takes 1 packet of the stream at a time.
final class _StreamConn extends _Conn {
  final Socket socket;
  final StreamIterator<Uint8List> packets;

  _StreamConn(this.socket)
    : packets = StreamIterator<Uint8List>(socket),
      super(
        socket.address,
        socket.port,
        socket.remoteAddress,
        socket.remotePort,
      );

  @override
  Future<int> fill() => packets.moveNext().then((bool more) {
    if (!more) {
      ended = true;
      return 1;
    }
    rest = packets.current;
    return 0;
  });

  @override
  Future<int> send(Uint8List bytes) {
    socket.add(bytes);
    return socket.flush().then((_) => bytes.length);
  }

  @override
  void endWrite() {
    socket.close().ignore();
  }

  @override
  String endRead() =>
      'close: the secure socket of Dart holds no half close of the read side';

  @override
  void destroy() {
    unawaited(packets.cancel());
    socket.destroy();
  }

  @override
  void setFlag(SocketOption option, bool on) => socket.setOption(option, on);
  @override
  void setRaw(RawSocketOption option) => socket.setRawOption(option);
}

/// A connection over a RawSocket of dart:io.
///
/// The class drives the byte stream itself. A read enables the read event and
/// waits for it, thus the socket holds the bytes that no read took and the
/// system applies the back pressure of TCP. A write repeats until the socket
/// took each byte, because 1 write of a raw socket can take a part alone.
final class _RawConn extends _Conn {
  final RawSocket socket;
  late final StreamSubscription<RawSocketEvent> events;

  /// The read that waits for the next event of the socket.
  Completer<int>? reader;

  /// The write that waits for room in the buffer of the socket.
  Completer<void>? writer;
  Future<void> writes = Future<void>.value();

  /// True after a close or a failure took the socket. A write then reports the
  /// cause, because a write of a socket that no longer stands takes no byte
  /// and the loop of send would never end.
  bool gone = false;

  _RawConn(this.socket)
    : super(
        socket.address,
        socket.port,
        socket.remoteAddress,
        socket.remotePort,
      ) {
    socket.readEventsEnabled = false;
    socket.writeEventsEnabled = false;
    events = socket.listen(
      _event,
      onDone: _end,
      onError: (Object cause, StackTrace trace) => _fail(cause),
    );
  }

  void _event(RawSocketEvent e) {
    if (e == RawSocketEvent.read) {
      _wakeRead();
      return;
    }
    if (e == RawSocketEvent.write) {
      final Completer<void>? w = writer;
      writer = null;
      w?.complete();
      return;
    }
    if (e == RawSocketEvent.readClosed || e == RawSocketEvent.closed) {
      _end();
    }
  }

  void _wakeRead() {
    final Completer<int>? w = reader;
    if (w == null) {
      socket.readEventsEnabled = false;
      return;
    }
    final Uint8List? got = _drain();
    if (got == null) return;
    reader = null;
    socket.readEventsEnabled = false;
    rest = got;
    w.complete(0);
  }

  /// _drain takes each byte that the socket holds, and it gives null for a
  /// socket that holds none.
  ///
  /// The count stands in the call, because a read with no count asserts that
  /// the socket holds at least 1 byte. A build with the assertions on then
  /// reports the failure `Read failed` in the place of an empty answer.
  Uint8List? _drain() {
    final int have = socket.available();
    if (have <= 0) return null;
    final Uint8List? got = socket.read(have);
    if (got == null || got.isEmpty) return null;
    return got;
  }

  void _end() {
    ended = true;
    final Completer<int>? w = reader;
    if (w == null) return;
    reader = null;
    w.complete(1);
  }

  void _fail(Object cause) {
    ended = true;
    gone = true;
    final Completer<int>? w = reader;
    if (w != null) {
      reader = null;
      w.completeError(cause);
    }
    final Completer<void>? p = writer;
    if (p != null) {
      writer = null;
      p.completeError(cause);
    }
  }

  @override
  Future<int> fill() {
    // The socket can hold bytes that no event announced, because the read
    // event stands off while no read waits.
    final Uint8List? got = _drain();
    if (got != null) {
      rest = got;
      return Future<int>.value(0);
    }
    if (ended) return Future<int>.value(1);
    final Completer<int> done = Completer<int>();
    reader = done;
    socket.readEventsEnabled = true;
    return done.future;
  }

  @override
  Future<int> send(Uint8List bytes) {
    final next = writes.then((_) => _send(bytes));
    // A failed write must release the next writer.
    writes = next.then<void>((_) {}, onError: (Object _) {});
    return next;
  }

  Future<int> _send(Uint8List bytes) async {
    int off = 0;
    try {
      while (true) {
        if (gone) {
          throw const SocketException('use of closed network connection');
        }
        if (writeDeadline.expired) throw const SocketException('i/o timeout');
        if (off == bytes.length) return off;
        final int n = socket.write(bytes, off, bytes.length - off);
        off += n;
        if (off >= bytes.length) break;
        final Completer<void> room = Completer<void>();
        writer = room;
        socket.writeEventsEnabled = true;
        final ready = await writeDeadline.limit<bool>(
          room.future.then((_) => true),
          false,
        );
        if (!ready) {
          writer = null;
          socket.writeEventsEnabled = false;
          throw const SocketException('i/o timeout');
        }
      }
      return off;
    } catch (cause) {
      throw _WriteFailure(off, cause);
    }
  }

  @override
  void endWrite() {
    socket.shutdown(SocketDirection.send);
  }

  @override
  String endRead() {
    socket.shutdown(SocketDirection.receive);
    return '';
  }

  /// destroy ends the socket, and it wakes each operation that waits.
  ///
  /// A goroutine that waits for a read or for room holds the frame of that
  /// wait. A close that woke no such frame would hold the goroutine for ever,
  /// exactly as a cancel of a request of net/http showed.
  @override
  void destroy() {
    gone = true;
    ended = true;
    unawaited(events.cancel());
    unawaited(socket.close());
    final Completer<int>? r = reader;
    if (r != null) {
      reader = null;
      r.complete(1);
    }
    final Completer<void>? w = writer;
    if (w != null) {
      writer = null;
      w.complete();
    }
  }

  @override
  void setFlag(SocketOption option, bool on) => socket.setOption(option, on);
  @override
  void setRaw(RawSocketOption option) => socket.setRawOption(option);
}

/// The open connections of the program, by handle.
///
/// A handle is an index into this list. A closed slot holds null, and a new
/// dial takes the first free slot.
final List<_Conn?> _open = <_Conn?>[];

/// One listener.
///
/// The iterator takes 1 connection of the stream at a time. A cancel of the
/// iterator closes the server socket, and a pending accept then gives no
/// connection.
///
/// A plain listener gives a RawSocket and a TLS listener gives a SecureSocket,
/// thus the element type of the stream is Object and each accept casts it.
final class _Listener {
  final StreamIterator<Object> incoming;
  Future<void> ready = Future<void>.value();
  Future<Object?>? pending;
  final deadline = _Deadline();
  bool closed = false;

  /// close ends the server socket. A plain listener and a TLS listener hold
  /// 2 server classes with no shared base, thus the class keeps a function.
  final Future<void> Function() close;

  final String tcpHost;

  _Listener(Stream<Object> stream, this.close, [this.tcpHost = ''])
    : incoming = StreamIterator<Object>(stream);

  Future<Object?> next() {
    final result = ready.then((_) async {
      if (closed) return null;
      pending ??= incoming.moveNext().then(
        (available) => available ? incoming.current : null,
      );
      Object? socket;
      try {
        socket = await deadline.limit(pending!, _acceptExpired);
      } catch (_) {
        pending = null;
        rethrow;
      }
      // A timeout retains the source accept for the next caller.
      if (identical(socket, _acceptExpired)) return socket;
      pending = null;
      return closed ? null : socket;
    });
    // A failed accept must release the next waiter.
    ready = result.then<void>(
      (_) {},
      onError: (Object error, StackTrace stack) {},
    );
    return result;
  }

  Future<void> stop() {
    closed = true;
    final source = pending;
    if (source != null) {
      unawaited(
        source.then<void>((socket) {
          if (socket is RawSocket) socket.close();
          if (socket is Socket) socket.destroy();
        }, onError: (Object error, StackTrace stack) {}),
      );
    }
    unawaited(incoming.cancel());
    return close();
  }
}

final Object _acceptExpired = Object();

/// The open listeners of the program, by handle. The handles of a listener, of
/// a connection, and of a datagram socket belong to 3 separate lists.
final List<_Listener?> _listeners = <_Listener?>[];

/// One datagram socket.
///
/// The socket gives a stream of events, and a read event holds each datagram
/// that arrived. The held list keeps a datagram that no read took, thus a
/// deadline of a read drops nothing.
final class _Packet {
  final RawDatagramSocket socket;
  final List<Datagram> held = <Datagram>[];

  // The Dart socket owns this descriptor.
  int nativeFD = -1;
  String localHost;
  final closed = Completer<void>();

  /// The host is empty when the native socket has no peer.
  String remoteHost = '';
  int remotePort = 0;

  bool ended = false;
  Object? failure;
  Future<void> reads = Future<void>.value();
  Completer<int>? waiter;
  Future<int>? pending;
  final _Deadline readDeadline = _Deadline();
  final _Deadline writeDeadline = _Deadline();

  _Packet(this.socket) : localHost = socket.address.address {
    socket.listen(
      _event,
      onDone: () {
        _end();
        closed.complete();
      },
      onError: _fail,
    );
  }

  void _event(RawSocketEvent event) {
    if (event == RawSocketEvent.closed) {
      _end();
      return;
    }
    if (event != RawSocketEvent.read) return;
    try {
      for (Datagram? d = _receive(); d != null; d = _receive()) {
        if (remoteHost.isNotEmpty &&
            (d.address.address != remoteHost || d.port != remotePort)) {
          continue;
        }
        held.add(d);
      }
    } catch (cause) {
      _fail(cause);
    }
    if (held.isNotEmpty) _wake(0);
  }

  Datagram? _receive() {
    if (Platform.isLinux || Platform.isMacOS) {
      final empty = receiveEmptyDatagram(socket);
      if (empty != null) return empty;
    }
    return socket.receive();
  }

  void _fail(Object cause) {
    final waiting = waiter;
    if (waiting == null) {
      failure = cause;
    } else {
      waiter = null;
      waiting.completeError(cause);
    }
  }

  void _end() {
    ended = true;
    _wake(1);
  }

  void _wake(int code) {
    final Completer<int>? w = waiter;
    if (w == null) return;
    waiter = null;
    w.complete(code);
  }
}

/// The open datagram sockets of the program, by handle.
final List<_Packet?> _packets = <_Packet?>[];

final GoString _empty = GoString.empty;

int _socketErrno(Object cause) {
  if (cause is OSError) return cause.errorCode > 0 ? cause.errorCode : 0;
  if (cause is! SocketException) return 0;
  final code = cause.osError?.errorCode ?? 0;
  return code > 0 ? code : 0;
}

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

/// True after the release of the network stands in the list of the scheduler.
bool _released = false;

/// _register asks the scheduler to close each open handle when package main
/// returns. Go ends the process at that point, and an open socket of Dart
/// keeps the isolate alive.
void _register() {
  if (_released) return;
  _released = true;
  $releases.add(_closeAll);
}

void _closeAll() {
  for (int i = 0; i < _open.length; i++) {
    final _Conn? c = _open[i];
    if (c == null) continue;
    _open[i] = null;
    c.destroy();
  }
  for (int i = 0; i < _listeners.length; i++) {
    final _Listener? l = _listeners[i];
    if (l == null) continue;
    _listeners[i] = null;
    unawaited(l.stop());
  }
  for (int i = 0; i < _packets.length; i++) {
    final _Packet? s = _packets[i];
    if (s == null) continue;
    _packets[i] = null;
    s.socket.close();
  }
}

int _slot(_Conn c) {
  _register();
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

/// Takes the socket of a connection out of the table. The HTTP/2 transport
/// wraps the socket, and the handle then answers no read and no write. The
/// take works only while no read ran, because the first read subscribes the
/// iterator of the connection to the stream.
///
/// Only a TLS connection reaches the HTTP/2 session, and such a connection
/// holds a Socket. A plain connection holds a raw socket and gives null.
Socket? $takeSocket(int handle) {
  final _Conn? c = _at(handle);
  if (c is! _StreamConn) return null;
  _open[handle] = null;
  return c.socket;
}

int _listenSlot(_Listener l) {
  _register();
  for (int i = 0; i < _listeners.length; i++) {
    if (_listeners[i] == null) {
      _listeners[i] = l;
      return i;
    }
  }
  _listeners.add(l);
  return _listeners.length - 1;
}

_Listener? _listenerAt(int handle) {
  if (handle < 0 || handle >= _listeners.length) return null;
  return _listeners[handle];
}

int _packetSlot(_Packet s) {
  _register();
  for (int i = 0; i < _packets.length; i++) {
    if (_packets[i] == null) {
      _packets[i] = s;
      return i;
    }
  }
  _packets.add(s);
  return _packets.length - 1;
}

_Packet? _packetAt(int handle) {
  if (handle < 0 || handle >= _packets.length) return null;
  return _packets[handle];
}

GoString _closed() => GoString.fromString('use of closed network connection');

GoString _timeout() => GoString.fromString('i/o timeout');

// SetDeadline must also change operations that already await the socket.
final class _Deadline {
  int nanos = 0;
  bool get expired => nanos != 0 && nanos <= $ix($nowNanos());

  final Set<void Function()> waits = <void Function()>{};

  void set(int value) {
    nanos = value;
    for (final reset in waits) {
      reset();
    }
  }

  Future<T> limit<T>(Future<T> source, T expired) {
    final done = Completer<T>();
    Timer? timer;
    void reset() {
      timer?.cancel();
      if (done.isCompleted || nanos == 0) return;
      final left = nanos - $ix($nowNanos());
      if (left <= 0) {
        done.complete(expired);
      } else {
        timer = Timer(Duration(microseconds: (left + 999) ~/ 1000), reset);
      }
    }

    waits.add(reset);
    reset();
    source.then(
      (value) {
        if (!done.isCompleted) done.complete(value);
      },
      onError: (Object error, StackTrace stack) {
        if (!done.isCompleted) done.completeError(error, stack);
      },
    );
    return done.future.whenComplete(() {
      timer?.cancel();
      waits.remove(reset);
    });
  }
}

/// _lowerFirst writes the first letter of a text in the lower case. The
/// message of the system starts with a capital letter, and the same message of
/// Go starts with a small letter.
String _lowerFirst(String text) {
  if (text.isEmpty) return text;
  return text[0].toLowerCase() + text.substring(1);
}

/// _connectCause writes the message of a failure of a connect.
///
/// Go names the operation of the system and the message of the errno, thus a
/// refusal reads `connect: connection refused`. The 2 tables of the message
/// hold the same text, and only the first letter differs.
GoString _connectCause(Object e) {
  if (e is SocketException) {
    final String os = e.osError?.message ?? '';
    if (os.isNotEmpty) {
      return GoString.fromString('connect: ${_lowerFirst(os)}');
    }
  }
  return _cause(e);
}

/// The codes of the resolver that Go names with a text of its own. The
/// platforms hold 2 tables: BSD counts up, Linux counts down, and Windows
/// holds a table of its own.
///
/// EAI_NONAME and EAI_NODATA each name a host that the resolver did not find,
/// and EAI_AGAIN names a failure that a second try may pass.
const int _eaiNoNameBSD = 8;
const int _eaiNoDataBSD = 7;
const int _eaiAgainBSD = 2;
const int _eaiNoNameLinux = -2;
const int _eaiNoDataLinux = -5;
const int _eaiAgainLinux = -3;
const int _hostNotFoundWindows = 11001;
const int _tryAgainWindows = 11002;

/// _lookupCause classifies a failure of the resolver of the system. Go gives
/// the text `no such host` for a name that the resolver did not find, and the
/// text `server misbehaving` for a failure that a second try may pass.
String _lookupCause(Object e) {
  if (e is SocketException) {
    final OSError? os = e.osError;
    if (os != null) {
      final int code = os.errorCode;
      if (code == _eaiNoNameBSD ||
          code == _eaiNoDataBSD ||
          code == _eaiNoNameLinux ||
          code == _eaiNoDataLinux ||
          code == _hostNotFoundWindows) {
        return 'no such host';
      }
      if (code == _eaiAgainBSD ||
          code == _eaiAgainLinux ||
          code == _tryAgainWindows) {
        return 'server misbehaving';
      }
      if (os.message.isNotEmpty) return _lowerFirst(os.message);
    }
    return e.message;
  }
  return e.toString();
}

/// The type of a lookup of 1 family. Go names the network ip4, ip6, or ip,
/// and the overlay carries 4, 6, and 0.
InternetAddressType _familyOf(int family) {
  if (family == 4) return InternetAddressType.IPv4;
  if (family == 6) return InternetAddressType.IPv6;
  return InternetAddressType.any;
}

/// Reads each address of a host name through the resolver of the system.
///
/// family holds 4 for IPv4, 6 for IPv6, and 0 for each family. The result
/// holds the cause of a failure on the first line, and 1 address on each line
/// after it. An empty first line means a success.
///
/// The resolver of the system orders the addresses with RFC 6724, exactly as
/// the resolver of Go does. That order reads the source address of each
/// destination, and dart:io opens no connected datagram socket, thus no code
/// of the overlay can compute it.
///
/// The call gives up control, thus the result of the call itself is dead. The
/// frame of the wait gives the value to the caller.
String $lookupAddrs(String host, int family) {
  $await(
    InternetAddress.lookup(host, type: _familyOf(family)).then((
      List<InternetAddress> found,
    ) {
      final StringBuffer out = StringBuffer();
      for (final InternetAddress a in found) {
        out.write('\n');
        out.write(a.address);
      }
      return GoString.fromString(out.toString());
    }),
    (Object cause) => GoString.fromString(_lookupCause(cause)),
  );
  return '';
}

/// _address reads a numeric address, or it asks the system for the name.
Future<InternetAddress> _address(String host) async {
  final InternetAddress? found = InternetAddress.tryParse(host);
  if (found != null) return found;
  final List<InternetAddress> names = await InternetAddress.lookup(host);
  if (names.isEmpty) throw SocketException('no such host: $host');
  return names.first;
}

/// The bytes of a slice, as a list of Dart.
///
/// A byte slice holds its storage in a Uint8List, thus setRange moves the whole
/// range in 1 call. A loop over the bytes reads and writes 1 element at a time,
/// and a write of 1 megabyte then costs a million bounds checks. A store into a
/// Uint8List holds the low 8 bits, exactly as the mask of the loop did.
Uint8List _bytes(Slice<int> p) {
  final Uint8List out = Uint8List(p.len);
  if (p.len != 0) out.setRange(0, p.len, p.array, p.off);
  return out;
}

/// _take moves the held bytes of a connection into the slice and gives the
/// count. A Go read takes no more bytes than the slice holds.
int _take(_Conn c, Slice<int> p) {
  final int n = c.rest.length < p.len ? c.rest.length : p.len;
  if (n != 0) p.$fill(p.off, p.off + n, c.rest, 0);
  c.rest = n == c.rest.length ? Uint8List(0) : Uint8List.sublistView(c.rest, n);
  return n;
}

/// _fill reads the next bytes of a connection into its rest. It gives 0 for
/// bytes and 1 for the end of the stream.
///
/// The socket holds 1 open read, thus the connection keeps the future. A read
/// that a deadline ended leaves that future open, and the next read takes it
/// again.
Future<int> _fill(_Conn c) {
  final Future<int>? open = c.pending;
  if (open != null) return open;
  final Future<int> next = c.fill().then((int code) {
    c.pending = null;
    return code;
  });
  c.pending = next;
  return next;
}

/// Opens a TCP connection to a host and a port, and gives its handle. The
/// second result holds the cause of a failure, and it is empty after a
/// success.
///
/// The call gives up control, thus the result of the call itself is dead. The
/// frame of the wait gives the value to the caller.
(int, GoString, int, GoString) $dialTCP(
  GoString host,
  int port,
  GoString localHost,
  int localPort,
) {
  $await(
    RawSocket.connect(
      host.toDartString(),
      port,
      sourceAddress: localHost.length == 0 ? null : localHost.toDartString(),
      sourcePort: localPort,
    ).then((RawSocket s) => (_slot(_RawConn(s)), _empty, 0, _empty)),
    (Object cause) => (
      -1,
      _connectCause(cause),
      _socketErrno(cause),
      GoString.fromString(
        cause is SocketException && cause.message == 'Bind failed'
            ? 'bind'
            : 'connect',
      ),
    ),
  );
  return (-1, _empty, 0, _empty);
}

/// Opens a TLS connection to a host and a port, and gives its handle.
///
/// The handle joins the table of the connections, thus each read, each write,
/// each deadline, and the close of a TCP connection also work on it.
///
/// insecure drops the check of the certificate. sni names the host of the check
/// and of the extension of the name of the server. protos holds the list of
/// ALPN, and the second result names the protocol that the handshake took.
(int, GoString, GoString) $dialTLS(
  GoString host,
  int port,
  bool insecure,
  GoString sni,
  Slice<GoString> protos,
) {
  final String name = sni.length == 0
      ? host.toDartString()
      : sni.toDartString();
  final List<String> alpn = <String>[];
  for (int i = 0; i < protos.len; i++) {
    alpn.add(protos[i].toDartString());
  }
  $await(
    Socket.connect(host.toDartString(), port)
        .then(
          (Socket s) => SecureSocket.secure(
            s,
            host: name,
            supportedProtocols: alpn.isEmpty ? null : alpn,
            onBadCertificate: insecure ? (X509Certificate c) => true : null,
          ),
        )
        .then(
          (SecureSocket s) => (
            _slot(_StreamConn(s)),
            GoString.fromString(s.selectedProtocol ?? ''),
            _empty,
          ),
        ),
    (Object cause) => (-1, GoString.empty, _cause(cause)),
  );
  return (-1, GoString.empty, _empty);
}

/// Opens a TLS connection through an HTTP proxy and gives its handle.
///
/// head holds the whole CONNECT request, thus the Go side names the target and
/// the authorization. The proxy answers with a status line and a header block,
/// and the handshake then runs on the same socket.
(int, GoString, GoString) $dialTLSProxy(
  GoString host,
  int port,
  GoString head,
  bool insecure,
  GoString sni,
  Slice<GoString> protos,
) {
  final List<String> alpn = <String>[];
  for (int i = 0; i < protos.len; i++) {
    alpn.add(protos[i].toDartString());
  }
  $await(
    _tunnel(
      host.toDartString(),
      port,
      head.toDartString(),
      sni.toDartString(),
      insecure,
      alpn,
    ),
    (Object cause) => (-1, GoString.empty, _cause(cause)),
  );
  return (-1, GoString.empty, _empty);
}

Future<(int, GoString, GoString)> _tunnel(
  String host,
  int port,
  String head,
  String name,
  bool insecure,
  List<String> alpn,
) async {
  final Socket raw = await Socket.connect(host, port);
  raw.add(utf8.encode(head));
  await raw.flush();
  final BytesBuilder held = BytesBuilder();
  final Completer<void> ready = Completer<void>();
  // RawSecureSocket.secure takes over the subscription of the socket, and it
  // rejects a subscription that stands paused. Thus the handler stays live, and
  // the length test below reports a proxy that wrote too much.
  raw.listen(
    (Uint8List chunk) {
      held.add(chunk);
      if (!ready.isCompleted && _headEnd(held.toBytes()) >= 0) ready.complete();
    },
    onDone: () {
      if (!ready.isCompleted) ready.complete();
    },
    onError: (Object cause) {
      if (!ready.isCompleted) ready.completeError(cause);
    },
  );
  await ready.future;
  final Uint8List answer = held.toBytes();
  final int end = _headEnd(answer);
  if (end < 0) {
    throw const SocketException('the proxy wrote no answer to CONNECT');
  }
  if (end != answer.length) {
    throw const SocketException(
      'the proxy wrote bytes after the answer to CONNECT',
    );
  }
  final String line = _statusLine(answer);
  if (!line.contains(' 200 ') && !line.endsWith(' 200')) {
    throw SocketException('the proxy answered CONNECT with $line');
  }
  final SecureSocket s = await SecureSocket.secure(
    raw,
    host: name,
    supportedProtocols: alpn.isEmpty ? null : alpn,
    onBadCertificate: insecure ? (X509Certificate c) => true : null,
  );
  return (
    _slot(_StreamConn(s)),
    GoString.fromString(s.selectedProtocol ?? ''),
    _empty,
  );
}

/// _headEnd gives the index after the blank line of a header block, and it
/// gives -1 for a block that no blank line ends.
int _headEnd(Uint8List b) {
  for (int i = 0; i + 3 < b.length; i++) {
    if (b[i] == 13 && b[i + 1] == 10 && b[i + 2] == 13 && b[i + 3] == 10) {
      return i + 4;
    }
  }
  return -1;
}

String _statusLine(Uint8List b) {
  int end = 0;
  while (end < b.length && b[end] != 13) {
    end++;
  }
  return ascii.decode(b.sublist(0, end), allowInvalid: true);
}

/// Opens a connection to a Unix socket and gives its handle.
(int, GoString) $dialUnix(GoString path) {
  $await(
    RawSocket.connect(
      InternetAddress(path.toDartString(), type: InternetAddressType.unix),
      0,
    ).then((RawSocket s) => (_slot(_RawConn(s)), _empty)),
    (Object cause) => (-1, _connectCause(cause)),
  );
  return (-1, _empty);
}

/// Reads bytes into the slice and gives the count, error text, and native error code.
/// A non-empty read with no bytes and no error means the end of the stream.
(int, GoString, int) $connRead(int handle, Slice<int> p) {
  final _Conn? c = _at(handle);
  if (c == null) return (0, _closed(), 0);
  final next = c.reads.then((_) => _readConn(handle, c, p));
  // A failed read must release the next reader.
  c.reads = next.then<void>((_) {}, onError: (Object _) {});
  $await(next, (Object cause) => (0, _cause(cause), _socketErrno(cause)));
  return (0, _empty, 0);
}

Future<(int, GoString, int)> _readConn(int handle, _Conn c, Slice<int> p) {
  if (_at(handle) != c) return Future.value((0, _closed(), 0));
  if (p.len == 0) return Future.value((0, _empty, 0));
  if (c.readDeadline.expired) return Future.value((0, _timeout(), 0));
  if (c.rest.isNotEmpty) return Future.value((_take(c, p), _empty, 0));
  if (c.ended) return Future.value((0, _empty, 0));
  return c.readDeadline.limit<int>(_fill(c), 2).then((int code) {
    if (code == 2) return (0, _timeout(), 0);
    if (code == 1) return (0, _at(handle) != c ? _closed() : _empty, 0);
    return (_take(c, p), _empty, 0);
  });
}

/// Writes the bytes of the slice and gives the count.
///
/// The call waits for the flush of the socket, exactly as a Go write waits for
/// the operating system.
(int, GoString, int) $connWrite(int handle, Slice<int> p) {
  final _Conn? c = _at(handle);
  if (c == null) return (0, _closed(), 0);
  if (c.writeDeadline.expired) return (0, _timeout(), 0);
  if (c.writeEnded) return (0, GoString.fromString('broken pipe'), 0);
  if (p.len == 0 && c is! _RawConn) return (0, _empty, 0);
  final next = c.send(_bytes(p)).then((int n) => (n, _empty, 0));
  $await(
    c is _RawConn ? next : c.writeDeadline.limit(next, (0, _timeout(), 0)),
    (Object cause) {
      if (cause is _WriteFailure) {
        return (cause.count, _cause(cause.cause), _socketErrno(cause.cause));
      }
      return (0, _cause(cause), _socketErrno(cause));
    },
  );
  return (0, _empty, 0);
}

/// Closes a connection in both directions.
/// Gives the local address, the local port, the peer address, and the peer
/// port of a connection. A handle that no connection holds gives a cause.
(GoString, int, GoString, int, GoString) $connAddrs(int handle) {
  final _Conn? c = _at(handle);
  if (c == null) {
    return (
      GoString.empty,
      0,
      GoString.empty,
      0,
      GoString.fromString('use of closed network connection'),
    );
  }
  return (
    GoString.fromString(c.local.address),
    c.localPort,
    GoString.fromString(c.remote.address),
    c.remotePort,
    GoString.empty,
  );
}

/// The number of a socket option of the system.
///
/// The 2 platform families give 2 numbers for 1 option, thus the table names
/// each 1. macOS, iOS, and Windows each keep the numbers of BSD, and Linux and
/// Android each keep the numbers of Linux. A platform outside the table gives
/// an error, because a wrong number writes another option with no report.
int? _sockOptionNumber(String name) {
  final bool bsd = Platform.isMacOS || Platform.isIOS || Platform.isWindows;
  final bool linux = Platform.isLinux || Platform.isAndroid;
  if (!bsd && !linux) return null;
  switch (name) {
    case 'keepalive':
      return bsd ? 0x0008 : 9;
    case 'rcvbuf':
      return bsd ? 0x1002 : 8;
    case 'sndbuf':
      return bsd ? 0x1001 : 7;
    case 'linger':
      return bsd ? 0x0080 : 13;
  }
  return null;
}

int? _tcpOptionNumber(String name) {
  if (Platform.isLinux || Platform.isAndroid) {
    return switch (name) {
      'keepidle' => 4,
      'keepinterval' => 5,
      'keepcount' => 6,
      _ => null,
    };
  }
  if (Platform.isMacOS || Platform.isIOS) {
    return switch (name) {
      'keepidle' => 0x10,
      'keepinterval' => 0x101,
      'keepcount' => 0x102,
      _ => null,
    };
  }
  if (Platform.isWindows) {
    return switch (name) {
      'keepidle' => 3,
      'keepinterval' => 17,
      'keepcount' => 16,
      _ => null,
    };
  }
  return null;
}

GoString _optionCause(String name) => GoString.fromString(
  'setsockopt: the option $name holds no number on ${Platform.operatingSystem}',
);

/// Writes a socket option of a connection. The name holds `nodelay`,
/// `keepalive`, `keepidle`, `keepinterval`, `keepcount`, `rcvbuf`, or
/// `sndbuf`, or `linger`. A flag reads the value 0 or 1.
(GoString, int) $connOption(int handle, GoString name, int value) {
  final _Conn? c = _at(handle);
  if (c == null) return (_closed(), 0);
  final String key = name.toDartString();
  try {
    if (key == 'nodelay') {
      c.setFlag(SocketOption.tcpNoDelay, value != 0);
      return (_empty, 0);
    }
    final int? tcpOption = _tcpOptionNumber(key);
    final int? option = tcpOption ?? _sockOptionNumber(key);
    if (option == null) return (_optionCause(key), 0);
    if (key == 'linger') {
      final data = ByteData(Platform.isWindows ? 4 : 8);
      if (Platform.isWindows) {
        data.setUint16(0, value >= 0 ? 1 : 0, Endian.host);
        data.setUint16(2, value >= 0 ? value.toUnsigned(16) : 0, Endian.host);
      } else {
        data.setInt32(0, value >= 0 ? 1 : 0, Endian.host);
        data.setInt32(4, value >= 0 ? value.toSigned(32) : 0, Endian.host);
      }
      c.setRaw(
        RawSocketOption(
          RawSocketOption.levelSocket,
          option,
          data.buffer.asUint8List(),
        ),
      );
      return (_empty, 0);
    }
    c.setRaw(
      RawSocketOption.fromInt(
        tcpOption == null
            ? RawSocketOption.levelSocket
            : RawSocketOption.levelTcp,
        option,
        value,
      ),
    );
    return (_empty, 0);
  } on Object catch (cause) {
    return (GoString.fromString('setsockopt: $cause'), _socketErrno(cause));
  }
}

/// Writes a socket option of a datagram socket. The name holds `rcvbuf` or
/// `sndbuf`.
(GoString, int) $packetOption(int handle, GoString name, int value) {
  final _Packet? p = _packetAt(handle);
  if (p == null) return (_closed(), 0);
  final String key = name.toDartString();
  final int? option = _sockOptionNumber(key);
  if (option == null) return (_optionCause(key), 0);
  try {
    p.socket.setRawOption(
      RawSocketOption.fromInt(RawSocketOption.levelSocket, option, value),
    );
    return (_empty, 0);
  } on Object catch (cause) {
    return (GoString.fromString('setsockopt: $cause'), _socketErrno(cause));
  }
}

GoString $connClose(int handle) {
  final _Conn? c = _at(handle);
  if (c == null) return _closed();
  _open[handle] = null;
  c.destroy();
  return _empty;
}

/// Ends the write side of a connection, and gives an empty text or the cause.
/// The peer then reads the end of the stream, and this side still reads.
///
/// A raw socket writes the shutdown of the send direction. A Socket of Dart
/// reaches the same operation through close, which shuts that direction down
/// after the flush of each pending byte. Each write of this runtime waits for
/// its own flush, thus the close holds no pending byte.
///
/// The result is a Dart String, because an external declaration of the overlay
/// reads it.
String $connCloseWrite(int handle) {
  final _Conn? c = _at(handle);
  if (c == null) return 'use of closed network connection';
  if (!c.writeEnded) {
    c.writeEnded = true;
    c.endWrite();
  }
  return '';
}

/// Ends the read side of a connection, and gives an empty text or the cause.
/// Each read of this side then gives the end of the stream.
///
/// RawSocket.shutdown of dart:io writes the shutdown of the receive direction.
/// The system then drops each byte that arrives after it, and the peer of a
/// BSD system reads a reset for the bytes that it wrote. Go holds the same
/// behavior, because CloseRead of Go writes the same call of the system.
String $connCloseRead(int handle) {
  final _Conn? c = _at(handle);
  if (c == null) return 'use of closed network connection';
  if (c.readEnded) return '';
  final String cause = c.endRead();
  if (cause != '') return cause;
  c.readEnded = true;
  c.ended = true;
  c.rest = Uint8List(0);
  return '';
}

/// Writes the 2 deadlines of a connection, in nanoseconds after the start of
/// the Unix epoch. A value of 0 holds no limit.
GoString $connDeadline(int handle, int read, int write) {
  final _Conn? c = _at(handle);
  if (c == null) return _closed();
  c.readDeadline.set(read);
  c.writeDeadline.set(write);
  return _empty;
}

/// Gives the listener handle, bound port, and failure cause.
/// A port of 0 asks the system for a free port.
/// The call yields control. The synchronous return value is unused.
(int, int, GoString) $listenTCP(GoString host, int port, bool ipv6Only) {
  $await(
    _bindTCP(host.toDartString(), port, ipv6Only).then(
      (RawServerSocket s) => (
        _listenSlot(_Listener(s, () => s.close(), s.address.address)),
        s.port,
        _empty,
      ),
    ),
    (Object cause) => (-1, 0, _cause(cause)),
  );
  return (-1, 0, _empty);
}

Future<RawServerSocket> _bindTCP(String host, int port, bool ipv6Only) async {
  if (host.isNotEmpty) {
    return RawServerSocket.bind(host, port, v6Only: ipv6Only);
  }
  try {
    return await RawServerSocket.bind(InternetAddress.anyIPv6, port);
  } on SocketException catch (error) {
    final code = error.osError?.errorCode;
    final unavailable = Platform.isMacOS
        ? code == 47 || code == 49
        : code == 97 || code == 99;
    if (!unavailable) rethrow;
    return RawServerSocket.bind(InternetAddress.anyIPv4, port);
  }
}

GoString $listenTCPHost(int handle) =>
    GoString.fromString(_listenerAt(handle)?.tcpHost ?? '');

/// Opens a TLS listener on a host and a port, and gives its handle and the
/// port that the system took. The certificate chain and the key each hold the
/// PEM form, and the list of alpn holds each protocol that the server offers.
///
/// The call gives up control, thus the result of the call itself is dead.
(int, int, GoString) $listenTLS(
  GoString host,
  int port,
  Slice<int> cert,
  Slice<int> key,
  Slice<GoString> alpn,
) {
  final SecurityContext context = SecurityContext();
  try {
    context.useCertificateChainBytes(_bytes(cert));
    context.usePrivateKeyBytes(_bytes(key));
    final List<String> protocols = <String>[
      for (int i = 0; i < alpn.len; i++)
        alpn.array[alpn.off + i].toDartString(),
    ];
    if (protocols.isNotEmpty) context.setAlpnProtocols(protocols, true);
  } catch (e) {
    return (-1, 0, _cause(e));
  }
  $await(
    SecureServerSocket.bind(host.toDartString(), port, context).then(
      (SecureServerSocket s) =>
          (_listenSlot(_Listener(s, () => s.close())), s.port, _empty),
    ),
    (Object cause) => (-1, 0, _cause(cause)),
  );
  return (-1, 0, _empty);
}

/// Waits for 1 connection of a TLS listener, and gives its handle and the
/// protocol that ALPN took.
(int, GoString, GoString) $acceptTLS(int handle) {
  final _Listener? l = _listenerAt(handle);
  if (l == null) return (-1, _empty, _closed());
  $await(
    l.next().then((Object? socket) {
      if (socket == null) return (-1, _empty, _closed());
      if (identical(socket, _acceptExpired)) return (-1, _empty, _timeout());
      final Socket s = socket as Socket;
      final String proto = s is SecureSocket ? s.selectedProtocol ?? '' : '';
      return (_slot(_StreamConn(s)), GoString.fromString(proto), _empty);
    }),
    (Object cause) => (-1, _empty, _cause(cause)),
  );
  return (-1, _empty, _empty);
}

/// Opens a listener on a Unix socket and gives its handle.
(int, GoString) $listenUnix(GoString path) {
  final String name = path.toDartString();
  $await(
    RawServerSocket.bind(
      InternetAddress(name, type: InternetAddressType.unix),
      0,
    ).then(
      (RawServerSocket s) =>
          (_listenSlot(_Listener(s, () => s.close())), _empty),
    ),
    (Object cause) => (-1, _cause(cause)),
  );
  return (-1, _empty);
}

/// Waits for 1 connection of a listener and gives its handle.
(int, GoString) $accept(int handle) {
  final _Listener? l = _listenerAt(handle);
  if (l == null) return (-1, _closed());
  $await(
    l.next().then((Object? socket) {
      if (socket == null) return (-1, _closed());
      if (identical(socket, _acceptExpired)) return (-1, _timeout());
      return (_slot(_RawConn(socket as RawSocket)), _empty);
    }),
    (Object cause) => (-1, _cause(cause)),
  );
  return (-1, _empty);
}

GoString $listenDeadline(int handle, int nanos) {
  final listener = _listenerAt(handle);
  if (listener == null) return _closed();
  listener.deadline.set(nanos);
  return _empty;
}

/// Closes a listener. Each connection of it stays open, exactly as in Go.
///
/// A listener of a Unix socket also removes the file of the socket. Go removes
/// it at the close of a listener that Listen opened.
GoString $listenClose(int handle) {
  final _Listener? l = _listenerAt(handle);
  if (l == null) return _closed();
  _listeners[handle] = null;
  // The iterator of a stream subscribes at the first accept. A listener that
  // accepted nothing holds no subscription, thus the cancel closes no server
  // socket and the isolate stays alive.
  // Dart removes a Unix socket path before the close future completes.
  // Return after that removal to preserve a subsequent path replacement.
  $await(l.stop().then((_) => _empty), (Object cause) => _cause(cause));
  return _empty;
}

/// Opens a datagram socket on a host and a port, and gives its handle and the
/// port that the system took.
(int, int, GoString, int) $listenUDP(GoString host, int port) {
  $await(
    _bindUDP(
      host.toDartString(),
      port,
    ).then((_Packet s) => (_packetSlot(s), s.socket.port, _empty, 0)),
    (Object cause) => (-1, 0, _cause(cause), _socketErrno(cause)),
  );
  return (-1, 0, _empty, 0);
}

/// Joins a multicast group on a datagram socket. Index 0 lets
/// the system select the interface, as a nil Interface does in Go.
///
/// The call blocks the goroutine, thus the caller becomes an async function.
GoString $joinMulticast(int handle, GoString group, int index) {
  final _Packet? p = _packetAt(handle);
  if (p == null) return _closed();
  $await(
    _joinGroup(p, group.toDartString(), index).then((_) => _empty),
    (Object cause) => _cause(cause),
  );
  return _empty;
}

Future<void> _joinGroup(_Packet p, String group, int index) async {
  NetworkInterface? found;
  if (index != 0) {
    for (final NetworkInterface n in await NetworkInterface.list(
      includeLoopback: true,
      includeLinkLocal: true,
    )) {
      if (n.index == index) {
        found = n;
        break;
      }
    }
    if (found == null) {
      throw 'the interface $index is not present';
    }
  }
  final groupAddress = await _address(group);
  if (found != null && groupAddress.type == InternetAddressType.IPv4) {
    final Uint8List optionValue;
    if (Platform.isLinux) {
      optionValue = Uint8List(12);
      ByteData.sublistView(optionValue).setInt32(8, found.index, Endian.host);
    } else {
      InternetAddress? ipv4;
      for (final address in found.addresses) {
        if (address.type == InternetAddressType.IPv4) {
          ipv4 = address;
          break;
        }
      }
      if (ipv4 == null) throw 'no such network interface';
      optionValue = ipv4.rawAddress;
    }
    p.socket.setRawOption(
      RawSocketOption(
        RawSocketOption.levelIPv4,
        RawSocketOption.IPv4MulticastInterface,
        optionValue,
      ),
    );
  }
  p.socket.multicastLoopback = false;
  p.socket.joinMulticast(groupAddress, found);
}

Future<_Packet> _bindUDP(String host, int port) async {
  final address = await _address(host);
  if (port < 0 || port > 65535) {
    throw const OSError('invalid argument', 22);
  }
  return _Packet(
    await RawDatagramSocket.bind(address, port, reuseAddress: false),
  );
}

/// Opens a datagram socket that shares its port with another socket of the
/// system. A multicast listen of Go takes such a socket.
(int, int, GoString) $listenMulticastUDP(GoString host, int port) {
  $await(
    _bindShared(
      host.toDartString(),
      port,
    ).then((_Packet s) => (_packetSlot(s), s.socket.port, _empty)),
    (Object cause) => (-1, 0, _cause(cause)),
  );
  return (-1, 0, _empty);
}

Future<_Packet> _bindShared(String host, int port) async {
  return _Packet(
    await RawDatagramSocket.bind(
      await _address(host),
      port,
      reuseAddress: true,
      reusePort: true,
    ),
  );
}

/// Connects a bound datagram socket and reports its actual addresses.
(GoString, GoString, GoString, int) $connectUDP(
  int handle,
  GoString host,
  int port,
) {
  $await(
    _connectUDP(_packetAt(handle)!, host.toDartString(), port),
    (Object cause) => (_empty, _empty, _cause(cause), _socketErrno(cause)),
  );
  return (_empty, _empty, _empty, 0);
}

Future<(GoString, GoString, GoString, int)> _connectUDP(
  _Packet s,
  String host,
  int port,
) async {
  final target = await _address(host);
  if (port < 0 || port > 65535) {
    throw const OSError('invalid argument', 22);
  }
  s.remoteHost = target.address;
  if (Platform.isLinux || Platform.isMacOS) {
    final local = connectDatagram(s.socket, target, port);
    s.nativeFD = local.$1;
    s.localHost = local.$2;
    s.remoteHost = local.$3;
  }
  s.remotePort = port;
  return (
    GoString.fromString(s.localHost),
    GoString.fromString(s.remoteHost),
    _empty,
    0,
  );
}

/// _fillPacket waits for 1 datagram. It gives 0 for a datagram and 1 for the
/// end of the socket.
Future<int> _fillPacket(_Packet s) {
  if (s.held.isNotEmpty) return Future<int>.value(0);
  final failure = s.failure;
  if (failure != null) {
    s.failure = null;
    return Future<int>.error(failure);
  }
  if (s.ended) return Future<int>.value(1);
  final Future<int>? open = s.pending;
  if (open != null) return open;
  final Completer<int> done = Completer<int>();
  s.waiter = done;
  final Future<int> next = done.future.whenComplete(() {
    s.pending = null;
  });
  s.pending = next;
  return next;
}

/// Reads 1 datagram into the slice. The results are the count, the host of the
/// source, the port of the source, and the cause of a failure.
/// The last result preserves an available native error code.
///
/// A datagram that holds more bytes than the slice loses the rest, exactly as
/// a Go read of a datagram does.
(int, GoString, int, GoString, int) $packetRead(int handle, Slice<int> p) {
  final _Packet? s = _packetAt(handle);
  if (s == null) return (0, _empty, 0, _closed(), 0);
  final next = s.reads.then((_) => _readPacket(s, p));
  // A failed read must release the next reader.
  s.reads = next.then<void>((_) {}, onError: (Object _) {});
  $await(
    next,
    (Object cause) => (0, _empty, 0, _cause(cause), _socketErrno(cause)),
  );
  return (0, _empty, 0, _empty, 0);
}

Future<(int, GoString, int, GoString, int)> _readPacket(
  _Packet s,
  Slice<int> p,
) {
  if (s.ended) return Future.value((0, _empty, 0, _closed(), 0));
  if (s.readDeadline.expired) {
    return Future.value((0, _empty, 0, _timeout(), 0));
  }
  Timer? errors;
  if (Platform.isLinux || Platform.isMacOS) {
    // Dart can omit read events, empty datagrams, and remote UDP errors.
    errors = Timer.periodic(const Duration(milliseconds: 10), (_) {
      if (s.waiter == null) return;
      s._event(RawSocketEvent.read);
      final waiter = s.waiter;
      if (waiter == null || s.nativeFD < 0) return;
      try {
        final code = datagramPendingError(s.socket);
        if (code != 0) {
          s.waiter = null;
          waiter.completeError(datagramError(code));
        }
      } catch (error, stack) {
        s.waiter = null;
        waiter.completeError(error, stack);
      }
    });
  }
  return s.readDeadline
      .limit<int>(_fillPacket(s), 2)
      .whenComplete(() {
        errors?.cancel();
      })
      .then((int code) {
        if (code == 2) return (0, _empty, 0, _timeout(), 0);
        if (code == 1 && s.held.isEmpty) return (0, _empty, 0, _closed(), 0);
        final Datagram d = s.held.removeAt(0);
        final int n = d.data.length < p.len ? d.data.length : p.len;
        if (n != 0) p.$fill(p.off, p.off + n, d.data, 0);
        return (n, GoString.fromString(d.address.address), d.port, _empty, 0);
      });
}

/// Writes the bytes of the slice to a host and a port, and gives the count.
(int, GoString, int) $packetWrite(
  int handle,
  Slice<int> p,
  GoString host,
  int port,
) {
  final _Packet? s = _packetAt(handle);
  if (s == null) return (0, _closed(), 0);
  if (s.writeDeadline.expired) return (0, _timeout(), 0);
  $await(
    s.writeDeadline.limit<(int, GoString, int)>(
      _sendTo(s, _bytes(p), host.toDartString(), port),
      (0, _timeout(), 0),
    ),
    (Object cause) => (0, _cause(cause), _socketErrno(cause)),
  );
  return (0, _empty, 0);
}

Future<(int, GoString, int)> _sendTo(
  _Packet s,
  Uint8List bytes,
  String host,
  int port,
) async {
  final int n;
  if (s.nativeFD >= 0) {
    n = sendDatagram(s.nativeFD, bytes);
  } else if (host.isEmpty) {
    if (!Platform.isLinux && !Platform.isMacOS) {
      return (0, GoString.fromString('destination address required'), 0);
    }
    n = sendUnconnectedDatagram(s.socket, bytes);
  } else {
    final address = await _address(host);
    if (port < 0 || port > 65535) {
      throw const OSError('invalid argument', 22);
    }
    n = Platform.isLinux || Platform.isMacOS
        ? sendDatagramTo(s.socket, bytes, address, port)
        : s.socket.send(bytes, address, port);
  }
  if (n == 0 && bytes.isNotEmpty) {
    return (0, GoString.fromString('the send buffer of the socket is full'), 0);
  }
  return (n, _empty, 0);
}

/// Closes a datagram socket.
GoString $packetClose(int handle) {
  final _Packet? s = _packetAt(handle);
  if (s == null) return _closed();
  _packets[handle] = null;
  s._end();
  s.socket.close();
  $await(s.closed.future.then((_) => _empty), (Object cause) => _cause(cause));
  return _empty;
}

/// Writes the 2 deadlines of a datagram socket.
GoString $packetDeadline(int handle, int read, int write) {
  final _Packet? s = _packetAt(handle);
  if (s == null) return _closed();
  s.readDeadline.set(read);
  s.writeDeadline.set(write);
  return _empty;
}

/// Gives the network interfaces of the host.
///
/// The first result holds 1 item for each interface, in the form
/// `<index>|<name>`. The second result holds 1 item for each address, in the
/// form `<index>|<address>/<prefix>`. The third result holds the cause of a
/// failure.
///
/// The list holds the loopback interface and each link local address, because
/// Interfaces of Go holds them too.
///
/// The call gives up control, thus the result of the call itself is dead.
(Slice<GoString>, Slice<GoString>, GoString) $interfaces() {
  $await(
    NetworkInterface.list(
      includeLoopback: true,
      includeLinkLocal: true,
    ).then(_interfaceLists),
    (Object cause) => (
      Slice<GoString>.of(<GoString>[]),
      Slice<GoString>.of(<GoString>[]),
      _cause(cause),
    ),
  );
  return (
    Slice<GoString>.of(<GoString>[]),
    Slice<GoString>.of(<GoString>[]),
    _empty,
  );
}

/// _interfaceLists writes the 2 flat lists of the interfaces.
(Slice<GoString>, Slice<GoString>, GoString) _interfaceLists(
  List<NetworkInterface> found,
) {
  final List<GoString> names = <GoString>[];
  final List<GoString> addrs = <GoString>[];
  for (final NetworkInterface n in found) {
    names.add(GoString.fromString('${n.index}|${n.name}'));
    for (final InterfaceAddress a in n.addresses) {
      addrs.add(
        GoString.fromString('${n.index}|${a.address}/${a.prefixLength}'),
      );
    }
  }
  return (Slice<GoString>.of(names), Slice<GoString>.of(addrs), _empty);
}

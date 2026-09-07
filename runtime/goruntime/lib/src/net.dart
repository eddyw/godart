import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'chan.dart';
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

  /// True after this side closed the write side. Each write then gives a
  /// cause, because the sink of the socket rejects a late byte with a throw.
  bool writeEnded = false;

  /// The read that stands open. A deadline ends the wait of the caller, and
  /// the read of the stream stays, thus the next read takes the same future.
  Future<int>? pending;

  /// The 2 deadlines, in nanoseconds after the start of the Unix epoch. A
  /// value of 0 holds no limit.
  int readDeadline = 0;
  int writeDeadline = 0;

  _Conn(this.socket) : packets = StreamIterator<Uint8List>(socket);
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
final class _Listener {
  final StreamIterator<Socket> incoming;

  /// close ends the server socket. A plain listener and a TLS listener hold
  /// 2 server classes with no shared base, thus the class keeps a function.
  final Future<void> Function() close;

  /// The path of a Unix socket. A close removes the file, exactly as Go does.
  /// The value is empty for a TCP listener.
  final String unixPath;

  _Listener(Stream<Socket> stream, this.close, [this.unixPath = ''])
    : incoming = StreamIterator<Socket>(stream);
}

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

  /// The remote address of a socket of a dial. Such a socket drops a datagram
  /// of another source, exactly as a connected socket of the system does. The
  /// host is empty for a socket of a listen.
  String remoteHost = '';
  int remotePort = 0;

  bool ended = false;
  Completer<int>? waiter;
  Future<int>? pending;
  int readDeadline = 0;
  int writeDeadline = 0;

  _Packet(this.socket) {
    socket.listen(_event, onDone: _end, onError: (Object cause) => _end());
  }

  void _event(RawSocketEvent event) {
    if (event == RawSocketEvent.closed) {
      _end();
      return;
    }
    if (event != RawSocketEvent.read) return;
    for (Datagram? d = socket.receive(); d != null; d = socket.receive()) {
      if (remoteHost.isNotEmpty &&
          (d.address.address != remoteHost || d.port != remotePort)) {
        continue;
      }
      held.add(d);
    }
    if (held.isNotEmpty) _wake(0);
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
    unawaited(c.packets.cancel());
    c.socket.destroy();
  }
  for (int i = 0; i < _listeners.length; i++) {
    final _Listener? l = _listeners[i];
    if (l == null) continue;
    _listeners[i] = null;
    unawaited(l.incoming.cancel());
    unawaited(l.close());
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
Socket? $takeSocket(int handle) {
  final _Conn? c = _at(handle);
  if (c == null) return null;
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

/// _limit ends a wait at a deadline. A deadline of 0 holds no limit, and a
/// deadline that passed already gives the expired value at once.
///
/// The source future stays open. Each caller holds it in a field, thus the
/// next call of the same operation takes the same future.
Future<T> _limit<T>(Future<T> f, int deadline, T expired) {
  if (deadline == 0) return f;
  final int left = deadline - $nowNanos();
  if (left <= 0) return Future<T>.value(expired);
  return f.timeout(
    Duration(microseconds: left ~/ 1000),
    onTimeout: () => expired,
  );
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

/// _fill reads the next packet of a connection into its rest. It gives 0 for a
/// packet and 1 for the end of the stream.
///
/// The iterator of a stream holds 1 open read, thus the connection keeps the
/// future. A read that a deadline ended leaves that future open, and the next
/// read takes it again.
Future<int> _fill(_Conn c) {
  final Future<int>? open = c.pending;
  if (open != null) return open;
  final Future<int> next = c.packets.moveNext().then((bool more) {
    c.pending = null;
    if (!more) {
      c.ended = true;
      return 1;
    }
    c.rest = c.packets.current;
    return 0;
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
            _slot(_Conn(s)),
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
    _slot(_Conn(s)),
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
    Socket.connect(
      InternetAddress(path.toDartString(), type: InternetAddressType.unix),
      0,
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
    _limit<int>(_fill(c), c.readDeadline, 2).then((int code) {
      if (code == 2) return (0, _timeout());
      if (code == 1) return (0, _empty);
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
  if (c.writeEnded) return (0, GoString.fromString('broken pipe'));
  if (p.len == 0) return (0, _empty);
  c.socket.add(_bytes(p));
  $await(
    _limit<(int, GoString)>(
      c.socket.flush().then((_) => (p.len, _empty)),
      c.writeDeadline,
      (0, _timeout()),
    ),
    (Object cause) => (0, _cause(cause)),
  );
  return (0, _empty);
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
    GoString.fromString(c.socket.address.address),
    c.socket.port,
    GoString.fromString(c.socket.remoteAddress.address),
    c.socket.remotePort,
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
  }
  return null;
}

GoString _optionCause(String name) => GoString.fromString(
  'setsockopt: the option $name holds no number on ${Platform.operatingSystem}',
);

/// Writes a socket option of a connection. The name holds `nodelay`,
/// `keepalive`, `rcvbuf`, or `sndbuf`. A flag reads the value 0 or 1.
GoString $connOption(int handle, GoString name, int value) {
  final _Conn? c = _at(handle);
  if (c == null) return _closed();
  final String key = name.toDartString();
  try {
    if (key == 'nodelay') {
      c.socket.setOption(SocketOption.tcpNoDelay, value != 0);
      return _empty;
    }
    final int? option = _sockOptionNumber(key);
    if (option == null) return _optionCause(key);
    c.socket.setRawOption(
      RawSocketOption.fromInt(RawSocketOption.levelSocket, option, value),
    );
    return _empty;
  } on Object catch (cause) {
    return GoString.fromString('setsockopt: $cause');
  }
}

/// Writes a socket option of a datagram socket. The name holds `rcvbuf` or
/// `sndbuf`.
GoString $packetOption(int handle, GoString name, int value) {
  final _Packet? p = _packetAt(handle);
  if (p == null) return _closed();
  final String key = name.toDartString();
  final int? option = _sockOptionNumber(key);
  if (option == null) return _optionCause(key);
  try {
    p.socket.setRawOption(
      RawSocketOption.fromInt(RawSocketOption.levelSocket, option, value),
    );
    return _empty;
  } on Object catch (cause) {
    return GoString.fromString('setsockopt: $cause');
  }
}

GoString $connClose(int handle) {
  final _Conn? c = _at(handle);
  if (c == null) return _closed();
  _open[handle] = null;
  unawaited(c.packets.cancel());
  c.socket.destroy();
  return _empty;
}

/// Ends the write side of a connection, and gives an empty text or the cause.
/// The peer then reads the end of the stream, and this side still reads.
///
/// Socket.close of dart:io shuts the send direction down after the flush of
/// each pending byte, and each write of this runtime waits for its own flush,
/// thus the close holds no pending byte. The result is a Dart String, because
/// an external declaration of the overlay reads it.
String $connCloseWrite(int handle) {
  final _Conn? c = _at(handle);
  if (c == null) return 'use of closed network connection';
  if (!c.writeEnded) {
    c.writeEnded = true;
    c.socket.close().ignore();
  }
  return '';
}

/// Writes the 2 deadlines of a connection, in nanoseconds after the start of
/// the Unix epoch. A value of 0 holds no limit.
GoString $connDeadline(int handle, int read, int write) {
  final _Conn? c = _at(handle);
  if (c == null) return _closed();
  c.readDeadline = read;
  c.writeDeadline = write;
  return _empty;
}

/// Opens a listener on a host and a port, and gives its handle and the port
/// that the system took. A port of 0 asks the system for a free port.
///
/// The call gives up control, thus the result of the call itself is dead.
(int, int, GoString) $listenTCP(GoString host, int port) {
  $await(
    ServerSocket.bind(host.toDartString(), port).then(
      (ServerSocket s) =>
          (_listenSlot(_Listener(s, () => s.close())), s.port, _empty),
    ),
    (Object cause) => (-1, 0, _cause(cause)),
  );
  return (-1, 0, _empty);
}

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
    l.incoming.moveNext().then((bool more) {
      if (!more) return (-1, _empty, _closed());
      final Socket s = l.incoming.current;
      final String proto = s is SecureSocket ? s.selectedProtocol ?? '' : '';
      return (_slot(_Conn(s)), GoString.fromString(proto), _empty);
    }),
    (Object cause) => (-1, _empty, _cause(cause)),
  );
  return (-1, _empty, _empty);
}

/// Opens a listener on a Unix socket and gives its handle.
(int, GoString) $listenUnix(GoString path) {
  final String name = path.toDartString();
  $await(
    ServerSocket.bind(
      InternetAddress(name, type: InternetAddressType.unix),
      0,
    ).then(
      (ServerSocket s) =>
          (_listenSlot(_Listener(s, () => s.close(), name)), _empty),
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
    l.incoming.moveNext().then((bool more) {
      if (!more) return (-1, _closed());
      return (_slot(_Conn(l.incoming.current)), _empty);
    }),
    (Object cause) => (-1, _cause(cause)),
  );
  return (-1, _empty);
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
  unawaited(l.incoming.cancel());
  unawaited(l.close());
  if (l.unixPath.isNotEmpty) {
    try {
      File(l.unixPath).deleteSync();
    } on FileSystemException {
      // Go also leaves the file when the remove fails.
    }
  }
  return _empty;
}

/// Opens a datagram socket on a host and a port, and gives its handle and the
/// port that the system took.
(int, int, GoString) $listenUDP(GoString host, int port) {
  $await(
    _bindUDP(
      host.toDartString(),
      port,
    ).then((_Packet s) => (_packetSlot(s), s.socket.port, _empty)),
    (Object cause) => (-1, 0, _cause(cause)),
  );
  return (-1, 0, _empty);
}

/// Joins a multicast group on a datagram socket. An empty interface name reads
/// each interface of the system, exactly as a nil Interface does in Go.
///
/// The call blocks the goroutine, thus the caller becomes an async function.
GoString $joinMulticast(int handle, GoString group, GoString iface) {
  final _Packet? p = _packetAt(handle);
  if (p == null) return _closed();
  $await(
    _joinGroup(
      p,
      group.toDartString(),
      iface.toDartString(),
    ).then((_) => _empty),
    (Object cause) => _cause(cause),
  );
  return _empty;
}

Future<void> _joinGroup(_Packet p, String group, String name) async {
  NetworkInterface? found;
  if (name.isNotEmpty) {
    for (final NetworkInterface n in await NetworkInterface.list(
      includeLoopback: true,
      includeLinkLocal: true,
    )) {
      if (n.name == name) {
        found = n;
        break;
      }
    }
    if (found == null) {
      throw 'the interface $name is not present';
    }
  }
  p.socket.joinMulticast(await _address(group), found);
}

Future<_Packet> _bindUDP(String host, int port) async {
  return _Packet(await RawDatagramSocket.bind(await _address(host), port));
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

/// Opens a datagram socket that holds a remote address, and gives its handle
/// and the local port that the system took.
///
/// The socket drops a datagram of another source, exactly as a connected
/// socket of the system does.
(int, int, GoString) $dialUDP(GoString host, int port) {
  $await(
    _connectUDP(
      host.toDartString(),
      port,
    ).then((_Packet s) => (_packetSlot(s), s.socket.port, _empty)),
    (Object cause) => (-1, 0, _cause(cause)),
  );
  return (-1, 0, _empty);
}

Future<_Packet> _connectUDP(String host, int port) async {
  final InternetAddress target = await _address(host);
  final RawDatagramSocket socket = await RawDatagramSocket.bind(
    target.type == InternetAddressType.IPv6
        ? InternetAddress.anyIPv6
        : InternetAddress.anyIPv4,
    0,
  );
  final _Packet s = _Packet(socket);
  s.remoteHost = target.address;
  s.remotePort = port;
  return s;
}

/// _fillPacket waits for 1 datagram. It gives 0 for a datagram and 1 for the
/// end of the socket.
Future<int> _fillPacket(_Packet s) {
  if (s.held.isNotEmpty) return Future<int>.value(0);
  if (s.ended) return Future<int>.value(1);
  final Future<int>? open = s.pending;
  if (open != null) return open;
  final Completer<int> done = Completer<int>();
  s.waiter = done;
  final Future<int> next = done.future.then((int code) {
    s.pending = null;
    return code;
  });
  s.pending = next;
  return next;
}

/// Reads 1 datagram into the slice. The results are the count, the host of the
/// source, the port of the source, and the cause of a failure.
///
/// A datagram that holds more bytes than the slice loses the rest, exactly as
/// a Go read of a datagram does.
(int, GoString, int, GoString) $packetRead(int handle, Slice<int> p) {
  final _Packet? s = _packetAt(handle);
  if (s == null) return (0, _empty, 0, _closed());
  $await(
    _limit<int>(_fillPacket(s), s.readDeadline, 2).then((int code) {
      if (code == 2) return (0, _empty, 0, _timeout());
      if (code == 1 && s.held.isEmpty) return (0, _empty, 0, _closed());
      final Datagram d = s.held.removeAt(0);
      final int n = d.data.length < p.len ? d.data.length : p.len;
      if (n != 0) p.$fill(p.off, p.off + n, d.data, 0);
      return (n, GoString.fromString(d.address.address), d.port, _empty);
    }),
    (Object cause) => (0, _empty, 0, _cause(cause)),
  );
  return (0, _empty, 0, _empty);
}

/// Writes the bytes of the slice to a host and a port, and gives the count.
(int, GoString) $packetWrite(
  int handle,
  Slice<int> p,
  GoString host,
  int port,
) {
  final _Packet? s = _packetAt(handle);
  if (s == null) return (0, _closed());
  $await(
    _limit<(int, GoString)>(
      _sendTo(s, _bytes(p), host.toDartString(), port),
      s.writeDeadline,
      (0, _timeout()),
    ),
    (Object cause) => (0, _cause(cause)),
  );
  return (0, _empty);
}

Future<(int, GoString)> _sendTo(
  _Packet s,
  Uint8List bytes,
  String host,
  int port,
) async {
  final int n = s.socket.send(bytes, await _address(host), port);
  if (n == 0 && bytes.isNotEmpty) {
    return (0, GoString.fromString('the send buffer of the socket is full'));
  }
  return (n, _empty);
}

/// Closes a datagram socket.
GoString $packetClose(int handle) {
  final _Packet? s = _packetAt(handle);
  if (s == null) return _closed();
  _packets[handle] = null;
  s.socket.close();
  return _empty;
}

/// Writes the 2 deadlines of a datagram socket.
GoString $packetDeadline(int handle, int read, int write) {
  final _Packet? s = _packetAt(handle);
  if (s == null) return _closed();
  s.readDeadline = read;
  s.writeDeadline = write;
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

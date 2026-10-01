/// The web form of net.dart.
///
/// A web target holds no socket. This file still declares each name of
/// net.dart, because the Go package net names each 1, and net/http reads that
/// package. A program that only builds a URL would otherwise fail in the Dart
/// compile, and the whole web target would stay out of reach.
///
/// Each operation gives the cause "operation not supported". The Go side
/// wraps that text in a net.OpError, thus the program reads the same shape of
/// error that Go gives under GOOS=js.
library;

import 'goint.dart';
import 'slice.dart';
import 'string.dart';

/// The cause of each operation that the web target does not hold. Go gives
/// the same text for the errno ENOTSUP.
final GoString _noNet = GoString.fromString('operation not supported');

final GoString _empty = GoString.empty;

(int, GoString, int, GoString) $dialTCP(
  GoString host,
  int port,
  GoString localHost,
  int localPort,
) => (0, _noNet, 0, _empty);

/// Reads each address of a host name. A web target holds no resolver, thus the
/// first line names the cause and no address follows it.
String $lookupAddrs(String host, int family) => 'operation not supported';

(int, GoString, GoString) $dialTLS(
  GoString host,
  int port,
  bool insecure,
  GoString sni,
  Slice<GoString> protos,
) => (0, _empty, _noNet);

(int, GoString, GoString) $dialTLSProxy(
  GoString host,
  int port,
  GoString head,
  bool insecure,
  GoString sni,
  Slice<GoString> protos,
) => (0, _empty, _noNet);

(int, GoString) $dialUnix(GoString path) => (0, _noNet);

(int, GoString, int) $connRead(int handle, Slice<GoInt> p) => (0, _noNet, 0);

(int, GoString, int) $connWrite(int handle, Slice<GoInt> p) => (0, _noNet, 0);

(GoString, int, GoString, int, GoString) $connAddrs(int handle) =>
    (_empty, 0, _empty, 0, _noNet);

GoString $connClose(int handle) => _noNet;

String $connCloseWrite(int handle) => 'operation not supported';

String $connCloseRead(int handle) => 'operation not supported';

GoString $connDeadline(int handle, int read, int write) => _noNet;

(GoString, int) $connOption(int handle, GoString name, int value) =>
    (_noNet, 0);

(GoString, int) $packetOption(int handle, GoString name, int value) =>
    (_noNet, 0);

(int, int, GoString) $listenTCP(GoString host, int port, bool ipv6Only) =>
    (0, 0, _noNet);

GoString $listenTCPHost(int handle) => GoString.empty;

(int, int, GoString) $listenTLS(
  GoString host,
  int port,
  Slice<GoInt> cert,
  Slice<GoInt> key,
  Slice<GoString> alpn,
) => (0, 0, _noNet);

(int, GoString, GoString) $acceptTLS(int handle) => (0, _empty, _noNet);

(int, GoString) $listenUnix(GoString path) => (0, _noNet);

(int, GoString) $accept(int handle) => (0, _noNet);

GoString $listenClose(int handle) => _noNet;

GoString $listenDeadline(int handle, int nanos) => _noNet;

(int, int, GoString, int) $listenUDP(GoString host, int port) =>
    (0, 0, _noNet, 0);

(GoString, GoString, GoString, int) $connectUDP(
  int handle,
  GoString host,
  int port,
) => (_empty, _empty, _noNet, 0);

(int, int, GoString) $listenMulticastUDP(GoString host, int port) =>
    (0, 0, _noNet);

GoString $joinMulticast(int handle, GoString group, int index) => _noNet;

(int, GoString, int, GoString, int) $packetRead(int handle, Slice<GoInt> p) =>
    (0, _empty, 0, _noNet, 0);

(int, GoString, int) $packetWrite(
  int handle,
  Slice<GoInt> p,
  GoString host,
  int port,
) => (0, _noNet, 0);

GoString $packetClose(int handle) => _noNet;

GoString $packetDeadline(int handle, int read, int write) => _noNet;

/// Gives the network interfaces of the host.
///
/// A browser holds no such API. Go under GOOS=js also gives an empty table
/// and no cause. Thus this stub gives 2 empty lists and no cause. The Go side
/// then reports "no such network interface" for a name and for an index,
/// exactly as Go does.
(Slice<GoString>, Slice<GoString>, GoString) $interfaces() => (
  Slice<GoString>.of(<GoString>[]),
  Slice<GoString>.of(<GoString>[]),
  _empty,
);

(Slice<GoString>, GoString) $interfaceTable() =>
    (Slice<GoString>.nil(), _empty);

(Slice<GoString>, GoString) $interfaceMulticastAddrs(int index) =>
    (Slice<GoString>.nil(), _empty);

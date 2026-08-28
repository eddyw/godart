import 'dart:io';

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

/// The frame and the machine below hold the shape that stage 5 writes for a
/// function that can block. The machine dials a server, writes 4 bytes, reads
/// the answer, and closes the connection. Each of the first 3 operations waits
/// for a Dart future.
final class _EchoFrame implements Frame {
  final int port;
  final List<String> log;
  final Completer<void> done;
  int s = 0;
  bool c = false;
  Frame? r;
  int handle = -1;
  GoString cause = GoString.empty;
  int n = 0;
  Slice<int> buf = Slice.of(Uint8List(8));

  _EchoFrame(this.port, this.log, this.done);

  @override
  Object? resume() {
    _echo(port, log, done, this);
    return null;
  }
}

void _echo(int port, List<String> log, Completer<void> done, [_EchoFrame? f]) {
  int s = 0;
  bool c = false;
  Frame? r;
  int handle = -1;
  GoString cause = GoString.empty;
  int n = 0;
  Slice<int> buf = Slice.of(Uint8List(8));
  if (f != null) {
    s = f.s;
    c = f.c;
    r = f.r;
    handle = f.handle;
    cause = f.cause;
    n = f.n;
    buf = f.buf;
  }
  machine:
  while (true) {
    switch (s) {
      case 0:
        (handle, cause) = $dialTCP(GoString.fromString('127.0.0.1'), port);
        s = 1;
        continue machine;
      case 1:
        if (c) {
          c = false;
          (handle, cause) = r!.resume() as (int, GoString);
        }
        if ($blocked != null) break machine;
        log.add('dial cause=${cause.toDartString()}');
        (n, cause) = $connWrite(
          handle,
          $stringToBytes(GoString.fromString('ping')),
        );
        s = 2;
        continue machine;
      case 2:
        if (c) {
          c = false;
          (n, cause) = r!.resume() as (int, GoString);
        }
        if ($blocked != null) break machine;
        log.add('write n=$n cause=${cause.toDartString()}');
        (n, cause) = $connRead(handle, buf);
        s = 3;
        continue machine;
      case 3:
        if (c) {
          c = false;
          (n, cause) = r!.resume() as (int, GoString);
        }
        if ($blocked != null) break machine;
        final String text = String.fromCharCodes(buf.array.take(n));
        log.add('read text=$text cause=${cause.toDartString()}');
        log.add('close cause=${$connClose(handle).toDartString()}');
        done.complete();
        return;
    }
  }
  $block(
    _EchoFrame(port, log, done)
      ..s = s
      ..c = true
      ..r = $blocked
      ..handle = handle
      ..cause = cause
      ..n = n
      ..buf = buf,
  );
}

/// Starts a server that writes each packet back.
Future<ServerSocket> _startEcho() async {
  final ServerSocket server = await ServerSocket.bind(
    InternetAddress.loopbackIPv4,
    0,
  );
  server.listen((Socket socket) {
    socket.listen(socket.add, onDone: socket.destroy);
  });
  return server;
}

void main() {
  group('the network access', () {
    test('a dial, a write, and a read pass through a frame', () async {
      final ServerSocket server = await _startEcho();
      final List<String> log = <String>[];
      final Completer<void> done = Completer<void>();
      $go(() => _echo(server.port, log, done));
      $schedule();
      await done.future;
      await server.close();
      expect(log, <String>[
        'dial cause=',
        'write n=4 cause=',
        'read text=ping cause=',
        'close cause=',
      ]);
    });

    test('a dial to a closed port gives a cause', () async {
      final ServerSocket probe = await ServerSocket.bind(
        InternetAddress.loopbackIPv4,
        0,
      );
      final int port = probe.port;
      await probe.close();
      final List<String> log = <String>[];
      final Completer<void> done = Completer<void>();
      $go(() => _echo(port, log, done));
      $schedule();
      await done.future;
      expect(log[0], startsWith('dial cause='));
      expect(log[0], isNot('dial cause='));
      expect(log[1], 'write n=0 cause=use of closed network connection');
      expect(log[2], 'read text= cause=use of closed network connection');
      expect(log[3], 'close cause=use of closed network connection');
    });
  });
}

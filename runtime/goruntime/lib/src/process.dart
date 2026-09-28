import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'sched.dart';
import 'slice.dart';
import 'string.dart';

/// One pipe of a child process.
///
/// The iterator takes 1 chunk of the stream at a time. The rest holds the
/// bytes of a chunk that a read did not take, because a Go read gives only
/// the count that the slice holds.
final class _Pipe {
  final StreamIterator<List<int>> chunks;
  Uint8List rest = Uint8List(0);
  bool ended = false;
  Future<int>? pending;

  _Pipe(Stream<List<int>> stream) : chunks = StreamIterator<List<int>>(stream);
}

/// One child process, by process id. Go names a process by its id, thus the
/// id is the handle of each call.
final class _Proc {
  final Process process;
  final _Pipe out;
  final _Pipe err;

  _Proc(this.process)
    : out = _Pipe(process.stdout),
      err = _Pipe(process.stderr);
}

final Map<int, _Proc> _procs = <int, _Proc>{};

final GoString _empty = GoString.empty;

GoString _cause(Object e) {
  if (e is ProcessException) {
    return GoString.fromString('${e.executable}: ${e.message}');
  }
  return GoString.fromString(e.toString());
}

GoString _unknown(int pid) =>
    GoString.fromString('process $pid is not a child');

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

int _take(_Pipe pipe, Slice<int> p) {
  final int n = pipe.rest.length < p.len ? pipe.rest.length : p.len;
  if (n != 0) p.$fill(p.off, p.off + n, pipe.rest, 0);
  pipe.rest = n == pipe.rest.length
      ? Uint8List(0)
      : Uint8List.sublistView(pipe.rest, n);
  return n;
}

/// _fill reads the next chunk of a pipe into its rest. It gives 0 for a chunk
/// and 1 for the end of the stream.
Future<int> _fill(_Pipe pipe) {
  final Future<int>? open = pipe.pending;
  if (open != null) return open;
  final Future<int> next = pipe.chunks.moveNext().then((bool more) {
    pipe.pending = null;
    if (!more) {
      pipe.ended = true;
      return 1;
    }
    final List<int> chunk = pipe.chunks.current;
    pipe.rest = chunk is Uint8List ? chunk : Uint8List.fromList(chunk);
    return 0;
  });
  pipe.pending = next;
  return next;
}

/// Starts a process and gives its process id. The results are the id and the
/// cause of a failure. Go keeps a child alive when the program ends, thus the
/// table holds no release hook.
(int, GoString) $procStart(
  Slice<GoString> argv,
  GoString dir,
  Slice<GoString> env,
  bool useEnv,
) {
  final List<String> parts = <String>[];
  for (int i = 0; i < argv.len; i++) {
    parts.add(argv[i].toDartString());
  }
  Map<String, String>? environment;
  if (useEnv) {
    environment = <String, String>{};
    for (int i = 0; i < env.len; i++) {
      final String pair = env[i].toDartString();
      final int eq = pair.indexOf('=');
      if (eq <= 0) continue;
      environment[pair.substring(0, eq)] = pair.substring(eq + 1);
    }
  }
  $await(
    Process.start(
      parts.first,
      parts.sublist(1),
      workingDirectory: dir.length == 0 ? null : dir.toDartString(),
      environment: environment,
      includeParentEnvironment: !useEnv,
    ).then((Process p) {
      _procs[p.pid] = _Proc(p);
      return (p.pid, _empty);
    }),
    (Object cause) => (-1, _cause(cause)),
  );
  return (-1, _empty);
}

(int, GoString) $procWrite(int pid, Slice<int> p) {
  final _Proc? proc = _procs[pid];
  if (proc == null) return (0, _unknown(pid));
  if (p.len == 0) return (0, _empty);
  proc.process.stdin.add(_bytes(p));
  $await(
    proc.process.stdin.flush().then((_) => (p.len, _empty)),
    (Object cause) => (0, _cause(cause)),
  );
  return (0, _empty);
}

GoString $procCloseStdin(int pid) {
  final _Proc? proc = _procs[pid];
  if (proc == null) return _unknown(pid);
  $await(
    proc.process.stdin.close().then((_) => _empty),
    (Object cause) => _cause(cause),
  );
  return _empty;
}

(int, GoString) _pipeRead(_Pipe pipe, Slice<int> p) {
  if (p.len == 0) return (0, _empty);
  if (pipe.rest.isNotEmpty) return (_take(pipe, p), _empty);
  if (pipe.ended) return (0, _empty);
  $await(
    _fill(pipe).then((int code) {
      if (code == 1) return (0, _empty);
      return (_take(pipe, p), _empty);
    }),
    (Object cause) => (0, _cause(cause)),
  );
  return (0, _empty);
}

/// Closes the 2 output pipes of a process.
///
/// A child that ends can leave an orphan that still holds the pipes, and a read
/// of such a pipe never ends. This call cancels the 2 subscriptions, thus a
/// read that waits gives the end of the stream at once.
GoString $procCloseIO(int pid) {
  final _Proc? proc = _procs[pid];
  if (proc == null) return _unknown(pid);
  _closePipe(proc.out);
  _closePipe(proc.err);
  return _empty;
}

void _closePipe(_Pipe pipe) {
  if (pipe.ended) return;
  pipe.ended = true;
  pipe.rest = Uint8List(0);
  unawaited(pipe.chunks.cancel());
}

(int, GoString) $procRead(int pid, Slice<int> p) {
  final _Proc? proc = _procs[pid];
  if (proc == null) return (0, _unknown(pid));
  return _pipeRead(proc.out, p);
}

(int, GoString) $procReadErr(int pid, Slice<int> p) {
  final _Proc? proc = _procs[pid];
  if (proc == null) return (0, _unknown(pid));
  return _pipeRead(proc.err, p);
}

/// Waits for the end of a process and gives the exit code. A process that a
/// signal ended gives the negative number of the signal, as Dart does.
(int, GoString) $procWait(int pid) {
  final _Proc? proc = _procs[pid];
  if (proc == null) return (0, _unknown(pid));
  $await(
    proc.process.exitCode.then((int code) => (code, _empty)),
    (Object cause) => (0, _cause(cause)),
  );
  return (0, _empty);
}

GoString $procKill(int pid, int signal) {
  final _Proc? proc = _procs[pid];
  if (proc == null) return _unknown(pid);
  final ProcessSignal sig = switch (signal) {
    2 => ProcessSignal.sigint,
    9 => ProcessSignal.sigkill,
    15 => ProcessSignal.sigterm,
    28 => ProcessSignal.sigwinch,
    _ => ProcessSignal.sigkill,
  };
  if (!proc.process.kill(sig)) {
    return GoString.fromString('os: process already finished');
  }
  return _empty;
}

/// Sends a signal to any process by its id. $procKill reaches a child only,
/// and a program that signals itself needs this call.
GoString $killPid(int pid, int number) {
  final ProcessSignal? sig =
      _signalOf(number) ?? (number == 9 ? ProcessSignal.sigkill : null);
  if (sig == null) {
    return GoString.fromString('signal $number cannot be sent');
  }
  if (!Process.killPid(pid, sig)) {
    return GoString.fromString('os: process already finished');
  }
  return _empty;
}

/// One watch of an operating system signal.
///
/// The subscription starts at once, because `ProcessSignal.watch` of dart:io
/// installs the handler of the platform only when a listener joins. A watch
/// that waits for the first read of Go lets the default action of the platform
/// end the program.
final class _Watch {
  final int number;
  late final StreamSubscription<ProcessSignal> sub;

  /// The events that no read took yet. Go reads them 1 at a time.
  final List<int> queue = <int>[];
  Completer<int>? waiter;
  bool closed = false;

  _Watch(Stream<ProcessSignal> stream, this.number) {
    sub = stream.listen((ProcessSignal _) {
      final Completer<int>? w = waiter;
      if (w != null) {
        waiter = null;
        w.complete(number);
        return;
      }
      queue.add(number);
    });
  }

  void close() {
    if (closed) return;
    closed = true;
    unawaited(sub.cancel());
    final Completer<int>? w = waiter;
    if (w != null) {
      waiter = null;
      w.complete(0);
    }
  }
}

final Map<int, _Watch> _watches = <int, _Watch>{};
int _nextWatch = 1;
bool _watchRelease = false;

ProcessSignal? _signalOf(int number) => switch (number) {
  1 => ProcessSignal.sighup,
  2 => ProcessSignal.sigint,
  3 => ProcessSignal.sigquit,
  6 => ProcessSignal.sigabrt,
  14 => ProcessSignal.sigalrm,
  15 => ProcessSignal.sigterm,
  28 => ProcessSignal.sigwinch,
  10 when Platform.isLinux || Platform.isAndroid => ProcessSignal.sigusr1,
  12 when Platform.isLinux || Platform.isAndroid => ProcessSignal.sigusr2,
  30 when Platform.isMacOS || Platform.isIOS => ProcessSignal.sigusr1,
  31 when Platform.isMacOS || Platform.isIOS => ProcessSignal.sigusr2,
  _ => null,
};

void _closeWatches() {
  for (final _Watch w in _watches.values) {
    w.close();
  }
  _watches.clear();
}

/// Starts a watch of 1 signal and gives its handle.
(int, GoString) $signalWatch(int number) {
  final ProcessSignal? sig = _signalOf(number);
  if (sig == null) {
    return (0, GoString.fromString('signal $number cannot be caught'));
  }
  if (!_watchRelease) {
    // A live subscription keeps the isolate alive, thus the end of package
    // main must cancel each 1.
    _watchRelease = true;
    $releases.add(_closeWatches);
  }
  final int id = _nextWatch++;
  try {
    _watches[id] = _Watch(sig.watch(), number);
  } catch (e) {
    return (0, _cause(e));
  }
  return (id, _empty);
}

/// Waits for the next event of a watch and gives the signal number. A number
/// of 0 with no cause means that $signalStop closed the watch.
///
/// The call blocks the goroutine.
(int, GoString) $signalWait(int id) {
  final _Watch? watch = _watches[id];
  if (watch == null) return (0, _empty);
  if (watch.queue.isNotEmpty) return (watch.queue.removeAt(0), _empty);
  if (watch.closed) return (0, _empty);
  final Completer<int> next = Completer<int>();
  watch.waiter = next;
  $await(
    next.future.then((int n) => (n, _empty)),
    (Object cause) => (0, _cause(cause)),
  );
  return (0, _empty);
}

/// Stops a watch. The goroutine that waits on it wakes with a number of 0.
GoString $signalStop(int id) {
  _watches.remove(id)?.close();
  return _empty;
}

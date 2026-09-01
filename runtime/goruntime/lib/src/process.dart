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

Uint8List _bytes(Slice<int> p) {
  final Uint8List out = Uint8List(p.len);
  for (int i = 0; i < p.len; i++) {
    out[i] = p.array[p.off + i] & 0xFF;
  }
  return out;
}

int _take(_Pipe pipe, Slice<int> p) {
  final int n = pipe.rest.length < p.len ? pipe.rest.length : p.len;
  for (int i = 0; i < n; i++) {
    p.array[p.off + i] = pipe.rest[i];
  }
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
    _ => ProcessSignal.sigkill,
  };
  if (!proc.process.kill(sig)) {
    return GoString.fromString('os: process already finished');
  }
  return _empty;
}

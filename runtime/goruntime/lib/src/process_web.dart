/// The web form of process.dart.
///
/// A web target starts no process. This file still declares each name of
/// process.dart, because the Go packages os and os/exec name each 1. A
/// program that only reads an environment value would otherwise fail in the
/// Dart compile.
///
/// Each operation gives the cause "function not implemented", and the Go side
/// wraps that text in an error.
library;

import 'goint.dart';
import 'slice.dart';
import 'string.dart';

/// The cause of each operation that the web target does not hold. Go gives
/// the same text for the errno ENOSYS.
final GoString _noSys = GoString.fromString('function not implemented');

(int, int, GoString) $procStart(
  GoString executable,
  Slice<GoString> argv,
  GoString dir,
  Slice<GoString> env,
) => (0, 0, _noSys);

(int, GoString) $procWrite(int pid, Slice<GoInt> p) => (0, _noSys);

GoString $procCloseStdin(int pid) => _noSys;

GoString $procCloseIO(int pid) => _noSys;

GoString $procCloseOutput(int pid, bool errStream) => _noSys;

(int, GoString) $procRead(int pid, Slice<GoInt> p) => (0, _noSys);

(int, GoString) $procReadErr(int pid, Slice<GoInt> p) => (0, _noSys);

(int, bool, GoString) $procWait(int pid) => (0, false, _noSys);

GoString $killPid(int pid, int number) => _noSys;

(int, GoString) $signalWatch(int number) => (0, _noSys);

(int, GoString) $signalWait(int id) => (0, _noSys);

GoString $signalStop(int id) => _noSys;

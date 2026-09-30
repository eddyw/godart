import 'dart:js_interop';
import 'dart:js_interop_unsafe';

import 'slice.dart';

@JS('globalThis.process')
external JSObject? get _process;

int _identity(String method) {
  final process = _process;
  if (process == null || !process.has(method)) return -1;
  return process.callMethod<JSNumber>(method.toJS).toDartInt;
}

int $getuid() => _identity('getuid');
int $geteuid() => _identity('geteuid');
int $getgid() => _identity('getgid');
int $getegid() => _identity('getegid');
int $getppid() => 0;
int $getpgrp() => -1;
(int, int) $getpgid(int pid) => (-1, -1);
(int, int) $getsid(int pid) => (-1, -1);
int $getpagesize() => 4096;

(Slice<int>, int) $getgroups() => (Slice<int>.nil(), -1);

int $sendSignal(int pid, int signal) => -1;

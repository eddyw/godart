import 'panic.dart';
import 'platform_native.dart' if (dart.library.js_interop) 'platform_web.dart';
import 'sched.dart';

/// The signal of `testing.T.FailNow`.
///
/// Go stops such a call chain with `runtime.Goexit`, and each deferred
/// function of the chain still runs. A throw gives an equal result here,
/// because the subset has no `recover`.
final class GoStop implements Exception {
  const GoStop();

  @override
  String toString() => 'the goroutine stopped';
}

/// Stops the call chain. Each deferred function of the chain still runs.
void $stop() => throw const GoStop();

/// Calls f and holds a stop signal.
///
/// It reports false when f called [$stop], and true in each other case. A
/// panic passes through, thus a test binary ends on a panic exactly as Go
/// does. The caller reads [$blocked] after the call, because f can block.
bool $catch(void Function()? f) {
  if (f == null) $panicNil();
  final void Function() body = f;
  try {
    body();
  } on GoStop {
    return false;
  }
  if ($blocked != null) {
    $block(_CatchWait($blocked!));
    return false;
  }
  return true;
}

/// The frame of a call of [$catch] that gave up control.
///
/// The frame holds the stop signal of the inner chain, exactly as the first
/// call does.
final class _CatchWait implements WaitFrame {
  Frame inner;

  _CatchWait(this.inner);

  @override
  String get $reason {
    final Frame f = inner;
    if (f is WaitFrame) return f.$reason;
    return 'waiting';
  }

  @override
  Object? resume() {
    try {
      inner.resume();
    } on GoStop {
      return false;
    }
    if ($blocked != null) {
      inner = $blocked!;
      $block(this);
      return false;
    }
    return true;
  }
}

/// Ends the program with a status code.
void $exit(int code) => $platformExit(code);

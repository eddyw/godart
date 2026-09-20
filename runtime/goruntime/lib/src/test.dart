import 'panic.dart';
import 'platform_native.dart' if (dart.library.js_interop) 'platform_web.dart';
import 'sched.dart';

/// Stops the call chain. Each deferred function of the chain still runs.
///
/// `panic.dart` holds the class of the signal, because the scheduler catches
/// the same signal for `runtime.Goexit`.
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
final class _CatchWait implements WaitFrame, DurableWait, NestedFrame {
  Frame inner;

  _CatchWait(this.inner);

  @override
  Frame get $inner => inner;
  @override
  set $inner(Frame frame) {
    inner = frame;
  }

  @override
  String get $reason => $waitReason(inner);

  /// The wait of the inner chain holds the answer, thus the mark passes down.
  @override
  bool $durableIn(Bubble b) => $durableWait(inner, b);

  @override
  Object? resume() {
    try {
      $resumeFrame(inner);
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

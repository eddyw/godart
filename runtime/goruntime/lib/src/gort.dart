import 'panic.dart';
import 'sched.dart';
import 'slice.dart';
import 'string.dart';

/// The number of the goroutine that runs now.
///
/// Go numbers each goroutine from 1, and the scheduler holds the same number.
int $goid() => $current?.id ?? 1;

/// The frame of a goroutine that gave up control for [$gosched].
final class _Yield implements WaitFrame {
  const _Yield();

  @override
  String get $reason => 'runnable';

  @override
  Object? resume() => null;
}

/// Gives up control and puts the goroutine on the run queue again.
///
/// The goroutine stands behind each goroutine that waits already, thus each
/// other goroutine takes a turn first. This is the shape of runtime.Gosched.
///
/// A goroutine that spins on this call while no other goroutine can run must
/// give the Dart event loop a turn. The run queue would never empty, thus a
/// timer of a Sleep and a read of a socket would never reach the program. Go
/// runs its timers and its network poller outside the run queue, thus it needs
/// no such rule.
void $gosched() {
  final Goroutine? g = $current;
  if (g == null) return;
  if ($runqEmpty && $eventWaits > 0) {
    _yieldEvent(g);
    return;
  }
  $ready(g);
  $block(const _Yield());
}

final class _SyncYield implements Frame {
  const _SyncYield();
  @override
  Object? resume() => null;
}

void $poll() {
  if (--$pollBudget > 0) return;
  $pollBudget = 128;
  if ($syncActive) {
    $block(const _SyncYield());
    return;
  }
  final g = $current;
  if (g == null) return;
  if ($eventWaits > 0 || $hosted) {
    _yieldEvent(g);
    return;
  }
  $gosched();
}

void _yieldEvent(Goroutine g) {
  $wakeAfter(g, 0);
  $block(const _Yield());
}

/// Ends the goroutine that runs now, and runs each deferred call of its chain.
///
/// The scheduler holds the signal, thus the goroutine ends and each other
/// goroutine runs on. This is the shape of runtime.Goexit.
Never $goexit() => throw const GoStop();

GoString $creator() => GoString.fromString($current?.creatorFrame ?? '');

/// The state of each goroutine that gave up control, except the goroutine that
/// runs now.
///
/// 1 item holds 1 goroutine. The first line holds the number and the reason,
/// with a bar between them. Each line after it holds 1 frame in the form
/// "function|file|line".
Slice<GoString> $parked() {
  final List<GoString> out = <GoString>[];
  for (final Goroutine g in $goroutines) {
    if (identical(g, $current)) continue;
    final String reason = g.frame == null ? 'runnable' : $waitReason(g.frame);
    final List<TraceFrame> stack = $traceFrames(g.frame);
    final StringBuffer text = StringBuffer('${g.id}|$reason');
    for (final TraceFrame t in stack.reversed) {
      text.write('\n${t.$name}|${t.$goFile}|${t.$goLine}');
    }
    if (g.creator != null) text.write('\n${g.creatorFrame}');
    out.add(GoString.fromString(text.toString()));
  }
  return Slice.of(out);
}

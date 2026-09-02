import 'dart:async';
import 'dart:collection';

import 'panic.dart';
import 'platform_native.dart' if (dart.library.js_interop) 'platform_web.dart';

/// The saved state of 1 call that gave up control.
///
/// A function that can block writes its locals and the number of its next
/// state into such a frame. The compiler makes 1 frame class for each such
/// function. The runtime makes a frame for a channel operation, for a select
/// statement, for a sleep, and for an isolate worker.
abstract class Frame {
  /// Runs the call again from its saved state and gives its result.
  ///
  /// A frame of the runtime gives the value of the operation that woke it. A
  /// frame of a compiled function gives the result of that function. The
  /// caller reads [$blocked] after the call, because the call can block again.
  Object? resume();
}

/// The frame of a compiled function. The compiler writes the 4 members, and
/// the deadlock report walks the chain through them.
abstract interface class TraceFrame implements Frame {
  /// The Go name of the function.
  String get $name;

  /// The file of the function, in the form of the source map.
  String get $goFile;

  /// The source line of the state that resumes.
  int get $goLine;

  /// The frame under this one.
  Frame? get $next;
}

/// A frame of the runtime that holds a goroutine while it waits. The reason
/// takes the words of the goroutine header of Go, such as `chan receive`.
abstract interface class WaitFrame implements Frame {
  String get $reason;
}

/// Not null when the current goroutine must give up control.
///
/// The runtime writes the frame of the operation that blocks. Each caller on
/// the stack sees the value, saves its own locals into a frame, and returns.
/// Thus the stack unwinds up to the scheduler. The scheduler keeps the frame
/// of the last caller, and a wake runs the chain again.
Frame? $blocked;

/// One goroutine.
final class Goroutine {
  /// The frame chain of a goroutine that gave up control.
  Frame? frame;

  /// The body of a goroutine that did not start.
  void Function()? start;

  /// True for the goroutine of package main. Go ends the program when that
  /// goroutine returns.
  final bool isMain;

  /// The number of the goroutine. Go numbers each goroutine from 1.
  final int id;

  Goroutine({this.isMain = false}) : id = ++_lastId;
}

int _lastId = 0;

/// Each goroutine that did not return. The deadlock report walks it.
final Set<Goroutine> _liveSet = <Goroutine>{};

/// The goroutine that runs now.
Goroutine? $current;

final ListQueue<Goroutine> _runq = ListQueue<Goroutine>();
bool _running = false;
bool _mainDone = false;

/// The count of wake sources outside the scheduler. A timer of a sleep and a
/// future of an isolate worker each hold 1. A deadlock needs a count of 0.
int _pending = 0;

/// The count of goroutines that did not return.
int _live = 0;

/// Puts a goroutine on the run queue.
void $ready(Goroutine g) => _runq.add(g);

/// Gives up control. The caller returns a value that no code reads.
void $block(Frame f) => $blocked = f;

/// The frame of a goroutine that no operation wakes again.
final class Park implements WaitFrame {
  final String reason;

  const Park([this.reason = 'waiting']);

  @override
  String get $reason => reason;

  @override
  Object? resume() => null;
}

/// Gives up control until the end of the program. The reason reaches the
/// goroutine header of the deadlock report.
void $park([String reason = 'waiting']) => $block(Park(reason));

/// Puts a goroutine on the run queue and runs the scheduler.
///
/// A timer and a future of the Dart event loop call this function, thus the
/// scheduler starts again after the event loop woke the isolate.
void $wake(Goroutine g) {
  $ready(g);
  $schedule();
}

/// Runs after the run queue empties.
///
/// A host outside Go sets it. A Flutter host reads the result of the turn
/// there: it applies the work that a goroutine posted, and it rebuilds each
/// part of the screen that the turn changed. The hook runs 1 time for each
/// turn, thus a call of $schedule inside the hook starts no second call.
void Function()? $onIdle;

bool _inIdle = false;

/// Tells the scheduler that a host outside Go starts a goroutine.
///
/// A goroutine that waits for an event of the host is not a deadlock, thus the
/// report stays off while this flag holds true.
bool $hosted = false;

/// Runs each goroutine of the run queue until the queue is empty.
///
/// The function returns when no goroutine can run. The Dart event loop then
/// takes control, and a timer or a future starts the scheduler again.
void $schedule() {
  if (_running) return;
  _running = true;
  while (_runq.isNotEmpty && !_mainDone) {
    final Goroutine g = _runq.removeFirst();
    $current = g;
    $blocked = null;
    try {
      final Frame? f = g.frame;
      if (f != null) {
        g.frame = null;
        f.resume();
      } else {
        final void Function()? body = g.start;
        g.start = null;
        if (body != null) body();
      }
    } on GoPanic catch (p, trace) {
      _running = false;
      $current = null;
      $goPanic(p, trace);
      return;
    }
    if ($blocked != null) {
      g.frame = $blocked;
      $blocked = null;
    } else {
      _live--;
      _liveSet.remove(g);
      if (g.isMain) _mainDone = true;
    }
  }
  _running = false;
  $current = null;
  if (_mainDone) {
    _exitMain();
    return;
  }
  final void Function()? idle = $onIdle;
  if (idle != null && !_inIdle) {
    _inIdle = true;
    try {
      idle();
    } finally {
      _inIdle = false;
    }
  }
  if (!$hosted && _runq.isEmpty && _pending == 0 && _live > 0) _deadlock();
}

/// Starts a goroutine. Go runs the body after the statement, thus the body
/// waits for the scheduler.
void $go(void Function() body) {
  _live++;
  final Goroutine g = Goroutine()..start = body;
  _liveSet.add(g);
  $ready(g);
}

/// Runs package main as the first goroutine.
void $boot(void Function() body) {
  _live++;
  final Goroutine g = Goroutine(isMain: true)..start = body;
  _liveSet.add(g);
  $ready(g);
  $schedule();
}

/// The cleanups of the program, by token. A cleanup that Stop dropped holds
/// null.
final List<Object?> _cleanupTokens = <Object?>[];

final Finalizer<void Function()> _cleanups = Finalizer<void Function()>((
  void Function() body,
) {
  $go(body);
  $schedule();
});

/// Attaches a cleanup to a value and gives its handle.
///
/// Dart gives no way to force a collection, thus the time of the run follows the
/// collector of Dart. Read docs/parity.md.
int $addCleanup(Object target, void Function() run) {
  final Object token = Object();
  _cleanups.attach(target, run, detach: token);
  _cleanupTokens.add(token);
  return _cleanupTokens.length;
}

/// Drops a cleanup that no collection ran.
void $stopCleanup(int handle) {
  if (handle <= 0 || handle > _cleanupTokens.length) return;
  final Object? token = _cleanupTokens[handle - 1];
  if (token == null) return;
  _cleanups.detach(token);
  _cleanupTokens[handle - 1] = null;
}

/// Marks the start of a wait on the Dart event loop.
void $holdEvent() => _pending++;

/// Marks the end of a wait on the Dart event loop.
void $freeEvent() => _pending--;

/// The frame of a goroutine that waits for the Dart event loop.
///
/// The value of the operation reaches the caller through the resume, thus the
/// dead result of the operation itself needs no value.
final class EventWait implements WaitFrame {
  Object? value;

  @override
  String get $reason => 'IO wait';

  @override
  Object? resume() => value;
}

/// Gives up control until the future completes.
///
/// The value of the future reaches the caller through the frame. A failure
/// reaches onError, which writes the Go form of the result. Each operation of
/// dart:io that gives a future reads this function.
///
/// The value must hold the Go form of each integer, thus a count and a handle
/// each ride a GoInt. The direct result of the shim keeps the plain int of the
/// runtime, because the compiler wraps that result and the frame answer skips
/// the wrapper. The 2 forms are equal on each target except dart2js, where
/// GoInt is a class of its own.
void $await(Future<Object?> future, Object? Function(Object cause) onError) {
  final Goroutine g = $current!;
  final EventWait waiting = EventWait();
  $holdEvent();
  unawaited(
    future.then(
      (Object? value) {
        waiting.value = value;
        $freeEvent();
        $wake(g);
      },
      onError: (Object cause) {
        waiting.value = onError(cause);
        $freeEvent();
        $wake(g);
      },
    ),
  );
  $block(waiting);
}

/// The timers of the program, by handle. A slot of a timer that ran or that a
/// stop cancelled holds null.
final List<Timer?> _timers = <Timer?>[];

int _timerSlot() {
  for (int i = 0; i < _timers.length; i++) {
    if (_timers[i] == null) return i;
  }
  _timers.add(null);
  return _timers.length - 1;
}

/// Starts a timer that runs a Go function after a time, and gives its handle.
///
/// Go runs the function on a new goroutine, thus the function can block. The
/// timer holds 1 wake source of the scheduler until it runs.
int $startTimer(int nanoseconds, void Function()? body) {
  if (body == null) $panicNil();
  final void Function() run = body;
  final int handle = _timerSlot();
  $holdEvent();
  _timers[handle] = Timer(Duration(microseconds: nanoseconds ~/ 1000), () {
    _timers[handle] = null;
    $freeEvent();
    $go(run);
    $schedule();
  });
  return handle;
}

/// Cancels a timer. It reports false when the timer ran already, exactly as
/// the Stop method of Go does.
bool $stopTimer(int handle) {
  if (handle < 0 || handle >= _timers.length) return false;
  final Timer? timer = _timers[handle];
  if (timer == null) return false;
  _timers[handle] = null;
  timer.cancel();
  $freeEvent();
  return true;
}

/// The releases that run when package main returns.
///
/// Go ends the process at that point, thus no open handle survives. An open
/// socket of Dart keeps the isolate alive, thus the network library adds a
/// release of its own.
final List<void Function()> $releases = <void Function()>[];

/// Ends the program when package main returns.
///
/// Go ends the program at that point, and it does not wait for another
/// goroutine. A pending timer keeps the Dart isolate alive, thus the program
/// needs an explicit exit.
void _exitMain() {
  for (final void Function() release in $releases) {
    release();
  }
  if (_pending > 0) $platformExit(0);
}

/// Reports that no goroutine can run again. The report walks the frame chain
/// of each live goroutine, thus the trace holds the Go frames of the wait. Go
/// also writes the arguments and an offset in each frame, and the chain holds
/// no such value.
void _deadlock() {
  $platformStderrText('fatal error: all goroutines are asleep - deadlock!\n');
  for (final Goroutine g in _liveSet) {
    var reason = 'waiting';
    final List<TraceFrame> stack = <TraceFrame>[];
    Frame? f = g.frame;
    while (f != null) {
      if (f is TraceFrame) {
        stack.add(f);
        f = f.$next;
        continue;
      }
      if (f is WaitFrame) reason = f.$reason;
      break;
    }
    $platformStderrText('\ngoroutine ${g.id} [$reason]:\n');
    for (final TraceFrame t in stack.reversed) {
      $platformStderrText('${t.$name}()\n\t${t.$goFile}:${t.$goLine}\n');
    }
  }
  $platformExit(2);
}

/// The report of a panic that leaves a goroutine. The build driver sets it,
/// because the report needs the entry point of the program.
void Function(GoPanic, StackTrace) $goPanic = (GoPanic p, StackTrace trace) {
  throw p;
};

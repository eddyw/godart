import 'dart:async';
import 'dart:collection';

import 'goint.dart';
import 'panic.dart';
import 'platform_native.dart' if (dart.library.js_interop) 'platform_web.dart';
import 'string.dart';

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
  set $next(Frame? frame);
}

/// A frame of the runtime that holds a goroutine while it waits. The reason
/// takes the words of the goroutine header of Go, such as `chan receive`.
abstract interface class WaitFrame implements Frame {
  String get $reason;
}

/// A frame that holds the chain of an inner call in front of the wait.
///
/// `$catch` of the test harness and the 2 wrappers of the package reflect each
/// hold such a chain. The wait of the inner chain gives the reason and the
/// durable mark. Each walk of the runtime thus reads through the wrapper.
abstract interface class NestedFrame implements Frame {
  Frame get $inner;
  set $inner(Frame frame);
}

/// A wait that only a goroutine of 1 bubble can end.
///
/// synctest names such a state durably blocked. A frame of the IO of Dart
/// holds no such mark, because an event outside the bubble ends that wait.
abstract interface class DurableWait implements Frame {
  /// True when a goroutine of the bubble b can end the wait, and no event
  /// outside b can.
  bool $durableIn(Bubble b);
}

/// The frame of the wait at the end of a chain, and null for a chain that
/// holds none.
///
/// The chain holds the frame of each caller in front of the frame of the wait,
/// and a nested frame holds the chain of an inner call.
Frame? $waitFrame(Frame? f) {
  Frame? at = f;
  while (true) {
    if (at is TraceFrame) {
      at = at.$next;
      continue;
    }
    if (at is NestedFrame) {
      at = at.$inner;
      continue;
    }
    return at;
  }
}

/// The reason of the wait at the end of a chain, in the words of Go.
String $waitReason(Frame? f) {
  final Frame? at = $waitFrame(f);
  return at is WaitFrame ? at.$reason : 'waiting';
}

/// The Go frames of a chain, from the outermost caller to the innermost.
List<TraceFrame> $traceFrames(Frame? f) {
  final List<TraceFrame> out = <TraceFrame>[];
  Frame? at = f;
  while (true) {
    if (at is TraceFrame) {
      out.add(at);
      at = at.$next;
      continue;
    }
    if (at is NestedFrame) {
      at = at.$inner;
      continue;
    }
    return out;
  }
}

/// True when only a goroutine of the bubble b can end the wait of the chain.
bool $durableWait(Frame? f, Bubble b) {
  final Frame? at = $waitFrame(f);
  return at is DurableWait && at.$durableIn(b);
}

/// Not null when the current goroutine must give up control.
///
/// The runtime writes the frame of the operation that blocks. Each caller on
/// the stack sees the value, saves its own locals into a frame, and returns.
/// Thus the stack unwinds up to the scheduler. The scheduler keeps the frame
/// of the last caller, and a wake runs the chain again.
Frame? $blocked;

/// One goroutine, and the handle of its end.
///
/// `$go` gives the handle to a Dart host outside Go. The emitted Dart of a Go
/// program drops it, thus a Go program reads no member of this class.
final class Goroutine {
  /// The frame chain of a goroutine that gave up control.
  Frame? frame;

  /// The body of a goroutine that did not start.
  Object? Function()? start;

  /// True for the goroutine of package main. Go ends the program when that
  /// goroutine returns.
  final bool isMain;

  /// The number of the goroutine. Go numbers each goroutine from 1.
  final int id;

  /// The bubble of the goroutine, and null outside a bubble. A goroutine that
  /// a bubble starts joins that bubble.
  Bubble? bubble;

  ChainState _chain = const ChainState(null, false);
  final _resumeFrames = <Frame>[];

  bool _ended = false;
  Object? _result;
  void Function()? _hook;

  Goroutine({this.isMain = false}) : id = ++_lastId;

  /// True after the goroutine ended. A goroutine ends 3 ways: the body
  /// returns, a panic leaves the body, and `runtime.Goexit` stops it.
  bool get ended => _ended;

  /// The value that the body gave. A panic and `runtime.Goexit` each leave
  /// null, and a body of the type `void Function()` leaves null too.
  ///
  /// A body that blocks gives its value through the frame of the last resume,
  /// thus a Dart wrapper around the body loses that value. A host that reads
  /// the result must give the Go function itself to `$go`.
  Object? get result => _result;

  /// Runs [hook] 1 time, at the end of the goroutine. A goroutine that ended
  /// already runs it now. A second call replaces the hook.
  void whenEnd(void Function() hook) {
    if (_ended) {
      hook();
      return;
    }
    _hook = hook;
  }

  void _end(Object? value) {
    if (_ended) return;
    _ended = true;
    _resumeFrames.clear();
    _result = value;
    final void Function()? hook = _hook;
    if (hook == null) return;
    _hook = null;
    hook();
  }
}

int _lastId = 0;

/// Each goroutine that did not return. The deadlock report walks it.
final Set<Goroutine> _liveSet = <Goroutine>{};

/// Each goroutine that did not return. `runtime.Stack` of the surface reads it
/// for the trace of a goroutine that waits.
Iterable<Goroutine> get $goroutines => _liveSet;

/// The goroutine that runs now.
Goroutine? $current;

/// Saved callers that the iterative resume has not entered yet.
Iterable<TraceFrame> get $activeCallerFrames =>
    $current?._resumeFrames.reversed.whereType<TraceFrame>() ??
    const <TraceFrame>[];

final ListQueue<Goroutine> _runq = ListQueue<Goroutine>();
bool _running = false;
bool _mainDone = false;

/// True when the goroutine of main ended through `runtime.Goexit`.
///
/// Go keeps the program alive in that case, and it reports a crash when each
/// other goroutine ends too.
bool _mainExited = false;

/// The count of wake sources outside the scheduler. A timer of a sleep and a
/// future of an isolate worker each hold 1. A deadlock needs a count of 0.
int _pending = 0;

/// The count of goroutines that did not return.
int _live = 0;

/// Puts a goroutine on the run queue.
void $ready(Goroutine g) => _runq.add(g);

/// True when the run queue holds no goroutine.
bool get $runqEmpty => _runq.isEmpty;

/// The count of the waits on the Dart event loop.
int get $eventWaits => _pending;

/// Gives up control. The caller returns a value that no code reads.
void $block(Frame f) => $blocked = f;

/// The frame of a goroutine that no operation wakes again.
///
/// Go counts such a wait as durably blocked. A nil channel and a select
/// statement with no case each hold a goroutine for the whole program.
final class Park implements WaitFrame, DurableWait {
  final String reason;

  const Park([this.reason = 'waiting']);

  @override
  String get $reason => reason;

  @override
  bool $durableIn(Bubble b) => true;

  @override
  Object? resume() => null;
}

/// Gives up control until the end of the program. The reason reaches the
/// goroutine header of the deadlock report.
void $park([String reason = 'waiting']) => $block(Park(reason));

/// The frame of a goroutine that waits for a lock.
///
/// Go marks such a wait not durable, because a goroutine outside the bubble
/// can give the lock back. The frame thus holds no DurableWait.
final class _LockWait implements WaitFrame {
  const _LockWait();

  @override
  String get $reason => 'sync.Mutex.Lock';

  @override
  Object? resume() => null;
}

/// The goroutine of each wait for a lock, by token.
final Map<int, Goroutine> _lockWaits = <int, Goroutine>{};

/// The count of the tokens that $lockToken gave.
int _lastLockToken = 0;

/// Gives a token for 1 wait on a lock.
int $lockToken() => ++_lastLockToken;

/// Gives up control until $lockWake runs with the same token.
///
/// The wait is not durable. A bubble of synctest that holds such a goroutine
/// thus stands still, and its clock does not move.
void $lockWait(int token) {
  _lockWaits[token] = $current!;
  $block(const _LockWait());
}

/// Ends the wait of the token. It performs no operation for a token that no
/// goroutine waits on.
void $lockWake(int token) {
  final Goroutine? g = _lockWaits.remove(token);
  if (g != null) $ready(g);
}

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
  while (!_mainDone) {
    if (_runq.isEmpty) {
      if (!_stepBubbles()) break;
      continue;
    }
    final Goroutine g = _runq.removeFirst();
    $current = g;
    $blocked = null;
    Object? value;
    final outerChain = ChainState($chainPanic, $recovered);
    $chainClose(g._chain);
    try {
      final Frame? f = g.frame;
      if (f != null) {
        g.frame = null;
        value = _resumeGraph(f);
      } else {
        final Object? Function()? body = g.start;
        g.start = null;
        if (body != null) value = body();
      }
    } on GoPanic catch (p, trace) {
      _running = false;
      $current = null;
      // The goroutine ends here, thus the handle reports the end. The live set
      // keeps the goroutine, because the program itself ends now.
      g._end(null);
      $goPanic(p, trace);
      return;
    } on GoStop {
      // runtime.Goexit ended the goroutine, and each deferred call of the
      // chain already ran. Go keeps the program alive when the goroutine of
      // main ends this way, thus the flag replaces the end of the program.
      $blocked = null;
      _live--;
      _liveSet.remove(g);
      g.bubble?.leave(g);
      if (g.isMain) _mainExited = true;
      g._end(null);
      continue;
    } finally {
      g._chain = ChainState($chainPanic, $recovered);
      $chainClose(outerChain);
    }
    if ($blocked != null) {
      g.frame = $blocked;
      $blocked = null;
    } else {
      _live--;
      _liveSet.remove(g);
      g.bubble?.leave(g);
      if (g.isMain) _mainDone = true;
      g._end(value);
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
  if (!$hosted &&
      _runq.isEmpty &&
      _pending == 0 &&
      (_live > 0 || _mainExited)) {
    _deadlock();
  }
}

/// Starts a goroutine and gives its handle. Go runs the body after the
/// statement, thus the body waits for the scheduler.
///
/// The emitted Dart of a `go` statement drops the handle, thus a Go program
/// pays nothing for it. A Dart host reads the end of the goroutine there.
///
/// The parameter takes the type `void Function()`, because the emitted Dart
/// writes a closure with no result. A body that gives a value is a subtype of
/// that type, thus the field [Goroutine.start] still keeps the value.
Goroutine $go(void Function() body) => _spawn(body, $current?.bubble);

/// Starts a goroutine inside the bubble b, and gives its handle. A null bubble
/// starts a goroutine outside each bubble.
Goroutine _spawn(void Function() body, Bubble? b) {
  _live++;
  final Goroutine g = Goroutine()..start = body;
  if (b != null) {
    g.bubble = b;
    b.members.add(g);
  }
  _liveSet.add(g);
  $ready(g);
  return g;
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

/// The timers that did not run, by handle. An entry holds a Timer of Dart, or
/// a _FakeTimer of a bubble. A timer that ran, and a timer that a stop
/// cancelled, holds no entry.
final Map<int, Object> _timers = <int, Object>{};

/// The count of the handles that _timerHandle gave.
int _lastTimer = 0;

/// Gives a handle that no earlier timer holds.
///
/// An odd handle names a timer of a bubble, exactly as the flag isFake of the
/// runtime of Go does. The handle of a timer that ran keeps that mark, thus a
/// stop from outside each bubble still finds the mark. No handle repeats, thus
/// a stop of a timer that ran reaches no later timer.
int _timerHandle(bool fake) => (++_lastTimer) * 2 + (fake ? 1 : 0);

/// One timer of a bubble. The fake clock of the bubble runs it, thus it takes
/// no real time and it holds no wake source of the Dart event loop.
final class _FakeTimer {
  final Bubble bubble;

  /// The value of the fake clock at which the timer runs.
  final GoInt when;

  final void Function() run;

  _FakeTimer(this.bubble, this.when, this.run);
}

/// The mark of a timer that stood armed at the end of its bubble.
///
/// Go keeps such a timer, thus a stop of it reports true. The mark carries no
/// bubble and no body, thus the timer runs never.
const Object _endedTimer = Object();

void $wakeAfter(Goroutine g, int nanoseconds) {
  final handle = _timerHandle(false);
  $holdEvent();
  _timers[handle] = Timer(Duration(microseconds: nanoseconds ~/ 1000), () {
    _timers.remove(handle);
    $freeEvent();
    $wake(g);
  });
}

/// Starts a timer that runs a Go function after a time, and gives its handle.
///
/// Go runs the function on a new goroutine, thus the function can block. The
/// timer holds 1 wake source of the scheduler until it runs.
///
/// A timer of a bubble reads the clock of that bubble. Such a timer waits for
/// the scheduler and not for the Dart event loop.
int $startTimer(int nanoseconds, void Function()? body) {
  if (body == null) $panicNil();
  final void Function() run = body;
  final Bubble? b = $current?.bubble;
  if (b != null) {
    final int handle = _timerHandle(true);
    _timers[handle] = _FakeTimer(
      b,
      b.now + $gi(nanoseconds),
      () => _spawn(run, b),
    );
    return handle;
  }
  final int handle = _timerHandle(false);
  $holdEvent();
  _timers[handle] = Timer(Duration(microseconds: nanoseconds ~/ 1000), () {
    _timers.remove(handle);
    $freeEvent();
    $go(run);
    $schedule();
  });
  return handle;
}

/// Cancels a timer. It reports false when the timer ran already, exactly as
/// the Stop method of Go does.
bool $stopTimer(int handle) => _cancelTimer(handle, 'stop');

/// Cancels a timer for a restart. It reports false when the timer ran already.
///
/// Go names the operation in the report of a timer of a bubble. A reset thus
/// needs a name of its own.
bool $resetTimer(int handle) => _cancelTimer(handle, 'reset');

bool _cancelTimer(int handle, String op) {
  if (handle.isOdd && $current?.bubble == null) {
    $fatalError('$op of synctest timer from outside bubble');
  }
  final Object? held = _timers.remove(handle);
  if (held == null) return false;
  if (held is Timer) {
    held.cancel();
    $freeEvent();
  }
  return true;
}

/// The base of the fake clock of a bubble. Go starts each bubble at midnight
/// UTC of 2000-01-01.
final GoInt $bubbleBase = $gi(946684800) * $gi(1000000000);

/// Each bubble that did not end. A goroutine outside a bubble can start a
/// bubble of its own, thus the list holds more than 1 entry.
final List<Bubble> _bubbles = <Bubble>[];

/// The clock, the timers, and the goroutines of 1 synctest bubble.
///
/// Go gives each bubble a clock of its own. The clock moves only when each
/// goroutine of the bubble is durably blocked, thus a sleep inside a bubble
/// takes no real time.
final class Bubble {
  /// The fake clock, in nanoseconds since the start of the Unix epoch.
  GoInt now = $bubbleBase;

  /// Each goroutine of the bubble that did not end.
  final Set<Goroutine> members = <Goroutine>{};

  /// The goroutine outside the bubble that started it.
  final Goroutine owner;

  /// The frame of the owner. The end of the bubble writes the cause of a
  /// deadlock into it.
  final _BubbleWait _wait;

  /// The first goroutine of the bubble. Go stops the clock after it ends.
  Goroutine? root;

  /// The goroutine that waits inside a call of synctest.Wait.
  Goroutine? waiter;

  /// True after the first goroutine of the bubble ended.
  bool rootEnded = false;

  Bubble._(this.owner, this._wait);

  /// Drops a goroutine that ended.
  void leave(Goroutine g) {
    members.remove(g);
    g.bubble = null;
    if (identical(g, waiter)) waiter = null;
    if (identical(g, root)) rootEnded = true;
  }

  /// True when each goroutine of the bubble is durably blocked.
  bool get _idle {
    for (final Goroutine g in members) {
      if (!$durableWait(g.frame, this)) return false;
    }
    return true;
  }

  /// The time of the first timer of the bubble, and null when it holds none.
  GoInt? _nextTime() {
    GoInt? out;
    for (final Object t in _timers.values) {
      if (t is! _FakeTimer || !identical(t.bubble, this)) continue;
      if (out == null || t.when < out) out = t.when;
    }
    return out;
  }

  /// Runs each timer that the clock reached, and reports whether 1 ran.
  bool _fire() {
    var ran = false;
    for (final int handle in _timers.keys.toList()) {
      final Object? t = _timers[handle];
      if (t is! _FakeTimer || !identical(t.bubble, this) || t.when > now) {
        continue;
      }
      _timers.remove(handle);
      t.run();
      ran = true;
    }
    return ran;
  }

  /// Moves the bubble 1 step and reports whether a goroutine can run now.
  ///
  /// The order follows Go. A timer that the clock reached runs first, then a
  /// call of synctest.Wait ends, and then the clock moves to the next timer.
  bool step() {
    if (!_idle) return false;
    final GoInt? next = _nextTime();
    if (next != null && next <= now && _fire()) return true;
    final Goroutine? w = waiter;
    if (w != null) {
      waiter = null;
      $ready(w);
      return true;
    }
    if (next != null && !rootEnded) {
      now = next;
      if (_fire()) return true;
    }
    _end();
    return true;
  }

  /// Ends the bubble and wakes the goroutine that started it.
  void _end() {
    _bubbles.remove(this);
    for (final int handle in _timers.keys.toList()) {
      final Object? t = _timers[handle];
      if (t is _FakeTimer && identical(t.bubble, this)) {
        _timers[handle] = _endedTimer;
      }
    }
    if (members.isNotEmpty) {
      _wait.cause = GoString.fromString(
        rootEnded
            ? 'deadlock: main bubble goroutine has exited but blocked '
                  'goroutines remain'
            : 'deadlock: all goroutines in bubble are blocked',
      );
    }
    for (final Goroutine g in members) {
      g.bubble = null;
    }
    members.clear();
    $ready(owner);
  }
}

/// Moves 1 bubble that is idle, and reports whether a goroutine can run now.
///
/// The bubble that moved goes to the end of the list. Go runs 2 bubbles at the
/// same time, thus a bubble with a timer that never ends must not hold another
/// bubble back.
bool _stepBubbles() {
  for (var i = 0; i < _bubbles.length; i++) {
    final Bubble b = _bubbles[i];
    if (!b.step()) continue;
    if (_bubbles.remove(b)) _bubbles.add(b);
    return true;
  }
  return false;
}

/// The frame of the goroutine that waits for the end of a bubble.
final class _BubbleWait implements WaitFrame {
  /// The cause of a deadlock of the bubble. It stays empty after a normal end.
  GoString cause = GoString.empty;

  @override
  String get $reason => 'synctest.Run';

  @override
  Object? resume() => cause;
}

/// The frame of a goroutine that sleeps on the clock of a bubble.
final class _BubbleSleep implements WaitFrame, DurableWait {
  const _BubbleSleep();

  @override
  String get $reason => 'sleep';

  @override
  bool $durableIn(Bubble b) => true;

  @override
  Object? resume() => null;
}

/// The frame of a goroutine inside a call of synctest.Wait.
final class _BubbleWaitCall implements WaitFrame, DurableWait {
  const _BubbleWaitCall();

  @override
  String get $reason => 'synctest.Wait';

  @override
  bool $durableIn(Bubble b) => true;

  @override
  Object? resume() => true;
}

/// True when the goroutine of the caller stands inside a bubble.
bool $inBubble() => $current?.bubble != null;

/// Runs body on the first goroutine of a new bubble, and waits for the end of
/// each goroutine of the bubble.
///
/// The result names the cause of a deadlock of the bubble, and it is empty
/// after a normal end.
GoString $bubbleRun(void Function()? body) {
  if (body == null) $panicNil();
  final void Function() start = body;
  final _BubbleWait waiting = _BubbleWait();
  final Bubble b = Bubble._($current!, waiting);
  _bubbles.add(b);
  b.root = _spawn(start, b);
  $block(waiting);
  return GoString.empty;
}

/// Waits until each other goroutine of the bubble of the caller is durably
/// blocked.
///
/// It reports false when another goroutine of the bubble waits already, and it
/// then gives up no control.
bool $bubbleWait() {
  final Goroutine g = $current!;
  final Bubble? b = g.bubble;
  if (b == null) return true;
  if (b.waiter != null) return false;
  b.waiter = g;
  $block(const _BubbleWaitCall());
  return true;
}

/// Gives up control until the clock of the bubble b moves the count of
/// nanoseconds. The sleep of the package time reads it inside a bubble.
void $bubbleSleep(Bubble b, Goroutine g, int nanoseconds) {
  final int handle = _timerHandle(true);
  _timers[handle] = _FakeTimer(b, b.now + $gi(nanoseconds), () => $ready(g));
  $block(const _BubbleSleep());
}

/// The releases that run when package main returns.
///
/// Go ends the process at that point, thus no open handle survives. An open
/// socket of Dart keeps the isolate alive, thus the network library adds a
/// release of its own.
final List<void Function()> $releases = <void Function()>[];

/// Ends the program when package main returns.
///
/// Go does not wait for another goroutine. Cancel timers and release handles
/// that can keep the Dart isolate alive.
void _exitMain() {
  for (final Object timer in _timers.values) {
    if (timer is Timer) {
      timer.cancel();
      $freeEvent();
    }
  }
  _timers.clear();
  for (final void Function() release in $releases) {
    release();
  }
  if (_pending > 0) $platformExit(0);
}

/// Ends the program with a report that no recover reads.
///
/// Go writes such a report for a fault that a program must not handle. A
/// channel of a bubble that a goroutine outside it reached is 1 such fault.
Never $fatalError(String text) {
  $platformStderrText('fatal error: $text\n');
  $platformExit(2);
}

/// Reports that no goroutine can run again. The report walks the frame chain
/// of each live goroutine, thus the trace holds the Go frames of the wait. Go
/// also writes the arguments and an offset in each frame, and the chain holds
/// no such value.
void _deadlock() {
  $platformStderrText(
    _mainExited
        ? 'fatal error: no goroutines (main called runtime.Goexit) - deadlock!\n'
        : 'fatal error: all goroutines are asleep - deadlock!\n',
  );
  for (final Goroutine g in _liveSet) {
    final String reason = $waitReason(g.frame);
    final List<TraceFrame> stack = $traceFrames(g.frame);
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

final _frameResults = Map<Frame, _FrameResult>.identity();

final class _FrameResult {
  final Object? value;
  final Object? error;
  final StackTrace? trace;
  _FrameResult(this.value) : error = null, trace = null;
  _FrameResult.error(this.error, this.trace) : value = null;
  Object? deliver() {
    if (error != null) Error.throwWithStackTrace(error!, trace!);
    return value;
  }
}

/// Consumes a child result without another recursive resume.
Object? $resumeFrame(Frame frame) {
  final result = _frameResults.remove(frame);
  return result == null ? frame.resume() : result.deliver();
}

void _appendFrames(List<Frame> frames, Frame? frame) {
  while (frame != null) {
    frames.add(frame);
    frame = switch (frame) {
      final TraceFrame f => f.$next,
      final NestedFrame f => f.$inner,
      _ => null,
    };
  }
}

Object? _resumeGraph(Frame root) {
  final frames = $current!._resumeFrames;
  if (frames.isEmpty) _appendFrames(frames, root);
  try {
    while (true) {
      final frame = frames.removeLast();
      late _FrameResult result;
      try {
        result = _FrameResult(frame.resume());
      } catch (error, trace) {
        result = _FrameResult.error(error, trace);
      }
      if ($blocked != null) {
        final blocked = $blocked!;
        if (frames.isNotEmpty) {
          switch (frames.last) {
            case final TraceFrame parent:
              parent.$next = blocked;
            case final NestedFrame parent:
              parent.$inner = blocked;
          }
        }
        _appendFrames(frames, blocked);
        $blocked = frames.first;
        return null;
      }
      if (frames.isEmpty) return result.deliver();
      _frameResults[frame] = result;
    }
  } finally {
    _frameResults.clear();
  }
}

import 'dart:collection';
import 'dart:io';

import 'panic.dart';

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

  Goroutine({this.isMain = false});
}

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
final class Park implements Frame {
  const Park();

  @override
  Object? resume() => null;
}

/// Gives up control until the end of the program.
void $park() => $block(const Park());

/// Puts a goroutine on the run queue and runs the scheduler.
///
/// A timer and a future of the Dart event loop call this function, thus the
/// scheduler starts again after the event loop woke the isolate.
void $wake(Goroutine g) {
  $ready(g);
  $schedule();
}

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
    } on GoPanic catch (p) {
      _running = false;
      $current = null;
      $goPanic(p);
      return;
    }
    if ($blocked != null) {
      g.frame = $blocked;
      $blocked = null;
    } else {
      _live--;
      if (g.isMain) _mainDone = true;
    }
  }
  _running = false;
  $current = null;
  if (_mainDone) {
    _exitMain();
    return;
  }
  if (_runq.isEmpty && _pending == 0 && _live > 0) _deadlock();
}

/// Starts a goroutine. Go runs the body after the statement, thus the body
/// waits for the scheduler.
void $go(void Function() body) {
  _live++;
  $ready(Goroutine()..start = body);
}

/// Runs package main as the first goroutine.
void $boot(void Function() body) {
  _live++;
  $ready(Goroutine(isMain: true)..start = body);
  $schedule();
}

/// Marks the start of a wait on the Dart event loop.
void $holdEvent() => _pending++;

/// Marks the end of a wait on the Dart event loop.
void $freeEvent() => _pending--;

/// Ends the program when package main returns.
///
/// Go ends the program at that point, and it does not wait for another
/// goroutine. A pending timer keeps the Dart isolate alive, thus the program
/// needs an explicit exit.
void _exitMain() {
  if (_pending > 0) exit(0);
}

/// Reports that no goroutine can run again.
void _deadlock() {
  stderr.writeln('fatal error: all goroutines are asleep - deadlock!');
  exit(2);
}

/// The report of a panic that leaves a goroutine. The build driver sets it,
/// because the report needs the entry point of the program.
void Function(GoPanic) $goPanic = (GoPanic p) {
  throw p;
};

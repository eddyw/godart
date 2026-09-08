import 'dart:async';
import 'dart:collection';
import 'dart:isolate';
import 'dart:typed_data';

import 'goint.dart';
import 'iface.dart';
import 'panic.dart';
import 'sched.dart';
import 'slice.dart';
import 'string.dart';

/// A Go channel.
///
/// All goroutines run in 1 Dart isolate and give up control only at a channel
/// operation, at a select statement, at a sleep, and at an isolate worker.
/// Thus the queues need no lock.
final class Chan<T> {
  final int capacity;
  final T Function() _zero;

  // Each queue takes its value on the first read. The zero value of a channel
  // type reaches each variable of a frame, thus a nil channel must stay cheap.
  late final ListQueue<T> _buffer = ListQueue<T>();
  late final ListQueue<_RecvWait<T>> _receivers = ListQueue<_RecvWait<T>>();
  late final ListQueue<_SendWait<T>> _senders = ListQueue<_SendWait<T>>();
  bool _closed = false;

  /// True for the zero value of a channel type. A send and a receive on such a
  /// channel never end.
  final bool isNil;

  /// The bubble of the goroutine that made the channel, and null outside a
  /// bubble. synctest reads it: a wait on a channel of the bubble of the
  /// goroutine is durable, and no event outside that bubble ends it.
  final Bubble? bubble;

  Chan(this.capacity, this._zero) : isNil = false, bubble = $current?.bubble {
    // Go reports a capacity below 0 with a plain text and no prefix.
    if (capacity < 0) {
      $panic(
        Iface(
          $typeGoError,
          const GoError('makechan: size out of range', plain: true),
        ),
      );
    }
  }

  Chan.nil(this._zero) : capacity = 0, isNil = true, bubble = null;

  int get length => _buffer.length;

  bool get isClosed => _closed;

  /// The equality of Go. 2 nil channels of 1 element type are 1 value, and
  /// each other pair compares by identity.
  ///
  /// Each library writes its own nil channel of an element type, and the file
  /// of a shared unnamed struct type writes 1 more. The identity of the object
  /// thus names no value of Go. The type stands outside the test, because the
  /// caller already compared the 2 types: Go rejects a comparison of 2 channel
  /// types that differ, and the equality of an interface value reads the name
  /// of the descriptor first.
  @override
  bool operator ==(Object other) =>
      identical(this, other) || (other is Chan && isNil && other.isNil);

  @override
  int get hashCode => isNil ? 0 : identityHashCode(this);

  /// True when a receive gives a value at this moment.
  bool get readyToReceive =>
      !isNil && (_buffer.isNotEmpty || _hasSender || _closed);

  /// True when a send gives up no control at this moment.
  bool get readyToSend =>
      !isNil && (_closed || _hasReceiver || _buffer.length < capacity);

  bool get _hasSender => _senders.any((_SendWait<T> w) => w.sel?.done != true);

  bool get _hasReceiver =>
      _receivers.any((_RecvWait<T> w) => w.sel?.done != true);

  /// Ends the program when a goroutine outside the bubble of the channel
  /// reaches the channel. Go holds the same rule, because such a call breaks
  /// the isolation of the bubble.
  void _outside(String op) {
    if (identical(bubble, $current?.bubble)) return;
    $fatalError('$op synctest channel from outside bubble');
  }

  /// Reads 1 value. The caller reads [$blocked] after the call.
  (T, bool) receive() {
    if (bubble != null) _outside('receive on');
    if (isNil) {
      $park('chan receive (nil chan)');
      return (_zero(), false);
    }
    if (readyToReceive) return receiveNow();
    final _RecvWait<T> waiting = _RecvWait<T>($current!, _zero, bubble);
    _receivers.add(waiting);
    $block(waiting);
    return (_zero(), false);
  }

  /// Receives with no wait. The caller reads readyToReceive first.
  (T, bool) receiveNow() {
    if (_buffer.isNotEmpty) {
      final T value = _buffer.removeFirst();
      final _SendWait<T>? waiting = _takeSender();
      if (waiting != null) {
        _buffer.add(waiting.value);
        $ready(waiting.g);
      }
      return (value, true);
    }
    final _SendWait<T>? waiting = _takeSender();
    if (waiting != null) {
      $ready(waiting.g);
      return (waiting.value, true);
    }
    return (_zero(), false);
  }

  /// Takes the first sender that waits. A waiter of a select statement that
  /// already took a case holds no value, thus the queue drops it.
  _SendWait<T>? _takeSender() {
    while (_senders.isNotEmpty) {
      final _SendWait<T> waiting = _senders.removeFirst();
      final SelectState? sel = waiting.sel;
      if (sel == null) return waiting;
      if (sel.done) continue;
      sel.taken = waiting.index;
      return waiting;
    }
    return null;
  }

  /// Takes the first receiver that waits.
  _RecvWait<T>? _takeReceiver() {
    while (_receivers.isNotEmpty) {
      final _RecvWait<T> waiting = _receivers.removeFirst();
      final SelectState? sel = waiting.sel;
      if (sel == null) return waiting;
      if (sel.done) continue;
      sel.taken = waiting.index;
      return waiting;
    }
    return null;
  }

  /// Puts the waiter of 1 case of a select statement into the queue. A second
  /// goroutine then meets the statement, also from a select statement of its
  /// own.
  void armSend(SelectState sel, int index, Goroutine g, T value) {
    _senders.add(_SendWait<T>(g, value, bubble, sel, index));
  }

  void armRecv(SelectState sel, int index, Goroutine g) {
    _receivers.add(_RecvWait<T>(g, _zero, bubble, sel, index));
  }

  /// Drops each waiter of a select statement that ended.
  void disarm(SelectState sel) {
    _senders.removeWhere((_SendWait<T> w) => identical(w.sel, sel));
    _receivers.removeWhere((_RecvWait<T> w) => identical(w.sel, sel));
  }

  /// Writes 1 value. The caller reads [$blocked] after the call.
  void send(T value) {
    if (bubble != null) _outside('send on');
    if (isNil) {
      $park('chan send (nil chan)');
      return;
    }
    if (readyToSend) {
      sendNow(value);
      return;
    }
    final _SendWait<T> waiting = _SendWait<T>($current!, value, bubble);
    _senders.add(waiting);
    $block(waiting);
  }

  /// Sends with no wait. The caller reads readyToSend first.
  void sendNow(T value) {
    if (_closed) {
      $panic(
        Iface(
          $typeGoError,
          const GoError('send on closed channel', plain: true),
        ),
      );
    }
    final _RecvWait<T>? waiting = _takeReceiver();
    if (waiting != null) {
      waiting.value = (value, true);
      final SelectState? sel = waiting.sel;
      if (sel != null) {
        sel.value = value;
        sel.ok = true;
      }
      $ready(waiting.g);
      return;
    }
    _buffer.add(value);
  }

  void close() {
    if (bubble != null) _outside('close of');
    if (isNil) {
      $panic(
        Iface($typeGoError, const GoError('close of nil channel', plain: true)),
      );
    }
    if (_closed) {
      $panic(
        Iface(
          $typeGoError,
          const GoError('close of closed channel', plain: true),
        ),
      );
    }
    _closed = true;
    while (true) {
      final _RecvWait<T>? waiting = _takeReceiver();
      if (waiting == null) break;
      waiting.value = (_zero(), false);
      final SelectState? sel = waiting.sel;
      if (sel != null) {
        sel.value = _zero();
        sel.ok = false;
      }
      $ready(waiting.g);
    }
  }
}

/// The state of 1 select statement that waits.
///
/// A select statement puts 1 waiter into each channel of its cases. The first
/// channel that reaches the waiter writes the number of the case here, and each
/// other waiter of that state then holds no value.
final class SelectState {
  /// The number of the case that a channel took, and -1 while the statement
  /// waits.
  int taken = -1;

  /// The value of a receive case that a channel took.
  Object? value;
  bool ok = false;

  bool get done => taken >= 0;
}

/// The frame of a goroutine that waits for a value of a channel.
final class _RecvWait<T> implements WaitFrame, DurableWait {
  final Goroutine g;
  final T Function() _zero;
  (T, bool)? value;

  /// The bubble of the channel of the wait.
  final Bubble? bubble;

  /// The select statement of the waiter, and null for a plain receive.
  final SelectState? sel;
  final int index;

  _RecvWait(this.g, this._zero, this.bubble, [this.sel, this.index = 0]);

  @override
  String get $reason => sel == null ? 'chan receive' : 'select';

  @override
  bool $durableIn(Bubble b) => identical(bubble, b);

  @override
  Object? resume() => value ?? (_zero(), false);
}

/// The frame of a goroutine that waits to write a value into a channel.
final class _SendWait<T> implements WaitFrame, DurableWait {
  final Goroutine g;
  final T value;

  /// The bubble of the channel of the wait.
  final Bubble? bubble;

  /// The select statement of the waiter, and null for a plain send.
  final SelectState? sel;
  final int index;

  _SendWait(this.g, this.value, this.bubble, [this.sel, this.index = 0]);

  @override
  String get $reason => sel == null ? 'chan send' : 'select';

  @override
  bool $durableIn(Bubble b) => identical(bubble, b);

  @override
  Object? resume() => null;
}

/// Runs a pure function in a new isolate and gives the result.
///
/// The 2 isolates share no memory, thus the function must not read a package
/// variable and must not write one. The input and the result cross the
/// boundary as bytes, because bytes cross at a low cost.
Slice<GoInt> $worker(
  Slice<GoInt> Function(Slice<GoInt>)? body,
  Slice<GoInt> input,
) {
  if (body == null) $panicNil();
  final Slice<GoInt> Function(Slice<GoInt>) run = body;
  final Uint8List bytes = Uint8List(input.len);
  for (var i = 0; i < input.len; i++) {
    bytes[i] = $ix(input.array[input.off + i]);
  }
  final Goroutine g = $current!;
  final EventWait waiting = EventWait();
  $holdEvent();
  unawaited(
    Isolate.run(() {
      final Slice<GoInt> result = run($gils(Slice.of(bytes)));
      final Uint8List copy = Uint8List(result.len);
      for (var i = 0; i < result.len; i++) {
        copy[i] = $ix(result.array[result.off + i]);
      }
      return copy;
    }).then((Uint8List out) {
      waiting.value = $gils(Slice.of(out));
      $freeEvent();
      $wake(g);
    }),
  );
  $block(waiting);
  return Slice.of(<GoInt>[]);
}

/// Gives up control for a time.
///
/// A goroutine of a bubble waits on the clock of that bubble. Such a sleep
/// takes no real time, exactly as Go does inside a bubble.
void $sleep(int nanoseconds) {
  if (nanoseconds <= 0) return;
  final Goroutine g = $current!;
  final Bubble? b = g.bubble;
  if (b != null) {
    $bubbleSleep(b, g, nanoseconds);
    return;
  }
  final EventWait waiting = EventWait();
  $holdEvent();
  Timer(Duration(microseconds: nanoseconds ~/ 1000), () {
    $freeEvent();
    $wake(g);
  });
  $block(waiting);
}

/// Provokes a collection and gives up control, thus each cleanup of a value
/// that no code reaches runs.
///
/// Dart holds no call that forces a collection. The function allocates a block
/// of garbage, and the collector of the Dart VM runs for it. A cleanup runs on
/// a later turn of the event loop, thus the goroutine waits for 1 timer. The
/// collector follows no rule of Go, thus a cleanup can stay for a later call.
void $gc() {
  for (var round = 0; round < 2; round++) {
    final List<Object> garbage = <Object>[];
    for (var i = 0; i < 20000; i++) {
      garbage.add(List<int>.filled(64, i));
    }
    garbage.clear();
  }
  final Goroutine g = $current!;
  final EventWait waiting = EventWait();
  $holdEvent();
  Timer(Duration.zero, () {
    $freeEvent();
    $wake(g);
  });
  $block(waiting);
}

/// The count of nanoseconds since the start of the Unix epoch.
///
/// A goroutine of a bubble reads the fake clock of that bubble.
int $nowNanos() {
  final Bubble? b = $current?.bubble;
  if (b != null) return b.now;
  return DateTime.now().microsecondsSinceEpoch * 1000;
}

/// The local zone at an instant of the Unix epoch.
///
/// The instant is a count of seconds, because a Time of the package time
/// reaches the year 1 and the count of nanoseconds of that instant does not
/// fit an int64.
///
/// The first value is the offset in seconds east of UTC, and the second value
/// is the name of the zone. Go writes both in the text of a time.
(int, GoString) $zoneOffset(int seconds) {
  final t = DateTime.fromMillisecondsSinceEpoch(seconds * 1000);
  return (t.timeZoneOffset.inSeconds, GoString.fromString(t.timeZoneName));
}

/// The clock of the program. It starts with the isolate.
final Stopwatch $clock = Stopwatch()..start();

/// The count of nanoseconds since the start of the program.
///
/// The value comes from a monotonic clock, thus a change of the system time
/// does not move it. The clock of the Dart VM ticks each nanosecond, and the
/// wall clock of DateTime holds only microseconds. A benchmark needs the finer
/// value.
int $monoNanos() {
  final int frequency = $clock.frequency;
  if (frequency == 1000000000) return $clock.elapsedTicks;
  return $clock.elapsedTicks * 1000000000 ~/ frequency;
}

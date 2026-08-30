import 'dart:async';
import 'dart:collection';
import 'dart:isolate';
import 'dart:typed_data';

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

  Chan(this.capacity, this._zero) : isNil = false;

  Chan.nil(this._zero) : capacity = 0, isNil = true;

  int get length => _buffer.length;

  bool get isClosed => _closed;

  /// True when a receive gives a value at this moment.
  bool get readyToReceive =>
      !isNil && (_buffer.isNotEmpty || _hasSender || _closed);

  /// True when a send gives up no control at this moment.
  bool get readyToSend =>
      !isNil && (_closed || _hasReceiver || _buffer.length < capacity);

  bool get _hasSender => _senders.any((_SendWait<T> w) => w.sel?.done != true);

  bool get _hasReceiver =>
      _receivers.any((_RecvWait<T> w) => w.sel?.done != true);

  /// Reads 1 value. The caller reads [$blocked] after the call.
  (T, bool) receive() {
    if (isNil) {
      $park();
      return (_zero(), false);
    }
    if (readyToReceive) return receiveNow();
    final _RecvWait<T> waiting = _RecvWait<T>($current!, _zero);
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
    _senders.add(_SendWait<T>(g, value, sel, index));
  }

  void armRecv(SelectState sel, int index, Goroutine g) {
    _receivers.add(_RecvWait<T>(g, _zero, sel, index));
  }

  /// Drops each waiter of a select statement that ended.
  void disarm(SelectState sel) {
    _senders.removeWhere((_SendWait<T> w) => identical(w.sel, sel));
    _receivers.removeWhere((_RecvWait<T> w) => identical(w.sel, sel));
  }

  /// Writes 1 value. The caller reads [$blocked] after the call.
  void send(T value) {
    if (isNil) {
      $park();
      return;
    }
    if (readyToSend) {
      sendNow(value);
      return;
    }
    final _SendWait<T> waiting = _SendWait<T>($current!, value);
    _senders.add(waiting);
    $block(waiting);
  }

  /// Sends with no wait. The caller reads readyToSend first.
  void sendNow(T value) {
    if (_closed) $panic(const GoError('send on closed channel'));
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
    if (isNil) $panic(const GoError('close of nil channel'));
    if (_closed) $panic(const GoError('close of closed channel'));
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
final class _RecvWait<T> implements Frame {
  final Goroutine g;
  final T Function() _zero;
  (T, bool)? value;

  /// The select statement of the waiter, and null for a plain receive.
  final SelectState? sel;
  final int index;

  _RecvWait(this.g, this._zero, [this.sel, this.index = 0]);

  @override
  Object? resume() => value ?? (_zero(), false);
}

/// The frame of a goroutine that waits to write a value into a channel.
final class _SendWait<T> implements Frame {
  final Goroutine g;
  final T value;

  /// The select statement of the waiter, and null for a plain send.
  final SelectState? sel;
  final int index;

  _SendWait(this.g, this.value, [this.sel, this.index = 0]);

  @override
  Object? resume() => null;
}

/// Runs a pure function in a new isolate and gives the result.
///
/// The 2 isolates share no memory, thus the function must not read a package
/// variable and must not write one. The input and the result cross the
/// boundary as bytes, because bytes cross at a low cost.
Slice<int> $worker(Slice<int> Function(Slice<int>)? body, Slice<int> input) {
  if (body == null) $panicNil();
  final Slice<int> Function(Slice<int>) run = body;
  final Uint8List bytes = Uint8List(input.len);
  for (var i = 0; i < input.len; i++) {
    bytes[i] = input.array[input.off + i];
  }
  final Goroutine g = $current!;
  final EventWait waiting = EventWait();
  $holdEvent();
  unawaited(
    Isolate.run(() {
      final Slice<int> result = run(Slice.of(bytes));
      final Uint8List copy = Uint8List(result.len);
      for (var i = 0; i < result.len; i++) {
        copy[i] = result.array[result.off + i];
      }
      return copy;
    }).then((Uint8List out) {
      waiting.value = Slice.of(out);
      $freeEvent();
      $wake(g);
    }),
  );
  $block(waiting);
  return Slice.of(Uint8List(0));
}

/// Gives up control for a time.
void $sleep(int nanoseconds) {
  if (nanoseconds <= 0) return;
  final Goroutine g = $current!;
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
int $nowNanos() => DateTime.now().microsecondsSinceEpoch * 1000;

/// The local zone at an instant of the Unix epoch.
///
/// The first value is the offset in seconds east of UTC, and the second value
/// is the name of the zone. Go writes both in the text of a time.
(int, GoString) $zoneOffset(int nanos) {
  final t = DateTime.fromMicrosecondsSinceEpoch(nanos ~/ 1000);
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

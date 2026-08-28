import 'dart:async';
import 'dart:collection';
import 'dart:isolate';
import 'dart:typed_data';

import 'panic.dart';
import 'sched.dart';
import 'slice.dart';

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
  late final List<void Function()> _watchers = <void Function()>[];
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
      !isNil && (_buffer.isNotEmpty || _senders.isNotEmpty || _closed);

  /// True when a send gives up no control at this moment.
  bool get readyToSend =>
      !isNil && (_closed || _receivers.isNotEmpty || _buffer.length < capacity);

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
      if (_senders.isNotEmpty) {
        final _SendWait<T> waiting = _senders.removeFirst();
        _buffer.add(waiting.value);
        $ready(waiting.g);
      }
      _notify();
      return (value, true);
    }
    if (_senders.isNotEmpty) {
      final _SendWait<T> waiting = _senders.removeFirst();
      $ready(waiting.g);
      _notify();
      return (waiting.value, true);
    }
    return (_zero(), false);
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
    if (_receivers.isNotEmpty) {
      final _RecvWait<T> waiting = _receivers.removeFirst();
      waiting.value = (value, true);
      $ready(waiting.g);
      _notify();
      return;
    }
    _buffer.add(value);
    _notify();
  }

  void close() {
    if (isNil) $panic(const GoError('close of nil channel'));
    if (_closed) $panic(const GoError('close of closed channel'));
    _closed = true;
    while (_receivers.isNotEmpty) {
      final _RecvWait<T> waiting = _receivers.removeFirst();
      waiting.value = (_zero(), false);
      $ready(waiting.g);
    }
    _notify();
  }

  void watch(void Function() notify) => _watchers.add(notify);

  void unwatch(void Function() notify) => _watchers.remove(notify);

  void _notify() {
    if (_watchers.isEmpty) return;
    for (final notify in List<void Function()>.of(_watchers)) {
      notify();
    }
  }
}

/// The frame of a goroutine that waits for a value of a channel.
final class _RecvWait<T> implements Frame {
  final Goroutine g;
  final T Function() _zero;
  (T, bool)? value;

  _RecvWait(this.g, this._zero);

  @override
  Object? resume() => value ?? (_zero(), false);
}

/// The frame of a goroutine that waits to write a value into a channel.
final class _SendWait<T> implements Frame {
  final Goroutine g;
  final T value;

  _SendWait(this.g, this.value);

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

/// The count of nanoseconds since the start of the Unix epoch.
int $nowNanos() => DateTime.now().microsecondsSinceEpoch * 1000;

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

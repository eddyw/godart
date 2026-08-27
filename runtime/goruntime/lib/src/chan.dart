import 'dart:async';
import 'dart:collection';

import 'panic.dart';

/// A Go channel.
///
/// All goroutines run in 1 Dart isolate and give up control at an await
/// expression, thus the queues need no lock.
final class Chan<T> {
  final int capacity;
  final T Function() _zero;

  final ListQueue<T> _buffer = ListQueue<T>();
  final ListQueue<Completer<(T, bool)>> _receivers =
      ListQueue<Completer<(T, bool)>>();
  final ListQueue<_Send<T>> _senders = ListQueue<_Send<T>>();
  final List<void Function()> _watchers = <void Function()>[];
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

  Future<void> send(T value) {
    if (isNil) return Completer<void>().future;
    if (readyToSend) {
      sendNow(value);
      return Future<void>.value();
    }
    final _Send<T> waiting = _Send<T>(value, Completer<void>());
    _senders.add(waiting);
    return waiting.done.future;
  }

  /// Sends without a wait. The caller reads readyToSend first.
  void sendNow(T value) {
    if (_closed) $panic(const GoError('send on closed channel'));
    if (_receivers.isNotEmpty) {
      _receivers.removeFirst().complete((value, true));
      _wake();
      return;
    }
    _buffer.add(value);
    _wake();
  }

  Future<(T, bool)> receive() {
    if (isNil) return Completer<(T, bool)>().future;
    if (readyToReceive) return Future<(T, bool)>.value(receiveNow());
    final Completer<(T, bool)> waiting = Completer<(T, bool)>();
    _receivers.add(waiting);
    return waiting.future;
  }

  /// Receives without a wait. The caller reads readyToReceive first.
  (T, bool) receiveNow() {
    if (_buffer.isNotEmpty) {
      final T value = _buffer.removeFirst();
      if (_senders.isNotEmpty) {
        final _Send<T> waiting = _senders.removeFirst();
        _buffer.add(waiting.value);
        waiting.done.complete();
      }
      _wake();
      return (value, true);
    }
    if (_senders.isNotEmpty) {
      final _Send<T> waiting = _senders.removeFirst();
      waiting.done.complete();
      _wake();
      return (waiting.value, true);
    }
    return (_zero(), false);
  }

  void close() {
    if (isNil) $panic(const GoError('close of nil channel'));
    if (_closed) $panic(const GoError('close of closed channel'));
    _closed = true;
    while (_receivers.isNotEmpty) {
      _receivers.removeFirst().complete((_zero(), false));
    }
    _wake();
  }

  void watch(void Function() notify) => _watchers.add(notify);

  void unwatch(void Function() notify) => _watchers.remove(notify);

  void _wake() {
    if (_watchers.isEmpty) return;
    for (final notify in List<void Function()>.of(_watchers)) {
      notify();
    }
  }
}

final class _Send<T> {
  final T value;
  final Completer<void> done;

  _Send(this.value, this.done);
}

/// Gives up control for a time. A Go sleep blocks the goroutine, thus the
/// caller becomes an async function.
Future<void> $sleep(int nanoseconds) {
  if (nanoseconds <= 0) return Future<void>.value();
  return Future<void>.delayed(Duration(microseconds: nanoseconds ~/ 1000));
}

/// The count of nanoseconds since the start of the Unix epoch.
int $nowNanos() => DateTime.now().microsecondsSinceEpoch * 1000;

/// Starts a goroutine. Go does not run the body at the point of the statement,
/// thus the body waits for the event loop.
void $go(FutureOr<void> Function() body) {
  scheduleMicrotask(() async {
    try {
      await body();
    } on GoPanic catch (p) {
      $goPanic(p);
    }
  });
}

/// The report of a panic that leaves a goroutine. The build driver sets it,
/// because the report needs the entry point of the program.
void Function(GoPanic) $goPanic = (GoPanic p) {
  throw p;
};

import 'chan.dart';
import 'sched.dart';

/// One case of a select statement.
sealed class SelectCase {
  bool get ready;

  void watch(void Function() notify);

  void unwatch(void Function() notify);
}

/// A case that receives 1 value.
final class RecvCase<T> extends SelectCase {
  final Chan<T> chan;

  /// The value that the case reads. The body of the case takes it.
  (T, bool)? result;

  RecvCase(this.chan);

  @override
  bool get ready => chan.readyToReceive;

  @override
  void watch(void Function() notify) => chan.watch(notify);

  @override
  void unwatch(void Function() notify) => chan.unwatch(notify);

  void take() => result = chan.receiveNow();
}

/// A case that sends 1 value.
final class SendCase<T> extends SelectCase {
  final Chan<T> chan;
  final T value;

  SendCase(this.chan, this.value);

  @override
  bool get ready => chan.readyToSend;

  @override
  void watch(void Function() notify) => chan.watch(notify);

  @override
  void unwatch(void Function() notify) => chan.unwatch(notify);

  void take() => chan.sendNow(value);
}

/// Runs a select statement and gives the number of the case that it takes.
///
/// The result is -1 when no case is ready and the statement holds a default.
/// The caller reads [$blocked] after the call, because the statement blocks
/// when no case is ready and the statement holds no default.
///
/// Go takes a ready case at random, and this function takes the first one. A
/// Go program must not depend on the choice.
int $select(List<SelectCase> cases, bool hasDefault) {
  for (var i = 0; i < cases.length; i++) {
    if (!cases[i].ready) continue;
    _take(cases[i]);
    return i;
  }
  if (hasDefault) return -1;
  if (cases.isEmpty) {
    $park();
    return -1;
  }
  _SelectWait(cases, $current!).arm();
  return -1;
}

void _take(SelectCase c) {
  switch (c) {
    case RecvCase():
      c.take();
    case SendCase():
      c.take();
  }
}

/// The frame of a goroutine that waits for 1 case of a select statement.
///
/// A wake runs the select statement again. A second goroutine can take the
/// value first, thus the state of each case needs a new read.
final class _SelectWait implements Frame {
  final List<SelectCase> cases;
  final Goroutine g;

  _SelectWait(this.cases, this.g);

  void arm() {
    for (final c in cases) {
      c.watch(_onReady);
    }
    $block(this);
  }

  void _onReady() {
    for (final c in cases) {
      c.unwatch(_onReady);
    }
    $ready(g);
  }

  @override
  Object? resume() => $select(cases, false);
}

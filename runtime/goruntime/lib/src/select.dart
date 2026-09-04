import 'chan.dart';
import 'sched.dart';

/// One case of a select statement.
sealed class SelectCase {
  bool get ready;

  /// The bubble of the channel of the case, and null outside a bubble.
  Bubble? get bubble;

  /// Puts a waiter of the statement into the channel of the case.
  void arm(SelectState sel, int index, Goroutine g);

  /// Drops the waiter of the statement.
  void disarm(SelectState sel);
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
  Bubble? get bubble => chan.bubble;

  @override
  void arm(SelectState sel, int index, Goroutine g) =>
      chan.armRecv(sel, index, g);

  @override
  void disarm(SelectState sel) => chan.disarm(sel);

  void take() => result = chan.receiveNow();

  /// Writes the value that a channel gave to the waiter of the statement.
  void give(SelectState sel) => result = (sel.value as T, sel.ok);
}

/// A case that sends 1 value.
final class SendCase<T> extends SelectCase {
  final Chan<T> chan;
  final T value;

  SendCase(this.chan, this.value);

  @override
  bool get ready => chan.readyToSend;

  @override
  Bubble? get bubble => chan.bubble;

  @override
  void arm(SelectState sel, int index, Goroutine g) =>
      chan.armSend(sel, index, g, value);

  @override
  void disarm(SelectState sel) => chan.disarm(sel);

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
    final Bubble? b = cases[i].bubble;
    if (b != null && !identical(b, $current?.bubble)) {
      $fatalError('select on synctest channel from outside bubble');
    }
    if (!cases[i].ready) continue;
    _take(cases[i]);
    return i;
  }
  if (hasDefault) return -1;
  if (cases.isEmpty) {
    $park('select (no cases)');
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
/// The statement puts 1 waiter into each channel of its cases. A channel that
/// reaches such a waiter writes the number of the case, thus 2 select
/// statements meet each other. A wake with no such number runs the statement
/// again, because a second goroutine can take the value first.
final class _SelectWait implements WaitFrame, DurableWait {
  final List<SelectCase> cases;
  final Goroutine g;
  final SelectState state = SelectState();

  _SelectWait(this.cases, this.g);

  @override
  String get $reason => 'select';

  /// Go names the wait durable only when each case reads a channel of the
  /// bubble. A case of another channel can end the wait from outside.
  @override
  bool $durableIn(Bubble b) {
    for (final SelectCase c in cases) {
      if (!identical(c.bubble, b)) return false;
    }
    return true;
  }

  void arm() {
    for (var i = 0; i < cases.length; i++) {
      cases[i].arm(state, i, g);
      // A wake of another goroutine can take a case at this point, thus each
      // later case stays out.
      if (state.done) break;
    }
    $block(this);
  }

  @override
  Object? resume() {
    for (final c in cases) {
      c.disarm(state);
    }
    if (!state.done) return $select(cases, false);
    final SelectCase taken = cases[state.taken];
    if (taken is RecvCase) {
      taken.give(state);
    }
    return state.taken;
  }
}

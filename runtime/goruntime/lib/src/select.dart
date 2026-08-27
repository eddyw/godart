import 'dart:async';

import 'chan.dart';

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
/// Go takes a ready case at random, and this function takes the first one. A
/// Go program must not depend on the choice.
Future<int> $select(List<SelectCase> cases, bool hasDefault) async {
  while (true) {
    for (var i = 0; i < cases.length; i++) {
      if (!cases[i].ready) continue;
      _take(cases[i]);
      return i;
    }
    if (hasDefault) return -1;
    if (cases.isEmpty) return await Completer<int>().future;
    final Completer<void> woke = Completer<void>();
    void notify() {
      if (!woke.isCompleted) woke.complete();
    }

    for (final c in cases) {
      c.watch(notify);
    }
    await woke.future;
    for (final c in cases) {
      c.unwatch(notify);
    }
  }
}

void _take(SelectCase c) {
  switch (c) {
    case RecvCase():
      c.take();
    case SendCase():
      c.take();
  }
}

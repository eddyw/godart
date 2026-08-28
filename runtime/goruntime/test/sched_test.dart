import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

/// The 2 functions below hold the shape that stage 5 writes for a function
/// that can block. Each one keeps its locals in a frame class, and it gives up
/// control with a plain return.

final class _SendFrame implements Frame {
  final Chan<int> ch;
  final List<int> log;
  int s = 0;
  bool c = false;
  Frame? r;
  int i = 0;

  _SendFrame(this.ch, this.log);

  @override
  Object? resume() {
    _send(ch, log, this);
    return null;
  }
}

void _send(Chan<int> ch, List<int> log, [_SendFrame? f]) {
  int s = 0;
  bool c = false;
  Frame? r;
  int i = 0;
  if (f != null) {
    s = f.s;
    c = f.c;
    r = f.r;
    i = f.i;
  }
  machine:
  while (true) {
    switch (s) {
      case 0:
        i = 0;
        s = 1;
        continue machine;
      case 1:
        if (!(i < 3)) {
          s = 4;
          continue machine;
        }
        ch.send(i);
        s = 2;
        continue machine;
      case 2:
        if (c) {
          c = false;
          r!.resume();
        }
        if ($blocked != null) break machine;
        log.add(i);
        i = i + 1;
        s = 1;
        continue machine;
      case 4:
        ch.close();
        return;
    }
  }
  $block(
    _SendFrame(ch, log)
      ..s = s
      ..c = true
      ..r = $blocked
      ..i = i,
  );
}

final class _RecvFrame implements Frame {
  final Chan<int> ch;
  final List<int> out;
  int s = 0;
  bool c = false;
  Frame? r;
  (int, bool) t = (0, false);

  _RecvFrame(this.ch, this.out);

  @override
  Object? resume() {
    _recv(ch, out, this);
    return null;
  }
}

void _recv(Chan<int> ch, List<int> out, [_RecvFrame? f]) {
  int s = 0;
  bool c = false;
  Frame? r;
  (int, bool) t = (0, false);
  if (f != null) {
    s = f.s;
    c = f.c;
    r = f.r;
    t = f.t;
  }
  machine:
  while (true) {
    switch (s) {
      case 0:
        t = ch.receive();
        s = 1;
        continue machine;
      case 1:
        if (c) {
          c = false;
          t = r!.resume() as (int, bool);
        }
        if ($blocked != null) break machine;
        if (!t.$2) return;
        out.add(t.$1);
        s = 0;
        continue machine;
    }
  }
  $block(
    _RecvFrame(ch, out)
      ..s = s
      ..c = true
      ..r = $blocked
      ..t = t,
  );
}

void main() {
  group('the unwind model', () {
    test('an unbuffered channel hands each value over', () {
      final Chan<int> ch = Chan<int>(0, () => 0);
      final List<int> sent = <int>[];
      final List<int> got = <int>[];
      $go(() => _send(ch, sent));
      $go(() => _recv(ch, got));
      $schedule();
      expect(sent, <int>[0, 1, 2]);
      expect(got, <int>[0, 1, 2]);
    });

    test('a buffered channel gives up no control while it has room', () {
      final Chan<int> ch = Chan<int>(4, () => 0);
      final List<int> sent = <int>[];
      $go(() => _send(ch, sent));
      $schedule();
      expect(sent, <int>[0, 1, 2]);
      expect(ch.length, 3);
      expect(ch.isClosed, isTrue);
    });

    test('a receive on a closed channel gives the zero value', () {
      final Chan<int> ch = Chan<int>(1, () => 0);
      ch.close();
      final List<int> got = <int>[];
      $go(() => _recv(ch, got));
      $schedule();
      expect(got, isEmpty);
      expect($blocked, isNull);
    });

    test('a select with a default takes no case when no channel is ready', () {
      final Chan<int> ch = Chan<int>(0, () => 0);
      final int taken = $select(<SelectCase>[RecvCase<int>(ch)], true);
      expect(taken, -1);
      expect($blocked, isNull);
    });

    test('a select takes the case that is ready', () {
      final Chan<int> ch = Chan<int>(1, () => 0);
      ch.sendNow(7);
      final RecvCase<int> one = RecvCase<int>(ch);
      final int taken = $select(<SelectCase>[one], false);
      expect(taken, 0);
      expect(one.result, (7, true));
    });
  });
}

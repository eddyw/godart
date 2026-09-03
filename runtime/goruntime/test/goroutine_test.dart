import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

/// These cases cover the handle that `$go` gives to a Dart host. A goroutine
/// ends 3 ways, and the handle reports each 1: the body returns, a panic
/// leaves the body, and `runtime.Goexit` stops the goroutine.

/// _AnswerFrame holds the shape that stage 5 writes for a Go function of the
/// type `func() bool` that blocks on a channel receive.
final class _AnswerFrame implements Frame {
  final Chan<bool> ch;
  bool c = false;
  Frame? r;
  (bool, bool) t = (false, false);

  _AnswerFrame(this.ch);

  @override
  Object? resume() => _answer(ch, this);
}

bool _answer(Chan<bool> ch, [_AnswerFrame? f]) {
  bool c = false;
  Frame? r;
  (bool, bool) t = (false, false);
  if (f != null) {
    c = f.c;
    r = f.r;
    t = f.t;
  }
  if (c) {
    t = r!.resume() as (bool, bool);
  } else {
    t = ch.receive();
  }
  if ($blocked != null) {
    $block(
      _AnswerFrame(ch)
        ..c = true
        ..r = $blocked
        ..t = t,
    );
    return false;
  }
  return t.$1;
}

void main() {
  // The cases drive the scheduler as a Dart host does, thus the deadlock
  // report stays off. A goroutine that a panic ended also stays in the live
  // set, exactly as it does in a real program.
  setUpAll(() => $hosted = true);
  tearDownAll(() => $hosted = false);

  group('the handle of a goroutine', () {
    test('reports a body that returns, and it gives the value', () {
      bool ran = false;
      final Goroutine g = $go(() => 7);
      g.whenEnd(() => ran = true);
      expect(g.ended, isFalse);
      $schedule();
      expect(ran, isTrue);
      expect(g.ended, isTrue);
      expect(g.result, 7);
    });

    test('runs a hook that a host adds after the end', () {
      final Goroutine g = $go(() {});
      $schedule();
      bool ran = false;
      g.whenEnd(() => ran = true);
      expect(ran, isTrue);
    });

    test('reports a body that blocks, and it gives the value of the frame', () {
      final Chan<bool> ch = Chan<bool>(0, () => false);
      bool ran = false;
      final Goroutine g = $go(() => _answer(ch));
      g.whenEnd(() => ran = true);
      $schedule();
      expect(ran, isFalse, reason: 'the body waits for the channel');
      expect(g.ended, isFalse);

      ch.sendNow(true);
      $schedule();
      expect(ran, isTrue);
      expect(g.ended, isTrue);
      expect(g.result, isTrue);
    });

    test('reports a goroutine that runtime.Goexit stops', () {
      bool ran = false;
      final Goroutine g = $go(() => throw const GoStop());
      g.whenEnd(() => ran = true);
      $schedule();
      expect(ran, isTrue);
      expect(g.ended, isTrue);
      expect(g.result, isNull);
    });

    // The panic leaves the goroutine in the live set, thus this case stands
    // last. Each case above needs a live set that holds no dead goroutine.
    test('reports a body that panics', () {
      final void Function(GoPanic, StackTrace) saved = $goPanic;
      Object? seen;
      $goPanic = (GoPanic p, StackTrace trace) => seen = p.value;
      bool ran = false;
      final Goroutine g = $go(() => throw GoPanic('boom'));
      g.whenEnd(() => ran = true);
      $schedule();
      $goPanic = saved;
      expect(ran, isTrue);
      expect(g.ended, isTrue);
      expect(g.result, isNull);
      expect(seen, 'boom');
    });
  });
}

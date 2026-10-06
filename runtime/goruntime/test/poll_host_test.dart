import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

final class _PollFrame implements Frame {
  final bool Function() stopped;
  Frame? pending;
  int polls = 0;

  _PollFrame(this.stopped);

  @override
  Object? resume() {
    pending?.resume();
    pending = null;
    while (!stopped()) {
      polls++;
      $poll();
      if ($blocked != null) {
        pending = $blocked;
        $block(this);
        return null;
      }
    }
    return polls;
  }
}

void main() {
  test('host timers progress while a runnable goroutine polls', () async {
    $hosted = true;
    var stopped = false;
    var peerRan = false;
    var timerSawActive = false;
    final frame = _PollFrame(() => stopped);
    final worker = $go(frame.resume);
    $go(() => peerRan = true);
    final ended = Completer<void>();
    worker.whenEnd(ended.complete);
    final timer = Timer(const Duration(milliseconds: 30), () {
      timerSawActive = !worker.ended && peerRan;
      stopped = true;
    });
    try {
      $schedule();
      await ended.future.timeout(const Duration(seconds: 2));
      expect(timerSawActive, isTrue);
      expect(worker.result, greaterThan(128));
    } finally {
      timer.cancel();
      $hosted = false;
    }
  });
}

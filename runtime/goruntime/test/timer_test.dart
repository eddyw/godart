import 'dart:async';

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

final class _ManualTimer implements Timer {
  final void Function() callback;

  @override
  bool isActive = true;

  @override
  int tick = 0;

  _ManualTimer(this.callback);

  void fire() {
    isActive = false;
    tick++;
    callback();
  }

  @override
  void cancel() => isActive = false;
}

void main() {
  test('a long timer remains cancellable after an interval', () {
    final timers = <_ManualTimer>[];
    final delays = <Duration>[];
    var called = false;
    runZoned(
      () {
        final handle = $startTimer(0xfffffffe * 1000000, () => called = true);
        expect(delays, [const Duration(milliseconds: 0x7fffffff)]);
        timers.single.fire();
        expect(called, isFalse);
        expect(timers, hasLength(2));
        expect(delays.last, lessThanOrEqualTo(delays.first));
        expect($stopTimer(handle), isTrue);
        expect(timers.last.isActive, isFalse);
        expect($stopTimer(handle), isFalse);
      },
      zoneSpecification: ZoneSpecification(
        createTimer: (self, parent, zone, delay, run) {
          delays.add(delay);
          final timer = _ManualTimer(run);
          timers.add(timer);
          return timer;
        },
      ),
    );
  });

  test('a short timer runs its callback and releases its handle', () {
    late _ManualTimer timer;
    var called = false;
    runZoned(
      () {
        final handle = $startTimer(10000000, () => called = true);
        timer.fire();
        expect(called, isTrue);
        expect($stopTimer(handle), isFalse);
      },
      zoneSpecification: ZoneSpecification(
        createTimer: (self, parent, zone, delay, run) {
          expect(delay, const Duration(milliseconds: 10));
          return timer = _ManualTimer(run);
        },
      ),
    );
  });
}

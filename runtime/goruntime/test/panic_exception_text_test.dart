import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

void main() {
  for (final method in ['Error()string', 'String()string']) {
    test('Dart exception text preserves Go state for $method', () {
      var calls = 0;
      final descriptor = TypeDesc('fixture.text', 'struct', {
        method: (Object? value) {
          calls++;
          return GoString.fromString('Go method text');
        },
      });
      final value = Iface(descriptor, Object());
      final panic = GoPanic(value);
      expect(panic.toString(), startsWith('panic: '));
      try {
        Error.throwWithStackTrace(panic, panic.trace);
      } on GoPanic catch (error, trace) {
        expect(error, same(panic));
        expect(trace.toString(), isNotEmpty);
      }
      expect(calls, 0);
      expect($blocked, isNull);
      expect($panicText(value), 'Go method text');
      expect(calls, 1);
    });
  }
}

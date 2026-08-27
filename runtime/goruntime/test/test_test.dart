import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

void main() {
  group('the stop signal', () {
    test('catch reports false when the body stops', () async {
      expect(await $catch(() => $stop()), isFalse);
    });

    test('catch reports true when the body returns', () async {
      expect(await $catch(() {}), isTrue);
    });

    test('catch holds an async body', () async {
      expect(await $catch(() async => $stop()), isFalse);
      expect(await $catch(() async {}), isTrue);
    });

    test('a deferred call runs on the way out', () async {
      final List<String> order = <String>[];
      await $catch(() {
        final DeferStack defer = DeferStack();
        try {
          defer.push(() => order.add('defer'));
          order.add('body');
          $stop();
        } finally {
          defer.run();
        }
      });
      expect(order, <String>['body', 'defer']);
    });

    test('a panic passes through', () async {
      expect(
        () => $catch(() => $panic(GoString.fromString('stop'))),
        throwsA(isA<GoPanic>()),
      );
    });

    test('catch panics on a nil function', () async {
      expect(() => $catch(null), throwsA(isA<GoPanic>()));
    });
  });

  group('the program arguments', () {
    test('args gives the list that the entry point wrote', () {
      $setArgs(<String>['one', 'two']);
      final Slice<GoString> got = $args();
      expect(got.len, 2);
      expect(got.array[0].toDartString(), 'one');
      expect(got.array[1].toDartString(), 'two');
    });
  });
}

import 'dart:io';

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

final class _Call implements Frame {
  final Object? Function() action;
  final void Function(Object?) complete;
  Frame? pending;

  _Call(this.action, this.complete);

  @override
  Object? resume() {
    final result = pending == null ? action() : pending!.resume();
    if ($blocked != null) {
      pending = $blocked;
      $block(this);
    } else {
      complete(result);
    }
    return null;
  }
}

Future<T> _invoke<T>(Object? Function() action, T Function(Object?) complete) {
  final done = Completer<T>();
  final frame = _Call(action, (value) => done.complete(complete(value)));
  $go(frame.resume);
  $schedule();
  return done.future;
}

void main() {
  test('a closed Unix listener cannot remove its replacement', () async {
    final directory = Directory.systemTemp.createTempSync('gart-close-');
    addTearDown(() => directory.deleteSync(recursive: true));
    final path = '${directory.path}/sock';
    for (var i = 0; i < 1000; i++) {
      final handle = await _invoke(
        () => $listenUnix(GoString.fromString(path)),
        (value) {
          final (handle, error) = value as (int, GoString);
          expect(error.toDartString(), isEmpty);
          return handle;
        },
      );
      await _invoke(() => $listenClose(handle), (value) {
        File(path).writeAsStringSync('replacement');
        expect((value as GoString).toDartString(), isEmpty);
      });
      await Future<void>.delayed(const Duration(milliseconds: 1));
      expect(File(path).existsSync(), isTrue, reason: 'iteration $i');
      File(path).deleteSync();
    }
  }, skip: !(Platform.isLinux || Platform.isMacOS));
}

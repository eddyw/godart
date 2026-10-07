import 'dart:async';
import 'dart:io';

import 'package:goruntime/goruntime.dart';
import 'package:goruntime/src/native_errno.dart';
import 'package:goruntime/src/native_process.dart';

Future<void> main(List<String> arguments) async {
  final environment = Slice.of(
    Platform.environment.entries
        .map((entry) => GoString.fromString('${entry.key}=${entry.value}'))
        .toList(),
  );
  final foreign = await Process.start('/bin/sh', ['-c', 'sleep 2']);
  try {
    for (final scenario in {
      'exit 0': 0,
      'exit 17': 17,
      'exit 127': 127,
      r'kill -TERM $$': -15,
    }.entries) {
      final child = await NativeChild.start(
        Slice.of(
          ['/bin/sh', '-c', scenario.key].map(GoString.fromString).toList(),
        ),
        GoString.empty,
        environment,
      );
      try {
        await child.closeInput();
        {
          // A separate wait consumes status before the owned wait uses native metadata.
          while (true) {
            final state = nativeProcessWait(child.pid, -1);
            if (state.complete != 0 || state.error == 10) break;
            if (state.error != 0 && state.error != 4) {
              throw StateError('external native wait errno ${state.error}');
            }
            await Future<void>.delayed(const Duration(milliseconds: 1));
          }
        }
        int code;
        try {
          code = await child.exitCode;
        } on ProcessException catch (error) {
          throw StateError(
            'scenario ${scenario.key}, native wait errno ${error.errorCode}',
          );
        }
        if (code != scenario.value) {
          throw StateError('actual status $code, expected ${scenario.value}');
        }
      } finally {
        child.closeOutput(false);
        child.closeOutput(true);
      }
    }
    if (await foreign.exitCode != 0) throw StateError('Dart child exit');
    print('native process statuses pass');
  } finally {
    foreign.kill();
    await foreign.exitCode;
  }
}

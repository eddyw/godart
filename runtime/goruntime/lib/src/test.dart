import 'dart:async';
import 'dart:io';

import 'panic.dart';

/// The signal of `testing.T.FailNow`.
///
/// Go stops such a call chain with `runtime.Goexit`, and each deferred
/// function of the chain still runs. A throw gives an equal result here,
/// because the subset has no `recover`.
final class GoStop implements Exception {
  const GoStop();

  @override
  String toString() => 'the goroutine stopped';
}

/// Stops the call chain. Each deferred function of the chain still runs.
void $stop() => throw const GoStop();

/// Calls f and holds a stop signal.
///
/// It reports false when f called [$stop], and true in each other case. A
/// panic passes through, thus a test binary ends on a panic exactly as Go does.
Future<bool> $catch(FutureOr<void> Function()? f) async {
  if (f == null) $panicNil();
  final FutureOr<void> Function() body = f;
  try {
    await body();
    return true;
  } on GoStop {
    return false;
  }
}

/// Ends the program with a status code.
void $exit(int code) => exit(code);

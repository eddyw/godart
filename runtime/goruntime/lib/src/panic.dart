import 'iface.dart';
import 'string.dart';

/// The value of a Go panic.
final class GoPanic implements Exception {
  final Object? value;

  GoPanic(this.value);

  @override
  String toString() => 'panic: $value';
}

/// A run time error of the Go runtime, for example an index out of range.
final class GoError {
  final String message;

  const GoError(this.message);

  @override
  String toString() => 'runtime error: $message';
}

Never $panic(Object? value) => throw GoPanic(value);

Never $panicIndex(int index, int length) => throw GoPanic(
  Iface(
    $typeGoError,
    GoError('index out of range [$index] with length $length'),
  ),
);

Never $panicSlice(int low, int high, int capacity) => throw GoPanic(
  Iface(
    $typeGoError,
    GoError('slice bounds out of range [$low:$high] with capacity $capacity'),
  ),
);

Never $panicDivide() =>
    throw GoPanic(Iface($typeGoError, const GoError('integer divide by zero')));

Never $panicNil() => throw GoPanic(
  Iface(
    $typeGoError,
    const GoError('invalid memory address or nil pointer dereference'),
  ),
);

/// Reads a value and drops it. The Go statement `_ = x` runs the read, thus a
/// bounds check still applies.
void $drop(Object? value) {}

/// The deferred calls of 1 function. Go runs them in reverse order when the
/// function returns, and also when a panic leaves the function.
final class DeferStack {
  final List<void Function()> _calls = <void Function()>[];

  void push(void Function() call) => _calls.add(call);

  /// True when the stack holds no call.
  bool get isEmpty => _calls.isEmpty;

  /// Runs each call for the chain of a normal return. A panic inside a call
  /// becomes the chain panic, and each call that remains runs for it. A panic
  /// that no call recovered leaves the function again.
  void run() {
    runChain(null);
  }

  /// Runs the last call and drops it. A machine whose deferred calls can block
  /// reads it, because such a call gives up control between 2 entries. A
  /// recover inside the call reads the chain panic, and each other read gives
  /// null.
  void runNext() {
    if (_calls.isEmpty) return;
    try {
      _calls.removeLast()();
    } catch (e) {
      if (e is GoPanic) {
        // A panic inside a deferred call replaces the chain panic, and the
        // calls that remain run for the new panic.
        $chainPanic = e;
        $recovered = false;
      } else {
        rethrow;
      }
    }
  }

  /// Runs each call for the chain panic [start], and gives true when a call
  /// recovered it. A panic inside a call replaces the chain panic, and each
  /// call that remains runs for the new panic. A panic that no call recovered
  /// leaves the function again.
  bool runRecovered() {
    $savedChain = $chainPanic;
    $savedRec = $recovered;
    $recovered = false;
    Object? thrown;
    // The loop consumes each call, thus a second run of the chain finds an
    // empty stack and does nothing. The return path and the exit path of 1
    // function can both reach the chain.
    while (_calls.isNotEmpty) {
      try {
        _calls.removeLast()();
      } catch (e) {
        if (e is GoPanic) {
          $chainPanic = e;
          $recovered = false;
          thrown = e;
        } else {
          $chainRestore();
          rethrow;
        }
      }
    }
    final rec = $recovered;
    $chainRestore();
    if (!rec && thrown != null) throw thrown;
    return rec;
  }

  /// Runs each call for the chain panic of the function, which [start] gives.
  bool runChain(GoPanic? start) {
    $chainPanic = start;
    return runRecovered();
  }
}

/// The panic that the chain of deferred calls of 1 function handles. Null on a
/// chain that a normal return starts.
GoPanic? $chainPanic;

/// True when a deferred call of the chain recovered the chain panic.
bool $recovered = false;

/// The state of the chain that holds this one. A deferred call can hold a
/// chain of its own, and its end gives the state back.
GoPanic? $savedChain;
bool $savedRec = false;

/// Opens the chain of the deferred calls of 1 function for a panic that
/// reached it. A panic of another class runs the chain too, and recover gives
/// null for it.
void $chainSave(Object e) {
  $savedChain = $chainPanic;
  $savedRec = $recovered;
  $recovered = false;
  $chainPanic = e is GoPanic ? e : null;
}

/// Closes the chain and gives the state of the chain that holds this one back.
void $chainRestore() {
  $chainPanic = $savedChain;
  $recovered = $savedRec;
}

/// The token of the function that the chain of the deferred calls runs now.
/// Go gives a recover the value only in the function that the defer pushed,
/// and the token tells such a function from each function that it calls.
int $pendingTok = 0;

/// The value of recover. A call outside the deferred function of a panicking
/// function gives null.
Iface? $recover(int tok) {
  final GoPanic? p = $chainPanic;
  if (p == null || tok == 0 || tok != $pendingTok) return null;
  $recovered = true;
  return p.value as Iface?;
}

/// Runs the body of 1 deferred call under the token of its function.
T $dcall<T>(int tok, T Function() f) {
  final saved = $pendingTok;
  $pendingTok = tok;
  try {
    return f();
  } finally {
    $pendingTok = saved;
  }
}

/// The description of a run time error. Go holds such an error in the
/// unexported type errorString of the runtime package.
final TypeDesc $typeGoError = TypeDesc(
  'runtime.errorString',
  'struct',
  <String, Function>{
    'Error()string': (Object? e) =>
        GoString.fromString('runtime error: ${(e as GoError).message}'),
  },
  text: 'errorString',
);

/// The end of a function that Go proves unreachable. Dart cannot prove it,
/// thus the function needs a statement that gives no value.
Never $unreachable() =>
    throw GoPanic(const GoError('the function reached its end'));

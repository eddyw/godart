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
///
/// Go holds 2 such types. errorString writes the prefix "runtime error: " in
/// front of the message, and plainError writes the message alone. A write to a
/// nil map and each fault of a channel take the plain form.
final class GoError {
  final String message;

  /// True for the plain form, which writes no prefix.
  final bool plain;

  const GoError(this.message, {this.plain = false});

  @override
  String toString() => plain ? message : 'runtime error: $message';
}

/// Builds the value of `panic(nil)`. The package runtime writes it, because
/// only the emitted Go type holds the class of the value.
Object? Function()? $panicNilValue;

/// Installs the builder of the value of `panic(nil)`. The initializer of the
/// package runtime calls it.
void $setPanicNil(Object? Function()? build) => $panicNilValue = build;

/// The value of `panic(nil)` for a program that does not import the package
/// runtime. Such a program cannot name the type, thus the description alone
/// carries the 2 methods of Go.
final class PanicNil {
  const PanicNil();

  @override
  String toString() => 'runtime error: panic called with nil argument';
}

/// The description of the value of `panic(nil)` for such a program.
final TypeDesc $typePanicNil = TypeDesc(
  '*runtime.PanicNilError',
  'ptr',
  <String, Function>{
    'Error()string': (Object? e) =>
        GoString.fromString('runtime error: panic called with nil argument'),
    'RuntimeError()': (Object? e) {},
  },
  text: '*runtime.PanicNilError',
);

/// Raises a Go panic.
///
/// Go 1.21 and each later version report `*runtime.PanicNilError` for a panic
/// of a nil value, thus recover gives a value that is not nil.
Never $panic(Object? value) {
  if (value == null) {
    final Object? Function()? build = $panicNilValue;
    throw GoPanic(
      build != null ? build() : Iface($typePanicNil, const PanicNil()),
    );
  }
  throw GoPanic(value);
}

/// The text of an index or of a slice bound. Go writes the unsigned value of
/// a bound of an unsigned type, thus a value above 2^63 reads as itself and
/// not as a negative number. Read boundsError of runtime/error.go.
String _boundText(int v, bool uns) =>
    uns && v < 0 ? BigInt.from(v).toUnsigned(64).toString() : '$v';

/// Raises the panic of an index. uns names an index of an unsigned type.
///
/// Go writes no length behind a negative index, because the index alone names
/// the fault. An index of an unsigned type is never negative in Go, thus it
/// always takes the form with the length.
Never $panicIndex(int index, int length, [bool uns = false]) {
  final String text = !uns && index < 0
      ? 'index out of range [$index]'
      : 'index out of range [${_boundText(index, uns)}] with length $length';
  throw GoPanic(Iface($typeGoError, GoError(text)));
}

/// The bound of a slice expression that failed. Go holds 1 message for each 1,
/// and a second message for a bound below 0.
///
/// The names follow the codes of the Go runtime: A names the last bound, B the
/// middle 1, and C the first 1. Read boundsErrorFmts of runtime/error.go.
enum SliceBound { aLen, aCap, b, threeALen, threeACap, threeB, threeC }

/// Raises the panic of a slice bound. x is the bound that failed, and y is the
/// limit that it broke.
Never $panicBound(SliceBound code, int x, int y, [bool uns = false]) {
  const String head = 'slice bounds out of range ';
  final bool neg = !uns && x < 0;
  final String v = _boundText(x, uns);
  String text;
  switch (code) {
    case SliceBound.aLen:
      text = neg ? '$head[:$v]' : '$head[:$v] with length $y';
    case SliceBound.aCap:
      text = neg ? '$head[:$v]' : '$head[:$v] with capacity $y';
    case SliceBound.b:
      text = neg ? '$head[$v:]' : '$head[$v:$y]';
    case SliceBound.threeALen:
      text = neg ? '$head[::$v]' : '$head[::$v] with length $y';
    case SliceBound.threeACap:
      text = neg ? '$head[::$v]' : '$head[::$v] with capacity $y';
    case SliceBound.threeB:
      text = neg ? '$head[:$v:]' : '$head[:$v:$y]';
    case SliceBound.threeC:
      text = neg ? '$head[$v::]' : '$head[$v:$y:]';
  }
  throw GoPanic(Iface($typeGoError, GoError(text)));
}

Never $panicDivide() =>
    throw GoPanic(Iface($typeGoError, const GoError('integer divide by zero')));

Never $panicShift() =>
    throw GoPanic(Iface($typeGoError, const GoError('negative shift amount')));

Never $panicNil() => throw GoPanic(
  Iface(
    $typeGoError,
    const GoError('invalid memory address or nil pointer dereference'),
  ),
);

/// The signal that ends 1 goroutine and runs each deferred call of its chain.
///
/// `runtime.Goexit` and `testing.T.FailNow` each throw it. The scheduler holds
/// it at the top of a goroutine, and the harness of a test holds it at the top
/// of a test function.
final class GoStop implements Exception {
  const GoStop();

  @override
  String toString() => 'the goroutine stopped';
}

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
        GoString.fromString((e as GoError).toString()),
    // The interface runtime.Error holds this method, and a recover of a run
    // time panic asserts that interface.
    'RuntimeError()': (Object? e) {},
  },
  text: 'errorString',
);

/// The end of a function that Go proves unreachable. Dart cannot prove it,
/// thus the function needs a statement that gives no value.
Never $unreachable() =>
    throw GoPanic(const GoError('the function reached its end'));

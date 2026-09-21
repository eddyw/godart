import 'goint.dart';
import 'godebug.dart';
import 'iface.dart';
import 'platform_native.dart' if (dart.library.js_interop) 'platform_web.dart';
import 'sched.dart' show $activeCallerFrames, $current;
import 'string.dart';

String $defaultGODEBUG = '';
int $runtimeGODEBUGVersion = 0;

void $setDefaultGODEBUG(String settings) {
  $defaultGODEBUG = settings;
  _panicNilLegacy = _legacyPanicNil(
    '$settings,${$platformGODEBUG()}',
    false,
    startup: true,
  );
}

bool _panicNilLegacy = _legacyPanicNil(
  $platformGODEBUG(),
  false,
  startup: true,
);

bool _legacyPanicNil(String settings, bool current, {bool startup = false}) {
  final entries = settings.split(',');
  for (final entry in startup ? entries : entries.reversed) {
    if (!entry.startsWith('panicnil=')) continue;
    final text = entry.substring('panicnil='.length);
    final value = RegExp(r'^[+-]?[0-9]+$').hasMatch(text)
        ? int.tryParse(text, radix: 10)
        : null;
    if (value != null && value >= -2147483648 && value <= 2147483647) {
      current = value == 1;
    }
    if (!startup) return current;
  }
  return startup ? current : false;
}

void $setRuntimeGODEBUG(GoString settings) {
  $runtimeGODEBUGVersion++;
  _panicNilLegacy = _legacyPanicNil(
    '${$defaultGODEBUG},${settings.toDartString()}',
    _panicNilLegacy,
  );
}

/// The value of a Go panic.
final class GoPanic implements Exception {
  final Object? value;

  final StackTrace trace;
  final List<(String, String, int)> callers;
  final int goroutineID;

  final GoPanic? traceParent;
  // A recovered panic remains visible until its deferred call returns.
  bool traceActive = true;

  /// The panic that stood when this one started. A deferred call that panics
  /// again holds such a link, and Go writes the whole chain.
  GoPanic? link;

  /// True when a deferred call recovered this panic. Go writes the mark
  /// [recovered] behind such a panic.
  bool recovered = false;

  GoPanic(this.value)
    : goroutineID = $current?.id ?? 1,
      trace = StackTrace.current,
      callers = [
        for (final frame in $activeCallerFrames)
          (frame.$name, frame.$goFile, frame.$goLine),
      ],
      traceParent = $chainPanic;

  @override
  String toString() => 'panic: ${$panicText(value)}';
}

/// The text of a panic value.
///
/// Go calls Error on a value that implements error, and String on a value that
/// implements Stringer. Read preprintpanics of runtime/panic.go.
String $panicText(Object? v) {
  if (v == null) return 'nil';
  if (v is Iface) {
    // The key of a method table holds the name and the signature.
    for (final String name in const <String>[
      'Error()string',
      'String()string',
    ]) {
      final Function? m = v.desc.methods[name];
      if (m == null || identical(m, $ifaceMethod)) continue;
      final Object? text = Function.apply(m, <Object?>[v.value]);
      if (text is GoString) return text.toDartString();
    }
  }
  return '$v';
}

/// The text of the chain of active panics. Go writes the oldest panic first,
/// it marks each panic that a deferred call recovered, and it puts each panic
/// behind the first one 1 tab in. Read printpanics of runtime/panic.go.
String $panicChain(GoPanic p) {
  final List<GoPanic> chain = <GoPanic>[];
  for (GoPanic? at = p; at != null; at = at.link) {
    chain.add(at);
  }
  final StringBuffer b = StringBuffer();
  for (final GoPanic at in chain.reversed) {
    if (!identical(at, chain.last)) b.write('\t');
    b.write('panic: ${$panicText(at.value)}');
    if (at.recovered) b.write(' [recovered]');
    b.write('\n');
  }
  return b.toString();
}

/// Links a panic that starts inside the chain of deferred calls of another
/// panic. Go keeps each panic of the chain, and it writes them together.
void _linkPanic(GoPanic e) {
  final GoPanic? prev = $chainPanic;
  if (prev == null || identical(prev, e) || e.link != null) return;
  prev.recovered = $recovered;
  e.link = prev;
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
/// The default nil panic reports `*runtime.PanicNilError`.
/// `GODEBUG=panicnil=1` restores a nil value for recover.
Never $panic(Object? value) {
  if (value == null && _panicNilLegacy && $godebugRegistered) {
    $incGODEBUG(GoString.fromString('panicnil'));
  }
  if (value == null && !_panicNilLegacy) {
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
String _boundText(GoInt v, bool uns) => uns ? $ustring(v) : '$v';

/// Raises the panic of an index. uns names an index of an unsigned type.
///
/// Go writes no length behind a negative index, because the index alone names
/// the fault. An index of an unsigned type is never negative in Go, thus it
/// always takes the form with the length.
Never $panicIndex(GoInt index, Object length, [bool uns = false]) {
  final String text = !uns && $ix(index) < 0
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
Never $panicBound(SliceBound code, GoInt x, Object y, [bool uns = false]) {
  const String head = 'slice bounds out of range ';
  final bool neg = !uns && $ix(x) < 0;
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

/// Raises the panic of a hash of a key whose type holds no identity.
///
/// Go writes 2 texts. A read and a delete take the plain form with a colon,
/// and a write takes the prefix "runtime error: " with no colon. Read
/// nilinterhash of runtime/alg.go and typehash of runtime/alg.go.
Never $panicHash(String type, bool write) => throw GoPanic(
  Iface(
    $typeGoError,
    write
        ? GoError('hash of unhashable type $type')
        : GoError('hash of unhashable type: $type', plain: true),
  ),
);

/// Raises the panic of a conversion of a slice to an array or to a pointer to
/// an array. Go gives a runtime.Error, thus the value carries the prefix.
Never $panicConvert(int len, int n) => throw GoPanic(
  Iface(
    $typeGoError,
    GoError(
      'cannot convert slice with length $len '
      'to array or pointer to array with length $n',
    ),
  ),
);

/// Raises the panic of a method wrapper that a nil pointer receiver reached.
/// A value method needs the value behind the pointer, thus Go stops. name
/// holds the type with its package, and short holds the type alone. Read
/// panicwrap of runtime/error.go.
Never $panicWrap(String name, String method, String short) => throw GoPanic(
  Iface(
    $typeGoError,
    GoError(
      'value method $name.$method called using nil *$short pointer',
      plain: true,
    ),
  ),
);

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
    if ($recovered) $chainPanic?.traceActive = false;
    try {
      _calls.removeLast()();
    } catch (e) {
      if (e is GoPanic) {
        // A panic inside a deferred call replaces the chain panic, and the
        // calls that remain run for the new panic.
        _linkPanic(e);
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
    // The state of the chain that holds this one stays in a local. A deferred
    // call can open a chain of its own, thus 1 pair of variables of the
    // library would lose the state of the outer chain.
    final GoPanic? outer = $chainPanic;
    final bool outerRec = $recovered;
    $recovered = false;
    Object? thrown;
    // The loop consumes each call, thus a second run of the chain finds an
    // empty stack and does nothing. The return path and the exit path of 1
    // function can both reach the chain.
    while (_calls.isNotEmpty) {
      if ($recovered) $chainPanic?.traceActive = false;
      try {
        _calls.removeLast()();
      } catch (e) {
        if (e is GoPanic) {
          _linkPanic(e);
          $chainPanic = e;
          $recovered = false;
          thrown = e;
        } else {
          $chainPanic = outer;
          $recovered = outerRec;
          rethrow;
        }
      }
    }
    final rec = $recovered;
    if (rec) $chainPanic?.traceActive = false;
    $chainPanic = outer;
    $recovered = outerRec;
    if (!rec && thrown != null) throw thrown;
    return rec;
  }

  /// Runs each call for the chain panic of the function, which [start] gives.
  bool runChain(GoPanic? start) {
    final GoPanic? outer = $chainPanic;
    $chainPanic = start;
    final bool rec = runRecovered();
    $chainPanic = outer;
    return rec;
  }
}

/// The panic that the chain of deferred calls of 1 function handles. Null on a
/// chain that a normal return starts.
GoPanic? $chainPanic;

/// True when a deferred call of the chain recovered the chain panic.
bool $recovered = false;

/// The state of the chain of deferred calls that holds another chain.
///
/// A deferred call can open a chain of its own, and 2 chains can stand open at
/// the same time. The state thus rides a local of the caller, and a machine
/// keeps it in its frame. A pair of variables of the library would lose the
/// state of the outer chain.
final class ChainState {
  final GoPanic? panic;
  final bool recovered;

  const ChainState(this.panic, this.recovered);
}

/// Opens the chain of the deferred calls of 1 function, and gives the state of
/// the chain that holds it.
///
/// e is the panic that reached the function, and null for a normal return. A
/// value of another class opens a chain with no panic, thus recover gives null
/// for it.
ChainState $chainOpen(Object? e) {
  final ChainState saved = ChainState($chainPanic, $recovered);
  $chainReset(e);
  return saved;
}

/// Puts a value that reached an open chain in the place of the panic of that
/// chain. A chain that already stands open thus keeps the state of the chain
/// that holds it.
void $chainReset(Object? e) {
  $recovered = false;
  $chainPanic = e is GoPanic ? e : null;
}

/// Closes the chain and gives the state of the chain that holds it back.
void $chainClose(ChainState saved) {
  $chainPanic = saved.panic;
  $recovered = saved.recovered;
}

void $chainEnd(ChainState saved) {
  if ($recovered) $chainPanic?.traceActive = false;
  $chainClose(saved);
}

/// The token of the function that the chain of the deferred calls runs now.
/// Go gives a recover the value only in the function that the defer pushed,
/// and the token tells such a function from each function that it calls.
int $pendingTok = 0;

/// The value of recover. A call outside the deferred function of a panicking
/// function gives null.
///
/// A recover consumes the panic. Each call after that gives null, exactly as
/// gorecover of Go does with the mark p.recovered. The flag [$recovered]
/// belongs to the chain, thus a new panic inside a deferred call clears it
/// and the calls that remain read that new panic.
Iface? $recover(int tok) {
  final GoPanic? p = $chainPanic;
  if (p == null || $recovered || tok == 0 || tok != $pendingTok) return null;
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

/// The value of a Go panic.
///
/// The subset has no `recover`, thus a panic always stops the program. The
/// deferred functions run through a `finally` block on the way out.
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

Never $panicIndex(int index, int length) =>
    throw GoPanic(GoError('index out of range [$index] with length $length'));

Never $panicSlice(int low, int high, int capacity) => throw GoPanic(
  GoError('slice bounds out of range [$low:$high] with capacity $capacity'),
);

Never $panicDivide() => throw GoPanic(const GoError('integer divide by zero'));

Never $panicNil() => throw GoPanic(
  const GoError('invalid memory address or nil pointer dereference'),
);

/// Reads a value and drops it. The Go statement `_ = x` runs the read, thus a
/// bounds check still applies.
void $drop(Object? value) {}

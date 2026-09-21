import 'sched.dart';

/// Runs initialization steps in order, including their saved continuations.
void $initSequence(List<void Function()> steps) {
  _InitSequence(steps).resume();
}

final class _InitSequence implements NestedFrame {
  final List<void Function()> steps;
  int index = 0;
  Frame? child;

  _InitSequence(this.steps);

  @override
  Frame get $inner => child!;

  @override
  set $inner(Frame frame) => child = frame;

  @override
  Object? resume() {
    if (child != null) {
      $resumeFrame(child!);
      if ($blocked != null) {
        child = $blocked;
        $block(this);
        return null;
      }
      child = null;
    }
    while (index < steps.length) {
      steps[index++]();
      if ($blocked != null) {
        child = $blocked;
        $block(this);
        return null;
      }
    }
    return null;
  }
}

/// Assigns a global only after its initializer returns the final result.
void $initValue<T>(T Function() initialize, void Function(T) assign) {
  final value = initialize();
  if ($blocked == null) {
    assign(value);
  } else {
    $block(_InitValue<T>($blocked!, assign));
  }
}

final class _InitValue<T> implements NestedFrame {
  Frame child;
  final void Function(T) assign;

  _InitValue(this.child, this.assign);

  @override
  Frame get $inner => child;

  @override
  set $inner(Frame frame) => child = frame;

  @override
  Object? resume() {
    final value = $resumeFrame(child);
    if ($blocked != null) {
      child = $blocked!;
      $block(this);
    } else {
      assign(value as T);
    }
    return null;
  }
}

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

final class _Leaf implements Frame {
  final bool fail;
  int calls = 0;
  _Leaf(this.fail);

  @override
  Object? resume() {
    calls++;
    if (calls <= 3) {
      $ready($current!);
      $block(this);
      return 0;
    }
    if (fail) throw StateError('leaf');
    return 7;
  }
}

final class _Link implements NestedFrame {
  Frame inner;
  final List<int> completed;
  final int index;
  _Link(this.inner, this.completed, this.index);

  @override
  Frame get $inner => inner;
  @override
  set $inner(Frame frame) {
    inner = frame;
  }

  @override
  Object? resume() {
    try {
      final value = $resumeFrame(inner);
      if ($blocked != null) {
        inner = $blocked!;
        $block(this);
        return 0;
      }
      return (value as int) + 1;
    } finally {
      if ($blocked == null) completed.add(index);
    }
  }
}

final class _Catch implements NestedFrame {
  Frame inner;
  _Catch(this.inner);

  @override
  Frame get $inner => inner;
  @override
  set $inner(Frame frame) {
    inner = frame;
  }

  @override
  Object? resume() {
    try {
      final value = $resumeFrame(inner);
      if ($blocked != null) {
        inner = $blocked!;
        $block(this);
      }
      return value;
    } on StateError catch (error) {
      return error.message;
    }
  }
}

void main() {
  for (final fail in [false, true]) {
    test('deep saved frames preserve repeated waits and errors: $fail', () {
      final leaf = _Leaf(fail);
      final completed = <int>[];
      Frame root = leaf;
      for (var i = 0; i < 12000; i++) {
        root = _Link(root, completed, i);
      }
      root = _Catch(root);
      final goroutine = $go(() {
        $ready($current!);
        $block(root);
      });
      $schedule();
      expect(goroutine.ended, isTrue);
      expect(goroutine.result, fail ? 'leaf' : 12007);
      expect(leaf.calls, 4);
      expect(completed, List<int>.generate(12000, (i) => i));
    });
  }
}

import 'package:flutter/widgets.dart';
import 'package:goruntime/goruntime.dart';

import 'bridge.dart';
import 'identity.dart';

/// GoRuntime drives 1 Go library from Flutter.
///
/// It runs a Go closure that an event starts, it applies the work that a
/// goroutine posted, and it rebuilds each boundary that the Go graph marked.
final class GoRuntime {
  final GoBridge bridge;
  final Map<int, VoidCallback> _boundaries = <int, VoidCallback>{};

  /// identities holds each Flutter object that a Go key names, for example the
  /// FocusNode of a ui.FocusKey.
  final GoIdentities identities = GoIdentities();

  /// onRoot rebuilds the whole tree. GoApp writes it.
  VoidCallback? onRoot;

  GoRuntime(this.bridge);

  /// start runs the init of the Go library, and it puts the tick of this host
  /// on the idle hook of the Go scheduler.
  ///
  /// The hook runs each time that the run queue of Go empties. A goroutine that
  /// wakes from a timer or from a future thus reaches the screen with no poll.
  void start() {
    bridge.init();
    $hosted = true;
    $onIdle = tick;
  }

  /// stop takes the tick off the idle hook, and it releases each Flutter
  /// object that a key names. No sweep follows the last frame of the tree.
  void stop() {
    if ($onIdle == tick) $onIdle = null;
    identities.dropAll();
  }

  /// bind registers the rebuild of 1 boundary.
  void bind(int id, VoidCallback rebuild) {
    _boundaries[id] = rebuild;
  }

  /// unbind drops the rebuild of 1 boundary.
  void unbind(int id) {
    _boundaries.remove(id);
  }

  /// run starts a Go closure that an event gives, and it applies the result.
  /// The closure may block, because it runs as a goroutine of its own.
  ///
  /// The handle gives the end of that goroutine. A host that waits for the
  /// work, such as a refresh indicator, reads it.
  Goroutine run(void Function() body) {
    final Goroutine g = $go(body);
    $schedule();
    return g;
  }

  /// tick applies the work that a goroutine posted, and it rebuilds each dirty
  /// boundary. The idle hook of the Go scheduler calls it.
  void tick() {
    bridge.pump();
    flush();
  }

  /// flush rebuilds each boundary that the Go graph marked.
  void flush() {
    final List<int> dirty = bridge.takeDirty();
    for (final int id in dirty) {
      if (id == rootID) {
        onRoot?.call();
        continue;
      }
      _boundaries[id]?.call();
    }
  }
}

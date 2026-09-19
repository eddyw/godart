import 'dart:convert';

import 'package:flutter/widgets.dart';
import 'package:goruntime/goruntime.dart';

import 'bridge.dart';
import 'calls.dart';
import 'identity.dart';

/// GoRuntime drives 1 Go library from Flutter.
///
/// It runs a Go closure that an event starts, it applies the work that a
/// goroutine posted, and it rebuilds each boundary that the Go graph marked.
final class GoRuntime {
  final GoBridge bridge;
  final Map<int, VoidCallback> _boundaries = <int, VoidCallback>{};
  final Map<int, StreamIterator<Object?>> _streams = {};
  int _nextStream = 0;
  bool _stopped = false;

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
    _stopped = false;
    bridge.init();
    $hosted = true;
    $onIdle = tick;
  }

  /// stop takes the tick off the idle hook, and it releases each Flutter
  /// object that a key names. No sweep follows the last frame of the tree.
  void stop() {
    _stopped = true;
    for (final stream in _streams.values) {
      unawaited(stream.cancel());
    }
    _streams.clear();
    if ($onIdle == tick) $onIdle = null;
    identities.dropAll();
  }

  int openStream(Stream<Object?> stream) {
    final id = ++_nextStream;
    _streams[id] = StreamIterator(stream);
    return id;
  }

  Future<Map<String, Object?>> nextStream(int id) async {
    final stream = _streams[id];
    if (stream == null) throw StateError('Stream $id is closed');
    final pending = stream.moveNext();
    try {
      if (await pending) {
        return {'Value': stream.current, 'Done': false};
      }
      _streams.remove(id);
      return {'Done': true};
    } catch (_) {
      _streams.remove(id);
      rethrow;
    }
  }

  Future<void> cancelStream(int id) async {
    final stream = _streams.remove(id);
    await stream?.cancel();
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
  ///
  /// A posted function can start a goroutine, for example a command that a new
  /// screen runs at once. The scheduler already left its loop when the hook
  /// runs, thus that goroutine waits for the next event of the host. The loop
  /// below runs it now.
  void tick() {
    bridge.pump();
    flush();
    while (!$runqEmpty) {
      $schedule();
      bridge.pump();
      flush();
    }
    _calls();
  }

  /// _calls gives each request of Go to the handler of the application.
  ///
  /// A goroutine that calls ui.Call parks on a channel, thus the scheduler
  /// stops before this line. The answer wakes the goroutine, and the tick of
  /// the answer runs it.
  void _calls() {
    final String text = bridge.takeCalls();
    if (text.isEmpty) return;
    for (final Object? item in jsonDecode(text) as List<Object?>) {
      final Map<String, Object?> m = item as Map<String, Object?>;
      _answer(m['id']! as int, m['name']! as String, m['arg']! as String);
    }
  }

  /// _answer runs the Dart work of 1 request and it writes the result back.
  Future<void> _answer(int id, String name, String arg) async {
    String value = '';
    String error = '';
    final GoCall handler = goCallHandler ?? goHostCall;
    try {
      final generated = _generatedCalls[name];
      value = generated == null
          ? await handler(name, arg)
          : await generated(this, arg);
    } catch (e) {
      error = e.toString();
    }
    if (_stopped) return;
    bridge.answer(id, value, error);
    tick();
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

/// GoCall runs the Dart work of 1 request of Go. The name tells it which call
/// it must run, and the argument and the answer each hold a text of the
/// application.
typedef GoCall = Future<String> Function(String name, String arg);

typedef GoGeneratedCall = Future<String> Function(
  GoRuntime runtime,
  String payload,
);

Map<String, GoGeneratedCall> _generatedCalls = {};

void goRegisterCalls(Map<String, GoGeneratedCall> calls) {
  _generatedCalls = Map.of(calls);
}

/// goCallHandler holds the handler that goSetCall installed.
GoCall? goCallHandler;

/// goSetCall installs the handler of each call of Go. The application calls it
/// before runApp.
///
/// A Go tree carries no call of Dart, thus a date picker, a share sheet, a
/// dialog with an answer, and each plugin of a platform channel reach Go
/// through this handler. Read flutter/ui/ask.go.
///
/// goHostCall answers each call of flutter/ui/calls.go, and it runs when the
/// application installs no handler. A handler of the application answers its
/// own names first, and it gives each other name to goHostCall.
void goSetCall(GoCall f) {
  goCallHandler = f;
}

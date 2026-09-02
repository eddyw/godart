import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';
import 'package:goruntime/goruntime.dart';

import 'node.dart';
import 'runtime.dart';

/// This file holds each Flutter object that a Go key names.
///
/// A Go tree is data, and it rebuilds on each change, thus a Go value carries
/// no FocusNode, no ScrollController, and no WidgetStatesController. The Go
/// side gives a name, and the table below owns the object under that name.
///
/// The lifetime follows the element tree, and never a build pass. A lazy list
/// builds a row long after the build of the list. A sweep of the keys that the
/// last build read would thus drop an object that a live widget holds.
/// goRender wraps each node that reads a key in a GoKeep, and that widget
/// holds the object while its element stands.

/// _Held is 1 object of the table.
final class _Held {
  _Held(this.kind, this.name, this.object, this.drop);

  final String kind;
  final String name;
  final Listenable object;

  /// drop releases the Flutter object. Each of the 3 kinds owns a dispose of
  /// its own, and this field saves a switch on the kind.
  final void Function() drop;

  /// users counts the GoKeep widgets that hold this object. A count above 1
  /// means that 1 key reached 2 widgets, and the sweep reports it.
  int users = 0;

  /// report gives the state of the object to the Go side. Each build writes it
  /// again, because a rebuild gives a fresh Go closure.
  void Function()? report;

  /// queued marks a report that waits for the end of the frame.
  bool queued = false;

  String get id => '$kind:$name';
}

/// GoIdentities holds 1 Flutter object for each key of the Go tree.
final class GoIdentities {
  final Map<String, _Held> _held = <String, _Held>{};
  final List<Set<String>> _open = <Set<String>>[];
  bool _sweepQueued = false;

  /// ids names each key that the table holds. A test reads it to prove that
  /// the host disposed an object.
  List<String> get ids => _held.keys.toList()..sort();

  /// open starts the collection of the keys that 1 node reads. goRender calls
  /// it around the build of 1 widget, and the calls nest with the tree.
  void open() => _open.add(<String>{});

  /// close ends the collection and gives the keys that the node read.
  Set<String> close() => _open.removeLast();

  /// focusFor gives the FocusNode of 1 name, and it makes the node on the
  /// first read. A hand written widget reaches the same node as the Go tree.
  FocusNode focusFor(String name) => _focus(name).object as FocusNode;

  /// scrollFor gives the ScrollController of 1 name.
  ScrollController scrollFor(String name, {double initial = 0}) =>
      _scroll(name, initial).object as ScrollController;

  /// statesFor gives the WidgetStatesController of 1 name.
  WidgetStatesController statesFor(String name) =>
      _states(name).object as WidgetStatesController;

  /// focus gives the FocusNode of 1 ui.FocusKey.
  FocusNode? focus(GoNode? key, GoRuntime rt) {
    if (key == null) return null;
    final _Held h = _focus(key.text('Name'));
    final FocusNode node = h.object as FocusNode;
    final void Function(bool)? f = key.onFlag('OnFocus');
    h.report = f == null ? null : () => rt.run(() => f(node.hasFocus));
    return node;
  }

  /// scroll gives the ScrollController of 1 ui.ScrollKey.
  ScrollController? scroll(GoNode? key, GoRuntime rt) {
    if (key == null) return null;
    final _Held h = _scroll(key.text('Name'), key.size('Initial'));
    final ScrollController c = h.object as ScrollController;
    final void Function(double)? f = key.onSize('OnScroll');
    h.report = f == null
        ? null
        : () {
            if (!c.hasClients) return;
            final double at = c.offset;
            rt.run(() => f(at));
          };
    return c;
  }

  /// states gives the WidgetStatesController of 1 ui.StatesKey.
  WidgetStatesController? states(GoNode? key, GoRuntime rt) {
    if (key == null) return null;
    final _Held h = _states(key.text('Name'));
    final WidgetStatesController c = h.object as WidgetStatesController;
    final void Function(Slice<GoString>)? f = key.onTexts('OnStates');
    h.report = f == null
        ? null
        : () {
            final Slice<GoString> on = Slice<GoString>.of(<GoString>[
              for (final WidgetState s in WidgetState.values)
                if (c.value.contains(s)) GoString.fromString(s.name),
            ]);
            rt.run(() => f(on));
          };
    return c;
  }

  /// retain marks each key as held by 1 more widget.
  void retain(Iterable<String> ids) {
    for (final String id in ids) {
      final _Held? h = _held[id];
      if (h != null) h.users++;
    }
    _needSweep();
  }

  /// release marks each key as held by 1 widget fewer.
  void release(Iterable<String> ids) {
    for (final String id in ids) {
      final _Held? h = _held[id];
      if (h != null) h.users--;
    }
    _needSweep();
  }

  /// sweep disposes each object that no widget holds, and it reports each key
  /// that reached 2 widgets.
  ///
  /// It runs after the frame, and never during the build. Flutter mounts a new
  /// element before it unmounts the old 1. A widget that moves thus holds its
  /// key twice for the length of 1 build.
  void sweep() {
    final List<String> gone = <String>[];
    final List<String> shared = <String>[];
    for (final _Held h in _held.values) {
      if (h.users <= 0) {
        gone.add(h.id);
      } else if (h.users > 1) {
        shared.add('${h.id} reaches ${h.users} widgets');
      }
    }
    for (final String id in gone) {
      _held.remove(id)!.drop();
    }
    if (shared.isEmpty) return;
    FlutterError.reportError(
      FlutterErrorDetails(
        exception: StateError(
          'the host holds 1 Flutter object for each key, thus 1 key cannot '
          'serve 2 widgets at the same time: ${shared.join(", ")}',
        ),
        library: 'goflutter',
        context: ErrorDescription('while the host swept the key table'),
      ),
    );
  }

  /// dropAll releases each object. GoApp calls it when the whole tree leaves
  /// the screen, because no sweep follows the last frame.
  void dropAll() {
    for (final _Held h in _held.values) {
      h.drop();
    }
    _held.clear();
    _open.clear();
  }

  _Held _focus(String name) => _obtain('focus', name, (String n) {
    final FocusNode node = FocusNode(debugLabel: 'gart focus $n');
    return _Held('focus', n, node, node.dispose);
  });

  _Held _scroll(String name, double initial) =>
      _obtain('scroll', name, (String n) {
        final ScrollController c = ScrollController(
          initialScrollOffset: initial,
          debugLabel: 'gart scroll $n',
        );
        return _Held('scroll', n, c, c.dispose);
      });

  _Held _states(String name) => _obtain('states', name, (String n) {
    final WidgetStatesController c = WidgetStatesController();
    return _Held('states', n, c, c.dispose);
  });

  _Held _obtain(String kind, String name, _Held Function(String) make) {
    final String id = '$kind:$name';
    if (_open.isNotEmpty) _open.last.add(id);
    final _Held? found = _held[id];
    if (found != null) return found;
    final _Held made = make(name);
    _held[id] = made;
    made.object.addListener(() => _changed(made));
    _needSweep();
    return made;
  }

  /// _changed gives the new state of 1 object to the Go side.
  ///
  /// A scroll notifies during the layout of Flutter, and a Go closure there
  /// would mark a boundary dirty in the middle of a build. The report thus
  /// waits for the end of the frame, and 1 frame carries 1 report for each
  /// object.
  void _changed(_Held h) {
    if (h.report == null || h.queued) return;
    if (SchedulerBinding.instance.schedulerPhase !=
        SchedulerPhase.persistentCallbacks) {
      h.report!.call();
      return;
    }
    h.queued = true;
    SchedulerBinding.instance.addPostFrameCallback((Duration _) {
      h.queued = false;
      if (_held[h.id] == h) h.report?.call();
    });
  }

  void _needSweep() {
    if (_sweepQueued) return;
    _sweepQueued = true;
    SchedulerBinding.instance.addPostFrameCallback((Duration _) {
      _sweepQueued = false;
      sweep();
    });
  }
}

/// GoKeep holds the keys of 1 node while its element stands in the tree.
///
/// goRender puts it around a node that reads a key, and around no other node.
/// A wrapper for each node would cost 1 element for each widget of the tree.
final class GoKeep extends StatefulWidget {
  const GoKeep({
    super.key,
    required this.ids,
    required this.table,
    required this.child,
  });

  final Set<String> ids;
  final GoIdentities table;
  final Widget child;

  @override
  State<GoKeep> createState() => _GoKeepState();
}

class _GoKeepState extends State<GoKeep> {
  @override
  void initState() {
    super.initState();
    widget.table.retain(widget.ids);
  }

  @override
  void didUpdateWidget(covariant GoKeep old) {
    super.didUpdateWidget(old);
    if (setEquals(old.ids, widget.ids)) return;
    widget.table.retain(widget.ids);
    old.table.release(old.ids);
  }

  @override
  void dispose() {
    widget.table.release(widget.ids);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}

/// goFocusNode reads the ui.FocusKey struct of the Go side.
FocusNode? goFocusNode(GoNode? key, GoRuntime rt) =>
    rt.identities.focus(key, rt);

/// goScrollController reads the ui.ScrollKey struct of the Go side.
ScrollController? goScrollController(GoNode? key, GoRuntime rt) =>
    rt.identities.scroll(key, rt);

/// goStatesController reads the ui.StatesKey struct of the Go side.
WidgetStatesController? goStatesController(GoNode? key, GoRuntime rt) =>
    rt.identities.states(key, rt);

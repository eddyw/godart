import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';
import 'package:goruntime/goruntime.dart';

import 'convert.dart';
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
  final Object object;

  /// drop releases the Flutter object. Each kind owns a dispose of its own,
  /// and this field saves a switch on the kind. An object that holds no
  /// dispose takes an empty function.
  final void Function() drop;

  /// state holds the last command that the host gave to the object. An
  /// animation, an overlay, and a menu each read a command from the Go tree,
  /// and the host gives it 1 time for each change. Read _drive.
  String state = '';

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

  /// vsync drives each animation that a key names. GoApp writes it, thus a
  /// ticker stops when the host leaves the screen.
  TickerProvider? vsync;

  /// focus gives the FocusNode of 1 ui.FocusKey.
  FocusNode? focus(GoNode? key, GoRuntime rt) {
    if (key == null) return null;
    final _Held h = _focus(key.text('Name', 0));
    final FocusNode node = h.object as FocusNode;
    final void Function(bool)? f = key.onFlag('OnFocus', 1);
    h.report = f == null ? null : () => rt.run(() => f(node.hasFocus));
    return node;
  }

  /// scroll gives the ScrollController of 1 ui.ScrollKey.
  ScrollController? scroll(GoNode? key, GoRuntime rt) {
    if (key == null) return null;
    final _Held h = _scroll(key.text('Name', 0), key.size('Initial', 1));
    final ScrollController c = h.object as ScrollController;
    final void Function(double)? f = key.onSize('OnScroll', 2);
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
    final _Held h = _states(key.text('Name', 0));
    final WidgetStatesController c = h.object as WidgetStatesController;
    final void Function(Slice<GoString>)? f = key.onTexts('OnStates', 1);
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

  /// text gives the TextEditingController of 1 ui.TextKey.
  TextEditingController? text(GoNode? key, GoRuntime rt) {
    if (key == null) return null;
    final _Held h = _text(key.text('Name', 0), key.text('Initial', 1));
    final TextEditingController c = h.object as TextEditingController;
    final void Function(GoString)? f = key.onText('OnChanged', 2);
    h.report = f == null
        ? null
        : () {
            final GoString v = GoString.fromString(c.text);
            rt.run(() => f(v));
          };
    return c;
  }

  /// animation gives the AnimationController of 1 ui.AnimationKey.
  Animation<double>? animation(GoNode? key, GoRuntime rt) {
    if (key == null) return null;
    final int millis = key.count('Millis', 1);
    final _Held h = _anim(key.text('Name', 0), millis);
    final AnimationController c = h.object as AnimationController;
    final double target = key.size('Target', 2);
    final String curve = key.text('Curve', 3);
    final bool repeat = key.flag('Repeat', 4);
    final bool reverse = key.flag('Reverse', 5);
    final void Function()? end = key.action('OnEnd', 6);
    _drive(h, '$millis/$target/$curve/$repeat/$reverse', () {
      c.duration = Duration(milliseconds: millis == 0 ? 300 : millis);
      if (repeat) {
        c.repeat(reverse: reverse);
        return;
      }
      if (c.value == target) return;
      final TickerFuture run = c.animateTo(
        target,
        curve: goCurve(curve) ?? Curves.linear,
      );
      if (end != null) run.then((void _) => rt.run(end));
    });
    return c;
  }

  /// link gives the LayerLink of 1 ui.LinkKey.
  LayerLink? link(GoNode? key) {
    if (key == null) return null;
    return _obtain('link', key.text('Name', 0), (String n) {
          return _Held('link', n, LayerLink(), () {});
        }).object
        as LayerLink;
  }

  /// overlap gives the SliverOverlapAbsorberHandle of 1 ui.OverlapKey.
  SliverOverlapAbsorberHandle? overlap(GoNode? key) {
    if (key == null) return null;
    return _obtain('overlap', key.text('Name', 0), (String n) {
          final SliverOverlapAbsorberHandle h = SliverOverlapAbsorberHandle();
          return _Held('overlap', n, h, h.dispose);
        }).object
        as SliverOverlapAbsorberHandle;
  }

  /// overlay gives the OverlayPortalController of 1 ui.OverlayKey.
  OverlayPortalController? overlay(GoNode? key) {
    if (key == null) return null;
    final _Held h = _obtain('overlay', key.text('Name', 0), (String n) {
      return _Held(
        'overlay',
        n,
        OverlayPortalController(debugLabel: 'gart overlay $n'),
        () {},
      );
    });
    final OverlayPortalController c = h.object as OverlayPortalController;
    final bool show = key.flag('Show', 1);
    _drive(h, show ? 'show' : 'hide', () {
      if (show == c.isShowing) return;
      if (show) {
        c.show();
      } else {
        c.hide();
      }
    });
    return c;
  }

  /// menu gives the MenuController of 1 ui.MenuKey.
  MenuController? menu(GoNode? key) {
    if (key == null) return null;
    final _Held h = _obtain('menu', key.text('Name', 0), (String n) {
      return _Held('menu', n, MenuController(), () {});
    });
    final MenuController c = h.object as MenuController;
    final bool open = key.flag('Open', 1);
    _drive(h, open ? 'open' : 'close', () {
      if (open == c.isOpen) return;
      if (open) {
        c.open();
      } else {
        c.close();
      }
    });
    return c;
  }

  /// scope gives the FocusScopeNode of 1 ui.ScopeKey.
  FocusScopeNode? scope(GoNode? key) {
    if (key == null) return null;
    return _obtain('scope', key.text('Name', 0), (String n) {
          final FocusScopeNode node = FocusScopeNode(
            debugLabel: 'gart scope $n',
          );
          return _Held('scope', n, node, node.dispose);
        }).object
        as FocusScopeNode;
  }

  /// storage gives the PageStorageBucket of 1 ui.StorageKey.
  PageStorageBucket? storage(GoNode? key) {
    if (key == null) return null;
    return _obtain('storage', key.text('Name', 0), (String n) {
          return _Held('storage', n, PageStorageBucket(), () {});
        }).object
        as PageStorageBucket;
  }

  /// _drive gives 1 command to an object of the table, and it gives it 1 time
  /// for each change of the Go tree.
  ///
  /// The command runs after the frame. An animation that starts during a build
  /// marks a widget dirty in the middle of that build, and Flutter reports
  /// such a change as an error.
  void _drive(_Held h, String want, void Function() run) {
    if (h.state == want) return;
    h.state = want;
    SchedulerBinding.instance.addPostFrameCallback((Duration _) {
      if (_held[h.id] == h) run();
    });
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

  _Held _text(String name, String initial) =>
      _obtain('text', name, (String n) {
        final TextEditingController c = TextEditingController(text: initial);
        return _Held('text', n, c, c.dispose);
      });

  _Held _anim(String name, int millis) => _obtain('anim', name, (String n) {
    final TickerProvider? v = vsync;
    if (v == null) {
      throw StateError(
        'the host holds no ticker provider, thus the key $n names no '
        'animation. GoApp writes the provider of the application.',
      );
    }
    final AnimationController c = AnimationController(
      vsync: v,
      duration: Duration(milliseconds: millis == 0 ? 300 : millis),
      debugLabel: 'gart animation $n',
    );
    return _Held('anim', n, c, c.dispose);
  });

  _Held _obtain(String kind, String name, _Held Function(String) make) {
    final String id = '$kind:$name';
    if (_open.isNotEmpty) _open.last.add(id);
    final _Held? found = _held[id];
    if (found != null) return found;
    final _Held made = make(name);
    _held[id] = made;
    final Object object = made.object;
    if (object is Listenable) object.addListener(() => _changed(made));
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

/// goTextController reads the ui.TextKey struct of the Go side.
TextEditingController? goTextController(GoNode? key, GoRuntime rt) =>
    rt.identities.text(key, rt);

/// goAnimation reads the ui.AnimationKey struct of the Go side.
Animation<double>? goAnimation(GoNode? key, GoRuntime rt) =>
    rt.identities.animation(key, rt);

/// goLayerLink reads the ui.LinkKey struct of the Go side.
LayerLink? goLayerLink(GoNode? key, GoRuntime rt) =>
    rt.identities.link(key);

/// goOverlapHandle reads the ui.OverlapKey struct of the Go side.
SliverOverlapAbsorberHandle? goOverlapHandle(GoNode? key, GoRuntime rt) =>
    rt.identities.overlap(key);

/// goOverlayController reads the ui.OverlayKey struct of the Go side.
OverlayPortalController? goOverlayController(GoNode? key, GoRuntime rt) =>
    rt.identities.overlay(key);

/// goMenuController reads the ui.MenuKey struct of the Go side.
MenuController? goMenuController(GoNode? key, GoRuntime rt) =>
    rt.identities.menu(key);

/// goFocusScope reads the ui.ScopeKey struct of the Go side.
FocusScopeNode? goFocusScope(GoNode? key, GoRuntime rt) =>
    rt.identities.scope(key);

/// goStorageBucket reads the ui.StorageKey struct of the Go side.
PageStorageBucket? goStorageBucket(GoNode? key, GoRuntime rt) =>
    rt.identities.storage(key);

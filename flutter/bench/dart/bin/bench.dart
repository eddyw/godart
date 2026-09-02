import 'dart:convert';

import 'package:bench_harness/baseline.dart';
import 'package:bench_harness/diff.dart';
import 'package:bench_harness/walk.dart';
import 'package:gobench/bench.dart' as bench;
import 'package:gobench/dep/gart.dev/ui.dart' as ui;
import 'package:goruntime/goruntime.dart';

/// This program measures the cost of a text property at the boundary between a
/// Go tree and the Dart host. It writes 1 CSV line for each batch, and
/// bin/report.dart turns the lines into the table.
///
/// Run it as an ahead of time build. The Dart VM hides the result behind its
/// warm up cost, thus `dart run` gives a number that no reader can act on.

/// The node counts of the sweep. 20 stands for a small screen, 200 for a dense
/// one, and 500 for the worst case that a scroll view holds on 1 frame.
const List<int> nodeCounts = <int>[20, 50, 200, 500];

/// The text lengths of the sweep, in characters. 5 stands for a label, 20 for
/// a line of a list, and 80 for a paragraph.
const List<int> textLengths = <int>[5, 20, 80];

/// batches gives the number of timed batches of 1 case. The report takes the
/// median over each batch of each process run.
const int batches = 15;

/// minBatchUs holds the target length of 1 batch. A shorter batch reads the
/// resolution of the clock, and a longer one costs time with no gain.
const int minBatchUs = 20000;

/// warmUs holds the length of the warm up of 1 case. The measurement runs on an
/// ahead of time build, thus the warm up serves the heap and the caches alone.
const int warmUs = 60000;

int _row = 0;
List<Object?> _builds = const <Object?>[];

void main(List<String> args) {
  final String filter = args.isEmpty ? '' : args[0];
  bench.$init();

  // The 2 micro cases give the price of 1 crossing of the boundary, thus a
  // reader can multiply it by a node count of a screen of their own.
  for (final int l in textLengths) {
    _micro(dLiteral(l), '$l', filter);
  }
  _micro('Цена заказа за товар', 'u20', filter);

  for (final int n in nodeCounts) {
    for (final int l in textLengths) {
      _sweepCell(n, l, false, filter);
    }
  }

  // A10 keeps the tree of the frame before and it diffs the new one against
  // it. The sweep answers whether the diff costs less than the walk that it
  // replaces.
  for (final int n in nodeCounts) {
    for (final int l in textLengths) {
      _sweepDiff(n, l, filter);
    }
  }

  // The UTF-8 case answers whether the length in bytes or the length in runes
  // drives the cost. 20 runes stand in 37 bytes here.
  for (final int n in <int>[200]) {
    _sweepCell(n, 20, true, filter);
  }

  // A rebuild of 1 boundary is the normal case of the framework. The node count
  // of the whole tree must not reach the number, and the 2 counts prove it.
  for (final int n in <int>[200, 500]) {
    for (final int l in textLengths) {
      _sweepWatch(n, l, filter);
    }
  }
}

/// _micro measures 1 conversion of 1 text. The node count of the row is 1,
/// because the case holds no tree.
///
/// The 2 extra cases give the floor of each direction. GoString.fromString
/// scans for an escape unit, it encodes, and GoString.fromBytes then copies the
/// result a second time. GoString.toDartString decodes an unmodifiable view,
/// and a plain Uint8List may take a faster path.
void _micro(String s, String tag, String filter) {
  final GoString held = GoString.fromString(s);
  final Uint8List plain = Uint8List.fromList(held.bytes);

  _run('micro.fromString', 1, tag, filter, () {}, () {
    _keepText = GoString.fromString(s);
  });
  _run('micro.take', 1, tag, filter, () {}, () {
    _keepText = GoString.take(utf8.encode(s));
  });
  _run('micro.toDartString', 1, tag, filter, () {}, () {
    sink += held.toDartString().length;
  });
  _run('micro.decodePlain', 1, tag, filter, () {}, () {
    sink += utf8.decode(plain).length;
  });
}

void _sweepCell(int n, int l, bool nonAscii, String filter) {
  final String tag = nonAscii ? 'u$l' : '$l';

  _run('go.held', n, tag, filter, () {
    bench.Setup(GoInt(n), GoInt(l), nonAscii);
  }, () {
    _keep = bench.TreeHeld();
  });

  _run('go.literal', n, tag, filter, () {
    bench.Setup(GoInt(n), GoInt(l), nonAscii);
  }, () {
    _keep = bench.TreeLiteral();
  });

  _run('go.itoa', n, tag, filter, () {
    bench.Setup(GoInt(n), GoInt(l), nonAscii);
  }, () {
    _keep = bench.TreeItoa();
  });

  Iface? tree;
  _run('dart.walk.text', n, tag, filter, () {
    bench.Setup(GoInt(n), GoInt(l), nonAscii);
    tree = bench.TreeLiteral();
  }, () {
    walkText(tree);
  });

  _run('dart.walk.raw', n, tag, filter, () {
    bench.Setup(GoInt(n), GoInt(l), nonAscii);
    tree = bench.TreeLiteral();
  }, () {
    walkRaw(tree);
  });

  _run('dart.walk.data', n, tag, filter, () {
    bench.Setup(GoInt(n), GoInt(l), nonAscii);
    tree = bench.TreeLiteral();
  }, () {
    walkData(tree);
  });

  _run('frame.go', n, tag, filter, () {
    bench.Setup(GoInt(n), GoInt(l), nonAscii);
  }, () {
    walkText(bench.TreeLiteral());
  });

  _run('frame.go.itoa', n, tag, filter, () {
    bench.Setup(GoInt(n), GoInt(l), nonAscii);
  }, () {
    walkText(bench.TreeItoa());
  });

  if (nonAscii) return;

  _run('frame.dart', n, tag, filter, () {}, () {
    dartFrameLiteral(n, l);
  });

  _run('frame.dart.concat', n, tag, filter, () {}, () {
    dartFrameConcat(n, l);
  });
}

/// _sweepDiff measures the reconciler of A10 against the plain walk of A5.
void _sweepDiff(int n, int l, String filter) {
  Iface? base;
  Iface? same;
  Iface? one;
  Iface? all;

  void hold() {
    bench.Setup(GoInt(n), GoInt(l), false);
    base = bench.TreePick(GoInt(0));
    same = bench.TreePick(GoInt(0));
    one = bench.TreePick(GoInt(1));
    all = bench.TreePick(GoInt(n));
  }

  _run('diff.same', n, '$l', filter, hold, () {
    changed.clear();
    diffFields(base, same);
    _sinkCount += changed.length;
  });

  _run('diff.one', n, '$l', filter, hold, () {
    changed.clear();
    diffFields(base, one);
    _sinkCount += changed.length;
  });

  _run('diff.all', n, '$l', filter, hold, () {
    changed.clear();
    diffFields(base, all);
    _sinkCount += changed.length;
  });

  _run('diff.eq.same', n, '$l', filter, hold, () {
    changed.clear();
    diffEqual(base, same);
    _sinkCount += changed.length;
  });

  _run('diff.eq.all', n, '$l', filter, hold, () {
    changed.clear();
    diffEqual(base, all);
    _sinkCount += changed.length;
  });

  // The 4 frames below hold the Go build too, thus the pair of A5 and A10
  // answers the question of the frame and not of the diff alone.
  _run('frame.a5', n, '$l', filter, () {
    bench.Setup(GoInt(n), GoInt(l), false);
  }, () {
    walkText(bench.TreePick(GoInt(0)));
  });

  _frame('frame.a10.same', n, l, 0, filter);
  _frame('frame.a10.one', n, l, 1, filter);
  _frame('frame.a10.all', n, l, n, filter);
}

Iface? _prev;
int _phase = 0;

/// _frame runs 1 frame of A10: Go builds the new tree, the reconciler diffs it
/// against the tree of the frame before, and the host builds the nodes that
/// changed. The 2 trees alternate, thus each frame finds k changes.
void _frame(String name, int n, int l, int k, String filter) {
  _run(name, n, '$l', filter, () {
    bench.Setup(GoInt(n), GoInt(l), false);
    _prev = bench.TreePick(GoInt(0));
    _phase = 0;
  }, () {
    _phase = _phase == 0 ? k : 0;
    final Iface? next = bench.TreePick(GoInt(_phase));
    changed.clear();
    diffFields(_prev, next);
    buildChanged();
    _prev = next;
  });
}

void _sweepWatch(int n, int l, String filter) {
  _run('watch.signal', n, '$l', filter, () {
    _mount(n, l, false);
  }, () {
    bench.Bump(GoInt(_next(n)));
    final Slice<GoInt> dirty = bench.Dirty();
    _sinkCount += dirty.len;
  });

  _run('watch.one', n, '$l', filter, () {
    _mount(n, l, false);
  }, _tick);

  _run('watch.three', n, '$l', filter, () {
    _mount(n, l, true);
  }, _tick);
}

/// _mount gives each row of the tree a boundary, exactly as the state of a
/// GoWatch does at its first build.
void _mount(int n, int l, bool wide) {
  bench.Setup(GoInt(n), GoInt(l), false);
  final Iface root = bench.TreeWatch(wide)!;
  final List<Iface?> kids = kidsOf(root);
  _builds = List<Object?>.filled(kids.length, null);
  for (int i = 0; i < kids.length; i++) {
    final int id = ui.NewWatch();
    if (id != i) {
      throw StateError('the boundary of row $i took the id $id, and the '
          'harness reads the id as an index');
    }
    final Object? build = buildOf(kids[i]!);
    _builds[id] = build;
    ui.RunWatch(GoInt(id), build as Iface? Function()?);
  }
  ui.TakeDirty();
  _row = 0;
}

/// _tick runs 1 frame of the normal case: 1 signal changes, and the host
/// rebuilds the 1 boundary that read it.
void _tick() {
  bench.Bump(GoInt(_next(_builds.length)));
  final Slice<GoInt> dirty = bench.Dirty();
  for (int i = 0; i < dirty.len; i++) {
    final int id = dirty[i];
    walkText(ui.RunWatch(GoInt(id), _builds[id] as Iface? Function()?));
  }
}

int _next(int n) {
  _row++;
  if (_row >= n) _row = 0;
  return _row;
}

Iface? _keep;
GoString? _keepText;
int _sinkCount = 0;

/// _run measures 1 case. It prints 1 line for each batch, thus the report can
/// read the spread and the median.
void _run(
  String name,
  int n,
  String tag,
  String filter,
  void Function() setup,
  void Function() op,
) {
  if (filter.isNotEmpty && !name.contains(filter)) return;
  setup();

  final Stopwatch warm = Stopwatch()..start();
  int warmIters = 0;
  while (warm.elapsedMicroseconds < warmUs) {
    op();
    warmIters++;
  }
  warm.stop();

  final double roughNs = warm.elapsedMicroseconds * 1000 / warmIters;
  int iters = (minBatchUs * 1000 / roughNs).ceil();
  if (iters < 1) iters = 1;

  for (int b = 0; b < batches; b++) {
    final Stopwatch s = Stopwatch()..start();
    for (int i = 0; i < iters; i++) {
      op();
    }
    s.stop();
    final double ns = s.elapsedMicroseconds * 1000 / iters;
    print('$name,$n,$tag,${ns.toStringAsFixed(2)}');
  }

  // The 2 reads keep the work of the case alive under the tree shaker.
  if (_keep != null && _sinkCount < 0) print(_keep!.desc.name);
  if (_keepText != null && _sinkCount < 0) print(_keepText!.length);
  if (sink < 0 || dartSink < 0) print('the sink went negative');
}

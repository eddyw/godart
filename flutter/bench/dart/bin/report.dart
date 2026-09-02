import 'dart:io';

/// report reads the CSV that bin/bench.dart writes over each process run, and
/// it prints the tables of the measurement.
///
///     dart run bin/report.dart ../out.csv
///
/// It gives 2 statistics for each cell. The median answers the task, and the
/// smallest batch answers a machine that another job shares. A batch that the
/// scheduler interrupts reads high and never low, thus the smallest batch is
/// the best estimate of the true cost. The 2 tables agree on an idle machine.

/// frameUs holds the frame budget at 60 hertz.
const double frameUs = 16700;

late Map<String, List<double>> cells;

void main(List<String> args) {
  if (args.isEmpty) {
    stderr.writeln('report takes the path of the CSV');
    exit(2);
  }
  cells = <String, List<double>>{};
  for (final String line in File(args[0]).readAsLinesSync()) {
    if (line.isEmpty || line.startsWith('#')) continue;
    final List<String> parts = line.split(',');
    if (parts.length != 4) continue;
    cells
        .putIfAbsent('${parts[0]}|${parts[1]}|${parts[2]}', () => <double>[])
        .add(double.parse(parts[3]));
  }

  final Set<String> names = <String>{};
  final Set<int> counts = <int>{};
  final Set<String> tags = <String>{};
  for (final String key in cells.keys) {
    final List<String> p = key.split('|');
    names.add(p[0]);
    counts.add(int.parse(p[1]));
    tags.add(p[2]);
  }
  final List<int> ns = counts.toList()..sort();
  final List<String> ls = tags.toList()..sort(_tagOrder);

  final int samples = cells.values.first.length;
  print('# The measurement of the text property');
  print('');
  print('$samples batches for each cell. Each number is a microsecond.');
  print('');

  for (final bool best in <bool>[true, false]) {
    print('## A rebuild of the whole tree, the '
        '${best ? 'smallest batch' : 'median'}');
    print('');
    print('| Nodes | Text | Go build | Dart walk | Frame | Part of 16,700 us | '
        'Dart baseline | GoString share |');
    print('|---|---|---|---|---|---|---|---|');
    for (final int n in ns) {
      for (final String l in ls) {
        final double? frame = _pick('frame.go', n, l, best);
        if (frame == null) continue;
        final double build = _pick('go.literal', n, l, best) ?? 0;
        final double walk = _pick('dart.walk.text', n, l, best) ?? 0;
        final double held = _pick('go.held', n, l, best) ?? 0;
        final double raw = _pick('dart.walk.raw', n, l, best) ?? 0;
        final double base = _pick('frame.dart', n, l, best) ?? double.nan;
        final double share = (build - held) + (walk - raw);
        print('| $n | ${_label(l)} | ${_us(build)} | ${_us(walk)} | '
            '${_us(frame)} | ${_pct(frame)} | ${_us(base)} | '
            '${_us(share)} (${_pct(share)}) |');
      }
    }
    print('');

    print('## A rebuild of 1 boundary, the '
        '${best ? 'smallest batch' : 'median'}');
    print('');
    print('| Nodes in the tree | Text | 1 text node | 3 text nodes | '
        'The signal alone | Part of 16,700 us, 1 node |');
    print('|---|---|---|---|---|---|');
    for (final int n in ns) {
      for (final String l in ls) {
        final double? one = _pick('watch.one', n, l, best);
        if (one == null) continue;
        final double three = _pick('watch.three', n, l, best) ?? double.nan;
        final double sig = _pick('watch.signal', n, l, best) ?? double.nan;
        print('| $n | ${_label(l)} | ${_us(one)} | ${_us(three)} | '
            '${_us(sig)} | ${_pct(one)} |');
      }
    }
    print('');
  }

  for (final bool best in <bool>[true, false]) {
    print('## The reconciler of A10 against the walk of A5, the '
        '${best ? 'smallest batch' : 'median'}');
    print('');
    print('| Nodes | Text | The diff, 0 changed | The diff, 1 changed | '
        'The diff, each changed | The `==` diff, 0 changed | '
        'A5, the whole frame | A10, 0 changed | A10, 1 changed | '
        'A10, each changed |');
    print('|---|---|---|---|---|---|---|---|---|---|');
    for (final int n in ns) {
      for (final String l in ls) {
        final double? a5 = _pick('frame.a5', n, l, best);
        if (a5 == null) continue;
        print('| $n | ${_label(l)} | ${_us(_pick('diff.same', n, l, best) ?? double.nan)} | '
            '${_us(_pick('diff.one', n, l, best) ?? double.nan)} | '
            '${_us(_pick('diff.all', n, l, best) ?? double.nan)} | '
            '${_us(_pick('diff.eq.same', n, l, best) ?? double.nan)} | '
            '${_us(a5)} | '
            '${_us(_pick('frame.a10.same', n, l, best) ?? double.nan)} | '
            '${_us(_pick('frame.a10.one', n, l, best) ?? double.nan)} | '
            '${_us(_pick('frame.a10.all', n, l, best) ?? double.nan)} |');
      }
    }
    print('');
  }

  print('## Where the time of a whole tree rebuild goes, the smallest batch');
  print('');
  print('| Nodes | Text | Go structure | Go fromString | Dart, the text field |'
      ' Dart, the other fields | Sum | The measured frame |');
  print('|---|---|---|---|---|---|---|---|');
  for (final int n in ns) {
    for (final String l in ls) {
      final double? frame = _pick('frame.go', n, l, true);
      if (frame == null) continue;
      final double held = _pick('go.held', n, l, true) ?? 0;
      final double lit = _pick('go.literal', n, l, true) ?? 0;
      final double data = _pick('dart.walk.data', n, l, true) ?? 0;
      final double text = _pick('dart.walk.text', n, l, true) ?? 0;
      print('| $n | ${_label(l)} | ${_us(held)} | ${_us(lit - held)} | '
          '${_us(data)} | ${_us(text - data)} | ${_us(held + (lit - held) + text)} | '
          '${_us(frame)} |');
    }
  }
  print('');

  print('## Each case');
  print('');
  print('| Case | Nodes | Text | Smallest | Quarter | Median | '
      '3 quarters | 9 tenths | Largest |');
  print('|---|---|---|---|---|---|---|---|---|');
  final List<String> ordered = names.toList()..sort();
  for (final String name in ordered) {
    for (final int n in ns) {
      for (final String l in ls) {
        final List<double>? v = cells['$name|$n|$l'];
        if (v == null) continue;
        final List<double> s = v.toList()..sort();
        print('| $name | $n | ${_label(l)} | ${_us(s.first)} | '
            '${_us(_at(s, 0.25))} | ${_us(_med(s))} | ${_us(_at(s, 0.75))} | '
            '${_us(_at(s, 0.90))} | ${_us(s.last)} |');
      }
    }
  }
}

int _tagOrder(String a, String b) {
  final int ka =
      a.startsWith('u') ? 1000 + int.parse(a.substring(1)) : int.parse(a);
  final int kb =
      b.startsWith('u') ? 1000 + int.parse(b.substring(1)) : int.parse(b);
  return ka.compareTo(kb);
}

String _label(String tag) =>
    tag.startsWith('u') ? '${tag.substring(1)} runes, 37 bytes' : tag;

double? _pick(String name, int n, String l, bool best) {
  final List<double>? v = cells['$name|$n|$l'];
  if (v == null) return null;
  final List<double> s = v.toList()..sort();
  return best ? s.first : _med(s);
}

/// _at gives the value at a share of the sorted samples. The largest batch
/// holds an interrupt of the scheduler, thus the report gives the quarters too.
double _at(List<double> sorted, double share) {
  int i = (sorted.length * share).floor();
  if (i >= sorted.length) i = sorted.length - 1;
  return sorted[i];
}

double _med(List<double> sorted) {
  final int n = sorted.length;
  if (n.isOdd) return sorted[n ~/ 2];
  return (sorted[n ~/ 2 - 1] + sorted[n ~/ 2]) / 2;
}

String _us(double ns) {
  if (ns.isNaN) return '-';
  final double us = ns / 1000;
  if (us >= 10) return us.toStringAsFixed(1);
  if (us >= 1) return us.toStringAsFixed(2);
  return us.toStringAsFixed(3);
}

String _pct(double ns) {
  if (ns.isNaN) return '-';
  final double p = ns / 1000 / frameUs * 100;
  if (p >= 1) return '${p.toStringAsFixed(1)}%';
  return '${p.toStringAsFixed(2)}%';
}

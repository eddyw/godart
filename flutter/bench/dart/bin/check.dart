import 'package:bench_harness/diff.dart';
import 'package:gobench/bench.dart' as bench;
import 'package:goruntime/goruntime.dart';

/// check proves what each diff path reports. It runs no measurement, and it
/// answers whether a diff that reads the generated `==` is correct.
///
///     dart run bin/check.dart

void main() {
  bench.$init();
  bench.Setup(GoInt(100), GoInt(20), false);

  _row('plain, nothing changed', bench.TreePick(GoInt(0))!,
      bench.TreePick(GoInt(0))!);
  _row('plain, 1 leaf changed', bench.TreePick(GoInt(0))!,
      bench.TreePick(GoInt(1))!);
  _row('plain, each leaf changed', bench.TreePick(GoInt(0))!,
      bench.TreePick(GoInt(100))!);
  _row('1 optional field set, nothing changed',
      bench.TreePickStyled(GoInt(0))!, bench.TreePickStyled(GoInt(0))!);
  _row('1 optional field set, 1 leaf changed',
      bench.TreePickStyled(GoInt(0))!, bench.TreePickStyled(GoInt(1))!);
}

void _row(String name, Iface a, Iface b) {
  changed.clear();
  diffFields(a, b);
  final int byField = changed.length;
  changed.clear();
  diffEqual(a, b);
  final int byEqual = changed.length;
  print('$name: the field diff reports $byField, and the == diff reports '
      '$byEqual, of 100 leaves');
}

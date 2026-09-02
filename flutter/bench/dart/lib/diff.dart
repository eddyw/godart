import 'package:goruntime/goruntime.dart';

import 'walk.dart';

/// This file holds the Dart reconciler of approach A10. It keeps the tree of
/// the frame before, it compares the new tree against it, and it gives the
/// nodes that changed. The host then builds a widget for those nodes alone.
///
/// The comparison reads the same descriptor path as walk.dart, thus the 2
/// numbers compare. A node of the Go tree is a fresh Dart object at each build,
/// because `box` of `internal/emit/iface.go:789` writes `Iface(desc, literal)`
/// over a fresh composite literal. Thus no identity test can stop the walk, and
/// the diff must read each field of both trees.

/// changed collects the node of the new tree at each place that differs.
final List<Iface> changed = <Iface>[];

/// diffFields compares 2 trees field by field, and it reads a pointer field by
/// its value. A Go build allocates a fresh pointer for each optional field that
/// it sets, thus a compare by identity would report a change that no reader
/// sees.
///
/// It reports true when the subtree changed.
bool diffFields(Iface? a, Iface? b) {
  if (a == null || b == null) {
    if (a == null && b == null) return false;
    if (b != null) changed.add(b);
    return true;
  }
  final GoNodeView x = GoNodeView(a);
  final GoNodeView y = GoNodeView(b);
  if (x.kind != y.kind) {
    changed.add(b);
    return true;
  }
  switch (y.kind) {
    case 'gart.dev/ui.Text':
      // The compare reads the bytes of a GoString, thus the diff pays no
      // decode and allocates no Dart string. This is the best case for A10.
      final bool same = x.goText('Data') == y.goText('Data') &&
          x.nested('Style') == y.nested('Style') &&
          x.goTextOpt('TextAlign') == y.goTextOpt('TextAlign') &&
          x.goTextOpt('TextDirection') == y.goTextOpt('TextDirection') &&
          x.goTextOpt('Locale') == y.goTextOpt('Locale') &&
          x.flagOpt('SoftWrap') == y.flagOpt('SoftWrap') &&
          x.goTextOpt('Overflow') == y.goTextOpt('Overflow') &&
          x.sizeOpt('TextScaleFactor') == y.sizeOpt('TextScaleFactor') &&
          x.countOpt('MaxLines') == y.countOpt('MaxLines') &&
          x.goTextOpt('SemanticsLabel') == y.goTextOpt('SemanticsLabel') &&
          x.goTextOpt('SemanticsIdentifier') == y.goTextOpt('SemanticsIdentifier') &&
          x.goTextOpt('TextWidthBasis') == y.goTextOpt('TextWidthBasis') &&
          x.countOpt('SelectionColor') == y.countOpt('SelectionColor');
      if (same) return false;
      changed.add(b);
      return true;
    case 'gart.dev/ui.Column':
    case 'gart.dev/ui.Row':
      bool moved = x.goTextOpt('MainAxisAlignment') != y.goTextOpt('MainAxisAlignment') ||
          x.goTextOpt('MainAxisSize') != y.goTextOpt('MainAxisSize') ||
          x.goTextOpt('CrossAxisAlignment') != y.goTextOpt('CrossAxisAlignment') ||
          x.goTextOpt('TextDirection') != y.goTextOpt('TextDirection') ||
          x.goTextOpt('VerticalDirection') != y.goTextOpt('VerticalDirection') ||
          x.goTextOpt('TextBaseline') != y.goTextOpt('TextBaseline') ||
          x.sizeOpt('Spacing') != y.sizeOpt('Spacing');
      final List<Iface?> old = x.children('Children');
      final List<Iface?> fresh = y.children('Children');
      // A child list of another length rebuilds the whole subtree. A real
      // reconciler reads a key here, and it moves the state of a child that
      // kept its key. This harness holds no key.
      if (old.length != fresh.length) {
        changed.add(b);
        return true;
      }
      for (int i = 0; i < fresh.length; i++) {
        if (diffFields(old[i], fresh[i])) moved = true;
      }
      if (moved) return true;
      return false;
  }
  throw StateError('the diff holds no case for the node ${y.kind}');
}

/// diffEqual reads the generated `==` of a node that Go compares, and it falls
/// back to the field path for a node that holds a slice.
///
/// `Text$.operator ==` of the generated Dart compares a pointer field with
/// `identical`, because Go compares a pointer by its address. Thus this path
/// reads a change at each frame for a node that sets an optional field, and
/// check.dart proves it.
bool diffEqual(Iface? a, Iface? b) {
  if (a == null || b == null) {
    if (a == null && b == null) return false;
    if (b != null) changed.add(b);
    return true;
  }
  if (a.desc.name != b.desc.name) {
    changed.add(b);
    return true;
  }
  if (b.desc.comparable) {
    if (a.value == b.value) return false;
    changed.add(b);
    return true;
  }
  final GoNodeView x = GoNodeView(a);
  final GoNodeView y = GoNodeView(b);
  final List<Iface?> old = x.children('Children');
  final List<Iface?> fresh = y.children('Children');
  if (old.length != fresh.length) {
    changed.add(b);
    return true;
  }
  bool moved = false;
  for (int i = 0; i < fresh.length; i++) {
    if (diffEqual(old[i], fresh[i])) moved = true;
  }
  return moved;
}

/// buildChanged stands for the widget build of the nodes that the diff named.
/// It runs the same read as walkText over each 1.
void buildChanged() {
  for (final Iface node in changed) {
    walkText(node);
  }
}

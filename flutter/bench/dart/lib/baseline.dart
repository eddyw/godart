/// This file holds the Dart baseline. A screen that a developer writes in Dart
/// pays only this, thus the difference against the Go path is the price of the
/// boundary.

/// dartSink holds the result of each pass, for the same reason as sink of
/// walk.dart.
int dartSink = 0;

sealed class DNode {}

final class DText extends DNode {
  final String data;

  DText(this.data);
}

final class DColumn extends DNode {
  final List<DNode> children;

  DColumn(this.children);
}

const String dLit5 = 'Ready';
const String dLit20 = 'Total price in euros';
const String dLit80 =
    'The order of 2026-09-02 holds 4 lines, and the driver waits for the answer today';

String dLiteral(int l) {
  switch (l) {
    case 5:
      return dLit5;
    case 20:
      return dLit20;
  }
  return dLit80;
}

/// dartFrameLiteral builds the same shape from a Dart literal and reads it
/// back. A Dart literal costs nothing at run time.
void dartFrameLiteral(int n, int l) {
  final DColumn root = DColumn(<DNode>[
    for (int i = 0; i < n; i++) DText(dLiteral(l)),
  ]);
  dartWalk(root);
}

/// dartFrameConcat builds the same shape from a concatenation of a literal and
/// a number. The measurement of 2026-09-01 holds this shape.
void dartFrameConcat(int n, int l) {
  final DColumn root = DColumn(<DNode>[
    for (int i = 0; i < n; i++) DText('${dLiteral(l)}$i'),
  ]);
  dartWalk(root);
}

void dartWalk(DNode node) {
  switch (node) {
    case DText():
      dartSink += node.data.length;
    case DColumn():
      for (final DNode kid in node.children) {
        dartWalk(kid);
      }
  }
}

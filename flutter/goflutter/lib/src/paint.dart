import 'package:flutter/rendering.dart';
import 'package:goruntime/goruntime.dart';

import 'node.dart';

CustomPainter? goPainter(GoNode? node) =>
    node == null ? null : _GoPainter(node);

class _GoPainter extends CustomPainter {
  _GoPainter(this.node);

  final GoNode node;

  @override
  void paint(Canvas canvas, Size size) {
    final callback = node.raw('Paint', 0) as Function;
    final descriptor = node.desc.fields
        .firstWhere((f) => f.name == 'Paint')
        .type()
        .params
        .first();
    final value = descriptor.zero!();
    var valid = true;
    void draw(void Function(Paint) operation, GoInt color) {
      if (!valid) throw StateError('Canvas escaped its paint callback');
      operation(Paint()..color = Color($ix(color)));
    }

    final callbacks = <String, Function>{
      'rectangle':
          (double left, double top, double right, double bottom, GoInt color) =>
              draw(
                (paint) => canvas.drawRect(
                  Rect.fromLTRB(left, top, right, bottom),
                  paint,
                ),
                color,
              ),
      'circle': (double x, double y, double radius, GoInt color) => draw(
        (paint) => canvas.drawCircle(Offset(x, y), radius, paint),
        color,
      ),
      'line':
          (
            double x1,
            double y1,
            double x2,
            double y2,
            double width,
            GoInt color,
          ) => draw(
            (paint) => canvas.drawLine(
              Offset(x1, y1),
              Offset(x2, y2),
              paint..strokeWidth = width,
            ),
            color,
          ),
    };
    for (final field in descriptor.fields) {
      field.set(value, callbacks[field.name]);
    }
    try {
      Function.apply(callback, [value, size.width, size.height]);
    } finally {
      valid = false;
    }
  }

  @override
  bool shouldRepaint(covariant _GoPainter oldDelegate) =>
      node.count('Revision', 1) != oldDelegate.node.count('Revision', 1);
}

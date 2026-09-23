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
          (
            GoFloat64 left,
            GoFloat64 top,
            GoFloat64 right,
            GoFloat64 bottom,
            GoInt color,
          ) => draw(
            (paint) => canvas.drawRect(
              Rect.fromLTRB(
                left.toDouble(),
                top.toDouble(),
                right.toDouble(),
                bottom.toDouble(),
              ),
              paint,
            ),
            color,
          ),
      'circle': (GoFloat64 x, GoFloat64 y, GoFloat64 radius, GoInt color) =>
          draw(
            (paint) => canvas.drawCircle(
              Offset(x.toDouble(), y.toDouble()),
              radius.toDouble(),
              paint,
            ),
            color,
          ),
      'line':
          (
            GoFloat64 x1,
            GoFloat64 y1,
            GoFloat64 x2,
            GoFloat64 y2,
            GoFloat64 width,
            GoInt color,
          ) => draw(
            (paint) => canvas.drawLine(
              Offset(x1.toDouble(), y1.toDouble()),
              Offset(x2.toDouble(), y2.toDouble()),
              paint..strokeWidth = width.toDouble(),
            ),
            color,
          ),
    };
    for (final field in descriptor.fields) {
      field.set(value, callbacks[field.name]);
    }
    try {
      Function.apply(callback, [value, $f64(size.width), $f64(size.height)]);
    } finally {
      valid = false;
    }
  }

  @override
  bool shouldRepaint(covariant _GoPainter oldDelegate) =>
      node.count('Revision', 1) != oldDelegate.node.count('Revision', 1);
}

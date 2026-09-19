import 'package:flutter/widgets.dart';

/// The tree of the measurement.
///
/// The root builds a fresh Column at each frame, thus Flutter walks each
/// child. The children of the fresh case are new objects, and the children of
/// the same case are the objects of the frame before.
///
/// Element.updateChild ends at the first test when the new widget is the
/// object that the element already holds. The difference of the 2 cases is
/// thus the Flutter work that reuse of 1 unchanged widget avoids.
class SkipTree extends StatefulWidget {
  const SkipTree({super.key, required this.nodes, required this.reuse});

  /// The count of the children.
  final int nodes;

  /// True gives the same child objects at each frame.
  final bool reuse;

  @override
  State<SkipTree> createState() => SkipTreeState();
}

class SkipTreeState extends State<SkipTree> {
  List<Widget>? _held;

  /// Marks the root, thus the next frame builds the Column again.
  void again() => setState(() {});

  List<Widget> _children() {
    if (widget.reuse) {
      return _held ??= _fresh();
    }
    return _fresh();
  }

  List<Widget> _fresh() => <Widget>[
    for (int i = 0; i < widget.nodes; i++)
      Text('row $i', textDirection: TextDirection.ltr),
  ];

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: _children(),
    );
  }
}

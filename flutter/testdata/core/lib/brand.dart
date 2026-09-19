import 'package:flutter/widgets.dart';

class Badge extends StatelessWidget {
  const Badge({super.key, required this.label, this.child, this.onPressed});

  final String label;
  final Widget? child;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) => GestureDetector(
    onTap: onPressed,
    child: Column(children: [Text(label), ?child]),
  );
}

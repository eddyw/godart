import 'package:flutter/widgets.dart';

class Badge extends StatelessWidget {
  const Badge({super.key, required this.label});

  final String label;

  @override
  Widget build(BuildContext context) => Text('other: $label');
}

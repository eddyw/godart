import 'package:cached_network_image/cached_network_image.dart';
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';
import 'package:material_ui/material_ui.dart';
import 'package:octo_image/octo_image.dart';

import 'main.dart' show shareButtonKey;
import 'photos.dart';
import 'tag_chip.dart';
import 'theme/colors.dart';

Widget? compassWidget(GoNode node, GoRuntime runtime) {
  switch (node.kind) {
    case 'example.dev/gocompass_v2.Label':
      return Builder(builder: (context) => _label(context, node));
    case 'example.dev/gocompass_v2.Photo':
      return Builder(builder: (context) => _photo(context, node));
    case 'example.dev/gocompass_v2.Tag':
      return Builder(
        builder: (context) {
          final theme = Theme.of(context);
          final large = node.flag('Large');
          return TagChip(
            tag: node.text('Text'),
            fontSize: large ? 16 : 10,
            height: large ? 32 : 20,
            chipColor: large
                ? theme.brightness == Brightness.dark
                      ? AppColors.whiteTransparent
                      : AppColors.blackTransparent
                : null,
            onChipColor: large ? theme.colorScheme.onSurface : null,
          );
        },
      );
    case 'example.dev/gocompass_v2.Input':
      return _Input(node: node, runtime: runtime);
    case 'example.dev/gocompass_v2.RouteView':
      final route = node.text('Route');
      final home = goRender(node.child('Home'), runtime);
      final screen = GoAnchor(child: goRender(node.child('Screen'), runtime));
      return Navigator(
        pages: [
          if (route != 'login')
            MaterialPage<void>(
              key: const ValueKey('home'),
              child: route == 'home' ? screen : home,
            ),
          if (route != 'home')
            MaterialPage<void>(
              key: ValueKey(route),
              child: PopScope<void>(
                canPop: false,
                onPopInvokedWithResult: (didPop, _) {
                  if (!didPop) runtime.run(node.action('OnBack')!);
                },
                child: screen,
              ),
            ),
        ],
        onDidRemovePage: (_) {},
      );
    case 'example.dev/gocompass_v2.TripButton':
      final share = node.flag('Share');
      final button = FloatingActionButton.extended(
        key: ValueKey(share ? 'share-button' : 'booking-button'),
        heroTag: null,
        onPressed: () => runtime.run(node.action('OnPressed')!),
        label: Text(share ? 'Share Trip' : 'Book New Trip'),
        icon: Icon(share ? Icons.share_outlined : Icons.add_location_outlined),
      );
      return share ? KeyedSubtree(key: shareButtonKey, child: button) : button;
  }
  return null;
}

Widget _label(BuildContext context, GoNode node) {
  final theme = Theme.of(context);
  final t = theme.textTheme;
  TextStyle? style = switch (node.text('Role')) {
    'headlineLarge' => t.headlineLarge,
    'headlineSmall' => t.headlineSmall,
    'titleLarge' => t.titleLarge,
    'titleMedium' => t.titleMedium,
    'bodyLarge' => t.bodyLarge,
    'bodyMedium' => t.bodyMedium,
    'bodySmall' => t.bodySmall,
    'labelLarge' => t.labelLarge,
    'labelSmall' => t.labelSmall,
    'hint' => theme.inputDecorationTheme.hintStyle,
    'continent' => const TextStyle(
      fontFamily: 'OpenSans',
      fontSize: 18,
      fontWeight: FontWeight.w500,
      color: AppColors.white1,
    ),
    'cardTitle' => const TextStyle(
      fontFamily: 'Rubik',
      fontWeight: FontWeight.w800,
      fontSize: 15,
      color: Colors.white,
      letterSpacing: 1,
      shadows: [Shadow(blurRadius: 3, color: Colors.black)],
    ),
    _ => null,
  };
  if (node.text('Font') == 'Rubik') {
    style = style!.copyWith(fontFamily: 'Rubik');
  }
  final maxLines = node.count('MaxLines');
  return Text(
    node.text('Text'),
    style: style,
    textAlign: node.text('Align') == 'center' ? TextAlign.center : null,
    maxLines: maxLines == 0 ? null : maxLines,
    overflow: maxLines == 0 ? null : TextOverflow.ellipsis,
  );
}

Widget _photo(BuildContext context, GoNode node) {
  final placeholder = node.text('Placeholder');
  final width = node.size('Width');
  final height = node.size('Height');
  final asset = bundledPhotos[node.text('URL')];
  if (asset != null) {
    return OctoImage(
      image: AssetImage(asset),
      placeholderBuilder: (_) => Container(),
      fit: BoxFit.cover,
      filterQuality: FilterQuality.low,
      color: node.flag('Darken') ? Colors.black.withValues(alpha: 0.5) : null,
      colorBlendMode: node.flag('Darken') ? BlendMode.darken : null,
    );
  }
  return CachedNetworkImage(
    imageUrl: node.text('URL'),
    width: width == 0 ? null : width,
    height: height == 0 ? null : height,
    fit: switch (node.text('Fit')) {
      'cover' => BoxFit.cover,
      'fitHeight' => BoxFit.fitHeight,
      'fitWidth' => BoxFit.fitWidth,
      _ => null,
    },
    color: node.flag('Darken') ? Colors.black.withValues(alpha: 0.5) : null,
    colorBlendMode: node.flag('Darken') ? BlendMode.darken : null,
    errorListener: (_) {},
    errorWidget: switch (placeholder) {
      'continent' => (_, _, _) => const ColoredBox(
        color: AppColors.grey3,
        child: SizedBox(width: 140, height: 140),
      ),
      'result' => (_, _, _) => const Icon(Icons.error),
      'activity' => (_, _, _) => Container(
        width: 80,
        height: 80,
        decoration: BoxDecoration(
          color: Colors.grey[200],
          borderRadius: BorderRadius.circular(8),
        ),
        child: Icon(
          Icons.broken_image_outlined,
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
      _ => null,
    },
  );
}

class _Input extends StatefulWidget {
  const _Input({required this.node, required this.runtime});

  final GoNode node;
  final GoRuntime runtime;

  @override
  State<_Input> createState() => _InputState();
}

class _InputState extends State<_Input> {
  late final _controller = TextEditingController(
    text: widget.node.text('Value'),
  );

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => TextField(
    controller: _controller,
    obscureText: widget.node.flag('Obscure'),
    onChanged: (value) => widget.runtime.run(
      () => widget.node.onText('OnChanged')!(GoString.fromString(value)),
    ),
  );
}

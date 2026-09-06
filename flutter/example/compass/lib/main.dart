import 'package:goflutter/goflutter.dart';
import 'package:material_ui/material_ui.dart';
import 'package:share_plus/share_plus.dart';

import 'bridge.dart';
import 'widgets_gen.dart';

void main() {
  goGenInstall();
  goSetCall(compassCall);
  runApp(const GoApp(bridge: CompassBridge()));
}

/// compassCall answers the 1 call that the host of goflutter does not hold.
///
/// The Go source calls ui.Share, and a share sheet needs a plugin of the
/// platform. Each other name reaches goHostCall.
Future<String> compassCall(String name, String arg) async {
  if (name == 'share') {
    await SharePlus.instance.share(ShareParams(text: arg));
    return '';
  }
  return goHostCall(name, arg);
}

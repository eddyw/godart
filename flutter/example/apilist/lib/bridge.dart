import 'package:goapilist/app.dart' as app;
import 'package:goapilist/dep/gart.dev/ui.dart' as ui;
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';

/// ApiListBridge forwards each call of the host into the Dart output of the Go
/// module. The generator of milestone 3 writes this class.
final class ApiListBridge implements GoBridge {
  const ApiListBridge();

  @override
  void init() => app.$init();

  @override
  Iface? build() => app.Tree();

  @override
  int newWatch() => ui.NewWatch();

  @override
  Iface? runWatch(int id, Object? build) =>
      ui.RunWatch(GoInt(id), build as Iface? Function()?);

  @override
  void disposeWatch(int id) => ui.DisposeWatch(GoInt(id));

  @override
  List<int> takeDirty() {
    final Slice<GoInt> ids = ui.TakeDirty();
    return <int>[for (int i = 0; i < ids.len; i++) ids[i]];
  }

  @override
  int pump() => ui.Pump();

  @override
  void setTheme({
    required bool dark,
    required int primary,
    required int surface,
    required int onSurface,
    required double textSize,
  }) {
    ui.SetTheme(
      ui.ThemeData$(
        dark,
        GoInt(primary),
        GoInt(surface),
        GoInt(onSurface),
        textSize,
      ),
    );
  }

  @override
  void setMedia({
    required double width,
    required double height,
    required double textScale,
    required double topInset,
  }) {
    ui.SetMedia(ui.MediaData$(width, height, textScale, topInset));
  }
}

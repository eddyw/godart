import 'package:gocore/app.dart' as app;
import 'package:gocore/dep/gart.dev/ui.dart' as ui;
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';

final class CoreBridge implements GoBridge {
  const CoreBridge({this.root});

  final Iface? Function()? root;

  @override
  void init() => app.$init();

  @override
  Iface? build() => root == null ? app.Tree() : root!();

  @override
  int newWatch() => $ix(ui.NewWatch());

  @override
  Iface? runWatch(int id, Object? build) =>
      ui.RunWatch($gi(id), build as Iface? Function()?);

  @override
  void disposeWatch(int id) => ui.DisposeWatch($gi(id));

  @override
  List<int> takeDirty() {
    final Slice<GoInt> ids = ui.TakeDirty();
    return <int>[for (int i = 0; i < ids.len; i++) $ix(ids[i])];
  }

  @override
  int pump() => $ix(ui.Pump());

  @override
  void setTheme({
    required bool dark,
    required int primary,
    required int surface,
    required int onSurface,
    required double textSize,
  }) {
    ui.SetTheme(
      ui.HostTheme$(
        dark,
        $gi(primary),
        $gi(surface),
        $gi(onSurface),
        $f64(textSize),
      ),
    );
  }

  @override
  void setMedia({
    required double width,
    required double height,
    required double textScale,
    required double topInset,
    required double bottomInset,
  }) {
    ui.SetMedia(
      ui.MediaData$(
        $f64(width),
        $f64(height),
        $f64(textScale),
        $f64(topInset),
        $f64(bottomInset),
      ),
    );
  }

  @override
  String takeCalls() => ui.TakeCalls().toDartString();

  @override
  void answer(int id, String value, String error) => ui.Answer(
    $gi(id),
    GoString.fromString(value),
    GoString.fromString(error),
  );
}

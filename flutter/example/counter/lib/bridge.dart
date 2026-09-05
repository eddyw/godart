import 'package:gocounter/app.dart' as app;
import 'package:gocounter/dep/gart.dev/ui.dart' as ui;
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';

/// CounterBridge forwards each call of the host into the Dart output of the Go
/// module. The generator of milestone 3 writes this class.
final class CounterBridge implements GoBridge {
  const CounterBridge();

  @override
  void init() => app.$init();

  @override
  Iface? build() => app.Tree();

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
      ui.ThemeData$(
        dark,
        $gi(primary),
        $gi(surface),
        $gi(onSurface),
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
    required double bottomInset,
  }) {
    ui.SetMedia(
      ui.MediaData$(width, height, textScale, topInset, bottomInset),
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

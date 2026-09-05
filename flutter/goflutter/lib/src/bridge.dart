import 'package:goruntime/goruntime.dart';

/// GoBridge holds each call into the Go library of the application.
///
/// The Dart output of a Go module lives in the package of that module, thus
/// this package cannot import it. The application writes 1 small class that
/// forwards each call, and the generator of milestone 3 writes that class.
abstract interface class GoBridge {
  /// Runs the init of the generated library. The application calls GoApp, and
  /// GoApp calls this 1 time.
  void init();

  /// Gives the tree of the whole application.
  Iface? build();

  /// Gives a new rebuild boundary.
  int newWatch();

  /// Builds 1 boundary under a tracked scope. The build value comes from the
  /// field Build of a Watch node.
  Iface? runWatch(int id, Object? build);

  /// Releases 1 boundary.
  void disposeWatch(int id);

  /// Gives each boundary that must rebuild, and it clears the set. The value
  /// of the constant rootID means the whole tree.
  List<int> takeDirty();

  /// Runs each function that a goroutine posted to the graph.
  int pump();

  /// Writes the theme that the Go build reads.
  void setTheme({
    required bool dark,
    required int primary,
    required int surface,
    required int onSurface,
    required double textSize,
  });

  /// Writes the media query that the Go build reads.
  void setMedia({
    required double width,
    required double height,
    required double textScale,
    required double topInset,
    required double bottomInset,
  });

  /// Gives each request of Go that waits for an answer of the host, as a JSON
  /// array. An empty text means that no request stands. Read
  /// flutter/ui/ask.go.
  String takeCalls();

  /// Writes the answer of 1 request. An empty error means that the call gave
  /// the value.
  void answer(int id, String value, String error);
}

/// rootID names the whole tree in the result of takeDirty. It holds the value
/// of ui.RootID of the Go package.
const int rootID = -1;

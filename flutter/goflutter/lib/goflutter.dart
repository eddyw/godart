/// The Flutter host of a Go widget tree.
///
/// A Go library that reads the package `gart.dev/ui` gives an immutable tree of
/// plain Go structs. This package reads that tree through its type descriptors,
/// and it builds real Flutter widgets from it. Flutter then owns the element
/// tree, the key, the State of each widget, the focus, the lazy list, and the
/// error boundary.
///
/// The application writes 1 class that implements GoBridge, and it puts GoApp
/// in its widget tree.
library;

export 'src/app.dart' show GoApp;
export 'src/bridge.dart' show GoBridge, rootID;
export 'src/convert.dart';
export 'src/host.dart'
    show GoTextField, GoWatch, GoWidgets, goIcons, goRender, goSetWidgets;
export 'src/identity.dart'
    show
        GoIdentities,
        GoKeep,
        goAnimation,
        goFocusNode,
        goFocusScope,
        goLayerLink,
        goMenuController,
        goOverlapHandle,
        goOverlayController,
        goScrollController,
        goStatesController,
        goStorageBucket,
        goTextController;
export 'src/node.dart' show GoNode;
export 'src/runtime.dart' show GoRuntime;

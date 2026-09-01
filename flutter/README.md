# Flutter

This directory holds milestone 1 of `docs/flutter-design.md`: a Go tree of plain
structs, and a Dart host that builds real Flutter widgets from it.
`docs/flutter-state.md` holds the state model.

`make ci` does not run this directory, because it needs the Flutter SDK. The
Go module `flutter/ui` still passes `gofmt` and `go vet` through the root
targets.

| Path | Content |
|---|---|
| `ui/` | The Go module `gart.dev/ui`. The signal graph, the state shapes, and 28 node types. |
| `goflutter/` | The Dart package that hosts a Go tree in Flutter. |
| `example/counter/` | The proof application. Go holds the whole screen. |

## Run the example

```sh
make build                          # from the root of the repository
cd flutter/example/counter
./build.sh                          # go vet, go test, and gart build into gen
flutter test                        # 6 widget tests
flutter run                         # on a device or a simulator
```

`build.sh` writes the Dart output of `./go` into `gen`, which the pubspec of the
application names as a path dependency. `gen` stays out of git.

## Test the Go side alone

The state layer needs no Flutter and no Dart.

```sh
cd flutter/ui && go test ./...
cd flutter/example/counter/go && go test ./...
```

## Write an application

1. A Go module that reads `gart.dev/ui`. A struct holds the state in signals, a
   method writes them, and a method gives the tree.
2. A package function that gives the tree of the root. Do not name it `Build`
   when a struct of the package holds a method of that name, because the Dart
   output then takes a suffix.
3. A Dart class that implements `GoBridge`. Copy
   `example/counter/lib/bridge.dart` and change the 2 import lines.
4. `GoApp(bridge: YourBridge())` inside a `MaterialApp`.

A widget outside the node set of `ui` costs 1 Go struct and 1 case of
`goflutter/lib/src/host.dart`. Nothing in the host is specific to Material, thus
a Cupertino widget and a widget of a `pub.dev` package each take the same 2
steps.

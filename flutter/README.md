# Flutter

This directory holds milestone 1 to milestone 3 of `docs/flutter-design.md`: a
Go tree of plain structs, and a Dart host that builds real Flutter widgets from
it. `docs/flutter-state.md` holds the state model, and
`docs/flutter-first.md` holds the 3 programs that prove the path.

`make ci` does not run this directory, because it needs the Flutter SDK. The
Go module `flutter/ui` still passes `gofmt` and `go vet` through the root
targets.

| Path | Content |
|---|---|
| `ui/` | The Go module `gart.dev/ui`. The signal graph, the state shapes, the value types, the identity keys, and 564 node types. |
| `goflutter/` | The Dart package that hosts a Go tree in Flutter. |
| `nodegen/` | The generator. It reads the Flutter API and writes the node set and the host. Read its README. |
| `example/counter/` | Proof program 1. Go holds the whole screen. |
| `example/apilist/` | Proof program 2. Go reads an HTTP API and gives a list. |
| `example/form/` | Proof program 3. Go validates a form and sends it. |
| `example/compass/` | The port of the Flutter sample compass_app. 6 screens, a stack of the navigator, and the server of the sample in Go. Read its README for each limit that the port found. |

## Run an example

```sh
make build                          # from the root of the repository
cd flutter/example/counter
./build.sh                          # go vet, go test, and gart build into gen
flutter test                        # 11 widget tests
```

Run `./build.sh` again after each change of the Go source.

`flutter test` needs no platform runner. `flutter run` needs one, and git holds
no such directory, because `flutter create` writes each file of it.
`example/platform.sh` writes the runner and cleans the template files:

```sh
cd flutter/example
./platform.sh counter macos         # or ios, android, linux, windows, web
cd counter
flutter run -d macos
```

The script also patches the 2 permissions that a network demo needs. A macOS
runner gets the entitlement `com.apple.security.network.client`, and an Android
runner gets `android.permission.INTERNET` in the main manifest. `flutter create`
writes neither 1, thus `apilist` and `form` would reach no server.

| Platform | State of 2026-09-02 |
|---|---|
| macOS | The 3 examples build and run. |
| iOS | The 3 examples build with `flutter build ios --debug --no-codesign`. No plugin stands in the way, thus CocoaPods is not needed. |
| Android | The 3 examples build with `flutter build apk --debug`. The build needs `ANDROID_HOME`, and it does not need the cmdline-tools that `flutter doctor` asks for. |
| Web | Blocked. `GoInt` has no constructor under the JS number model, and `$getenv` and `$readFile` of the runtime have no web form. Read `docs/js-integers.md`. |

`example/apilist` runs the same way, and it holds 6 widget tests. It reads the
issue list of `golang/go` over https, it decodes the answer into a Go struct,
and it gives a `ListView` of cards.

`example/form` holds 7 widget tests. 3 text fields, a validation in Go on each
keystroke, and a POST of the fields as JSON.

`example/compass` holds 5 widget tests and 8 Go tests. It is the reference
architecture of the Flutter team with the whole application in Go: 6 screens,
a stack of the navigator over `ui.Pages`, an API client with a token, and the
server of the sample in Go. `example/compass/README.md` names each limit of
the model that the port found, and each 1 stands on a package outside the
Flutter SDK or on a Flutter API that needs a `BuildContext`.

Each widget test of the 2 points the application at a local server of
`dart:io`, thus no test reaches the network.

A widget test holds a fake clock. A real socket completes in the real zone,
thus a test of a network read gives time to each zone in turn:

```dart
for (int i = 0; i < 200 && !app.Loaded(); i++) {
  await tester.runAsync(
    () => Future<void>.delayed(const Duration(milliseconds: 10)),
  );
  await tester.pump();
}
```

`build.sh` does 3 things. It writes the Dart output of `./go` into `gen`, which
the pubspec of the application names as a path dependency. It then runs
`nodescan`, which reads the type information of the Go tool chain and reports
each node type that the application names. It then runs `nodegen host`, which
writes `lib/widgets_gen.dart` with 1 case for each name of that list. `gen` and
`lib/widgets_gen.dart` each stay out of git.

## Test the Go side alone

The state layer needs no Flutter and no Dart.

```sh
cd flutter/ui && go test ./...
cd flutter/example/counter/go && go test ./...
cd flutter/example/apilist/go && go test ./...
cd flutter/example/form/go && go test ./...
```

## Write an application

1. A Go module that reads `gart.dev/ui`. A struct holds the state in signals, a
   method writes them, and a method gives the tree.
2. A package function that gives the tree of the root. Do not name it `Build`
   when a struct of the package holds a method of that name, because the Dart
   output then takes a suffix.
3. A Dart class that implements `GoBridge`. Copy
   `example/counter/lib/bridge.dart` and change the 2 import lines.
4. A `build.sh` that runs `gart build`, then `nodescan`, then `nodegen host`.
   Copy `example/counter/build.sh`.
5. `goSetWidgets(goGenWidget)` before `runApp`, and in the `setUp` of each
   widget test.
6. `GoApp(bridge: YourBridge())` inside a `MaterialApp`.

The node set holds 564 types of `widgets`, `material`, and `cupertino`. A
Cupertino screen needs no extra step. A widget of a `pub.dev` package costs 1 Go
struct in a package of your own and 1 case that you write, and `goSetWidgets`
takes a function that reads your case first and the generated 1 after it.

A field of a node that a call leaves unset is a pointer, and nil means "the
default of Flutter". Write `ui.P(14.0)` to set one.

## The rule of a builder

`ui.Builder` and `ui.ItemBuilder` are synchronous positions. Flutter calls such
a function and reads the result at once, thus the function must not block. A
build that reads a channel, that runs a `select`, or that calls a function which
does either, fails the build with a source position.

Move the work into a goroutine, and let the build read a signal:

```go
func (m *Model) Load() {
	go func() {
		rows := fetch()          // this may block
		ui.Post(func() {         // 1 goroutine writes each signal
			m.rows.Set(rows)
		})
	}()
}

func (m *Model) Build() ui.Node {  // this never blocks
	return ui.Text{Data: m.rows.Get()[0]}
}
```

A callback with no result, such as `OnPressed`, takes a function that blocks.
The host wraps it as a goroutine, thus no Dart caller waits for it.

A new synchronous position needs a named function type with the directive:

```go
//gart:sync
type RowBuilder func(int) Node
```

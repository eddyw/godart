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
| `ui/` | The Go module `gart.dev/ui`. The signal graph, the state shapes, the value types, the identity keys, and the 3 packs of 566 node types. |
| `goflutter/` | The Dart package that hosts a Go tree in Flutter. |
| `nodegen/` | The generator. It reads the Flutter API and writes the node set and the host. Read its README. |
| `example/counter/` | Proof program 1. Go holds the whole screen. |
| `example/apilist/` | Proof program 2. Go reads an HTTP API and gives a list. |
| `example/form/` | Proof program 3. Go validates a form and sends it. |
| `example/compass/` | Proof program 4. The architecture port of the Flutter sample compass_app. 7 screens, a stack of the navigator, and the server of the sample in Go. Read its README for each limit that the port found. |
| `example/compass_v2/` | Proof program 5. The pixel parity of the same sample, with no server. 32 captures match the pinned reference byte for byte, and the iOS runner stands. |

## Run an example

```sh
make build                          # from the root of the repository
cd flutter/example/counter
glutter gen                         # go vet, go test, and gart build into gen
flutter test                        # 13 widget tests
```

`glutter run -d macos` opens the application, and it holds the Go source under
watch. It writes the platform runner when the directory of the platform is
absent, and it patches the 2 permissions that a network demo needs. A macOS
runner gets the entitlement `com.apple.security.network.client`, and an Android
runner gets `android.permission.INTERNET` in the main manifest. `flutter
create` writes neither 1, thus `apilist` and `form` would reach no server.

`flutter test` needs no platform runner.

## Watch mode

`glutter run` compiles the package that changed and it asks `flutter run` for a
hot reload. The keys of `flutter run` stay, thus `q` stops the application and
the watch together.

| Step of 1 changed line, measured 2026-09-05 | Time |
|---|---|
| The change to the screen, with the counter | 1 second |
| The hot reload alone | 182 milliseconds, and 3 of 1,113 libraries |
| An incremental build of compass | 1.78 seconds, and `dart format` holds 0.3 of them |

The watch build passes `-format=false`, because a hot reload needs no format.
`dart format` reads the files that the build wrote alone, thus a build of the
whole output no longer holds 10.4 seconds of an 11.9 second cycle.

A compile error stops the reload, because `gart watch` writes the error and no
count of compiled packages. The application keeps the code that ran before.

A change that names a new node type of `gart.dev/ui` needs the host again.
glutter runs the scan after the reload, thus such a change reaches the screen
1 reload later. Read `docs/glutter.md`.

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
architecture of the Flutter team with the whole application in Go: 7 screens,
a stack of the navigator over `ui.Pages`, an API client with a token, and the
server of the sample in Go. `example/compass/README.md` names each limit of
the model that the port found, and each 1 stands on a package outside the
Flutter SDK or on a Flutter API that needs a `BuildContext`.

**The 2 Compass examples answer 2 different questions, thus the repository
keeps both.** `example/compass` asks what a real Go architecture can build. It
carries 3,676 lines of Go over MVVM, a repository for each source, a use case,
and a server of 9 routes over `net/http` and `encoding/json`.
`example/compass_v2` asks how close the pixels come. It carries 759 lines of
Go with local data and no server, and `tool/compare_upstream.py` builds the
pinned reference and matches 32 captures byte for byte, in 2 sizes and 2
themes. `example/compass_v2/doc/validation.md` holds that record, and
`doc/parity.json` holds the count. Only compass_v2 holds an iOS runner and a
test of the native share channel.

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

`glutter gen` does 3 things. It writes the Dart output of `./go` into `gen`, which
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
4. Nothing else. `glutter gen` runs `gart build`, then `nodescan`, then
   `nodegen host`, and `glutter new <name>` writes each file above.
5. `goSetWidgets(goGenWidget)` before `runApp`, and in the `setUp` of each
   widget test.
6. `GoApp(bridge: YourBridge())` inside a `MaterialApp`.

The node set holds 566 types in 3 packs. A pack holds the node types of 1 Dart
package, and it stands in a Go package of its own.

| Pack | The Dart package | The Go import | Node types |
|---|---|---|---|
| The core | `package:flutter/widgets.dart` | `gart.dev/ui` | 273 |
| Material | `package:material_ui/material_ui.dart` | `gart.dev/ui/material` | 218 |
| Cupertino | `package:cupertino_ui/cupertino_ui.dart` | `gart.dev/ui/cupertino` | 75 |

Write `ui.Column{}` and `material.Scaffold{}`. The generated host imports the
Dart package of each case that it holds and no other 1, thus an application
that names no Material widget carries no `material_ui`.

A widget of another `pub.dev` package costs 1 Go struct in a package of your
own. Give it the method `IsNode` that answers `ui.Mark`, and it joins the node
set. `goSetWidgets` takes a function that reads your case first and the
generated 1 after it.

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

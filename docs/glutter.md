# glutter

glutter runs a Flutter application whose widget tree comes from Go. It is to
the `flutter` command what `gart` is to the `go` command: 1 tool that holds the
Go build, the generation of the host, and the Flutter command together.

A Go program builds a tree of plain structs of `gart.dev/ui`. The Dart host
reads that tree through its type descriptors, and it builds real Flutter
widgets. Flutter then owns the element tree, each key, the State of each
widget, the focus, the lazy list, and the error boundary. Read
`docs/flutter-design.md` for the model, and `flutter/README.md` for the parts.

## The first application

```sh
make build                    # from the root of the repository
make install                  # or add build/ to the PATH
glutter new hello
cd hello
glutter run -d macos
```

`glutter new` writes 7 files: the pubspec, the analysis options, the Go module
with 1 screen, `lib/main.dart`, `lib/bridge.dart`, and a `.gitignore`. The
application holds a counter, thus the first run gives a screen that works.

`glutter run` writes the platform runner when it is absent, it compiles the Go
module, it writes the host, and it then runs `flutter run`.

## The loop of the day

`glutter run` holds the Go source under watch. Each change of a Go file gives
a new build, and the application takes the new code with a hot reload. The
screen and the state hold.

| Step, measured 2026-09-05 with the counter example | Time |
|---|---|
| A change of 1 Go line to the screen | 1 second |
| The hot reload alone | 182 milliseconds, and 3 of 1,113 libraries |
| The first `glutter gen` | 15.6 seconds, and nodegen holds 7.7 of them |
| The second `glutter gen` | 1.26 seconds |

The keys of `flutter run` stay, because the child reads the terminal of the
user. Press `r` for a hot reload of your own, `R` for a hot restart, and `q` to
stop the application and the watch together.

A change that names a new node type needs the host again, and glutter runs
`nodegen` for you. The scan of the node types runs after the reload, thus it
costs the fast path nothing. Such a change reaches the screen 1 reload later.

## The commands

| Command | What it does |
|---|---|
| `glutter run [-d device]` | The build, the watch, `flutter run`, and a hot reload after each change. |
| `glutter build <target>` | The build, the checks, and then `flutter build`. |
| `glutter test` | The build, the checks, and then `flutter test`. |
| `glutter gen` | The build alone. It writes `gen/` and `lib/widgets_gen.dart`. |
| `glutter create [platform]` | The platform runner, and the 2 permissions that a network call needs. |
| `glutter doctor` | The Flutter of the machine against the node set. |
| `glutter new <name>` | A new application. |
| `glutter version` | The version of the tool. |

Each other argument goes to the `flutter` command with no change, thus
`glutter devices` and `glutter clean` work.

`glutter run` runs `go vet` first. `glutter gen`, `glutter build`, and
`glutter test` also run `go test`, thus a broken test stops a release build.

## The layout

glutter reads the layout from `pubspec.yaml`, and the defaults hold for an
application that `glutter new` wrote:

| Part | The default | What it holds |
|---|---|---|
| The Go module | `go/` | The whole application in Go. |
| The Dart output | `gen/` | What `gart build` writes. git holds none of it. |
| The host | `lib/widgets_gen.dart` | The 1 case for each node type that the application names. |

A `glutter` block of `pubspec.yaml` names another 1:

```yaml
glutter:
  module: go
  out: gen
  host: lib/widgets_gen.dart
```

glutter finds the repository of godart above the working directory, or it
reads the name of the environment variable `GODART_ROOT`. The generator of the
host and the Dart runtime library stand there.

## The node packs

A pack holds the node types of 1 Dart package, and it stands in a Go package of
its own:

| Pack | The Dart package | The Go import | Node types |
|---|---|---|---|
| The core | `package:flutter/widgets.dart` | `gart.dev/ui` | 273 |
| Material | `package:material_ui/material_ui.dart` | `gart.dev/ui/material` | 218 |
| Cupertino | `package:cupertino_ui/cupertino_ui.dart` | `gart.dev/ui/cupertino` | 75 |

```go
import (
	"gart.dev/ui"
	"gart.dev/ui/material"
)

func Tree() ui.Node {
	return material.Scaffold{
		AppBar: material.AppBar{Title: ui.Text{Data: "This is Go"}},
		Body:   ui.Center{Child: ui.Text{Data: "Hello"}},
	}
}
```

The generated host imports the Dart package of each case that it holds and no
other 1. An application that names no Cupertino widget thus carries no
`cupertino_ui`.

A widget of another package of pub.dev, for example `macos_ui`, costs 1 Go
struct in a package of your own. Give it the method `IsNode` that answers
`ui.Mark`, and it joins the node set.

## The version rule

The node set comes from the Flutter of the machine that wrote it. A Flutter
that drops a parameter of a constructor gives a Go field that the host does not
read, and no error stands. nodegen thus writes the version into the head of
each generated file, and `glutter doctor` reads it:

```
The machine holds Flutter 3.47.2 and Dart 3.13.2.
The pack cupertino comes from Flutter 3.47.2 and cupertino_ui 1.0.2.
The pack material comes from Flutter 3.47.2 and material_ui 1.1.1.
The pack ui comes from Flutter 3.47.2.

The node set matches the machine.
```

A difference of the minor version stops the command with the list of each
difference. `nodegen go` writes the node set of the machine again, and
`flutter/nodegen/README.md` holds the command.

Dart needs no such rule. Each pubspec demands `sdk: ^3.13.0`, thus pub refuses
an older SDK. The minimum comes from records, patterns, class modifiers, and
extension types.

## What a Go tree cannot hold

A Go value holds no `BuildContext`, thus no node maps a Flutter API that needs
1. `Theme.of`, `MediaQuery.of`, and `Localizations.of` each need a context.
The host reads the theme and the media query before each build, and
`ui.Theme()` and `ui.Media()` give the snapshot. The strings of an application
live in Go.

A call that gives an answer needs `ui.Call`:

```go
func (a *App) share() error {
	answer, err := ui.Call("share", a.text())
	if err != nil {
		return err
	}
	ui.Post(func() { a.answer.Set(answer) })
	return nil
}
```

```dart
goSetCall((String name, String arg) async {
  switch (name) {
    case 'share':
      return await Share.share(arg);
  }
  throw StateError('no call of the name $name');
});
```

`ui.Call` parks the goroutine and the host wakes it with the answer. A share
sheet, a dialog with an answer, a date picker, and each plugin of a platform
channel reach Go this way. A build must not call it, and the verifier of the
compiler rejects a build that reads a channel.

`flutter/example/compass/README.md` names each limit that a port of a real
application found.

## What to read next

| Page | Content |
|---|---|
| `docs/gart.md` | The compiler and its 6 commands. |
| `flutter/README.md` | The parts of the Flutter model, and the rule of a builder. |
| `flutter/nodegen/README.md` | The generator of the node set and of the host. |
| `docs/flutter-design.md` | Why the model looks like this, and each measurement behind it. |
| `flutter/example/compass/` | A port of the Flutter sample compass_app, with the whole application in Go. |

# Glutter commands

Glutter builds Flutter applications whose widget trees come from Go.
The current installation requires this checkout, Go, Flutter, and its Dart SDK.
The linked compiler uses the same source revision as the Glutter binary.
Glutter does not invoke an unrelated `gart` binary from `PATH`.

## First application

From the repository root:

```sh
make build
fvm install
fvm exec ./build/glutter new hello
cd hello
fvm exec ../build/glutter run -d macos
```

The application contains a Go module, Dart bridge, Flutter package, and local
resource references. `glutter run` creates an absent platform runner, compiles
Go, generates the host, and invokes Flutter. Go edits trigger Flutter hot reload.
Changed string literals take effect without a restart. Existing Go globals and
Flutter state remain intact. Changes to package initializers need a hot restart.

## Commands

| Command | Purpose |
|---|---|
| `new name` | Create an application. |
| `gen` | Compile Go and generate the Dart host. |
| `bind` | Generate application-owned widget and service bindings. |
| `run` | Compile, watch Go source, and invoke `flutter run`. |
| `build` | Compile and invoke `flutter build`. |
| `test` | Compile and invoke `flutter test`. |
| `create` | Generate platform runner files. |
| `doctor` | Compare the Flutter version and generated node stamps. |
| `version` | Print the compiler version. |
| `licenses` | Print bundled compiler notices. |

Arguments after the command pass to Flutter where that command invokes Flutter.
Normal compilation does not execute native application tests or `go vet`.
Run those explicitly when the application needs native validation.

## Application layout

The defaults are `go/` for Go source, `gen/` for generated Dart, and
`lib/widgets_gen.dart` for the generated host. Override them in `pubspec.yaml`:

```yaml
glutter:
  module: go
  out: gen
  host: lib/widgets_gen.dart
```

Glutter finds resources above the application or executable, or through
`GODART_ROOT`. A binary-only install does not yet supply all Flutter resources.

## Libraries and state

Go components return ordinary `ui.Node` values. Shared values, signals, keys,
and controllers remain in `gart.dev/ui`. Existing Material, Cupertino, and SVG
packages remain available.

Use `glutter.bindings.json` and `glutter bind` for application-owned libraries.
Bindings use the application's package configuration and lockfile.
Repeat generation after a Dart API or manifest change.
See the [binding guide](../flutter/nodegen/README.md) for the full contract.

The host cache includes source, dependency, generator, SDK, manifest, and notice
inputs. An unchanged build can reuse the host. A changed API still needs valid
Go declarations from `glutter bind` before Go compilation.

Builders cannot block. Event handlers can start goroutines and use services.
`ui.Watch` and signals limit rebuilds. Flutter owns layout, elements, and rendering.
See [Flutter architecture](flutter-design.md) for lifecycle and callback rules.

## Validation and SDK changes

```sh
make flutter-go
fvm exec make flutter-test
```

The maintained fixture covers the core without local examples.
`doctor` is an explicit diagnostic, not an automatic guarantee of compatibility.
Use the [SDK upgrade procedure](sdk-upgrades.md) before a support claim.

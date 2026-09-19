# Flutter

Glutter combines Go compilation, widget host generation, and Flutter commands.
Go code builds trees of structs from `gart.dev/ui`. The Dart host creates
Flutter widgets from those trees. Go signals control state updates.

| Path | Content |
|---|---|
| `ui/` | Go signals, value types, identity keys, and generated widget structs. |
| `goflutter/` | The Dart host and its conversion and identity tests. |
| `nodegen/` | The widget and host generator. |
| `nodescan/` | The application node scanner. |
| `bench/` | Performance fixtures and measurements. |

Prototype applications under `example/` are local and ignored by Git.
A fresh checkout does not include them. Historical design documents can
refer to those prototypes.

## Create an application

```sh
make build
fvm exec ./build/glutter new hello
cd hello
fvm exec ../build/glutter run -d macos
```

`glutter run` generates the platform runner when necessary and supports
Flutter hot reload. `glutter test` generates the application and invokes
`flutter test`. Neither command runs native Go tests implicitly.

## Validate the core

```sh
make flutter-go
fvm exec make flutter-test
```

These targets require no local examples. Application tests remain available
through `glutter test` in each local application directory.

See [the Glutter guide](../docs/glutter.md) for commands and
[the generator guide](nodegen/README.md) for widget generation.

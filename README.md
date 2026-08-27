# godart

godart compiles Go source code into Dart source code. The command is `gart`.

The compiler supports a defined subset of the Go language. A verifier enforces
the subset at compile time with a clear error and a source position. Code inside
the subset gives the same result in Go and in Dart. A compromise is always a
rejection, never a silent change of meaning.

The output is a set of Dart packages. The output compiles with the Dart
toolchain, on the VM and with AOT. The web target is out of scope, because a
Dart `int` on the web has no 64 bits.

## Start

```bash
make get                      # download the Go and the Dart dependencies
make build                    # write build/gart
cd path/to/go/module
gart build -o dartout -runtime path/to/runtime/goruntime ./...
cd dartout && dart pub get && dart run bin/main.dart
```

`gart watch` compiles the module again after each change of a Go file. It
recompiles only the packages that need a second compile.

## State

The compiler holds all 6 stages. The subset covers:

| Group | Content |
|---|---|
| Values | Each integer width, `float32`, `float64`, `bool`, `string` |
| Containers | Slice, array, map, struct with value semantics |
| Pointers | A pointer to a struct only |
| Types | Method, interface, type assertion, type switch, embedded struct |
| Control | `if`, `for`, `switch`, `fallthrough`, label, `goto` |
| Functions | Closure, variadic, multiple results, named result, `defer` |
| Errors | `panic`. The subset has no `recover`. |
| Concurrency | Goroutine, channel, `select` |
| Generics | Generic function, generic type, generic method of Go 1.27 |

The standard library surface is empty. Read the workstream doc for the plan of
the overlays.

## The 6 stages

| Stage | Package | Work |
|---|---|---|
| 1 Load | `internal/load` | Read the module with `go/packages` and the `dart` tag |
| 2 Verify | `internal/verify` | Reject each construct outside the subset |
| 3 Analyze | `internal/analyze` | Find each function that can block |
| 4 Normalize | `internal/normalize` | Build the intermediate tree of `internal/ir` |
| 5 Emit | `internal/emit` | Write the Dart source code |
| 6 Metadata | `internal/meta` | Write the archive with the 2 hashes |

`internal/driver` runs the stages over a module and writes the Dart package.
`runtime/goruntime` holds the Dart runtime library.

## Tests

```bash
make ci
```

`internal/driver` holds the end to end test. It compiles each Go program of
`testdata/programs`, runs the Dart output, and compares the output with the
output of `go run`. The suite also proves the cache, the early cutoff, and the
byte equal output of 2 equal builds.

## Versions

The project targets Go 1.27.0 and Dart 3.13.0.

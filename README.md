# godart

godart compiles Go source code into Dart source code. The command is `gart`.

The compiler supports a defined subset of the Go language. A verifier enforces
the subset at compile time with a clear error and a source position. Code inside
the subset gives the same result in Go and in Dart. A compromise is always a
rejection, never a silent change of meaning.

The output is a set of Dart packages. The output compiles with the Dart
toolchain: the VM, AOT, Flutter on macOS, iOS, and Android, and the web with
`dart compile js` and `dart compile wasm`. A Dart `int` on dart2js holds 53
bits, thus the emitted code names the type `GoInt` and the JS build takes a
pair of 32-bit halves. A file operation and a socket operation each report the
error that Go reports under `GOOS=js`.

## Start

```bash
make get                      # download the Go and the Dart dependencies
make build                    # write build/gart
cd path/to/go/module
gart run -o dartout ./...
gart test -o dartout ./...
```

The compiler includes its matching Dart runtime. `-runtime <dir>` selects an explicit runtime override.
Source paths are absolute by default. `-trimpath` uses portable module/package paths for reproducible output across checkouts.

`gart build` writes the Dart package and stops. `gart run` compiles it and runs
package main with the Dart VM. `gart test` compiles the `_test.go` files and
runs each test binary with the Dart VM. The output of `gart test` holds the
format of `go test`. `gart test -bench <expression>` runs the benchmarks on an
ahead of time build.

To run 1 of the example programs of this repository:

```bash
make programs                 # list the programs of testdata/programs
make run PROG=ifaces          # compile the program and run the Dart output
make compare PROG=ifaces      # run it with go and with Dart, and diff
make suites                   # list the test modules of testdata/tests
make gart-test SUITE=basic    # run the tests of the module with gart
make compare-tests SUITE=basic  # run go test and gart test, and diff
make bench SUITE=async        # run the benchmarks of the module with gart
```

`gart watch` compiles the module again after each change of a Go file. It
recompiles only the packages that need a second compile.

`docs/gart.md` holds the whole command line.

## A Flutter application

`glutter` is to the `flutter` command what `gart` is to the `go` command. A Go
program builds a tree of plain structs, and the Dart host builds real Flutter
widgets from it.

```bash
make build
make install
glutter new hello
cd hello
glutter run -d macos
```

`glutter run` holds the Go source under watch. A change of 1 Go line reaches
the running application in about 1 second, and the screen and the state hold.
`docs/glutter.md` holds each command and each limit, and `flutter/README.md`
holds the parts of the model.

## State

The compiler holds all 6 stages. The subset covers:

| Group | Content |
|---|---|
| Values | Each integer width, `float32`, `float64`, `bool`, `string` |
| Containers | Slice, array, map, struct with value semantics |
| Pointers | A pointer to any value, also to a field and to an element |
| Types | Method, interface, type assertion, type switch, embedded struct |
| Control | `if`, `for`, `switch`, `fallthrough`, label, `goto` |
| Functions | Closure, variadic, multiple results, named result, `defer` |
| Errors | `panic` and `recover`. Read decision 201. |
| Concurrency | Goroutine, channel, `select` |
| Generics | Generic function, generic type, generic method of Go 1.27, a generic function as a value |
| Isolates | `dart.Worker` runs a pure function in a second isolate |
| Structs | A named struct and an unnamed struct, with value semantics |
| Library | 254 import paths. `docs/what-compiles.md` holds the whole list. |
| Tests | `gart test` runs a `TestXxx` function and a subtest |
| Iterators | A range loop over a function, and `iter.Pull` over a goroutine |
| Files | `os` and `io` read and write a file through `dart:io` |
| Network | `net.Dial` and `net.Listen` reach TCP through `dart:io` |
| HTTP | `net/http` gives an HTTP/1.1 client with a pool, and a server. `net/http/httptest` tests a handler. |
| TLS | `crypto/tls` gives a client over the secure socket of Dart |
| Compression | `compress/gzip`, `compress/flate`, and `compress/zlib` write the bytes of Go |
| Stack | A source map gives `runtime.Caller` and the Go trace of a panic |
| JSON | `json.Marshal` walks the type descriptor of a value |
| Benchmarks | `gart test -bench <expression>` runs a `BenchmarkXxx` function |

The standard library surface holds 254 import paths, for example `fmt`,
`net/http`, `regexp`, `reflect`, `encoding/json`, `text/template`, `embed`,
`database/sql`, `go/types`, and the interop package `syscall/dart`.
`make surface` measures 174 of them against the real API of Go, and 165 hold
each exported name. 16 packages of the standard library stay outside, and
`make surface ARGS=-absent` names each 1. An overlay replaces the Go source of a
package that needs one, and the build tag `dart` selects the overlay files.
Each other package of the surface reads the real Go source.

`docs/what-compiles.md` names the shape of a Go program that the compiler
accepts. `docs/parity.md` lists each difference between Go and the Dart output.
The section "Documents" below names each other file.

## The 6 stages

| Stage | Package | Work |
|---|---|---|
| 1 Load | `internal/load` | Read the module with `go/packages` and the `dart` tag |
| 2 Verify | `internal/verify` | Reject each construct outside the subset |
| 3 Analyze | `internal/analyze` | Find each function that can block |
| 4 Normalize | `internal/normalize` | Build the intermediate tree of `internal/ir` |
| 5 Emit | `internal/emit` | Write the Dart source code |
| 6 Metadata | `internal/meta` | Write the archive with the 2 hashes |

`internal/overlay` holds the Go source of each standard library package of the
curated surface.

`internal/driver` runs the stages over a module and writes the Dart package.
`runtime/goruntime` holds the Dart runtime library.

## Documents

| File | Content |
|---|---|
| `docs/gart.md` | The compiler command, and each flag of it |
| `docs/glutter.md` | The Flutter command, and the first application |
| `docs/what-compiles.md` | The shape of a Go program that the compiler accepts |
| `docs/parity.md` | Each difference between Go and the Dart output |
| `docs/design.md` | The architecture, each stage, and each compiler feature |
| `docs/block-model.md` | The unwind model of a function that can block |
| `docs/decisions.md` | Each decision, by number, and each answered question |
| `docs/library.md` | The standard library surface and each overlay |
| `docs/history.md` | Each milestone, each defect, and each verified fact |
| `docs/flutter-design.md` | Each approach that reaches Flutter from Go, the decision, and the state |

## Tests

```bash
make ci
```

`internal/driver` holds the end to end test. It compiles each Go program of
`testdata/programs`, runs the Dart output, and compares the output with the
output of `go run`. The suite also proves the cache, the early cutoff, and the
byte equal output of 2 equal builds.

## Versions

The project targets Go 1.27.0 and Dart 3.13.0. A Flutter application needs
Flutter 3.47, `material_ui` 1.1, and `cupertino_ui` 1.0. `glutter doctor`
reports the version of the machine against the node set.

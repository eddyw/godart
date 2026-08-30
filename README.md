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
gart run -o dartout -runtime path/to/runtime/goruntime ./...
gart test -o dartout -runtime path/to/runtime/goruntime ./...
```

`gart build` writes the Dart package and stops. `gart run` compiles it and runs
package main with the Dart VM. `gart test` compiles the `_test.go` files and
runs each test binary with the Dart VM. The output of `gart test` holds the
format of `go test`. `gart test -bench <text>` runs the benchmarks on an ahead
of time build.

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
| Errors | `panic`. The subset has no `recover`. |
| Concurrency | Goroutine, channel, `select` |
| Generics | Generic function, generic type, generic method of Go 1.27, a generic function as a value |
| Isolates | `dart.Worker` runs a pure function in a second isolate |
| Structs | A named struct and an unnamed struct, with value semantics |
| Library | 66 packages. `docs/what-compiles.md` holds the whole list. |
| Tests | `gart test` runs a `TestXxx` function and a subtest |
| Iterators | A range loop over a function, and `iter.Pull` over a goroutine |
| Files | `os` and `io` read and write a file through `dart:io` |
| Network | `net.Dial` and `net.Listen` reach TCP through `dart:io` |
| HTTP | `net/http` gives an HTTP/1.1 client with a pool, and a server. `net/http/httptest` tests a handler. |
| TLS | `crypto/tls` gives a client over the secure socket of Dart |
| Compression | `compress/gzip`, `compress/flate`, and `compress/zlib` write the bytes of Go |
| Stack | A source map gives `runtime.Caller` and the Go trace of a panic |
| JSON | `json.Marshal` walks the type descriptor of a value |
| Benchmarks | `gart test -bench <text>` runs a `BenchmarkXxx` function |

The standard library surface holds 66 packages, for example `fmt`, `net/http`,
`regexp`, `reflect`, `encoding/json`, and the interop package `syscall/dart`. An
overlay replaces the Go source of a package that needs one, and the build tag
`dart` selects the overlay files. Each other package of the surface reads the
real Go source.

`docs/what-compiles.md` names the shape of a Go program that the compiler
accepts. `docs/parity.md` lists each difference between Go and the Dart output.

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

# godart

godart compiles Go source code into Dart source code. The command is `gart`.

The compiler supports a defined subset of the Go language. A verifier enforces
the subset at compile time with a clear error and a source position. Code inside
the subset gives the same result in Go and in Dart. A compromise is always a
rejection, never a silent change of meaning.

The output is a set of Dart packages. The output compiles with the Dart
toolchain, on the VM and with AOT. The web target is out of scope, because a
Dart `int` on the web has no 64 bits.

## State

The project is at milestone M0. Stage 1 of the compiler and the class `GoString`
exist.

## Layout

| Path | Content |
|---|---|
| `cmd/gart` | The command. |
| `internal/load` | Stage 1. It reads a Go module with `go/packages`. |
| `runtime/goruntime` | The Dart runtime library. |

## Start

```bash
make get        # download the Go and the Dart dependencies
make ci         # format, vet, and the 2 test suites
make build      # write build/gart
```

## Versions

The project targets Go 1.27.0 and Dart 3.13.0.

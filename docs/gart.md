# gart

This page is the start of the project. It names what `gart` is, how to build it,
and what each of the 6 commands does. Each command below comes from a real run
of 2026-09-05. `README.md` holds the state of the project.

## What gart is

gart is a compiler. It reads a Go module, and it writes a Dart package. The
output runs on the Dart VM, on an ahead of time binary, and on the web. A
Flutter application holds the same output, thus a Go program gives a user
interface. Read `docs/glutter.md` for that path.

gart accepts a defined subset of Go 1.27. Stage 2 rejects each construct outside
the subset, and the error carries a source position. Code inside the subset
gives the same result in Go and in Dart. A compromise is always a rejection, and
never a silent change of meaning.

gart is not the go command with a second back end. It writes no Go binary and it
holds no Go runtime. The Dart tool chain owns the machine code, the garbage
collector, and each platform.

## Build gart and install it

The project needs Go 1.27.0 and Dart 3.13.3. Start at the root of the
repository:

```sh
make get       # download the Go and the Dart dependencies
make build     # write build/gart and build/glutter
make install   # put the 2 commands on the PATH
```

`make build` runs `go build -o build/gart ./cmd/gart`. `make install` runs
`go install ./cmd/gart ./cmd/glutter`, thus the 2 commands reach the bin
directory of the Go path. `go env GOPATH` names that directory.

Make sure that the install worked:

```sh
gart version
```

```
gart 0.1.0-dev
```

## The 6 commands

| Command | What it does |
|---|---|
| `build` | Compile the packages into a Dart package. |
| `run` | Compile the packages, and run package main with the Dart VM. |
| `test` | Compile the tests, and run them with the Dart VM. |
| `watch` | Compile the packages again after each change. |
| `load` | Load the packages, and list them. |
| `version` | Print the version of the compiler. |

`gart` with no command writes this list. `gart <command> -h` writes the flags of
1 command.

The commands `build`, `run`, `test`, and `watch` share these flags:

| Flag | Effect |
|---|---|
| `-o dir` | Write the Dart package into `dir`. The default is `dartout`. |
| `-runtime dir` | Override the bundled `goruntime` Dart package. |
| `-format=false` | Do not run `dart format` on the output. |
| `-trimpath` | Use portable module paths in source maps. |
| `-timings` | Report compiler phases and execution durations on stderr. |

The bundled runtime is the default. [Compilation profiles](compilation-time.md) explain phase durations and current performance costs.

Each example below runs in this repository. Give the root of the repository 1
name first:

```sh
export GODART=$PWD
```

### build

`gart build` writes the Dart package and it stops. The command writes no line
when it succeeds.

```sh
cd $GODART/testdata/programs/hello
gart build -o /tmp/hello -runtime $GODART/runtime/goruntime ./...
```

The output directory then holds 5 files:

```
/tmp/hello/.gart/example.com%2fhello.json
/tmp/hello/bin/main.dart
/tmp/hello/bin/main.map.dart
/tmp/hello/bin/main/main.part.dart
/tmp/hello/pubspec.yaml
```

### run

`gart run` compiles the module and resolves Dart dependencies when required.
It compiles `bin/main.dart` to a cached kernel and starts that kernel on the Dart VM.
Each argument after `--` reaches the program, and not the compiler.

```sh
cd $GODART/testdata/programs/args
gart run -o /tmp/args -runtime $GODART/runtime/goruntime ./... -- alpha beta
```

```
count 2
text alpha,beta
```

### test

`gart test` compiles each `_test.go` file, and it runs 1 test binary for each
package. The output holds the format of `go test`.

```sh
cd $GODART/testdata/tests/basic
gart test -o /tmp/basic -runtime $GODART/runtime/goruntime -v ./...
```

The flag `-v` gives 1 line for each test. The last 4 lines of the output are:

```
=== RUN   TestJoin
--- PASS: TestJoin (0.00s)
PASS
ok  	example.com/basic/calc	4.849s
```

The time of the last line differs at each run. The test command takes 8 flags of
its own:

| Flag | Effect |
|---|---|
| `-v` | Write 1 line for each test. |
| `-run expr` | Run only a test whose name matches the expression. |
| `-short` | Ask each test for its short form. |
| `-count n` | Run the whole list of the tests n times, and 0 runs none. |
| `-cover` | Report the part of the statements that the tests reached. |
| `-bench expr` | Run each benchmark whose name matches the expression. |
| `-benchtime d` | The length that 1 benchmark targets, for example `2s`. |
| `-benchcount n` | The fixed count of iterations of 1 benchmark. |

A filter and a coverage report read the same command:

```sh
gart test -o /tmp/basic -runtime $GODART/runtime/goruntime -run TestAdd -cover ./...
```

```
ok  	example.com/basic/calc	5.727s	coverage: 12.5% of statements
```

A benchmark run compiles an ahead of time binary first, because the warm up cost
of the Dart VM hides the result. The number measures the Dart output, and not
Go:

```sh
gart test -o /tmp/basic -runtime $GODART/runtime/goruntime -bench . -benchtime 200ms ./...
```

```
goos: dart
goarch: arm64
pkg: example.com/basic/calc
BenchmarkAdd-1	1000000000	0.3534 ns/op
BenchmarkDouble/one-1	1000000000	0.3448 ns/op
BenchmarkDouble/big-1	1000000000	0.3603 ns/op
BenchmarkLoop-1	500000000	0.6531 ns/op
PASS
ok  	example.com/basic/calc	7.498s
```

### watch

`gart watch` holds each Go file of the module under watch. It builds 1 time at
the start, and it builds again 120 milliseconds after the last change. Copy 1
program of `testdata` first, thus a change touches no file of git:

```sh
cp -r $GODART/testdata/programs/multi /tmp/multi-src
cd /tmp/multi-src
gart watch -o /tmp/multi -runtime $GODART/runtime/goruntime ./...
```

```
gart: 3 compiled, 0 cached
  example.com/multi/count
  example.com/multi/greet
  example.com/multi
gart: watch /tmp/multi-src
```

A change of 1 line of `greet/greet.go` gives the next 2 lines:

```
gart: 1 compiled, 2 cached
  example.com/multi/greet
```

The package `example.com/multi` imports `greet`, and it still needs no second
compile. Read "The cache and the watch mode" below.

### load

`gart load` reads the module with the build tag `dart`, and it lists each
package with the count of its Go files. It writes no Dart, thus it is the
cheapest way to see what a pattern matches.

```sh
cd $GODART/testdata/programs/multi
gart load ./...
```

```
example.com/multi	1 files
example.com/multi/count	1 files
example.com/multi/greet	1 files
```

The command takes no flag. A call with no pattern reads the package `.` alone.

### version

`gart version` prints the version of the tool chain. A compiled program reads
the same text through `runtime.Version`.

```sh
gart version
```

```
gart 0.1.0-dev
```

## What compiles

The subset covers the whole language of a normal program:

| Group | Content |
|---|---|
| Values | Each integer width, `float32`, `float64`, `bool`, `string`, and a complex number |
| Containers | A slice, an array, a map, and a struct with value semantics |
| Pointers | A pointer to a value, to a field, and to an element |
| Types | A method, an interface, a type assertion, a type switch, and an embedded struct |
| Control | `if`, `for`, `switch`, `fallthrough`, a label, and `goto` |
| Functions | A closure, a variadic call, several results, a named result, and `defer` |
| Errors | `panic` and `recover` |
| Concurrency | A goroutine, a channel, and a `select` |
| Generics | A generic function, a generic type, and the generic method of Go 1.27 |
| Iterators | A range loop over a function, and `iter.Pull` over a goroutine |

The compiler holds 5 non-goals. Each 1 is a decision of the project, and no
future work removes it.

| Non-goal | What it means |
|---|---|
| No `unsafe` | Stage 2 rejects the import with a source position. |
| No cgo | The output holds no C. `runtime/cgo` and `plugin` stand outside the surface. |
| No pointer arithmetic | A pointer names a cell. The address of a value behind a pointer, for example `&*p`, is a rejection. |
| No goroutine on a second thread | All Go code runs in 1 Dart isolate, and `GOMAXPROCS` gives 1. |
| No full `reflect` | `reflect` holds 183 of 188 names. Each of the 5 gaps needs an address or an `unsafe.Pointer`. |

An import of `unsafe` gives this error:

```
gart: /tmp/unsafe-src/main.go:3:8: the compiler rejects the package unsafe (the subset has no pointer arithmetic)
```

The address of a field and the address of an element each stay inside the
subset. `&s.F` and `&a[1]` each build and each run.

`docs/what-compiles.md` names the shape of a program that builds, each gate of
the language, and each target. `docs/parity.md` names each behavior that differs
between Go and the Dart output. Read the 2 pages before a large port.

## The standard library

The surface holds 254 import paths. The set covers 164 standard packages, the
interop package `syscall/dart`, 77 internal packages, and 12 vendored packages
of `golang.org/x`. Stage 2 rejects an import of each other package.

`make surface` measures the surface against the real API of Go 1.27. The run of
2026-09-07 gives this result:

```
165 packages of 174 hold each exported name, and 9 hold a part.
```

The 9 partial packages are `crypto/tls`, `net`, `net/http`, `os`, `os/exec`,
`reflect`, `runtime`, `sync/atomic`, and `syscall`. 16 packages of the standard
library stand outside the surface, and `make surface ARGS=-absent` names each 1.

`docs/support.md` holds the level of each package, the largest gap of each
partial package, and the cause of each absent package. `docs/library.md` holds
the overlay work behind those numbers.

## The output of a build

A build writes 1 Dart package for the whole Go module. The name of the package
comes from the last element of the module path, in lower case. `example.com/multi`
thus gives the Dart package `multi`.

```sh
cd $GODART/testdata/programs/multi
gart build -o /tmp/multi -runtime $GODART/runtime/goruntime ./...
```

```
/tmp/multi/.gart/example.com%2fmulti.json
/tmp/multi/.gart/example.com%2fmulti%2fcount.json
/tmp/multi/.gart/example.com%2fmulti%2fgreet.json
/tmp/multi/bin/main.dart
/tmp/multi/bin/main.map.dart
/tmp/multi/bin/main/main.part.dart
/tmp/multi/lib/count.dart
/tmp/multi/lib/count.map.dart
/tmp/multi/lib/count/count.part.dart
/tmp/multi/lib/greet.dart
/tmp/multi/lib/greet.map.dart
/tmp/multi/lib/greet/greet.part.dart
/tmp/multi/pubspec.yaml
```

| Path | Content |
|---|---|
| `lib/<pkg>.dart` | The Dart library of 1 Go package. The path mirrors the import path. |
| `lib/<pkg>/<file>.part.dart` | 1 Dart part file for each Go file of the package. |
| `bin/main.dart` | The entry point of package main. |
| `bin/<pkg>_test.dart` | The entry point of 1 test binary. |
| `<name>.map.dart` | The source map of 1 library or of 1 entry point. |
| `pubspec.yaml` | The Dart package, with `goruntime` as a path dependency. |
| `.gart/` | Package archives, overlay metadata, and compiled kernels. |

The driver writes the pubspec, thus the reader needs no hand edit. The `path`
line holds the absolute form of the `-runtime` flag:

```yaml
name: multi
description: The Dart output of the Go module example.com/multi.
publish_to: none

environment:
  sdk: ^3.13.0

dependencies:
  goruntime:
    path: /path/to/godart/runtime/goruntime
```

`dart pub get` and `dart run bin/main.dart` can also run the emitted source.
`gart run` resolves dependencies and uses the cached kernel automatically.

The source map is a `part` file of the library that it belongs to. It holds the
Go file and the Go line of each Dart line, thus `runtime.Caller` and a panic
each name the Go source. A program with a panic in `inner` gives the trace of
Go:

```go
func inner() {
	panic("boom")
}

func main() {
	inner()
}
```

```
panic: boom

goroutine 1 [running]:
main.inner()
	example.com/boom/main.go:4
main.main()
	example.com/boom/main.go:8
```

## The cache and the watch mode

The compiler treats each package as a pure function. The inputs are the source
files, the metadata of each dependency, the version of the compiler, and the
flags. Equal inputs give byte equal output, thus a second build of a package
with no change is a waste. The cache in `.gart/` holds 2 hashes of each package
and it removes that waste.

| Hash | What it decides |
|---|---|
| The source hash | The package needs a second compile, or the cache answers it. |
| The export hash | Each package that imports it needs a second compile, or it does not. |

A build thus compiles the packages that changed alone. The watch example above
holds the whole rule. A change of `greet` compiled 1 package of 3, and the
export hash of `greet` held. The package `example.com/multi` imports `greet`,
and it read the cache.

The driver writes a Dart file only when the content changed. An equal file keeps
its time stamp, thus the incremental compiler of Dart keeps its own cache.

`gart run` and ordinary `gart test` executions reuse source kernels under `.gart/kernel/`.
Each receipt records the Dart SDK identity, compiler options, resolved package configuration, compiler dependency contents, and kernel hash.
A source, runtime, dependency, SDK, or artifact change causes recompilation.
The cache compares file contents, thus a preserved timestamp does not hide an edit.
A benchmark still uses the existing AOT path.

Kernel compilation preserves non-product constants and assertion support.
The cached file also supports `dart run --enable-asserts` and `dart run --observe`.
Generated Dart source remains available for direct execution and debugger source positions.

`.gart/load/surface.json` holds the native file metadata for standard library overlays.
Gart validates the Go environment and native source contents before reuse.
It also retains 1 validated metadata snapshot within a process.
Application discovery remains fresh, thus new imports and files remain visible.
External Go overlays and custom package drivers bypass this metadata cache.

To remove only these new caches, remove `.gart/kernel/` and `.gart/load/` beneath the selected output directory.
Use separate output directories for concurrent builds.

`gart watch` is a thin layer over these rules. It watches each directory of the
module, and it skips the output directory. A change of a `.go` file or of a
`.mod` file starts a timer of 120 milliseconds, and the build runs after the
last write. An editor that writes 3 files at one time thus gives 1 build.

A compile error stops the build and it writes no count. The watch continues, and
the next change starts a new build. Read `docs/design.md`, section "Incremental
compilation and watch mode", for the rules behind this behavior.

## What to read next

| Page | Content |
|---|---|
| `docs/glutter.md` | The `glutter` command, and a Flutter application whose widget tree comes from Go. |
| `docs/design.md` | The architecture, and the 6 stages of the compiler. |
| `docs/support.md` | The level of each package of the standard library surface. |
| `docs/what-compiles.md` | The shape of a Go program that the compiler accepts. |
| `docs/parity.md` | Each difference between Go and the Dart output. |
| `flutter/README.md` | The parts of the Flutter work, and each example program. |

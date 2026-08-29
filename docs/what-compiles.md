# What compiles

This file names the shape of a Go program that the compiler accepts. It also
names the gate that stops each program that it rejects. `docs/parity.md` holds
each difference between Go and the Dart output.

The list comes from a measurement of 2026-08-30. Each `OK` row below is a real
build of a program that imports the package.

## The import surface

46 packages of the standard library reach the surface. Stage 2 rejects an import
of each other package with a source position.

| Present | Absent |
|---|---|
| `bufio`, `bytes`, `cmp` | `regexp` |
| `container/heap`, `container/list`, `container/ring` | `net/http`, `net/url` |
| `context`, `errors`, `flag`, `fmt`, `log` | `os/exec`, `os/signal` |
| `encoding/ascii85`, `encoding/base64`, `encoding/csv` | `encoding/binary`, `encoding/xml`, `encoding/gob` |
| `encoding/hex`, `encoding/json` | `crypto/...`, `compress/...`, `archive/...` |
| `hash`, `hash/adler32`, `hash/crc32`, `hash/fnv` | `text/template`, `html/template`, `text/tabwriter` |
| `io`, `iter`, `maps`, `slices`, `sort` | `io/fs`, `embed`, `database/sql` |
| `math`, `math/bits`, `math/rand`, `math/rand/v2` | `math/cmplx`, `go/...` |
| `net`, `os`, `path`, `path/filepath` | `unsafe`, cgo |
| `reflect`, `runtime`, `strconv`, `strings` | |
| `sync`, `sync/atomic`, `testing`, `time` | |
| `unicode`, `unicode/utf8`, `syscall/dart` | |

## The gates of the language

| Gate | Effect |
|---|---|
| `recover` | Rejected. A panic stops the program. |
| `unsafe` and cgo | Rejected |
| A complex number | Rejected |
| A pointer to a field of a type that is not a struct | Rejected |
| The address of a composite literal of a type that is not a struct | Rejected. A variable first, and the address of the variable. |
| A goroutine on a second thread | All Go code runs in 1 Dart isolate. `GOMAXPROCS` gives 1. |
| A conversion between 2 struct types | Rejected |

Each other construct of the language stays inside the subset: a closure, a
method, an interface, an embedded struct, a type switch, a channel, a `select`,
a `defer` that can block, `goto`, generics, a range loop over a function, and
the loop variable of Go 1.22.

## The shapes that build

Each program below is a real build of 2026-08-30. The Dart output gives the
output of `go run` for each one.

| Shape | Reads |
|---|---|
| A tokenizer, a Pratt parser, and an evaluator | `fmt`, `strconv`, `strings`, `hash/fnv` |
| An LRU cache | `container/list`, a map, a type assertion |
| The shortest path of Dijkstra | `container/heap`, `math`, a slice of a slice |
| A command line report tool | `flag`, `log`, `encoding/csv`, `path/filepath`, `sort` |
| The game of life of Conway | A method on a pointer, `strings.Builder` |
| A TCP chat server with a client | `net`, `bufio`, `sync`, a goroutine |
| A word frequency count | `os.ReadFile`, `strings.FieldsFunc`, `sort.Slice` |
| A JSON reader that the program writes itself | `errors`, `strconv`, `strings`, a type switch |
| A worker pool with a context | `context`, `sync`, a channel, a `select` |
| A generic set with an iterator | `cmp`, `maps`, `slices`, a generic method |

Thus these categories build:

- A command line tool that reads a file, writes a report, and takes a flag.
- An interpreter, a parser, a tokenizer, and a virtual machine.
- An algorithm collection: a graph, a tree, a sort, a search, dynamic
  programming, and a number theory problem.
- A solution of Advent of Code, of Project Euler, or of Exercism, when the
  solution reads no regular expression.
- A TCP server and a TCP client with a text protocol of its own.
- A simulation with a goroutine, a channel, and a `select`.
- A test suite of such a program. `gart test` gives the output of `go test`.

## The gate of each program that does not build

| Gate | What it stops |
|---|---|
| `net/http` | Each web server, each REST client, and each program that reads a URL. |
| `regexp` | Each program that matches a pattern. A `strings` search is the alternative. |
| `crypto/...` | Each program that hashes a password, signs a value, or reads TLS. |
| `os/exec` | Each program that starts another process. |
| `recover` | Each program that turns a panic into an error. A parser of the standard library often does. |
| `text/template`, `html/template` | Each program that writes a page or a file from a template. |
| `encoding/xml`, `encoding/binary` | Each program that reads a binary format or an XML file. |
| `io/fs` | `filepath.Walk` and `filepath.WalkDir`. |

Thus no large program of the Go world builds today. `kubectl`, `hugo`,
`docker`, `terraform`, and `gopls` each read `net/http`, `os/exec`, `regexp`, or
`recover`. The gate is the library surface, and not the language.

## The near misses

Each package below needs a small piece of work. The survey of 2026-08-30 gives
the exact gap.

| Package | The gap |
|---|---|
| `net/url` | `internal/godebug` and `net/netip`. An overlay would drop both. |
| `text/tabwriter` | 1 call of `recover`, and the address of a field of a slice. |
| `encoding/binary` | `reflect.Value.Kind` and the kind constants of `reflect`. |
| `io/fs` | An overlay of `FileInfo`, `DirEntry`, and `WalkDirFunc`. It would give `filepath.Walk`. |
| `hash/maphash`, `crypto/md5`, `crypto/sha256` | Each one reads a package of `crypto/internal` or of `internal`. A pure Go overlay of the algorithm would work. |

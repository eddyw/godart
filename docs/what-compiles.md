# What compiles

This file names the shape of a Go program that the compiler accepts. It also
names the gate that stops each program that it rejects. `docs/parity.md` holds
each difference between Go and the Dart output.

Each row comes from a measurement of 2026-08-30. An `Present` row is a real
build of a program that imports the package.

## The import surface

54 packages of the standard library reach the surface. Stage 2 rejects an import
of each other package with a source position.

| Present | Absent |
|---|---|
| `bufio`, `bytes`, `cmp` | `regexp` |
| `container/heap`, `container/list`, `container/ring` | `net/http` |
| `context`, `errors`, `flag`, `fmt`, `log` | `os/exec`, `os/signal` |
| `encoding/ascii85`, `encoding/base64`, `encoding/binary` | `encoding/xml`, `encoding/gob` |
| `encoding/csv`, `encoding/hex`, `encoding/json` | `text/template`, `html/template` |
| `crypto/md5`, `crypto/sha256` | `crypto/tls`, `crypto/rand`, `compress/...` |
| `hash`, `hash/adler32`, `hash/crc32`, `hash/fnv` | `archive/...`, `text/tabwriter` |
| `io`, `io/fs`, `iter`, `maps`, `slices`, `sort` | `embed`, `database/sql` |
| `math`, `math/bits`, `math/rand`, `math/rand/v2` | `math/cmplx`, `go/...` |
| `net`, `net/netip`, `net/url` | `unsafe`, cgo |
| `os`, `path`, `path/filepath` | |
| `reflect`, `runtime`, `strconv`, `strings` | |
| `sync`, `sync/atomic`, `testing`, `time` | |
| `unicode`, `unicode/utf8`, `unique` | |
| `internal/godebug`, `syscall/dart` | |

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
| A nil pointer receiver of a method of a generic type | Panics. Each other nil pointer receiver runs. |

Each other construct of the language stays inside the subset: a closure, a
method, an interface, an embedded struct, a type switch, a channel, a `select`,
a `defer` that can block, `goto`, generics, a range loop over a function, the
loop variable of Go 1.22, a conversion of a slice to an array, and a method with
a nil pointer receiver.

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
| A word frequency count | `os`, `strings.FieldsFunc`, `sort.Slice` |
| A word count of a file | `bufio.Scanner`, `os`, `path/filepath` |
| A JSON reader that the program writes itself | `errors`, `strconv`, `strings`, a type switch |
| A walk of a directory tree | `path/filepath.WalkDir`, `io/fs`, `os.ReadDir` |
| A reader of a URL and of a query | `net/url`, `net/netip` |
| A worker pool with a context | `context`, `sync`, a channel, a `select` |
| A generic set with an iterator | `cmp`, `maps`, `slices`, a generic method |
| A linked list with a nil safe method | A method with a nil pointer receiver |

Thus these categories build:

- A command line tool that reads a file, walks a directory, writes a report,
  and takes a flag.
- An interpreter, a parser, a tokenizer, and a virtual machine.
- An algorithm collection: a graph, a tree, a sort, a search, dynamic
  programming, and a number theory problem.
- A solution of Advent of Code, of Project Euler, or of Exercism, when the
  solution reads no regular expression.
- A TCP server and a TCP client with a text protocol of its own.
- A simulation with a goroutine, a channel, and a `select`.
- A reader and a writer of JSON, CSV, hex, and base64.
- A test suite of such a program. `gart test` gives the output of `go test`.

## The gate of each program that does not build

| Gate | What it stops |
|---|---|
| `net/http` | Each web server, each REST client, and each program that reads a page. |
| `regexp` | Each program that matches a pattern. A `strings` search is the alternative. |
| `crypto/...` | Each program that hashes a password, signs a value, or reads TLS. |
| `os/exec` | Each program that starts another process. |
| `recover` | Each program that turns a panic into an error. A parser of the standard library often does. |
| `text/template`, `html/template` | Each program that writes a page or a file from a template. |
| `encoding/xml`, `encoding/binary` | Each program that reads a binary format or an XML file. |

Thus no large program of the Go world builds today. `kubectl`, `hugo`,
`docker`, `terraform`, and `gopls` each read `net/http`, `os/exec`, `regexp`, or
`recover`. The gate is the library surface, and not the language.

## The near misses

Each package below needs a small piece of work. The survey of 2026-08-30 gives
the exact gap.

| Package | The gap |
|---|---|
| `text/tabwriter` | 1 call of `recover`, and the address of a field of a slice. |
| `encoding/binary` | `reflect.Value.Kind` and the kind constants of `reflect`. |
| `crypto/md5`, `crypto/sha256` | Each one reads a package of `crypto/internal`. A pure Go overlay of the algorithm would work. |
| `net/http` | The largest gap. It reads TLS, `crypto`, `regexp` for a cookie, and a server of its own. |

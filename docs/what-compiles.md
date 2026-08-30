# What compiles

This file names the shape of a Go program that the compiler accepts. It also
names the gate that stops each program that it rejects. `docs/parity.md` holds
each difference between Go and the Dart output.

Each row comes from a measurement of 2026-08-30. An `Present` row is a real
build of a program that imports the package.

## The import surface

62 packages of the standard library reach the surface. Stage 2 rejects an import
of each other package with a source position.

| Present | Absent |
|---|---|
| `bufio`, `bytes`, `cmp`, `net/http` | `net/http/httptest` |
| `container/heap`, `container/list`, `container/ring` | `os/exec`, `os/signal` |
| `context`, `errors`, `flag`, `fmt`, `log` | `encoding/xml`, `encoding/gob` |
| `regexp`, `regexp/syntax` | `text/template`, `html/template` |
| `encoding/ascii85`, `encoding/base64`, `encoding/binary` | `crypto/rand`, `crypto/x509` |
| `encoding/csv`, `encoding/hex`, `encoding/json` | `compress/...`, `archive/...` |
| `crypto/md5`, `crypto/sha1`, `crypto/sha256`, `crypto/tls` | `text/tabwriter`, `embed` |
| `hash`, `hash/adler32`, `hash/crc32`, `hash/fnv` | `database/sql`, `math/cmplx` |
| `io`, `io/fs`, `iter`, `maps`, `slices`, `sort` | `go/...` |
| `math`, `math/bits`, `math/rand`, `math/rand/v2` | `unsafe`, cgo |
| `net`, `net/netip`, `net/url` | |
| `os`, `path`, `path/filepath` | |
| `reflect`, `runtime`, `strconv`, `strings` | |
| `sync`, `sync/atomic`, `testing`, `testing/quick` | |
| `time`, `unique` | |
| `unicode`, `unicode/utf8` | |
| `encoding`, `internal/godebug`, `internal/byteorder` | |
| `internal/testhash` | |
| `testing/internal/testdeps`, `syscall/dart` | |

## The gates of the language

| Gate | Effect |
|---|---|
| `recover` | Rejected. A panic stops the program. |
| `unsafe` and cgo | Rejected |
| A complex number | Rejected |
| The address of a value behind a pointer, for example `&*p` | Rejected. Read `p`. |
| A goroutine on a second thread | All Go code runs in 1 Dart isolate. `GOMAXPROCS` gives 1. |
| A conversion between 2 struct types | Rejected |
| A generic function as a value | Rejected. Such a value would need the dictionary of each type argument. |

Each other construct of the language stays inside the subset: a closure, a
method, an interface, an embedded struct, a type switch, a channel, a `select`,
a `defer` that can block, `goto`, generics, a range loop over a function, the
loop variable of Go 1.22, a conversion of a slice to an array, a method with a
nil pointer receiver, a method expression, and a type declaration inside a
function.

## The shapes that build

Each program below is a real build of 2026-08-30. The Dart output gives the
output of `go run` for each one. The last 5 rows stand in `testdata/programs`,
and `make ci` runs each one.

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
| An HTTP server and a client in 1 program | `net/http`, a handler, a cookie, a chunked answer |
| A matcher of a pattern with a named group | `regexp`, a replace, a split |
| A checksum of a text with 4 algorithms | `crypto/md5`, `crypto/sha1`, `crypto/sha256`, `encoding/hex` |
| A read and a write of a binary record | `encoding/binary`, a struct, an array, a varint |
| A property test of a pure function | `testing/quick`, `reflect.Value.Call` |

Thus these categories build:

- A command line tool that reads a file, walks a directory, writes a report,
  and takes a flag.
- An interpreter, a parser, a tokenizer, and a virtual machine.
- An algorithm collection: a graph, a tree, a sort, a search, dynamic
  programming, and a number theory problem.
- A solution of Advent of Code, of Project Euler, or of Exercism.
- A TCP server and a TCP client with a text protocol of its own.
- A simulation with a goroutine, a channel, and a `select`.
- A reader and a writer of JSON, CSV, hex, base64, and a binary record.
- An HTTP server of HTTP/1.1, and a client of it. A client of an https address
  also runs, because Dart holds the handshake.
- A test suite of such a program. `gart test` gives the output of `go test`.

## The gate of each program that does not build

| Gate | What it stops |
|---|---|
| `os/exec` | Each program that starts another process. |
| `recover` | Each program that turns a panic into an error. A parser of the standard library often does. |
| `text/template`, `html/template` | Each program that writes a page or a file from a template. |
| `encoding/xml`, `encoding/gob` | Each program that reads an XML file or a gob stream. |
| `compress/...`, `archive/...` | Each program that reads a zip file, a tar file, or a gzip stream. |
| HTTP/2 and keep alive | Each client that needs a pool of connections or the newer protocol. |
| `net/http/httptest` | Each test of a handler that reads a recorder. |
| `database/sql` | Each program that reads a database. |
| A goroutine on a second thread | Each program that needs a parallel run. All Go code runs in 1 isolate. |

Thus no large program of the Go world builds today. `kubectl`, `hugo`,
`docker`, `terraform`, and `gopls` each read `os/exec`, `recover`, a template
package, or a compression package. The gate is the library surface, and not the
language.

## The near misses

Each package below needs a small piece of work. The survey of 2026-08-30 gives
the exact gap.

| Package | The gap |
|---|---|
| `text/tabwriter` | 1 call of `recover`, and the address of a field of a slice. |
| `compress/flate`, `compress/gzip` | A survey of the real source. `net/http` would then hold `Accept-Encoding: gzip`. |
| `net/http/httptest` | A listener of a test and a recorder of an answer, over the `net/http` overlay. |
| `crypto/hmac`, `crypto/sha512` | Each one follows the pattern of the `crypto/sha256` overlay. |

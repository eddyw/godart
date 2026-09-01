# What compiles

This file names the shape of a Go program that the compiler accepts. It also
names the gate that stops each program that it rejects. `docs/parity.md` holds
each difference between Go and the Dart output. `docs/support.md` holds the
count of the exported names that each package of the surface holds.

Each row comes from a measurement of 2026-08-31. A `Present` row is a real build
of a program that imports the package. The import surface holds the run of
`make surface` of 2026-09-02.

## The import surface

The surface holds 113 import paths: 99 standard packages, the interop package
`syscall/dart`, and 13 internal packages that the 99 need. Stage 2 rejects an
import of each other package with a source position.

86 of the 99 hold each exported name of Go, and 14 hold a part of them.
`docs/support.md` holds the level of each one and the largest gap of each
partial package. `make surface` measures each one again.

| Present | Absent |
|---|---|
| `bufio`, `bytes`, `cmp`, `net/http` | `net/http/cgi`, `net/rpc` |
| `container/heap`, `container/list`, `container/ring`, `os/exec` | `net/http/cgi` |
| `context`, `errors`, `flag`, `fmt`, `log` | `encoding/xml`, `encoding/gob` |
| `regexp`, `regexp/syntax` | `html/template` |
| `encoding/ascii85`, `encoding/base64`, `encoding/binary` | `crypto/x509`, `crypto/subtle` |
| `encoding/csv`, `encoding/hex`, `encoding/json` | `archive/...`, `compress/lzw` |
| `compress/flate`, `compress/gzip`, `compress/zlib` | `compress/bzip2` |
| `go/token`, `go/scanner`, `go/ast`, `go/parser` | `go/types`, `go/importer` |
| `go/printer`, `go/format`, `go/build/constraint`, `go/doc/comment` | `go/build` |
| `text/tabwriter`, `text/template`, `text/template/parse` | |
| `testing/iotest`, `runtime/debug`, `os/signal`, `syscall` | |
| `net/http/httptest`, `net/http/internal` | `net/smtp`, `net/mail` |
| `crypto/md5`, `crypto/sha1`, `crypto/sha256`, `crypto/sha512` | `embed` |
| `crypto/hmac`, `crypto/rand`, `crypto/tls` | `math/big` |
| `mime`, `mime/multipart`, `mime/quotedprintable` | |
| `net/textproto`, `net/http/cookiejar` | |
| `hash`, `hash/adler32`, `hash/crc32`, `hash/fnv` | `database/sql`, `math/cmplx` |
| `io`, `io/fs`, `iter`, `maps`, `slices`, `sort` | |
| `math`, `math/bits`, `math/rand`, `math/rand/v2` | `unsafe`, cgo |
| `net`, `net/netip`, `net/url` | |
| `os`, `path`, `path/filepath` | |
| `reflect`, `runtime`, `strconv`, `strings` | |
| `sync`, `sync/atomic`, `testing`, `testing/quick` | |
| `time`, `unique` | |
| `unicode`, `unicode/utf8` | |
| `encoding`, `internal/godebug`, `internal/byteorder` | |
| `internal/testhash`, `internal/testenv` | |
| `internal/chacha8rand`, `net/http/internal/ascii` | |
| `testing/internal/testdeps`, `syscall/dart` | |

## The gates of the language

| Gate | Effect |
|---|---|
| `unsafe` and cgo | Rejected |
| A complex number | Present. The runtime class GoComplex holds it. |
| The address of a value behind a pointer, for example `&*p` | Rejected. Read `p`. |
| A goroutine on a second thread | All Go code runs in 1 Dart isolate. `GOMAXPROCS` gives 1. |
| A conversion between 2 struct types | Rejected |
| A function that can block, in a value of a type that carries `//gart:sync` | Rejected. Read decision 221. |
| A function type that names itself, for example `type stateFn func(*lexer) stateFn` | Present. The type erases to `Object?`, and a call of such a value goes through a dynamic call. Read decision 227. |

Stage 2 accepts a local struct type that reads a type parameter, and the
`recover` builtin. Read decisions 200 and 201 of `docs/decisions.md`.

The last gate is the only 1 that a library declares. A named function type whose
declaration carries `//gart:sync` states that a host outside Go calls the value
and reads the result at once. A function that can block returns the zero value
there, thus the compiler rejects it. `gart.dev/ui` declares 2 such types, and no
package of the standard surface declares any.

Each other construct of the language stays inside the subset:

- A closure, a method, an interface, an embedded struct, and a type switch.
- A channel, a `select`, and a `defer` that can block.
- `goto`, and a type declaration inside a function.
- Generics, a generic function as a value, and a method expression.
- A range loop over a function, and the loop variable of Go 1.22.
- A conversion of a slice to an array, and a method with a nil pointer receiver.

## The shapes that build

Each program below is a real build of 2026-08-31. The Dart output gives the
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
| A merge of 2 sequences, step by step | `iter.Pull`, `iter.Pull2`, a goroutine |
| The size and the offset of each field of a struct | `reflect.Type.Size`, `Align`, `StructField.Offset` |
| A write of a byte that holds no rune of UTF-8 | `fmt`, `os.Stdout.Write` |
| A cleanup that runs after a collection | `runtime.AddCleanup`, `runtime.GC` |
| The text and the tag of each unnamed type | `reflect.Type.String`, `StructTag.Get` |
| A linked list with a nil safe method | A method with a nil pointer receiver |
| An HTTP server and a client in 1 program | `net/http`, a handler, a cookie, a chunked answer |
| A client that holds a connection open | `net/http` keep alive, a listener of its own |
| A round trip of a gzip stream | `compress/gzip`, `compress/flate`, `compress/zlib`, a dictionary, each level |
| A client that reads the gzip form of an answer | `net/http` with `Accept-Encoding: gzip` |
| A test of a handler with a recorder | `net/http/httptest`, `testing` |
| A matcher of a pattern with a named group | `regexp`, a replace, a split |
| A checksum of a text with 4 algorithms | `crypto/md5`, `crypto/sha1`, `crypto/sha256`, `encoding/hex` |
| A read and a write of a binary record | `encoding/binary`, a struct, an array, a varint |
| A property test of a pure function | `testing/quick`, `reflect.Value.Call` |

Thus these categories build:

- A command line tool that reads a file, walks a directory, writes a report, and
  takes a flag.
- An interpreter, a parser, a tokenizer, and a virtual machine.
- An algorithm collection: a graph, a tree, a sort, a search, dynamic
  programming, and a number theory problem.
- A solution of Advent of Code, of Project Euler, or of Exercism.
- A TCP server and a TCP client with a text protocol of its own.
- A simulation with a goroutine, a channel, and a `select`.
- A reader and a writer of JSON, CSV, hex, base64, and a binary record.
- An HTTP server of HTTP/1.1 or HTTP/2, and a client of it. A client of an
  https address also runs, because Dart holds the handshake.
- A test suite of such a program. `gart test` gives the output of `go test`.

## The gate of each program that does not build

| Gate | What it stops |
|---|---|
| `html/template` | Each program that writes an HTML page from a template. `text/template` builds since 2026-09-02. |
| `encoding/xml`, `encoding/gob` | Each program that reads an XML file or a gob stream. |
| `archive/...`, `compress/bzip2`, `compress/lzw` | Each program that reads a zip file or a tar file. |
| `database/sql` | Each program that reads a database. |
| A goroutine on a second thread | Each program that needs a parallel run. All Go code runs in 1 isolate. |

Thus no large program of the Go world builds today. `kubectl`, `hugo`,
`docker`, `terraform`, and `gopls` each read `html/template`, a compression
package, `encoding/xml`, or `database/sql`. The gate is the library surface,
and not the language.

## The near misses

Each package below needs a small piece of work. `docs/support.md` holds the
whole list of the packages that stay out, with the cause and the cost of each
one.

| Package | The gap |
|---|---|
| `html/template` | It reads `text/template`, which builds, and the escape pass of HTML. |
| `crypto/subtle` | Each function is a loop over 2 byte slices. |
| `math/cmplx` | The language holds the complex type already. The package needs a survey. |

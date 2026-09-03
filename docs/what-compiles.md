# What compiles

This file names the shape of a Go program that the compiler accepts. It also
names the gate that stops each program that it rejects. `docs/parity.md` holds
each difference between Go and the Dart output. `docs/support.md` holds the
count of the exported names that each package of the surface holds.

Each row comes from a measurement of 2026-08-31. A `Present` row is a real build
of a program that imports the package. The import surface holds the run of
`make surface` of 2026-09-03.

## The import surface

The surface holds 170 import paths: 140 standard packages, the interop package
`syscall/dart`, 23 internal packages, and 6 vendored packages of
`golang.org/x`. Stage 2 rejects an import of each other package with a source
position.

91 of the 103 hold each exported name of Go, and 12 hold a part of them.
`docs/support.md` holds the level of each one and the largest gap of each
partial package. `make surface` measures each one again.

| Present | Absent |
|---|---|
| `bufio`, `bytes`, `cmp`, `net/http` | `go/importer` |
| `container/heap`, `container/list`, `container/ring`, `os/exec` | |
| `context`, `errors`, `flag`, `fmt`, `log` | |
| `regexp`, `regexp/syntax`, `encoding/xml` | |
| `go/constant`, `go/types`, `go/build`, `go/doc` | |
| `net/mail`, `net/smtp`, `net/rpc`, `net/http/cgi` | |
| `encoding/ascii85`, `encoding/base64`, `encoding/binary`, `embed` | |
| `database/sql`, `database/sql/driver`, `uuid` | |
| `encoding/csv`, `encoding/hex`, `encoding/json`, `encoding/gob` | `unsafe`, cgo |
| `encoding/asn1`, `encoding/pem`, `unicode/utf16` | |
| `compress/flate`, `compress/gzip`, `compress/zlib` | |
| `compress/lzw`, `compress/bzip2`, `archive/zip`, `archive/tar` | |
| `html`, `html/template` | |
| `go/token`, `go/scanner`, `go/ast`, `go/parser` | |
| `go/printer`, `go/format`, `go/build/constraint`, `go/doc/comment` | |
| `text/tabwriter`, `text/template`, `text/template/parse` | |
| `testing/iotest`, `runtime/debug`, `os/signal`, `syscall` | |
| `net/http/httptest`, `net/http/internal` | |
| `crypto/md5`, `crypto/sha1`, `crypto/sha256`, `crypto/sha512`, `crypto/subtle` | |
| `crypto/hmac`, `crypto/rand`, `crypto/tls` | |
| `crypto/x509`, `crypto/x509/pkix` | |
| `mime`, `mime/multipart`, `mime/quotedprintable` | |
| `net/textproto`, `net/http/cookiejar` | |
| `hash`, `hash/adler32`, `hash/crc32`, `hash/fnv` | |
| `io`, `io/fs`, `iter`, `maps`, `slices`, `sort` | |
| `math`, `math/big`, `math/bits`, `math/cmplx`, `math/rand`, `math/rand/v2` | |
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
| A `//go:embed` directive | Present. Stage 1 writes the files into the declaration of the variable. |
| The address of a value behind a pointer, for example `&*p` | Rejected. Read `p`. |
| A goroutine on a second thread | All Go code runs in 1 Dart isolate. `GOMAXPROCS` gives 1. |
| A conversion between 2 struct types | Rejected |
| A function that can block, in a value of a type that carries `//gart:sync` | Rejected. Read decision 221. |
| A function declaration with no body | Rejected, and a `//gart:dart` directive accepts it. Read decision 239. |
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

## The targets

The compiler writes Dart, thus the target is each target of Dart. The table
holds the state of 2026-09-03.

| Target | State |
|---|---|
| The Dart VM, `dart run` | Present. `make run PROG=x` takes this path. |
| Ahead of time, `dart compile exe` | Present. Each measurement of `docs/flutter-design.md` reads a binary of this form. |
| macOS, through Flutter | Present. The 3 examples of `flutter/` each build and run. |
| iOS, through Flutter | Present. `flutter build ios --debug --no-codesign` builds the 3. No plugin stands in the way, thus CocoaPods is not needed. |
| Android, through Flutter | Present. `flutter build apk --debug` builds the 3. It needs `ANDROID_HOME`, and it does not need the cmdline-tools that `flutter doctor` asks for. |
| The web, `dart compile js` and `dart compile wasm` | Present. Each program compiles, and a file or a socket operation gives the error that Go gives under `GOOS=js`. A Flutter web build reads the same path. Read decision 251 and `docs/js-integers.md`. |

`flutter/example/platform.sh` writes the platform runner of an example. git
holds no such directory, because `flutter test` needs none and each file comes
from `flutter create`. The script also gives a macOS runner the entitlement
`com.apple.security.network.client`, and an Android runner
`android.permission.INTERNET` in the main manifest.

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
| `go/importer` | Each program that reads a compiled export file. A caller writes an importer of its own over `go/types`, thus the gate is narrow. |
| A goroutine on a second thread | Each program that needs a parallel run. All Go code runs in 1 isolate. |

The list shortened on 2026-09-03. `html/template`, `encoding/xml`,
`archive/zip`, `archive/tar`, `compress/lzw`, `compress/bzip2`, `math/big`,
`encoding/gob`, `crypto/x509`, `database/sql`, `database/sql/driver`, `uuid`,
`net/mail`, `net/smtp`, `net/rpc`, `net/http/cgi`, `go/constant`, `go/types`,
and `go/build` each build now. `hugo` still reads a large module set. The gate
is the library surface, and not the language.

## The near misses

Each package below needs a small piece of work. `docs/support.md` holds the
whole list of the packages that stay out, with the cause and the cost of each
one.

No package stands here. `docs/support.md` holds the whole list of the packages
that stay out, with the cause and the cost of each one.

`crypto/subtle` and `math/cmplx` stood here until 2026-09-02, and
`html/template`, `encoding/xml`, `archive/zip`, `archive/tar`, `compress/lzw`,
`compress/bzip2`, `math/big`, `encoding/gob`, and `crypto/x509` until
2026-09-03. Each 1 builds now. `encoding/gob` needed 6 reflect names and a
`reflect.Kind` that is a `uint`, and `crypto/x509` needed the whole ASN.1
stack. `crypto/x509` holds 115 of 261 names, and chain verification stays out
because `dart:io` owns the trust decision.

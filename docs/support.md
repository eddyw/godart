# The support of each package

This file names how much of each standard library package the surface holds.
`docs/what-compiles.md` names the shape of a program that builds.
`docs/parity.md` names each behavior that differs.

## The measure

The measure of 2026-09-01 counts the exported names of a package: each name of
the package scope, each exported method of the pointer to an exported named
type, and each exported field of an exported struct. It loads the package 2
times, first from the real Go 1.27 source and then through the overlay of the
compiler, and it compares the 2 sets.

The walk reads the method set of the pointer, thus it counts a method with a
pointer receiver. An earlier measure read the value method set only, and each
count of that measure was lower.

The ratio counts names, and not use. `net` holds 522 names, because the address
types and the resolver hold most of them, and a program that dials and listens
reads 26. Read the gap column, and not the percent alone.

The surface holds 92 import paths: 82 standard packages, the interop package
`syscall/dart`, and 9 internal packages that the 82 need.

## The 3 forms

| Form | Meaning |
|---|---|
| Native | The compiler reads the real Go source. |
| Patch | The compiler reads the real Go source, and 1 file or more of the overlay replaces a file of it. |
| Overlay | The overlay replaces the whole package. |

A native package and a patched package hold each name of Go, because the real
source stands behind them. Each partial package below is an overlay.

## Full support

66 packages hold each exported name of Go.

| Form | Packages |
|---|---|
| Native, 32 | `bufio`, `cmp`, `compress/flate`, `compress/gzip`, `compress/zlib`, `container/heap`, `container/list`, `container/ring`, `context`, `encoding`, `encoding/ascii85`, `encoding/base64`, `encoding/csv`, `encoding/hex`, `go/doc/comment`, `go/format`, `go/printer`, `go/scanner`, `go/token`, `hash`, `hash/adler32`, `hash/fnv`, `io`, `mime`, `mime/quotedprintable`, `net/http/cookiejar`, `net/textproto`, `net/url`, `path`, `regexp`, `testing/iotest`, `unicode` |
| Patch, 13 | `bytes`, `go/ast`, `go/build/constraint`, `go/parser`, `hash/crc32`, `math`, `math/bits`, `math/rand`, `math/rand/v2`, `mime/multipart`, `net/netip`, `regexp/syntax`, `text/tabwriter` |
| Overlay, 21 | `crypto/hmac`, `crypto/md5`, `crypto/sha1`, `crypto/sha256`, `crypto/sha512`, `encoding/binary`, `errors`, `flag`, `iter`, `log`, `maps`, `path/filepath`, `runtime/debug`, `slices`, `sort`, `strings`, `sync`, `testing/quick`, `time`, `unicode/utf8`, `unique` |

`time/tzdata` holds no exported name. The import gives the embedded zone
database to `time`, and it works.

A full package can still differ in behavior. `path/filepath` holds each name,
and the separator is always `/`. `docs/parity.md` holds each such difference.

## Partial support

15 packages hold a part of the names of Go.

| Package | Names | Held | The largest gap |
|---|---|---|---|
| `fmt` | 26/29 | 90% | `Formatter`, `State`, `FormatString` |
| `net/http/httptest` | 27/31 | 87% | `Server.TLS`, `Server.Certificate`, `CloseClientConnections`, `NewTestServer` |
| `io/fs` | 114/134 | 85% | The `FS` interface family: `FS`, `File`, `ReadDir`, `Glob`, `Lstat`, `Sub` |
| `os` | 207/271 | 76% | The `Root` type, `DirFS`, `Chmod`, `Chown`, the signal names, `NewFile`, `StartProcess` |
| `os/exec` | 37/49 | 76% | `CommandContext`, `ErrDot`, `ErrWaitDelay` |
| `strconv` | 30/43 | 70% | `QuoteRune` and its 3 forms, `IsPrint`, `IsGraphic`, `ParseComplex` |
| `encoding/json` | 52/82 | 63% | `Token`, `Delim`, `Decoder.Token`, `Decoder.UseNumber`, the option set of v2, the field `Type` of each error |
| `testing` | 92/145 | 63% | `Init`, `MainStart`, the cover types, `InternalExample`, `InternalFuzzTarget` |
| `crypto/rand` | 3/5 | 60% | `Int` and `Prime`. The 2 read `math/big`. |
| `sync/atomic` | 55/94 | 59% | `Uintptr`, the pointer functions, the `And` and `Or` forms |
| `reflect` | 110/219 | 50% | `SliceOf`, `MapOf`, `ChanOf`, `FuncOf`, `ArrayOf`, `Append`, `Copy`, `MakeFunc`, `Select` |
| `net/http` | 185/400 | 46% | `FileServer` and the `FS` family, `ServeContent`, `Hijacker`, `ResponseController`, `ConnState`, 30 status constants |
| `runtime` | 27/114 | 24% | `MemStats`, `ReadMemStats`, `Goexit`, `Breakpoint`, `SetFinalizer`, each profile |
| `crypto/tls` | 30/295 | 10% | The cipher suite constants, the curve names, `CipherSuites`, `ClientHelloInfo`, `LoadX509KeyPair`, `Client`, `Server` |
| `net` | 26/522 | 5% | `IPNet`, `ParseCIDR`, `TCPAddr`, `UDPAddr`, the `Lookup` family, `Interfaces`, the `Resolver` |

## No support

Stage 2 rejects an import of each package below with a source position.

| Package | The cause | The cost |
|---|---|---|
| `os/signal` | No overlay yet. `ProcessSignal.watch` of `dart:io` gives the event. | Small |
| `math/cmplx` | No survey yet. The language holds the complex type already. | Small |
| `crypto/subtle` | No overlay yet. Each function is a loop over 2 byte slices. | Small |
| `embed` | The compiler holds no `go:embed` directive. | Medium |
| `text/template`, `html/template` | The real source reads the deep reflect surface. | Medium |
| `encoding/xml` | The real source reads the deep reflect surface. | Medium |
| `archive/zip`, `archive/tar`, `compress/bzip2`, `compress/lzw` | No survey yet. Each one is pure Go. | Medium |
| `math/big` | No survey yet. The real source reads assembly for each word operation. | Large |
| `encoding/gob` | The real source reads `unsafe` and the deep reflect surface. | Large |
| `crypto/x509` | The whole ASN.1 stack and a root pool. | Large |
| `database/sql` | A driver, and a pool over a second thread. | Large |
| `go/types`, `go/importer`, `go/build` | Each one is large, and `go/build` reads the tool chain. | Large |
| `net/rpc`, `net/smtp`, `net/mail` | Each one reads `encoding/gob` or `net/textproto`. | Large |
| `net/http/cgi` | No process model of a server. | Large |
| `unsafe`, cgo | Outside the subset by design. | Rejected |

## The focus list

The order follows the value of the work against its cost. It reads a general Go
program. `docs/flutter-first.md` holds a different order for a Flutter client,
and `docs/flutter-design.md` holds the design of that client.

1. **The `FS` family.** `io/fs.FS`, `os.DirFS`, and `http.FileServer` together
   give a static file server, and they open the road to `embed`.
2. **The reflect gaps.** `SliceOf`, `MapOf`, `PointerTo`, `MakeFunc`, `Append`,
   and `Copy`. `text/template` needs each one.
3. **`text/template`.** It is the largest single gate of a real program. A
   command line tool, a code generator, and a web server each read it.
4. **The address types of `net`.** `IPNet`, `ParseCIDR`, `TCPAddr`, and the 2
   address methods of a connection. `Request.RemoteAddr` of `net/http` is empty
   without them.
5. **`os/signal`.** A graceful shutdown needs it, and the Dart call exists.
6. **The `Token` family of `encoding/json`.** `Decoder.Token`, `Delim`, and
   `UseNumber`. A stream of a large array reads them.
7. **`Formatter` and `State` of `fmt`.** A type that writes its own form under
   `%v` needs the pair. The path follows `Marshaler` of `encoding/json`: the
   Dart walk reads the method table of the type descriptor.
8. **The `And` and `Or` forms of `sync/atomic`,** and the pointer functions.
9. **`os.Root` and `os.DirFS`.** The 2 come with the `FS` family of item 1.

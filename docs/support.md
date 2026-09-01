# The support of each package

This file names how much of each standard library package the surface holds.
`docs/what-compiles.md` names the shape of a program that builds.
`docs/parity.md` names each behavior that differs.

## The measure

Run `make surface`. `internal/surface` holds the tool, and the numbers below
come from the run of 2026-09-02.

The measure counts the exported names of a package: each name of the package
scope, each exported method of the pointer to an exported named type, and each
exported field of an exported struct. It loads the package 2 times, first from
the real Go 1.27 source and then through the overlay of the compiler, and it
compares the 2 sets.

`make surface ARGS=-gaps` names each missing name, and
`make surface ARGS="-pkg os"` measures 1 package.

The ratio counts names, and not use. `net` holds 346 names, because the address
types and the resolver hold most of them, and a program that dials and listens
reads 26. Read the gap column, and not the percent alone.

The surface holds 95 import paths: 84 standard packages, the interop package
`syscall/dart`, and 10 internal packages that the 84 need. The measure skips
the interop package and each internal package, because no package of Go stands
behind them.

## The 3 forms

| Form | Meaning |
|---|---|
| Native | The compiler reads the real Go source. |
| Patch | The compiler reads the real Go source, and 1 file or more of the overlay replaces a file of it. |
| Overlay | The overlay replaces the whole package. |

A native package and a patched package hold each name of Go, because the real
source stands behind them. Each partial package below is an overlay.

## Full support

86 packages hold each exported name of Go.

| Form | Packages |
|---|---|
| Native, 38 | `bufio`, `cmp`, `compress/flate`, `compress/gzip`, `compress/zlib`, `container/heap`, `container/list`, `container/ring`, `context`, `encoding`, `encoding/ascii85`, `encoding/base64`, `encoding/csv`, `encoding/hex`, `go/doc/comment`, `go/format`, `go/printer`, `go/scanner`, `go/token`, `hash`, `hash/adler32`, `hash/fnv`, `io`, `io/fs`, `mime`, `mime/quotedprintable`, `net/http/cookiejar`, `net/textproto`, `net/url`, `path`, `regexp`, `testing/iotest`, `text/template`, `text/template/parse`, `unicode` |
| Patch, 13 | `bytes`, `go/ast`, `go/build/constraint`, `go/parser`, `hash/crc32`, `math`, `math/bits`, `math/rand`, `math/rand/v2`, `mime/multipart`, `net/netip`, `regexp/syntax`, `text/tabwriter` |
| Overlay, 23 | `crypto/hmac`, `crypto/md5`, `crypto/sha1`, `crypto/sha256`, `crypto/sha512`, `encoding/binary`, `errors`, `flag`, `fmt`, `iter`, `log`, `maps`, `os/signal`, `path/filepath`, `runtime/debug`, `slices`, `sort`, `strings`, `sync`, `testing/quick`, `time`, `unicode/utf8`, `unique` |

`time/tzdata` holds no exported name. The import gives the embedded zone
database to `time`, and it works.

A full package can still differ in behavior. `path/filepath` holds each name,
and the separator is always `/`. `docs/parity.md` holds each such difference.

## Partial support

15 packages hold a part of the names of Go. The run of 2026-09-02:

| Package | Names | Held | The largest gap |
|---|---|---|---|
| `sync/atomic` | 90/94 | 95% | The 4 `unsafe.Pointer` functions. `Pointer[T]` holds the same use. |
| `net/http/httptest` | 27/31 | 87% | `Server.TLS`, `Server.Certificate`, `CloseClientConnections`, `NewTestServer` |
| `os/exec` | 37/49 | 75% | `CommandContext`, `Cmd.Cancel`, `Cmd.Environ`, `Cmd.WaitDelay` |
| `os` | 146/207 | 70% | `Root`, `DirFS`, `CopyFS`, `Chmod`, `Chown`, `Chtimes` |
| `strconv` | 30/43 | 69% | `QuoteRune` and its 3 forms, `IsPrint`, `IsGraphic`, `ParseComplex` |
| `io/fs` | 39/59 | 66% | The `FS` interface family: `FS`, `File`, `Glob`, `ReadDir`, `Sub` |
| `encoding/json` | 59/92 | 64% | The option set of the v2 implementation, and the field `Type` of each error |
| `testing` | 92/145 | 63% | `Init`, `MainStart`, the cover types, `InternalExample`, `InternalFuzzTarget` |
| `crypto/rand` | 3/5 | 60% | `Int` and `Prime`. The 2 read `math/big`. |
| `net/http` | 185/392 | 47% | `FileServer` and the `FS` family, `ServeContent`, `Hijacker`, `ResponseController` |
| `reflect` | 126/188 | 67% | `MakeFunc`, `ChanOf`, `FuncOf`, `MakeChan`, `Select`, `Method.Func` |
| `runtime` | 31/114 | 27% | `MemStats`, `ReadMemStats`, `Goexit`, `Breakpoint`, `SetFinalizer`, each profile |
| `crypto/tls` | 32/263 | 12% | The cipher suite constants, the curve names, `CipherSuites`, `LoadX509KeyPair` |
| `net` | 26/346 | 7% | `IPNet`, `ParseCIDR`, `TCPAddr`, `UDPAddr`, the `Lookup` family, `Interfaces`, the `Resolver` |
| `syscall` | 15/2228 | 0% | Each system call, each error number, and each constant of the platform. |

`syscall` is a special case. The package of Go holds the whole system call
interface of the platform, and the surface holds no such interface. The overlay
holds the `Signal` type and the 12 signal numbers that macOS and Linux agree
on, because `os` and `os/signal` need them. A Go program reaches the system
through `os`, through `net`, and through the interop package `syscall/dart`.

The gap of `encoding/json` is the option set of the v2 implementation. Go 1.27
routes `encoding/json` through that implementation, thus the measure counts
`DefaultOptionsV1`, `FormatDurationAsNano`, and 20 more names of it. The v1 API
is complete, with `Token`, `Delim`, and `UseNumber` since 2026-09-02.

## No support

Stage 2 rejects an import of each package below with a source position.

| Package | The cause | The cost |
|---|---|---|
| `math/cmplx` | No survey yet. The language holds the complex type already. | Small |
| `crypto/subtle` | No overlay yet. Each function is a loop over 2 byte slices. | Small |
| `embed` | The compiler holds no `go:embed` directive. | Medium |
| `html/template` | It reads `text/template` and the escape pass of HTML. | Medium |
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
2. **The reflect gaps.** Done 2026-09-02, except `MakeFunc`. `SliceOf`,
   `ArrayOf`, `MapOf`, `PointerTo`, `PtrTo`, `TypeFor`, `Append`,
   `AppendSlice`, `Copy`, and `Value.Cap`. `MakeSlice` now honors its capacity.
   `MakeFunc` has no form: the Dart type of a Go function type is a closure of
   that exact signature, and the runtime holds no way to build one. Read
   `docs/parity.md`.
3. **`text/template`.** Done 2026-09-02. `text/template` and
   `text/template/parse` each read the real Go source, and each one holds every
   exported name. A field, a method, a range, a map, a variable, a pipe, a
   custom function, each builtin, and a nested template each give the text that
   Go gives.
4. **The address types of `net`.** `IPNet`, `ParseCIDR`, `TCPAddr`, and the 2
   address methods of a connection. `Request.RemoteAddr` of `net/http` is empty
   without them.
5. **`os/signal`.** Done 2026-09-02. A graceful shutdown reads it, and
   `(*os.Process).Signal` sends a signal to any process.
6. **The `Token` family of `encoding/json`.** Done 2026-09-02. `Decoder.Token`,
   `Delim`, and `Decoder.UseNumber`. The parse of the runtime now keeps the text
   of each number literal, thus a `uint64` above the largest `int64` keeps its
   value.
7. **`Formatter` and `State` of `fmt`.** Done 2026-09-02. `Sprintf` calls the
   method `Format` of a value that holds one, and the method then owns the
   width, the precision, and each flag. `FormatString` gives the verb again.
8. **The `And` and `Or` forms of `sync/atomic`,** and the pointer functions.
   Done 2026-09-02, except the 4 `unsafe.Pointer` functions, which stand
   outside the subset.
9. **`os.Root` and `os.DirFS`.** The 2 come with the `FS` family of item 1.

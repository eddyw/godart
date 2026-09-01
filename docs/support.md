# The support of each package

This file names how much of each standard library package the surface holds.
`docs/what-compiles.md` names the shape of a program that builds.
`docs/parity.md` names each behavior that differs.

## The measure

The measure of 2026-09-01 counts the exported names of a package: each name of
the package scope, each exported method of an exported named type, and each
exported field of an exported struct. It loads the package 2 times, first from
the real Go 1.27 source and then through the overlay of the compiler, and it
compares the 2 sets.

The ratio counts names, and not use. `net` holds 302 names, because the address
types and the resolver hold most of them, and a program that dials and listens
reads 15. Read the gap column, and not the percent alone.

The surface holds 83 import paths: 74 standard packages, the interop package
`syscall/dart`, and 8 internal packages that the 74 need.

## The 3 forms

| Form | Meaning |
|---|---|
| Native | The compiler reads the real Go source. |
| Patch | The compiler reads the real Go source, and 1 file or more of the overlay replaces a file of it. |
| Overlay | The overlay replaces the whole package. |

A native package and a patched package hold each name of Go, because the real
source stands behind them. Each partial package below is an overlay.

## Full support

52 packages hold each exported name of Go.

| Form | Packages |
|---|---|
| Native, 28 | `bufio`, `cmp`, `compress/flate`, `compress/gzip`, `compress/zlib`, `container/heap`, `container/list`, `container/ring`, `context`, `encoding`, `encoding/ascii85`, `encoding/base64`, `encoding/csv`, `encoding/hex`, `go/doc/comment`, `go/format`, `go/printer`, `go/scanner`, `go/token`, `hash`, `hash/adler32`, `hash/fnv`, `io`, `net/url`, `path`, `regexp`, `testing/iotest`, `unicode` |
| Patch, 12 | `bytes`, `go/ast`, `go/build/constraint`, `go/parser`, `hash/crc32`, `math`, `math/bits`, `math/rand`, `math/rand/v2`, `net/netip`, `regexp/syntax`, `text/tabwriter` |
| Overlay, 12 | `crypto/md5`, `crypto/sha1`, `crypto/sha256`, `encoding/binary`, `iter`, `log`, `maps`, `path/filepath`, `slices`, `testing/quick`, `unicode/utf8`, `unique` |

`time/tzdata` holds no exported name. The import gives the embedded zone
database to `time`, and it works.

A full package can still differ in behavior. `path/filepath` holds each name,
and the separator is always `/`. `docs/parity.md` holds each such difference.

## Partial support

21 packages hold a part of the names of Go.

| Package | Names | Held | The largest gap |
|---|---|---|---|
| `net/http/httptest` | 27/31 | 87% | `Server.TLS`, `Server.Certificate`, `CloseClientConnections` |
| `flag` | 73/85 | 85% | `Duration`, `DurationVar`, `Func`, `TextVar`, `BoolFunc` |
| `errors` | 6/7 | 85% | `ErrUnsupported` |
| `time` | 120/145 | 82% | `AddDate`, `ISOWeek`, `Zone`, `UnixMilli`, `UnixMicro`, the 6 marshal methods |
| `os/exec` | 32/40 | 80% | `CommandContext`, `Cmd.Environ`, `Cmd.Cancel`, `ExtraFiles`, `ErrDot` |
| `strings` | 61/83 | 73% | `NewReplacer`, `SplitAfter`, `Clone`, `ToValidUTF8`, `LastIndexAny`, `Lines` |
| `strconv` | 30/43 | 69% | `QuoteRune` and its 3 forms, `IsPrint`, `IsGraphic`, `ParseComplex` |
| `io/fs` | 39/59 | 66% | The `FS` interface family: `FS`, `File`, `ReadDir`, `Stat`, `Glob`, `Sub`, `WalkDir` |
| `sync/atomic` | 55/94 | 58% | `Uintptr`, the pointer functions, the `And` and `Or` forms |
| `testing` | 45/81 | 55% | `T.Deadline`, `T.Chdir`, the cover types, the memory of a benchmark |
| `sync` | 22/44 | 50% | `Map`, `Cond`, `Locker`, `WaitGroup.Go`, `RWMutex.TryLock` |
| `os` | 90/196 | 45% | The `Root` type, `DirFS`, the signal names, `Chmod`, `Getpid`, `Executable`, `UserHomeDir` |
| `reflect` | 82/188 | 43% | `Value.MapRange`, `Value.Convert`, `Value.IsZero`, `SliceOf`, `MapOf`, `PointerTo`, `MakeFunc`, `Select` |
| `net/http` | 167/392 | 42% | `FileServer` and the `FS` family, `ServeContent`, `CookieJar`, `Hijacker`, `ResponseController`, 30 status constants |
| `fmt` | 12/29 | 41% | The scan family: `Sscanf`, `Sscan`, `Fscan`, `Scan`. Also the `Append` family, `Formatter`, `State` |
| `sort` | 12/37 | 32% | `Reverse`, `IsSorted`, `IntSlice`, `StringSlice`, `Float64Slice`, `SearchStrings` |
| `runtime` | 27/113 | 23% | `MemStats`, `ReadMemStats`, `Stack`, `Gosched`, `Goexit`, `KeepAlive`, `SetFinalizer`, each profile |
| `crypto/tls` | 30/263 | 11% | The cipher suite constants, the curve names, `CipherSuites`, `ClientHelloInfo`, `LoadX509KeyPair` |
| `encoding/json` | 7/82 | 8% | `Encoder`, `Decoder`, `RawMessage`, `Marshaler`, `Unmarshaler`, `Valid`, `Indent`, `Number` |
| `net` | 15/302 | 4% | `IP`, `ParseIP`, `IPNet`, `ParseCIDR`, `TCPAddr`, `UDPAddr`, the `Lookup` family, `Interfaces` |
| `runtime/debug` | 1/37 | 2% | `Stack`, `PrintStack`, `ReadBuildInfo`, `SetGCPercent`, `GCStats` |

## No support

Stage 2 rejects an import of each package below with a source position.

| Package | The cause | The cost |
|---|---|---|
| `crypto/hmac`, `crypto/sha512` | No overlay yet. Each one follows the pattern of the `crypto/sha256` overlay. | Small |
| `os/signal` | No overlay yet. `ProcessSignal.watch` of `dart:io` gives the event. | Small |
| `math/cmplx` | No survey yet. The language holds the complex type already. | Small |
| `crypto/rand` | No overlay yet. `Random.secure` of Dart gives the bytes. | Small |
| `embed` | The compiler holds no `go:embed` directive. | Medium |
| `text/template`, `html/template` | The real source reads the deep reflect surface. | Medium |
| `encoding/xml` | The real source reads the deep reflect surface. | Medium |
| `mime`, `net/textproto`, `mime/multipart` | No overlay yet. A form of `net/http` needs them. | Medium |
| `archive/zip`, `archive/tar`, `compress/bzip2`, `compress/lzw` | No survey yet. Each one is pure Go. | Medium |
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
where `encoding/json` and the callback boundary come first.

1. **The small gaps of 7 packages.** `sort.Reverse` and the 3 typed slices,
   `strings.NewReplacer` and `SplitAfter`, `sync.Map` and `sync.Cond`,
   `time.AddDate` and `UnixMilli`, `flag.Duration`, `os.Getpid` and
   `UserHomeDir`, and `debug.Stack`. Each one is pure Go inside the subset. A
   plain program reads these names each day.
2. **The scan family of `fmt`.** `Sscanf` and `Fscan` read the input of a
   program. The verb table of the writer already stands.
3. **`encoding/json`: `Encoder`, `Decoder`, `RawMessage`, `Marshaler`.** The
   type descriptor already holds the walk. A stream reader and the 2 interfaces
   are the gap.
4. **The `FS` family.** `io/fs.FS`, `os.DirFS`, and `http.FileServer` together
   give a static file server, and they open the road to `embed`.
5. **The reflect gaps.** `Value.MapRange`, `Value.Convert`, `Value.IsZero`,
   `SliceOf`, `MapOf`, and `PointerTo`. `text/template` needs each one.
6. **`text/template`.** It is the largest single gate of a real program. A
   command line tool, a code generator, and a web server each read it.
7. **`os/signal`.** A graceful shutdown needs it, and the Dart call exists.
8. **`crypto/hmac` and `crypto/sha512`.** Each one follows the overlay that
   `crypto/sha256` holds.
9. **The address types of `net`.** `IP`, `ParseIP`, `TCPAddr`, and the 2
   address methods of a connection. `Request.RemoteAddr` of `net/http` is empty
   without them.

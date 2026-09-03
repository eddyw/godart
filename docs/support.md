# The support of each package

This file names how much of each standard library package the surface holds.
`docs/what-compiles.md` names the shape of a program that builds.
`docs/parity.md` names each behavior that differs.

## The measure

Run `make surface`. `internal/surface` holds the tool, and the numbers below
come from the run of 2026-09-03.

The measure counts the exported names of a package: each name of the package
scope, each exported method of the pointer to an exported named type, and each
exported field of an exported struct. It loads the package 2 times, first from
the real Go 1.27 source and then through the overlay of the compiler, and it
compares the 2 sets.

`make surface ARGS=-gaps` names each missing name, and
`make surface ARGS="-pkg os"` measures 1 package.

The ratio counts names, and not use. `net` holds 346 names, because the typed
connections and the interface list hold most of them. Read the gap column, and
not the percent alone.

The surface holds 170 import paths: 140 standard packages, the interop package
`syscall/dart`, 23 internal packages, and 6 vendored packages of
`golang.org/x`. `make surface` measures 123 of the standard packages against
the real API, and 112 of those hold each exported name. The measure skips
the interop package and each internal package, because no package of Go stands
behind them.

A native package that also holds a patch stands 2 times in the list of the
surface, thus the measure removes each repeat first.

## The 3 forms

| Form | Meaning |
|---|---|
| Native | The compiler reads the real Go source. |
| Patch | The compiler reads the real Go source, and 1 file or more of the overlay replaces a file of it. |
| Overlay | The overlay replaces the whole package. |

A native package and a patched package hold each name of Go, because the real
source stands behind them. Each partial package below is an overlay.

## Full support

91 of the 103 standard packages hold each exported name of Go.

| Form | Packages |
|---|---|
| Native, 45 | `archive/zip`, `bufio`, `cmp`, `compress/bzip2`, `compress/flate`, `compress/gzip`, `compress/lzw`, `compress/zlib`, `container/heap`, `container/list`, `container/ring`, `context`, `crypto/x509/pkix`, `encoding`, `encoding/ascii85`, `encoding/base64`, `encoding/csv`, `encoding/hex`, `encoding/pem`, `encoding/xml`, `go/doc/comment`, `go/format`, `go/printer`, `go/scanner`, `go/token`, `hash`, `hash/adler32`, `hash/fnv`, `html`, `html/template`, `io`, `io/fs`, `math/cmplx`, `mime`, `mime/quotedprintable`, `net/http/cookiejar`, `net/textproto`, `net/url`, `path`, `regexp`, `testing/iotest`, `text/template`, `text/template/parse`, `unicode`, `unicode/utf16` |
| Patch, 16 | `archive/tar`, `bytes`, `encoding/asn1`, `go/ast`, `go/build/constraint`, `go/parser`, `hash/crc32`, `math`, `math/big`, `math/bits`, `math/rand`, `math/rand/v2`, `mime/multipart`, `net/netip`, `regexp/syntax`, `text/tabwriter` |
| Overlay, 30 | `crypto/hmac`, `crypto/md5`, `crypto/rand`, `crypto/sha1`, `crypto/sha256`, `crypto/sha512`, `crypto/subtle`, `embed`, `encoding/binary`, `encoding/gob`, `errors`, `flag`, `fmt`, `iter`, `log`, `maps`, `net/http/httptest`, `os/signal`, `path/filepath`, `runtime/debug`, `slices`, `sort`, `strconv`, `strings`, `sync`, `testing/quick`, `time`, `time/tzdata`, `unicode/utf8`, `unique` |

`time/tzdata` holds no exported name. The import gives the embedded zone
database to `time`, and it works.

A full package can still differ in behavior. `path/filepath` holds each name,
and the separator is always `/`. `docs/parity.md` holds each such difference.

## Partial support

11 packages hold a part of the names of Go. The run of 2026-09-03, after the
fifteenth session:

| Package | Names | Held | The largest gap |
|---|---|---|---|
| `sync/atomic` | 90/94 | 95% | The 4 `unsafe.Pointer` functions. `Pointer[T]` holds the same use. |
| `os` | 193/207 | 93% | `File.Fd`, `File.SyscallConn`, `NewFile`, `StartProcess`, `ProcAttr`, and the 4 time names of `ProcessState`. `dart:io` hides the descriptor, and it reports no CPU time. |
| `os/exec` | 43/49 | 87% | `Cmd.ExtraFiles`, `Cmd.SysProcAttr`, `ExitError.Sys` and its 3 companions |
| `reflect` | 173/188 | 92% | `FuncOf`, `NewAt`, `SliceAt`, `SliceHeader`, `StringHeader`, `Method.Func`. Each 1 needs a type that no emitted code names, or `unsafe`. |
| `encoding/json` | 67/92 | 72% | The option set of the v2 implementation, and the field `Type` of each error |
| `testing` | 145/145 | 100% | Full since 2026-09-03. |
| `net/http` | 341/392 | 87% | `HTTP2Config`, `Pusher`, `ClientConn`, `CrossOriginProtection`, `ListenAndServeTLS`. Dart owns the handshake and the HTTP 2 frame layer. |
| `runtime` | 43/114 | 37% | `MemStats`, `ReadMemStats`, `Breakpoint`, `SetFinalizer`, `Pinner`, each profile, and the trace. The Dart VM reports none of the numbers. |
| `net` | 218/346 | 63% | `IPConn`, `Resolver` as a configurable type, each socket option method, `FileConn`, and `FileListener`. Each 1 needs a raw socket or a file descriptor. |
| `crypto/tls` | 118/263 | 44% | The QUIC connection, `ConnectionState.PeerCertificates`, and each `Config` field that would change the handshake |
| `crypto/x509` | 115/261 | 44% | `Verify` and each verify error, `CheckSignature`, `PublicKey`, and each name that signs |
| `syscall` | 15/2228 | 0% | Each system call, each error number, and each constant of the platform. |

`testing` left this table on 2026-09-03. `reflect` went from 130 names to 173,
`net/http` from 240 to 335, `net` from 142 to 218, `runtime` from 31 to 43,
`encoding/json` from 59 to 67, and `crypto/x509` from 106 to 115.

`syscall` is a special case. The package of Go holds the whole system call
interface of the platform, and the surface holds no such interface. The overlay
holds the `Signal` type and the 12 signal numbers that macOS and Linux agree
on, because `os` and `os/signal` need them. A Go program reaches the system
through `os`, through `net`, and through the interop package `syscall/dart`.

The gap of `encoding/json` is the option set of the v2 implementation. Go 1.27
routes `encoding/json` through that implementation, thus the measure counts
`DefaultOptionsV1`, `FormatDurationAsNano`, and 20 more names of it. The v1 API
is complete, with `Token`, `Delim`, and `UseNumber` since 2026-09-02.

## The open file limit of the machine

A run of `make ci` writes more than 1100 temporary Go modules: 200 programs of
`testdata/programs`, and 931 files of the test directory of Go. Each 1 holds a
`go.mod`, thus `gopls` reads each 1 that appears.

A measure of 2026-09-04 named `go mod tidy` of `gopls` at 5273 open files, 3
seconds after the editor started. A report of the same day named 10004 for the
same process. The machine then reaches `kern.maxfiles` and each process on it
fails, `lsof` included.

The repository answers its own half. `output` of
`internal/driver/exec_test.go` gives each external command a deadline of 5
minutes and a process group of its own, and `spawnSlots` holds 8 slots. Read
decision 317.

The editor answers the other half. `.vscode/settings.json` keeps `testdata`,
`flutter/example`, `.claude`, and `build` out of the workspace of `gopls`:

```json
{
  "gopls": {
    "build.directoryFilters": ["-testdata", "-flutter/example", "-.claude", "-build"]
  }
}
```

`sysctl` needs no file descriptor, thus it answers when nothing else does:

```sh
sysctl kern.num_files kern.maxfiles
```

`scripts/fdwatch.sh` samples that counter each second, and it names the top
holders before the machine runs out.

## No support

Stage 2 rejects an import of each package below with a source position.

| Package | The cause | The cost |
|---|---|---|
| `go/importer` | `Default` reads `runtime.Compiler`, which gart reports as `gart`. The gc form reads a compiled export file that gart never writes, the gccgo form needs 5 more packages and the gccgo driver, and the source form fails on the same compiler name. A caller writes an importer of its own over `go/types`. | Rejected |
| `unsafe`, cgo | Outside the subset by design. | Rejected |

## The focus list

The order follows the value of the work against its cost. It reads a general Go
program. `docs/flutter-first.md` holds a different order for a Flutter client,
and `docs/flutter-design.md` holds the design of that client.

1. **The `FS` family.** Done 2026-09-02. `io/fs` reads the real Go source and
   holds each exported name. `os.DirFS`, `os.Root`, and `http.FileServer` give
   a static file server, and they open the road to `embed`.
2. **The reflect gaps.** Done 2026-09-02, and `MakeFunc` followed on
   2026-09-03. `SliceOf`, `ArrayOf`, `MapOf`, `PointerTo`, `PtrTo`, `TypeFor`,
   `Append`, `AppendSlice`, `Copy`, and `Value.Cap`. `MakeSlice` honors its
   capacity. `MakeFunc` reads the bind factory of the signature descriptor,
   which builds a closure of the exact Dart type. `FuncOf` and `StructOf` stay
   out, because each 1 builds a type that no emitted code names. Read
   `docs/parity.md`.
3. **`text/template`.** Done 2026-09-02. `text/template` and
   `text/template/parse` each read the real Go source, and each one holds every
   exported name. A field, a method, a range, a map, a variable, a pipe, a
   custom function, each builtin, and a nested template each give the text that
   Go gives.
4. **The address types of `net`.** Done 2026-09-02. `IPNet`, `ParseCIDR`,
   `TCPAddr`, `UDPAddr`, and the 2 address methods of a connection.
   `Request.RemoteAddr` of `net/http` reads now.
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
9. **`os.Root` and `os.DirFS`.** Done 2026-09-02. The 2 came with the `FS`
   family of item 1. `os` went from 146 names to 172.

**The list is closed since 2026-09-02,** and `reflect.MakeFunc` followed on
2026-09-03.
`docs/parity.md` holds the rule for that name.

## The next list

The order again follows the value of the work against its cost.

1. **The 23 status constants of `net/http`.** Done 2026-09-02. The overlay
   holds `status.go` of the real Go 1.27 source, thus the 62 codes and
   `StatusText` each match. `net/http` went from 200 names to 223.
2. **`math/cmplx` and `crypto/subtle`.** Done 2026-09-02. `math/cmplx` reads
   the real Go source and needed no compiler change, because it imports `math`
   and `math/bits` alone. `crypto/subtle` takes an overlay of 8 names, because
   `boolToUint8` of `crypto/internal/constanttime` panics and waits for an
   instruction of the machine.
3. **`embed`.** Done 2026-09-02. The loader reads each `//go:embed` directive,
   it resolves the patterns, and it writes the files into the declaration of
   the variable. `docs/parity.md` holds the 2 differences.
4. **The `Lookup` family of `net`.** Done 2026-09-02. The overlay holds a
   resolver of its own, and it needs no `InternetAddress.lookup` of `dart:io`.
   It reads /etc/hosts, /etc/resolv.conf, and /etc/services, and it speaks the
   DNS over UDP with a fallback to TCP. `net` went from 73 names to 127.
5. **`strconv` to full.** Done 2026-09-03. `IsPrint` and `IsGraphic` read
   `unicode`, the quote family takes the machinery of Go, and `ParseComplex`
   and `FormatComplex` come from `atoc.go` and `ctoa.go`.
6. **The reachable names of `os` and `os/exec`.** Done 2026-09-03. `os` went
   from 172 names to 193, and `os/exec` from 37 to 43. `Chmod`, `Chown`,
   `Chtimes`, and `Link` each took a new interop function.
7. **`net/http/httptest`.** Done 2026-09-03. 27 names to 30. The overlay tracks
   each accepted connection itself, thus `CloseClientConnections` works.
8. **`html/template`, `archive/zip`, `archive/tar`, `compress/lzw`, and
   `compress/bzip2`.** Done 2026-09-03. Each 1 reads the real Go source.
   `archive/tar` takes a 3 file patch, because `stat_unix.go` reads
   `syscall.Stat_t` and `os/user`.
9. **`math/big`.** Done 2026-09-03. A 2 file patch gives the 6 functions of
   `arith_decl.go` the pure Go body that `arith.go` already holds.
   `crypto/rand.Int` and `crypto/rand.Prime` stay out, because the 2 would give
   `math/big` to each program that reads `crypto/tls`. Read the section "The
   method group of an interface call" of `docs/parity.md`.
10. **`encoding/xml`.** Done 2026-09-03. It needed 6 reflect names:
    `Type.Bits`, `Type.FieldByIndex`, `Value.Bytes`, `Value.SetBytes`,
    `Value.Grow`, and `Value.SetLen`.
11. **The value types of `crypto/tls`.** Done 2026-09-03. 32 names to 115. Each
    constant, each enumeration with its `String` method, and the 3 cipher suite
    functions. A `Config` field that would change the handshake stays out.

12. **`testing`.** Done 2026-09-03. 92 names to 132. `Chdir`, `Output`,
    `ArtifactDir`, `Attr`, `Deadline`, and the cover surface. `TB` holds the 7
    names of the real `TB`.
13. **`crypto/x509` and `encoding/gob`.** Done 2026-09-03. `crypto/x509` parses
    a certificate and holds a `CertPool`, and chain verification stays out
    because `dart:io` owns the trust decision. `encoding/gob` holds each
    exported name, and `reflect.Kind` became an integer for it.
14. **`net` and `net/http`.** Done 2026-09-03. `TCPConn`, `DialTCP`,
    `ListenConfig`, `Protocols`, `ResponseController`, and `Hijacker`.
15. **An HTTP client over `fetch` for the web target.** Done 2026-09-03.
    `net/http` reaches a real server on the web, and `runtime.GOOS` picks the
    path at run time. Read decision 252.

**The list of 2026-09-02 is closed. The 4 items below opened on 2026-09-03,
and each 1 closed the same day.**

| # | Item | State |
|---|---|---|
| 1 | `Value.Addr().Interface()` gave a plain reference and no typed pointer | **Done.** The cell carries the Dart type of the value. The real `encoding/asn1` reads its own type switch again. |
| 2 | `reflect.New` of a type that is not a struct gave an untyped box | **Done.** A gob decode into `**int` reads. |
| 3 | A pointer to a type parameter lost its method table | **Done.** The description reads the type argument through the dictionary. |
| 4 | `Type.String` held the alias of the source | **Done.** `[]byte` gives `[]uint8`, and `rune` gives `int32`. |

The description of a slice, a map, a channel, a function, and a pointer carries
a factory of the Dart type since 2026-09-03. The kind alone names the Dart type
of each other type, thus only these 5 carry it. Read decision 254.

**The 3 items of that list are closed too, and the fifteenth session of
2026-09-03 closed a backlog of 43 tasks on top of them.**

| # | Item | State |
|---|---|---|
| 1 | `reflect.MakeFunc`, and a method value with a result | **Done.** Each function type descriptor carries a bind factory that builds a closure of the exact Dart type. 1 field serves the method value and `MakeFunc`. Read decision 254 and the report of the fifteenth session. |
| 2 | A type that the program never names gives an untyped container | **Done.** `_withType` threads through `$sliceOf`, `$arrayOf`, and `$mapOf`, and a struct descriptor and each named descriptor carry a factory. |
| 3 | A sound source of the used node types of a Flutter host | **Done.** `flutter/nodescan` reads the Go module with full type information. Read decision 268. |

**1 item stays open, and no work names a date for it.** The seed corpus of a
fuzz target is done, and decision 312 holds it. The verb `%#v` is done, and
decision 315 holds it. The comparison of a pointer inside an interface is done,
and decision 314 holds it.

| # | Item | Cost | What it blocks |
|---|---|---|---|
| 1 | A slice bound of an unsigned type above 2^63 | Small | The report reads a negative number, and Go writes the unsigned value. The runtime call carries no sign of the bound. |

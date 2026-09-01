# The standard library

This file holds the work of the standard library surface. It names the 3 forms
that reach a package, and the design of each package that needed one. Read it
before a change of an overlay or before a new package of the surface.

`docs/what-compiles.md` holds the list of each package of the surface.
`docs/parity.md` holds each difference between Go and the Dart output.

## Standard library

The surface holds 83 import paths: 74 standard packages, `syscall/dart`, and 8
internal packages. `docs/what-compiles.md` holds the whole list and the gate of
each package that stays out. `docs/support.md` holds the count of the exported
names that each one holds. A package reaches the surface 3 ways.

An **overlay** replaces the real source. `internal/overlay/std/<path>/` holds
the files, and each one carries the build tag `dart`. The go command skips such
a file, and the compiler reads it. The load pairs the files by base name. Thus
the name of the Dart part file of the output holds the name of the overlay file.

A **native** package reads the real Go source. `internal/overlay/native.go`
holds the list. Such a package needs no overlay, because its source already
stays inside the subset.

A **patch** replaces 1 file of a native package by its base name. Each other
file keeps the real Go source. A patch that matches no file stays out, because a
build tag of the architecture selects the file set of a package.

`internal/overlay/native.go` holds the native list and each patch.
`internal/overlay/std/` holds the source of each overlay. The 2 files are the
authority, because each build reads them.

3 packages need a note:

- `math` takes 7 patches. The go command selects an assembly file for each
  architecture, thus 6 patches replace such a file and 1 replaces `unsafe.go`.
- `syscall/dart` is the interop package. The compiler replaces each call of it,
  and the standard library holds no such package.
- A stub overlay is an empty package that the fixed import list of a test binary
  drags in. `overlay.Paths()` leaves a stub out, thus stage 2 still rejects an
  import of it. Read decision 19.

## The network access

A Go program reaches TCP through `syscall/dart`. The Dart side uses `Socket` of
`dart:io`, in `runtime/goruntime/lib/src/net.dart`.

| Function | Blocks | Result |
|---|---|---|
| `DialTCP(host, port)` | Yes | A handle and a cause. |
| `ConnRead(handle, p)` | Yes | A count and a cause. A count of 0 with no cause is the end of the stream. |
| `ConnWrite(handle, p)` | Yes | A count and a cause. The call waits for the flush of the socket. |
| `ConnClose(handle)` | No | A cause. The call closes both directions. |
| `ListenTCP(host, port)` | Yes | A handle, the port that the system took, and a cause. |
| `Accept(handle)` | Yes | A handle of a connection and a cause. |
| `ListenClose(handle)` | No | A cause. Each connection of the listener stays open. |

`$await` is the wait primitive. It takes a future and a function for the cause
of a failure. It writes an `EventWait` frame, and it holds 1 wake source of the
scheduler. The value of the future reaches the caller through the resume of the
frame, exactly as the value of a channel operation does. Thus a caller of
`DialTCP` becomes a state machine of stage 4, and no `await` reaches the emitted
Dart.

A connection holds a `StreamIterator` of the socket and the rest of the last
packet. A Dart socket gives a stream of packets, and a Go read takes only the
count that the slice holds.

The 3 tests:

- `runtime/goruntime/test/net_test.dart` drives a hand written machine against a
  local echo server. It proves a dial, a write, a read, and a close. It also
  proves the cause of a dial to a closed port.
- `testdata/programs/netdial` proves that 1 Go source gives the same output with
  `go run` and with `gart`. The Go build reads `net.Dial`, and the Dart build
  reads `dart.DialTCP`. A dial of port 0 fails on both, thus the program needs
  no server.
- `testdata/programs/netecho` holds a server and a client in 1 program. It
  listens on a free port, it accepts on a goroutine, and it sends 4 bytes. Both
  builds write the same 5 lines. The Go build holds the same handle model as the
  interop package, thus the 2 builds share the whole program.

The `net` overlay holds `Dial`, `Listen`, `ListenPacket`, `Conn`, `PacketConn`,
`Listener`, `Addr`, `Error`, `Dialer`, `SplitHostPort`, and `JoinHostPort`. It
reads the 16 interop functions and needs no new compiler feature. `TCPAddr`,
`UDPAddr`, `UnixAddr`, and `IP` stay out, thus a type assertion of an `Addr`
gives a compile error and not a wrong value.

The work of 2026-08-30 added 9 interop functions:

| Function | Blocks | Result |
|---|---|---|
| `DialUnix(path)` | Yes | A handle of the table of the connections, and a cause. |
| `ListenUnix(path)` | Yes | A handle of a listener, and a cause. |
| `ConnDeadline(handle, read, write)` | No | A cause. The 2 values are absolute times of the Unix epoch, and 0 holds no limit. |
| `ListenUDP(host, port)` | Yes | A handle, the port that the system took, and a cause. |
| `DialUDP(host, port)` | Yes | A handle, the local port, and a cause. Such a socket drops a datagram of another source. |
| `PacketRead(handle, p)` | Yes | A count, the host of the source, the port of the source, and a cause. |
| `PacketWrite(handle, p, host, port)` | Yes | A count and a cause. |
| `PacketClose(handle)` | No | A cause. |
| `PacketDeadline(handle, read, write)` | No | A cause. |

A deadline reads the timeout of a Dart future. The source future stays open,
thus the next call of the same operation takes it again and loses no byte and no
datagram. A deadline that a read already waits on reaches the next read only.

`testdata/programs/netwide` proves each name against `go run`. It reads the real
`net` API with no build tag, thus 1 Go source runs in both builds.

`crypto/tls` gives a client. `$dialTLS` opens a plain socket, calls
`SecureSocket.secure` with the name of the server, and puts the result in the
table of the connections. Thus each read, each write, each deadline, and the
close of a TCP connection also work on a TLS handle.

Dart holds the whole handshake, thus the overlay honors `InsecureSkipVerify`,
`ServerName`, `NextProtos`, and the first `Certificate` of a listen only.
`MinVersion`, `MaxVersion`, and `CipherSuites` each stay a field of the
`Config`, and a dial or a listen with a value stops the program.

The server landed on 2026-09-01. `tls.Listen` rides `SecureServerSocket` of
Dart through the interop calls `ListenTLS` and `AcceptTLS`, and the accepted
connection joins the table of the plain connections. The surface holds no
`crypto/x509`, thus a `Certificate` keeps the 2 PEM blocks that `X509KeyPair`
took, and Dart parses them. `AcceptTLS` gives the protocol that ALPN took.
`httptest.NewTLSServer` and `StartTLS` ride `tls.Listen` with a self signed
certificate of the loopback address, and `testdata/programs/tlserve` proves
the parity against `go run`. Read decision 210. HTTP/2 landed over the Dart
package `http2` on 2026-09-01, and the section of the HTTP access holds it.

`runtime/goruntime/test/tls_test.dart` proves a handshake, ALPN, a write, and a
read against a local server. The certificate comes from
`net/http/internal/testcert` of the Go repository.

`testdata/programs/netpkg` reads the real `net` API with no build tag. The go
command takes the real package, and `gart` takes the overlay. Both builds write
the same 9 lines.

## The process access

`os/exec` is an overlay over `Process.start` of `dart:io`. The runtime holds
the child in a table by its process id, and 7 interop calls carry the start,
the 3 pipes, the close of the standard input, the wait, and the kill.
`os.Process` and `os.ProcessState` live in the `os` overlay, and the state
holds the exit code and the signal only. Decision 212 holds the shape, and
`docs/parity.md` holds the gaps. The program `execuse` proves the parity.

## The HTTP access

HTTP/2 rides the Dart package `http2`, behind ALPN. The runtime file
`http2.dart` wraps the socket of a TLS connection whose handshake took h2:
`$takeSocket` lifts the socket out of the table of `net.dart`, and 9 interop
calls carry the 2 sides. 1 session carries each request of a host on the
client, and the server answers each stream on a goroutine of its own.
`goruntime` thus holds 1 pub dependency, `http2 ^2.3.1`. Decision 213 holds
the shape, and the program `h2serve` proves the parity.

The real `net/http` holds 53000 lines and reads a vendored HTTP/2 stack, a
cookie jar, `mime`, `net/textproto`, and `compress/gzip`. The overlay holds an
HTTP/1.1 client and an HTTP/1.1 server over the `net` package, in 8 files and 1
directory:

- `http.go` holds each method, each status code, and the status text.
- `header.go` holds `Header` with the canonical form of a key.
- `request.go` holds `Request`, `NewRequest`, the write of a request, and the
  read of a request of a server.
- `response.go` holds `Response`, the read of a response, and the reader of a
  chunked body.
- `transport.go` holds `Transport`, `RoundTripper`, and the pool of the
  connections. The body of the answer holds its connection, and a Close gives
  the connection back to the pool.
- `client.go` holds `Client`, `Do`, and each redirect.
- `server.go` holds `Server`, `ServeMux`, `ResponseWriter`, and `Flusher`.
- `cookie.go` holds `Cookie`, `SetCookie`, and each read of a cookie.
- `httptest/` holds `ResponseRecorder`, `NewRequest`, and the `Server` of a
  test. `net/http` exports `ReadRequest` for it.

`docs/parity.md` holds a table of each gap. The largest one is HTTP/2. The
client reads gzip, and `net/http/httptest` holds a recorder and a server of a
test.

`testdata/programs/httpuse` runs a server and a client in 1 program. It proves a
GET, a POST, a form, a HEAD, and a redirect. It also proves a subtree pattern,
`StripPrefix`, a chunked answer through `Flush`, a body of 5000 bytes, and each
cookie. The output of `go run` and of the Dart build stays equal.

The work found 1 defect of the compiler. A composite literal of a struct of
another package writes the zero value of each field, and a field type belongs to
a third package. Such a package now takes a prefix at the point of the use, and
the library file writes its import.

## The regexp package

`regexp` and `regexp/syntax` are native, and no patch remains. The patch of
`regexp/syntax/parse.go` is gone, because the subset holds `recover` now, and
the real file turns a panic of `ErrLarge` and a panic of `ErrNestingDepth` into
an error through its own deferred call. Read decision 201.

The 4 addresses of a `[]rune` field, as in `cleanClass(&re.Rune)`, needed no
patch. The work of 2026-08-30 gave the subset a pointer to a field of a struct.

`gart test -run` and `gart test -bench` each read a regular expression through
the harness. The driver reads the same file for the skip line of an unsupported
function. Read decision 124.

## The context package

`context` reads the real Go source. It needed 2 parts, and the work of
2026-08-29 gave both.

1. `internal/reflectlite`. The overlay gives `TypeOf`, `Type.String`, and
   `Type.Comparable`. The type descriptor of the runtime holds a comparable
   flag, and the new intrinsic `dart.TypeInfo` reads the text and the flag.
2. `time.Time.String`. The `time` overlay now holds a calendar. The conversion
   from a count of nanoseconds to a year, a month, and a day follows the
   algorithm of Howard Hinnant. The new intrinsic `dart.ZoneOffset` gives the
   offset and the name of the local zone.

`defer cancel()` also needed the defer that can block. Read the section "A
deferred call that can block" of `docs/design.md`.

`testdata/programs/ctxuse` proves `WithCancel`, `WithTimeout`, `WithDeadline`,
`WithValue`, `WithCancelCause`, `Cause`, a nested value, and a deferred cancel.

The work found 3 defects. Read the defect table.

## The calendar of the time overlay

M16 gave `Time.String` a calendar of 1 direction: a count of nanoseconds to a
date. M18 gave the other direction and the layout scanner of Go.

- `Location` holds a name and a flag. `UTC` and `Local` are the 2 locations, and
  the overlay holds no zone database. `Local` reads `dart.ZoneOffset` at each
  instant.
- `Date` reads `daysFromCivil`, the reverse of the algorithm of Howard Hinnant.
  A value outside its range moves into the next unit, exactly as in Go. `Date`
  with `Local` reads the offset of the system 2 times, because the argument is a
  wall clock value and not an instant.
- `Format` holds `nextStdChunk` of Go, byte for byte, and the 19 layout
  constants. `Time.String` now calls `Format` with the layout of Go, thus the 2
  hold 1 rule.
- `Month` and `Weekday` hold the names of Go and the `%!Month(13)` form of an
  out of range value.

`testdata/programs/timecal` proves 7 dates against 25 layouts, and each output
of `go run` and of the Dart build stays equal.

`Time.Parse`, `ParseInLocation`, and the accessors `Year`, `Month`, `Day`,
`Hour`, `Minute`, `Second`, `Weekday`, and `YearDay` are present.
`docs/parity.md` holds the whole table.

The zone database landed on 2026-09-01. `LoadLocation` reads the variable
ZONEINFO, then each zone directory of the system, and then the database that
`time/tzdata` embeds. `zoneinfo_read.go` of the overlay holds the TZif parser
and the zip reader of Go, over `dart.ReadFile`. `zoneinfo.go` holds the lookup
and the tzset chain of Go. The overlay of `time/tzdata` holds the generated
`zzipdata.go` of the Go tree and calls `time.RegisterEmbeddedTZData`, because
the subset holds no linkname directive. The programs `tzuse` and `tzdatause`
and the test `TestZoneinfoZip` prove the parity. Read decision 205.

## The read and the write of reflect

The `reflect` overlay held a description of a type only. M18 gave it a read and
a write through a pointer, because the tests of `errors` of the Go repository
hold that pattern:

```go
rtarget := reflect.ValueOf(tc.target)
rtarget.Elem().Set(reflect.Zero(reflect.TypeOf(tc.target).Elem()))
```

`reflect.Type` now holds a handle of a type descriptor of the runtime, and not a
text. Thus `Type.Elem` names a type that no value holds. 6 intrinsics reach the
runtime: `DescOf`, `DescInfo`, `DescElem`, `DescZero`, `PointerElem`, and
`PointerSet`.

A `Value` that `Elem` read keeps the pointer, thus `Set` writes through it. A
pointer to a value that is not a struct holds a box. A pointer to an interface
holds a box of an interface value. A pointer to a struct is the object itself.
The write of a struct copies each field, and a field of a struct type keeps the
reference of the source.

Each predeclared descriptor of the runtime now holds a zero factory. A tear-off
of a top level function is a constant expression of Dart, thus each descriptor
stays `const`.

## The 2 forms of math/rand

Each form reads the real Go source, thus a seeded sequence gives the values of
Go. `testdata/programs/randuse` proves that against `go run`.

| Package | Patch | Cause |
|---|---|---|
| `internal/chacha8rand` | `chacha8.go` | The real file declares `block` with no body, and an assembly file holds that body. It also reads `internal/cpu` through `unsafe` for 2 offsets of the assembly. |
| `internal/chacha8rand` | `chacha8_generic.go` | The real file reads a `[16][4]uint32` and a `[32]uint64` as 2 views of 1 memory block. The patch writes each 32-bit word and packs 2 words into each 64-bit word. |
| `math/rand` | `rand.go` | `internal/godebug`, `runtime.rand`, the address of a field in `Read` and `Seed`, and the mutex of `lockedSource`. |
| `math/rand/v2` | `rand.go` | `runtime.rand` only. |

The algorithm of each form stays in a file that the compiler reads whole.
`rng.go` of the first form holds the lagged Fibonacci generator and its table of
607 values. `pcg.go` and `chacha8.go` of the second form hold the 2 generators.

The top-level functions of each form read a source that the clock seeds. Go
seeds its own source with a random value of the operating system, and Go 1.24
made the top-level `Seed` a no-op. Thus no program can depend on the values of
either one.

`N` and `Rand.N` of the second form stay out. Read item 1 of the work that stays
open.

## The reflect call and testing/quick

`testing/quick` builds a value of each type through `reflect` and calls the
function of a property through `reflect.Value.Call`. The work gave:

- A function type holds a description of its parameters and of its results.
  `TypeDesc` holds `params` and `results`, and each one is a list of thunks.
- `$callValue` reads `Function.apply` of Dart. A call that gives many results
  reads the record through `.$1` to `.$8`, because Dart holds no index of a
  record. A function that gives up control stays out, because the call reads no
  scheduler.
- `reflect.MakeMap`, `Value.SetMapIndex`, `Value.SetZero`, `Type.Key`,
  `Type.NumIn`, `In`, `NumOut`, `Out`, and `reflect.TypeAssert`.
- A `reflect.Value` that names a location now reads through that location. A Set
  writes through it, thus a read after a Set gives the new value. The value of
  the struct held a copy before, and each read after a write gave the old value.
- A type assertion to a type parameter reads the kind of the type argument. An
  interface type argument needs the test of the method set, and the test read
  the name of the type before. The description of an interface type now holds
  the key of each method that the interface needs.

The overlay of `testing/quick` is the source of Go with no complex case.
`testdata/programs/quickuse` proves `Check`, `CheckEqual`, `Value` of 8 kinds,
and a call through `reflect`.

## The tests of the Go repository

The user asked on 2026-08-29 whether the compiler can run the tests of the Go
repository, exactly as GopherJS does. The answer is yes for a package whose test
files stay inside the subset. 17 packages now run their real tests, and `make
ci` proves each one through `TestTestsOfGoroot`.

```bash
make std-test STD=path            # 8 PASS, 1 SKIP, and 8 examples skipped
make std-test STD=container/list  # each test passes
make std-test STD=container/heap  # each test passes
make std-test STD=container/ring  # each test passes
make std-test STD=unicode         # each test passes
make std-test STD=encoding/hex    # each test passes
make std-test STD=encoding/ascii85  # each test passes
make std-test STD=encoding/csv    # each test passes
make std-test STD=hash/crc32      # each test passes
make std-test STD=hash/fnv        # each test passes
make std-test STD=hash/adler32    # each test passes
make std-test STD=maps            # each test passes
make std-test STD=regexp/syntax   # 11 of 12 pass. Read the note below.
make std-test STD=io              # each test passes
make std-test STD=internal/chacha8rand  # each test passes
make std-test STD=compress/zlib   # each test passes
make std-test STD=unicode/utf8    # each test passes
```

`TestParseInvalidRegexps` of `regexp/syntax` reads a limit of a resource, and
the recovered panic of the real `parse.go` serves it. `TestTestsOfGoroot` runs
each test of that package.

The work gave:

- `testing.AllocsPerRun`, `testing.B.Loop`, `testing.Benchmark`, and
  `strconv.AppendInt`.
- A `runtime` overlay with `GOMAXPROCS`, `NumCPU`, `NumGoroutine`, `GOOS`, and
  `GOARCH`.
- A mode for a package of GOROOT. Such a package belongs to no module, thus
  `moduleOf` gives it the module `std`. The test variants of it reach the build
  through the 2 reserved suffixes `_test` and `.test`.
- `TestTestsOfGoroot` in `internal/driver` runs the 3 packages from the Go
  installation of the machine. A new Go release can add a test file outside the
  subset, and the test then fails.

A survey of the native packages on 2026-08-29, after each part of the work:

| Package | State |
|---|---|
| `path` | Runs. 8 PASS, and `TestCleanMallocs` skips for `AllocsPerRun`. |
| `container/list` | Runs. Each test passes. |
| `unicode` | Runs. Each test passes. |
| `cmp` | 1 rejection: `cmp_test.go` imports `unsafe`. |
| `math` | 1 rejection: `all_test.go` imports `unsafe`. |
| `math/bits` | 6 rejections: the test reads `runtime.Error` after a recover. |
| `bufio` | `net.UnixAddr`, `net.ListenUnix`, and `net.DialUnix`. |
| `context` | Not measured. |
| `encoding/base64` | `runtime/debug`, the address of a byte of a slice, and `recover`. |
| `errors` | `time.Date`, `time.UTC`, `reflect.ValueOf`, `reflect.Zero`, and `Type.Elem`. |
| `io` | Runs. Each test passes. |
| `bytes` | 22 rejections in 5 groups. Read below. |

`bytes` does not reach a run. The 22 rejections hold 5 groups, and 4 groups are
non-goals of the subset:

| Count | Rejection | State |
|---|---|---|
| 7 | `recover` | Non-goal N2 |
| 4 | The address of a byte of a slice | Non-goal N3 |
| 1 | The import of `unsafe` | Non-goal N2 |
| 4 | An exported variable of an unnamed struct type | A rule of stage 2 |
| 6 | `internal/testenv`, `math/rand`, `internal/asan`, `encoding/base64` | A bigger surface |

`encoding/base64` reached the surface, thus the last group holds 5 names now.
The rule of the exported unnamed struct type also relaxed: M15 rejects only a
shape with an unexported field.

A second survey on 2026-08-30, after the work of that day:

| Package | Cause that blocks the run |
|---|---|
| `errors` | 1 call of `recover` in `wrap_test.go`, and the package `io/fs`. |
| `io` | Runs. Each test passes. `runtime.GC` closed the last rejection. |
| `math/rand/v2` | `rand.N`, and 9 names of `reflect` that the surface does not hold: `NumMethod`, `Value.Type`, `Value.Method`, and each Kind constant. |

A third survey on 2026-08-31, after the work of that day:

| Package | Cause that blocks the run |
|---|---|
| `errors` | 1 call of `recover` in `wrap_test.go`. `io/fs` reached the surface. |
| `math/rand/v2` | `Type.NumMethod`, `Type.Method`, `Value.Method`, and `os.Pipe` in `regress_test.go`. The Kind constants reached the surface. |
| `bytes` | `internal/testenv`, `internal/asan`, and 7 calls of `recover`. |

The overlay of `internal/testenv` gives 2 functions. `Builder` gives an empty
name, and `SkipIfOptimizationOff` does nothing. The real package starts a
process of the go command, and gart holds no such access. Read decision 129.

A fourth survey on 2026-08-31, over each native package that no run covers:

| Package | Cause that blocks the run |
|---|---|
| `internal/chacha8rand` | Runs. Each test passes. It needed no work. |
| `unicode/utf8` | Runs. Each test passes. It needed the strict rune decoder of decision 131. |
| `compress/zlib` | Runs. Each test passes. It needed `internal/testenv`, the working directory of the package, and the fix of decision 126. |
| `compress/flate` | `strings.FieldsSeq` in `example_test.go`. Read the note below. |
| `compress/gzip` | `testenv.MustHaveSource`, `filepath.EvalSymlinks`, and `runtime.GOROOT`. |
| `math/rand` | `testenv.Command`, `testenv.Executable`, `testenv.CleanCmdEnv`, `Value.NumMethod`, `Type.Method`, and `Value.Method`. |
| `math/rand/v2` | `os.Pipe`, `Type.NumMethod`, `Type.Method`, and `Value.Method`. |
| `bytes` | 7 calls of `recover`, `internal/asan`, and `unsafe`. |
| `bufio` | `net.UnixAddr`. |
| `context` | `sync.Cond`. |
| `net/netip` | `net.ParseIP`. |
| `net/url` | `encoding/gob`, `internal/diff`, and 1 call of `recover`. |
| `regexp` | `compress/bzip2`. |
| `hash` | `crypto/sha512`. |
| `encoding/base64` | `runtime/debug`. |
| `math/bits` | `runtime.Error` after a `recover`. |
| `cmp`, `math` | `unsafe`. |

`strings.FieldsSeq` is the only gap of `compress/flate`. The overlay of
`strings` must import `iter` for it, and each program that imports `strings`
then carries the `iter` package. A measurement of 2026-08-31 gave 17479 more
bytes of Dart source for `testdata/programs/args`, which is 4 percent. No other
package changed its form. The work waits for the value flow of stage 3, because
`iter.Pull` in a program marks each `iter.Seq` closure as a function that can
block.

A fifth survey on 2026-08-31, over each overlay package. Such a package holds a
subset of the real API, thus a test file of the Go repository names a function
that the overlay does not hold. `unicode/utf8` is the only one that runs.

| Package | Cause that blocks the run |
|---|---|
| `unicode/utf8` | Runs. Each test passes. |
| `slices` | `internal/asan`, `internal/msan`, `internal/race`, `unsafe`, and 3 calls of `recover`. |
| `sort` | `SearchStrings`, `SearchFloat64s`, `SliceIsSorted`, `Float64sAreSorted`, `Reverse`, and `IntSlice`. |
| `strconv` | `IsGraphic`, `IsPrint`, `QuoteRune`, `QuoteRuneToASCII`, and `QuoteRuneToGraphic`. |
| `strings` | `Builder.Cap`. |
| `errors` | 1 call of `recover`. |
| `iter` | `runtime.Gosched` and `runtime.Goexit`. |
| `sync` | `sync.NewCond`. |
| `sync/atomic` | `CompareAndSwapUint32` and each other named function of the package. |
| `time` | `absDays`, an unexported name of the real source. |
| `unique` | `canonMap`, an unexported name of the real source. |
| `fmt` | `Fscanf` and each other scan function. |
| `flag` | `FlagSet.Duration`. |
| `os` | `Clearenv`. |
| `io/fs` | `Glob` and `ReadFile`. |
| `path/filepath` | `lstat`, an unexported name of the real source. |
| `reflect`, `internal/reflectlite` | A large part of the real API. |
| `runtime` | `SetFinalizer` and each other name of the real runtime. |
| `testing` | `B.Context`. |
| `testing/quick` | `complex64`. |
| `log` | `testenv.MustHaveExec`. |
| `internal/godebug` | `testenv.MustHaveGoRun`. |
| `crypto/md5`, `crypto/sha256` | `crypto/internal/cryptotest`. |
| `crypto/sha1` | `crypto/internal/boring`. |
| `crypto/tls` | `SignatureScheme`. |
| `encoding/binary` | `reflect.Value.Slice`. |
| `encoding/json` | `NewEncoder`. |
| `net` | `ParseIP`. |
| `net/http` | `Request.TLS`. |
| `net/http/httptest` | `Server.EnableHTTP2`. |

The work that remains, in the order of the value:

1. `Type.NumMethod`, `Type.Method`, `Value.Method`, and `os.Pipe`, for the tests
   of `math/rand` and of `math/rand/v2`.
2. `filepath.EvalSymlinks`, `runtime.GOROOT`, and `testenv.MustHaveSource`, for
   the tests of `compress/gzip`.
3. A test file that reads `recover` never runs. Such a file needs a fork of the
   test source, and that ends the value of a parity measurement.

## The tests of the third session of 2026-08-31

The third session closed item 1 and item 2 of the list above. The real tests
of `math/rand/v2`, of `compress/gzip`, and of `compress/flate` run, thus
`TestTestsOfGoroot` covers 20 packages. The road to them found 5 defects of
the compiler, and each one holds a decision: 140 to 142 and the earlier 137
and 138. The surface gained the go/format subtree from the real Go source,
with 4 patches that remove a deferred recover. Read decision 148.

## The test directory of the Go repository

`TestRunFilesOfGoTest` of `internal/driver/gotest_test.go` runs each file of
`$GOROOT/test` whose header is `// run`: the 145 files of the top level, the
141 of typeparam, and the 645 of fixedbugs. Each run compares with `go run`
under a cap of 2 minutes. A file outside the subset skips with the rejection
of stage 2.

The state of 2026-09-01, after the sixth session: 682 files match, and no
run differs. The complex numbers and the source map work of that day
unlocked 13 files. 217 files stand outside the subset, most through
`unsafe`, and the parts of `runtime` and `reflect` that the surface omits.
`chanlinear` measures time, and it can fail 1 time under a heavy parallel
load. `goTestSkips` names the 32 runs that the net holds back, each with
its cause. The groups:

| Group | Files | Cause |
|---|---|---|
| The fixed Dart stack | `stack`, `stackobj2`, `fixedbugs/issue75764` | The deep recursion overflows. A parity limit. |
| The self type | `peano`, `fixedbugs/issue4316` | A deref of `type Number *Number` has no Dart form. Decision 149 stops the crash. |
| The reflect gaps | `reflectmethod4` | A bound method of reflect is no plain Dart function. |
| The line directives | `fixedbugs/issue22662`, `issue5856` | A directive with no file gives `??` in Go, and the panic path of Go holds a gopanic frame. `issue14646` and `issue18149` joined the comparison on 2026-09-01. |
| The machine addresses | `fixedbugs/bug260`, `bug352`, `issue29735`, `issue4667` | Dart gives no machine address, no program counter, and no allocation count. |
| The deep types | `fixedbugs/issue72063`, `issue29312`, `typeparam/issue48137` | A type that nests 30 levels or more erases at 2 depths. |
| The local generic identity | `typeparam/nested` | The reflect identity of a local type of a generic function drops the outer type argument. |
| The run caps | `fixedbugs/issue13169`, `issue34395`, `issue78081`, `issue19182` | The run needs more than the cap, or a busy loop never yields. |
| The recover corners | `convert4`, `issue26094`, `issue4562`, `issue19040`, `issue48898`, `issue23734`, `issue29504`, `issue34123`, `devirtualization_nil_panics` | The text of a panic, the address of a view element, the frames of a recovered stack, and the identity of a map key each differ in a corner. |
| The memory of Dart | `fixedbugs/issue29190`, `issue7550` | The run allocates past the memory of the Dart VM. |

The road of the fifth session stands in decisions 200 to 202.


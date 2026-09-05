# The standard library

This file holds the work of the standard library surface. It names the 3 forms
that reach a package, and the design of each package that needed one. Read it
before a change of an overlay or before a new package of the surface.

`docs/what-compiles.md` holds the list of each package of the surface.
`docs/parity.md` holds each difference between Go and the Dart output.

## Standard library

The surface holds 253 import paths: 164 standard packages, `syscall/dart`, 76
internal packages, and 12 vendored packages of `golang.org/x`. The run of
`make surface` of 2026-09-04 gives the count. `docs/what-compiles.md` holds the whole list and the gate of
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

`make surface` measures how much of each package the surface holds.
`internal/surface` holds the tool.

5 packages need a note:

- `math` takes 7 patches. The go command selects an assembly file for each
  architecture, thus 6 patches replace such a file and 1 replaces `unsafe.go`.
- `syscall/dart` is the interop package. The compiler replaces each call of it,
  and the standard library holds no such package.
- `syscall` holds the `Signal` type and the 12 signal numbers that macOS and
  Linux agree on. `os` and `os/signal` need them, and the surface holds no
  system call interface. Read `docs/parity.md`.
- `os` cannot read `path/filepath`, because that package reads `os`. The
  overlay holds its own `localize`, exactly as the real `os` reads
  `internal/filepathlite`.
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
connection joins the table of the plain connections. A `Certificate` keeps the
2 PEM blocks that `X509KeyPair` took, because Dart parses them, and since
2026-09-03 it also holds the DER chain and the parsed leaf. `AcceptTLS` gives
the protocol that ALPN took.
`httptest.NewTLSServer` and `StartTLS` ride `tls.Listen` with the test pair of
`net/http/internal/testcert`, and `testdata/programs/tlserve` proves
the parity against `go run`. Read decision 210. HTTP/2 landed over the Dart
package `http2` on 2026-09-01, and the section of the HTTP access holds it.

`runtime/goruntime/test/tls_test.dart` proves a handshake, ALPN, a write, and a
read against a local server. The certificate comes from
`net/http/internal/testcert` of the Go repository.

`testdata/programs/netpkg` reads the real `net` API with no build tag. The go
command takes the real package, and `gart` takes the overlay. Both builds write
the same 9 lines.

## The resolver of the DNS

The Lookup family of `net` reads a resolver of the overlay, and no call of the
platform. `InternetAddress.lookup` of `dart:io` gives an A record and an AAAA
record alone, thus `LookupMX`, `LookupNS`, `LookupSRV`, `LookupTXT`, and
`LookupCNAME` would each stay out. The socket of the overlay already carries
UDP and TCP, thus the resolver needs no interop function of its own. Decision
233.

| File | Content |
|---|---|
| `internal/overlay/std/net/dnsmsg.go` | The message format. 8 record types: A, AAAA, CNAME, NS, PTR, MX, TXT, and SRV. |
| `internal/overlay/std/net/lookup.go` | The `Resolver`, each `Lookup` function, `DNSError`, `MX`, `NS`, and `SRV`. |

The resolver reads `/etc/hosts` first. It then asks each server of
`/etc/resolv.conf` over UDP, and a truncated answer takes a second exchange
over TCP. `LookupPort` reads `/etc/services`, and `parsePort` of Go holds the
rule for a number. Each of the 3 files reads through `dart.ReadFile`, thus the
overlay imports no `os` and holds no cycle.

`Resolver.Dial` is the seam of a test.
`testdata/programs/dnslookup` runs a DNS server of its own on the loopback
interface, thus the program reads no network. It covers each function of the
family, the fallback to TCP, and the 4 error shapes.
`docs/parity.md` holds the 5 differences against Go.

## The embed package

A `//go:embed` directive takes its value at the declaration, and stage 1 writes
it. The compiler of Go holds a directive of the linker for the file list of an
`embed.FS`, and this compiler holds none. Decision 232.

`internal/load/embed.go` reads each directive of the module on the first of the
2 loads. It resolves the patterns against the directory of the package, and it
writes the files into the source of the declaration. The second load reads the
output through the same overlay map that carries the standard library.

An `init` of the generated code was the other candidate, and it breaks the
initialization order. A package level variable that reads an embedded string
runs before each `init`, thus `var count = len(greeting)` would give 0.

The go command rejects a directive on a variable that holds a value, thus the
same rewrite turns `//go:embed` into `// go:embed`. A comment with a space is
no directive.

`internal/overlay/std/embed/embed.go` is the source of Go with 3 changes:
`strings` in place of `internal/bytealg` and `internal/stringslite`, no `hash`
field, and the constructor `New`. A package cannot reach an unexported field of
another package, thus the generated source names `New`. The real package of Go
holds no such name, thus stage 2 rejects a read of it in a file that holds no
directive, and `verify.Package` takes the set of those files.

`New` builds 1 entry for each directory of a name, and it sorts the list by the
directory and then by the element. Thus the generated source holds the leaf
files alone.

## The constant time package

`crypto/subtle` needs an overlay. `boolToUint8` of
`crypto/internal/constanttime` panics, and the compiler of Go replaces it with
an instruction of the machine. `dit.go` of Go reads `unsafe` and
`internal/runtime/sys`. The overlay holds the same 8 names, and each arithmetic
body takes its portable form. `docs/parity.md` holds the 2 differences, and
`testdata/programs/subtleuse` covers each name.

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
- `fetch.go` holds the client of the web target. A browser holds no socket,
  thus `RoundTrip` takes the fetch API of the host when `runtime.GOOS` gives
  `js`. Read decision 252.
- `httptest/` holds `ResponseRecorder`, `NewRequest`, and the `Server` of a
  test. `net/http` exports `ReadRequest` for it.

`docs/parity.md` holds a table of each gap. The largest one is HTTP/2. The
client reads gzip, and `net/http/httptest` holds a recorder and a server of a
test.

`testdata/web/webfetch` runs the client of the web target against a real
server. `TestWebFetch` of `internal/driver` builds it 3 times: with the go
command, with `dart compile js`, and with `dart compile wasm`. The 3 outputs
are equal. The body of an answer rides the `ReadableStream` of the fetch API,
thus a program reads 100000 bytes through a buffer of 512 bytes and holds no
whole body.

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

## The patched packages of 2026-09-03

`archive/tar`, `math/big`, and `encoding/asn1` each read the real Go source,
and a patch replaces the files that reach outside the surface.

**`archive/tar`** takes 3 patch files, and each 1 is empty. `stat_unix.go`
reads `syscall.Stat_t` and `os/user`, and `stat_actime1.go` and
`stat_actime2.go` each read `syscall.Stat_t`. The variable `sysStat` of
`common.go` then stays nil, and `FileInfoHeader` already tests it for nil. A
header thus carries no owner name and no group name, exactly as Go behaves on
a platform with no such file.

**`math/big`** takes 5 patch files. `arith_decl.go` declares 6 functions with
no body, because an assembly file of each architecture holds the code. Go ships
the pure Go form of the same 6 behind the tag `math_big_pure_go`, and `arith.go`
holds each `_g` function that the form calls. The patch writes those 6 calls.
`arith_amd64.go` reads `internal/cpu` for the ADX instruction, and its patch
gives the flag the value false.

The last 3 files landed later on 2026-09-03, and each 1 serves the block model
and not the compile. `intconv.go` and `ftoa.go` wrote the message of an unknown
verb with `fmt.Fprintf` and `fmt.Fprint`. The 2 take an `io.Writer`, thus the
flow of stage 3 gave the call the group of each `Write` method of the program,
and that group holds a `Write` that blocks. The 2 files now write through the
`fmt.State` that the caller gives. `natconv.go` guarded the divisor cache of
base 10 with a `sync.Mutex`, and `Mutex.Lock` of the overlay waits on a
channel. The critical section holds `nat` arithmetic alone, thus no goroutine
of the 1 isolate reaches the cache while another builds it. Read decision 250
and the section "The method group of an interface call" of `docs/parity.md`.

**`encoding/asn1`** takes 2 patch files. `asn1.go` drops `internal/saferio`,
which holds `unsafe.Sizeof`, and a local function holds the body of
`SliceCapWithSize`. The same file also reads the type of a field in place of a
type assertion of its address, because `Value.Addr().Interface()` of the
reflect subset gives a plain reference and no typed pointer. `marshal.go`
writes a composite literal in place of a conversion between 2 struct types.

The result needs no new interop and no Dart arithmetic. A Dart form of the same
word operations would be a second implementation to keep correct, and the Go
form is correct already. `testdata/programs/biguse` proves an `Int`, a `Float`
of 200 bits, and a `Rat`, and `testdata/programs/bigfmt` reads each line that
the 3 later patches touch.

## The packages of the fifteenth session

11 import paths entered the surface on 2026-09-03, after `database/sql`.

**`net/mail`, `net/smtp`, `net/rpc`, and `net/rpc/jsonrpc`** each read the real
Go source. `net/mail` needs no patch. `net/smtp` takes a patch of `StartTLS`,
which reports a cause and sends no command, because the overlay of `crypto/tls`
raises no open connection. `net/rpc` takes a patch of 1 line, because
`reflect.Method.Func` stays out. `net.Pipe` and `net.OpError` entered with
them, and `pipe.go` of the real Go source needed no change.

**`net/http/cgi`** reads the real Go source, and it needed 3 fields of
`http.Request` and 5 vendored packages:
`vendor/golang.org/x/net/http/httpguts`, `vendor/golang.org/x/net/idna`, and
the 3 `x/text` packages that idna reads. Each vendored package compiles with no
patch. Read decision 304 for the 3 fields.

**A vendored package takes its full path.** The list of `native.go` names
`vendor/golang.org/x/net/idna`, and not `golang.org/x/net/idna`. The loader
reads the path that the go command reports, and the go command rewrites an
import of the standard library to the vendored form.

**The 4 `go` packages** stand in the same list. Read the report of that work
below.

## The database access

`database/sql`, `database/sql/driver`, and `uuid` each read the real Go source.
1 patch file of 1 line serves the 3.

**The pool needs no second thread.** The backlog held the opposite reading, and
the work of 2026-09-03 disproved it. `database/sql` is pure Go, and each
package that it reads already stood in the surface. The pool runs a goroutine
and a mutex, and 1 isolate serves the 2. A second thread is a property of a
driver.

**The patch.** `driver.go` writes `type ScanContext = internal.ScanContext` in
place of a named type of its own. Go declares the 2 as separate types and
converts between them at 2 places. The Dart class of `internal.ScanContext`
holds its unexported field as a private member, and no other Dart library reads
such a member. Read decision 289 for the price: `%T` gives
`internal.ScanContext`.

**`uuid` is a standard package of Go 1.27.** `go list -f {{.Standard}} uuid`
reports true, and `database/sql/driver` imports it. It reads `bytes`, `cmp`,
`crypto/rand`, `encoding/binary`, `encoding/hex`, `errors`, `sync`, and `time`,
and it needs no patch.

**No driver ships with the surface.** `testdata/programs/sqlmem` holds a driver
in memory, and it runs an open, a query, a scan, a prepared statement, a
transaction, a context that a cancel stops, a deadline, and the pool under
load. A real driver of a network database is Go code like any other, thus a
program can write 1 over `net`.

**The tests of the Go repository do not run.** `convert_test.go` reads
`internal/asan` and `sql_test.go` reads `internal/race`, and the surface holds
neither package. `driver/types_test.go` holds a conversion between 2 named
struct types. Read the limits of `docs/parity.md`.

## The value surface of crypto/tls

`dart:io` owns the TLS handshake, thus the overlay is a thin cover over a host
connection. The package holds 115 of the 263 names of Go, and the split follows
1 rule: a name that carries a value enters, and a name that would change the
handshake stays out.

`values.go` holds each cipher suite constant, `VersionSSL30`, `VersionName`,
and the 5 enumerations `CurveID`, `SignatureScheme`, `ClientAuthType`,
`RenegotiationSupport`, and `QUICEncryptionLevel` with the `String` method of
each 1. It also holds the struct `CipherSuite` and the 3 functions
`CipherSuites`, `InsecureCipherSuites`, and `CipherSuiteName`.

Each `String` method is a switch, and not a generated table. Each range arm of
`common_string.go` of Go covers defined constants alone, thus a switch gives
the same text for each input, the fallback included. The package holds no
`fmt`, thus a local `hex4` writes the `%04X` of a fallback.

A `Config` field that would change the handshake stays out, because such a
field can only report support and change nothing.
`Config.PreferServerCipherSuites` is the 1 field that enters: Go documents it
as ignored since Go 1.18, thus an inert field reproduces Go exactly.

`testdata/programs/tlsvalues` prints each value, each `String` result, the 2
cipher suite lists in full, and each fallback branch.

## The permission and the owner of a file

`dart:io` names no permission bit, no owner, and no hard link. `os` reaches the
3 through a command of the platform, and `attr.go` of the overlay holds them.

| Name | The form |
|---|---|
| `Chmod` | `Process.runSync('chmod', ...)` |
| `Chown` and `Lchown` | `Process.runSync('chown', ...)` |
| `Link` | `Process.runSync('ln', ...)`. The `Link` of `dart:io` is a symbolic link. |
| `Chtimes` | `setLastAccessedSync` and `setLastModifiedSync` of `dart:io`. The precision is 1 millisecond. |
| `Truncate` | `truncateSync` of a `RandomAccessFile`. |

`Stat` still reports a fixed mode, because `dart:io` gives no permission bit
back. `docs/parity.md` holds that row and the 5 that stand with it.

A name that needs a file descriptor stays out. The runtime names a file by a
`RandomAccessFile` object alone, thus `File.Fd`, `File.SyscallConn`, `NewFile`,
`ProcAttr.Files`, and `Cmd.ExtraFiles` reach no form. A name that needs a wait
status or a resource usage stays out for the same kind of cause:
`process.dart` reads `Process.exitCode` and nothing else.

## The connection tracking of httptest

`Server.CloseClientConnections` needs the server to hold each accepted
connection. The overlay of `net/http` carries no `ConnState` hook, and a change
of it would reach each caller of the server.

`httptest` wraps the listener and each connection itself. A `sync.Once` in the
wrapper makes the close idempotent, thus a close from a handler and a close
from `CloseClientConnections` cannot close 1 Dart socket handle 2 times.

The wrapper keys a connection by an `int`, and not by the connection. The
runtime hashes a map key by its structure, and a `*tls.Conn` changes its byte
counters while it stands in the map. A key of the connection thus reached no
end in `Iface.hashCode`.

`NewTestServer` binds 2 loopback listeners inside `Client()` and gives the
client a transport that rewrites the host of the URL alone. Go runs an
in-memory network there, and the surface holds no `net.Pipe` and no dial hook.
The contract that a caller reads holds: `URL` is `http://example.com`,
`Listener` stays nil, and a request of any host reaches the handler with the
host of the caller.

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

## The packages of the eighteenth session

12 packages joined the surface on 2026-09-04, in the session after the 7
phases. Each 1 holds every exported name of Go.

| Package | Form | The design |
|---|---|---|
| `crypto/mlkem`, `crypto/mlkem/mlkemtest` | Native | The real Go source. The emitter needed a composite literal whose type is a type parameter. Read decision 338. |
| `crypto/mldsa` | Native | The real Go source, with no change. |
| `crypto/hpke` | Native | The real Go source. 2 vendored packages of `golang.org/x/crypto` took a patch: `internal/alias` reads `unsafe`, and `chacha20` declares a function of assembly with no body. |
| `testing/cryptotest` | Patch | `rand.go` reaches 2 names with a `go:linkname` directive, and the import rule of Go keeps each 1 apart. The new package `internal/linkhook` carries the 2. Read decision 339. |
| `debug/gosym` | Native | The new overlay of `internal/abi` holds the 5 names of the PC/line table alone. Each other name of that package reads `unsafe`. |
| `weak` | Overlay | `Pointer[T]` holds a `WeakReference` of `dart:core`. The runtime keeps 1 reference for each value in an `Expando`, thus 2 weak pointers of 1 value compare equal. Read decision 340. |
| `net/http/httptrace` | Native | The overlay of `net/http` calls 14 hooks of 16. `internal/nettrace` joined the surface with it, and `net` carries it exactly as the real `net` does. |
| `net/http/httputil` | Native | The real Go source, with no change. |
| `encoding/json/jsontext` | Native | The real Go source. The 4 internal packages of `encoding/json` joined with it. It needed a type parameter over a byte slice and a string. Read decision 344. |
| `encoding/json` | Native | The real Go source, and each of its 92 exported names. The hand-written overlay of 1850 lines is gone. It reads `reflect` and `encoding/json/v2`, thus the Dart source of a program that reads it grows. Read decision 350. |
| `encoding/json/v2` | Native | The real Go source. It needed 2 rules of the emitter: a defined type over an instance of a generic type, and a wrapper for a function value of a narrower Dart parameter type. Read decisions 347 and 348. |
| `image/png` | Native | The real Go source. `writer.go` converts `*encoder` to `*EncoderBuffer` and back, and decision 341 gives the 2 names 1 Dart class. |

`internal/linkhook` is a package of gart, and Go holds no such path. It carries
a name of 1 standard library package to another 1 that the import rule of Go
keeps apart. Go reaches such a name with a `go:linkname` directive, and the
subset holds no such directive. The package that owns a name writes its field
at the start of the program.

`internal/abi` is a package of Go, and this overlay holds 5 of its names. Each
other name reads `unsafe` or a value of the runtime of Go, and no package of
the surface reads one.

## The bubble of testing/synctest

`testing/synctest` runs a function in a bubble. The bubble holds a clock of its
own, and that clock moves only when each goroutine of the bubble is durably
blocked. A test of a timeout thus takes no real time. Go builds the bubble in
its runtime, and gart builds it in the scheduler of the Dart runtime library.

`runtime/goruntime/lib/src/sched.dart` holds the class `Bubble`. It carries the
fake clock, the set of the goroutines of the bubble, the goroutine that waits
inside `Wait`, and the goroutine that started the bubble. A timer of a
goroutine of a bubble becomes a `_FakeTimer` of that bubble in place of a
`Timer` of Dart. Such a timer waits for the scheduler and not for the event
loop.

The scheduler runs a step of each bubble when the run queue empties. A step
first reads whether each goroutine of the bubble is durably blocked, and the
frame of the wait holds that answer. A channel of the bubble, a select over
such channels, a sleep of the bubble, a nil channel, and a call of `Wait` each
mark the wait durable. A wait for the IO of Dart does not. `sync.Cond.Wait`,
`sync.WaitGroup.Wait`, and `sync.Mutex.Lock` each wait on a channel that the
caller makes, thus the same rule reaches them.

An idle bubble takes 1 of 3 paths, in the order of Go. A timer that the clock
reached runs first. A call of `Wait` ends next. The clock then moves to the
next timer, and it stops after the first goroutine of the bubble ends. A bubble
that reaches none of the 3 ends, and `Test` panics with the text of Go.

`syscall/dart` carries the 3 new interop functions `InBubble`, `BubbleRun`, and
`BubbleWait`. `time.Sleep` and `time.Now` each read the clock of the bubble of
the caller through the runtime. `internal/linkhook` carries `SynctestTest`,
because `testing/synctest` needs the T of the bubble and the import rule of Go
keeps the 2 packages apart. `testdata/programs/synctestuse` holds the witness,
and `make std-test STD=testing/synctest` runs the 13 real tests of the package.
Read decision 346 and the limits of `docs/parity.md`.

## The tests of the Go repository

The user asked on 2026-08-29 whether the compiler can run the tests of the Go
repository, exactly as GopherJS does. The answer is yes for a package whose test
files stay inside the subset. 21 packages now run their real tests, and `make
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
- `TestTestsOfGoroot` in `internal/driver` runs 21 packages from the Go
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


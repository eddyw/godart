# The support of each package

This file names how much of each standard library package the surface holds.
`docs/what-compiles.md` names the shape of a program that builds.
`docs/parity.md` names each behavior that differs.

## The measure

Run `make surface`. `internal/surface` holds the tool. The API name counts
come from 2026-09-07. The import path counts below come from 2026-09-15.

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

The surface holds 262 import paths: 164 standard packages, the interop package
`syscall/dart`, 85 internal packages, and 12 vendored packages of
`golang.org/x`. `make surface` measures 174 of them against the real API, and
165 of those hold each exported name. The measure skips the interop package
and each internal package, because no package of Go stands behind them.
`make surface ARGS=-absent` names each package of the standard
library that the surface does not hold.

The earlier count omitted `internal/asan`, `internal/msan`, and `internal/race`.
Their overlays already existed before this validation. Each exposes a disabled
sanitizer or race detector flag.

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

165 of the 174 measured packages hold each exported name of Go. The table
below names the 129 of them that this document lists by form. The 39 that stay
out of the table are the packages of `docs/library.md` that no standard list
of this document holds.

| Form | Packages |
|---|---|
| Native, 76 | `bufio`, `cmp`, `compress/bzip2`, `compress/flate`, `compress/gzip`, `compress/lzw`, `compress/zlib`, `container/heap`, `container/list`, `container/ring`, `context`, `crypto`, `crypto/aes`, `crypto/des`, `crypto/dsa`, `crypto/ecdh`, `crypto/ecdsa`, `crypto/ed25519`, `crypto/elliptic`, `crypto/hkdf`, `crypto/hmac`, `crypto/pbkdf2`, `crypto/rc4`, `crypto/rsa`, `crypto/sha3`, `crypto/x509/pkix`, `debug/dwarf`, `debug/macho`, `debug/plan9obj`, `encoding`, `encoding/ascii85`, `encoding/base32`, `encoding/base64`, `encoding/csv`, `encoding/hex`, `encoding/pem`, `encoding/xml`, `go/ast`, `go/build/constraint`, `go/doc/comment`, `go/format`, `go/parser`, `go/printer`, `go/scanner`, `go/token`, `hash`, `hash/adler32`, `hash/crc64`, `hash/fnv`, `html`, `html/template`, `image`, `image/color`, `image/color/palette`, `image/draw`, `image/gif`, `image/jpeg`, `index/suffixarray`, `io`, `io/fs`, `io/ioutil`, `math/cmplx`, `mime`, `mime/quotedprintable`, `net/http/cookiejar`, `net/url`, `path`, `regexp`, `testing/fstest`, `testing/iotest`, `testing/slogtest`, `text/tabwriter`, `text/template`, `text/template/parse`, `unicode`, `unicode/utf16` |
| Patch, 25 | `archive/tar`, `archive/zip`, `bytes`, `crypto/cipher`, `crypto/fips140`, `debug/elf`, `debug/pe`, `encoding/asn1`, `encoding/json`, `encoding/json/jsontext`, `encoding/json/v2`, `fmt`, `hash/crc32`, `log/slog`, `math`, `math/big`, `math/bits`, `math/rand`, `math/rand/v2`, `mime/multipart`, `net/netip`, `net/textproto`, `regexp/syntax`, `slices`, `sort` |
| Overlay, 28 | `crypto/md5`, `crypto/rand`, `crypto/sha1`, `crypto/sha256`, `crypto/sha512`, `crypto/subtle`, `crypto/x509`, `embed`, `encoding/binary`, `encoding/gob`, `errors`, `flag`, `iter`, `log`, `maps`, `net/http/httptest`, `os/signal`, `path/filepath`, `runtime/debug`, `strconv`, `strings`, `sync`, `testing/quick`, `testing/synctest`, `time`, `time/tzdata`, `unicode/utf8`, `unique` |

`time/tzdata` holds no exported name. The import gives the embedded zone
database to `time`, and it works.

A full package can still differ in behavior. `path/filepath` holds each name,
and the separator is always `/`. `docs/parity.md` holds each such difference.

## Partial support

9 packages hold a part of the names of Go. The run of 2026-09-05, after the
nineteenth session:

| Package | Names | Held | The largest gap |
|---|---|---|---|
| `sync/atomic` | 90/94 | 95% | The 4 `unsafe.Pointer` functions. `Pointer[T]` holds the same 4 operations with a typed pointer. |
| `os` | 195/207 | 94% | `File.Fd`, `File.SyscallConn`, `NewFile`, `StartProcess`, `ProcAttr`, and the 3 time names of `ProcessState`. `dart:io` hides the descriptor, and it reports no CPU time. |
| `os/exec` | 44/49 | 89% | `Cmd.ExtraFiles`, `Cmd.SysProcAttr`, and the 3 time names of `ExitError` |
| `reflect` | 183/188 | 97% | `NewAt`, `SliceAt`, `Value.SetPointer`, `Value.InterfaceData`, `Value.UnsafeAddr`. Each 1 needs an address or `unsafe.Pointer`, which non-goal N3 holds out. |
| `net/http` | 362/392 | 92% | `HTTP2Config`, `ClientConn`, `ServeTLS`, and `Transport.TLSHandshakeTimeout`. The runtime owns the HTTP/2 frame layer, and the secure socket of Dart takes over no socket that stands open. |
| `runtime` | 52/114 | 45% | `MemStats`, `ReadMemStats`, `SetFinalizer`, `Breakpoint`, the profile of the memory, of the block, and of the mutex, and the trace. The Dart VM reports none of the numbers. |
| `net` | 272/346 | 78% | `IPConn`, `FileConn`, `FileListener`, and the keep alive timers. Each 1 needs a raw socket, a file descriptor, or an ioctl of Windows. 8 socket options joined on 2026-09-04. Read decision 330. |
| `crypto/tls` | 126/263 | 47% | The QUIC connection, `ConnectionState.PeerCertificates`, and each `Config` field that would change the handshake. Dart owns the handshake, and `SecureSocket.peerCertificate` gives the leaf alone and not the chain. Read decision 331. |
| `syscall` | 55/2228 | 2% | Each system call, each constant of the platform, and each error number whose value or its text differs between macOS and Linux. |

`testing` left this table on 2026-09-03, `crypto/x509` left it on 2026-09-04,
and `encoding/json` left it on 2026-09-04. The sixteenth session added 166 names, and decision 319 and
decision 320 name each 1 that stayed out. The seventeenth session then took
`crypto/x509` from 193 of 261 to each of the 261, because the crypto stack
joined the surface. It also gave `net` 8 socket options, 259 names to 267.

`syscall` is a special case. The package of Go holds the whole system call
interface of the platform, and the surface holds no such interface. The overlay
holds the `Signal` type with the 12 signal numbers, and the `Errno` type with
the 28 error numbers, that macOS and Linux agree on, because `os`,
`os/signal`, and `path/filepath` need them. A number that the 2 systems write
differently, for example `SIGUSR1` and `ENOTEMPTY`, stays out. A Go program
reaches the system through `os`, through `net`, and through the interop
package `syscall/dart`.

`encoding/json` left the table on 2026-09-06. The package reads the real source
of Go with 2 patches since then, thus it holds each name of the v1 API and of
the option set of the v2 implementation. Decision 386 holds the patches.

## Why a name of 5 packages stays out

The work of 2026-09-04 read each missing name of `runtime`, `os`, `os/exec`,
`sync/atomic`, and `reflect`, and it asked 1 question of each 1: can the Dart
VM or `dart:io` carry the name with the behavior of Go? A name that reached
that bar joined the overlay. Each name below did not reach it, and the cause
names the real limit. Read decision 319.

A stub that gives a zero value is worse than a missing name, because the
program then reads a wrong number with no report. A missing name gives an
error of the compiler, and that error names the gap at the earliest point.

The session added 19 names. `runtime` went from 43 to 52, `reflect` from 173 to
180, `os` from 193 to 195, and `os/exec` from 43 to 44. `sync/atomic` stays at
90, because each name of its gap needs the package `unsafe`.

### runtime, 62 names

| Name | The cause |
|---|---|
| `MemStats`, `ReadMemStats`, 34 names | The Dart VM reports no number of its own heap inside the process. `ProcessInfo.currentRss` of `dart:io` names the resident set of the whole process, and Go names no field that way. A read that fills 1 field and leaves 32 at 0 reports a wrong heap. |
| `MemProfile`, `MemProfileRecord`, `MemProfileRate`, 11 names | The Dart VM counts no allocation of the program. Each record would carry an invented number. |
| `BlockProfile`, `BlockProfileRecord`, `SetBlockProfileRate`, 7 names | The scheduler lives in the Dart runtime library, and it measures no time of a block. |
| `MutexProfile`, `SetMutexProfileFraction` | A mutex of gart is a flag with a waiter queue, and it counts no contention. |
| `SetCPUProfileRate` | The rate reaches no profile, because the Dart VM samples no Go frame. |
| `ThreadCreateProfile` | The Dart VM makes each thread, and gart records no stack of such a creation. |
| `StartTrace`, `StopTrace`, `ReadTrace` | The trace of Go holds the events of the Go runtime in the binary format of Go. The Dart VM writes no such event. |
| `SetCgoTraceback` | The 3 parameters name `unsafe.Pointer`, and the subset holds no cgo. |
| `SetFinalizer` | The finalizer of Go takes the object itself. A Dart `Finalizer` must hold no reference of the object that it watches, thus such a capture would stop each collection of it and the finalizer would never run. `runtime.AddCleanup` maps to the Dart form, and it stands already. |
| `Breakpoint` | `debugger` of `dart:developer` is the Dart form. With no debugger it waits for a client of the service with no report, and Go ends the program with SIGTRAP. A wait with no report is a silent answer. |

### os, 12 names

| Name | The cause |
|---|---|
| `File.Fd` | `dart:io` gives a `RandomAccessFile` and no descriptor of the system. The index of the handle table is no descriptor, and a program compares the result against 0, 1, and 2. |
| `File.SyscallConn` | The result holds the type `syscall.RawConn`, and the surface holds no such type. |
| `NewFile` | The call takes a descriptor. Read `File.Fd`. |
| `StartProcess`, `ProcAttr`, 6 names | `ProcAttr.Files` names 1 descriptor for each stream of the child, and `ProcAttr.Sys` names `syscall.SysProcAttr`. `dart:io` gives the child 3 streams and no other descriptor. |
| `ProcessState.UserTime`, `SystemTime`, `SysUsage` | `dart:io` reports no CPU time of a child and no `rusage`. |

### os/exec, 5 names

| Name | The cause |
|---|---|
| `Cmd.ExtraFiles` | The child takes no descriptor beyond its 3 streams. Read `os.StartProcess`. |
| `Cmd.SysProcAttr` | The field names `syscall.SysProcAttr`, and `dart:io` honors no field of it. |
| `ExitError.UserTime`, `SystemTime`, `SysUsage` | Each 1 comes from `os.ProcessState`. Read the row of that type. |

### sync/atomic, 4 names

`LoadPointer`, `StorePointer`, `SwapPointer`, and `CompareAndSwapPointer` each
name `unsafe.Pointer` in the signature, and stage 2 rejects an import of
`unsafe`. `Pointer[T]` holds the same 4 operations with a typed pointer, and Go
names it the safer form of each 1.

### reflect, 5 names

`FuncOf` joined the surface on 2026-09-03. The 5 names below each need a
machine address, and non-goal N3 holds such an address out.

| Name | The cause |
|---|---|
| `NewAt`, `SliceAt`, `Value.SetPointer` | Each 1 takes an `unsafe.Pointer`. |
| `Value.InterfaceData` | The result is the 2 words of an interface value. Dart holds no such pair. |
| `Value.UnsafeAddr` | Go gives the address of the value. `Value.Pointer` gives the identity of an object instead, and `Value.Addr` builds a new cell at each call for a field of a scalar type. Thus 2 reads of 1 field would give 2 numbers, and a number that changes is a wrong answer. |

## Why a name of the 5 network packages stays out

The work of 2026-09-04 read each missing name of `net`, `net/http`,
`crypto/tls`, `crypto/x509`, and `encoding/json`, and it asked 1 question of
each 1: can `dart:io`, the ASN.1 parser of the overlay, or plain Go carry the
name with the behavior of Go? A name that reached that bar joined the overlay.
Each name below did not reach it, and the cause names the real limit. Read
decision 320.

The session added 147 names. `net` went from 218 to 259, `net/http` from 341
to 361, `crypto/tls` from 118 to 126, and `crypto/x509` from 115 to 193.
`encoding/json` stays at 67, and the section below holds the measure that kept
its last 2 reachable names out.

### net, 79 names

| Name | The cause |
|---|---|
| `IPConn` and its 18 methods, `ListenIP`, `DialIP`, `Dialer.DialIP`, 22 names | A connection of the network ip is a raw socket. `dart:io` opens a stream socket, a datagram socket of UDP, and a socket of the Unix domain, and no raw socket. |
| `FileConn`, `FileListener`, `FilePacketConn`, and the `File` method of 5 types, 8 names | Each 1 takes or gives an `*os.File` of the descriptor of a socket. `dart:io` hides the descriptor of each socket. |
| The `SyscallConn` method of 5 types, 5 names | The result holds the type `syscall.RawConn`, and the surface holds no such type. |
| `SetLinger` | The option takes a `struct linger` of 2 numbers, and the byte layout of that struct differs between the platforms. `setRawOption` of Dart takes the bytes themselves. |
| `KeepAliveConfig` and its 4 fields, `Dialer.KeepAlive`, `Dialer.KeepAliveConfig`, `ListenConfig.KeepAlive`, `ListenConfig.KeepAliveConfig`, `TCPConn.SetKeepAliveConfig`, `SetKeepAlivePeriod`, 11 names | Each 1 carries the 3 timers of the keep alive. macOS and Linux each hold a socket option for them, and Windows writes the 3 through the ioctl `SIO_KEEPALIVE_VALS`. A field that reaches no operation lies. |

**8 socket options joined on 2026-09-04.** `TCPConn.SetNoDelay`,
`SetKeepAlive`, `SetReadBuffer`, and `SetWriteBuffer`, and `SetReadBuffer` and
`SetWriteBuffer` of `UDPConn` and of `UnixConn`. `Socket.setOption` of Dart
reaches `TCP_NODELAY` by name, and `Socket.setRawOption` reaches each other 1
with the number of the system. `_sockOptionNumber` of
`runtime/goruntime/lib/src/net.dart` holds the table of the platform, and a
platform outside the table gives an error. Read decision 330.
| `Dialer.Control`, `Dialer.ControlContext`, `ListenConfig.Control` | Each 1 takes a `syscall.RawConn` of the new socket. |
| `Dialer.LocalAddr` | Dart binds no local address of a dial. `DialTCP`, `DialUDP`, and `DialUnix` of this overlay each give an error for a laddr that holds a value, thus no dial binds the wrong address with no report. |
| `Dialer.DualStack`, `Dialer.FallbackDelay` | Each 1 tunes the 2 dials of Happy Eyeballs. 1 dial of Dart resolves the name and opens the socket. |
| `Dialer.MultipathTCP`, `Dialer.SetMultipathTCP`, `ListenConfig.MultipathTCP`, `ListenConfig.SetMultipathTCP`, `TCPConn.MultipathTCP` | Multipath TCP is a transport of Linux that Dart never opens. |
| `Interface.MTU`, `Interface.HardwareAddr`, `Interface.Flags`, `Interface.MulticastAddrs` | `NetworkInterface.list` of `dart:io` gives the name, the index, and each address of an interface, and nothing else. |
| `ListenUnixgram` | Dart opens a stream socket of the Unix domain alone. |
| `UDPConn.ReadMsgUDP`, `WriteMsgUDP`, `ReadMsgUDPAddrPort`, `WriteMsgUDPAddrPort`, `UnixConn.ReadMsgUnix`, `WriteMsgUnix` | Each 1 reads or writes a control message. The read also reports MSG_TRUNC for a datagram that lost bytes, and the read of the runtime reports no such flag. |
| `UnixConn.WriteTo`, `UnixConn.WriteToUnix` | Go writes the datagram with sendto. A socket that a dial connected gives `ErrWriteToConnected`, and a socket that an accept gave holds no such mark: macOS then writes the bytes and Linux gives EOPNOTSUPP. Dart holds no sendto, thus the overlay can match neither. |
| `TCPListener.SetDeadline`, `UnixListener.SetDeadline` | The deadline ends a wait for an accept. The accept of the runtime takes no deadline, and a cancel of the Go side would drop a client that already arrived. |
| `UnixListener.SetUnlinkOnClose` | The close of the runtime always removes the file of the socket. |

### net/http, 31 names

| Name | The cause |
|---|---|
| `HTTP2Config` and its 12 fields, `Server.HTTP2`, `Transport.HTTP2`, `Server.DisableClientPriority`, 16 names | The HTTP/2 session lives in the Dart runtime library, and it exposes no window, no frame size, no header table, and no ping. A config that reaches no operation lies. |
| `ClientConn` and its 8 methods, `Transport.NewClientConn`, 10 names | The API of Go 1.27 hands 1 connection to the caller, with a count of the free streams and a hook of the state. The pool of the overlay owns each connection, and it reports no such count. |
| `ServeTLS`, `Server.ServeTLS` | Each 1 wraps a listener that stands open. The secure socket of Dart takes over no socket that another call opened. `ListenAndServeTLS` opens the secure listener itself, thus it works. |
| `Transport.TLSHandshakeTimeout` | The runtime opens the socket and runs the handshake in 1 call, thus no limit covers the handshake alone. A limit of the whole dial would end a slow connection that Go keeps. |
| `Transport.OnProxyConnectResponse` | The runtime writes the CONNECT request and reads its answer inside `DialTLSProxy`, thus the answer reaches no Go code. |

### crypto/tls, 137 names

| Name | The cause |
|---|---|
| Each QUIC name, 37 names | `dart:io` holds no QUIC transport, and it exports no TLS record layer of its own. |
| Each `Config` field of the handshake, 28 names | Dart owns the whole handshake. `RootCAs`, `ClientCAs`, `ClientAuth`, `CurvePreferences`, `GetCertificate`, `VerifyPeerCertificate`, and the 22 others each reach no operation. |
| `ClientHelloInfo` and its 12 members, `CertificateRequestInfo` and its 5 members, 19 names | Each 1 reaches Go through a callback of the handshake, and Dart calls none. |
| `AlertError`, `CertificateVerificationError`, `RecordHeaderError`, `ECHRejectionError`, and their members, 15 names | Go gives each 1 at a point of the handshake. Dart reports a text, thus no path of the overlay builds them. A type that no path gives makes `errors.As` report false where Go reports true. |
| `ConnectionState.Version`, `CipherSuite`, `CurveID`, `DidResume`, `ECHAccepted`, `HelloRetryRequest`, `LocalCertificate`, `TLSUnique`, `ExportKeyingMaterial`, 9 names | Dart reports the protocol of ALPN alone. |
| `ConnectionState.PeerCertificates`, `VerifiedChains`, `OCSPResponse`, `SignedCertificateTimestamps`, `Conn.OCSPResponse`, `Conn.VerifyHostname`, 6 names | `SecureSocket.peerCertificate` of Dart gives 1 `X509Certificate`, and Go names the whole chain that the peer sent. A slice of 1 element would answer `PeerCertificates[0]` right and `len(PeerCertificates)` wrong, with no report. Dart exports no verified chain, no OCSP answer, and no timestamp. `Conn.VerifyHostname` needs a parse of the leaf, thus it would give `crypto/x509` to each program that reads `net/http`. Read decision 331. |
| `SessionState`, `ClientSessionCache`, `ClientSessionState`, `NewLRUClientSessionCache`, `ParseSessionState`, `NewResumptionState`, and their members, 10 names | Dart owns the cache of the session, and it exports no ticket. |
| `Certificate.PrivateKey`, `OCSPStaple`, `SignedCertificateTimestamps`, `SupportedSignatureAlgorithms` | The overlay parses no private key, and Dart reads the 2 PEM blocks that `X509KeyPair` took. |
| `EncryptedClientHelloKey` and its 3 fields | The encrypted client hello needs a key of the server inside the handshake, and Dart takes none. |
| `Client`, `Server`, `NewListener` | Each 1 wraps a `net.Conn` that stands open. The secure socket of Dart takes over no socket that another call opened. |
| `Conn.NetConn` | The result must name the socket under the TLS stream. A read of the value of the overlay would give the plain text of the stream, and Go gives the cipher text. |
| `Conn.CloseWrite` | A half close. Dart owns the record layer, thus the runtime holds a `Socket` for a TLS connection and it reaches no `shutdown` of 1 direction. |

### crypto/x509, 0 names

Each of the 261 names stands since 2026-09-04. Phase 6 landed the crypto stack,
thus each gate of the table below opened.

| Name | The state |
|---|---|
| `Certificate.Verify`, `VerifyOptions` and its 7 fields, `InvalidReason` and its 11 constants, `CertificateInvalidError`, `UnknownAuthorityError`, `ConstraintViolationError`, `UnhandledCriticalExtension`, and their members, 33 names | Present. The file `verify.go` and the file `constraints.go` of the overlay each come from the real Go source, thus the chain check is the check of Go. The block that calls the verifier of the platform stays out, because the Dart target reaches no such verifier. |
| `Certificate.CheckSignature`, `CheckSignatureFrom`, `CheckCRLSignature`, `CertificateRequest.CheckSignature`, `RevocationList.CheckSignatureFrom`, `ErrUnsupportedAlgorithm`, `InsecureAlgorithmError`, 8 names | Present. Each check reads `crypto/rsa`, `crypto/ecdsa`, or `crypto/ed25519`. |
| `Certificate.PublicKey`, `CertificateRequest.PublicKey` | Present. The field holds a `*rsa.PublicKey`, a `*dsa.PublicKey`, a `*ecdsa.PublicKey`, an `ed25519.PublicKey`, or a `*ecdh.PublicKey`. A key of ML-DSA gives an error of the parse, because `crypto/mldsa` stands outside the surface. |
| `ParsePKIXPublicKey`, `MarshalPKIXPublicKey`, `ParsePKCS1PublicKey`, `MarshalPKCS1PublicKey`, `ParsePKCS1PrivateKey`, `MarshalPKCS1PrivateKey`, `ParseECPrivateKey`, `MarshalECPrivateKey`, `ParsePKCS8PrivateKey`, `MarshalPKCS8PrivateKey` | Present. Each file comes from the real Go source, less the 2 branches of ML-DSA of `pkcs8.go`. |
| `CreateCertificate`, `CreateCertificateRequest`, `CreateRevocationList`, `Certificate.CreateCRL` | Present, for a key of RSA, of ECDSA, and of Ed25519. |
| `DecryptPEMBlock`, `EncryptPEMBlock`, `IncorrectPasswordError`, `PEMCipher` and its 5 constants, 9 names | Present. The file `pem_decrypt.go` comes from the real Go source, and it reads `crypto/aes` and `crypto/des`. |
| `SetFallbackRoots` | Present. The Dart target holds no store of the platform, thus the pool always reaches `Verify`. |
| `CertPool.AddCertWithConstraint` | Present. The constraint reaches `Verify`. |

`SystemCertPool` still gives a `SystemRootsError`. The secure socket of Dart
owns the store of the platform, and it exports no root certificate. A program
that checks a chain names its own roots, or it calls `SetFallbackRoots`.

`testdata/programs/x509verify` builds a root, an intermediate, and a leaf, and
it signs each 1. It then builds a forged leaf with another key and the same
subject, and `Verify` rejects it.

### encoding/json, 0 names

The package left this list on 2026-09-06. It reads the real source of Go now,
thus each name of it stands in the surface. The 25 names below stayed out while
the overlay held a walk of the type descriptor of its own, and each 1 named a
`reflect.Type` or an offset of the input that the walk did not hold.

## jsontest joined the surface on 2026-09-06

`encoding/json/internal/jsontest` holds the corpus of the real test files of
`encoding/json` and of `encoding/json/v2`. The 7 files of `_embed` hold
275,490 bytes of JSON under zstd, and the package reads them through `embed`
and `internal/zstd`. Both stood in the surface already, thus the package needed
1 line of `internal/overlay/native.go`.

The surface holds 254 import paths now, and 253 stood before. `make surface`
measures no internal package, thus the count of 165 packages of 174 holds.

The work needed 2 rules of stage 1. The loader read a `//go:embed` directive of
the module alone, and a native package of GOROOT holds one too. It also
rejected `_embed/*.json.zst`, and the go command embeds a file that a pattern
names with no test of the dot and of the low line. `docs/parity.md` holds the
2 rules.

## A fused multiply and addition moves the last bit

The specification of Go lets an implementation join a multiply and an addition
into 1 operation, and give a result that differs from the 2 rounded steps. gc
joins them on arm64 and the Dart compiler does not, thus 8 functions of `math`
differ from real Go by 1 unit of the last place.

`testdata/programs/mathbits` measured it on 2026-09-06. It reads 517 inputs
that hold each special value of IEEE 754 and a spread of bit patterns, and it
covers 33 functions of `math`.

| Function | Values that differ | Largest difference |
|---|---|---|
| `Hypot` | 11 of 3,102 | 2 units |
| `Tan` | 9 of 517 | 1 unit |
| `Sin` | 5 of 517 | 1 unit |
| `Log2` | 5 of 517 | 1 unit |
| `Cos` | 3 of 517 | 1 unit |
| `Asin`, `Atan2`, `Expm1` | 1 each | 1 unit |

The Linux/AMD64 repeat on 2026-09-16 found 1-unit differences in `Exp` (19 inputs),
`Sinh` (16), `Cosh` (16), and `Tanh` (1).
The fixture uses 11 significant digits for these 4 functions and the 8 functions above.
All other results retain the complete bit pattern, with the explicit exception below.

The AMD64 `archLog` assembly in Go 1.27.0 does not normalize positive subnormal inputs.
Its result differs substantially from the portable Go implementation that gart uses.
`Log10` inherits that difference. The fixture has 3 such inputs for each function.
These 6 comparisons print `host-dependent-subnormal` and do not count as parity passes.
The fixture retains all other logarithm inputs and comparisons.
The `confmath` fixture marks 9 additional result occurrences in `Log`, `Log10`, `Y0`, and `Lgamma`.
`Y0` and `Lgamma` inherit the same subnormal `Log` difference. Other results and the `Lgamma` sign remain exact comparisons.

**A program must not depend on the last bit of a transcendental function.**
Go can produce different final bits on AMD64 and ARM64 for the same source.

## A conversion of a float to an integer outside the range

`int32(f)` of a float64 above the range of an int32 gives a result that the
specification of Go leaves to the implementation. gc saturates on arm64,
`2147483647`, and it gives the indefinite value on amd64, `-2147483648`. gart
holds the low 32 bits of the int64 form.

The same limit applies to `int64(f)` outside `[-2^63, 2^63)`.
For positive overflow, Go on AMD64 gives `MinInt64`. Go on ARM64 and gart give `MaxInt64`.
The [Go specification](https://go.dev/ref/spec#Conversions_between_numeric_types) permits different results when the result type cannot represent the value.

`testdata/programs/intfloat` compares integer results only within the representable range.
It marks the other int64 cases as `implementation-dependent` and retains all float-to-float comparisons.
It also covers the largest float64 below `2^63`, which remains a valid int64 input.

## The open file limit of the machine

A run of `make ci` writes more than 1100 temporary Go modules: 366 programs of
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

Stage 2 rejects an import of each package below with a source position. 16
packages of the standard library stand here after the nineteenth session. 30
stood here at the start of the eighteenth session, and 62 at the start of
2026-09-04.

| Package | The cause |
|---|---|
| `unsafe`, `runtime/cgo`, `plugin`, `runtime/race`, `structs` | A non-goal of the project. No cgo, and no goroutine on a second thread. |
| `runtime/pprof`, `runtime/trace`, `runtime/metrics`, `runtime/coverage`, `net/http/pprof`, `expvar` | The Dart VM counts no allocation, no block, and no contention, and it writes no event of the tracer of Go. `expvar` publishes `memstats`, which reads `runtime.MemStats`. |
| `debug/buildinfo` | It reads the build record of a Go binary, and gart writes no such binary. |
| `go/importer` | `Default` reads `runtime.Compiler`, which gart reports as `gart`. The gc form reads a compiled export file that gart never writes, the gccgo form needs 5 more packages and the gccgo driver, and the source form fails on the same compiler name. A caller writes an importer of its own over `go/types`. |
| `net/http/fcgi` | `child.go` reads `net.FileListener`, and `dart:io` hides the descriptor of a socket. `Serve(nil, h)` is the standard entry of FastCGI, thus a patch reaches no answer. |
| `log/syslog` | `unixSyslog` needs a datagram socket of the Unix domain, and `dart:io` opens the stream form alone. `Dial` over udp and over a stream socket each write the message end to end, and `New` and `NewLogger` each fail. |
| `os/user` | The pure Go path of Go compiles under gart with the build tag `osusergo`, and `Current` still fails. It reads `os.Getuid`, and the overlay reports -1 because `dart:io` gives no user id. |

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
   which builds a closure of the exact Dart type. `FuncOf` and `StructOf` each
   joined later, and the note below holds the earlier state.
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

**Each item of the 3 lists above is closed.** The seed corpus of a fuzz target,
the verb `%#v`, the comparison of a pointer inside an interface, and the
unsigned slice bound each landed on 2026-09-04. Decisions 312 to 326 hold them.

## The road to full parity

**The 7 phases of 2026-09-04 are closed, the eighteenth session closed 12 of
the 15 items that stood after them, and the nineteenth session closed each item
that stayed.** No name and no package of the standard library stays reachable
inside the non-goals now. The table below holds the state after each
block of work, and the section after it names each item that stays out and its
cause.

| Set | At the start of 2026-09-04 | After the 7 phases | After the nineteenth session |
|---|---|---|---|
| Packages that hold each exported name | 112 of 123 | 149 of 159 | **165 of 174** |
| Standard packages outside the surface | 62 | 30 | **16** |
| Import paths of the surface | 210 | 226 | **254** |
| Names of the surface that stay reachable | 60 | 29 | **0** |
| Entries of the skip table that hold a compiler defect | 5 | 2 | **2** |
| Positions of the order of the evaluation that differ from gc | 5 | 0 | **0** |

**What the 7 phases gave.**

| Phase | Result |
|---|---|
| 1. The package variable initializer | Done. |
| 2. The 4 other order positions | Done. No position differs from gc. |
| 3. The 5 skip entries | 3 of 5. `peano` and `typeparam/nested` each took a corrected cause, and the eighteenth session corrected the cause of `typeparam/nested` again. |
| 4. The 2 intrinsics | The socket options gave 8 names of `net`. The peer certificate does not land, and the cause below names why. |
| 5. The 32 pure Go packages | 19 of 32, and `text/scanner` followed after a defect of the emitter fell. |
| 6. The crypto stack | **Done, and it holds each name.** 16 packages, and `crypto/x509` went from 193 of 261 to 261 of 261. |
| 7. The last reachable names | 68 of `crypto/x509` and 8 of `net`. 29 stay reachable, and each 1 waits for a gate below. |

**`crypto/x509` holds each name of Go.** `Certificate.Verify` builds a chain
and it checks each signature, thus a forged certificate never passes.
`testdata/programs/x509verify` signs a real chain of a root, an intermediate,
and a leaf with ECDSA, RSA, and Ed25519. It then builds a forged leaf with the
same subject and the same key, signed by another intermediate, and `Verify`
rejects it.

### The reachable names and the reachable packages are closed

**No name of the surface and no package of the standard library stays reachable
inside the non-goals.** The nineteenth session closed the last 29 names and the
last 2 packages.

| Item | Result |
|---|---|
| `encoding/json`, 25 names | **Closed.** The package reads the real Go source now, and it holds each of its 92 exported names. The hand-written overlay of 1850 lines is gone. |
| `encoding/json/v2` | **Closed.** 2 rules of the emitter opened it: a defined type over an instance of a generic type, and a wrapper for a function value of a narrower Dart parameter type. Read decisions 347 and 348. |
| `testing/synctest` | **Closed.** The scheduler holds a bubble with a clock of its own. Read decision 346, and decision 351 for the rule of a lock and of a timer. |
| `reflect.StructOf` | **Closed.** A shape that the program names reads the class of that shape, and a fresh shape holds its value in a list of the fields. Read decision 349. |
| `reflect.Method.Func` | **Closed.** `_callValue` reads `Function.apply`, thus Call of the value needs no factory of a Dart type. The 2.7 percent measure of decision 325 covered such a factory. |
| `net.ListenMulticastUDP` | **Closed.** 2 intrinsics carry it, over `RawDatagramSocket.bind` with a shared port and over `joinMulticast`. |
| `net/http.Request.Cancel` | **Closed.** `Transport.track` already lives up to the close of the body of the answer, thus the watch of the channel stands there. |

**The measured cost of the real `encoding/json`.** The Dart source of
`testdata/programs/jsonuse` grows from 2,072,121 bytes to 4,548,555 bytes,
because the real package reads `reflect` and `encoding/json/v2`. The earlier
measure of 2026-09-04 put that cost at 14 percent, and the real figure is 120
percent. The project takes the real package, because a hand-written
reimplementation of 1850 lines is a permanent risk of a silent divergence from
Go.

**The 9 packages that the eighteenth session closed:** `crypto/mlkem`,
`crypto/mldsa`, `crypto/hpke`, `testing/cryptotest`, `encoding/json/jsontext`,
`debug/gosym`, `weak`, `net/http/httptrace`, and `net/http/httputil`.
`image/png` and `crypto/mlkem/mlkemtest` joined them, and neither stood on that
list.

**`testing/synctest` holds each name of Go.** The scheduler holds a bubble: a
fake clock, a timer queue of that clock, and the set of the goroutines of the
bubble. The clock moves only when each goroutine of the bubble is durably
blocked, thus a sleep inside a bubble takes no real time. Decision 346 holds
the design, and decision 351 holds the rule of a lock and of a timer.
`testdata/programs/synctestuse` holds the witness, and the 13 real tests of the
package pass under `TestTestsOfGoroot`. The 6 other tests each skip, because
each 1 runs the test binary again as a subprocess.

### What stays out, and why no work removes it

**16 packages.**

| Cause | Packages |
|---|---|
| A non-goal: no cgo, and no second thread | `unsafe`, `runtime/cgo`, `plugin`, `runtime/race`, `structs` |
| The Dart VM counts no allocation and writes no profile | `runtime/pprof`, `runtime/trace`, `runtime/metrics`, `runtime/coverage`, `net/http/pprof`, `expvar` |
| gart writes no Go binary | `debug/buildinfo` |
| `runtime.Compiler` reports `gart` | `go/importer` |
| `dart:io` hides the descriptor of a socket | `net/http/fcgi`, through `net.FileListener` |
| `dart:io` opens no datagram socket of the Unix domain | `log/syslog`. `Dial` works over udp and over a stream socket, and `New` and `NewLogger` each need the datagram form. |
| `dart:io` reports no user id | `os/user`. The pure Go path of Go compiles under gart, and `Current` still fails because `os.Getuid` reports -1. |

**329 names of the 9 partial packages, and 2173 names of `syscall`.** No name
of them stays reachable. The nineteenth session closed the last 29.

| Cause | The count |
|---|---|
| Dart owns the TLS handshake and the session | 137 names of `crypto/tls`. Each QUIC name, each `Config` field of the handshake, each callback, and each error type of a handshake point. `SecureSocket.peerCertificate` gives the leaf alone and not the chain, thus `PeerCertificates` would answer `len` wrong with no report. |
| `dart:io` opens no raw socket and reads no control message | 74 names of `net` |
| The HTTP 2 session lives in the Dart runtime library | 30 names of `net/http` |
| The Dart VM counts no allocation, no block, and no contention | 62 names of `runtime` |
| `dart:io` hides the descriptor of a file and of a socket | 12 names of `os`, 5 of `os/exec` |
| The subset holds no `unsafe` and no address | 4 names of `sync/atomic`, 5 names of `reflect` |
| A constant of the system that reaches no operation | 2173 names of `syscall`. A constant that no call reads is a lie, and a number that macOS and Linux write differently is 1 too. |

**2 entries of the skip table hold a compiler defect that no small change
removes.** `peano` is a torture test of the segmented stack of Go, and the Dart
VM holds a fixed stack. `typeparam/nested` compares the `reflect.Type` identity
of a local generic type of a generic function. The name of such a type now
holds the type argument of that function, and the name of it inside a type
argument of another instance still holds none. Read decision 343. The other 28
entries each hold a hard limit of the Dart VM, of the machine, or of the shape
of the comparison.

**A run of each entry on 2026-09-05 confirmed the cause of 28 of the 30, and it
opened 2.** `fixedbugs/issue13169` runs 100000 channel rounds with a deep
stack, and `fixedbugs/issue78081` runs 1600000 rounds of a byte append. The 2
needed more capacity than the machine gave, and the 2 pass now. They left the
table, and `TestRunFilesOfGoTest` runs them. `fixedbugs/issue34395` fills a
global array of 100 MB, and the run reached no end inside 10 minutes.

### The rule that decided each call

A stub that gives a zero value is worse than a missing name, because the
program then reads a wrong number with no report. A missing name gives an error
of the compiler, and that error names the gap at the earliest point.

A package that compiles and whose hooks never run is a rejection too.
`net/http/httptrace` failed exactly that way.

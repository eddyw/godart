# Compatibility

Gart aims to preserve Go behavior within its supported subset.
A successful build does not establish complete Go compatibility.
Upstream tests establish parity only for their recorded inputs, targets,
SDK versions, modes, and exclusions.

## Language and runtime limits

| Area | Contract or limit |
|---|---|
| Language versions | Package and file versions govern language semantics. The installed SDK still supplies standard library code. |
| Goroutines | Cooperative execution in 1 Dart isolate. Loops in state machines yield periodically. Busy synchronous functions can block other work. |
| Parallelism | No shared-memory Go execution across native threads. `GOMAXPROCS` reports 1. `NumCPU` records native OS CPU availability at startup. JS and Wasm report 1, as Go js/wasm does. |
| Type layout | `unsafe.Sizeof`, `unsafe.Alignof`, and `unsafe.Offsetof` use 64-bit Go type metadata, including generic types. They do not describe Dart object storage. |
| Compiler diagnostics | Native Go optimization messages and instruction selection are outside the Gart compiler contract. Source compilation and runtime behavior require separate coverage. |
| Native access | Nil `unsafe.Pointer` values and constant-zero conversions are supported. No cgo, native pointer arithmetic, or pointer reinterpretation. |
| Pointer output | Process-local identities can represent pointers. They are not native memory addresses. |
| Map iteration | The runtime can expose insertion order. Programs must not depend on Go's unspecified iteration order. |
| Experimental methods | With `GOEXPERIMENT=fieldtrack`, `go:nointerface` methods remain callable but cannot satisfy interfaces or appear in reflection method tables. |
| Reflection | Descriptors cover a substantial subset. Dynamically created function types need an emitted callable factory. |
| Function metadata | Function values expose synthetic PCs and Go source metadata. Return PCs and their preceding call PCs resolve to the same source position. Reflection method wrappers can have no Go source location. PCs do not identify native instructions. |
| Garbage collection | Native hosts request collection through the Dart embedder API when available. Other hosts use best-effort collection. Cleanup deadlines can differ. |
| Runtime metrics | `runtime/metrics` exposes GODEBUG event counters and value accessors. Unknown metrics return `KindBad`. Native Go heap, scheduler, and collector measurements remain unavailable. |
| Deep calls | Recursive call cycles use resumable frames. Synchronous callbacks execute those frames without scheduler waits. Actual channel and I/O waits remain invalid in synchronous callbacks. |
| Linkname | Module variable aliases share storage with variables of the same type in imported packages. Module function aliases resolve to functions of the same type in the program import graph. Alias initializers, function alias bodies, and aliases to other aliases remain unsupported. |
| Stack traces | Go source positions, without native instruction offsets. `-trimpath` selects portable paths. Synthetic runtime frames use Dart implementation paths and can report line 0. |
| Zero-size arrays and slices | Zero-size elements use virtual storage. Logical lengths remain 64-bit values on VM, AOT, JS, and Wasm. |
| Numbers | Target representations differ. JavaScript conversion boundaries need particular care. |
| Time | Monotonic values support elapsed-time operations. Time-zone behavior can differ from Go's own database. |
| Environment | `os` and `syscall` share environment values. Node supplies the initial JS and Wasm environment. Browsers start with an empty environment. Child processes receive this state through `os/exec`. |
| Source tools | `go/build` accepts the `gart` compiler and `dart64` suffixes without a native archive path. `runtime.GOROOT` reads the process-start `GOROOT` value, or the selected build SDK path when that value is empty. `-trimpath` omits the default SDK path. |
| GODEBUG | Executables use the Go toolchain's module, workspace, and source defaults. Environment values override these defaults for library readers and `panicnil`. Other runtime controls and bisect patterns remain incomplete. The FIPS module rejects non-default modes. |

The verifier reports unsupported source constructs with source positions.
`runtime.Caller` and `runtime.Callers` retain Go source frames on VM, AOT,
JavaScript, and Wasm.
The verifier tests define precise rejection cases. Runtime and target limits
can still produce errors after compilation.

## Platform boundaries

Dart VM and AOT programs use `dart:io` adapters for files, processes, sockets,
and HTTP. These adapters do not provide all native descriptor, process-group,
file metadata, TLS configuration, or runtime inspection APIs.

Linux and macOS support `os.File.Fd` for regular files, pipes, directories,
and standard streams. The file owns the descriptor. Call `Close` explicitly.
`os.NewFile` adopts native file and directory descriptors on these hosts.
It preserves descriptor identity, the caller name, append mode, and ownership.
Native poll integration for adopted descriptors remains unsupported.
`os.Pipe` creates actual native descriptors on Linux and macOS.
The files own their descriptors and use close-on-exec and nonblocking flags.
Pipe reads and writes suspend the goroutine after actual EAGAIN results.
A timer schedules each syscall retry without a Dart descriptor registration.
Close cancels pending retries and preserves partial write counts.
Pipe Fd retains the nonblocking flags.
A blocking descriptor can stop the Dart scheduler.
Do not clear O_NONBLOCK before pipe reads or writes through os.File.
Files from `os.Pipe` support read and write deadlines, including pending I/O.
A zero deadline cancels the limit. Timeout errors wrap `os.ErrDeadlineExceeded`.
Regular files and adopted descriptors still return `os.ErrNoDeadline`.
Pipe descriptor inheritance remains unsupported.
The existing native errno capture limitation also applies to pipes.
`syscall.Dup` and `Dup2` duplicate actual descriptors and preserve shared file offsets.
Each duplicate owns its descriptor. `Dup2` replaces the target descriptor through the native syscall.
Linux also supports `syscall.Dup3`, with atomic target replacement and close-on-exec flags.
`syscall.CloseOnExec` and `SetNonblock` update actual native descriptor flags.
`syscall.Fsync` calls the native filesystem sync operation. Linux also supports `syscall.Fdatasync`.
Automatic file closure through Go finalizers remains unsupported.
Gart does not raise the inherited open-file limit at startup.
Programs can reach that limit sooner than native Go programs.
Explicit `syscall.Getrlimit` and `Setrlimit` calls use actual kernel resource limits on Linux and macOS.
`syscall.Getrusage` returns actual kernel resource counters on these hosts.
Counter values and units follow the host ABI.
Directory creation on Linux and macOS applies permission bits and the process umask at creation.
`syscall.Umask` changes the actual process mask and returns its previous value.
`syscall.Access` queries native permissions with the real user and group IDs on Linux and macOS.
Linux `syscall.Faccessat` queries descriptor-relative permissions with native flags and Go fallback rules.
Linux `syscall.Fchmodat` changes mode bits relative to an actual directory descriptor and preserves native flag errors.
`syscall.Mkdirat`, `syscall.Fchownat`, `syscall.Unlinkat`, and `syscall.Renameat` mutate paths relative to native directory descriptors on Linux.
`syscall.Getdents` and `syscall.ReadDirent` preserve native directory records and caller buffer padding on Linux.
macOS `syscall.Getdirentries` uses the native directory cursor and preserves caller buffer boundaries.
Like Go on macOS, it uses the descriptor offset and leaves `basep` unchanged.
JS and Wasm return `errors.ErrUnsupported`.
The original unsafe Getdirentries fixture remains outside this public API comparison.
Linux `syscall.Getcwd` returns native path bytes and the final NUL byte through the caller buffer.
Linux `syscall.UtimesNano`, `Utimes`, `Futimesat`, and `Futimes` use native timestamp operations and preserve Go's architecture-specific timeval conversions.
macOS also supports `syscall.UtimesNano`, `Utimes`, and descriptor-based `Futimes` through native timestamp operations.
`syscall.Utime` preserves native Go behavior for each Linux architecture.
A nil timestamp buffer updates timestamps on AMD64 and returns `EINVAL` on ARM64.
Temporary directories request mode `0700`.
Directory descriptors retain their identity after a rename or removal.
`File.Chmod` and `File.Chown` use native descriptors on Linux and macOS.
`os.Chmod`, `Chown`, and `Lchown` preserve native errors and raw path bytes on these hosts.
`Open`, `Create`, `OpenFile`, `ReadFile`, and `WriteFile` preserve raw file path bytes.
The host filesystem still decides which byte sequences it accepts.
`File.Chdir` uses the open descriptor and preserves directory identity after a rename.
On Unix platforms, `Getwd` preserves an absolute `PWD` alias only when it names the current directory.
Linux and macOS on x64 and arm64 use native `Stat`, `Lstat`, and descriptor-based `File.Stat`.
The same hosts support `syscall.Stat`, `Lstat`, and `Fstat` with `syscall.Stat_t` output and native errors.
Read the output structure only after a successful call.
These APIs preserve device/inode identity, permission bits, file types, and nanosecond modification times.
`os.Chtimes` preserves nanoseconds and omitted timestamps through native APIs on these hosts.
Filesystem timestamp limits still apply.
`SameFile` compares native identity on these targets and rejects external `FileInfo` implementations.
`FileInfo.Sys` returns the native SDK `*syscall.Stat_t` on these targets.
Other native hosts retain approximate metadata and a nil `FileInfo.Sys`.
Other native hosts and browsers do not support `os.File.Fd`.
Unreachable `os.File` state releases its file handle or directory descriptor through runtime cleanup.

`net/http/fcgi` supports supplied listeners. `Serve` with a nil listener
returns an explicit error because inherited listener descriptors remain unsupported.

`os.Root.FS` implements `fs.ReadDirFS` and `fs.ReadLinkFS`, with sorted directory entries and link metadata.
Rooted directory entries retain descriptor-derived metadata after file and root closure.
On supported Linux and macOS hosts, `os.Root` retains a native directory descriptor.
Root operations use that descriptor after directory renames. Native errno capture retains the limitation below.
On Linux and macOS, `Root.Link` preserves the final symbolic link and its inode identity.

`os.CreateTemp` and `os.MkdirTemp` use random decimal `uint32` suffixes and exclusive creation.
They preserve Go's collision retry limit and error metadata.

`os.UserConfigDir` and `os.UserCacheDir` use the native macOS locations.
Unix targets reject relative XDG directory values and preserve the HOME fallback.
Windows user-directory behavior remains incomplete.
On Unix targets, `os.TempDir` reads the current Go `TMPDIR` value.

`os/exec.LookPath` uses effective user and group execute access on Linux and macOS.
It rejects directories and continues past non-executable PATH candidates.
The native errno capture limitation also applies to these access calls.

After `Start`, `StdoutPipe` and `StderrPipe` support explicit `Close` and preserve closed-pipe errors.
Pipe adapters still lack pre-Start operations and native descriptor access.
Dart stream buffering does not preserve native pipe backpressure.
Dart can start a shell for an invalid executable format where Go returns `ENOEXEC`.

Linux and macOS support `syscall.Kill` and `syscall.Getpid` through native APIs.
`os.Process.Signal` supports the standard signals that Dart exposes, plus SIGIO and the native signal-0 query.
The native errno capture limitation also applies to signal calls.
Linux VM runs with Go 1.27.0 and Dart 3.13.3 intermittently fail the absent-process signal-0 assertion, while AOT passes.
The failure cause remains unproved.
SIGSTOP and SIGCONT preserve child stop and resume behavior.
Other platform-specific signal numbers remain unsupported by `os.Process.Signal`.
Signal watches support SIGHUP, SIGINT, SIGTERM, SIGUSR1, SIGUSR2, and SIGWINCH.
Standalone native programs ignore SIGUSR1 and SIGUSR2 before registration, as Go does.
Dart-hosted libraries do not install these default handlers.
`Stop` releases a watch when no channel or explicit ignore needs it.
Concurrent `Stop` and signal delivery can lose a pending signal.
Shell children can inherit Dart's ignored SIGPIPE disposition on Linux and macOS.
The same child terminates on SIGPIPE under native Go.
A child can also inherit a blocked SIGTTOU mask from Dart on these hosts.
Process-state text preserves the actual exit result, but child signal disposition parity remains incomplete.
Other native signal watches remain unsupported. Browser builds cannot send
or receive operating system signals.

Linux and macOS VM/AOT programs support Unix datagrams through actual native
nonblocking descriptors. Bind, connect, packet data, sender names, message
flags, control bytes, buffer sizes, directional shutdown and deadlines use
native operations. Received descriptor control messages retain close-on-exec.
Unix streams still lack control messages, File and SyscallConn.
Unix packet sockets and web Unix datagrams remain unsupported.
The native errno capture limitation also applies to Unix datagram calls.

`log/syslog` supports TCP, UDP, Unix streams and Unix datagrams on these hosts.
Local daemon availability remains an external requirement.

On Linux and macOS, x64 and arm64 builds use native directory descriptors
for recursive removal. Other native hosts retain Dart filesystem adapters.

Linux and macOS programs read native user IDs, group IDs, supplementary
groups, parent process IDs, and memory page sizes. Node programs read real
and effective user IDs and group IDs when Node exposes these methods.
Node retains placeholders for the other numeric values and does not support
supplementary groups. Browser targets retain placeholders for all these
numeric values and do not support supplementary groups.

Linux and macOS support native `syscall.Getpriority`, `syscall.Setpriority`, `syscall.Setpgid`, and `syscall.Setsid`.
Linux queries preserve the raw kernel priority value that Go exposes.
JS and Wasm targets report these APIs as unsupported.
The documented native errno atomicity limitation also applies to these calls.

macOS supports native `syscall.Sysctl` and `syscall.SysctlUint32` queries.
String queries preserve binary bytes and remove only the final NUL byte, as Go does.
Integer queries retain native size errors and name validation.
JS and Wasm report both APIs as unsupported.
The documented native errno capture limitation also applies to these queries.

Linux and macOS `syscall.Mkfifo` create actual FIFOs with native modes, the process umask, and native path errors.
JS and Wasm return `errors.ErrUnsupported`.
This creation API does not establish blocking FIFO I/O parity.
macOS `syscall.Chflags` and `syscall.Fchflags` change actual file flags through paths and native descriptors.
They preserve native errors and return explicit unsupported errors on web targets.
macOS `syscall.Pathconf` and `syscall.Fpathconf` query actual filesystem configuration values.
They preserve native name, path, and descriptor errors, with explicit unsupported errors on web targets.
Linux and macOS support native `syscall.Statfs` and `syscall.Fstatfs` with actual filesystem statistics.
Failed queries preserve caller output, and open descriptors retain access after path removal.
JS and Wasm report both APIs as unsupported.

macOS supports native `syscall.Getfsstat`, including count queries and short buffers.
The adapter retains actual mount statistics and unused caller buffer entries.
JS and Wasm report this API as unsupported.
The documented native errno capture limitation also applies to this API.

Linux and macOS support native `syscall.Gettimeofday`, including nil output pointers.
The adapter preserves the platform timeval layout, caller padding, and native errors.
Linux `syscall.Time` uses native `Gettimeofday` and preserves its optional output pointer.
JS and Wasm report this API as unsupported.

Linux and macOS expose network interface indexes, names, MTU values,
hardware addresses, and flags. The table includes interfaces without IP addresses.
Other native hosts report interface metadata as unsupported.
Address enumeration remains separate. Browsers return an empty interface table.
`Interface.Addrs` returns all interface addresses for index `0` on macOS.

TCP dials accept `tcp4` and `tcp6` and restrict numeric addresses and DNS results to the requested family.
Typed TCP dials reject malformed IP values and IPv6 addresses with `tcp4`.
`Dialer.LocalAddr` supports TCP and UDP source addresses and ports, including native bind errors.
`LocalAddr` for other networks returns an explicit unsupported error.
Both `DialTCP` forms accept explicit local addresses and ports.
TCP dials resolve service names and validate remote ports before socket creation.
They preserve cancellation metadata and expired-deadline errors.
Typed `tcp6` dials with mapped IPv4 addresses remain incomplete.
System service resolver selection and some resolver error details still differ.

On Linux and macOS, `net.DialUDP` supports an explicit local address and port.
Connected UDP sockets report the local address that the operating system selects.
The context-based UDP dialer supports `udp`, `udp4`, and `udp6`.
For typed UDP dials, bind and connect errors take precedence over an expired or canceled context.
Invalid remote addresses retain `AddrError` metadata after a successful explicit local bind.
Failed remote conversion releases that bound socket.
IPv4 `ListenMulticastUDP` disables multicast loopback and selects the supplied native interface for outgoing packets.
For valid interface indexes, selection uses `Interface.Index` and ignores `Interface.Name`.
An explicit index `0` selects the first IPv4 interface address on macOS and the system default on Linux.
A nil interface retains system selection on both hosts.
IPv4 multicast errors retain `OpError` address metadata and available native syscall error codes.
Linux IPv4 multicast preserves the Go `int32` conversion of interface indexes and kernel error codes.
On macOS, absent interface indexes retain the native `no such network interface` error.
`errors.Is` recognizes the same missing-interface cause as `InterfaceByName`.
`ListenMulticastUDP` with network `udp` supports IPv6 groups on Linux and macOS.
IPv6 multicast preserves native interface index conversions and kernel error codes.
`udp6` requires an IPv6-only bind, which remains incomplete.
Mixed-family destinations, IPv6 zones, and socket controls remain incomplete.

Native UDP error codes remain unreliable across separate Dart FFI calls.
A Linux VM trace confirms that native `EAGAIN` can become `EINVAL` before Dart reads `errno`.
Thus, a UDP read can fail with an incorrect error.
[Dart issue 38832](https://github.com/dart-lang/sdk/issues/38832) describes this missing atomicity.

Linux VM directory unlink can return typed `EINVAL` where native Go returns
typed `EISDIR`. Diagnostics do not establish the cause of this mismatch.

On Linux and macOS, TCP connection failures and socket-option failures retain available native error codes through `net.OpError` and `os.SyscallError`.
Closed sockets retain `net.ErrClosed`.

Linux `syscall.Uname` queries the native kernel identity and preserves all 6 raw fields.
Nil output retains the native error. JS and Wasm return `errors.ErrUnsupported`.

Browser builds use browser APIs. HTTP clients use fetch. Browser restrictions
on headers, credentials, network access, and cross-origin requests still apply.
Native filesystem operations, raw sockets, and HTTP servers do not become
available in a browser merely because a program compiles.

Node fetch can discard the original bytes of non-ASCII HTTP status text.
Gart cannot recover bytes that the host discards.

Native HTTP and TLS use the active Go SDK sources over Dart socket adapters.
Browser HTTP retains fetch. The project has no security audit.
Read [SECURITY.md](../SECURITY.md) before a security-sensitive deployment.

`crypto/x509` reads Linux certificate files and uses the macOS trust policy.
Root overrides and fallback roots follow Go. Native trust evaluation can
block the current Dart isolate. Windows system trust remains unavailable.
iOS and Android need device validation.

## Tests and coverage

`testing.T.TempDir` uses `GOTMPDIR` when set and removes its directories after the test.
Cleanup callbacks can register further cleanup callbacks. They execute in reverse registration order.

`gart test` supports tests, examples, fuzz seeds, filters, coverage, and AOT
benchmarks. It does not provide a fuzz-input generator or native allocation
measurements. Native `go test` remains a separate comparison baseline.

`gart test -timeout` defaults to 10 minutes. A value of 0 disables the timeout.
The deadline covers tests, fuzz seeds, and examples, but excludes benchmarks.
`T.Deadline` reports that deadline, including in subtests.
The timeout report does not yet list active tests.

Use these sources for current coverage:

- `make surface` for exported names and absent APIs.
- `internal/parity/testdata/coverage-manifest.json` for package classifications.
- `internal/parity/testdata/omitted-tests.json` for omitted assertions.
- The platform exception manifests for explicit skips and differences.
- [Upstream parity](upstream-parity.md) for evidence and validation rules.
- [JavaScript integers](js-integers.md) for the integer representation boundary.

A blocked package remains in the coverage denominator.
A skip, compile failure, or unsupported target does not count as a parity pass.
The source fingerprint must match before an SDK change receives acceptance.

The language adapter reads expected failures from the active Go test runner.
Reports retain these as `upstream-expected-failure`, separate from successful
diagnostic assertions. Both compilers must reject the original source, and
both original diagnostic matchers must fail as upstream expects.

## Flutter

Glutter uses shared Go nodes, signals, and generated Dart host adapters.
Application-owned bindings cover selected constructors and scalar services.
They do not yet provide arbitrary Dart object interop, custom value classes,
all generic APIs, or the complete Flutter Canvas API.
See the [Flutter architecture](flutter-design.md) and
[binding guide](../flutter/nodegen/README.md) for precise boundaries.

Linux supports native `syscall.Getxattr`, `Listxattr`, `Setxattr`, and `Removexattr` on actual files.
The adapters preserve byte paths, attribute names, values, size queries, flags, buffers, and native errors.
JS and Wasm return `errors.ErrUnsupported` without buffer changes.
The documented native errno limitation also applies to these calls.

`syscall.Socket` creates actual Linux/macOS socket descriptors through Dart FFI.
The caller owns each descriptor and must close it. SocketDisableIPv6 preserves
the SDK IPv6 rejection contract. The existing non-atomic errno limit applies.
Linux/macOS 64-bit programs use the selected SDK Cmsghdr types and constants.
`CmsgLen`, `CmsgSpace`, `UnixRights`, `ParseSocketControlMessage` and
`ParseUnixRights` preserve native header lengths, alignment and byte order.
The codecs validate external control-message header lengths and identities.
They do not validate descriptor ownership. Actual sendmsg/recvmsg operations
transfer descriptors, and the receiver owns each received descriptor.

Linux 64-bit programs support `syscall.ParseNetlinkMessage` with the selected
SDK `NlMsghdr` type and native byte order. The parser preserves payload
aliases, slice capacity, header value copies, and 4-byte message alignment.
Malformed lengths and missing alignment bytes return typed `EINVAL` and a
nil result. This byte parser does not execute kernel route queries.

`syscall.ParseNetlinkRouteAttr` supports the 6 SDK link, address, and route
message types on 64-bit Linux. It preserves native byte order, payload
aliases, slice capacity, and attribute header copies. Short message prefixes
and incomplete attribute alignment retain SDK slice panics. Other malformed
attribute lengths return typed `EINVAL` and a nil result.

`syscall.SockaddrUnix` and `Sendto` use actual native Unix destination bytes.
The caller owns the raw descriptor. These calls do not supply raw polling,
deadlines or descriptor inheritance. A blocking raw descriptor can stop the
Dart scheduler. Other socket address types remain unsupported.

Linux 64-bit programs support raw `syscall.Openat` through Dart FFI.
The caller owns the descriptor. Relative paths retain the directory
descriptor identity after a rename. Native flags, modes, raw path bytes,
and typed errors follow the selected SDK. Web targets return `ENOSYS`.
The documented non-atomic errno limit also applies to this call.
This API does not supply polling or automatic nonblocking changes.

The original syscall error assertions retain the invalid Seek descriptor and
ENFILE temporary-error contract. Native VM/AOT preserve their SDK errors.

Unix datagram listeners accept an empty address and preserve the native
kernel result. Linux supplies an actual abstract autobind address. Darwin
returns EINVAL. A nil listener address retains the missing-address error.

`syscall.Read` and `syscall.Write` execute 1 actual native descriptor call on
Linux/macOS 64-bit targets. They preserve byte counts, file offsets and native
errors. The caller owns each descriptor. A blocking descriptor can stop the
scheduler. These calls provide no polling, deadlines or inheritance. The
existing non-atomic errno limitation also applies to these calls.

Unix datagram dials treat empty local and remote addresses as wildcards.
An empty local address remains unbound. A named local address can dial
without a peer. Dial errors retain the caller address metadata.

Native 64-bit Linux and macOS programs support `syscall.Socketpair`
through Dart FFI. The kernel creates independent caller-owned descriptors
with the actual socket type, protocol and flags. Error results retain the
SDK zero descriptor array. Web targets return `ENOSYS`.
These descriptors do not supply raw polling or process inheritance.
Blocking raw descriptors can stop the scheduler. The non-atomic errno
limit also applies.

Native 64-bit Linux and macOS programs support `syscall.GetsockoptInt`
and `syscall.SetsockoptInt` through Dart FFI. Both calls use the SDK
signed 32-bit value and 4-byte option length. The kernel supplies the
actual option state and typed errors. The caller owns the descriptor.
Web targets return `ENOSYS`. The non-atomic errno limit also applies.

Native 64-bit Linux and macOS programs also support
`syscall.GetsockoptInet4Addr` and `syscall.SetsockoptInet4Addr`.
These calls preserve the exact 4-byte SDK value and option length
through the same native integer socket-option calls. Returned arrays
are independent values. Typed errors and descriptor ownership remain.

Native 64-bit Linux and macOS programs support `syscall.GetsockoptIPMreq`
and `syscall.SetsockoptIPMreq` through Dart FFI. Both calls preserve the
SDK 8-byte request and exact address arrays. Getters return independent
values with initialized partial/error results. Nil setter pointers retain
the SDK length and actual kernel errors. The caller owns the descriptor.
Web targets return `ENOSYS`. The non-atomic errno limit applies.

Native 64-bit Linux programs support `syscall.GetsockoptIPMreqn`
and `syscall.SetsockoptIPMreqn` through Dart FFI. Both calls preserve
the SDK 12-byte request, exact address arrays and signed interface index.
Getters return independent values with initialized partial/error results.
Nil setter pointers retain the SDK length and actual kernel errors.
The caller owns the descriptor. Web targets return `ENOSYS`.
The non-atomic errno limit applies.

Native 64-bit Linux programs support `syscall.GetsockoptUcred` through Dart FFI.
The call preserves the SDK 12-byte value, signed PID and unsigned UID/GID.
The kernel supplies the fields. Returned pointers hold independent values
for partial option writes and initialized error results.
The caller owns the descriptor. Web targets return `ENOSYS`.
The non-atomic errno limit applies. SCM credential codecs and process
inheritance remain separate limits.

Native 64-bit macOS programs support `syscall.GetsockoptByte` through Dart FFI.
The call preserves the BSD SDK byte result and 1-byte option length,
including partial scalar values. Typed errors and descriptor ownership
remain. Web targets return `ENOSYS`. The non-atomic errno limit applies.
The selected Linux SDK does not expose this API.

Native 64-bit Linux and macOS programs support `syscall.SetsockoptByte`.
The call preserves the exact byte and SDK 1-byte option length through
the same native string socket-option call. NUL and high bytes remain
actual byte values. Typed errors and descriptor ownership remain.

`syscall.SetsockoptString` passes the exact string bytes and length to the
native Linux/macOS `setsockopt` call. An empty value uses a null pointer and
0 length. The caller retains descriptor ownership. The documented
non-atomic errno limit also applies to this call.

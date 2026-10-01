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
| Parallelism | No shared-memory Go execution across native threads. `GOMAXPROCS` and `NumCPU` report 1. |
| Type layout | `unsafe.Sizeof`, `unsafe.Alignof`, and `unsafe.Offsetof` use 64-bit Go type metadata, including generic types. They do not describe Dart object storage. |
| Compiler diagnostics | Native Go optimization messages and instruction selection are outside the Gart compiler contract. Source compilation and runtime behavior require separate coverage. |
| Native access | Nil `unsafe.Pointer` values and constant-zero conversions are supported. No cgo, native pointer arithmetic, or pointer reinterpretation. |
| Pointer output | Process-local identities can represent pointers. They are not native memory addresses. |
| Map iteration | The runtime can expose insertion order. Programs must not depend on Go's unspecified iteration order. |
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

Linux and macOS support `os.File.Fd` for regular files, directories, and
standard streams. The file owns the descriptor. Call `Close` explicitly.
Automatic file closure through Go finalizers remains unsupported.
Gart does not raise the inherited open-file limit at startup.
Programs can reach that limit sooner than native Go programs.
Directory creation on Linux and macOS applies permission bits and the process umask at creation.
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

`log/syslog` supports TCP, UDP, and Unix streams. Unix datagrams and local
daemons that require them remain unsupported.

On Linux and macOS, x64 and arm64 builds use native directory descriptors
for recursive removal. Other native hosts retain Dart filesystem adapters.

Linux and macOS programs read native user IDs, group IDs, supplementary
groups, parent process IDs, and memory page sizes. Node programs read real
and effective user IDs and group IDs when Node exposes these methods.
Node retains placeholders for the other numeric values and does not support
supplementary groups. Browser targets retain placeholders for all these
numeric values and do not support supplementary groups.

Linux and macOS expose network interface indexes, names, MTU values,
hardware addresses, and flags. The table includes interfaces without IP addresses.
Other native hosts report interface metadata as unsupported.
Address enumeration remains separate. Browsers return an empty interface table.

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
Mixed-family destinations, IPv6 zones, and socket controls remain incomplete.

Native UDP error codes remain unreliable across separate Dart FFI calls.
A Linux VM trace confirms that native `EAGAIN` can become `EINVAL` before Dart reads `errno`.
Thus, a UDP read can fail with an incorrect error.
[Dart issue 38832](https://github.com/dart-lang/sdk/issues/38832) describes this missing atomicity.

On Linux and macOS, TCP connection failures and socket-option failures retain available native error codes through `net.OpError` and `os.SyscallError`.
Closed sockets retain `net.ErrClosed`.

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

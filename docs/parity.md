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
| Native access | Nil `unsafe.Pointer` values and constant-zero conversions are supported. No cgo, native pointer arithmetic, or pointer reinterpretation. |
| Pointer output | Process-local identities can represent pointers. They are not native memory addresses. |
| Map iteration | The runtime can expose insertion order. Programs must not depend on Go's unspecified iteration order. |
| Reflection | Descriptors cover a substantial subset. Dynamically created function types need an emitted callable factory. |
| Function metadata | Named functions and closures expose synthetic source PCs. Reflection method values share a dispatch PC with no Go source location. Direct method values and generic function instances remain incomplete. |
| Garbage collection | Dart controls collection and cleanup timing. Native Go heap and collector controls do not transfer directly. |
| Runtime metrics | `runtime/metrics` exposes GODEBUG event counters and value accessors. Unknown metrics return `KindBad`. Native Go heap, scheduler, and collector measurements remain unavailable. |
| Deep calls | Saved frame chains resume iteratively and retain suspended ancestors. Initial synchronous recursion still uses the fixed Dart stack. |
| Linkname | Module variable aliases share storage with variables of the same type in imported packages. Module function aliases resolve to functions of the same type in the program import graph. Alias initializers, function alias bodies, and aliases to other aliases remain unsupported. |
| Stack traces | Go source positions, without native instruction offsets. `-trimpath` selects portable paths. Runtime initialization and panic frames use their Dart source locations. |
| Zero-size slices | Logical lengths and capacities retain all 64 bits on VM, AOT, JS, and Wasm. Zero-size elements use virtual storage. Large arrays remain separate. |
| Numbers | Target representations differ. JavaScript conversion boundaries need particular care. |
| Time | Monotonic values support elapsed-time operations. Time-zone behavior can differ from Go's own database. |
| Environment | `os` and `syscall` share environment values. Child processes receive this state through `os/exec`. |
| Source tools | `go/build` accepts the `gart` compiler and `dart64` suffixes without a native archive path. `runtime.GOROOT` reads the process-start `GOROOT` value. No SDK path travels with the Dart output. |
| GODEBUG | Executables use the Go toolchain's module, workspace, and source defaults. Environment values override these defaults for library readers and `panicnil`. Other runtime controls and bisect patterns remain incomplete. The FIPS module rejects non-default modes. |

The verifier reports unsupported source constructs with source positions.
The verifier tests define precise rejection cases. Runtime and target limits
can still produce errors after compilation.

## Platform boundaries

Dart VM and AOT programs use `dart:io` adapters for files, processes, sockets,
and HTTP. These adapters do not provide all native descriptor, process-group,
file metadata, TLS configuration, or runtime inspection APIs.

Linux and macOS support `os.File.Fd` for regular files, directories, and
standard streams. The file owns the descriptor. Call `Close` explicitly.
Automatic file closure through Go finalizers remains unsupported.
Directory creation on Linux and macOS applies permission bits and the process umask at creation.
Temporary directories request mode `0700`.
Directory descriptors retain their identity after a rename or removal.
`File.Chmod` and `File.Chown` use native descriptors on Linux and macOS.
`os.Chmod`, `Chown`, and `Lchown` preserve native errors and raw path bytes on these hosts.
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
Directory enumeration still uses paths.
Other native hosts and browsers do not support `os.File.Fd`.

`net/http/fcgi` supports supplied listeners. `Serve` with a nil listener
returns an explicit error because inherited listener descriptors remain unsupported.

Linux and macOS support `syscall.Kill` and `syscall.Getpid` through native APIs.
Signal watches currently support SIGHUP, SIGINT, SIGTERM, and SIGWINCH.
Other native signal watches remain unsupported. Browser builds cannot send
or receive operating system signals.

`log/syslog` supports TCP, UDP, and Unix streams. Unix datagrams and local
daemons that require them remain unsupported.

On Linux and macOS, x64 and arm64 builds use native directory descriptors
for recursive removal. Other native hosts retain Dart filesystem adapters.

Linux and macOS programs read native user IDs, group IDs, supplementary
groups, parent process IDs, and memory page sizes. Other targets retain
placeholder numeric values and report supplementary groups as unsupported.

Linux and macOS expose network interface indexes, names, MTU values,
hardware addresses, and flags. The table includes interfaces without IP addresses.
Other native hosts report interface metadata as unsupported.
Address enumeration remains separate. Browsers return an empty interface table.

On Linux and macOS, `net.DialUDP` supports an explicit local address and port.
Connected UDP sockets report the local address that the operating system selects.
UDP network-family selection and the context-based UDP dialer still have gaps.

Browser builds use browser APIs. HTTP clients use fetch. Browser restrictions
on headers, credentials, network access, and cross-origin requests still apply.
Native filesystem operations, raw sockets, and HTTP servers do not become
available in a browser merely because a program compiles.

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

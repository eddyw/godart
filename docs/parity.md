# Compatibility

Gart aims to preserve Go behavior within its supported subset.
A successful build does not establish complete Go compatibility.
Upstream tests establish parity only for their recorded inputs, targets,
SDK versions, modes, and exclusions.

## Language and runtime limits

| Area | Contract or limit |
|---|---|
| Language versions | Package and file versions govern language semantics. The installed SDK still supplies standard library code. |
| Goroutines | Cooperative execution in 1 Dart isolate. A busy loop without a yield can block other work. |
| Parallelism | No shared-memory Go execution across native threads. `GOMAXPROCS` and `NumCPU` report 1. |
| Type layout | Constant `unsafe.Sizeof`, `unsafe.Alignof`, and `unsafe.Offsetof` use source type metadata. They do not describe Dart object storage. |
| Native access | No cgo, native pointer arithmetic, or general `unsafe` operations. |
| Pointer output | Process-local identities can represent pointers. They are not native memory addresses. |
| Map iteration | The runtime can expose insertion order. Programs must not depend on Go's unspecified iteration order. |
| Reflection | Descriptors cover a substantial subset. Dynamically created function types need an emitted callable factory. |
| Garbage collection | Dart controls collection and cleanup timing. Native Go heap and collector controls do not transfer directly. |
| Stack traces | Go source positions, without native instruction offsets. `-trimpath` selects portable paths. |
| Numbers | Target representations differ. JavaScript conversion boundaries need particular care. |
| Time | Monotonic values support elapsed-time operations. Time-zone behavior can differ from Go's own database. |
| Environment | `os` and `syscall` share environment values. Child processes receive this state through `os/exec`. |
| GODEBUG | Executables use the Go toolchain's module, workspace, and source defaults. Environment values override these defaults for library readers and `panicnil`. Other runtime controls and bisect patterns remain incomplete. The FIPS module rejects non-default modes. |

The verifier reports unsupported source constructs with source positions.
The verifier tests define precise rejection cases. Runtime and target limits
can still produce errors after compilation.

## Platform boundaries

Dart VM and AOT programs use `dart:io` adapters for files, processes, sockets,
and HTTP. These adapters do not provide all native descriptor, process-group,
file metadata, TLS configuration, or runtime inspection APIs.

Browser builds use browser APIs. HTTP clients use fetch. Browser restrictions
on headers, credentials, network access, and cross-origin requests still apply.
Native filesystem operations, raw sockets, and HTTP servers do not become
available in a browser merely because a program compiles.

TLS uses Dart's secure socket implementation. A Go API name does not imply
control over all native Go TLS internals. The project has no security audit.
Read [SECURITY.md](../SECURITY.md) before a security-sensitive deployment.

## Tests and coverage

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

## Flutter

Glutter uses shared Go nodes, signals, and generated Dart host adapters.
Application-owned bindings cover selected constructors and scalar services.
They do not yet provide arbitrary Dart object interop, custom value classes,
all generic APIs, or the complete Flutter Canvas API.
See the [Flutter architecture](flutter-design.md) and
[binding guide](../flutter/nodegen/README.md) for precise boundaries.

# Upstream compile blockers

This inventory classifies the first known compile failure of each blocked package.
A repaired first failure can expose another dependency. A blocked suite is not a parity pass.

The blocker manifest retains exact platform diagnostics in the coverage manifest.

| Package | Cause | Required work or boundary |
|---|---|---|
| `cmp` | runtime-boundary | The suite imports unsafe and requires native memory operations outside the Dart contract. |
| `crypto/fips140` | private-test-dependency | The upstream helper configures native FIPS behavior through GODEBUG. |
| `crypto/internal/fips140/bigmod` | runtime-boundary | The suite calls architecture assembly helpers such as addMulVVW1024. Dart has no native assembly entry points. |
| `crypto/internal/fips140/nistec` | runtime-boundary | The suite requires native memory-page sizes and memory protection through syscall. |
| `crypto/internal/fips140cache` | private-test-dependency | The suite inspects a private cache field that the overlay does not retain. |
| `crypto/internal/sysrand` | private-test-dependency | The suite injects native random-source failures through private hooks and errno values. |
| `crypto/md5` | runtime-boundary | The suite imports unsafe and requires native memory operations outside the Dart contract. |
| `crypto/subtle` | runtime-boundary | The suite requires native CPU feature detection through internal/cpu. |
| `crypto/tls` | private-test-dependency | The suite calls private Go TLS implementation functions. Dart owns the TLS implementation. |
| `crypto/x509` | private-test-dependency | The suite requires the upstream external-network capability helper before further classification. |
| `encoding/asn1` | runtime-boundary | The suite requires Go allocation and heap statistics through runtime.MemStats. |
| `go/types` | runtime-boundary | The suite converts an unsafe address into a typed pointer. Dart has no native pointer conversion. |
| `hash/maphash` | runtime-boundary | The suite imports unsafe and requires native memory operations outside the Dart contract. |
| `html/template` | private-test-dependency | The suite requires the upstream GODEBUG configuration helper. |
| `image/gif` | runtime-boundary | The suite requires Go allocation and heap statistics through runtime.MemStats. |
| `internal/abi` | runtime-boundary | The suite requires native function program counters through abi.FuncPCABI0. |
| `internal/fmtsort` | runtime-boundary | The suite imports unsafe and requires native memory operations outside the Dart contract. |
| `internal/godebug` | runtime-boundary | The suite requires native Go runtime metrics. |
| `internal/reflectlite` | private-test-dependency | The overlay does not retain private reflectlite implementation names used by its tests. |
| `internal/testenv` | private-test-dependency | The complete native test infrastructure exceeds the selected testenv adapter. |
| `internal/types/errors` | private-test-dependency | The suite depends on go/importer, which remains outside the surface. |
| `log/slog` | runtime-boundary | The suite imports unsafe and requires native memory operations outside the Dart contract. |
| `math` | runtime-boundary | The suite imports unsafe and requires native memory operations outside the Dart contract. |
| `math/big` | runtime-boundary | The suite requires Go allocation and heap statistics through runtime.MemStats. |
| `net` | private-test-dependency | The suite calls the private native RFC6724 source-address sorter. |
| `net/http` | public-api-gap | The complete suite requires TLS root-certificate configuration. TLS expansion has separate scope. |
| `net/http/httptest` | public-api-gap | The complete suite requires TLS connection-version metadata. TLS expansion has separate scope. |
| `net/http/httputil` | runtime-boundary | The suite requires Go runtime profiling through runtime/pprof. |
| `net/smtp` | public-api-gap | The complete suite requires TLS connection-version metadata. TLS expansion has separate scope. |
| `net/textproto` | runtime-boundary | The suite requires Go allocation and heap statistics through runtime.MemStats. |
| `os` | public-api-gap | The complete suite requires native errno and file-descriptor behavior beyond the current surface. |
| `os/exec` | public-api-gap | The environment helper now matches upstream. The complete suite next requires native process signals and process attributes. |
| `os/signal` | private-test-dependency | The suite requires native subprocess test infrastructure and signal behavior. |
| `reflect` | runtime-boundary | The suite inspects native register ABI constants such as abi.IntArgRegs. |
| `runtime` | runtime-boundary | The suite requires Go finalizer semantics through runtime.SetFinalizer. |
| `runtime/debug` | runtime-boundary | The suite requires Go allocation and heap statistics through runtime.MemStats. |
| `sync` | runtime-boundary | The suite pins goroutines to native Go execution threads through runtime_procPin. |
| `sync/atomic` | runtime-boundary | The suite requires atomic operations on unsafe native pointers. |
| `syscall` | runtime-boundary | The suite requires raw native file descriptors through syscall.Open. |
| `testing` | private-test-dependency | The native harness tests depend on test-only exports and a different harness implementation. |
| `text/template` | runtime-boundary | The suite imports unsafe and requires native memory operations outside the Dart contract. |
| `time` | private-test-dependency | The complete native test graph creates an import cycle with the time overlay. Selected public assertions execute separately. |
| `unique` | private-test-dependency | The suite inspects the private Go weak canonicalization table. The overlay uses a process-lifetime map. |
| `weak` | runtime-boundary | The suite requires Go finalizer semantics through runtime.SetFinalizer. |

The classification identifies no independent compiler defect from these first diagnostics.
This does not establish the absence of compiler defects behind the first failure.

Prioritize missing public behavior that the Dart APIs can support. Preserve the native runtime boundaries.
Do not add empty private helpers, remove assertions, or treat excluded tests as passes.
TLS, GC, unsafe memory, and native thread facilities require separate scope.

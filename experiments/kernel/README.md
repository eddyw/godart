# Direct kernel prototype

This isolated experiment converts a narrow Gart IR subset directly into Dart Kernel.
It retains the Dart runtime and does not change normal `gart` commands.

## Pin and execution

The prototype requires Dart **3.13.3**, SDK revision **1d1a730ef918d602aedafc939a4cf5940e7589ab**.
The setup verifies both the installed SDK revision and the SDK source checkout.
The kernel writer embeds the SDK hash `1d1a730ef9`. The probe verifies rejection of a different hash.
A kernel file remains a VM input, not an AOT executable. Execute it with `dart run`.

Run the complete experiment from the repository root:

```sh
python3 experiments/kernel/probe.py
```

Requirements: the pinned Dart SDK, the project Go toolchain, Python 3.9 or later, Git, and Node.js with WasmGC support.
The verified Node.js version is 26.8.2.
The first run obtains SDK source from the official repository and resolves the existing Dart runtime dependencies.
The script uses `build/kernel-prototype/` for the SDK checkout, tools, kernels, native binaries, and results.
It creates an ignored `.dart_tool/package_config.json` for this experiment.
It does not dispatch GitHub Actions.

## Pipeline

1. Load and type-analyze a Go file or module with `go/packages`.
2. Reuse Gart verification, block analysis, and normalization.
3. Export the supported IR nodes as JSON.
4. Load a precompiled Dart runtime component.
5. Construct typed kernel nodes and resolve runtime references.
6. Serialize the component through the pinned `package:kernel` implementation.
7. Execute the component with `dart run`, or pass it to a compatible backend.

The Go adapter selects direct IR output, Dart source output, or both for the measurement scripts.
Both backends use the same selected Go declarations and normalized IR.
The direct backend does not invoke the Dart source emitter during complete-build measurements.
A probe deletes the application Dart files before another direct build and verifies the output.
The runtime component contains the real `goruntime` libraries. Builtin `println` calls the actual runtime helper.
The original `probe.py` runner rebuilds runtime components per invocation.
The imported-package experiment below adds an isolated content-addressed cache and invalidation probes.

## Supported subset

The entry package must be `main`. The frontend accepts a single Go file or a module with imports.
Local dependency packages can contain functions and constants within the supported subset.
The standard library selection contains 4 original `math/bits` functions, as described below.
Other standard library packages remain outside this prototype.

The prototype supports synchronous functions, direct calls, recursion, 64-bit integer/bool parameters and results, local variables, assignments, branches, and simple loops.
Integer operators include addition, subtraction, multiplication, comparisons, bitwise operations, and shifts.
VM integer operations use Dart integers and the existing runtime helpers for unsigned comparison, shifts, and unsigned text.
Builtin `println` accepts integers, booleans, and UTF-8 string literals.
Unsupported IR nodes fail explicitly. The float rejection probe verifies this path.

This is not the complete Gart backend. It excludes package state, named types, structs, interfaces, general strings, generics, goroutines, channels, defer, and panic recovery.
It also omits the normal entry-point scheduler wrapper and complete Go stack mapping.
The source baseline has empty Go source maps for this focused comparison.
The kernel retains Go file content and function positions, but debugger fidelity requires separate work.

VM and AOT probes include signed 64-bit overflow and integers above the JavaScript precision limit.
The prototype does not lower its application integer operations through GoInt on JavaScript.
Thus its JavaScript probes use only values within the exact integer range.
The WebAssembly probes currently use the same small-value selection. No large-integer WebAssembly claim follows from them.

## Kernel reuse across targets

| Input | Output | Local result with Dart 3.13.3 |
|---|---|---|
| VM kernel | `dart run` | Pass |
| VM kernel | `dart compile exe` | Pass, including direct prototype kernels |
| VM kernel | `dart compile aot-snapshot` | Pass for the portable Dart fixture |
| VM kernel | JavaScript | The portable fixture failed inside the compiler |
| VM kernel | Public `dart compile wasm` | Rejected because the default CFE phase expects Dart source |
| JavaScript-target kernel | `dart compile js` | Pass, including direct prototype kernels |
| WebAssembly-target kernel | Internal TFA/codegen phases | Pass, including direct prototype kernels |

A VM kernel is not a universal artifact for all backends.
Platform libraries, conditional imports, compiler transformations, and metadata depend on the target.
The direct prototype reuses the same application IR, but loads a separate runtime kernel for each target.

For VM-to-AOT conversion:

```sh
dart compile exe -o program program.dill
dart compile aot-snapshot -o program.aot program.dill
dartaotruntime program.aot
```

The kernel must include compatible platform libraries. The default `dart compile kernel` command includes them.

The JavaScript probe creates its runtime kernel with `dart compile js --cfe-only`.
It then passes the direct kernel to `dart compile js`.
The WebAssembly probe uses the SDK's `dart2wasm_product.snapshot` with `--phases=cfe` for its runtime component.
It uses `--phases=tfa,codegen` for the direct application kernel.
These compiler-stage options are internal integration points. The SDK pin applies to them too.

The SDK source establishes these input paths:

- [Kernel input in dart2js](https://github.com/dart-lang/sdk/blob/1d1a730ef918d602aedafc939a4cf5940e7589ab/pkg/compiler/lib/src/phase/load_kernel.dart)
- [WebAssembly phase input rules](https://github.com/dart-lang/sdk/blob/1d1a730ef918d602aedafc939a4cf5940e7589ab/pkg/dart2wasm/lib/compiler_options.dart)
- [VM kernel and AOT generation](https://github.com/dart-lang/sdk/blob/1d1a730ef918d602aedafc939a4cf5940e7589ab/pkg/dart2native/lib/generate.dart)

## Initial backend measurements, 2026-09-18

The focused validation passes:

- Native Go, existing Dart source output, direct VM kernel, and direct AOT output agree for all 3 fixtures.
- The small-value arithmetic and recursion fixtures pass JavaScript and WebAssembly execution through Node.js.
- The integer fixture covers overflow, large integers, nested variable scope, and zero values on VM and AOT.
- The frontend rejects unsupported float input.
- The VM rejects an artifact with a different SDK hash.
- Direct generation succeeds without application Dart source files.
- Dart analysis reports no issues for the prototype and runtime seed source.

The host is macOS ARM64 on an Apple M4 Max. All file and dependency caches remain warm.
Each row has 5 trials, with alternating backend order and fresh output writes.
The programs contain distinct arithmetic functions and compare the final result across both execution paths.

| Application functions | Direct kernel median | Dart source-to-kernel median |
|---:|---:|---:|
| 10 | 0.105 s | 0.231 s |
| 250 | 0.111 s | 0.246 s |
| 2500 | 0.136 s | 0.312 s |

The direct measurement includes process startup, runtime-kernel load, JSON decode, node construction, and full kernel serialization.
The source measurement includes `dart compile kernel` over the existing emitter output and Dart runtime source.
Both measurements exclude the common Go frontend, prepared IR/source generation, and program execution.
The result combines source bypass and runtime-component reuse. It does not isolate their individual contributions.

Initial helper compilation costs 2.08 seconds. Runtime kernel preparation costs 0.24 seconds for VM, 0.30 seconds for JavaScript, and 0.75 seconds for WebAssembly.
The direct path plus initial VM runtime preparation exceeds the source compiler time on the smallest fixture.
The current evidence establishes a repeated-build opportunity, not a first-build or whole-project speed guarantee.
`build/kernel-prototype/results.json` retains setup durations, individual trials, outcome comparisons, and the exact SDK identity.

## Imported-package experiment, 2026-09-18

Run the complete experiment from the repository root:

```sh
python3 experiments/kernel/imported.py
```

The workload imports `math/bits` through a local Go module and a transitive settings package.
It combines these original Go functions in a deterministic digest loop:

- `OnesCount64`
- `Reverse64`
- `ReverseBytes64`
- `RotateLeft64`

The frontend selects these 4 original function bodies after Go type analysis.
It preserves their statements and resolves original package constants through Go type information.
Both backends receive the same selection. The direct path does not gain an advantage through a smaller function selection.
This proves selected-function behavior, not complete `math/bits` support or upstream test-suite coverage.
The native Go reference can use compiler intrinsics for these functions. The prototype emits their original Go bodies.

The fixture covers 0, 1, the high bit, the maximum unsigned value, a mixed bit pattern, and 256 deterministic values.
It exercises rotation counts from -65 through 65, overflow, signed conversions, unsigned comparisons, and shifts at or above 64 bits.
Native Go, the source kernel, the direct kernel, and direct AOT output agree across 936 output lines.
The imported experiment targets the VM and AOT. It makes no new JavaScript or WebAssembly parity claim.

### Cache identity and invalidation

The script creates disposable copies under `build/kernel-prototype/imported/`.
It never changes the project runtime, the original fixture, or the shared pub cache during mutation probes.
`results.json` retains toolchain identity, Go source hashes, individual trials, selected functions, phase durations, and outcomes.

A build request discovers the Go dependency graph and hashes dependency content before cache reuse.
The final kernel identity includes:

- Effective Go build environment, dependency paths, imports, source bytes, and module metadata.
- Dart SDK revision, target, runtime source, transitive Dart source, package configuration, and compiler defines.
- Frontend and kernel-helper executable bytes, backend choice, and output directory.

The direct runtime component has a separate identity. A Go edit reuses that component.
A runtime, transitive Dart dependency, package configuration, or compiler-option edit rebuilds the component.
The source backend also caches its final kernel. Thus unchanged requests compare equivalent artifact reuse.
Both caches verify artifact hashes and receipt structure before reuse.
A mismatched installed SDK fails before cached execution.

All 18 cache scenarios and the separate SDK-rejection probe pass.
The invalidation probes cover:

- Application content changes with identical size and modification time.
- A transitive Go dependency edit and a new dependency source file.
- Removal of that file, which restores a previously known dependency graph and reuses its valid artifact.
- Go module metadata and Dart package configuration changes.
- Runtime source and transitive Dart dependency changes.
- Compiler defines, repeated requests, and restoration of the original inputs.
- Corrupt or missing kernels, corrupt runtime components, and malformed or incomplete receipts.

Runtime and compiler-define probes deliberately add output prefixes in disposable copies.
Their expected output contains those prefixes. They establish invalidation and execution of changed code, not Go parity for the altered runtime.

The cache assumes serial requests and stable source files during each request.
It does not implement file-watch coordination, concurrent edits, eviction, package-level kernel reuse, or production driver integration.
It hashes all Dart library sources in resolved packages. This conservative rule can invalidate an unused file.
Go discovery occurs before a hit, and the Go frontend loads packages again after a miss.

### Complete request measurements

The host is Apple M4 Max, macOS ARM64, Go 1.27.0, and Dart 3.13.3.
Each scenario has 5 trials with alternating backend order and warm filesystem/toolchain caches.
The clean-artifact scenario removes application and runtime kernel caches before each trial.

| Scenario | Direct build | Source build | Direct build + execution | Source build + execution |
|---|---:|---:|---:|---:|
| No cached artifacts | 672 ms | 551 ms | 753 ms | 634 ms |
| Unchanged cache hit | 234 ms | 243 ms | 313 ms | 326 ms |
| Application edit | 417 ms | 559 ms | 494 ms | 643 ms |
| Transitive Go edit | 425 ms | 556 ms | 501 ms | 641 ms |

These are complete prototype requests from dependency discovery through artifact validation and publication.
The direct request includes runtime compilation when its component is absent.
Both paths include frontend process startup, package load, type analysis, verification, normalization, and their own backend work after a miss.
The direct frontend writes JSON only. The source frontend invokes the existing Gart source emitter.
The execution columns also include `dart run` startup and the workload itself.
Each column reports its own median. A sum of separate phase medians need not equal the total median.

The table excludes helper compilation, dependency installation, initial SDK download, and the Python runner startup.
This run spent 0.12 seconds on the Go helper and 2.03 seconds on the Dart helper.
It does not measure normal `gart` CLI behavior, production overlay discovery, formatting, or complete source-map work.

For an application edit, the direct build spends approximately:

| Phase | Median |
|---|---:|
| Dependency discovery and content hashes | 233 ms |
| Go frontend process | 68 ms |
| Direct kernel helper | 106 ms |
| Artifact validation and publication | Included in the total |

Go dependency discovery accounts for approximately 114 ms of cache validation.
The conservative Dart dependency scan adds approximately 87 ms, including resolved development packages.
The frontend spends approximately 62 ms on package load and verification.
The source frontend spends approximately 1.4 ms on source emission and writes.
Its Dart source-to-kernel compiler takes approximately 242 ms.
These results point to dependency-discovery reuse and Dart compilation before further Go source-emitter optimization.

The complete edit rebuilds take approximately 24–25% less time with the direct backend.
Build plus execution improves by approximately 22–23%.
The clean-artifact direct build takes approximately 22% more time because it must compile the runtime component.
The approximately 2.2x backend-only result does not translate into an equivalent complete-build speedup.


## Decision

Direct kernel generation provides a measurable rebuild benefit for this selected imported workload.
The complete request benefit is smaller than the initial backend-only result.
Runtime preparation makes a clean-artifact build slower, and both backends perform similarly on an unchanged cache hit.

Prioritize a production source-kernel cache and dependency-discovery reuse before a complete kernel backend.
Retain this prototype as evidence for a later backend decision.
Further direct-backend work should address package state, richer data types, source maps, scheduler semantics, and target-specific integers.
The experiment does not establish performance for large application packages or production startup behavior.

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

1. Parse and type-analyze 1 Go file.
2. Reuse Gart verification, block analysis, and normalization.
3. Export the supported IR nodes as JSON.
4. Load a precompiled Dart runtime component.
5. Construct typed kernel nodes and resolve runtime references.
6. Serialize the component through the pinned `package:kernel` implementation.
7. Execute the component with `dart run`, or pass it to a compatible backend.

The Go adapter also invokes the existing Dart source emitter to create the comparison baseline.
The direct backend does not consume those Dart files.
A probe deletes the application Dart files before another direct build and verifies the output.
The runtime component contains the real `goruntime` libraries. Builtin `println` calls the actual runtime helper.
Each probe run rebuilds the runtime components. This experiment does not implement persistent cache invalidation.

## Supported subset

The input must contain `package main` and no imports.
The prototype supports synchronous functions, direct calls, recursion, int/bool parameters and results, local variables, assignments, branches, and simple loops.
Integer operators include addition, subtraction, multiplication, and comparisons.
Builtin `println` accepts integers, booleans, and UTF-8 string literals.
Unsupported IR nodes fail explicitly. The float rejection probe verifies this path.

This is not the complete Gart backend. It does not implement structs, interfaces, imports, general strings, generics, goroutines, channels, defer, or panic recovery.
It also omits the normal entry-point scheduler wrapper and complete Go stack mapping.
The source baseline has an empty Go source map for this focused comparison.
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

## Validation and measurements, 2026-09-18

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

## Next decision

The prototype establishes direct generation and target-specific kernel reuse for the stated subset.
Before expansion, compare a representative imported Go package and measure dependency changes with correct runtime invalidation.
A production backend also needs the complete Go semantic transformations, source mapping, scheduler entry points, and target integer models.
Keep the source backend until those requirements pass the existing parity contract.

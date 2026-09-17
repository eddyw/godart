# Compilation time

Use `-timings` to report compiler phases on stderr:

```sh
gart run -timings -o ./output ./...
```

The flag applies to build, run, test, and watch. Program output retains its existing stream behavior.
Each line gives the phase and elapsed seconds. The test harness also records these phases.

| Phase | Work |
|---|---|
| `load` | Total source load |
| `load-discovery` | Go package and file discovery |
| `load-surface` | Standard library overlay discovery |
| `load-overlay` | Overlay construction, embed rewrites, and test entry points |
| `load-types` | Go package reload, source parsing, and type analysis |
| `verify` | Supported-subset verification |
| `analyze` | Block analysis |
| `emit` | Cache lookup, normalization, Dart generation, and metadata |
| `write` | Runtime resolution and output files |
| `format` | External `dart format` command |
| `source-maps` | Source maps |
| `pub` | Dependency configuration, including `dart pub get` when required |
| `kernel-cache` | SDK identity and compiled-kernel cache validation |
| `kernel-compile` | Dart source compilation on a kernel cache miss |
| `execute` | Dart VM startup, JIT work, and program execution |

The 4 detailed load phases are parts of `load`. Do not add them to `load` again.
`load-types` does not separate parser CPU time from type analysis or the Go subprocess.
`execute` does not identify application execution alone.

## Repeatable probes

Build and measure a deterministic fixture:

```sh
python3 scripts/profile-compile.py --program hello --output build/profile-hello
python3 scripts/profile-compile.py --program jsonv2use --output build/profile-json
python3 scripts/profile-compile.py --program httpwire --output build/profile-http
```

The script records 3 trials by default. Each trial has a fresh Gart output, a warm repeat, kernel compilation, kernel execution, and direct source execution.
It preserves raw output and requires equal application output across the execution forms.
Cold means a fresh Gart output directory. Go compilation, filesystem, and Pub download caches remain warm.
The script does not clear global caches or use a network request as application work.
Kernel execution includes VM startup and JIT work. The source/kernel difference is an estimate, not a direct compiler timer.

For compiler CPU samples:

```sh
go test -count=3 -run '^TestPrograms$/^jsonv2use$' -cpuprofile build/compiler-cpu.out -o build/profile-driver.test ./internal/driver
go tool pprof -top -cum build/compiler-cpu.out
```

These CPU samples cover the Go test process. They exclude child Go and Dart processes.

## Measured baseline, 2026-09-17

The host uses an Apple M4 Max, Go 1.27.0, and Dart 3.13.3.
The probes use `GOMAXPROCS=4` and `GOFLAGS=-p=2`.
The table gives medians from 3 trials, in seconds. Independent medians do not necessarily sum to the total.
The Linux preflight overlapped the first probes. These measurements identify major costs, but do not establish precise optimization ratios.

| Fixture | Cold total | Load | Emit | Format | Dart execute | Warm total | Warm Dart execute |
|---|---:|---:|---:|---:|---:|---:|---:|
| `hello` | 1.911 | 0.466 | 0.007 | 0.031 | 0.631 | 1.261 | 0.597 |
| `jsonv2use` | 9.307 | 0.590 | 0.798 | 4.881 | 2.438 | 3.206 | 2.386 |
| `httpwire` | 16.483 | 0.649 | 3.818 | 6.029 | 5.058 | 6.561 | 4.973 |

| Fixture | Direct Dart source | Kernel compilation | Kernel execution |
|---|---:|---:|---:|
| `hello` | 0.529 | 0.252 | 0.062 |
| `jsonv2use` | 2.323 | 1.041 | 0.287 |
| `httpwire` | 4.900 | 2.605 | 0.582 |

The JSON Go CPU profile contains 5.55 seconds of samples across 3 fixture runs.
Cumulative parser samples total 0.03 seconds. Type analysis totals 0.09 seconds. The driver compile function totals 0.73 seconds.
Background GC has substantial samples. Cumulative rows overlap and cannot form an additive budget.
The profile excludes child-process work and elapsed wait time.

## Production cache, 2026-09-18

`gart run` and ordinary test executions now compile generated Dart to a kernel and reuse that kernel when its inputs match.
`kernel-compile` appears only on a cache miss. `kernel-cache` includes content hashes and SDK identity validation.
The driver retains non-product constants and assertion support. Benchmark tests retain the existing AOT path.

The kernel receipt covers all compiler depfile inputs, the resolved package configuration, compiler options, SDK identity, and artifact contents.
A missing, changed, or corrupt input causes recompilation. Content hashes detect edits that preserve file sizes and timestamps.
The cache uses `.gart/kernel/` beneath the output directory.

The loader reuses native overlay file metadata under `.gart/load/surface.json` and within each process.
Its identity covers the Go environment and production Go files in the overlay package directories.
Generated native inputs outside GOROOT have separate hashes.
Application package discovery remains fresh because new imports and files can change the dependency graph.
External Go overlays and custom package drivers bypass metadata reuse.

The internal driver option `NoCache` bypasses cache reads. Separate concurrent builds need separate output directories.
A source edit during compilation prevents receipt publication when its timestamp exceeds the compilation start.
This guard does not provide an atomic filesystem snapshot across concurrent edits.

For a development run without Dart formatting:

```sh
gart run -timings -format=false -o ./output ./...
```

## Production cache measurements

The host uses an Apple M4 Max, Go 1.27.0, and Dart 3.13.3.
These medians cover 3 alternating-order trials against commit `ceb3877`, with `GOMAXPROCS=4`, `GOFLAGS=-p=2`, and `-format=false`.
No test suite overlapped these trials.
Cold means a fresh output directory with warm Go and Pub caches.
Each edit adds a real print statement to the Go entry point.
All 54 executions matched native Go output for the corresponding source.

| Fixture | Cold before | Cold after | Warm before | Warm after | Edit before | Edit after |
|---|---:|---:|---:|---:|---:|---:|
| `hello` | 1.304 s | 1.350 s | 1.025 s | 0.558 s | 0.988 s | 0.821 s |
| `jsonv2use` | 4.155 s | 3.337 s | 3.019 s | 1.041 s | 3.058 s | 2.090 s |
| `httpwire` | 10.944 s | 9.156 s | 6.632 s | 2.233 s | 6.377 s | 5.073 s |

Warm total time fell by 45–66%. Edit total time fell by 17–32%.
The cold `hello` median rose by 3.5%. Initial metadata validation adds work that larger fixtures offset through faster Dart compilation.
Warm overlay discovery fell from 0.189 to 0.129 seconds for `hello`, 0.119 to 0.098 for JSON, and 0.146 to 0.102 for HTTP.
Content validation still costs time. These results do not imply complete reuse of application package discovery.

The focused regression tests cover source and transitive edits, preserved timestamps, package configuration, SDK identity, corrupt artifacts, and edits during compilation.
Execution tests cover arguments, standard input/output, exit codes, working directories, assertions, non-product constants, and Go panic paths.
The upstream cache regression compares cold and warm kernels with native Go for `path`, `io`, and selected `fmt` assertions.
The `fmt` selection covers `TestErrorf`, `TestSprintf`, `TestScanf`, and `TestScan`.
The existing exclusion for `fmt.TestScanInts` still applies because 800 nested scanners exceed the Dart stack limit.
A manual generated-program probe also verifies debugger breakpoints, resume, SIGINT, and SIGTERM with source and kernel execution.

The next performance targets are HTTP emission metadata and source edits that still invoke the full Dart kernel compiler.
The experimental direct-kernel backend remains separate.

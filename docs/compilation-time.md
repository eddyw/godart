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
| `kernel-compile` | Input snapshots, resident compilation, validation, and kernel publication on a cache miss |
| `kernel-cold` | Cache-miss work through a fresh compiler response |
| `kernel-incremental` | Cache-miss work through an incremental compiler response |
| `execute` | Dart VM startup, JIT work, and program execution |

The 4 detailed load phases are parts of `load`. Do not add them to `load` again.
`kernel-cold` and `kernel-incremental` are details within `kernel-compile`. Do not add them to the total again.
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

The kernel receipt covers compiler-reported source dependencies, package configuration, compiler options, SDK identity, and artifact contents.
A missing, changed, or corrupt input causes recompilation. Content hashes detect edits that preserve file sizes and timestamps.
The cache uses `.gart/kernel/` beneath the output directory.

The loader reuses native overlay file metadata under `.gart/load/surface.json` and within each process.
Its identity covers the Go environment and production Go files in the overlay package directories.
Generated native inputs outside GOROOT have separate hashes.
Application package discovery remains fresh because new imports and files can change the dependency graph.
External Go overlays and custom package drivers bypass metadata reuse.

The internal driver option `NoCache` bypasses cache reads. Separate concurrent builds need separate output directories.
Content hashes before and after compilation detect source edits during compilation, including edits with unchanged timestamps.
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

The initial cache measurements identified HTTP emission metadata and full Dart compilation after source edits as the next performance targets.
The experimental direct-kernel backend remains separate.

## Resident compiler comparison, 2026-09-18

The resident compiler experiment compares complete requests with identical Go frontend work on a playground-derived fixture.
It uses a local HTTP endpoint, a fixed clock, and a no-HTTP control.
Each mode has 5 trials with a fresh output, an unchanged repeat, and a real source edit.

| Workload after an edit | Dart source path | Current kernel cache | Resident compiler |
|---|---:|---:|---:|
| HTTP | 8.135 s | 5.810 s | 2.790 s |
| No HTTP | 3.637 s | 2.326 s | 1.085 s |

The resident compiler reduces the edit total by 52–53% against the current kernel cache in these fixtures.
Cold and warm differences are smaller. The no-HTTP warm median is 8% slower with the resident compiler.
Desktop background load remained active, and no test suite overlapped the measurements.
All 90 measured outputs match native Go after pointer normalization. All 10 private resident compilers shut down successfully.

These initial measurements led to a content-validated prototype and the production integration below.
The [experiment documentation](../experiments/resident/README.md) contains the complete table, reproduction command, limitations, and raw trial durations.

## Production resident compiler

The driver now uses a persistent Dart compiler service on a kernel cache miss.
Separate `gart run` and ordinary `gart test` commands can reuse frontend state.
The service retains at most 2 compilers per SDK/service identity and serves requests serially.
A third entry point replaces the least recently used compiler.
Output locks prevent concurrent commands from changing the same generated package.

The service uses a private directory under `/tmp`, with a namespace derived from the system temporary directory.
This short path also supports long `TMPDIR` values within the Unix socket path limit.
Directory ownership and permissions restrict access to the current user.
A startup lock prevents duplicate service creation across CLI processes.
The service starts a new process session, thus application signals do not terminate its retained compiler.
Compiler or service death causes a cold restart on the next request.
A compile error fails the command and preserves the previous valid receipt without executing it.

The cache-hit path hashes exact dependencies and the artifact. It does not scan all package sources or contact the service.
The receipt has a checksum that detects partial receipt corruption.
On a cache miss, a source snapshot covers generated Dart and resolved package library roots.
The snapshot retains URI paths and resolved paths, thus preserved timestamps and runtime symlink retargets cause invalidation.
The service sends explicit changed URIs through the Dart frontend protocol.
It resets the delta base after success and emits a complete kernel for a fresh application process.

SDK binaries, platform data, package configuration, fixed compiler options, and service source contribute to cache identity.
A changed SDK or package configuration requires a cold compiler.
The driver requires Dart 3.13.3 because the frontend protocol is internal to that SDK.
The current service supports macOS and Linux.
Untracked source dependencies cause a clear error. Arbitrary external Dart imports and directory symlinks outside the source snapshot remain unsupported.

Before each compile request, the service releases compilers with at least 60 idle seconds.
It also releases a compiler after 100 incremental edits.
It also releases a compiler when post-compile RSS exceeds 1 GiB or the memory measurement is unavailable.
Linux reads RSS from `/proc`. macOS uses `ps`.
These are retention limits. A compile can exceed 1 GiB before the service releases its compiler.
The 2-compiler limit excludes application memory and the service process itself.
The service also exits after 60 idle seconds. Monotonic clocks control both idle limits.

The HTTP prototype completed 100 consecutive edits with a peak compiler RSS of 672.656 MiB.
All 4 edit-quartile medians were 633.969 MiB. All 101 program executions matched native Go output.
The fixture uses a fixed clock and a local empty HTTP response.
Later samples overlapped development commands. Their compile durations do not establish isolated performance results.
This measurement supports the retention policy for that fixture, not a universal compiler memory bound.
[Memory evidence](../experiments/resident/memory-results-2026-09-18.json) retains all samples.

Reproduce memory, production performance, and Go frontend profiles with:

```sh
python3 -B experiments/resident/memory_probe.py --output build/resident-memory
python3 -B experiments/resident/production_probe.py --baseline /path/to/previous/gart --output build/resident-production
python3 -B experiments/resident/profile_frontend.py --output build/resident-frontend
```

The production comparison uses actual CLI commands for both binaries and includes complete requests.
It uses 5 alternating-order trials, fresh private services, local HTTP output, and native Go output comparisons.
The frontend profile uses a fresh Go process for each request and retains CPU profiles separately from phase durations.
Its CPU profiles exclude child Go and Dart processes.

The resident parity regression compares native Go with cold, cached, and incremental kernels.
It covers `path`, `io`, `math/bits`, `hash/crc32`, and the existing selected `fmt` assertions.
Each state has 179 successful upstream outcomes and 5 explicit exclusions across those packages.
Routine CI applies the same short-mode flag to native Go and Gart. The manual full mode remains available.
The production contracts also cover structural Go edits, debugger breakpoints, application signals, concurrent output ownership, and compiler/service crash recovery.
These results establish parity for the selected assertions. They do not establish complete Go compatibility.


## Production resident results, 2026-09-18

These medians cover 5 alternating-order trials against `4de8724` on the same Apple M4 Max.
Both modes use actual `gart run` commands, `-format=false`, `GOMAXPROCS=4`, and `GOFLAGS=-p=2`.
Cold requests use fresh outputs and services with warm Go, Pub, and filesystem caches.
No test suite overlapped these measurements. Desktop background load remained active.

| Workload | State | Previous kernel cache | Resident compiler |
|---|---|---:|---:|
| HTTP | Cold | 10.391 s | 10.762 s |
| HTTP | Unchanged | 2.429 s | 2.361 s |
| HTTP | Edit | 5.559 s | 2.740 s |
| No HTTP | Cold | 3.858 s | 3.709 s |
| No HTTP | Unchanged | 1.051 s | 0.981 s |
| No HTTP | Edit | 2.115 s | 1.136 s |

Edit totals fall by 51% with HTTP and 46% without HTTP.
The edit Dart portion falls from 3.799 to 0.998 seconds with HTTP and from 1.391 to 0.412 seconds without HTTP.
Unchanged medians improve by 3% and 7%. These small differences do not establish a general speed guarantee.
The unchanged path now uses exact dependency receipts and avoids the prototype's full source scan.
It also reads SDK metadata directly without a `dart --version` subprocess.

Cold Dart work rises in both fixtures because a fresh request also starts the service.
The HTTP cold total rises by 0.371 seconds. Frontend variation offsets that cost in the smaller fixture.
These results support faster edit cycles while the compiler remains available. They do not establish a general cold-start improvement.

All 60 measured outputs match native Go after pointer normalization.
The local server records all 32 expected HTTP requests, including the native references.
All 10 private services terminate after their trials.
[The production evidence](../experiments/resident/production-results-2026-09-18.json) retains raw trial durations, phase durations, source hashes, and binary hashes.

## Remaining Go frontend cost

A separate profile uses fresh Go processes for 1 cold build, 3 unchanged builds, and 3 source edits of the HTTP fixture.
The unchanged builds reuse all 128 packages. Each source edit recompiles 1 package and reuses 127 packages.
The profile excludes application execution and dependency setup.

| Median phase | Unchanged | Edit |
|---|---:|---:|
| Complete frontend request | 1.626 s | 1.793 s |
| Load | 0.626 s | 0.616 s |
| Type load, within load | 0.375 s | 0.380 s |
| Emit phase | 0.734 s | 0.876 s |
| Analyze | 0.093 s | 0.097 s |
| Write | 0.002 s | 0.002 s |

The warm emit phase includes cache validation and dependency classification.
`driver.compiled` calls `overlay.Surface` during package order and dependency hash work.
`Surface` calls `Paths`, which calls `Packages` twice. Each call scans and copies embedded overlay files.
This work repeats even when all generated packages remain cached.
`overlay.Packages` accounts for 16.1% cumulative CPU in the combined warm/edit profile.
`runtime.gcDrain` accounts for 39.9% cumulative CPU. The profile shows substantial allocation and garbage-collection cost.
Cumulative nodes overlap. Do not add their percentages.
CPU samples exclude child Go and Dart processes, while phase durations include their waits.

The next optimization should reuse the immutable overlay package paths for membership lookups, then measure the allocation cost again.
Subsequent work can target cached metadata reads and the remaining type-load cost if the new profile supports it.
[The frontend evidence](../experiments/resident/frontend-results-2026-09-18.json) contains all phase samples and flat, cumulative, and project CPU views.
The raw profiles remain local under `build/resident-frontend-20260918/` because they contain checkout paths.

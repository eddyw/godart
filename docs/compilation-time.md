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
| `execute` | Dart process startup, source compilation, and program execution |

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

## Recommended order

1. Avoid Dart formatting during development runs when readable generated files are unnecessary. The existing `-format=false` flag already permits this.
2. Investigate a generated-kernel cache for repeat runs. Key it by generated sources, runtime, package configuration, Dart SDK, and relevant flags.
3. Profile HTTP generation and its cache metadata. The HTTP warm build still spends about 0.67 seconds in the combined emit phase.
4. Reduce repeated Go package discovery where a complete invalidation contract permits reuse.
5. Defer parser-specific optimization. The present evidence identifies larger costs elsewhere.

The kernel measurements show potential. They do not validate a persistent kernel cache or prove its invalidation behavior.
This batch adds measurements and diagnosis. It does not change execution formats or introduce a compiled-artifact cache.

For a current development run:

```sh
gart run -timings -format=false -o ./output ./...
```

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

## Interpretation

Compare complete cold, unchanged, and edited runs with identical application work.
Record SDK versions, host, outputs, and background load.
A kernel compile plus execution can differ from direct source execution.
An unchanged kernel avoids compilation, but still includes VM startup and JIT work.
Do not infer a universal speedup from a single timing.

Resident execution retains a bounded compiler service. Cold execution and source
fallback remain relevant. The [SDK guide](sdk-upgrades.md) defines their selection.
Keep dated measurement output beside its experiment, not as a second changelog.

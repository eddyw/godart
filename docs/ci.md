# Continuous integration

The CI toolchain uses Go 1.27.0 and Dart 3.13.3.
The JavaScript and WebAssembly tests use Node.js 26.8.2.
GitHub Actions uses Ubuntu 24.04 on AMD64. The final local baseline uses Linux/AMD64 containers under emulation.

The workflow executes the current `make ci` test selection across 13 jobs:

- 1 job runs format validation, vet, Flutter Go tests, Dart runtime tests, and compiler tests.
- 4 jobs run all 376 custom Go/Dart parity programs.
- 4 jobs run all 931 discovered upstream Go language cases, with explicit exclusions.
- 4 jobs run the 37 current GOROOT package suites, with their existing filters and exclusions.

The workflow preserves the current coverage scope. It does not establish complete upstream Go parity.
Native memory-page tests, allocator tests, and other documented exclusions do not count as parity passes.
The 2 math fixtures also mark 15 host-dependent subnormal result comparisons. See [support limits](support.md).

## Local commands

Resolve dependencies first:

```bash
make get
```

Run the full local selection:

```bash
make ci
```

Run 1 complete CI suite locally:

```bash
GOMAXPROCS=2 GOFLAGS=-p=2 bash scripts/ci.sh programs
```

Run the same shard as a GitHub Actions job:

```bash
GOMAXPROCS=2 GOFLAGS=-p=2 bash scripts/ci.sh programs 0 4
```

The suite names are `checks`, `programs`, `language`, and `stdlib`.
Shard indexes start at 0. Without shard arguments, the command runs the complete named suite.
The driver assigns each case to 1 shard through a stable hash of its name.
The default Go test flags select all shards as 1 complete suite.

All Go CI tests use `-count=1`. Go compilation and dependency caches can remain warm.
The wrapper reports elapsed seconds and preserves the command exit status.
GitHub Actions retains the log for 7 days and reports the duration in the job summary.
A new run cancels a superseded run for the same branch or pull request.

## Duration baseline

The baseline date is 2026-09-16. The host used an Apple M4 Max with 16 cores.
The Docker VM has 4 CPUs. Each Linux/AMD64 container has a 2-CPU limit.
The baseline includes concurrent containers and reuses compilation and dependency caches.
Each job also sets `GOMAXPROCS=2` and `GOFLAGS=-p=2`. All test results are uncached.
These limits cover Go and Dart child processes together.
The filesystem permission suite uses an unprivileged container user, as GitHub Actions does.
A root process bypasses the permission errors that these upstream tests require.
The current Dart `os.Getuid` reports `-1`, which cannot activate the upstream root-user skip.
The earlier Mac baseline limited only Go workers and produced insufficient GitHub timeout limits.

| Suite | Shard | Local duration | Test step limit | Job limit |
|---|---|---|---|---|
| checks | all | 1410s | 47 min | 52 min |
| programs | 0 | 1423s | 48 min | 53 min |
| programs | 1 | 903s | 31 min | 36 min |
| programs | 2 | 970s | 33 min | 38 min |
| programs | 3 | 1280s | 43 min | 48 min |
| language | 0 | 1139s | 38 min | 43 min |
| language | 1 | 883s | 30 min | 35 min |
| language | 2 | 856s | 29 min | 34 min |
| language | 3 | 706s | 24 min | 29 min |
| stdlib | 0 | 1428s | 48 min | 53 min |
| stdlib | 1 | 137s | 5 min | 10 min |
| stdlib | 2 | 151s | 6 min | 11 min |
| stdlib | 3 | 166s | 6 min | 11 min |

The baseline retains complete shard traversals, including failures that exposed Linux portability defects.
Focused replays verify the corrected fixtures. Program shard 0 also has a complete corrected replay.
The final GitHub run validates the complete corrected selection.
The longest local case is `compress/flate`, at 1406.11 seconds within its 1428-second shard.

Each test step uses twice its measured local duration, with whole-minute rounding.
Job limits allow 5 additional minutes for toolchain setup and log upload.
Suite jobs compile the driver test binary before the timed test step.
This step matches the local compilation-cache state and exposes a cold build as a separate CI step.
`GART_TEST_TIMEOUT` controls the Go test timeout for a manual wrapper run.

The Linux/AMD64 `fixedbugs/issue13169` stress case completed in 129.35 seconds in the CPU-limited local matrix.
Its generated program has a 5-minute limit, which follows the same doubled-duration rule.
Other generated language programs retain their 2-minute limit.

## Platform exclusions

The language harness reads upstream build constraints before compilation.
A source file that excludes the host platform reports an explicit skip.
A supported host still requires the exact recorded rejection for unsupported language or runtime features.

`bytes/boundary_test.go` requires `mmap` and protected native memory pages.
The Dart runtime exposes neither API. All 4 upstream boundary tests report explicit skips.
The overlay changes only their shared allocation helper. It preserves the test assertions.

`fixedbugs/issue10607` exercises native Go linker flags and requires a companion source outside the single-file harness.
It reports a fixture exclusion. It does not count as a successful parity case.

Local validation covers all 95 root Go test entry points.
The 4 program shards execute all 376 cases exactly once.
The Linux language shards execute all 931 cases exactly once: 775 pass and 156 report explicit exclusions.
All 37 declared GOROOT package suites pass across their 4 shards.
The runtime tests, Flutter Go tests, format validation, vet, and Dart analyzer also pass.

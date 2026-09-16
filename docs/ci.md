# Continuous integration

The CI toolchain uses Go 1.27.0 and Dart 3.13.3.
The JavaScript and WebAssembly tests use Node.js 26.8.2.
GitHub Actions uses Ubuntu 24.04 on AMD64. Local duration measurements use macOS/ARM64.

The workflow executes the current `make ci` test selection across 13 jobs:

- 1 job runs format validation, vet, Flutter Go tests, Dart runtime tests, and compiler tests.
- 4 jobs run all 376 custom Go/Dart parity programs.
- 4 jobs run all 931 discovered upstream Go language cases, with explicit exclusions.
- 4 jobs run the 37 current GOROOT package suites, with their existing filters and exclusions.

The workflow preserves the current coverage scope. It does not establish complete upstream Go parity.
Native memory-page tests, allocator tests, and other documented exclusions do not count as parity passes.

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
Each job set `GOMAXPROCS=2` and `GOFLAGS=-p=2`.
The measurements include concurrent local jobs and reuse compilation and dependency caches.
All test results are uncached.

| Suite | Shard | Local duration | Test step limit | Job limit |
|---|---|---|---|---|
| checks | all | 584.108s | 20 min | 25 min |
| programs | 0 | 360.429s | 13 min | 18 min |
| programs | 1 | 278.109s | 10 min | 15 min |
| programs | 2 | 226.487s | 8 min | 13 min |
| programs | 3 | 305.146s | 11 min | 16 min |
| language | 0 | 305.440s | 11 min | 16 min |
| language | 1 | 212.768s | 8 min | 13 min |
| language | 2 | 234.478s | 8 min | 13 min |
| language | 3 | 244.708s | 9 min | 14 min |
| stdlib | 0 | 779.662s | 26 min | 31 min |
| stdlib | 1 | 55.658s | 2 min | 7 min |
| stdlib | 2 | 77.764s | 3 min | 8 min |
| stdlib | 3 | 72.940s | 3 min | 8 min |

Each test step uses twice its measured local duration, with whole-minute rounding.
Job limits allow 5 additional minutes for toolchain setup and log upload.
Suite jobs compile the driver test binary before the timed test step.
This step matches the local compilation-cache state and exposes a cold build as a separate CI step.
`GART_TEST_TIMEOUT` controls the Go test timeout for a manual wrapper run.

The Linux/AMD64 `fixedbugs/issue13169` stress case completed in 116.89 seconds under local emulation.
Its generated program has a 4-minute limit, which follows the same doubled-duration rule.
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
The 4 language shards execute all 931 cases exactly once: 775 pass and 156 report explicit exclusions.
All 37 declared GOROOT package suites pass across their 4 shards.
The runtime tests, Flutter Go tests, format validation, vet, and Dart analyzer also pass.

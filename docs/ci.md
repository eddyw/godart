# Continuous integration

CI runs only through manual dispatch while the repository is private.
Pushes and pull requests do not start a run.
Run the local tests before a push. Use GitHub Actions for occasional Linux validation.
Restore automatic push and pull request triggers when the repository becomes public.

Start a manual run from **Actions → CI → Run workflow**, or use:

```bash
gh workflow run ci.yml --repo eddyw/godart --ref main
```

The CI toolchain uses Go 1.27.0 and Dart 3.13.3.
The JavaScript and WebAssembly tests use Node.js 26.8.2.
GitHub Actions uses Ubuntu 24.04 on AMD64. The final local baseline uses Linux/AMD64 containers under emulation.

The workflow executes the current `make ci` test selection across 19 test jobs and 1 matrix-definition job:

- 1 job runs format validation, vet, Flutter Go tests, Dart runtime tests, and native Go packages outside the driver.
- 1 job runs the driver tests outside the larger integration and target groups.
- 1 job runs the plain and verbose integration comparisons.
- 3 jobs run AOT/JS/wasm basics, web I/O, and web fetch.
- 6 jobs run all 376 custom Go/Dart parity programs.
- 4 jobs run all 931 discovered upstream Go language cases, with explicit exclusions.
- 3 jobs run the 37 current GOROOT package suites, with their existing filters and exclusions.

`scripts/ci-matrix.json` defines the job selection and limits.
The workflow uses Node24 versions of checkout, setup-go, setup-node, and upload-artifact.
The Node.js version for generated program execution remains 26.8.2.

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
GOMAXPROCS=2 GOFLAGS=-p=2 bash scripts/ci.sh programs 0 6
```

The suite names are `checks`, `driver`, `integration`, `targets`, `web-io`, `web-fetch`, `programs`, `language`, and `stdlib`.
Shard indexes start at 0. Without shard arguments, the command runs the complete named suite.
The driver assigns the longest estimated cases to the shard with the least estimated work.
`internal/driver/testdata/ci-durations.json` holds the estimates. New cases receive a nonzero estimate and remain in the selection.
Source enumeration order does not change the assignments. Each case belongs to exactly 1 shard.
The default Go test flags select all shards as 1 complete suite.

The initial estimates come from GitHub run 35097575007.
The split flate estimates use the optimized Mac probe durations with a factor of 3 for Linux emulation.
These estimates select shards. They do not determine test timeout limits.

All Go CI tests use `-count=1`. Go compilation and dependency caches can remain warm.
The wrapper reports elapsed seconds and preserves the command exit status.
GitHub Actions retains the log for 7 days and reports the duration in the job summary.
A new manual run cancels a superseded run for the same ref.

## Cost and coverage controls

Verbose driver tests report elapsed build phases: load, verify, analyze, emit, write, format, and source maps.
They also report dependency setup and execution durations.
External-command durations include process startup. A Dart VM execution duration also includes its compilation cost.
GOROOT logs retain successful upstream test names, subtests, skips, and durations.

The compiler uses lazy zero initialization for unboxed array locals in a state machine.
An assignment before the first read avoids an unused allocation.
A read before assignment, including a frame save, still obtains the required zero value.
A regression covers goto, alias writes, zero values, named results, and channel suspension.

The flate CPU profile found 6 temporary arrays of 32768 structs per compression call.
The complete 2.3 GB sparse case took 117.25 seconds before the change and 15.48 seconds after it on the Mac.
The probes overlapped other work, thus they do not establish an isolated speed ratio.
All 128 flate test identities and outcomes match across native Go and both complete Dart probes.

The standard-library harness divides flate into 4 executions:

- All discovered upstream tests except `TestWriter_Reset`.
- `TestWriter_Reset` levels 0, 1, 5, and 7.
- `TestWriter_Reset` levels 2, 6, and 8.
- `TestWriter_Reset` levels 3, 4, and 9.

The 10 level subtests retain their inputs, iteration counts, assertions, and upstream skips.
The outer reset setup executes for each group. Each level subtest executes once.
The remaining test filter comes from native Go test discovery, which includes new upstream test functions.
The 37 package suites now use 40 harness cases.

Program and language fixtures share 1 real Pub dependency resolution per test process.
The helper relocates the generated package entry and preserves resolved dependency paths.
A regression compares the resulting package configuration with fresh Pub output.
Explicit package configuration prevents another implicit Pub resolution during fixture execution.
CLI and target tests retain real Pub resolution.

The plain/verbose integration pair reuses its build output. Both test executions still occur.
Program, GOROOT, and shared integration helpers omit redundant Dart formatting.
Golden and target tests retain formatting coverage. Cache invalidation tests remain separate.
The harness does not share compiled Go packages across different program fixtures.
Such reuse would require the complete source, compiler, target, path, dependency, and block-analysis identity.

## Duration baseline

The baseline date is 2026-09-16. The host used an Apple M4 Max with 16 cores.
The Docker VM has 4 CPUs. Each Linux/AMD64 container has a 2-CPU limit.
All containers run as UID/GID 65534, with uncached test results and warm compilation/dependency caches.
Each job sets `GOMAXPROCS=2` and `GOFLAGS=-p=2`.
The complete 19-job traversal took 45m45s with at most 3 concurrent containers. All jobs passed.
The earlier 13-job traversal took about 64 minutes and required extra corrected replays.
These local totals include limited concurrency and Linux emulation. They are not GitHub wall-time estimates.

| Suite | Shard | Local duration | Test step limit | Job limit |
|---|---|---|---|---|
| checks | 0 | 48s | 2 min | 7 min |
| driver | 0 | 381s | 13 min | 18 min |
| integration | 0 | 195s | 7 min | 12 min |
| targets | 0 | 118s | 4 min | 9 min |
| web-io | 0 | 179s | 6 min | 11 min |
| web-fetch | 0 | 177s | 6 min | 11 min |
| programs | 0 | 490s | 17 min | 22 min |
| programs | 1 | 502s | 17 min | 22 min |
| programs | 2 | 501s | 17 min | 22 min |
| programs | 3 | 489s | 17 min | 22 min |
| programs | 4 | 504s | 17 min | 22 min |
| programs | 5 | 508s | 17 min | 22 min |
| language | 0 | 749s | 25 min | 30 min |
| language | 1 | 755s | 26 min | 31 min |
| language | 2 | 756s | 26 min | 31 min |
| language | 3 | 612s | 21 min | 26 min |
| stdlib | 0 | 337s | 12 min | 17 min |
| stdlib | 1 | 386s | 13 min | 18 min |
| stdlib | 2 | 375s | 13 min | 18 min |

Each test step uses twice its measured local duration, rounded up to whole minutes.
Job limits allow 5 additional minutes for toolchain setup and log upload.
Suite jobs compile the driver test binary before the timed test step.
`GART_TEST_TIMEOUT` controls the Go test timeout for a manual wrapper run.

The filesystem suites run unprivileged. A root process bypasses permission failures that upstream tests require.
The current Dart `os.Getuid` reports `-1`, which cannot activate the upstream root-user skip.

The unchanged `fixedbugs/issue13169` stress case retains its 5-minute subprocess limit.
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

The current root inventory contains 98 Go test entry points.
The 6 program shards must execute all 376 cases exactly once.
The Linux language shards execute all 931 cases exactly once: 775 pass and 156 report explicit exclusions.
All 37 declared GOROOT package suites must pass across their 3 shards.
The runtime tests, Flutter Go tests, format validation, vet, and Dart analyzer also pass.

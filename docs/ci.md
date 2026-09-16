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
GitHub Actions uses Ubuntu 24.04 on AMD64 and a macOS ARM64 priority job. Local Linux validation uses AMD64 containers under emulation.

The workflow executes the current `make ci` test selection across 26 test jobs and 1 matrix-definition job:

- 1 job runs format validation, vet, Flutter Go tests, Dart runtime tests, and native Go packages outside the driver.
- 1 job runs the driver tests outside the larger integration and target groups.
- 1 job runs the plain and verbose integration comparisons.
- 3 jobs run AOT/JS/wasm basics, web I/O, and web fetch.
- 6 jobs run all 376 custom Go/Dart parity programs.
- 4 jobs run all 931 discovered upstream Go language cases, with explicit exclusions.
- 6 jobs run the 116 declared upstream package suites, with exact outcome comparisons and explicit exclusions.
- 1 job runs original upstream assertions across AOT, JS, and wasm.
- 2 jobs run priority upstream assertions on Linux and macOS, including independent native HTTP servers.
- 1 job verifies the 77 known package compile blockers.

`scripts/ci-matrix.json` defines the job selection and limits.
The workflow uses Node24 versions of checkout, setup-go, setup-node, and upload-artifact.
The Node.js version for generated program execution remains 26.8.2.

The workflow enforces the declared [upstream parity contract](upstream-parity.md). It does not establish complete upstream Go parity.
Native memory-page tests, allocator tests, and other documented exclusions do not count as parity passes.
The 2 math fixtures also mark 15 host-dependent subnormal result comparisons. See [support limits](support.md).

Routine standard library jobs pass upstream `-short` to both Go and Gart.
The workflow input `upstream_mode=full` selects the original long suites.
Both modes retain their exact exclusions and record their configuration in the evidence.
Linux standard library jobs install `zstd` and `xxhash` for the original decompressor and hash tests.

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

The suite names are `checks`, `driver`, `integration`, `targets`, `web-io`, `web-fetch`, `programs`, `language`, `stdlib`, `upstream-priority`, `upstream-targets`, and `upstream-blockers`.
Shard indexes start at 0. Without shard arguments, the command runs the complete named suite.
The driver assigns the longest estimated cases to the shard with the least estimated work.
`internal/driver/testdata/ci-durations.json` holds the estimates. New cases receive a nonzero estimate and remain in the selection.
Source enumeration order does not change the assignments. Each case belongs to exactly 1 shard.
The default Go test flags select all shards as 1 complete suite.

The program, language, and full-mode estimates retain the earlier measured baseline.
Short standard library estimates use the 2026-09-17 Linux package durations.
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

The full-mode standard library harness divides flate into 4 executions:

- All discovered upstream tests except `TestWriter_Reset`.
- `TestWriter_Reset` levels 0, 1, 5, and 7.
- `TestWriter_Reset` levels 2, 6, and 8.
- `TestWriter_Reset` levels 3, 4, and 9.

The 10 level subtests retain their inputs, iteration counts, assertions, and upstream skips.
The outer reset setup executes for each group. Each level subtest executes once.
The remaining test filter comes from native Go test discovery, which includes new upstream test functions.
Full mode uses 119 harness cases. Short mode uses 116 cases and preserves Go’s original reset levels 0 and 1.

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

The host uses an Apple M4 Max with 16 cores. The Docker VM has 4 CPUs.
Each Linux/AMD64 container has a 2-CPU limit and runs as UID/GID 65534.
The 2026-09-17 acceptance replay uses at most 2 concurrent containers.
All measurements use uncached test results and warm compilation/dependency caches.
Each Linux job sets `GOMAXPROCS=2` and `GOFLAGS=-p=2`.

The dated rows distinguish the earlier complete Linux regression baseline from the new upstream parity measurements.
The macOS priority measurement uses the same 2-core Go settings.
These are local measurements under the stated conditions, not GitHub wall-time measurements.
No GitHub Actions run occurred during this stabilization batch.

| Suite | Platform | Shard | Measurement date | Local seconds | Test step minutes | Job minutes |
|---|---|---:|---|---:|---:|---:|
| checks | Linux | 0 | 2026-09-17 | 32.0 | 2 | 7 |
| driver | Linux | 0 | 2026-09-16 | 381.0 | 13 | 18 |
| integration | Linux | 0 | 2026-09-16 | 195.0 | 7 | 12 |
| targets | Linux | 0 | 2026-09-16 | 118.0 | 4 | 9 |
| web-io | Linux | 0 | 2026-09-16 | 179.0 | 6 | 11 |
| web-fetch | Linux | 0 | 2026-09-16 | 177.0 | 6 | 11 |
| programs | Linux | 0 | 2026-09-16 | 490.0 | 17 | 22 |
| programs | Linux | 1 | 2026-09-16 | 502.0 | 17 | 22 |
| programs | Linux | 2 | 2026-09-16 | 501.0 | 17 | 22 |
| programs | Linux | 3 | 2026-09-16 | 489.0 | 17 | 22 |
| programs | Linux | 4 | 2026-09-16 | 504.0 | 17 | 22 |
| programs | Linux | 5 | 2026-09-16 | 508.0 | 17 | 22 |
| language | Linux | 0 | 2026-09-16 | 749.0 | 25 | 30 |
| language | Linux | 1 | 2026-09-16 | 755.0 | 26 | 31 |
| language | Linux | 2 | 2026-09-16 | 756.0 | 26 | 31 |
| language | Linux | 3 | 2026-09-16 | 612.0 | 21 | 26 |
| upstream-targets | Linux | 0 | 2026-09-17 | 475.6 | 16 | 21 |
| upstream-priority | Linux | 0 | 2026-09-17 | 125.8 | 5 | 10 |
| upstream-blockers | Linux | 0 | 2026-09-17 | 78.1 | 3 | 8 |
| upstream-priority | macOS | 0 | 2026-09-17 | 116.0 | 4 | 9 |
| stdlib (short) | Linux | 0 | 2026-09-17 | 199.8 | 7 | 12 |
| stdlib (full) | Linux | 0 | 2026-09-17 | 808.1 | 27 | 32 |
| stdlib (short) | Linux | 1 | 2026-09-17 | 222.8 | 8 | 13 |
| stdlib (full) | Linux | 1 | 2026-09-17 | 399.5 | 14 | 19 |
| stdlib (short) | Linux | 2 | 2026-09-17 | 196.2 | 7 | 12 |
| stdlib (full) | Linux | 2 | 2026-09-17 | 244.1 | 9 | 14 |
| stdlib (short) | Linux | 3 | 2026-09-17 | 196.1 | 7 | 12 |
| stdlib (full) | Linux | 3 | 2026-09-17 | 527.4 | 18 | 23 |
| stdlib (short) | Linux | 4 | 2026-09-17 | 194.3 | 7 | 12 |
| stdlib (full) | Linux | 4 | 2026-09-17 | 796.2 | 27 | 32 |
| stdlib (short) | Linux | 5 | 2026-09-17 | 202.2 | 7 | 12 |
| stdlib (full) | Linux | 5 | 2026-09-17 | 221.6 | 8 | 13 |

Each test step uses twice its measured local duration, rounded up to whole minutes.
Job limits allow 5 additional minutes for toolchain setup and log upload.
The standard library jobs select separate short and full limits from the workflow input.
Suite jobs compile the driver test binary before the timed test step.
`GART_TEST_TIMEOUT` controls the Go test timeout for a local wrapper run.

The filesystem suites run unprivileged. A root process bypasses permission failures that upstream tests require.
The current Dart `os.Getuid` reports `-1`, which cannot activate the upstream root-user skip.

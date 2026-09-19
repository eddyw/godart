# Upstream Go parity

The parity gate compares original Go assertions with Gart results under a recorded configuration.
A pass requires successful native Go and Gart processes and matching test identities.
It does not establish behavior for inputs that the upstream tests do not exercise.

## Source and coverage

The baseline uses Go 1.27.0 and Dart 3.13.3.
The browser runners use Node.js 26.8.2. The zstd suite requires the `zstd` and `xxhsum` commands.
The local acceptance environment uses zstd 1.5.7 and xxHash 0.8.3 on both platforms.
The platform manifests cover Darwin/ARM64 and Linux/AMD64.

`internal/parity/testdata/` contains these manifests:

| Manifest | Contract |
|---|---|
| `go<version>/darwin-arm64.json`, `go<version>/linux-amd64.json` | Source hashes, build configuration, declarations, file eligibility, corpus bytes, and test patches |
| `coverage-manifest.json` | All packages in the declared standard library surface |
| `vm-suites.json` | Required package suites |
| `targets.json` | Required AOT, JS, and wasm executions and browser exclusions |
| `exceptions-*.json` | Exact native/Gart outcomes per platform and short/full mode, with an explicit reason |
| `external-fixtures.json` | Original external module versions and Go module checksums |
| `omitted-tests.json` | Tests that require an unavailable runtime or native executable facility |

The source inventory contains 260 packages and 9375 declarations.
Of those declarations, 7706 are tests, examples, or fuzz targets.
Darwin/ARM64 selects 7297 runnable declarations before compilation. Linux/AMD64 selects 7373.

| Platform | Declarations in VM suites | Declarations in blocked suites |
|---|---:|---:|
| Darwin/ARM64 | 3348 | 3949 |
| Linux/AMD64 | 3352 | 4021 |

These counts describe eligible root declarations. Runtime outcome counts also include subtests and corpus cases, which form a different denominator.
Benchmarks remain visible in the inventory and do not count as assertion coverage.

The VM manifest declares 150 package suites.
Another 45 packages have an exact, verified compile blocker.
The remaining 65 packages have no runnable upstream declaration.
Selected original declarations also exercise supported APIs in 8 blocked packages.
The HTTP adapter preserves 10 original test bodies and their H1/H2 assertions.

A package suite can contain explicit exclusions or partial assertions.
Thus a package count does not imply complete behavioral parity for that package.
A compile blocker never removes declarations from the denominator.

## Assertion integrity

The native baseline runs the clean installed Go source.
It does not use Gart overlays or test patches.
The 20 declared crypto suites use native Go `-tags=purego` to select portable implementations.
Their manifest entries and reports record that choice. A gate rejects any change to upstream test-file eligibility from those tags.
This configuration does not claim assembly implementation or native FIPS module parity.
The Gart side preserves the original assertions, expected values, and fixture bytes.
Adapters supply unavailable private test infrastructure without a duplicate assertion implementation.

The inventory gate rejects changes to upstream source, fixtures, build tags, and test patches.
Each SDK version has a separate inventory. A candidate inventory records reviewed
inputs without promotion of the release baseline. Exact `_go.mod` duplicates
from Go toolchain module distributions do not change the source fingerprint.
Different duplicate contents still change the fingerprint.
The coverage gate rejects missing package classifications and lost package suites.
The compiler gate rejects a new diagnostic for a known blocker.
The coverage gate also requires a cause and reason for each blocker. [Blocker classifications](upstream-blockers.md) distinguish public API gaps from private test dependencies and native runtime boundaries.
It also requires promotion when a blocked package compiles successfully.

The result gate rejects these conditions:

- A failed process, including matching failures on both runtimes.
- An empty selection or an absent expected test.
- An unexpected test, duplicate identity, or unknown skip.
- A changed or unused exception.
- A missing subtest, even when its parent passes.

An excluded assertion prevents its parent from counting as a complete parity pass.
The negative integration tests exercise deliberate failures, skips, and missing subtests.

Original fuzz corpus bytes enter the generated test binary at compilation.
The cache key includes changed and removed corpus files.
Explicit upstream seeds remain unchanged.
Full crypto suites retain the original pinned Wycheproof and Ed25519 vector modules.
The report records their versions and checksums. The gate also hashes the extracted module files and rejects an unknown version or changed content.
Short mode retains the original external-network skip.
Tests that choose a time-based random seed retain that upstream behavior.
Their successful runs do not prove equality for an identical random input sequence.

## Short and full modes

Routine standard library and target CI use the upstream `-short` configuration for both Go and Gart.
The original test code controls its short-mode branches and iteration counts.
The compiler does not edit those branches, inputs, or assertions.
Short-mode skips remain explicit exclusions, and reports record the selected mode.
A short-mode pass establishes parity only for that configuration.

The manual workflow accepts `upstream_mode=full` for the original long suites.
Locally, set `GART_UPSTREAM_MODE=full` before the standard library or upstream target command.
Direct `go test` runs use full mode unless the command includes `-short`.

## Evidence and local commands

Set `GART_PARITY_REPORT_DIR` to retain the structured comparison and both raw outputs.
Each report records source and fixture hashes, toolchain versions, target, platform, build tags, filters, short/full mode, timeout, exclusions, and process exits.

```sh
GART_PARITY_REPORT_DIR="$PWD/build/parity-results" bash scripts/ci.sh stdlib 0 6
GART_PARITY_REPORT_DIR="$PWD/build/parity-results" bash scripts/ci.sh upstream-priority
GART_PARITY_REPORT_DIR="$PWD/build/parity-results" bash scripts/ci.sh upstream-targets
bash scripts/ci.sh upstream-blockers
bash scripts/ci.sh checks
GART_UPSTREAM_MODE=full bash scripts/ci.sh stdlib 0 6
```

Run all 6 standard library shards for the complete VM manifest.
Use the same commands locally and in GitHub Actions.
The workflow requires manual dispatch and retains the parity reports as artifacts.
The macOS priority job uses the ARM64 `macos-15` runner from the [GitHub runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).
Scheduled and automatic runs remain disabled under the current repository policy.

## Current boundaries

The reports distinguish complete passes, partial assertions, explicit exclusions, and compile blockers.
Unavailable allocation counters, native machine addresses, unsafe operations, extra Go execution threads, and fixed-stack limits remain outside the documented runtime contract.
Native executable inspection and test reexecution require separate adapters.
The browser target has no local filesystem or listening socket.
These boundaries do not excuse an unexpected failure in supported behavior.

## Second stabilization batch

The batch adds flag, log, encoding/binary, and encoding/gob to the VM manifest.
Original Go source replaces the flag, log, and binary overlays.
Binary includes complex values. Gob preserves destination slice storage when capacity permits.
The HTTP header writer rejects invalid names and replaces CR/LF in values with spaces.
HTTP date parsing uses the original RFC850 layout.

Priority adapters add original assertions for header serialization, HTTP dates, time formats, file permissions, reflection conversions, WaitGroup, and process environment deduplication.
The process environment fixture uses the actual overlay helper. Its native baseline executes the original Go package tests.
The time fixture sets the original Los Angeles zone before the test data initializes.
All selected assertion bodies remain original.

Binary adds an AOT, JavaScript, and WebAssembly suite. The target manifest now has 21 packages and 63 executions per platform and mode.
Its unsafe native-memory assertion remains an explicit exclusion.
Gob uses the target integer width for the upstream uintptr-size predicate. This preserves all assertions on the 64-bit supported baselines.
Gob also retains an explicit exclusion for deep synthetic reflection types that exceed the Dart stack.
Original opt-in fuzz skips, allocation limits, and native test-executable relaunches remain explicit exclusions.

## Acceptance snapshot: 2026-09-18

All declared suites pass locally across the accepted original runs and corrected replays.
Full VM mode uses 166 harness executions across 150 packages. Short VM mode uses 150 executions.
The full replay retains all original long iterations, ZIP fixtures, external crypto vectors, and installed zstd hash tools.

| Platform | Scope | Selected roots | Complete outcome matches | Partial parents | Excluded outcomes |
|---|---|---:|---:|---:|---:|
| darwin/arm64 | VM short | 3348 | 47197 | 1020 | 2386 |
| darwin/arm64 | VM full | 3348 | 77934 | 1023 | 2981 |
| darwin/arm64 | AOT short/full | 426 | 836 | 2 | 114 |
| darwin/arm64 | JS short/full | 421 | 795 | 2 | 114 |
| darwin/arm64 | WASM short/full | 421 | 795 | 2 | 114 |
| darwin/arm64 | Priority VM | 190 | 231 | 0 | 2 |
| linux/amd64 | VM short | 3352 | 47197 | 1020 | 2390 |
| linux/amd64 | VM full | 3352 | 77935 | 1023 | 2984 |
| linux/amd64 | AOT short/full | 426 | 836 | 2 | 114 |
| linux/amd64 | JS short/full | 421 | 795 | 2 | 114 |
| linux/amd64 | WASM short/full | 421 | 795 | 2 | 114 |
| linux/amd64 | Priority VM | 190 | 231 | 0 | 2 |

Outcome totals include root tests, subtests, examples, and corpus cases. They do not count individual assertions.
Excluded outcomes include upstream skips, runtime exclusions, and patched tests with partial assertions.
They do not count as complete matches. Repeated parent identities count once.
The source inventory retains all declarations from the 45 blocked packages.

Both modes pass all 63 target executions across 21 packages on each platform.
Their outcome counts match, while their original upstream iteration counts differ.
The browser manifest excludes 5 filesystem-dependent roots from the 426-root target selection before execution.
Those roots remain in the source inventory.
Priority VM results cover 190 roots across 8 packages whose complete suites remain blocked.
The independent native HTTP servers pass all 14 client comparisons across HTTP/1 and HTTP/2.

The Linux regression replay exposed concurrent output in the custom pipe fixture. The fixture now prints its results in a fixed order.
The original allocation stress case `fixedbugs/issue78081` exceeded its 2-minute Linux execution limit under concurrent load.
Its isolated Linux run completes in 55.4 seconds. The case now uses the existing 5-minute stress budget, with all 104 million recursive calls unchanged.
Both complete regression shard replays pass. The stress case takes 64.3 seconds in its accepted replay.
The Linux corpus has 376 program passes and 775 language passes, with 156 explicit language exclusions.
The full log policy also required correction. The parent passes on both runtimes, while 14 child assertions remain excluded.
Both complete affected full shards pass after the policy correction. The parent counts as partial, not complete.
The local runner retains all original failures. No failed attempt counts as an acceptance pass.
Both platforms pass the driver, integration, basic target, browser, runtime, Go unit, vet, format, and Flutter Go gates.
The rebuilt CLI runs without an explicit runtime path and reports durations separately with `-timings`.

The [CI measurements](ci.md#duration-baseline) define the doubled manual-workflow limits.
GitHub Actions remains manual-only. This batch used no GitHub Actions minutes.

## Declared language versions

`TestVersionedGoFixtures` compares unchanged upstream loopvar sources under
Go 1.21 and Go 1.22 module declarations. It includes per-file build constraints,
address captures, closures, method captures, and nested loops.
The adapter compares complete output and exact process exit codes.
A nonzero native result remains the expected result for the corresponding
Gart run. It does not mean that the adapter changed an upstream assertion.

`internal/driver/testdata/versioned-go-tests.json` records source paths and
SHA-256 hashes. Review source changes during an SDK update before the hash
update. Native compiler diagnostics and inlining assertions remain outside
this runtime comparison. The adapter does not claim full compiler-suite parity.

Run `go test ./internal/driver -run '^TestVersionedGoFixtures$'` for this matrix.
The language CI suite includes it. The same output directory across module
versions also verifies cache invalidation when only the language version changes.

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
| `darwin-arm64.json`, `linux-amd64.json` | Source hashes, build configuration, declarations, file eligibility, corpus bytes, and test patches |
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
| Darwin/ARM64 | 3163 | 4134 |
| Linux/AMD64 | 3167 | 4206 |

These counts describe eligible root declarations. Runtime outcome counts also include subtests and corpus cases, which form a different denominator.
Benchmarks remain visible in the inventory and do not count as assertion coverage.

The VM manifest declares 146 package suites.
Another 49 packages have an exact, verified compile blocker.
The remaining 65 packages have no runnable upstream declaration.
Selected original declarations also exercise supported APIs in 7 blocked packages.
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
The coverage gate rejects missing package classifications and lost package suites.
The compiler gate rejects a new diagnostic for a known blocker.
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

## Acceptance snapshot: 2026-09-17

All declared suite gates pass locally under the recorded platform and mode.
Full VM mode uses 162 harness executions across 146 packages. Short VM mode uses 146 executions.
The full replay includes all original long iterations, ZIP fixtures, external crypto vectors, and installed zstd hash tools.
Repeated parent identities count once in these totals.

| Platform | Scope | Selected roots | Complete outcome matches | Partial parents | Excluded outcomes |
|---|---|---:|---:|---:|---:|
| darwin/arm64 | VM short | 3163 | 46920 | 1019 | 2357 |
| darwin/arm64 | VM full | 3163 | 77657 | 1021 | 2939 |
| darwin/arm64 | AOT short/full | 384 | 701 | 1 | 94 |
| darwin/arm64 | JS short/full | 379 | 660 | 1 | 94 |
| darwin/arm64 | WASM short/full | 379 | 660 | 1 | 94 |
| darwin/arm64 | Priority VM | 167 | 208 | 0 | 2 |
| linux/amd64 | VM short | 3167 | 46920 | 1019 | 2361 |
| linux/amd64 | VM full | 3167 | 77658 | 1021 | 2942 |
| linux/amd64 | AOT short/full | 384 | 701 | 1 | 94 |
| linux/amd64 | JS short/full | 379 | 660 | 1 | 94 |
| linux/amd64 | WASM short/full | 379 | 660 | 1 | 94 |
| linux/amd64 | Priority VM | 167 | 208 | 0 | 2 |

Outcome totals include root tests, subtests, examples, and corpus cases. They do not count individual assertions.
Excluded outcomes include upstream skips, runtime exclusions, and patched tests with partial assertions.
They do not count as complete matches.
Both short and full target modes pass all 60 executions across 20 packages on each platform.
Their outcome counts match, while their original upstream iteration counts differ.
The browser manifest excludes 5 filesystem-dependent roots from the 384-root target selection before execution.
Those roots remain in the source inventory.
Priority VM results cover 167 roots across 7 packages whose complete suites remain blocked.
Both platforms reproduce all 49 expected compile-blocker diagnostics. Their declarations remain in the denominator.

The independent native HTTP servers pass all 14 client comparisons across HTTP/1 and HTTP/2.
The macOS regression corpus passes all 376 programs and 775 Go language cases, with 156 explicit language skips.
The complete driver, runtime, Go unit, vet, format, and Flutter suites pass.
The rebuilt CLI prints `hello, world` without an explicit runtime path.
The [CI measurements](ci.md#duration-baseline) define the doubled manual-workflow limits.
GitHub Actions remains manual-only. This batch used no GitHub Actions minutes.

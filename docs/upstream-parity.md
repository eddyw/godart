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
| `omitted-tests.json` | Tests that require an unavailable runtime or native executable facility |

The source inventory contains 255 packages and 9373 declarations.
Of those declarations, 7704 are tests, examples, or fuzz targets.
Darwin/ARM64 selects 7295 runnable declarations before compilation. Linux/AMD64 selects 7371.

| Platform | Declarations in VM suites | Declarations in blocked suites |
|---|---:|---:|
| Darwin/ARM64 | 2710 | 4585 |
| Linux/AMD64 | 2714 | 4657 |

These counts describe eligible root declarations. Runtime outcome counts also include subtests and corpus cases, which form a different denominator.
Benchmarks remain visible in the inventory and do not count as assertion coverage.

The VM manifest declares 116 package suites.
Another 77 packages have an exact, verified compile blocker.
The remaining 62 packages have no runnable upstream declaration.
Selected original files also exercise supported APIs in 6 blocked packages.
The HTTP adapter preserves 10 original test bodies and their H1/H2 assertions.

A package suite can contain explicit exclusions or partial assertions.
Thus a package count does not imply complete behavioral parity for that package.
A compile blocker never removes declarations from the denominator.

## Assertion integrity

The native baseline runs the clean installed Go source.
It does not use Gart overlays or test patches.
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
Tests that choose a time-based random seed retain that upstream behavior.
Their successful runs do not prove equality for an identical random input sequence.

## Short and full modes

Routine CI uses the standard upstream `-short` configuration for both Go and Gart.
The original test code controls its short-mode branches and iteration counts.
The compiler does not edit those branches, inputs, or assertions.
Short-mode skips remain explicit exclusions, and reports record the selected mode.
A short-mode pass establishes parity only for that configuration.

The manual workflow accepts `upstream_mode=full` for the original long suites.
Locally, set `GART_UPSTREAM_MODE=full` before the standard library command.
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

All declared suite gates passed locally under the recorded platform and mode.
Full-mode evidence includes 119 harness executions over 116 packages. Short mode uses 116 executions.
The final zstd replays include the installed hash tool and all 5 original hash seeds.
Repeated flate parent identities count once in the totals below.

| Platform | Scope | Selected roots | Complete outcome matches | Partial parents | Excluded outcomes |
|---|---|---:|---:|---:|---:|
| darwin/arm64 | VM short | 2710 | 8499 | 15 | 436 |
| darwin/arm64 | VM full | 2710 | 8687 | 14 | 517 |
| linux/amd64 | VM short | 2714 | 8499 | 15 | 440 |
| linux/amd64 | VM full | 2714 | 8688 | 14 | 520 |
| darwin/arm64 | AOT | 215 | 487 | 1 | 89 |
| darwin/arm64 | JS | 210 | 446 | 1 | 89 |
| darwin/arm64 | Priority VM | 83 | 124 | 0 | 2 |
| darwin/arm64 | WASM | 210 | 446 | 1 | 89 |
| linux/amd64 | AOT | 215 | 487 | 1 | 89 |
| linux/amd64 | JS | 210 | 446 | 1 | 89 |
| linux/amd64 | Priority VM | 83 | 124 | 0 | 2 |
| linux/amd64 | WASM | 210 | 446 | 1 | 89 |

Outcome totals include root tests, subtests, examples, and corpus cases. They are not counts of individual assertions.
Excluded outcomes include upstream skips, runtime exclusions, and patched tests with partial assertions.
They do not count as complete matches.
The browser manifests separately exclude 5 filesystem-dependent root declarations before execution, from a declared target baseline of 215 roots.
Those declarations remain in the source inventory.
Priority VM results cover 83 root declarations across 7 packages whose complete suites remain blocked.
Both platforms reproduce all 77 expected compile-blocker diagnostics. Their declarations remain in the denominator.

The independent native HTTP servers also pass all 14 client comparisons across HTTP/1 and HTTP/2.
The macOS regression corpus passes all 376 programs and 775 Go language cases, with 156 explicit language skips.
The current driver suite, runtime analysis and tests, Go unit tests, vet, format gates, and complete Flutter suite pass.
The CLI smoke prints `hello, world` without an explicit runtime path.
The [CI measurements](ci.md#duration-baseline) define the doubled manual-workflow limits.
GitHub Actions remains manual-only and was not executed for this snapshot.

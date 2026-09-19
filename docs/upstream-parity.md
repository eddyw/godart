# Upstream Go parity

The parity comparison uses original Go assertions and Gart results under a recorded configuration.
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
| `go<version>/darwin-arm64.json`, `go<version>/linux-amd64.json` | SDK-specific source hashes, build configuration, declarations, file eligibility, corpus bytes, and test patches |
| `coverage-manifest.json` | All packages in the declared standard library surface |
| `vm-suites.json` | Required package suites |
| `targets.json` | Required AOT, JS, and wasm executions and browser exclusions |
| `exceptions-*.json` | Exact native/Gart outcomes per platform and short/full mode, with an explicit reason |
| `external-fixtures.json` | Original external module versions and Go module checksums |
| `omitted-tests.json` | Omitted assertions with an explicit classification and reason |

The inventories retain eligible and ineligible declarations. Runtime outcomes
also include subtests and corpus cases. These are different denominators.
Benchmarks remain visible but do not count as assertion coverage.
Use the manifests and retained reports for current counts.

A package suite can contain explicit exclusions or partial assertions.
Thus a package count does not imply complete behavioral parity for that package.
A compile blocker never removes declarations from the denominator.

Omissions distinguish implementation gaps, test adapters, target limits, and
unassessed cases. They do not all represent unreachable behavior.
Deep recursion can require an iterative implementation before the same upstream
input can pass. An environment-controlled library setting does not require
native runtime control merely because Go names it through GODEBUG.
Native executable assumptions can require test adapters. Mixed `go/build`
assumptions still need individual review. The report retains all these groups.

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

### Repeatable measurements

`scripts/parity.py` produces JSON and Markdown reports for 3 separate measurements:

- API declarations include interface methods, signatures, constants, public fields, and receiver availability.
- Upstream results retain compiler snapshots, SDKs, platforms, targets, and short/full modes separately.
- Application results distinguish compilation, equal execution output, and equal test outcomes.

```sh
python3 scripts/parity.py report --reports build/parity-results --output build/parity-summary
python3 scripts/parity.py application --project /path/to/app --output build/app-compile
python3 scripts/parity.py application --project /path/to/app --mode run --output build/app-run
python3 scripts/parity.py application --project /path/to/app --mode test --output build/app-tests
python3 scripts/parity.py applications --output build/pinned-applications
```

Use a new output directory for each invocation. The application command defaults to compilation only.
It compiles both a native Go executable and a Dart kernel. Run mode executes both programs with closed stdin.
Run mode requires identical stdout, stderr, and successful exits. Use deterministic programs without external side effects.
Test mode selects 1 package and compares short-mode test identities and outcomes. Skips remain partial results.
Use `--tags safe` when a dependency provides a portable implementation under that tag.
The command applies identical tags to native Go and Gart and records them in the result.
The application command records command arguments, exits, durations, logs, SDKs, source identity, and the Gart binary hash.
It does not prove a match for nondeterministic inputs or unexecuted behavior.

The `applications` command builds Gart and compares the repository's pinned
[application fixtures](../testdata/applications/README.md). It returns a failure
if any program or original dependency test fails the comparison.
The local CI matrix and the manual GitHub workflow include this suite.
Use `python3 scripts/ci-local.py --suite applications --output build/application-run`
for an isolated source snapshot and retained command results.

Add `--applications build/app-results` to include application records in a report.
Repeat `--reports` for separate evidence directories. The report never blends incompatible configurations or compiler snapshots.
Reports without compiler provenance remain explicitly unattributed. Existing reports do not establish parity for the current checkout.
Duplicate report bytes count once. Conflicting outcomes remain visible and cannot count as passes.

API declaration differences require review. Equivalent aliases or generic parameter names can produce differences.
An API match does not prove behavior, private layout compatibility, or complete type equivalence.
The report retains absent standard packages as unassessed. It does not infer that they are unreachable.
Blocked suites retain their unobserved test inventory. Individual portable tests can still need adapters.
No report claims a percentage of all reachable Go parity before that scope receives review.

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

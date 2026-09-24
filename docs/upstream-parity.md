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
| `language-targets.json` | Original language fixtures required on VM, AOT, JavaScript, and Wasm |
| `language-sources.json` | Reviewed recipe and companion-source fingerprints per Go SDK |
| `language-skips.json`, `language-rejections.json` | Shared language fixture exclusions and exact compile diagnostics |
| `scope-assessment.json` | Source review of absent packages, target boundaries, and next implementation steps |

The inventories retain eligible and ineligible declarations. Runtime outcomes
also include subtests and corpus cases. These are different denominators.
Benchmarks remain visible but do not count as assertion coverage.
Use the manifests and retained reports for current counts.

The scope assessment does not establish API or behavioral parity.
It separates portable candidates, host adapters, instrumentation, mixed contracts,
and explicit native limits. The report retains absent packages in the denominator.
A new SDK marks old assessments for review. A new package remains unassessed.

A package suite can contain explicit exclusions or partial assertions.
Thus a package count does not imply complete behavioral parity for that package.
A compile or runtime blocker never removes declarations from the denominator.
`runtime-blocked` identifies a suite that compiles but lacks a complete successful execution.
Its diagnostic records the observed failure. A successful build does not count as a behavior pass.

Omissions distinguish implementation gaps, test adapters, target limits, and
unassessed cases. They do not all represent unreachable behavior.
Deep recursion can require an iterative implementation before the same upstream
input can pass. An environment-controlled library setting does not require
native runtime control merely because Go names it through GODEBUG.
Native executable assumptions can require test adapters. Mixed `go/build`
assumptions still need individual review. The report retains all these groups.

The X.509 suites use the original certificate factory through native Go.
Both compilers receive the same generated certificates through source adapters.
The adapter preserves the original 4096-bit keys, SAN counts, and verification assertions.
Reports retain the generator source, original source, exact substitutions, output hashes, and process results.
This evidence covers certificate verification. It does not establish Dart key-generation performance.

The Linux fallback-root adapter replaces namespace mounts with private directories and certificate environment paths.
Both compilers receive the same fixture patch. The original assertions and separate child processes remain unchanged.
Reports retain the original source hash, patch, adapted source, and adapted source hash.

## Assertion integrity

The native baseline runs the installed Go source without Gart implementation overlays or assertion patches.
The X.509 adapters substitute the shared certificate inputs and Linux filesystem setup described above.
The 20 declared crypto suites use native Go `-tags=purego` to select portable implementations.
Their manifest entries and reports record that choice. A gate rejects any change to upstream test-file eligibility from those tags.
This configuration does not claim assembly implementation or native FIPS module parity.
The Gart side preserves the original assertions, expected values, and fixture bytes.
Adapters supply unavailable private test infrastructure without a duplicate assertion implementation.

The `go/build`, `debug/buildinfo`, and X.509 adapters supply the active Go SDK through `GOROOT`.
The full buildinfo suite uses its original `-all` option and selects the SDK platform's `pie` and `exe` fixtures.
Both Go and Gart read native artifacts from that SDK. This selection does not cover `c-shared` or cgo.
Gart preserves its compiler and architecture identity during source selection.
Its module lookup uses the SDK compiler identity because `go list` does not accept `gart` or `dart64`.
The ELF and Go symbol adapters supply a native SDK test executable as `os.Args[0]`.
Reports retain its build command and SHA-256. Original platform and absent-section skips remain explicit.

### Fuzz corpus replay

The `fuzz-replay` suite uses unchanged upstream CSV, string replacement, and PEM fuzz assertions.
Native Go generates inputs in an isolated module and a fresh fuzz cache.
Both native Go and Gart then replay the same retained corpus.
Short mode requests 1,000 native fuzz executions per target. Full mode requests 10,000.
The native fuzzer retains coverage-relevant inputs, not all generated inputs.
Thus this comparison covers the retained corpus, not all native fuzz executions.

```sh
python3 scripts/ci-local.py --platform linux --suite fuzz-replay --output build/fuzz-replay
python3 scripts/parity.py report --reports build/fuzz-replay --output build/fuzz-report
```

Each attempt retains the source wrapper, upstream license, corpus bytes, SHA-256 values, commands, SDK versions, and process results.
The report rejects a pass with missing inputs, changed files, unmatched test identities, or failed processes.
Keep a failed attempt intact. Reduce a failing corpus input before a permanent regression addition.

The inventory test rejects changes to upstream source, fixtures, build tags, and test patches.
Each SDK version has a separate inventory. A candidate inventory records reviewed
inputs without promotion of the release baseline. Exact `_go.mod` duplicates
from Go toolchain module distributions do not change the source fingerprint.
Different duplicate contents still change the fingerprint.
The coverage test rejects missing package classifications and lost package suites.
The compiler test rejects a new diagnostic for a known blocker.
The coverage test also requires a cause and reason for each blocker. [Blocker classifications](upstream-blockers.md) distinguish public API gaps from private test dependencies and native runtime boundaries.
It also requires promotion when a blocked package compiles successfully.

The result comparison rejects these conditions:

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
Full crypto suites retain the original pinned Wycheproof, Ed25519 vector, and X.509 Limbo modules.
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

The report also inventories all recipe directories from the active Go test runner.
It retains actions, options, source hashes, companion files, build eligibility, and missing adapters.
The source assertion rejects an SDK fingerprint change without review.
Language inventory counts establish no execution passes. Known rejections and exclusions remain visible.
The language runner and report use the same fixture selection and policy manifests.
Use `go run ./internal/cmd/parity -action language` for the standalone inventory.
Use `--language /path/to/inventory.json` to include a retained inventory.

The language adapter runs plain `run` recipes from all upstream recipe directories.
Both executions must exit successfully and match the original `.out` bytes.
An absent `.out` requires empty output. Only CRLF normalization follows the upstream runner.
With `GART_PARITY_REPORT_DIR`, language reports retain command arguments, exits,
output hashes, source hashes, exclusions, SDK versions, targets, and incomplete selections.
The aggregate refuses a pass if an output hash differs or either execution fails.
The `language-compile` suite compares exact `compile` recipes. Native Go compiles the original file to an object.
Gart compiles the same source and an inert Dart entry point to kernel.
The adapter uses a package URI for library imports to preserve Dart type identity.
It never executes compile-only programs or their initializers. Compilation does not establish runtime parity.
Compiler-option recipes remain unassessed until an adapter preserves their options.
Known compiler rejections retain exact diagnostics and never count as passes.
Compile evidence records the wrapper hash and remains separate from runtime evidence.

The `language-reject` audit compares exact `errorcheck` recipes. It preserves original source bytes and uses the installed SDK's annotation matcher.
Both compilers must reject the source and satisfy the active upstream policy.
Reports retain diagnostics, matcher results, and source fingerprints.
A successful diagnostic result requires all original assertions to pass.
The adapter reads expected failures from the upstream runner and requires both original matchers to fail for those fixtures.
Reports retain those results as `upstream-expected-failure`, separate from passed assertions.
Other diagnostic text and source-line mismatches count as failures.
Native assertion failures remain visible. Recipe options require separate adapters.

```sh
GART_PARITY_REPORT_DIR="$PWD/build/rejection-audit" bash scripts/ci.sh language-reject
```

This audit can fail on unresolved compiler differences. It is separate from the routine CI matrix.
The driver suite includes focused original rejection regressions through `TestRejectionContracts`.

The `language-generated` audit compares exact `runoutput` recipes. Native Go executes the unchanged generator.
Both compilers then receive its exact output bytes. Reports retain the generator command, exit, source bytes, and source hash.
Both programs must satisfy the original output assertion. This establishes parity for the generated program, not for generator execution.
Generator arguments require a separate adapter. This audit remains separate from the routine CI matrix because unresolved cases can fail.

The `language-directories` suite executes `runindir` recipes on VM, AOT,
JS, and Wasm. It copies original companion sources without import changes.
The adapter reads the default module version from the active upstream runner.
The `-gomodversion` recipe option overrides that default, including quoted versions.
Both programs use the same source directory and private `GOPATH` at runtime.
Reports retain the generated module bytes, hash, and original source fingerprints.
Native assembly, cgo, and hardware-fault fixtures retain explicit target exclusions.
Other recipe options remain unassessed.

```sh
GART_PARITY_REPORT_DIR="$PWD/build/directory-audit" bash scripts/ci.sh language-directories
```

The `language-targets` suite also includes these directory fixtures.

The `language-packages` audit compares `compiledir`, `rundir`, and `errorcheckdir` recipes on VM.
It also supports `compiledir -s` and `errorcheckdir -s`.
The `-s` option compiles each source file as a separate package, even when package names match.
Native Go compiles the original files in the upstream package order.
Gart places each package in a module subdirectory and changes only relative import paths within source files.
Go module paths reject Unicode characters that the native compiler accepts in relative imports.
For these fixtures, Gart uses indexed ASCII module paths and retains the original package names and source filenames.
Reports retain both path sets. Native compilation still uses the original paths.
It appends `.gart.go` to `_test.go` filenames because these recipes compile them as ordinary source files.
Reports retain original source hashes, adapted source bytes, package groups, compiler commands, and module and wrapper identities.
Both programs must satisfy the original expected output for `rundir`.
A `compiledir` pass requires native object compilation and Dart kernel compilation of all package imports.
An `errorcheckdir` pass requires both compilers to satisfy the original diagnostic assertions.
Expected upstream failures and exact known compiler rejections remain separate from passes.
Other recipe options remain unassessed.
The local CI matrix and manual GitHub workflow include all package-directory shards.

```sh
GART_PARITY_REPORT_DIR="$PWD/build/package-audit" bash scripts/ci.sh language-packages
```

```sh
GART_PARITY_REPORT_DIR="$PWD/build/generated-audit" bash scripts/ci.sh language-generated
```

The original `slice3` fixture inspects native slice memory layout through `unsafe`.
The `convinline` generator records host-specific results for out-of-range numeric conversions.
The audit retains these failures without weaker assertions or new exclusions.

The `language-targets` suite executes the selected original fixtures on all 4 Dart targets.
Its 4 shards keep these comparisons separate from the existing target fixtures.

Add `--applications build/app-results` to include application records in a report.
Repeat `--reports` for separate evidence directories. The report never blends incompatible configurations or compiler snapshots.
Reports without compiler provenance remain explicitly unattributed. Existing reports do not establish parity for the current checkout.
Duplicate report bytes count once. Conflicting outcomes remain visible and cannot count as passes.

API declaration differences require review. Equivalent aliases can produce differences.
Signature comparison ignores parameter names, including nested callbacks and generic parameters.
Generic constraints and references to distinct type parameters remain part of the contract.
An API match does not prove behavior, private layout compatibility, or complete type equivalence.
The report retains absent standard packages as unassessed. It does not infer that they are unreachable.
Blocked suites retain their unobserved test inventory. Individual portable tests can still need adapters.
No report claims a percentage of all reachable Go parity before that scope receives review.

Set `GART_PARITY_REPORT_DIR` to retain standard library comparisons and both raw outputs.
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

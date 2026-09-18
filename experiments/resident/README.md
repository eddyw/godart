# Dart resident compiler comparison

This benchmark compares 3 complete Go-to-execution paths:

| Mode | Request |
|---|---|
| `source` | `gart build`, dependency setup when required, then `dart run bin/main.dart` |
| `kernel` | The production `gart run` command with its source-kernel cache |
| `resident` | `gart build`, dependency setup when required, then `dart run --resident bin/main.dart` |

All modes include the Go frontend, package emission, and application execution.
The source and resident modes use separate subprocesses for dependency setup and execution.
The kernel mode uses the production driver for those operations.
All modes use `-format=false`, `GOMAXPROCS=4`, and `GOFLAGS=-p=2`.

Run the benchmark from the repository root:

```sh
python3 -B experiments/resident/probe.py --output build/resident-comparison
```

The default selects 5 trials for each workload and mode.
`--trials 1` provides a shorter validation run.
The output directory must be new and must reside under `build/`.
The benchmark requires Go, Dart with `--resident` support, Python 3.9 or later, and local loopback sockets.
It does not dispatch GitHub Actions or change production execution.

## Fixture

The fixture derives from the playground on 2026-09-18.
It retains JSON, fmt, time, pointers, maps, and the HTTP request followed by body close.
It replaces the external HTTP endpoint with a local empty response and replaces the clock with a fixed UTC time.
The benchmark normalizes only the pointer identities after `456 ` before output comparison.
These changes remove internet latency, clock variation, and native address variation.
The HTTP workload still executes the request and body close.
It does not measure external DNS, TLS, or response-body transfer.
The no-HTTP control removes the HTTP import and request block.

The benchmark changes `greet("world")` to `greet("earth")` for the source-edit scenario.
This same-size source edit changes observable program output.
All measured executions must match native Go output for their exact fixture state.
The local server also verifies the expected request count.
The benchmark leaves the user's `playground/` directory untouched.

## Scenarios and isolation

Each trial uses a different output directory for each mode.
The benchmark rotates and reverses mode order between trials.
It executes these scenarios in order:

1. Cold: a fresh Gart output, no resident process, and no resident kernel artifact.
2. Warm: the same source, output directory, and resident process.
3. Edit: the same output directory and resident process after the Go source edit.

Cold does not mean an empty machine cache. Go, Pub, and filesystem caches remain warm.
The benchmark resolves Dart dependencies on the cold request for all modes.
The fixture does not change manifests during later requests.
Each resident trial has a private compiler info file and temporary directory.
The temporary directory also isolates Dart's resident kernel cache.
The benchmark shuts down only its own resident compiler after each trial, including failed trials.
Shutdown time remains outside the request measurements and appears separately in the report.

## Evidence

The output directory contains raw command logs, fixture copies, generated Dart, kernel artifacts, and `trials.jsonl`.
`results.json` records tool versions, compiler and fixture hashes, source hashes, artifact hashes, load averages, and trial medians.
The `seconds` field measures the complete request through all required subprocesses.
The `dart` field measures compilation and execution, including kernel cache validation in production mode.
The production frontend field sums its reported phases. Other modes measure the `gart build` subprocess directly.
Thus those frontend fields have a small process-overhead difference. The complete request totals include that overhead in all modes.

This benchmark measures speed and output equivalence for the selected fixture.
It does not establish resident invalidation correctness for preserved Dart timestamps, SDK changes, runtime edits, or concurrent requests.
It also does not establish debugger, assertion, signal, or failure-recovery parity for a production resident integration.
Dart 3.13.3 selects changed resident inputs through source timestamps.
A production integration must preserve Gart’s content-hash guarantees when it coordinates that compiler.

## Results, 2026-09-18

The host uses an Apple M4 Max, macOS ARM64, Go 1.27.0, and Dart 3.13.3.
The baseline compiler revision is `da5d9a6`.
All rows show medians from 5 trials, in seconds, for complete requests.
Desktop background load remained active. No test suite overlapped these trials.
The rotated mode order reduces order bias, but the measurements do not describe an idle machine.

| Workload | Scenario | Source | Kernel cache | Resident |
|---|---|---:|---:|---:|
| HTTP | Cold | 13.080 | 10.973 | 10.736 |
| HTTP | Warm | 7.714 | 2.968 | 2.745 |
| HTTP | Edit | 8.135 | 5.810 | 2.790 |
| No HTTP | Cold | 5.259 | 4.167 | 4.003 |
| No HTTP | Warm | 3.235 | 1.164 | 1.260 |
| No HTTP | Edit | 3.637 | 2.326 | 1.085 |

Compared with the production kernel cache, resident execution reduces the edit total by 52% with HTTP and 53% without HTTP.
The HTTP Dart portion falls from 3.899 to 0.692 seconds after an edit.
The no-HTTP Dart portion falls from 1.475 to 0.265 seconds.
These durations include Dart compilation and program execution, not just compiler CPU time.
The Go frontend still runs on each request and limits the complete-request improvement.

Cold and warm differences are smaller and vary by workload.
The no-HTTP warm median is 8% slower with the resident compiler.
These results support resident compilation for source edits, not a claim that it improves all requests.

All 90 measured executions match native Go output after pointer normalization.
The local server records the expected 47 requests, including 2 native references.
The production cache compiles on all cold and edit requests and reuses kernels on all warm requests.
All 10 private resident compilers shut down successfully. No task-owned resident compiler remains alive after the run.
The separate initial probe also passed 18 execution comparisons and 2 compiler shutdowns.

[The measurement data](results-2026-09-18.json) retains all trial durations and the environment identity.
Raw logs and generated artifacts reside locally under `build/resident-comparison-20260918/`.

## Initial measurement decision

The initial measurement supported a bounded resident integration prototype for source edits.
Retain the current content-validated kernel cache as the unchanged-input path during that evaluation.
Measure both paths through the production driver before a default execution change.
Verify source, runtime, package configuration, SDK, and compiler-option invalidation first.
Also verify assertions, debugger behavior, arguments, input/output, exit codes, signals, and compiler failure recovery.

Do not rely only on resident timestamp detection when Gart already knows that source contents changed.
The resident process also retains compiler memory until shutdown.
This experiment does not measure that memory cost or establish a production lifetime policy.
No production code changed in this task. Direct-kernel generation remains a separate experiment.

## Content-validated session prototype

`session.py` evaluates resident compilation through the production Gart frontend.
`integration_probe.py` compares that prototype with the production `gart run` command.
The prototype adds no production CLI flag and changes no production default.
It requires Dart 3.13.3 and Python 3.9 or later.

The session owns 1 frontend compiler child and serves serial compile requests.
The caller must own its output directory exclusively and close the session after use.
Use `Session` as a context manager to terminate its child on success or failure.
Each program execution uses a fresh `dart run <kernel>` process.
The compiler retains frontend state across source edits. The application retains no runtime state.

The protocol uses the installed `frontend_server_aot.dart.snapshot` directly.
A cold request sends `compile`. Later edits send `recompile` with explicit changed file URIs.
After success, `accept` and `reset` retain frontend state and reset the delta base.
Thus each result is a complete kernel that a fresh Dart process can execute.
This protocol is internal to the pinned SDK. It is not a stable Dart API.

The content identity includes:

- The SDK path and hashes for its VM, frontend snapshot, platform kernel, and metadata.
- The entry point, package configuration bytes, compiler defines, fixed flags, and prototype source hash.
- All Dart source files under the generated output and resolved package library roots.
- The complete kernel hash.

The source scan deduplicates resolved paths before hashes.
A changed, added, or removed source produces an explicit invalidation URI.
The prototype does not depend on source timestamps.
A second input snapshot detects content changes during the compile request and prevents artifact publication.
It does not provide an atomic filesystem snapshot or support concurrent edits during compilation.
Compiler dependencies outside the source snapshot cause a failure.
Hidden Dart sources, directory symlink layouts outside the snapshot, and arbitrary external source imports are outside this bounded prototype.
The source scan can invalidate unused files. Production cache depfiles remain more selective.

Validated kernels and receipts reside under `.gart/resident-prototype/`.
An unchanged request validates the receipt and executes the kernel without a compiler request.
A new session can reuse that disk kernel without a compiler child.
The first changed request in a new session requires a cold compiler start.
Incremental compilation requires the same live session across edits.
Separate invocations of the Python script do not share a compiler.

SDK, package configuration, entry point, or compiler-option changes restart the compiler.
A corrupt artifact or receipt requires a new kernel.
A compiler error terminates its child, reports failure, and preserves the last valid receipt.
The prototype never executes that previous artifact as a fallback for a failed request.
The next valid request can start a new compiler.
Compiler requests have a 120-second timeout. Session close allows 3 seconds before forced termination.

Run the regression suite and complete-request comparison from the repository root:

```sh
python3 -B -m unittest discover -s experiments/resident -p test_session.py -v
python3 -B experiments/resident/integration_probe.py --output build/resident-integration
```

The comparison uses 5 trials with alternating mode order and separate output directories.
It uses the same HTTP fixture and no-HTTP control as the earlier benchmark.
It preserves the Go source timestamp during the same-size edit.
Both modes include the production Go frontend, dependency setup, and fresh application execution.
The prototype invokes `gart build`, then resolves dependencies when the production freshness rule requires it.
Its source scan and pre/post content hashes remain inside the request duration.
The production mode invokes the actual `gart run` command.
All measured outputs must match native Go after the same pointer normalization.
Cold requests use fresh output and compiler state. Go, Pub, and filesystem caches remain warm.
Session shutdown remains outside request durations and appears separately in the report.

The 11 regression tests cover:

- Cold requests, unchanged requests, timestamp-preserved edits, and disk reuse after session close.
- Transitive source additions, changes, removals, compiler errors, and repair.
- Runtime source changes, package configuration changes, SDK identity, defines, and unsupported SDK rejection.
- Artifact corruption, receipt corruption, worker termination, cleanup, and edits during compilation.
- Arguments, stdin, stdout, stderr, current directory, exit codes, product constants, and assertions.
- Generated Dart breakpoints and resume after an incremental source edit, with source execution as the reference.
- SIGINT and SIGTERM through fresh source and kernel processes.
- Go panic exit codes, absolute Go paths, and `-trimpath` paths through the production frontend.

These are focused prototype tests. They do not establish complete upstream Go parity, Windows support, or production daemon behavior.

## Integration results, 2026-09-18

The host uses the same Apple M4 Max, macOS ARM64, Go 1.27.0, and Dart 3.13.3.
The production baseline is `2463398`.
The table shows complete-request medians from 5 trials, in seconds.
Desktop background load remained active. No test suite overlapped these measurements.

| Workload | Scenario | Production kernel cache | Content-validated session |
|---|---|---:|---:|
| HTTP | Cold | 10.392 | 10.128 |
| HTTP | Warm | 2.389 | 2.501 |
| HTTP | Edit | 5.506 | 2.817 |
| No HTTP | Cold | 3.680 | 3.964 |
| No HTTP | Warm | 1.033 | 1.039 |
| No HTTP | Edit | 2.125 | 1.212 |

The prototype reduces the median edit total by 49% with HTTP and 43% without HTTP.
The Dart portion includes content validation, compilation when required, and program execution.
That portion falls from 3.714 to 1.077 seconds with HTTP, and from 1.386 to 0.448 seconds without HTTP.
The Go frontend still runs on each request.

Cold differences vary by workload and do not establish a cold-start improvement.
The HTTP unchanged request is 5% slower. The no-HTTP unchanged difference is less than 1%.
The conservative source scan adds overhead relative to the production depfile cache.
These results support a persistent development session for source edits.
They do not support a claim that a resident compiler improves all requests.

All 60 measured outputs match native Go after pointer normalization.
The loopback server records the expected 32 requests, including 2 native references.
All cold, unchanged, and edit requests use their expected compiler/cache state.
All 10 session compilers terminate after their trials.
[The integration data](integration-results-2026-09-18.json) retains all trial durations, phase durations, source hashes, and tool identities.
Raw logs and generated artifacts reside locally under `build/resident-integration-20260918/`.

The 11 regression tests pass on macOS ARM64 and local Linux AMD64.
Linux uses the pinned CI image, 2 CPUs, `GOMAXPROCS=2`, and `GOFLAGS=-p=2`.
The debugger and signal tests use a kernel after an actual Go source edit and incremental recompilation.
No GitHub Actions run or full upstream Go suite forms part of this prototype validation.

The next production task needs an explicit session lifecycle across CLI requests, output ownership, idle shutdown, and a memory budget.
Measure retained compiler memory across repeated edits before that integration.
Keep the exact SDK protocol dependency visible in that design.
The current Python session ends when its caller closes it and does not provide a shared daemon or watch command.
The production `gart run` command remains unchanged.

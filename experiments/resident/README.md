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

## Recommendation

Proceed to a bounded resident integration prototype for source edits.
Retain the current content-validated kernel cache as the unchanged-input path during that evaluation.
Measure both paths through the production driver before a default execution change.
Verify source, runtime, package configuration, SDK, and compiler-option invalidation first.
Also verify assertions, debugger behavior, arguments, input/output, exit codes, signals, and compiler failure recovery.

Do not rely only on resident timestamp detection when Gart already knows that source contents changed.
The resident process also retains compiler memory until shutdown.
This experiment does not measure that memory cost or establish a production lifetime policy.
No production code changed in this task. Direct-kernel generation remains a separate experiment.

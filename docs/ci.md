# Validation and CI

GitHub Actions requires manual dispatch. Pushes and pull requests do not start
runs. Local validation uses the same suite commands without Actions minutes.

The SDK manifest defines Go and Dart baselines. `scripts/Dockerfile.ci` pins
the Linux image and Node version. `scripts/ci-matrix.json` defines suites,
shards, resource limits, and separate short/full timeouts.

## Local commands

```sh
make get
make ci
```

Use FVM when the selected Dart SDK comes from Flutter:

```sh
fvm exec make get
fvm exec python3 scripts/ci-local.py --output build/local-short
```

The local runner retains an isolated source snapshot, SDK identities, durations,
logs, and parity reports. Host runs execute 1 suite at a time.
Use repeated `--suite` arguments and `--shard` for a smaller selection:

```sh
python3 scripts/ci-local.py --suite checks --suite driver --output build/core-validation
GOMAXPROCS=2 GOFLAGS=-p=2 bash scripts/ci.sh programs 0 6
```

Use [ci-matrix.json](../scripts/ci-matrix.json) for suite names and shard counts.
Shard indexes start at 0.
Without shard arguments, `ci.sh` executes the complete named suite.
The priority suite distributes selected packages across shards.
Only shard 0 also executes the shared HTTP and parity-harness tests.

Run filesystem tests as an unprivileged user. Root bypasses permission failures
that the upstream assertions require.

## Short and full modes

Routine upstream suites pass identical `-short` flags to native Go and Gart.
The original upstream code controls its short branches and iteration counts.
Short-mode exclusions remain explicit. They do not count as complete parity.

```sh
python3 scripts/ci-local.py --mode full --suite stdlib --suite upstream-targets --output build/full-validation
```

Full suites can take substantially longer. They remain a separate manual task.
See [upstream parity](upstream-parity.md) for evidence and assertion rules.

## Linux validation

```sh
docker build --platform linux/amd64 -f scripts/Dockerfile.ci -t godart-ci:go1.27.0-dart3.13.3-node26.8.2 .
python3 scripts/ci-local.py --platform linux --jobs 2 --output build/linux-short
```

The runner permits at most 2 Linux containers with 2 CPUs each.
Linux runs retain the image identity and use writable Go and Pub caches.
Base images have pinned digests. Support packages still resolve through Debian.

Add `--resume` with identical arguments to retry failed or absent jobs.
Successful jobs remain intact. Changed source, SDKs, selection, image, or
concurrency require a new output directory. Each attempt retains its own log.

## Shard duration updates

Refresh stdlib and selected upstream weights after a completed local run:

```sh
python3 scripts/ci-durations.py build/linux-short \
  --output internal/driver/testdata/ci-durations.json \
  --evidence build/ci-duration-evidence.json
```

The command retains separate short and full weights. It uses only completed,
passing package cases from attempts with a final result. A failed shard can
still supply durations for its passing cases. Repeated samples use the maximum.
The evidence records source identities, log hashes, and the measured durations.
Review the changed weights before a commit. Measure the revised shard allocation
before a timeout change.

## Budgets and remote execution

Use twice the largest relevant local duration, rounded up to whole minutes,
for a test timeout. Job timeouts also include setup and artifact allowance.
Do not reduce coverage to satisfy a timeout without an explicit scope decision.
`GART_TEST_TIMEOUT` overrides the local wrapper timeout.

Resident cache measurements require isolated runs. Concurrent compiler workloads
can evict retained state and invalidate an incremental-hit expectation.
The functional result must remain correct after a cold rebuild.

A manual Actions run uses **Actions → CI → Run workflow**.
The workflow input selects short or full upstream mode.
Public release preparation can restore automatic triggers after an explicit policy change.

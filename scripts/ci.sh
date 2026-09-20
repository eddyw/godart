#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
suite=${1:?usage: scripts/ci.sh checks|driver|applications|integration|targets|fuzz-replay|upstream-targets|upstream-priority|upstream-blockers|web-io|web-fetch|programs|language|language-compile|language-reject|language-targets|stdlib [shard-index] [shard-count]}
shard_index=${2:-0}
shard_count=${3:-1}
test_timeout=${GART_TEST_TIMEOUT:-30m}
started=$SECONDS
short_flag=-short=false

finish() {
  status=$?
  elapsed=$((SECONDS - started))
  printf '\nCI %s shard %s/%s: %ss, exit %s\n' "$suite" "$shard_index" "$shard_count" "$elapsed" "$status"
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    printf '| Suite | Shard | Duration | Exit |\n|---|---|---|---|\n| %s | %s/%s | %ss | %s |\n' \
      "$suite" "$shard_index" "$shard_count" "$elapsed" "$status" >> "$GITHUB_STEP_SUMMARY"
  fi
  exit "$status"
}
trap finish EXIT

if [[ $suite == fuzz-replay || $suite == driver || $suite == stdlib || $suite == upstream-targets || $suite == upstream-priority ]]; then
  if [[ ${GART_UPSTREAM_MODE:-short} == short ]]; then
    short_flag=-short=true
  elif [[ ${GART_UPSTREAM_MODE} != full ]]; then
    printf 'unknown upstream mode: %s\n' "$GART_UPSTREAM_MODE" >&2
    exit 2
  fi
fi

large_tests='TestRejectFilesOfGoTest|TestCompileFilesOfGoTest|TestUpstreamFuzzReplay|TestPrograms|TestLanguageTargets|TestRunFilesOfGoTest|TestVersionedGoFixtures|TestTestsOfGoroot|TestTests|TestAOT|TestWasm|TestJS|TestWebIO|TestWebFetch|TestUpstreamTargets|TestUpstreamCompileBlockers|TestSelectedUpstream|TestHTTPUpstream|TestHTTPNativeServer|TestParityHarness'
case "$suite" in
  checks)
    python3 -B -m unittest discover -s scripts -p test_ci_local.py
    python3 -B -m unittest discover -s scripts -p test_parity.py
    make fmt-check vet flutter-go dart-analyze dart-test
    package_list=$(go list ./...)
    packages=()
    while IFS= read -r package; do
      [[ $package == */internal/driver ]] || packages+=("$package")
    done <<< "$package_list"
    go test -v -p=1 -count=1 -timeout "$test_timeout" "${packages[@]}"
    ;;
  driver)
    go test -v -count=1 -timeout "$test_timeout" "$short_flag" ./internal/driver -skip "^(${large_tests})$"
    python3 -B -m unittest discover -s scripts -p test_resident.py -v
    ;;
  integration) test_name=TestTests ;;
  applications)
    python3 -B scripts/parity.py applications --shard-index "$shard_index" --shard-count "$shard_count" --output "${GART_PARITY_REPORT_DIR:-build/parity-results}/applications"
    ;;
  targets) test_name='TestAOT|TestWasm|TestJS' ;;
  language-targets) test_name=TestLanguageTargets ;;
  language-compile) test_name=TestCompileFilesOfGoTest ;;
  language-reject) test_name=TestRejectFilesOfGoTest ;;
  fuzz-replay) test_name=TestUpstreamFuzzReplay ;;
  upstream-targets) test_name=TestUpstreamTargets ;;
  upstream-blockers) test_name=TestUpstreamCompileBlockers ;;
  upstream-priority)
    go test -count=1 ./internal/parity
    test_name='TestSelectedUpstream|TestHTTPUpstream|TestHTTPNativeServer|TestParityHarness'
    ;;
  web-io) test_name=TestWebIO ;;
  web-fetch) test_name=TestWebFetch ;;
  programs) test_name=TestPrograms ;;
  language) test_name='TestRunFilesOfGoTest|TestVersionedGoFixtures' ;;
  stdlib) test_name=TestTestsOfGoroot ;;
  *) printf 'unknown CI suite: %s\n' "$suite" >&2; exit 2 ;;
esac

if [[ $suite != checks && $suite != driver && $suite != applications ]]; then
  go test -v -count=1 -timeout "$test_timeout" "$short_flag" ./internal/driver \
    -run "^(${test_name})$" -shard-index "$shard_index" -shard-count "$shard_count"
fi

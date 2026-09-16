#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
suite=${1:?usage: scripts/ci.sh checks|driver|integration|targets|web-io|web-fetch|programs|language|stdlib [shard-index] [shard-count]}
shard_index=${2:-0}
shard_count=${3:-1}
test_timeout=${GART_TEST_TIMEOUT:-30m}
started=$SECONDS

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

large_tests='TestPrograms|TestRunFilesOfGoTest|TestTestsOfGoroot|TestTests|TestAOT|TestWasm|TestJS|TestWebIO|TestWebFetch'
case "$suite" in
  checks)
    make fmt-check vet flutter-go dart-analyze dart-test
    package_list=$(go list ./...)
    packages=()
    while IFS= read -r package; do
      [[ $package == */internal/driver ]] || packages+=("$package")
    done <<< "$package_list"
    go test -v -p=1 -count=1 -timeout "$test_timeout" "${packages[@]}"
    ;;
  driver)
    go test -v -count=1 -timeout "$test_timeout" ./internal/driver -skip "^(${large_tests})$"
    ;;
  integration) test_name=TestTests ;;
  targets) test_name='TestAOT|TestWasm|TestJS' ;;
  web-io) test_name=TestWebIO ;;
  web-fetch) test_name=TestWebFetch ;;
  programs) test_name=TestPrograms ;;
  language) test_name=TestRunFilesOfGoTest ;;
  stdlib) test_name=TestTestsOfGoroot ;;
  *) printf 'unknown CI suite: %s\n' "$suite" >&2; exit 2 ;;
esac

if [[ $suite != checks && $suite != driver ]]; then
  go test -v -count=1 -timeout "$test_timeout" ./internal/driver \
    -run "^(${test_name})$" -shard-index "$shard_index" -shard-count "$shard_count"
fi

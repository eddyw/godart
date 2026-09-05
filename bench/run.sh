#!/bin/sh
# Runs each case on the 4 runtimes and writes out.csv.
#
# The machine must stand near idle. A parallel build moves each number by a
# large factor, thus the script prints the load average before the run.
set -e
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/.." && pwd)
data=${DATA:-$root/flutter/example/compass/assets/activities.json}

if [ ! -x "$root/build/gart" ]; then
  echo "build/gart is missing. Run make build from $root first." >&2
  exit 1
fi

echo "== the load before the run =="
uptime

cd "$here/go"
go vet ./...
go build -o "$here/gobench" ./cmd/gobench
"$root/build/gart" build -o "$here/gen" -runtime "$root/runtime/goruntime" ./...

cd "$here/dart"
dart pub get >/dev/null
dart compile exe bin/bench.dart -o "$here/bench.exe" >/dev/null

: >"$here/out.csv"
echo "case,runtime,ns,count" >>"$here/out.csv"
"$here/gobench" "$data" >>"$here/out.csv"
"$here/bench.exe" "$data" >>"$here/out.csv"
if command -v bun >/dev/null 2>&1; then
  bun "$here/js/bench.js" "$data" >>"$here/out.csv"
fi

echo "== the load after the run =="
uptime
echo "out.csv holds $(wc -l <"$here/out.csv") lines"

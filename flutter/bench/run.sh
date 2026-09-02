#!/bin/sh
# Runs the whole measurement of the text property and writes report.md.
#
# The machine must stand near idle. A parallel build moves each number by a
# factor of 3, and the script prints the load average before and after, thus a
# reader can throw a run away.
set -e
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
runs=${RUNS:-5}

if [ ! -x "$root/build/gart" ]; then
  echo "build/gart is missing. Run make build from $root first." >&2
  exit 1
fi

echo "== the load before the run =="
uptime

cd "$here/go"
go vet ./...
go test ./...

"$root/build/gart" build -o "$here/gen" -runtime "$root/runtime/goruntime" ./...

cd "$here/dart"
dart pub get >/dev/null
dart compile exe bin/bench.dart -o "$here/bench.exe" >/dev/null

: >"$here/out.csv"
i=1
while [ "$i" -le "$runs" ]; do
  echo "== process run $i of $runs, $(uptime) =="
  "$here/bench.exe" >>"$here/out.csv"
  i=$((i + 1))
done

echo "== the load after the run =="
uptime

dart run bin/report.dart "$here/out.csv" >"$here/report.md"
echo "report.md holds the tables, and out.csv holds each batch."

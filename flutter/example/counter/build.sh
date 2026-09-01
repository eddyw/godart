#!/bin/sh
# Compiles the Go application to Dart, into the package gen.
set -e
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../../.." && pwd)
cd "$here/go"
go vet ./...
go test ./...
"$root/build/gart" build -o "$here/gen" -runtime "$root/runtime/goruntime" ./...
echo "gen holds the Dart output of ./go"

#!/bin/sh
# Compiles the Go application to Dart, into the package gen, and writes the
# Flutter host of the node types that the application names.
set -e
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../../.." && pwd)
cd "$here/go"
go vet ./...
go test ./...
"$root/build/gart" build -o "$here/gen" -runtime "$root/runtime/goruntime" ./...
echo "gen holds the Dart output of ./go"
cd "$root/flutter/nodegen"
dart run bin/nodegen.dart host "$root/flutter/goflutter" \
  "$here/lib/widgets_gen.dart" "$here/go"

#!/bin/sh
# Runs an example, and it writes the Dart again after each change of the Go
# source. gart watch compiles, and SIGUSR1 asks flutter run for a hot reload.
#
# The keys of flutter run stay: r reloads, R restarts, and q stops.
#
# dart format costs 10 seconds of each cycle for a large example, thus the
# watch build writes the Dart with no format. build.sh formats it again.
#
# A change that names a new node type of gart.dev/ui needs build.sh again,
# because nodegen writes lib/widgets_gen.dart. The host names such a node in
# its error, and it asks for nodegen.
set -e

example=$1
device=${2:-macos}
if [ -z "$example" ]; then
  echo "usage: watch.sh <counter|apilist|form|compass> [macos|ios|android|linux|windows|web]" >&2
  exit 2
fi

here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
app="$here/$example"
if [ ! -f "$app/pubspec.yaml" ]; then
  echo "no example at $app" >&2
  exit 1
fi
if [ ! -f "$app/lib/widgets_gen.dart" ]; then
  echo "run $example/build.sh first, because lib/widgets_gen.dart is absent" >&2
  exit 1
fi

pidfile=$(mktemp)
# The trap runs on the clean stop of flutter run. Ctrl-C reaches each process
# of the group already. kill 0 ends gart watch and the loop below.
trap 'rm -f "$pidfile"; kill 0' EXIT

cd "$app/go"
"$root/build/gart" watch -o "$app/gen" -format=false -runtime "$root/runtime/goruntime" ./... |
  while IFS= read -r line; do
    echo "$line"
    case "$line" in
    'gart: 0 compiled'*) ;;
    'gart: '*' compiled'*)
      if [ -s "$pidfile" ]; then
        kill -USR1 "$(cat "$pidfile")"
      fi
      ;;
    esac
  done &

cd "$app"
flutter run -d "$device" --pid-file="$pidfile"

#!/bin/sh
# Measures the Flutter work that reuse of 1 unchanged widget avoids.
#
# The run needs a profile build: a widget test runs in debug mode, where the
# Dart VM interprets the code and each assert of the framework runs. A profile
# build drops both, thus only this run gives a number that an application sees.
#
# The build opens a window of the application on macOS. The machine must stand
# near idle, exactly as flutter/bench/run.sh asks.
set -e
here=$(cd "$(dirname "$0")" && pwd)
runs=${RUNS:-3}
flutter=${FLUTTER:-flutter}

cd "$here"
# git holds no runner directory, because each file comes from flutter create.
"$flutter" create --platforms=macos --project-name skipbench . >/dev/null
rm -rf .idea .metadata README.md ./*.iml test/widget_test.dart lib/main.dart
"$flutter" pub get >/dev/null

i=1
while [ "$i" -le "$runs" ]; do
  echo "== run $i of $runs, $(uptime) =="
  "$flutter" drive \
    --driver=test_driver/integration_test.dart \
    --target=integration_test/skip_test.dart \
    -d macos --profile 2>&1 |
    grep -E '^flutter: (nodes|[0-9]|slope|reuse_ns)'
  i=$((i + 1))
done

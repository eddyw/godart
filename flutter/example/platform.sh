#!/bin/sh
# Writes the platform runner of an example. git holds no such directory,
# because flutter test needs none and each file comes from flutter create.
set -e

example=$1
platform=${2:-macos}
if [ -z "$example" ]; then
  echo "usage: platform.sh <counter|apilist|form> [macos|ios|android|linux|windows|web]" >&2
  exit 2
fi

here=$(cd "$(dirname "$0")" && pwd)
app="$here/$example"
if [ ! -f "$app/pubspec.yaml" ]; then
  echo "no example at $app" >&2
  exit 1
fi

cd "$app"
flutter create --platforms="$platform" . > /dev/null

# flutter create writes a template test and 3 editor files that the example
# does not want. The template test names a widget that no example declares.
rm -rf .idea .metadata README.md ./*.iml test/widget_test.dart
git checkout -- pubspec.lock 2>/dev/null || true

# The macOS sandbox blocks an outgoing connection until the entitlement names
# it. apilist reads an HTTP API, and form sends one.
if [ "$platform" = "macos" ]; then
  for f in macos/Runner/DebugProfile.entitlements macos/Runner/Release.entitlements; do
    if ! grep -q "network.client" "$f"; then
      /usr/bin/sed -i '' \
        's|<key>com.apple.security.app-sandbox</key>|<key>com.apple.security.network.client</key>\n\t<true/>\n\t<key>com.apple.security.app-sandbox</key>|' \
        "$f"
    fi
  done
fi

echo "$example holds a $platform runner."
echo "Run it: cd $app && flutter run -d $platform"

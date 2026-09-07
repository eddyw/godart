# Compass v2

Compass v2 implements the original Flutter Compass sample with new Go screens
and state. It runs on macOS and iOS with the original light and dark themes.
The system appearance selects the theme.

The example includes login, trip search, continent selection, dates, guests,
destination results, activity selection, trip details, trip deletion, and native
share. The original catalog contains 137 destinations and 2723 activities.

## Run

Use Go 1.27, Flutter 3.47.2, and Xcode with the required platform SDK.
Place `go`, `flutter`, and `dart` on `PATH`.

From the `godart` repository root:

```sh
make build
cd flutter/example/compass_v2
../../../build/glutter run -d macos
```

For iOS, start a simulator or connect a device. Use its ID from `flutter devices`:

```sh
../../../build/glutter run -d <device-id>
```

The native iOS project uses `dev.gart.compassV2`. A physical device requires a
development team and a provisioning profile in Xcode.

The login fields contain the sample credentials. Press **Login** to enter.
The example uses the sample development authentication and local data. It needs
no separate server. Trips remain in memory until the application exits.
The home screen starts with the sample Alaska trip.

The 10 original login and continent photographs ship as local assets.
They display without network access. Destination and activity images use the
original network service. That service returned HTTP 403 on 2026-09-06.
Those images use the original sample error widgets.

For a standalone macOS application:

```sh
../../../build/glutter gen
flutter build macos --release
open 'build/macos/Build/Products/Release/Compass v2.app'
```

## Validate

Generate the Dart output before the Flutter tests:

```sh
../../../build/glutter gen
flutter analyze
flutter test test/compass_test.dart
flutter test integration_test/compass_test.dart -d macos
flutter test integration_test/compass_test.dart -d <ios-simulator-id>
python3 tool/compare_upstream.py
```

Run `go test ./...` from `go/` for the state tests.

The comparison tool downloads the pinned original sample. It compares 32 PNG
captures across 8 screens, 2 window sizes, and both themes. Both applications
use the original photographs, equal catalog image fixtures, and exact fonts. The tool requires
byte equality and writes `doc/parity.json`. Native tests use the real image
services, local assets, and plugins. See [the validation record](doc/validation.md) for results
and limits.

## Source

The reference is [Flutter Compass](https://github.com/flutter/samples/tree/463e365e4842f252ffab9c6198594a504d69469f/compass_app),
commit `463e365e4842f252ffab9c6198594a504d69469f`.

- `go/screens.go` defines the screen trees and interactions.
- `go/model.go` owns the local catalog, search state, and trips.
- `lib/host.dart` maps example nodes to native Flutter widgets.
- `lib/theme/` preserves the original theme values.
- `go/data/`, `assets/logo.svg`, and `assets/user.jpg` come from the sample.
- `assets/photos/` contains exact copies from the sample's `rstr.in` URLs.
  `sources.json` records each source URL and SHA256.
- `assets/fonts/` contains the exact Rubik and Open Sans faces from the
  sample's `google_fonts` package. Each family includes its font license.

The Flutter source uses the BSD license in [LICENSE](LICENSE). Generated Dart
files, build outputs, and comparison PNGs stay outside version control.

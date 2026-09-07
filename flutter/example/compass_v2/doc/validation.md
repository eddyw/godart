# Validation record

Date: 2026-09-06.

The reference is Flutter samples commit
`463e365e4842f252ffab9c6198594a504d69469f`.
The validation environment used Go 1.27, Flutter 3.47.2, and Dart 3.13.2.

| Validation | Result |
|---|---|
| Go state tests | 3 pass |
| Flutter analyzer | No issues |
| Widget tests | 3 pass |
| Native macOS trip flow | Light and dark themes pass |
| Native iOS trip flow | Light and dark themes pass on iPhone 17 Pro, iOS 26.2 |
| Original sample comparison | All 32 PNG captures match byte for byte |
| macOS release build | Pass |
| iOS simulator application build | Pass |

The complete trip flow covers login, search guards, continent selection, dates,
guests, results, activity selection, trip creation, trip reopen, trip deletion,
the deletion snackbar, and logout. It also verifies activity reset after a
destination change. The desktop widget test covers the mouse carousel, a live
system theme change, and the native share request with a valid button rectangle.

The share request test intercepts the plugin channel. It verifies the trip text
and sheet origin. A native share sheet interaction remains unverified.
No physical iOS device test forms part of this record.

## Visual comparison

`tool/compare_upstream.py` creates an independent application from the pinned
original source. It preserves the original screen and theme code. It selects
the sample local repositories and development authentication for the test.

The capture sizes are 430×932 and 800×900, with a pixel ratio of 1.
Each size uses both themes. The 8 states are login, home, search, date dialog,
complete search form, results, activities, and trip details.

Both applications use the 10 original login and continent photographs.
Compass v2 loads the local assets. The reference loads the same bytes through
its network image cache. Both applications use the original `user.jpg` as a
fixture for the unavailable destination and activity images.
Thus the comparison measures layout, typography, colors, image transforms,
and controls under equal image inputs. It does not measure image service
availability. The native tests use the local photographs and real network image cache.

Both applications load the same Roboto and Material Icons from the Flutter SDK.
The other faces match these original `google_fonts` assets:

| Face | SHA256 |
|---|---|
| Rubik Medium | `6f341ec62d7f8662cb57c4d637699f8bd012b4d692be97112640d003eb4c6063` |
| Rubik ExtraBold | `1aca64f1e87e2319783c7572ac8664212508aa65404b31a7cb051d534e67b636` |
| Open Sans Medium | `a8496c5a42a57ee2bf5fdab6e10b9258496e99f276727c7f94859f40ca39c34e` |

Each download uses `https://fonts.gstatic.com/s/a/<SHA256>.ttf`.

## External image limit

The original `storage.googleapis.com/tripedia-images` destination and activity
images returned HTTP 403. The application preserves the source URLs and error
widgets. It does not substitute unrelated photographs. Native requests to
`rstr.in` did not complete in this environment, although command-line requests
returned the images. Thus Compass v2 includes those 10 original photographs
as assets. `assets/photos/sources.json` records their URLs and hashes.

## Native launch

The macOS release application and the iOS simulator application both launch.
Native integration tests exercise the complete interface on both platforms.
The macOS test tool reports `Failed to foreground app; open returned 1` in this
session. The tests still render the interface, save native captures, and pass.
The session cannot capture the macOS window through `screencapture`.

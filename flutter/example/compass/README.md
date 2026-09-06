# compass

This directory holds the Flutter sample
[compass_app](https://github.com/flutter/samples/tree/main/compass_app), with
the whole application in Go. The sample is the reference architecture of the
Flutter team: MVVM, a repository for each source, a use case, and a server.

The port answers 1 question: **what can a Go program build with Flutter
today, and where does the model stop?** The section "What the port could not
hold" names each limit that the work found, with the cause of each 1.

## The parts

| Path | Content |
|---|---|
| `server/` | The server of the sample in Go. 9 routes over `net/http` and `encoding/json`. |
| `go/` | The application. The model, the API client, each repository, the use case, each view model, and the 7 screens. |
| `lib/` | The Dart host. 24 lines, and `widgets_gen.dart` that `nodegen` writes. |
| `assets/` | The 2 JSON files of the sample, the logo, the picture of the account, and the 2 fonts. |
| `doc/` | 1 picture of each screen. |
| `tool/` | `shot.dart` writes those pictures. |

## Run it

```sh
make build                     # from the root of the repository
cd flutter/example/compass/server
go run . &                     # the server on the port 8080
cd ..
glutter gen                    # go vet, go test, and gart build into gen
flutter test                   # 5 widget tests
```

`glutter run -d macos` opens the application against the server, and it writes
the platform runner when that directory is absent.
`flutter build macos --debug` wrote `compass_app.app` on 2026-09-06.

The login screen carries the account of the sample, thus the button needs no
keyboard.

**The image host of the sample answers 403.** The bucket
`storage.googleapis.com/tripedia-images` gives `AccessDenied` to each request
of a destination and of an activity, thus the sample itself draws no such
picture today. The 2 JSON files of `assets/` name a host that answers, and the
seed of each URL holds the reference of the destination, thus 1 place keeps 1
picture. The 3 cards of the login screen and the 7 continents keep the host of
the sample, because `rstr.in` answers each of those requests.

## The screens

`doc/` holds 1 picture of each screen. `flutter test tool/shot.dart
--update-goldens` writes them again.

| Screen | Picture |
|---|---|
| The login, with the 3 tilted cards and the logo | `doc/1-login.png` |
| The home, with the title under a radial gradient | `doc/2-home.png` |
| The search form, with the carousel of the continents | `doc/3-search.png` |
| The same form after 2 choices | `doc/4-search-filled.png` |
| The date range dialog of Material | `doc/4b-dates.png` |
| The grid of the results, with the tag chips | `doc/5-results.png` |
| The 2 lists of the activities | `doc/6-activities.png` |
| The booking, with the gradient over the picture | `doc/7-booking.png` |

The tool loads the 2 fonts of the application and it answers each request of an
image with 1 small picture, thus the shot holds the text and the layout of each
screen. It stands outside `test/`, thus `flutter test` does not run it: a
golden file reads the font of the machine, and a second machine would give
another picture.

`glutter run -d macos` also holds the Go source under watch. A change of 1 Go
line reaches the running application in about 1 second, and it keeps the screen
and the state. Read `docs/glutter.md`.

The server also runs on the Dart VM. `server/prove.sh` runs it 2 times, first
with `go run` and then with the Dart output of gart, and it diffs the 2
answers of the 17 requests of `server/requests.sh`. The 2 answers are equal.

## The size of the port

| Part | The sample | The port |
|---|---|---|
| The application | 5,804 lines of Dart, and 3,331 lines that `build_runner` writes | 2,675 lines of Go |
| The server | 620 lines of Dart over `shelf`, and 14 generated files | 300 lines of Go over `net/http` |
| The host of the application | — | 24 lines of Dart, and 1,450 lines that `nodegen` writes |
| The packages of pub.dev | 11 | 2: `flutter_svg` for the logo, and `share_plus` for the share sheet |

The port needs no code generation of the data classes. A Go struct tag and
`encoding/json` hold what `freezed` and `json_serializable` write in the
sample.

## The proof

| Test | Count | What it covers |
|---|---|---|
| `go test ./go/...` | 11 | The whole flow with no Flutter: the login, the home, the search, the calendar, the results, the activities, and the booking. It reads a real HTTP server of `httptest`. |
| `flutter test` | 5 | The same flow in a real Flutter tree, with a real navigator, a real page transition, and a real socket. |
| `go test ./server/...` | 8 | Each route of the server, and each status of the sample. |
| `server/prove.sh` | 17 requests | The Go run and the Dart run of the server give the same bytes. |

## What the port found

**2 changes outside this directory.**

1. `ui.Pages` carries the stack of the navigator. The host builds 1
   `MaterialPage` for each child, and a back gesture of the platform reaches
   `OnPop`. The generated `Navigator` carries no page, because a Go value
   holds no Flutter `Page`. Before this node, a Go application held 1 screen.
2. `GoRuntime.tick` runs the queue of the scheduler again. A posted function
   can start a goroutine, for example the command of a new screen. The
   scheduler already left its loop when the idle hook runs, thus such a
   goroutine waited for the next gesture of the user. A screen that opens and
   reads the server at once never reached the server. Read decision 353.

## What the port could not hold

**Nothing of the sample stays out.** The port of 2026-09-05 named 8 items, and
the work of 2026-09-06 closed each 1. The table holds the answer of each.

| The item of the sample | The answer of the port |
|---|---|
| The theme of `ui/core/themes/` | `go/theme.go`. `ui.ColorScheme` holds each of the 47 colors of the Flutter class and it derives each 1 that the program leaves out. `ui.UseTheme` names the 2 themes, and `material.MaterialApp` of the Go tree carries them to Flutter. `ui.Theme()` gives the same values to each build, thus no `BuildContext` stands in the way. |
| `google_fonts` | The package exports no widget, thus no pack of `nodegen` reaches it. `assets/fonts/` holds Rubik and Open Sans, and `ui.TextStyle` names the axis `wght` of each variable font. |
| `flutter_svg` | A pack of `nodegen`. `gart.dev/ui/svg` holds 4 node types, and the login screen draws the real `assets/logo.svg`. |
| `cached_network_image` | `ui.Picture`. It gives a node while the image loads and a node after a failure. The cache of the sample is a plugin of the file system, and the port holds the cache of Flutter alone. |
| `share_plus` | `ui.Share` reaches the handler of `goSetCall` in `lib/main.dart`, and that handler calls the plugin. |
| `Dismissible` | `ui.Dismissible`. `nodegen` writes a Go field `Key` for each widget whose constructor demands a `Key`, and `ConfirmDismiss` waits for the server and keeps the row after a failure. |
| `showDateRangePicker` | `ui.DateRange`. The call reaches the host through `ui.Call`, the host opens the dialog of Material under the `ui.Anchor` node, and the answer wakes the goroutine of the tap. |
| `ScaffoldMessenger` | `ui.SnackBar`. The same path, and the answer names the button that the user pressed. |
| `go_router` | `ui.Pages` and a stack of routes in Go. It holds no URL and no deep link. |
| `provider` | The `App` struct holds each view model. |
| `shared_preferences` | `ui.SaveValue` and `ui.ReadValue` reach the host. The port keeps the token in memory, exactly as the development flavor of the sample does, thus a restart asks for the login again. |
| `AppLocalization` | The strings live in `go/text.go`. |

**The 3 defects that the work found**, each in a tool and not in the compiler:

1. An optional key field wrote `ValueKey<String>("")` in the host, thus the
   `center` of a `CustomScrollView` named a sliver that no key carries and the
   home screen did not build.
2. `glutter` kept the binary of `nodescan` in its cache and it never built it
   again, thus a change of the scanner never reached an application.
3. `BackdropFilter` of the generated set carries no filter, because the
   parameter of Flutter takes an `ImageFilter`. The node `ui.Blur` names the 2
   sigma values.

## The origin

The sample carries the BSD license of the Flutter team. `assets/` holds the 2
JSON files of the sample with no change. The Go source is a port, and it
follows the architecture and the behavior of the sample.

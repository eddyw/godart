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
| `lib/` | The Dart host. 101 lines, and `widgets_gen.dart` that `nodegen` writes. |
| `assets/` | The 2 JSON files of the sample: 137 destinations and 2,723 activities. |

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
`flutter build macos --debug` wrote `compass_app.app` on 2026-09-05.

The login screen carries the account of the sample, thus the button needs no
keyboard. The destination images stay empty, because the image host of the
sample answers 403 to each request. The 2 JSON files hold the URLs.

`glutter run -d macos` also holds the Go source under watch. A change of 1 Go
line reaches the running application in about 1 second, and it keeps the screen
and the state. Read `docs/glutter.md`.

The server also runs on the Dart VM. `server/prove.sh` runs it 2 times, first
with `go run` and then with the Dart output of gart, and it diffs the 2
answers of the 17 requests of `server/requests.sh`. The 2 answers are equal.

## The size of the port

| Part | The sample | The port |
|---|---|---|
| The application | 5,804 lines of Dart, and 3,331 lines that `build_runner` writes | 1,827 lines of Go |
| The server | 620 lines of Dart over `shelf`, and 14 generated files | 300 lines of Go over `net/http` |
| The host of the application | — | 101 lines of Dart, and 1,324 lines that `nodegen` writes |
| The packages of pub.dev | 11 | 0 |

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

| The item of the sample | The cause | What the port holds |
|---|---|---|
| `go_router` | The node set holds widgets, and a router is no widget. A pack of nodegen reads the widgets of a package of pub.dev, and `go_router` exports none that the mapping covers. | `ui.Pages` and a stack of routes in Go. It holds no URL and no deep link. |
| `provider` | The same cause. | The `App` struct holds each view model. |
| `shared_preferences` | A plugin needs a platform channel. `ui.Call` reaches the host now, thus a port of this row costs 1 case of the handler. | The token lives in memory. A restart asks for the login again, exactly as the dev flavor of the sample does. |
| `flutter_svg`, `google_fonts`, `cached_network_image`, `share_plus` | The 3 first ones export a widget, thus a pack of nodegen reaches each 1. `share_plus` is a call, and `ui.Call` reaches it. | A text logo, the default font, `Image.network` with no cache, and a share button that does nothing. |
| `Dismissible`, to delete a booking with a swipe | The constructor takes a `Key`, and `nodegen` drops each widget that needs 1. Read the section "What the tool drops" of `flutter/nodegen/README.md`. | A delete button in the row. |
| `showDateRangePicker` | The call gives a `Future` and it needs a `BuildContext`. A node describes a widget, and not a call. The generated `DateRangePickerDialog` also carries no callback, because the dialog gives its answer to `Navigator.pop`. | A screen of its own holds a real `CalendarDatePicker`. The first tap names the first day, the second one names the last day, and Go holds the 2. |
| `AppLocalization` | `Localizations.of(context)` needs a `BuildContext`. | The strings live in `go/text.go`. |
| The theme of `ui/core/themes/` | `Theme.of(context)` needs a `BuildContext`. The host reads the theme and it passes a snapshot to the Go build. | The default Material theme of the host, and `ui.Theme()` for the colors of the snapshot. |

**Each limit above stands on 1 of 2 causes: a package that exports no widget of
the mapping, or a Flutter API that needs a `BuildContext`.** No limit comes from the Go
subset of the compiler. The whole application, the whole server, and each
model of the sample compile with no change of the compiler.

## The origin

The sample carries the BSD license of the Flutter team. `assets/` holds the 2
JSON files of the sample with no change. The Go source is a port, and it
follows the architecture and the behavior of the sample.

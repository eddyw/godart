# What Flutter needs first

The user asks which language features and which packages come first, and the
user assumes that the contact with Flutter is solved. This file answers that
question in 2 ordered lists. `docs/flutter-design.md` holds each shape of a
binding and each measurement of the Flutter SDK. This file holds the order of
the work, and it takes the target of a client app: a widget, a state, and a
call of an HTTP API.

Each claim below names a file and a symbol. A line number rots on the next
commit, thus this file names none. The state is of 2026-09-01.

**The state of 2026-09-02.** Milestone 1, milestone 2, and milestone 3 of
`docs/flutter-design.md` are complete, and the directory `flutter/` holds them.
The node set holds 564 types of `widgets`, `material`, and `cupertino`, with
4975 fields. That work answers each item of this file. Read the status line of each 1, and read
`docs/flutter-state.md` for the state model that the application uses.

## What already works

Do not plan this work again. The list corrects 2 rows of the option survey of
2026-08-28. `docs/flutter-design.md` holds each correction.

| Item | Evidence |
|---|---|
| The scheduler is a guest of the Dart event loop. `$schedule` runs the queue until it is empty and returns, and a timer or a future starts it again. Flutter owns the loop already. | `$schedule` of `sched.dart` |
| A Go function value is a plain Dart closure. Its Dart type is `R Function(P...)` over the Dart form of each Go type. | `funcType` of `internal/emit/type.go` |
| A Dart event that starts Go work needs 2 lines. The finalizer of the runtime already runs `$go(body)` and `$schedule()`. | `_cleanups` of `sched.dart` |
| A Go call of an async Dart API blocks the goroutine and holds no `Future`. `$await` writes a frame, and the event loop wakes it. | `$await` of `sched.dart` |
| The output holds no `async` and no `await`. The unwind model replaced that form on 2026-08-29, 1 day after the Flutter research. A function that can block keeps its Dart return type, and the extra cost is about 1.6 nanoseconds for each frame. A build path of Go code thus pays no async cost. | `docs/block-model.md` |
| An https request verifies the certificate against the trust store of the platform. Only `InsecureSkipVerify` drops the check. | `SecureSocket.secure` and its `onBadCertificate` of `net.dart` |
| A gzip answer decompresses already. `compress/gzip`, `compress/flate`, and `compress/zlib` read the real Go source, and the client sets `Accept-Encoding` itself. | `docs/support.md`, `docs/parity.md` |
| A third-party Go module compiles when it stays in the subset. **This row was false until 2026-09-02.** Stage 2 accepted the import, and stage 5 then failed with "the package X has no import prefix", because `compiled` of the driver read the main module and the surface only. Decision 217 fixed it. | `NativePaths` of `internal/overlay/native.go`, `compiled` of the driver |

## The compiler work, in order

### 1. An external Dart declaration &mdash; not needed for A13

`intrinsic` of stage 5 is a flat switch on the name of each interop function,
thus each Dart call needs an edit of the compiler. A Flutter binding names
thousands of constructors, and that switch does not scale.

A Go function declaration with no body already parses, and stage 2 rejects a
read of it. Give such a declaration a marker that names a Dart library and a
Dart name, and emit a direct call of it. A widget then costs 1 Go declaration,
and no compiler change.

Evidence: `intrinsic` and `intrinsicText` of `internal/emit/call.go`, and
`docs/what-compiles.md`. Cost: medium.

**Status of 2026-09-03. Done.** The directive `//gart:dart` names a Dart
library and a Dart symbol, and stage 5 writes a direct call of it. The value
boundary carries `bool`, `int`, `float64`, `string`, and `dart.Object`, and
stage 2 rejects each other type. Read decision 239 and the section "An external
Dart declaration" of `docs/design.md`.

A13 still needs no such declaration, because the Dart host holds each widget
call. The item earns its cost for an approach that makes a Go type a Dart
widget class, and for item 2.

**The tool chain, measured 2026-09-02.** `go build` rejects a bodyless
declaration with `missing function body`. `go vet` accepts it, and `gopls`
reports nothing. A control file with a type error gives a gopls diagnostic,
thus the probe stands. The editor thus costs this item nothing, and a module
that carries such a declaration needs a twin file behind a build tag.

### 2. An opaque Dart handle in Go &mdash; not needed for A13

A widget, a controller, and a `BuildContext` must live in a Go value. 5
interop tables already hold that pattern with an `int` handle: the socket, the
file, the process, the HTTP/2 session, and the TLS connection. Thus a first
app needs no compiler change at all.

A named type `dart.Object` that emits as `Object?` removes the table, the
lookup, and the risk of a stale handle. Only `unsafe.Pointer` maps to `Object?`
today, and stage 2 rejects `unsafe`.

Evidence: `dartType` of `internal/emit/type.go`. Cost: small with an int,
medium with a type.

**Status of 2026-09-02.** Milestone 1 holds no Dart handle in a Go value. The
host owns each Flutter object, and the Go tree carries data alone.

**Status of 2026-09-03. Done.** `syscall/dart` holds the type, and the emitter
gives it the Dart form `Object?`. The 2 conditions of the note below each hold:

1. **A producer.** Item 1 is done, thus a `//gart:dart` declaration gives such
   a value. A Go program still makes none of its own.
2. **A twin declaration.** It holds as written. `testdata/programs/dartobj`
   carries a file for each build tag and an alias of its own, and `make
   compare` runs the Go form and the Dart form and diffs the 2.

Read decision 240.

**The measure of 2026-09-02 lowered the value of this item, and the value types
took its place.** A count of the blocking parameter of each of the 273 dropped
constructors gave 176 distinct Dart types, and the largest held 11. The 15
value types of decision 231 and decision 246, and the 3 identity keys of
decision 247, took 16 of the 273 without any Dart handle in a Go value.

### 3. A callback export, in 2 flavors &mdash; done 2026-09-02

Flutter calls Go, and the 2 directions of that call need 2 rules.

**A callback with no result, which may block.** Wrap the Go closure as
`() { $go(f); $schedule(); }`. This is the finalizer pattern of the runtime.
It fits `onPressed`, `onChanged`, and a listener of a stream.

**A callback with a result, which must not block.** Dart calls the Go closure
directly. A function that blocks writes its frame to `$blocked` and returns the
zero value, and only a Go caller tests that field. A Dart caller reads the zero
value and sees no error, thus a `build` method that blocks gives a wrong widget
with no report. The rule of the project rejects such a compromise.

The compiler must reject a blocking function in that position, and the
information already exists: `blockingExports` reads the block set of stage 3 for
each exported function. The rule is a small pass, and not a new analysis.

Evidence: `blockingExports` of `internal/driver/driver.go`, and the machine of
`internal/emit/machine.go`. Cost: small. This item makes a build method safe.

**Status of 2026-09-02.** Both flavors work. The first one needed no change,
and the host of `flutter/goflutter` runs each event through `$go`. The second
one is milestone 2, and it is done: a named function type that carries
`//gart:sync` names the position, and a pass after stage 3 rejects a function
that can block there. Read decision 221 and the milestone 2 section of
`docs/flutter-design.md`.

### 4. A library entry point &mdash; not needed

`writeMain` emits `bin/main.dart`, and the program ends when the first goroutine
returns. A Flutter app needs the opposite: a `lib/` library, an init that starts
the runtime and keeps the isolate alive, and a pubspec that holds the Flutter
dependency.

Evidence: `writeMain` of `internal/emit/emit.go`, and `$init` of `sched.dart`.
Cost: small.

**Status of 2026-09-02.** A Go application that is a library package needs no
change. `gart build` writes `lib/<name>.dart` and no `bin/main.dart`, and the
Flutter application calls `$init` of that library. Take this item only for a
`package main`.

### 5. The value boundary &mdash; enough for milestone 1

The interop rule converts an integer, a `GoString`, a byte slice, and a record.
A widget property also needs a double, a bool, a list, a map, and a Go struct as
a Dart value, in both directions. A text property crosses the boundary at each
build, thus the cost of `GoString` matters at 60 frames each second.

Evidence: `internal/emit/call.go`, `docs/js-integers.md`. Cost: medium.

**Status of 2026-09-02.** The boundary carries each type that the node set
needs: a `GoString`, a double, a bool, a `GoInt`, a slice of interface values,
and a closure of 4 shapes. A `GoInt` erases to a Dart int on the Flutter
target.

**The cost, measured 2026-09-03. The concern is false.** A whole tree rebuild
of 200 nodes with a text of 20 characters costs 61.4 microseconds, which is
0.37% of the 16,700 microsecond frame. The normal case, 1 signal and 1
boundary, costs 0.58 microseconds. A screen needs about 20,000 text properties
before the `GoString` conversions take 10% of a frame.

2 things cost more than the text. `GoString.fromString` ran 3 to 6 times the
cost of `toDartString`, and the adopted buffer of `GoString.take` closed 27% to
32% of that on 2026-09-03. The read of the optional fields cost 37.4
microseconds of a 200 node frame against 8.04 for the text, and the read by
position of the same day brought it to 20.3 against 6.85. Read decision 248,
and `flutter/bench/README.md` for each number.

## The compiler work that a first app does not need

A binding where Go calls Dart, and where a hand written Dart adapter holds the
widget classes, needs none of the items below. Each one belongs to the shape
where a Go type **is** a Dart widget class, which is A4 of
`docs/flutter-design.md`.

| Item | Why a first app lives without it |
|---|---|
| `extends`, `implements`, `with`, `@override` | The Dart adapter holds the class, and the Go side gives it a closure. |
| A named parameter and a default value | The Go side takes an option struct, and the adapter names each parameter. |
| A getter and a setter | A Go method carries the same value. |
| A const constructor | Flutter skips a subtree of a canonical const widget. Its loss costs a rebuild, and not a failure. |
| A super call, and a Dart subclass of a gart class | Out of reach, and no first app reads either one. |

## The packages, in order

The target is a client, and not a server. `docs/support.md` holds the count of
each package.

### 1. `encoding/json`, with `time` &mdash; done 2026-09-01

This is the largest gap of a client, and it fails silently today. A verified
run of 2026-09-01 with 1 struct of 3 fields:

| Value | Go | godart |
|---|---|---|
| `time.Time` in a struct | `"when":"2023-11-14T22:13:20Z"` | `"when":{}` |
| `[]byte` in a struct | `"data":"aGk="` | `"data":[104,105]` |
| `Unmarshal` of an RFC 3339 text into a `time.Time` | The value | The error `json: cannot unmarshal string into Go value of type time.Time` |

`Marshal` gives no error in the 2 wrong cases. The cause is 1 missing feature:
the overlay reads no `Marshaler` and no `Unmarshaler`, thus the walk of the
descriptor writes the unexported fields of `time.Time` as an empty object.

Needed, in order: the 2 interfaces, the base64 rule of a byte slice, the 6
marshal methods of `time.Time`, `RawMessage`, `Decoder`, `Encoder`, and
`Number`.

### 2. The client gaps of `net/http` &mdash; done 2026-09-01

The request path works: `Get`, `Post`, `Client.Do`, a header, a redirect, a
timeout, a cookie of 1 request, and the gzip answer. 3 gaps hit a real client:

- No cookie jar, thus a session API needs a header of its own.
- No `mime/multipart`, thus no file upload.
- No `Transport.Proxy`, thus no system proxy.

### 3. The small crypto packages &mdash; done 2026-09-01

`crypto/rand` gives a token and an idempotency key. `crypto/hmac` and
`crypto/sha512` sign a request and read a JWT. Each one follows a pattern that
the surface already holds, and each cost is small.

### 4. The general focus list &mdash; done 2026-09-02

The 9 items of the focus list of `docs/support.md` are done, except
`reflect.MakeFunc`. The surface grew from 92 import paths to 113, and from 66
full packages to 86.

### Out of scope for a client

The HTTP server, `os/exec`, and `os/signal`. A mobile app reads its storage
through a Flutter plugin, and not through `os`.

**Status of 2026-09-02.** `os/signal` and `text/template` each build now. The
work of the focus list of `docs/support.md` took them, and a client still reads
neither one.

## The 3 programs that prove the path

Each one is small, and together they cover the whole mechanism. The 3 are done
since 2026-09-02, and `make flutter-test` runs each 1.

1. **A counter.** Done 2026-09-02. `flutter/example/counter` holds it, and 6
   widget tests pass under Flutter 3.47.0. It needed no item of the compiler
   list.
2. **A list from an API.** Done 2026-09-02. `flutter/example/apilist` holds
   it, and 6 widget tests pass under Flutter 3.47.0. It reads the issue list of
   `golang/go`, it decodes each `created_at` into a `time.Time`, and it gives a
   `ListView` of cards. It needed no item of the compiler list, and it found
   the bug of decision 229.
3. **A form.** Done 2026-09-02. `flutter/example/form` holds it, and 6 widget
   tests pass under Flutter 3.47.0. 3 text fields, a validation in Go on each
   keystroke, and a POST of the fields as JSON. It needed no item of the
   compiler list. `net/mail` stands outside the subset, thus the address rule
   of the form reads the text itself.

## The fourth program: a whole application

**The port of `compass_app`.** Done 2026-09-05, and 1 for 1 since 2026-09-06.
Commit `f2652a3` retains the earlier `flutter/example/compass` source.
The cleanup removed the example on 2026-09-14. Compass v2 remains at
`flutter/example/compass_v2`. The sample of the Flutter team carries the
reference architecture: MVVM, a repository for each source, a use case, and a
server. The port holds 2,675 lines of Go against 5,804 lines of Dart, and it
needs no code generation of a data class.

It needed 2 items outside the example, and decision 353 holds the 2:

- `ui.Pages` carries the stack of the navigator, thus a Go application holds
  more than 1 screen. Item 6 of the note list of `docs/flutter-design.md`
  named the gap, and this node closes the part that a page stack needs.
- `GoRuntime.tick` runs the queue of the scheduler again. A goroutine that a
  posted function starts waited for the next gesture of the user, thus a
  screen that reads the server when it opens read nothing.

**The port named 8 limits on 2026-09-05, and the work of 2026-09-06 closed each
1.** The theme, the 2 fonts, the logo, the picture of a network, the share
sheet, the swipe of a `Dismissible`, the date range dialog, and the snack bar
each reach a Go program now. The README and captures of the earlier example
remain in commit `f2652a3`. The retained example has current validation in
`flutter/example/compass_v2/doc/validation.md`.

3 mechanisms closed the 8:

- `ui.ThemeStyle` carries a color scheme, a text theme, and the decoration of
  an input, and the Go tree holds `material.MaterialApp` itself. Read decision
  367.
- `ui.Call` carries a call that needs a `BuildContext`, and the node
  `ui.Anchor` keeps that context. Read decision 370.
- A pack of `nodegen` reads a package of pub.dev with no change of the reader,
  and `flutter_svg` proves it. Read decision 371.

No limit came from the Go subset of the compiler.

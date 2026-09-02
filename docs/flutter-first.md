# What Flutter needs first

The user asks which language features and which packages come first, and the
user assumes that the contact with Flutter is solved. This file answers that
question in 2 ordered lists. `docs/flutter-options.md` holds the 6 shapes of a
binding and the measurement of the Flutter SDK. This file holds the order of
the work, and it takes the target of a client app: a widget, a state, and a
call of an HTTP API.

Each claim below names a file and a line. The state is of 2026-09-01.

**The state of 2026-09-02.** Milestone 1, milestone 2, and milestone 3 of
`docs/flutter-design.md` are complete, and the directory `flutter/` holds them.
The node set holds 414 types of `widgets`, `material`, and `cupertino`, with
3851 fields. That work answers each item of this file. Read the status line of each 1, and read
`docs/flutter-state.md` for the state model that the application uses.

## What already works

Do not plan this work again. The list corrects 2 rows of
`docs/flutter-options.md`, which stands from 2026-08-28.

| Item | Evidence |
|---|---|
| The scheduler is a guest of the Dart event loop. `$schedule` runs the queue until it is empty and returns, and a timer or a future starts it again. Flutter owns the loop already. | `runtime/goruntime/lib/src/sched.dart:125` |
| A Go function value is a plain Dart closure. Its Dart type is `R Function(P...)` over the Dart form of each Go type. | `internal/emit/type.go:128` |
| A Dart event that starts Go work needs 2 lines. The finalizer of the runtime already runs `$go(body)` and `$schedule()`. | `runtime/goruntime/lib/src/sched.dart:188` |
| A Go call of an async Dart API blocks the goroutine and holds no `Future`. `$await` writes a frame, and the event loop wakes it. | `runtime/goruntime/lib/src/sched.dart:240` |
| The output holds no `async` and no `await`. The unwind model replaced that form on 2026-08-29, 1 day after the Flutter research. A function that can block keeps its Dart return type, and the extra cost is about 1.6 nanoseconds for each frame. A build path of Go code thus pays no async cost. | `docs/block-model.md` |
| An https request verifies the certificate against the trust store of the platform. Only `InsecureSkipVerify` drops the check. | `runtime/goruntime/lib/src/net.dart:347` |
| A gzip answer decompresses already. `compress/gzip`, `compress/flate`, and `compress/zlib` read the real Go source, and the client sets `Accept-Encoding` itself. | `docs/support.md`, `docs/parity.md` |
| A third-party Go module compiles when it stays in the subset. **This row was false until 2026-09-02.** Stage 2 accepted the import, and stage 5 then failed with "the package X has no import prefix", because `compiled` of the driver read the main module and the surface only. Decision 217 fixed it. | `internal/overlay/native.go:82`, `internal/driver/driver.go` |

## The compiler work, in order

### 1. An external Dart declaration &mdash; not needed for A13

`intrinsic` of stage 5 is a flat switch on the name of each interop function,
thus each Dart call needs an edit of the compiler. A Flutter binding names
thousands of constructors, and that switch does not scale.

A Go function declaration with no body already parses, and stage 2 rejects a
read of it. Give such a declaration a marker that names a Dart library and a
Dart name, and emit a direct call of it. A widget then costs 1 Go declaration,
and no compiler change.

Evidence: `internal/emit/call.go:46`, `docs/what-compiles.md`. Cost: medium.

**Status of 2026-09-02.** A13 needs no such declaration. The Dart host holds
each widget call, and Go names no Dart symbol. Take this item only for an
approach that makes a Go type a Dart widget class.

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

Evidence: `internal/emit/type.go:64`. Cost: small with an int, medium with a
type.

**Status of 2026-09-02.** Milestone 1 holds no Dart handle in a Go value. The
host owns each Flutter object, and the Go tree carries data alone.

**The item stays open, and the measure of 2026-09-02 lowered its value.** A
count of the blocking parameter of each of the 273 dropped constructors gave
176 distinct Dart types, and the largest holds 11. `FocusNode` blocks 3, and
`Decoration` blocks 2. Thus `dart.Object` unlocks a handful of constructors,
and not the group that the earlier note named.

2 conditions must hold before the item earns its cost:

1. **A producer.** A Go program cannot make a `FocusNode` or a
   `ScrollController`. Only the host can. Thus `dart.Object` needs item 1, the
   external Dart declaration, or a host call that gives such a value.
2. **A twin declaration.** `syscall/dart` carries `//go:build dart`, thus a Go
   module that names `dart.Object` no longer builds with the go command. A
   module of the Flutter path must hold a pair of files, and an alias of its
   own, to keep the rule that `go test` runs the whole screen with no Flutter.

The 9 value types of decision 231 took the part of the surface that needs
neither condition.

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

Evidence: `internal/emit/machine.go:349`, `:469`, `internal/driver/driver.go:274`.
Cost: small. This item makes a build method safe.

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

Evidence: `internal/emit/emit.go:633`, `runtime/goruntime/lib/src/sched.dart:176`.
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
target. The cost of a text property at 60 frames each second is unmeasured.

## The compiler work that a first app does not need

A binding where Go calls Dart, and where a hand written Dart adapter holds the
widget classes, needs none of the items below. Each one belongs to the shape
where a Go type **is** a Dart widget class, which is option 4 of
`docs/flutter-options.md`.

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

The HTTP server, `os/exec`, `os/signal`, `text/template`, and `database/sql`.
A mobile app reads its storage through a Flutter plugin, and not through `os`.

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

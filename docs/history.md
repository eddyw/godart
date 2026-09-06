# History

This file holds the record of the work. It holds each milestone, each session,
each commit, each defect that a measurement found, and each verified fact. No
session needs this file to continue the work. Read it to answer 1 question: did
the work already measure this, and what did it find?

## The milestones

The steps, in order. Each one is done, and each one passes its subset of the
tests in CI.

- M0. Done. The name, the repository, the Go module, the `goruntime` package,
  and the CI setup.
- M1. Done. `package main`, functions, integers, strings, slices, structs,
  methods, and the verifier.
- M2. Done. Interfaces, maps, and closures.
- M3. Done. `defer`, `panic`, and `goto`.
- M4. Done. Goroutines, channels, `select`, and the isolate worker API with the
  async model.
- M5. Done. Generics, and the generic method of Go 1.27.
- M6. Done. The `reflect` subset, the parity file `docs/parity.md`, and `fmt`.
- M7. Done. The overlays for `time`, `sync`, `strings`, `strconv`, `sort`,
  `testing`, and `encoding/json`, and the network with a client and a listener.
- M8. Done. The build driver, the metadata cache, watch mode, and the
  documentation.
- M9. Done. The commands `gart run` and `gart test`. The `testing` overlay, the
  test variants of stage 1, the stub overlays, and the end to end test of
  `testdata/tests`.
- M10. Done. The unnamed struct type, and 16 packages of the standard library.
  The core type of a type parameter, the patch overlay, and the 13 defects that
  the real library source found.
- M11. Done. The benchmark support of `gart test`.
- M12. Done. The file access, the `TestMain` hook, and a pointer to any type.
  Also a `return` inside the body of a range loop over a function, and
  `encoding/json` with no code generator. The surface grew from 17 packages to
  22. The real `testing.B`, the search of the count of iterations, the ahead of
      time build, the monotonic clock, and the measurement of the async model.
- M13. Done. The block model of GopherJS. The flat state machine of stage 4, the
  frame class of stage 5, the scheduler of the runtime, and the box of a
  captured variable. The cost of a chain of 20 frames fell from 3900 to 189
  nanoseconds.
- M14. Done. The package `bytes`, `json.Unmarshal`, and the method value of a
  named type that is not a struct. Also the type descriptor of a type parameter,
  and a labelled branch that leaves the body of a range loop over a function.

- M15. Done. 1 Dart class for each unnamed struct type of a build. Each shape
  takes a file of its own, and the loop variable of Go 1.22 takes a copy for
  each pass.

- M16. Done. The `context` package. It took a calendar for `time.Time.String`,
  an `internal/reflectlite` overlay, and 2 new intrinsics.

- M17. Done. A deferred call that can block, and the source position of a test
  log line. Also a descriptor for an instance of a generic named type, and a
  wider surface of the overlays.

- M18. Done. The calendar of `time`, a wider `reflect`, `t.Setenv`, and
  `t.Context`. Also the network of UDP and of a Unix socket, with a deadline and
  a cancel, and `math/rand` with `math/rand/v2`. The surface grew from 32
  packages to 36.

- M19. Done. The accessors of `time.Time`, `Parse`, `ParseInLocation`,
  `ParseError`, `FixedZone`, `ParseDuration`, `Truncate`, and `Round`.

- M20. Done. 15 more packages of the standard library.

- M21. Done. A method with a nil pointer receiver. Such a method of a struct
  type holds its body in a top level function.

## The sessions

Each table names the commits of 1 session, in order. `git log --oneline` gives
the same hashes.

### 2026-09-06, the twenty-third session

The user parked the name of the project, and asked for each open task end to
end. The user also asked for a clone of the Flutter sample compass_app that
holds each part of the sample, and named the goal of a first alpha release:
the syntax and the features of Go must carry a test. 3 streams ran: this
session for Flutter, 1 agent for the standard library and the runtime, and 1
agent for a conformance suite of the Go language.

**The port of compass_app is 1 for 1 now.** The port of 2026-09-05 named 8
items that it could not hold, and each 1 closed. `ui.ThemeStyle` carries a
color scheme of 47 colors, a text theme of 15 styles, and the decoration of an
input, thus a Go program names its own theme and `material.MaterialApp` of the
Go tree carries it to Flutter. `ui.Call` carries a call that needs a
`BuildContext`, thus `showDateRangePicker` and the `ScaffoldMessenger` reach
Go. `flutter_svg` joined the packs of `nodegen`, thus the login screen draws
the real logo of the sample. `Dismissible` joined the node set, because a
widget that demands a `Key` now carries a Go field of the type string.
`tool/shot.dart` writes 1 picture of each of the 8 screens into
`flutter/example/compass/doc/`.

**The image host of the sample answers 403.** The bucket of the destinations
and of the activities gives `AccessDenied` to each request, thus the sample
itself draws no such picture today. The 2 JSON files name a host that answers.

**7 defects of the tools fell.** The font weight table held 7 names, thus a Go
program that wrote `w600` read no weight and the text drew with the default.
`nodegen` skipped a private field formal, thus 4 node types did not compile,
and a callback of 1 enum named a table that `writeEnums` did not write, thus 4
more did not. An optional key field wrote `ValueKey<String>("")`, thus the
`center` of a `CustomScrollView` named a sliver that no key carries. `glutter`
kept the binary of `nodescan` in its cache and it never built it again.
`BackdropFilter` of the generated set carries no filter, thus each blur of the
sample drew nothing. The search bar of the results screen wrote the hint in the
place of the query.

**The value files join the node filter.** `themes.part.dart` of the compass
output fell from 139,198 bytes to 8,021, and `types.part.dart` from 69,670 to
53,650.

**The lazy package variable saves 508,896 bytes, and dart2wasm rejects it.**
The type flow analysis stops with the message of decision 323. A bisect named
the cause: 2 map variables of `net/http` break it together. The work stays
open, and decision 373 holds the measure and the next step.

**A conformance suite of 24 programs covers the Go language.** The 24 write
2,184 labelled lines, and each 1 matches real Go byte for byte. The suite found
17 defects, and 10 closed. 2 of the 10 give a silent wrong answer: a `range`
over an array read the array itself, and a short declaration wrote a name in
front of the read of a later value.

### 2026-09-06, the twenty-second session

The user asked 3 things: run the memo of the decode against the real
benchmark, read what GopherJS does with the same problem, and add a benchmark
of an integer, a float64, and a basic type. The group of the server stays out,
because the question is the cost of a client.

**A GoString holds the decode and the hash after the first read.** A Go string
never changes, thus 1 decode and 1 hash hold for the life of the value. The
measurement ran A, B, A, because the machine drifted 4.5 percent between the 2
baselines. A screen of 500 labels of 20 bytes falls from 7.65 to 0.78
microseconds, 200 labels of 500 bytes fall from 35.09 to 0.32, and a string of
2 megabytes that crosses again falls from 777 microseconds to 18 nanoseconds.
`map.string.100k` falls from 76.24 to 58.59 milliseconds, and the decode of the
document falls 7 percent. A string that no code reads 2 times pays 1 to 3
percent for the field.

The first mix of the hash was the Jenkins form of Dart, and it crashed
dart2wasm with the message of decision 323. That decision names a call that
fills an optional parameter of a method of `GoString`, and the Jenkins mix
holds no parameter. The rule is thus narrower than the defect. The mix is
FNV-1a with a mask of 26 bits now, and the 3 wasm tests pass.

**14 scalar cases join the benchmark.** The integer of gart costs what a Dart
int costs, and it holds 64 bits. `int.sum.1m` and `int.hash.1m` each sit at 1.0
against Dart and against Go, and `int.div.1m` sits at 0.7. bun needs a BigInt
for a 64 bit multiply, thus `int.hash.1m` costs 23.09 milliseconds there
against 1.02. That is the first case where the ahead of time build of Dart
stands far ahead of the JIT of bun.

**The benchmark found 1 defect.** `math.Sqrt` calls `math.sqrt`, and the
compiler of Go replaces that call with 1 machine instruction. gart holds no
such rule, thus it compiles the portable Go form: a loop of 54 steps of 64 bit
integer arithmetic. `float.math.1m` costs 144 milliseconds under gart and 767
microseconds under Dart, a factor of 188. `Exp`, `Log`, `Floor`, `Ceil`,
`Trunc`, `Max`, and `Min` each take the portable path too.

**GopherJS takes the opposite decision, and it is right for JavaScript.** A Go
string is a JS string with 1 byte in each code unit, thus `s[i]` is
`charCodeAt(i)` and `len(s)` is `s.length`. `$externalize` gives the same
string back when each code unit stands under 128, thus the boundary is free for
ASCII, and GopherJS holds no memo. A JS engine holds a rope and a sliced
string, and Dart holds neither: the substring of gart shares the byte storage
and it beats the substring of Dart, 4.72 milliseconds against 5.91.

**5 changes of the speed and of the correctness followed, and each 1 carries a
witness that compares the output against real Go.**

**math.Sqrt compiled to a loop of 54 steps.** The compiler of Go replaces
`math.sqrt` with 1 machine instruction, and gart holds no such rule. `Sqrt`,
`Floor`, `Ceil`, `Trunc`, and `Abs` read the instruction of Dart now, and IEEE
754 names 1 correct result for each input of the 5. `float.math.1m` falls from
144 milliseconds to 0.74.

**A conversion of an integer to a float64 cost 5 times too much.**
`int.toDouble` is a virtual call that the ahead of time compiler does not fold,
because a Dart int holds 2 forms. The operator + of int reads a double operand
and gives the same value. `float.sum.1m` falls from 3.43 milliseconds to 0.69,
and `struct.alloc.1m` falls from 3.90 to 0.68, which is 5 times less than the
same loop in Dart and equal to Go.

**strconv.FormatInt joined 1 digit at a time to the front.** 1 `Itoa` of 9
digits thus allocated 9 Go strings. `int.format.100k` falls from 21.97
milliseconds to 1.96, and `map.string.100k` falls from 57.92 to 25.27 with it.

**The float formatter held 7 defects.** `%.0f` of 0.5 gave 1 and Go gives 0,
because Dart rounds away from zero and Go rounds to the even digit. A precision
above 20 crashed the program, because `toStringAsFixed` of Dart rejects such a
count. The verbs `b` and `x` of `strconv.FormatFloat` gave an error text. 3
rules of the verb stood wrong. `testdata/programs/floattext` gave 6,155
differences of 28,414 lines, and each 1 is closed.

**A cache of the type descriptor handle gave 0 gain and it went back.** A JIT
profile of the JSON encode put 13 percent of the time in a map keyed by the
full Go name of a type. The ahead of time build showed no difference, and the
median of 22 cases stood at 0.995. A profile of the Dart VM service ranks the
wrong work for an ahead of time build.

**2 agents held the other 2 streams.**

**A byte slice holds its storage in a field of the type `Uint8List`.**
`Slice.array` held a `List<T>` whose real class varies, thus each read of a
byte took a dispatch over many targets. `bytes.count` now costs what the same
loop costs in Dart, and `slice.append.1m` fell 20 percent. A generic form of
the subclass was a loss over the whole program, and 3 dense classes were worse
than 1: the gain comes from the count of the classes.

**The overlay of `strings` holds the real `Replacer` of Go**, and the real
tests of `encoding/json` run. That package went from "does not compile" to
"compiles and runs", and `TestUnmarshal` passes. 8 items landed, and 1 of them
closed a silent wrong answer: `reflect.Value.SetMapIndex` gave each key of a
`map[K]Struct` the same object.

**glutter filters the node types of an application at the build.** compass
names 40 of the 491 node types that its 2 packs hold. The whole Dart output
falls 12.2 percent, and the tree of `gart.dev/ui` falls 76.3 percent. The
filter rides the overlay of the loader, thus nothing on disk changes and the
type checker of Go validates the filtered source. A stale filter is a loud
error of the compile.

**The scan of the `strings` overlay reads bytes now.** `Index`, `Count`,
`Split`, `SplitSeq`, and `Replace` built 1 Go string for each position of the
text. `str.split.100k` fell 47 percent. A new witness also found 2 defects of
an empty separator, and both are closed.

**A Go string of ASCII skips the UTF-8 encoder**, and it holds that text as its
own decode. A write of 100,000 bytes into Go fell from 195.32 microseconds to
51.63, and `json.encode.typed` fell 28 percent.

Read decisions 357 to 366.

### 2026-09-05, the twenty-first session

The user asked for each of the 15 open tasks, end to end. 3 streams ran: the
main session held the Flutter model, 1 agent held the performance of the
standard library, and 1 agent held the driver and the size of the output. Each
agent worked in a git worktree of its own, thus no build output collided.

**1 node pack for each Dart package.** Flutter moved Material and Cupertino
out of the framework, thus the node set reads `material_ui` 1.1.1 and
`cupertino_ui` 1.0.2 of flutter.dev now. The set grows from 564 types to 566,
and each pack stands in a Go package of its own: `gart.dev/ui` with 273 types,
`gart.dev/ui/material` with 218, and `gart.dev/ui/cupertino` with 75. The
interface `ui.Node` holds the method `IsNode` that answers `ui.Mark`, thus a
struct of another module joins the node set with no change of the reader.

The generated host imports the Dart package of each case that it holds and the
converter of each enum that those cases read. The host of compass falls from
3,195 lines to 1,324, and the descriptor library of the core falls from
1,911,012 bytes to 948,233.

**glutter.** 1 command runs the Go build and the Flutter command. It replaces
`build.sh`, `platform.sh`, and `watch.sh`, and git holds none of the 3 now. A
change of 1 Go line reaches the running application in 1 second, and the hot
reload takes 182 milliseconds. `glutter doctor` reads the version stamp that
nodegen writes, and it names each difference against the Flutter of the
machine.

**`ui.Call` asks the host for a value.** The bridge ran 1 way before this
change. A goroutine parks on a channel, the host runs the handler of
`goSetCall`, and `ui.Answer` wakes the goroutine. The counter example holds the
proof, and 2 widget tests cover the answer and the error.

**Each node type carries the dartdoc of Flutter.** `documentationComment` of
the analyzer gives the text, thus `go doc ui.Column` answers with the words of
the Flutter team. The 3 packs grow from 8,994 lines to 17,851, and the Dart
output does not change.

**`dart format` reads the files of the build alone.** An incremental build of
compass falls from 11.94 seconds to 1.78. The work also found 2 defects: the
old format collapsed an empty function body of a cached package and it moved
each source map line after it by 1, and the file list of a source map kept the
order of a map, thus 2 builds wrote 38 different map files for `httpuse`.

**The start time of a large program.** The compass server takes 5.41 seconds
to its first answer under `dart run`, and 0.41 seconds as an ahead of time
build. The Go binary takes 0.012.

**A shake at the level of the package gives 0 bytes.** Each package of the
output stands in the import closure already, and each `$init` calls the `$init`
of each import. The tables of `unicode` need a walk at the level of the
package variable, and that walk needs 2 proofs that the compiler lacks.

### 2026-09-05, the twentieth session

The user asked for each open task, end to end, and for a port of the Flutter
sample `compass_app`. 2 agents ran in a git worktree of their own, and the main
session held the port, the Flutter host, and the docs.

**The port of `compass_app` runs.** The sample of the Flutter team carries the
reference architecture of Flutter: MVVM, a repository for each source, a use
case, and a server. `flutter/example/compass` holds the whole application in
Go, 1,816 lines against 5,804 lines of Dart, with no package of pub.dev and no
code generation of a data class. 5 widget tests run the whole flow in a real
Flutter tree, and 8 Go tests run it with no Flutter. The server of the sample
also stands in Go, and `server/prove.sh` proves that the Go run and the Dart
run give the same bytes for 17 requests.

The port needed 2 rules of the host, and decision 353 holds the 2:

- `ui.Pages` carries the stack of the navigator. A Go application held 1 screen
  before it.
- `GoRuntime.tick` runs the queue of the scheduler again. A goroutine that a
  posted function starts waited for the next gesture of the user, thus a screen
  that reads the server when it opens read nothing.

**Each limit of the port stands outside the compiler.** A package of pub.dev
holds no node type, and a Flutter API that needs a `BuildContext` reaches no Go
value. `flutter/example/compass/README.md` names each 1.

**3 gaps of `testing/synctest` closed.** A wait for a lock is not durable now,
and each timer holds a handle that no other timer takes. Read decision 351.

**The Dart output is 12.4 percent smaller** for a program that reads
`encoding/json`. The library holds 1 variable for the description of each type
with no name of its own. A tree shake of the emitter stays out, and decision
352 holds the measure of it.

**4 defects fell, and each 1 gave a wrong answer with no report:**

| Defect | The report before the fix |
|---|---|
| `Timer.Reset` of a timer of `AfterFunc` | The function never ran again, and the program waited forever |
| `Timer.Stop` and `Timer.Reset` of a Timer that no constructor made | No panic, and Go panics |
| A timer of a bubble took the slot of another timer | A stop cancelled the wrong timer |
| A goroutine that a posted function starts | It waited for the next event of the host |

The documentation sweep of the nineteenth session left 5 stale claims in
`docs/support.md`, and each 1 now agrees with the tool that measures it.
`TestTestsOfGoroot` holds `testing/synctest` now, thus 21 packages of GOROOT
run their real tests under `make ci`.

**A run of each entry of the skip table confirmed 28 causes, and it opened 2.**
`fixedbugs/issue13169` and `fixedbugs/issue78081` each needed more capacity
than the machine gave, and the 2 pass now. The table holds 30 entries.

### 2026-09-04, the nineteenth session

The user asked for each open task, end to end, until no task stays. 1 agent ran
on `testing/synctest` in a git worktree of its own, and the main session held
the compiler core, the runtime, and the docs. **Each item closed.**

**No name of the surface and no package of the standard library stays reachable
inside the non-goals.** The surface reaches 165 packages of 174 and 253 import
paths. 16 packages of the standard library stay outside, and `unsafe` is 1 of
them. `make surface ARGS=-absent` names each 1.

| Set | At the start of the session | Now |
|---|---|---|
| Packages that hold each exported name | 162 of 172 | **165 of 174** |
| Import paths of the surface | 251 | **253** |
| Names of the surface that stay reachable | 29 | **0** |
| Packages of the standard library outside the surface | 17 | **16** |

**5 packages joined:** `encoding/json/v2`, `testing/synctest`, and the real
`encoding/json` with each of its 92 exported names. `crypto/cipher` dropped its
patch, and `encoding/json` dropped a hand-written overlay of 1850 lines.

**5 defects fell, and 2 of them gave a silent wrong answer.**

1. `reflect.Value.Type` and `reflect.Value.Kind` read the type of the value
   inside a location of an interface type. Go reads the type of the location.
   `IsNil` of such a location panicked with the kind of the value.
2. `reflect.Value.Elem` of an interface value read the value as a pointer.
3. The description of a pointer that stage 5 did not write held an empty method
   table, thus a type assertion of `*T` to an interface that `T` satisfies
   reported false.
4. 2 bubbles of `testing/synctest` that ran at the same time starved the second
   one, because the step always read the first bubble.
5. The accessor of a field description cast a function value to the substituted
   type of an instance, and the field holds the type of the declaration.

**5 rules of the emitter and of the runtime landed.**

- A defined type over an instance of a generic type reads the class of the
  instance. Decision 347.
- A function value whose Dart parameter type differs from the parameter type of
  the target takes a wrapper. Decision 348.
- The scheduler holds a bubble with a clock of its own. Decision 346.
- `reflect.StructOf` builds a struct type at run time, in 2 forms. Decision 349.
- A location of an interface type reads the type of the location. Decision 350.

**The measured cost of the real `encoding/json`.** The Dart source of
`testdata/programs/jsonuse` grows from 2,072,121 bytes to 4,548,555 bytes. The
earlier measure put the cost at 14 percent, and the real figure is 120 percent.
The project takes the real package, because a hand-written reimplementation is
a permanent risk of a silent divergence from Go.

**7 witness programs joined:** `geninstconv`, `jsonv2use`, `synctestuse`,
`methodfunc`, `multicastuse`, `reqcancel`, and `structof`.

**The full CI found 2 more defects, and each 1 came from the work above.**

1. The `encoding/gob` overlay read `iv.Type()` of an interface value, because
   the earlier `reflect` reported the type of the value inside the location.
   Decision 350 corrected that rule, thus the encoder reported `type not
   registered for interface`. The function reads `iv.Elem()` now, exactly as the
   real Go source does.
2. `net_web.dart` held no form of the 2 multicast intrinsics, thus
   `dart compile js` reported `Method not found`. A browser opens no socket, and
   the web form gives the cause that Go gives under `GOOS=js`.

Neither defect reached a witness program of the repository. `TestPrograms` found
the first 1 and `TestWebIO` found the second 1, thus 3 runs of `make ci` stood
between the last code change and a green log.

### 2026-09-04, the eighteenth session

The user asked for each of the 15 open items of the parity list, end to end. 2
agents ran on the overlay, and the main session held the compiler core, the
runtime, and the docs. 12 of the 15 items closed, and the 3 that stay each hold
a measured cause.

**The surface reaches 162 packages of 172.** 12 packages joined it:
`crypto/mlkem`, `crypto/mlkem/mlkemtest`, `crypto/mldsa`, `crypto/hpke`,
`testing/cryptotest`, `debug/gosym`, `weak`, `net/http/httptrace`,
`net/http/httputil`, `encoding/json/jsontext`, `image/png`, and
`internal/abi`. 17 packages of the standard library stay outside, and 30 stood
there at the start of the session.

**4 defects fell, and 2 of them gave a silent wrong answer.**

| The defect | What a program saw |
|---|---|
| A method that spells `uint8` held another key than an interface that spells `byte` | The program compiled clean and read a null value inside `io.ReadAll` |
| A local type of a generic function held no outer type argument | 2 instances of 1 generic function gave 1 `reflect.Type` |
| A deferred call of a method with a pointer receiver hoisted the value | The report named no source position, and the build stopped |
| `fmt` called the placeholder of an interface method table with 1 argument | A `NoSuchMethodError` of Dart in place of the panic of Go |

**3 defects of the HTTP client and server fell too.** The server answered no
request that carries the field `Expect`. `awaitContinue` left an expired read
deadline on the connection. And the field `Accept-Encoding` stood inside the
sorted header block, where Go writes it after the header of the caller.

**3 rules of the language landed.**

- A pointer conversion between 2 defined struct types of 1 package. The 2 names
  now hold 1 Dart class, and each 1 keeps a type descriptor of its own. Decision
  335 closes, and `image/png` needed no patch after it.
- A type parameter over a byte slice and a string. The dictionary carries the
  length, the index, the slice, and the 2 conversions.
- A composite literal whose type is a type parameter.

The branch `backlog-43` holds 10 commits of the session.

| Commit | Content |
|---|---|
| `acc0cdc` | The post-quantum stack, and a literal of a type parameter. |
| `0236772` | `testing/cryptotest`, and the parallel check of `Setenv` and `Chdir`. |
| `d4ff99e` | `debug/gosym` and `weak`. |
| `734b23a` | An interface call of a method that reads a type parameter. |
| `6327ee9` | A pointer conversion between 2 defined struct types. |
| `21fb378` | The name of a local type of a generic function, and 2 defects near it. |
| `7813e13` | A type parameter over a byte slice and a string, and 2 fixes. |
| `288286e` | A method that spells `uint8` satisfies an interface that spells `byte`. |
| `9ceea65` | 5 packages join the surface, and 3 defects of the HTTP client fall. |
| `e2991f7` | `crypto/mlkem/mlkemtest`. |

**3 items stay, and each holds a measured cause.** `encoding/json/v2` needs an
adapter for a function value whose Dart parameter type is narrower than the
type of the target. `testing/synctest` needs a timer queue in the scheduler for
the clock of a bubble. And the name of a local type of a generic function
inside a type argument of another instance still holds no outer argument, which
is the last cause of the skip entry `typeparam/nested`.

### 2026-09-04, the seventeenth session

The user asked for the 7 phases of the parity roadmap, end to end. 3 agents ran
in sequence on the overlay, and the main session held the compiler core, the
runtime, and the docs. Read "The road to full parity" of `docs/support.md` for
the plan, and decisions 327 to 335 for the whole account.

**The order of the evaluation is closed.** 5 positions differed from gc, and
each 1 now holds the order of gc. The condition of a `for` loop and its third
clause each move to the top of the body, the search of a `switch` becomes a
nested chain, the initializer of a package variable takes a temporary variable
in `$init`, and a conversion between a string and a slice takes 1 too.

The first form of the hoist over applied. It marked each operation that stood
behind a read, thus `for i := 0; i < len(s); i++` took a temporary variable.
`len` writes nothing, thus the hoist changed no answer of the program. It
changed 155 files of 1 output, and `dart compile wasm` then crashed with
`type 'Call' is not a subtype of type 'Parameter' in type cast`. Only an
operation that can write forces the hoisted form now.

**6 defects fell, and 4 of them gave a silent wrong answer.**

| The defect | What a program saw |
|---|---|
| `any` and `interface{}` held 2 type identities | Each assertion of a decoded JSON object to `map[string]any` reported false |
| A `PathError` held the sentinel of the package | The text read "file does not exist" where Go writes "no such file or directory" |
| The space flag of the verb `x` wrote no separator | `% x` of 2 bytes gave `4d5a`, and Go gives `4d 5a` |
| A shift by a count of an unsigned type | `1 << uint(-1)` panicked, and Go gives 0 |
| A pointer to a type that names itself held 2 Dart forms | The output did not compile |
| A constraint that names its own type parameter | The bound expanded 33 levels, and the output did not compile |

**3 entries of the skip table close.** `fixedbugs/issue4316`,
`fixedbugs/issue72063`, and `typeparam/issue48137`. `fixedbugs/bug369` and
`fixedbugs/issue9355` each pass too, because the comparison reads the exit
status of the 2 runs on its own now.

**2 items hold a measured cause and no date.** A pointer conversion between 2
defined struct types needs 1 Dart class for 2 named types, and the 2 method
sets differ. `typeparam/nested` needs a descriptor that keys on each type
argument of a nested generic declaration.

### 2026-09-04, the sixteenth session

The user asked for each open task, end to end. 3 agents ran on disjoint
directories, and the main session held the compiler core, the runtime, and the
docs. The branch `backlog-43` holds 6 commits of the session, and each 1 stands
behind a green run of `make ci`.

| Commit | Content |
|---|---|
| `ba28b10` | An index and a slice bound of an unsigned type write the unsigned value. |
| `36f8eae` | 13 defects of the skip table, and the wasm build of a bound flag. |
| `fa22342` | The per frame cost of A9 is measured. |
| `2ce75a2` | 166 names join the overlay of 9 packages. |
| `060eef2` | `reflect.FuncOf` builds a function type at run time. |
| `1b17894` | A pointer to an erased type wrote 2 question marks. |
| `51595dc` | An operand list takes the order of the evaluation of gc. |

**The open file limit of the machine.** The user ran out of file descriptors 3
times and restarted the Mac each time. A sampler of `sysctl kern.num_files`
named `go mod tidy` of `gopls` at 5273 open files, 3 seconds after the editor
started, and the report of the user named 10004 for the same process. The
suite writes more than 1100 temporary Go modules, and `gopls` reads each 1
that appears. The repository answered its own half: 3 helpers ran
`exec.Command` with no deadline, and a probe proved that a deadline alone
leaves an orphan, because `go run` starts the program as a child of its own.
Each command takes a process group now, and `spawnSlots` holds 8 slots. A full
run peaked at 8496 open files, against a limit of 491520.

**11 defects behind the 5 silent wrong answers.** 3 of the 5 entries held more
than 1. The hash of a key with no identity gave 0 in silence. A store through
a nil pointer gave an error of Dart. A conversion of a slice to an array held
3. The method table held 3. `reflect.MapIter` held a snapshot of the keys, and
`reflect.Value.Clear` removed each key by a read, thus each NaN entry stayed,
because a NaN never compares equal to itself.

**A lost recover.** `runRecovered` kept the state of the outer chain in 2
variables of the library, and a deferred call that opens a chain of its own
overwrote them. A function that Go returns from normally let the panic escape.
`fixedbugs/issue48898` gave `1` where Go gives `1234`.

**A defect of dart2wasm.** A call that fills an optional parameter of a method
of `GoString` crashes the type flow analysis of the Dart VM. A bisect against
the last green tree proved each half, and the unsigned string index moved to a
top level function.

**166 names of the overlay.** `crypto/x509` went from 115 to 193, `net` from
218 to 259, `net/http` from 341 to 361, and 6 more packages grew.
`docs/support.md` holds a table of each name that stays out, with its cause. 3
names were written, measured, and removed again, because each 1 gave a wrong
answer or cost more than it gave.

**The A9 measure.** A tree of 200 nodes costs 169.8 microseconds for an encode
and a decode of an `rfw` blob, which is 1.02% of the frame budget at 60 hertz.
That is 20 times the 8.66 microseconds of the Go tree of A5.
`flutter/rfwbench` holds the harness.

**The skip table holds 30 entries, and it held 43.** 3 of the 4 that stay hold
a corrected cause, because the old text named the wrong thing.

**The order of the evaluation.** gart read a non-call operand before a call of
the same list, and gc reads it after. The specification leaves that order open,
and real Go source depends on the answer of gc. The port of
`crypto/x509.ParseOID` gave an empty OID until a person split the statement.
Stage 4 holds the rule of gc now, at a cost of 2.8% more lines of Dart over 198
programs. Each operation of a statement must take a variable when any 1 does,
because a partial hoist would break the left to right order of the calls
themselves. 5 positions still differ, and 4 of the 5 hold no place for a Dart
statement in front of them.

### 2026-09-01, the eighth session

The session measured the surface and wrote `docs/support.md`. A throwaway
program loaded each package of the surface 2 times, from the real Go 1.27
source and through the overlay, and it compared the exported names. 52 of
the 74 standard packages hold each name of Go, 21 hold a part, and
`time/tzdata` holds none. The new file names the largest gap of each partial
package, the cause and the cost of each package that stays out, and a focus
list of 9 items. `docs/parity.md` took an at-a-glance table of the
differences, and 6 stale rows of it now read the measured state: `sort`
holds `Sort` with an interface, `sync/atomic` holds the 5 typed values, and
`runtime` holds 12 names.

The session then answered what a Flutter client needs first.
`docs/flutter-first.md` holds the 2 ordered lists, and each claim names a
file and a line. The pass found that the unwind model of 2026-08-29 removed
the hardest Flutter blocker, and that `docs/flutter-options.md` of
2026-08-28 still described the async model. That file now carries the
correction in 3 places. A verified run of 1 struct showed that a
`time.Time` field marshals as `{}` with no error.

| Commit | Work |
|---|---|
| `bcbd099` | The measurement of the surface: `docs/support.md`, the at-a-glance table and the 6 corrections of `docs/parity.md`, and the exact counts of the 2 other files. |
| `4525496` | The first Flutter work: `docs/flutter-first.md`, the 3 corrections of `docs/flutter-options.md`, and the 2 JSON rows of `docs/parity.md`. |

### 2026-09-01, the ninth session

The session ran the 6 tasks of the priority list of the user, and it took the
Flutter decision.

`encoding/json` grew from 7 names to 52. The Dart walk of the package read the
type descriptor only, thus a type with a JSON form of its own gave a silent
wrong result. The walk now reads the method table of that descriptor, and it
calls `MarshalJSON` and `UnmarshalJSON`. A method with a pointer receiver stays
out of the value method set, thus the description of a value type now names the
description of its pointer type, and that pointer description builds a pointer
to a fresh value. The compiler writes the 2 links only when the pointer type
holds a method that the value type does not hold. A byte slice gives base64
text, `time.Time` holds the 8 marshal methods of Go, and a scanner in the Go
half gives `Decoder`, `Encoder`, `Valid`, `Compact`, `Indent`, and each syntax
error with the text and the offset of Go.

The surface grew by 9 packages. `mime`, `mime/quotedprintable`,
`mime/multipart`, `net/textproto`, `net/http/cookiejar`, and
`net/http/internal/ascii` read the real Go source, and a patch of
`mime/multipart` replaces the `go:linkname` stub of `readMIMEHeader`.
`crypto/sha512`, `crypto/hmac`, and `crypto/rand` hold new overlays.

`net/http` took the cookie jar, the form API, and the proxy. An https request
through a proxy needed a new name that Go does not hold: Go writes the CONNECT
request itself and then wraps the connection with `tls.Client`, and the secure
socket of Dart takes over only a socket that no read touched. Thus the runtime
holds the whole exchange, and `crypto/tls.DialProxy` names it.

6 packages now hold each name of Go: `sort`, `strings`, `time`, `flag`,
`errors`, and `runtime/debug`. `fmt` took the whole scan family, and it dropped
the invented name `Writer`, because `io` is importable after all.

| Commit | Work |
|---|---|
| `b00076b` | `json`: the 2 marshal interfaces, base64 bytes, and the 8 marshal methods of `time.Time`. |
| `d7d263e` | `docs/flutter-design.md`: 13 approaches, a matrix of 20 features, and 1 decision. |
| `8c8884b` | `json`: the scanner, `Decoder`, `Encoder`, `RawMessage`, `Number`, and the 6 error types. |
| `a2e1de2` | The 9 new packages of the surface, and the 9 names that they needed. |
| `cdedf0a` | `http`: the cookie jar, the form API, and the proxy with its CONNECT tunnel. |
| `d128d1b` | The small gaps of `sort`, `strings`, `time`, `flag`, `os`, and the scan family of `fmt`. |

### 2026-09-01, the seventh session

The session worked the 4 candidates of the sixth session end to end: the
lazy map iterator, `os/exec`, HTTP/2, and the build of the JS integer. The
range loop over a map now starts in constant time, and `maplinear` joined
the net. `os/exec` rides `Process.start` over 7 interop calls. HTTP/2 rides
the Dart package `http2` behind ALPN, for the client and the server, and
`httptest.Server` takes `EnableHTTP2`. The Go integer of the emitted code is
the type GoInt, the JS form holds the pair of GopherJS, and a program of
64-bit work runs under node byte-equal with `go run`. The go test net now
holds 683 matches of 931 files with 0 differences, and 32 named limits.
Decisions 211 to 214 hold each design.

| Commit | Work |
|---|---|
| `3319bfe` | The lazy map iterator: the entry list of GoMap, the dead links, `GoMapIter`, and `ir.MapIterType`. |
| `0d274bd` | `os/exec` over `Process.start`: the 7 interop calls, `os.Process`, the pipes, and the program `execuse`. |
| `cc36c97` | HTTP/2 over the Dart package `http2`: `$takeSocket`, the 9 interop calls, the transport rules of Go, and the program `h2serve`. |
| `1b8f814` | The 2 forms of the Go integer and their conformance test, behind `dart.library.js`. |
| `52c8772` | The cutover of the emitted code to GoInt: the literals, the boundaries, the uniform interop rule, and `TestJS` under node. |

### 2026-09-01, the sixth session

The session worked the 7 tasks of the open list end to end, in the order that
the user set: package coverage, parity, wasm, the JS design, and TLS last.
The symbolic link operations, the zone database of `time`, the module form of
the source map with the line directives, the complex numbers, the minimal web
target, the JS integer design doc, and the TLS server each landed. The
complex unlock also closed 4 defects of other features. The go test net now
holds 682 matches of 931 files with 0 differences, and 32 named limits.
Decisions 203 to 210 hold each design.

| Commit | Work |
|---|---|
| `7fb47c4` | `os.Symlink`, a true `os.Lstat`, `filepath.EvalSymlinks`, `Localize`, and `HasPrefix`, over 3 interop calls. The `os/exec` survey. |
| `18778b1` | The zone database: `LoadLocation`, the TZif parser and the zip reader of Go, and the embedded database of `time/tzdata`. |
| `5478b43` | The source map holds the module form and each line directive. `Llongfile`, the exit marks of the defer chain, and the deadlock report with the frame chain. |
| `3441575` | The complex numbers over the class GoComplex, with the division of the Go runtime, and 4 defects of the unlock closed. |
| `8bde58c` | The minimal web target: the platform shim, the conditional exports, and the wasm smoke test under node. |
| `63d4987` | The design of the JS integer, in `docs/js-integers.md`. |
| `5a1683a` | The TLS server: `tls.Listen` over `SecureServerSocket` with ALPN, and `httptest.NewTLSServer`. |

### 2026-09-01, the fifth session

The session built the 2 items of the open list: the local struct type that
reads a type parameter, and `recover`. The road also fixed the nil dereference,
the call through a nil function value, the comparison of an uncomparable type,
the delete of an interface key, and the text of the makeslice errors. The go
test net now holds 669 matches of 931 files with 0 differences, and 33 named
limits. Decisions 200 to 202 hold each design.

| Commit | Work |
|---|---|
| `31f77b1` | The shape class with its twins, the 2 pass emit, the recover chain with its tokens, the nil checks, and the 6 patches gone. |

### 2026-08-31, the fourth session

The session closed the 2 items of the open list: the value of a generic
method, and the subdirectories of the go test directory. The harness now runs
931 files, and 592 match `go run` with 0 differences. The road fixed the
generic method value, the dispatch of a method call on a type parameter, the
run time name of an open type, the source order of a keyed literal, the
staged range container, and about 30 more defects. Decisions 160 to 196 hold
each one.

| Commit | Work |
|---|---|
| `b67a9ba` | The dictionaries of a generic method value bind in a closure, and the frame of a top level method takes the joined type parameters. |
| `f289a55` | The typeparam wave: the dictionary dispatch, the interpolated names, the instance keys, 2 verify rules, and 104 files of typeparam. |
| `c168d10` | The fixedbugs wave: the source order of keyed literals, the compound bases, the blank writes, the range container, and the local type identity. |
| `1257b49` | The go test net of the subdirectories in `docs/library.md` and `docs/parity.md`. |
| `aa0906e` | The fourth session in the history. |
| `ab0073a` | The 3 last defects: the reflect Elem type, the union spread, and the field named as a type parameter. |
| the head | The measured counts of the net: 592 matches of 931 files, and 22 named limits. |

### 2026-08-31, the third session

The session closed the last 2 items of the open list: the value flow of stage
3, and the real tests of `math/rand/v2`, of `compress/gzip`, and of
`compress/flate`. It also opened the test directory of the Go repository, and
that run found 18 differences. The road found 6 soundness defects of the
compiler, and each one holds a decision and a program: the body of a range
loop over a function lost its block operation, a make read its length 2
times, a hoisted receiver and a hoisted field write lost their location, a
parallel assignment staged no copy of a struct value, and a type that names
itself stopped the compiler.

| Commit | Work |
|---|---|
| `2870142` | The reflect method API: `NumMethod`, `Method`, `IsNil`, `MapKeys`, `MapIndex`, and the method type table of a descriptor. |
| `f159c30` | The overlay gaps of the math/rand/v2 road: 3 strings functions, `strconv.QuotedPrefix`, `os.Pipe`, and the box rule of a test package. |
| `2deadc8` | The descriptor name of a function type holds no parameter name. |
| `d47e213` | The value flow of stage 3, and the state machine of a range loop body. |
| `1cd202d` | A make with no capacity reads its length 1 time. |
| `f7f9a60` | The hoist keeps the location of a receiver and of a field write. |
| `321a044` | `strings.SplitSeq` and `strings.FieldsSeq`, with the `iter` import. |
| `d3ae3f0` | The go/format subtree from the real Go source, with 4 recover patches and 4 subset rules. |
| `d7c8c5f` | The real tests of `math/rand/v2`. |
| `4c57aad` | A parallel assignment stages a copy of a struct value. |
| `0d2d133` | The real tests of `compress/flate` and of `compress/gzip`. |
| `1d1e352` | The run files of the test directory of the Go repository: 54 match, and 18 skips name their causes. |
| `7b4cfad` | A parallel assignment reads each location before the first write, in the range clause and in a select case too. |
| `8e03c79` | 7 emit fixes: the interface case of a tagless switch, len and cap of an array pointer, new of an array, the name of byte, the late nil channel, the identity of a struct pointer, the map key of an array type, and the walk of a method expression. |
| `35126f4` | The skip list of the run files fell from 18 to 5 entries, and 70 files match. |

### 2026-08-31, the second session

The session closed item 1 of the open list, and it found 2 more defects of the
same rule.

| Commit | Work |
|---|---|
| `96d2b8e` | 1 matcher for `-run` and for `-bench`, with the rule of `testing/match.go` of Go. |
| `b1051dd` | The record of the filter work. |
| `6fc8577` | A call through a method value gives up control. Stage 5 writes a method value of an interface value. |
| `2bed5ca` | The real tests of `internal/chacha8rand` and of `compress/zlib`. |
| `22d1f7d` | The record of the library work. |
| `1cec023` | The strict rune decoder, and the real tests of `unicode/utf8`. |

The defects of that session:

| Defect | Cause |
|---|---|
| `Run` of `T` ran a subtest that the filter excludes | The method never read the filter |
| `-run 'A/x\|B'` never ran `B` | The split cut the pattern at each slash before it read a vertical bar |
| `-run '^Fuzz$'` printed no skip line | `internal/driver/run.go` matched the name with `strings.Contains` |
| `Run` of `B` ran a subbenchmark that `-bench` excludes | The method never read the filter |
| A filter that keeps no test wrote no warning | The harness counted no run, and the driver wrote no mark |
| A call through `d.step` of `compress/flate` never gave up control | Stage 3 dropped each method that reaches a function value |
| A method value of an interface value gave no Dart code that compiles | Stage 5 wrote the name of a method function of the interface type |
| A test that reads `testdata/x` found no file | The binary ran in the output directory, and not in the directory of the package |
| A range loop over `"\xE0\x9F\xBF"` gave U+07FF | The rune decoder of the runtime held no smallest value for each size |
| `FullRune("\xc0")` gave false | `need` of the `unicode/utf8` overlay gave the size 2 for a start byte that Go rejects |

### 2026-08-31, the first session

The session closed each of the 8 items of the open list. It closed 2 of the 3
items that the work found, and it found 6 defects.

| Commit | Work |
|---|---|
| `f551872` | A generic function as a value. The Dart form is a closure that gives the dictionary of each type argument. |
| `5202398` | Keep alive of `net/http`. The client holds a pool of 2 connections for each host. |
| `c8a8153` | `compress/flate` and `compress/gzip`, each native. The client adds `Accept-Encoding: gzip`. |
| `00f25c6` | `net/http/httptest`, with `ResponseRecorder`, `NewRequest`, and the `Server` of a test. |
| `61111d2` | `iter.Pull` and `iter.Pull2`, over a goroutine and 2 channels. |
| `3099b9e` | `reflect.Type.Size`, `Type.Align`, and `StructField.Offset` of a struct. |
| `dbb0c69` | A byte that holds no rune of UTF-8 keeps its value through each write. |
| `69025b8` | `runtime.GC` runs each cleanup. The real tests of `io` each pass. |
| `b093738` | The text of a type descriptor follows `reflect`, and not `go/types`. |
| `943e524` | The block analysis in the cache key of a package. |
| `e7b77ef` | `compress/zlib`, and the audit of the 3 docs. |
| `7a7eb93` | Each stale claim of the parity doc. |

The 6 defects of that session:

| Defect | Cause |
|---|---|
| A conversion of a slice to a pointer to an array had no Dart form | The array now shares the elements of the slice through `$ArrayView` |
| A second package reached no unexported type name | The Dart name held a `_` prefix |
| `$arrayIndex` gave `num` | The call held no type argument, and Dart reads the context |
| A local named `resume` collided with the frame class | `resume` is now a reserved name |
| `Type.Name` gave the name of the element type of a composite type | The scan of the text stopped at no marker |
| The cache served a package of the block form before | The source hash held no digest of the analysis |

### 2026-08-30, the third session

The session closed each item of the open list, and it found 18 defects. Each
defect now has a program of `testdata/programs` or a test of the Go repository.

| Commit | Work |
|---|---|
| `137b6ae` | The source map of the output, with 4 readers. |
| `6fd53b4` | `crypto/md5`, `crypto/sha1`, and `crypto/sha256`, each an overlay of the algorithm of Go. |
| `4053945` | A wider `reflect`, and `encoding/binary` with its path of `reflect`. |
| `269072f` | A pointer to a field of a struct and to an element of a slice or an array. |
| `2dc5ed2` | `regexp` and `regexp/syntax`. `-run` of the test command now reads a regular expression. |
| `d285936` | `crypto/tls`, with a client. |
| `c0477a7` | A method with a nil pointer receiver on a generic type. |
| `03b4fd5` | The real Go tests of 12 packages, and 5 defects. |
| `c731bb6` | The io tests, and a select statement that meets another select statement. |
| `ad7409f` | `net/http`, with a client and a server. |
| `3a5c257` | `testing/quick`, and `reflect.Value.Call`. |
| `ea91db9` | The audit of the 3 docs. |

### 2026-08-30, the second session

The session found 9 defects.

| Commit | Work |
|---|---|
| `60d5c6a` | A type parameter of integer types takes the Dart type `int`. `Dict` holds a mask of the width. |
| `b3ed42e` | M19. The accessors of `time.Time`, `Parse`, and a zone of a fixed offset. |
| `48d1e4c` | A conversion of each result of a call that gives many values. |
| `a036bcf` | 10 more packages of the standard library. |
| `8cc5775` | The `log` package and the `path/filepath` package. |
| `8dba0a0` | A function literal of a package variable that can block. |
| `cd1ff5e` | `time.ParseDuration`, `Truncate`, and `Round`. |
| `6e0e6d5` | `io/fs`, and a walk of a directory tree. |
| `2ecce47` | M21. A method with a nil pointer receiver, and `net/url`. |
| `e9075ab` | The what-compiles doc. |

The session also gave stage 2 a rejection for an operation of a type parameter
that holds no single Dart form. `math/rand/v2` `N` and `Rand.N` reached the
surface. M20 closed with 15 more packages.

### 2026-08-30, the first session

The session found 5 defects.

| Commit | Work |
|---|---|
| `6348204` | M18. The calendar of `time`: `Date`, `Format`, `Location`, `UTC`, and `Local`. |
| `9141fc4` | `reflect.ValueOf`, `reflect.Zero`, `Type.Elem`, `Value.Elem`, `Value.Set`, and `Value.Interface`. |
| `10a2669` | `t.Setenv` and `t.Context`, with `os.Setenv` and `os.Unsetenv`. |
| `9d037b0` | The network: UDP, a Unix socket, a read deadline, a write deadline, and `Dialer.DialContext`. |
| `6ef1ee8` | The measurement of the liveness of a frame. The pass stays out. |
| `c257ca7` | `math/rand` and `math/rand/v2`, each with the exact algorithm of Go. |
| `3a95895` | The survey of `regexp` and of `runtime.Callers`, `os.File.WriteAt`, and the open mode of `O_RDWR`. |

### 2026-08-29

The session found 6 defects of the compiler.

| Commit | Work |
|---|---|
| `15ca814` | The loop variable of Go 1.22. Each pass of a loop holds its own copy. |
| `3213c06` | M15. Each unnamed struct type takes a file of its own, thus 2 packages share 1 Dart class. |
| `b15b683` | M16. The `context` package, with a calendar and `internal/reflectlite`. |
| `6642ee0` | A deferred call that can block. `defer cancel()` stays inside the subset. |
| `4df4ccf` | The source position of a test log line. |
| `f5ce941` | A descriptor for an instance of a generic named type. |
| `bb4d5b4` | A wider surface of the overlays. Read the note below. |
| `ca9782c` | `encoding/base64`, `strings.Map`, the special case functions of `strings`, and `testing.Benchmark`. |
| `f168d8c` | The `flag` package, with the rules of the parse and the format of the report. |
| `7bfa04f` | `reflect.DeepEqual`, `reflect.TypeOf`, and the kind of a type. |
| `0098074` | `errors.As`, `errors.AsType`, `errors.Join`, and the verb `%w` of `fmt.Errorf`. |

The wider surface of `bb4d5b4`: `fmt.Stringer`, the timer channels of `time`,
and each type and each function form of `sync/atomic`. It also gave
`os.CreateTemp`, `os.MkdirTemp`, `File.ReadAt`, `strings.Reader.ReadAt`,
`t.TempDir`, `runtime.GOOS`, and `runtime.GOARCH`.

## The defects that the library found

The real standard library source and the parity tests found 64 defects. Each one
also breaks plain user code, and each one now has a test program.

| Defect | Cause | Test |
|---|---|---|
| A wide `for` clause gave "the statement *ir.Block does not fit a for clause" | `i, j = i+1, j-1` needs a block | `control` |
| A pointer field compared the fields behind the pointer | Go compares the address | `structs` |
| A cyclic pointer made a comparison run forever | The same cause | `structs` |
| A package variable took its value in declaration order | Go uses the dependency order | `initorder` |
| Stage 2 accepted an index of a mixed type parameter | Stage 5 wrote broken Dart | `internal/verify` |
| A variadic generic call wrote the name of a type parameter | The emitter read the signature of the declaration | `generics` |
| The cache key held only a version string | A new build of the compiler kept each archive | `internal/driver` |
| The normalizer crashed on a function with no body | The nil check ran after the walk | `math` (assembly) |
| A package name that is a file name of its parent collided | `math` holds `bits.go` | `math`, `math/bits` |
| A mixed short declaration never declared its new names | `frac, e := f(frac)` | `mixdefine` |
| A range loop over a slice type parameter was rejected | The core type was not read | `coretype` |
| A type parameter counted as an interface type | Its underlying type is its constraint | `slicesuse` |
| A nil comparison through a type parameter gave false | The check read the constraint | `mapsuse` |
| The overlay of a package paired its files by index | The name of the Dart part file then held the wrong name | `slices` |
| A top level name collided with a method name of the package | A Dart class member hides a top level name | `os` |
| The inner emitter of a closure shared no cache of the package | A nil map, and a nil block analysis | `slices` |
| A call of a call with multiple results passed the Dart record | Go passes each result as its own argument | `spread` |
| An embedded interface field wrote a call of a top level function | Such a field holds an interface pair | `io` |
| A blank package variable of 2 files collided | Such a variable holds a static check | `io` |
| A blank import of `unsafe` was rejected | Such an import performs no operation | `bytes` |
| A type declaration of a struct inside a function was rejected | Such a type takes the class of its shape | `pathuse` |
| The address of a named result wrote no box | The declaration read the raw name | `boxptr` |
| A type switch over `any(v)` of a type parameter took no case | The dictionary held no type descriptor | `generics` |
| A method value of a named type that is not a struct was rejected | Such a method is a top level function | `bytesuse` |
| A labelled continue of a flattened loop lost its label | The flatten pass read no label of the loop | `bytesuse` |
| The comma form of a channel read held a doubled pair type | `types.Info` already gives a pair for that context | `chans` |
| A deferred call that can block lost the block | The deferred calls run outside the machine | `internal/driver` |
| A comparison of a value with an interface gave false | Only an untyped operand took the type of the other side | `ifaces` |
| The entry point of a nested package landed in a doubled directory | The base name of the file held the whole path | `internal/driver` |
| An overlay missed a package that the load holds only as a test variant | The map of the first load read only a package whose identifier is its path | `internal/load` |
| The report held each type error of a package outside the surface | The load visited the whole graph, thus the real rejection stayed out of sight | `internal/driver` |
| The method table of an embedded pointer to a struct called a top level function | A pointer to a struct is the Dart object reference, thus the method is a class member | `bufiouse` |
| The overlay of a package that the first load missed left the real files of it | The first load reads the imports of the module, and an overlay holds imports of its own | `netpkg` |
| A closure of a loop read the value of the last pass | Go 1.22 gives each pass its own variable | `loopvar` |
| A type assertion of an alias read the name of the alias | The name must come from the type behind the alias | `anonstructs` |
| The address of a package variable wrote the raw name of it | Such a variable holds a box, and the name needs its prefix | `ctxuse` |
| A comparison of a channel with nil read the null form | A channel holds its zero value as an object | `ctxuse` |
| A value of an instance of a generic type held an empty method table | Each instance needs its own description | `genericiface` |
| A conversion between 2 pointer types of 1 underlying type wrote an error | The Dart form of the 2 is equal | `flaguse` |
| The method table of a pointer to a named type that is not a struct cast the base type | Such a value holds a box | `flaguse` |
| A type assertion to a type parameter compared the name of the parameter | The name comes from the descriptor of the dictionary | `errorsuse` |
| A type switch of 2 interfaces of 1 method name took the first case | Go compares the name and the signature | `errorsuse` |
| The emitter named a descriptor variable of a pointer to an interface type | The package holds a variable for a named struct type only | `reflectuse` |
| A listener that accepted nothing kept the isolate alive | The iterator of a stream subscribes at the first accept, thus the cancel closed no server socket | `netwide` |
| The end of package main left each open handle | Go ends the process at that point | `netwide` |
| `%v` of a float32 wrote the shortest text of a 64-bit value | Go writes the shortest text of the width of the value | `randuse` |
| `os.OpenFile` with `O_RDWR` replaced the content of the file | Only `O_TRUNC` replaces it in Go | `stdwide` |
| The flag `0` of `fmt` wrote a space in place of a zero | The pad wrote only a space | `fmtuse` |
| The flag `+` of `fmt` wrote no sign for a value that is not negative | The Dart formatter read the flag for `%+v` of a struct only | `fmtuse` |
| A return of 1 call of many results wrote no conversion | A result of an interface type needs a box | `multiconv` |
| A multiple assignment of 1 call wrote no conversion | The temporary took the type of the location, and not of the call | `multiconv` |
| A function literal of a package variable crashed the compiler | The maps of the local names live only in the walk of a function declaration | `stdmore` |
| A method that writes its own receiver wrote broken Dart | The receiver took the keyword `final` | `stdmore` |
| A closure that can block inside a generic function named a type parameter of no scope | The frame class stands at package level, and it took no type parameter | `stdmore` |
| A function literal of a package variable that can block reached no
declaration | Only a statement drained the list that the builder holds |
`cliuse` |
| A method key wrote the name of an alias | 2 packages hold 2 alias names of 1 type, thus the 2 keys differed | `filepathuse` |
| A method call on a nil pointer receiver threw | Dart calls no member of null, and Go runs such a method | `nilrecv` |
| A conversion of a slice to a pointer to an array had no Dart form | The array shares the elements of the slice, thus a plain copy holds the wrong meaning | `gzipuse` |
| A second package reached no unexported type name | The Dart name held a `_` prefix, and a Dart library reaches no such name of another library | `gzipuse` |
| `$arrayIndex` gave `num` | The call held no type argument, and Dart reads the context of a call for one | `gzipuse` |
| A local variable named `resume` collided with the frame class | The frame of a function that can block holds a member of that name | `pulluse` |
| `Type.Name` gave the name of the element type of a composite type | The scan of the text stopped at no marker of a composite type | `sizeuse`, `typetext` |
| The cache served a package of the block form before | The source hash held no digest of the block analysis | `internal/driver` |

## The test suite

`make ci` runs each of them.

- The end to end test in `internal/driver`. It compiles each Go program of
  `testdata/programs`, runs the Dart output, and compares the output with the
  output of `go run`.
- The end to end test of `gart test`. It runs the tests of each module of
  `testdata/tests` with `go test` and with `gart test`. It runs the plain mode
  and the verbose mode, and it compares the 2 outputs.
- The end to end test of `gart run`. It reads the output of the program and the
  exit code.
- The cache test and the determinism test of a test build.
- A benchmark comparison of the async model, in `testdata/tests/async`.
- A test for each verifier rule. Each rule has a test that rejects and a test
  that accepts.
- Unit tests for the runtime library in Dart.
- Golden tests for the emitted Dart code, in `testdata/golden`.
- The cache test, the determinism test, and the AOT test.
- CI runs each of them through `make ci`.

The workstream doc holds the tests that stay open.

## Project name candidates

The project takes `godart` and the command takes `gart`. The user revisits the
decision later. The shortlist stays here for that review.

| Name | Reason |
|---|---|
| `gart` | Go plus Dart. 4 letters, easy to type, no known collision. |
| `sisal` | A dartboard is made of sisal fibre. The board receives the dart. |
| `ninedart` | A 9-dart leg is the perfect leg. The project value is exactness. |
| `tungsten` | A dart is made of tungsten. The name suggests density and precision. |
| `oche` | The throw line in darts. Short and distinctive, but hard to spell. |
| `flight` | The fin of a dart. Go code takes flight as Dart. |
| `dargo` | Dart plus Go. Easy to say. |

## Facts, by date

Each fact carries the date of its check.

### 2026-08-27

- `go version go1.27.0 darwin/arm64`. GOROOT is `/usr/local/go`. GOEXPERIMENT is
  empty. GOTOOLCHAIN is `auto`.
- `Dart SDK version: 3.13.0 (stable)` on `macos_arm64`. Dart 3.13 has records,
  patterns, class modifiers, and extension types.
- The spec is at `/usr/local/go/doc/go_spec.html`. The subtitle is "Language
  version go1.27 (May 26, 2026)".
- The Go 1.27 change list is in the spec at `go_spec.html:8842`, section
  `Go_1.27`.
- The struct literal key rule is at `go_spec.html:3247`. A key must be a valid
  field selector for a field, and the field may be promoted. An embedded field
  on the path must not have a pointer type.
- The assignability rule for a generic function is at `go_spec.html:1994`.
- `/usr/local/go/doc/godebug.md:159` states that Go 1.27 removed the
  `gotypesalias` setting.
- `go doc builtin.new` states that `new` accepts a type or an expression.

### 2026-08-28

- The repository is `/path/to/godart`. The Go module is
  `github.com/eddyw/godart`. The branch is `main`.
- The layout: `cmd/gart`, `internal/{analyze,driver,emit,ir,load,meta,name,
  normalize,overlay,verify}`, `runtime/goruntime`, `testdata/programs`,
  `testdata/golden`, and `docs/parity.md`.
- `golang.org/x/tools` v0.49.0 loads a Go 1.27 module with full type
  information. `github.com/fsnotify/fsnotify` v1.10.1 drives watch mode.
- `make ci` runs gofmt, `go vet`, `go test`, `dart analyze --fatal-infos`, and
  `dart test`. All 5 steps pass.
- The end to end test compiles 18 Go programs of `testdata/programs`, runs the
  Dart output, and compares it with the output of `go run`.
- `TestCache` proves the early cutoff. `TestDeterministic` proves byte equal
  output. `TestGolden` compares `testdata/golden`. `TestAOT` runs `dart compile
  exe`.
- `Uint8List.asUnmodifiableView()` gives the byte storage of `GoString`. A write
  to the view throws an `UnsupportedError`.
- `packages.Config.Overlay` replaces a file of the standard library, and it also
  adds a new directory inside GOROOT. A test proved both forms.
- Go 1.27 rejects a type parameter on an interface method with the message
  `interface method must have no type parameters`.
- `gart build -o <dir> -runtime <path> ./...` writes the Dart package. `gart
  watch` holds the same flags.

### 2026-08-28, the test command

- `packages.Config.Tests` gives 4 packages for a package with 2 test files. The
  identifiers are `p`, `p [p.test]`, `p_test [p.test]`, and `p.test`. The
  package `p.test` has the name `main` and 1 file in the build cache.
- The overlay replaces the content of that file, and the type checker reads the
  new content. The import of the package under test resolves to `p [p.test]`.
- The import LIST of `p.test` stays the list of the go command: `os`, `reflect`,
  `testing`, `testing/internal/testdeps`, and the 2 test variants. An import
  outside that list fails with `no metadata for <path>`.
- `packages.Package.ForTest` stays empty with this mode. The identifier gives
  the variant, thus the code reads the identifier.
- `go test -v` writes a log line with 4 spaces of indent, at each depth. A
  result line takes 4 spaces for each level. A `=== RUN` line takes no indent.
- `go test` with no `-v` writes a block only for a test that failed. The block
  holds the result line, then the log lines, then the blocks of the subtests.
- The test binary writes `PASS` or `FAIL` on its last line. The go command
  writes `ok` or `FAIL` with the import path and the time. In the plain mode it
  hides the output of a binary that passes.
- A Dart parameter of the type `FutureOr<void> Function()` accepts a synchronous
  closure and an async closure. A `void Function()` parameter accepts only the
  synchronous form.
- `gart test -o <dir> -runtime <path> [-v] [-run <text>] ./...` runs the tests.
  `gart run -o <dir> -runtime <path> ./... -- <args>` passes the arguments to
  the program.
- The 5 modules of `testdata/tests`. `basic` holds a subtest, a cleanup, a skip,
  and an external test package. `mixed` holds a channel, a failure, a fatal
  error, and a benchmark. `root` holds a test in the root package. `unsupported`
  holds a fuzz target and an example. `async` holds the 2 chains of the async
  measurement.

### 2026-08-29

- Dart copies an unmodifiable `Uint8List` across an isolate boundary. A child
  that holds the list does not observe a later write of the parent. 256 MB costs
  13 ms through `Isolate.run` and 23 ms through `SendPort.send`, thus the cost
  stays linear in the size. `TransferableTypedData.fromList` copies the bytes
  into its buffer first, thus it gives no escape. Worker rule 6 of the section
  "Isolate workers" is wrong.
- An extension type gives no cheaper `uint64`. An ahead of time build measures
  1.4 ns/op for a division, a shift, and a comparison through a top level
  function. It measures 1.9 ns/op through an extension type. Such a type erases
  to a plain `int`, thus it gives compile time safety only.
- The compiler needs no second GOROOT. `packages.Config.Overlay` is a map in
  memory, `x/tools` writes it into a JSON file of a temp directory, and it
  passes `-overlay=` to the go command. `/usr/local/go/src/syscall/dart` does
  not exist on disk, and `internal/load` writes no file.
- A Dart for loop gives the loop variable a new binding at each pass when the
  declaration stands in its header. `for (int i = 0; ...)` gives `[0, 1, 2]`,
  and a declaration in front of the loop gives `[3, 3, 3]`.
- A Dart URI is a string literal, thus a dollar sign in the path starts an
  interpolation. The compiler reports "Can't use string interpolation in a URI".
- A Dart record pattern takes no boxed variable. `(a.v, b) = f();` gives "Not a
  constant expression" and "Refutable patterns can't be used in an irrefutable
  context".

### 2026-08-28, the library work

- `FieldDesc` of the type descriptor holds the name, the struct tag, a read, and
  a write of each field. `encoding/json` reads it, thus the plan needs no code
  generator. The descriptor now also holds a `zero` factory.
- A chain of 20 async frames costs 2400 nanoseconds with the `async` form of
  Dart. It costs 107 nanoseconds with a `FutureOr` form that tests `is Future`
  and calls `.then`. The second form passes each continuation by hand.
- `dart:io` gives a synchronous form of each file operation: `File.openSync`,
  `readIntoSync`, `writeFromSync`, `setPositionSync`, `readAsBytesSync`, and
  `FileStat.statSync`.
- `dart:io` gives no handle of a directory, thus `os.Open` of a directory fails.
  Go gives a file for such a call.
- The go command holds no `_noasm.go` file for `hypot` and for `log` on arm64.
  `hypot_noasm.go` and `log_stub.go` cover the 2.
- `jsonEncode` of Dart writes `0.0` for a double zero. Go writes `0`, thus the
  encoder converts a whole double to an integer.
- `bytes` needs `recover` in `growSlice` of `buffer.go` only. Each other use of
  `unsafe` in the package is a blank import for a `go:linkname` directive.

- `packages.Load` with no overlay gives the real standard library source. A
  survey of the rejections of stage 2 over such a load finds the cause that
  blocks a package. The survey tool lived in `tmpsurvey/`, and I removed it.
- `types.Info.InitOrder` gives the order of each package variable that holds an
  initializer. A variable with no initializer is not in the list.
- `Stopwatch` of Dart ticks each nanosecond on this machine.
  `Stopwatch.frequency` gives 1000000000, and `DateTime` holds only
  microseconds.
- `ByteData(8)` holds a view of each type through `setFloat64` and `getInt64`,
  thus 1 buffer reinterprets the bytes of a value.
- `unicode/utf8` of Go declares `func word[T string | []byte](s T) uintptr`. The
  2 terms give `GoString` and `Slice<int>`, and no Dart operator fits both.
- `math` holds 6 files named `<name>_asm.go`. The go command selects a file only
  for the architecture of its build tag. On arm64 the set holds 4 of the 6.
- `slices` of Go takes the address of a local integer in its sort. The overlay
  uses a merge sort with 1 extra buffer.
- `maps` of Go imports `iter`. Each function of it that gives an iterator stays
  out of the overlay: `All`, `Keys`, `Values`, `Insert`, and `Collect`.
- The left side of `:=` in Go accepts only a name. A field and an index never
  appear there.
- A generic Go function used as a function value does not fit a plain Dart
  function type. The Dart form takes a dictionary for each type parameter, thus
  `slices.SortFunc(x, cmp.Compare[E])` needs a closure.

### 2026-08-30

- `make ci` passes on `main` at commit `3a95895`. The end to end test compiles
  59 Go programs and runs the tests of 8 modules.
- `internal/driver/test_test.go` keeps the source position of a log line, and it
  cuts a position that names a file outside the test files. The `compare-tests`
  target of the Makefile normalized the go side only, thus it reported a false
  difference after M17.
- `internal/load` asserts the test list of `testdata/tests/basic/calc`. A new
  test file of that module breaks 2 tests, thus a new module holds new tests.
- Dart holds no file mode that reads and writes and stands at the start of the
  file. `FileMode.append` keeps the content and stands at the end, and
  `setPositionSync(0)` then moves the position back. A write lands at the
  position, and not at the end.
- The iterator of a Dart stream subscribes at the first `moveNext`. A listener
  that accepted nothing holds no subscription, thus a cancel closes no server
  socket and the isolate stays alive.
- `Future.timeout` gives a new future, and the source future stays open. A later
  completion of the source reaches no unhandled error, because the timeout drops
  it.
- A chain of 20 frames that really unwinds costs 138 ns/op in Go and 540 ns/op
  in the Dart output.
- A frame that holds no local at all costs 532 ns/op, and the baseline costs 538
  to 558 ns/op. A frame that holds the live locals of its state costs 562 ns/op.
  The branch on the state costs more than the 2 field writes.
- `math/rand/v2` needs `internal/chacha8rand` only. Each other file of it stays
  inside the subset, except the go:linkname of `runtime.rand` in `rand.go`.
- The type set of `intType` of `math/rand/v2` holds 11 integer terms, thus
  `typeset.Core` gives no core type and the Dart type stays a type parameter.
  Each term maps to the Dart type `int`, thus a conversion is a Dart identity
  plus a mask of the target width.
- Stage 2 rejects an index, a slice expression, a range loop, `len`, and `cap`
  on a type parameter with more than 1 Dart form. It accepts arithmetic and a
  conversion, and stage 5 then fails with no source position.
- `regexp/syntax/parse.go` holds 2236 lines. The recover at line 893 holds 2
  causes only: `ErrLarge` and `ErrNestingDepth`.

### 2026-08-30, the second session

- The underlying type of a `*types.TypeParam` is its constraint, thus
  `types.IsInterface` gives true for each type parameter. A test of a target of
  a conversion must name the parameter first.
- Go rejects a range loop over a type parameter with no core type. Thus the
  rejection of stage 2 never fires for an integer type set. That rejection
  covers an index, a slice expression, a range loop, `len`, and `cap`.
- A Dart bound of `int` is legal. `<Int extends int>` compiles, and the type
  parameter then holds no value of the body.
- A generic function instantiation is a constant expression of Dart, thus
  `$identity<bool>` stands inside a `const Dict<bool>(...)`.
- A layout of `time.Parse` that names no year gives the year 0. That year stands
  outside the range of the overlay, thus the value wraps.
- The `+` flag of Go writes a sign for a numeric verb only. `%+v` of 42 gives
  `42`, and `%+d` gives `+42`.
- `container/heap`, `container/ring`, `encoding/ascii85`, `encoding/csv`,
  `encoding/hex`, `hash`, `hash/adler32`, `hash/fnv`, and `net/url` each compile
  from the real Go source with no patch.
- `hash/crc32` reads `internal/cpu` in the file of each architecture. A patch of
  each of the 5 files gives the portable path.
- `net/netip` needs only `strconv.AppendUint` and 1 patch of `uint128.go`.
- `net/url` needs `internal/godebug` and `net/netip`, and it holds 1 function
  with no body for a go:linkname directive.
- `path/filepath` of Go reads `os.Lstat`, `os.IsPathSeparator`,
  `os.PathSeparator`, and `File.Readdirnames`. The overlay reads `path`.
- `log` of Go reads `runtime.Caller`, `runtime.CallersFrames`, and the `Append`
  functions of `fmt`.
- `text/tabwriter` holds 1 call of `recover` and the address of a field of a
  slice.
- `os` can import `time`. The comment of the overlay named a cycle that does not
  exist.

### 2026-08-31, the first session

- `make ci` passes on `main` at commit `7a7eb93`. The end to end test compiles
  89 Go programs and runs the tests of 8 modules.
- The Dart VM runs a `Finalizer` callback after about 20000 small allocations.
  The callback needs 1 turn of the event loop, thus a call that gives up control
  reads the result. Dart holds no call that forces a collection.
- `http.DetectContentType(nil)` gives `text/plain; charset=utf-8`. The recorder
  of `httptest` names that type for an empty body too.
- Go itself races on a Close of the body of an answer that stands before the end
  of the body. 5 runs of 1 program gave 2 counts of the connections.
  `testdata/programs/httpkeep` holds no such case for that reason.
- `types.TypeString` and `reflect.Type.String` differ in 1 place. An unnamed
  struct type and an unnamed interface type each hold 1 space after the keyword
  in `reflect`.
- `compress/flate` writes the same bytes as Go for each level, for a dictionary,
  and for the empty text. `testdata/programs/gzipuse` compares the count of the
  bytes.
- A module path with no dot reads as a path of the standard library. `module
  cachetest` gives "the package cachetest/lib is outside the supported surface".
  A test module needs a path such as `example.com/cachetest`.
- The block analysis of 1 package reads the whole module. A new function of a
  package that this package does not import can change the output.
  `TestCacheBlockDigest` proves the rule.
- The output of `testdata/programs/iteruse` grows 47 percent with `iter.Pull`,
  and the output of `testdata/programs/slicesuse` grows 4 percent. A program
  that imports neither `iter` nor `slices` keeps its size.

### 2026-08-31, the second session

- A file of `internal/overlay/std` with no build tag reaches the go command and
  a test binary. The go command builds the directory as a package of the module,
  because the build tag `dart` excludes each other file of it. `//go:embed std`
  reads the same file, thus the overlay gives it to the compiler.
- A `_test.go` file inside `internal/overlay/std` would join the overlay, thus a
  test of such a file lives outside that tree.
- `go test` writes ` [no tests to run]` after the line of a package when the
  output of the binary holds `testing: warning: no tests to run`. The go command
  reads the line and writes the mark.
- `-bench .` needs no special case. The expression `.` matches each name that
  holds 1 character or more.
- `internal/chacha8rand` runs each of its real tests, and it needed no work.
- `compress/zlib` runs each of its real tests. It needed `internal/testenv`, the
  working directory of the package, and the fix of decision 126.
- The tests of `bytes`, of `compress/flate`, and of `compress/zlib` read 2
  functions of `internal/testenv`: `Builder` and `SkipIfOptimizationOff`.
- `strings.FieldsSeq` is the only gap of the tests of `compress/flate`. The
  overlay of `strings` must import `iter` for it. The output of
  `testdata/programs/args` then grows by 17479 bytes, which is 4 percent, and no
  other package changes its form.
- `net/url` cannot run its tests. It needs `encoding/gob`, `internal/diff`, and
  1 call of `recover`.
- `math/rand` cannot run its tests. `default_test.go` reads `testenv.Command`,
  and such a call starts a process of the go command.
- A survey of each overlay package found 1 more package that runs its real
  tests: `unicode/utf8`. Each other overlay holds a subset of the real API, thus
  a test file names a function that the overlay does not hold.
  `docs/library.md` holds the whole survey.
- `slices` needs `internal/asan`, `internal/msan`, `internal/race`, `unsafe`,
  and 3 calls of `recover`. `testenv.OptimizationOff` alone opens no path.
- The type checker reports the errors of 1 file group at a time. A first run can
  hide a later cause, thus a survey needs a second run after each fix.

### 2026-09-01, the ninth session

- `blockingIntrinsic` of `internal/analyze/analyze.go` holds the name of each
  interop function that gives up control. A new intrinsic that calls `$await`
  and that no entry names gives a silent wrong result: the caller reads the zero
  value, and no report reaches the program. `DialTLSProxy` showed the failure.
- `SecureSocket.secure` of Dart takes over the subscription of a socket that a
  listen already holds. It rejects a subscription that stands paused, thus the
  handler must stay live between the read of the answer and the call.
- A `StreamIterator` subscribes at its first `moveNext`, thus a `_Conn` that no
  read touched holds no subscription. A `_Conn` that a read touched holds one
  that no code can name, thus `tls.Client` over such a connection has no form.
- The loader rejects a package overlay that holds more files than the real
  package. `flag` holds 1 real file, thus each new name of it joins `flag.go`.
- `fmt` can import `io`. The comment of the overlay claimed a cycle, and no
  cycle exists. `strings` still cannot, thus its `Reader` holds its own EOF.
- An import of `os` in `fmt` breaks the web target. `os` reads `time`, `time`
  reads the zone file, and `file_web.dart` holds no `$readFile`. The Dart
  compile of the web target then fails for each program that reads `fmt`.
- A parameter of a runtime function that takes a byte slice must name
  `Slice<GoInt>` in a web file. `GoInt` is an extension type over `int` on the
  native target, thus `Slice<int>` works there and fails on the JS target.
- A measure that walks the method set of a pointer counts more names than a
  measure that walks the value method set. The count of `time` grew from 145
  to 224 with no change of the real package.


### 2026-09-02, the tenth session

- A Go module outside the main module wrote no Dart. Stage 2 accepted the
  import, because `overlay.Outside` reads a dot in the first path element, and
  `compiled` of the driver read the main module and the surface only. Stage 5
  then failed with "the package X has no import prefix". The 2 rules must name
  the same set.
- A library package of Go writes `lib/<name>.dart` and no `bin/main.dart`, thus
  a Flutter application needs no change of the entry point.
- A top level Go function takes the suffix `$f` when a struct of the same
  package holds a method of that name. `topName` of `internal/emit/emit.go:375`
  writes it, and a host that names the function by hand must read the output.
- The scheduler gave a host no signal after a goroutine woke. A poll of 16
  milliseconds also stops `pumpAndSettle` of a widget test from ever settling,
  thus a hook is the only shape that works. `$onIdle` runs after the run queue
  empties.
- The box of an interface value copied a composite literal. The literal builds
  a value that no other expression names, thus the copy is dead. The counter
  application went from 39 copies to 2, and each program of `testdata` gives
  the same output as before.
- A `GoString` can never be const. It holds a `final Uint8List bytes`, and Dart
  holds no const `Uint8List`. A const widget from a Go tree has no road.
- A copy of a struct that holds a reactive value splits its subscriber list,
  because `$copy` builds a new `dep` that holds the slice of the moment. The
  copylocks check of `go vet` reports each copy through a `noCopy` field, and
  the report follows the value through an enclosing struct.
- `flutter analyze` walks a path dependency, thus the generated code of a Go
  module reaches the report of the user. 10 more lint names in the header of a
  generated file took 1034 issues to 0.
- The Flutter SDK holds no `Command` class and no `Signal` class. The
  architecture guide of the Flutter team writes the Command pattern in
  application code, and it recommends the third-party `package:provider` for
  dependency injection.

### 2026-09-02, the eleventh session

Milestone 2 of `docs/flutter-design.md`, the synchronous guarantee.

- A named function type that carries `//gart:sync` names a synchronous export
  position. The 2 other candidates each miss a position that the node set
  holds: a struct tag reaches no function parameter, and a directive on a
  declaration reaches no closure literal.
- The pass needs no analysis of its own. The points-to sets of
  `internal/analyze/flow.go` already hold each function that reaches a struct
  field, a parameter, and a variable, thus the pass is a walk of `a.pts`.
- A value that the flow cannot name falls into the escape group of its
  signature, and the group flows into the cell of the position. Thus an
  unresolved value gives a rejection, and the safe direction needs no extra
  rule.
- A report that names the declaration of a function is not enough. The user
  needs the operation that blocks, thus `summary` now keeps the position of the
  first block operation of each body, and `blockCause` follows the call chain
  to it.
- The named type does not change the Dart type of a field. `Watch.Build` stays
  `Iface? Function()?`, and only the type descriptor names `$desc$Builder`. The
  6 widget tests of the counter pass with no change of the host.
- A `go.mod` of a test that requires a module with `go 1.27.0` must name the
  same patch version. `go 1.27` alone gives "updates to go.mod needed".

### 2026-09-02, milestone 3

The generated node set. `flutter/nodegen` reads the public API of Flutter and
writes 411 Go node types with 3576 fields.

- `package:analyzer` 8.4.1 reads `widgets.dart`, `material.dart`, and
  `cupertino.dart` in about 5 seconds, thus the cost of the read is no gate.
  The 3 libraries hold 940, 1324, and 1022 exported classes, and 323, 515, and
  382 of them are widgets.
- The library of a declaration is not the library that a host can import. The
  analyzer gives `package:flutter/src/material/app.dart`, and an import of that
  path breaks the lint `implementation_imports`. The tool now reads the export
  namespace of 12 public libraries, and it drops each name that no 1 of them
  exports.
- `dart:ui` cannot join the import list of the host. It exports `TextStyle` and
  `Image`, and painting and widgets each export a different declaration of the
  same name.
- An enum with a private constant leaves the switch of the host without a case,
  thus the whole enum stays out. `ContentSensitivity` holds `_unknown`.
- A default value of a parameter reaches the host verbatim. A default that
  names a static member of its own class, such as `defaultThickness`, does not
  resolve there, thus the tool reads each bare word of the default against the
  visible set.
- The Dart tree shaker drops a type descriptor that no code reads. The Go node
  set compiles to 60,518 lines of Dart, and it costs nothing that the
  application does not build.
- A shared host costs 7.2 megabytes. A case names a Flutter widget, thus 411
  cases keep the whole framework. The ahead of time snapshot went from
  13,989,280 bytes with the hand written set of 28 nodes to 20,844,352 bytes
  with a shared host, and back to 13,628,480 bytes with the 18 cases that the
  counter uses. Thus the host is per application. Decision 224.
- A Go node struct cannot live outside the package `ui`. `Node` holds the
  unexported method `isNode`, thus only the declaring package implements it.
- `Scaffold.appBar` takes a `PreferredSizeWidget` and not a `Widget`. A
  parameter that takes any subtype of `Widget` now maps to `Node`, and the host
  casts the result. 2 more node types reached the set for that reason.

### 2026-09-02, the focus list of docs/support.md

The 9 items of the focus list. `docs/support.md` holds the state of each 1.

- Go 1.27 routes `encoding/json` through the v2 implementation by default.
  `GOEXPERIMENT=none` gives the v1 behavior. The overlay follows v1, thus the
  text of a syntax error of `Decoder.Token` differs from the reference.
- `jsonDecode` of dart:convert loses the text of a number literal. A `uint64`
  above the largest `int64` came back as a double and clamped. The runtime now
  holds its own JSON parse.
- `ProcessSignal.watch` of dart:io installs the handler of the platform only
  when a listener joins. A watch that waits for the first read of Go lets the
  default action end the program.
- A live stream subscription keeps the Dart isolate alive after package main
  returns. `$releases` of the scheduler cancels each signal watch.
- `io/fs` compiles from the real Go source. It needed `internal/oserror` in the
  surface, and the overlay of `io/fs` is gone.
- `os` cannot read `path/filepath`, because that package reads `os`. The real
  `os` reads `internal/filepathlite` for the same reason.
- `net/http/fs.go` of the real Go 1.27 compiles with 1 change: the setting
  `httpservecontentkeepheaders` of `internal/godebug` holds its default.
- A function that holds a `defer` and that returns the results of 1 call wrote
  the whole record into the first result variable. The Dart output then held a
  type error. `io/fs` found the bug, and `deferReturn` of
  `internal/normalize/flatten.go` now writes an `AssignMulti`.
- A type that a reflect constructor builds carries an untyped Dart container.
  A real descriptor of the same name replaces it as soon as a value of that
  type reaches the package, thus the shadow of a synthetic descriptor never
  reaches a typed location.
- `reflect.Type` must be an interface. A method of an unexported type emits as
  a private Dart name, thus a pointer to an unexported struct reaches no other
  package.
- A function type that names itself has no finite Dart form. The lexer of
  `text/template/parse` holds `type stateFn func(*lexer) stateFn`, and the
  depth guard of `dartType` wrote 30 levels of it.
- A nil interface result of `reflect.Value.Call` lost its static type, thus
  `ret[1].IsNil()` of `safeCall` of `text/template` panicked. Each of `Call`,
  `Field`, `Index`, and `MapIndex` now carries the static type of the result.
- A method value of reflect must be a plain Dart closure. A pair of a function
  and a receiver holds no form that a Go program can assert to a function type.

### 2026-09-02, proof program 2 of the Flutter path

`flutter/example/apilist`. The application reads the issue list of `golang/go`
over https, it decodes the answer into a Go struct, and it gives a `ListView`
of cards. 6 widget tests pass under Flutter 3.47.0, and 6 Go tests run the same
screen with no Flutter.

- A generic function that gives a pointer needs a read at the call site, and
  not a cast. `ui.P(ui.Sides(12, 6))` wrote `as Insets$?` on a `Box<Insets$>`,
  and the row of the list threw a type error. The body of a generic function
  holds no type argument, thus `&v` there always writes a cell. Decision 229.
- A type assertion to a pointer to a type parameter wrote a cast to `Object?`,
  which `flutter analyze` reported. The package `unique` reaches that shape,
  and `net/http` reads `unique`, thus the counter never met it. Decision 230.
- A widget test holds a fake clock, and a socket of `dart:io` completes in the
  real zone. A poll that gives time to each zone in turn drives a real network
  read: `tester.runAsync` for the socket, and `tester.pump` for the Go
  scheduler. A single `runAsync` that waits for the answer never completes,
  because the continuation of the goroutine waits in the fake queue.
- The idle hook `$onIdle` of the scheduler needs no change for a real socket.
  The hook runs each time that the run queue of Go empties, thus the answer of
  the server reaches the screen with no poll of the host.
- `encoding/json` reads a `time.Time` field from an RFC 3339 text of the
  answer, and the screen holds the date. The nested object `user` reaches a
  named Go struct.
- A widget test needs no network. `HttpServer.bind` of `dart:io` gives a local
  server, and `SetBaseURL` of the Go package points the application at it.

### 2026-09-02, the value types of the node set

9 Dart value types joined the mapping of `flutter/nodegen`. The node set went
from 411 types with 3576 fields to 414 types with 3851 fields, thus 275 more
widget parameters reach Go. Decision 231.

- The earlier note named a controller, a focus node, a shape, a decoration, and
  a scroll physics as the cause of 275 dropped constructors. The measure of the
  blocking parameter of each of the 273 dropped constructors gave 176 distinct
  Dart types, and the largest holds 11. `FocusNode` blocks 3, and `Decoration`
  blocks 2. Thus no opaque handle unlocks a group.
- The value stands in the optional parameters of the 411 nodes that already
  map. 6 of the 9 largest unmapped types hold a value and no identity, thus a
  Go struct or a name carries each 1.
- `FocusNode` in 74 parameters, `ScrollController` in 31, and
  `WidgetStatesController` in 27 each hold an identity. A Go program cannot
  make one, thus an opaque handle needs a producer first.
- `syscall/dart` carries `//go:build dart`. A Go module that names a type of
  that package no longer builds with the go command, thus `gart.dev/ui` would
  need a pair of files and an alias of its own to keep `go test` working.
- A value type of the node set must not take the name of a widget.
  `IconThemeData` maps to `ui.IconStyle`, because the set already holds a
  widget `IconTheme`.
- A callback of 1 argument reaches `cb1Expr`, which throws for an argument that
  it cannot write. `mapFunction` now names the 6 kinds that it writes, thus a
  new value type adds no crash of the generator.

### 2026-09-02, proof program 3 of the Flutter path

`flutter/example/form`. 3 text fields, a validation in Go on each keystroke,
and a POST of the fields as JSON. 6 widget tests pass under Flutter 3.47.0, and
6 Go tests run the same screen with no Flutter. The 3 programs of
`docs/flutter-first.md` are done.

- The program needed no item of the compiler list. `ui.TextInput` of milestone
  1 already keeps the Flutter controller and the focus across a rebuild that Go
  drives, thus a form needs no opaque Dart handle.
- `net/mail` stands outside the subset, thus the address rule reads the text
  itself: 1 at sign, a local part that is not empty, and a domain of 2 labels
  or more.
- `Command.Done` reports a finish of any outcome. A screen that shows the
  answer must read `Done() && Err() == nil`, and a poll of a widget test reads
  `Done()` alone.
- A validation that runs on each keystroke needs no signal of its own. The body
  reads the field signal, thus 1 keystroke marks 1 boundary, and `Validate`
  runs 1 time for the whole screen.

### 2026-09-02, the status codes, math/cmplx, and crypto/subtle

3 items of the next list of `docs/support.md`. The surface went from 86
standard packages to 88, and from 72 full to 74.

- `net/http` holds `status.go` of the real Go 1.27 source now, thus the 62
  codes and `StatusText` each match. The overlay held 39 codes and a map, and
  Go 1.27 holds a switch. `net/http` went from 200 names to 223.
- `math/cmplx` reads the real Go source and needed no compiler change. It
  imports `math` and `math/bits` alone, and each one already stands in the
  surface. 1 line of `internal/overlay/native.go` holds the whole change.
- `crypto/subtle` needs an overlay. `boolToUint8` of
  `crypto/internal/constanttime` panics, and the compiler of Go replaces it
  with an instruction of the machine. The overlay holds the portable form of
  each of the 6 arithmetic names.
- `XORBytes` of Go panics for a destination that overlaps in part. The check
  reads the package `unsafe`, and the documentation of Go says "may panic",
  thus the overlay holds no check. `docs/parity.md` names it.
- `dit_test.go` of Go reads `internal/cpu` and `internal/runtime/sys`, thus
  `make std-test STD=crypto/subtle` gives a rejection of stage 2.
  `testdata/programs/subtleuse` covers the 8 names instead.

### 2026-09-02, the go:embed directive

`embed` joins the surface. The loader reads each directive, it resolves the
patterns, and it writes the files into the declaration of the variable. The
surface went to 89 standard packages, and 75 of them hold each exported name.

- The value must stand at the declaration, and not in an `init`. A package
  level variable that reads an embedded string runs before each `init`, thus
  `var count = len(greeting)` would give 0. Decision 232.
- The go command rejects a directive on a variable that holds a value. The
  rewrite turns `//go:embed` into `// go:embed`, and a comment with a space is
  no directive.
- The rewrite runs on the first of the 2 loads, and the second load reads its
  output through the same overlay map that carries the standard library.
- The overlay of `embed` is the source of Go with 3 changes: `strings` in place
  of `internal/bytealg` and `internal/stringslite`, no `hash` field, and the
  constructor `New`. Stage 2 rejects a read of `New` in a file of the user.
- A directory entry of an `embed.FS` carries a trailing slash, and the list
  sorts by the directory and then by the element, and not by the whole name.
  `New` builds each directory entry and sorts the list, thus the generated
  source holds the leaf files alone.
- The resolution reads the directory tree, and it holds no rule for a nested
  module. `docs/parity.md` names the difference.

### 2026-09-02, the Lookup family of net

The overlay holds a resolver of the DNS. `net` went from 73 names of 346 to
127, and each name of the Lookup family stands.

- The resolver needs no new interop. `dnsmsg.go` writes and reads the message,
  and the socket of `net` already carries UDP and TCP. The earlier note said
  that `dart:io` holds `InternetAddress.lookup` alone and that the family needs
  a resolver of the runtime. A resolver in Go is the better place: it holds the
  same 8 record types on each target, and a test reads it with no network.
- `/etc/hosts`, `/etc/resolv.conf`, and `/etc/services` each read through
  `dart.ReadFile`, thus the overlay imports no `os` and holds no cycle.
- `Resolver.Dial` is the test seam. `testdata/programs/dnslookup` runs a DNS
  server of its own on the loopback interface, and the program gives the same
  output under `go run` and under Dart.
- A method forwarder of a class passed the name of a parameter that the
  emitter had renamed. A boxed parameter takes the suffix `$p`, and
  `methodForward` of `internal/emit/decl.go` wrote the plain name in the
  argument list. `DialContext` is the first method that holds a boxed
  parameter: its body becomes a state machine, and a closure reads the
  parameter that the body writes. `testdata/programs/boxptr` holds the case.
- `LookupPort` needs the `parsePort` of Go. A resolver of the system gives a
  valid port for a number above 65536, thus Go reads the whole text before it
  rejects one, and the error of an unknown port is a `DNSError` and not an
  `AddrError`.
- `net.AddrError` reached the overlay with the family. The error of an unknown
  network reads `address sctp: unknown network`, and `UnknownNetworkError`
  gives another text.

### 2026-09-03, the twelfth session

The session took the whole list of 17 open tasks of `docs/support.md` and
`docs/flutter-first.md`.

**The library surface, 89 standard packages to 97, and 75 full to 84.**

- 5 packages needed no change at all: `compress/lzw`, `compress/bzip2`,
  `archive/zip`, `html`, and `html/template`. Each 1 entered the native list
  and the real Go source compiled. The earlier note "no survey yet" was the
  only thing that held them out.
- `archive/tar` needed a patch of 3 files. `stat_unix.go` reads
  `syscall.Stat_t` and `os/user`, and `sysStat` of `common.go` tests for nil
  already, thus a nil value is the behavior of Go on a platform with no such
  file.
- `math/big` needed a patch of 2 files. `arith_decl.go` declares 6 functions
  with no body, and Go itself ships the pure Go form behind the tag
  `math_big_pure_go`. The patch writes those 6 calls.
- `encoding/xml` needed 6 reflect names, and each 1 builds on a primitive that
  the overlay already holds. `reflect` went from 126 names to 130.
- `crypto/rand` reached 5 of 5, because `Int` and `Prime` read `math/big`.

**4 defects that the work found.**

- A package may hold many functions of the blank name, and `stringer` writes 1
  for each type that it generates. Each 1 took the Dart name `__`, thus the
  output of `html/template` held 6 declarations of that name. That defect
  blocked the package, and it blocked no other package only because no other
  package of the surface holds 2 such functions.
- A generic struct held no `==` and no `hashCode`. `types.Comparable` gives
  false for a struct with a field of a type parameter whose constraint is
  `any`, and the emitter read that answer. Dart then compared 2 instances by
  identity, and a map with such a key found nothing. The GOROOT test
  `typeparam/issue48318` failed on it, and that test reached the compiler for
  the first time on this day, because it reads `encoding/xml`.
- The flag 0 of `fmt` wrote its zeros in front of the base prefix, thus
  `%#04x` of 5 gave `00x5`. The value work of `crypto/tls` found it. A sweep of
  each flag against the go command found 3 more: `%#o` of 0 doubled the zero,
  the verb `O` reached no case, and the flag space reached no code at all.
- `os.OpenFile` dropped `O_EXCL`. `CopyFS` needs that flag, thus the gap showed
  itself as a wrong result and not as a compile error.

**The compiler.**

- `//gart:dart` names a Dart library and a Dart symbol for a Go function
  declaration with no body. Item 1 of `docs/flutter-first.md` asked for it, and
  the value boundary carries 5 types.
- `dart.Object` came with it. Item 2 of that file named 2 conditions, and the
  directive answers the first 1. The second 1 holds: a module that names the
  type needs a twin file behind the tag `!dart`.

**3 measurements that closed an unverified claim.**

- `go build` rejects a bodyless declaration and `go vet` accepts it. `gopls`
  reports nothing, and a control file with a type error gives a diagnostic,
  thus the probe stands. The editor thus costs the external declaration
  nothing.
- The hot reload chain works. `gart watch` read a change of 1 Go line and wrote
  the Dart again, 1 package compiled and 13 stayed cached, and the Flutter hot
  reload took the new Dart in 601 milliseconds. A `fmt.Println` that the change
  added wrote its line after the reassemble.
- The 3 examples each build for macOS, for iOS, and for Android. A web build
  fails: `GoInt` has no constructor under the JS number model, and `$getenv`
  and `$readFile` have no web form. The fourteenth session closed each cause,
  and `flutter build web` writes `build/web` for the 3.

**The Flutter path.** The node set went from 414 types to 427, and from 3851
fields to 4111. 257 constructors stay out against 270 before.

- 6 value types entered: `ButtonStyle`, `ImageSource`, `Decoration`, `Size`,
  `Shadow`, and `Gradient`.
- 3 identity keys entered: `FocusKey`, `ScrollKey`, and `StatesKey`. The host
  owns the object, and the Go program names it. The lifetime follows the
  element tree, and a post frame sweep disposes an object with no user.
- 24 widget tests pass, 6 more than before.

**The regression that the Flutter work found.** `crypto/rand.Int` and
`crypto/rand.Prime` read `math/big`, thus each program that reads `crypto/tls`
carried that package. `math/big` holds a `fmt.Formatter`, and the flow of stage
3 resolves an interface method call to each method of that name and that
signature. `fmt.State.Write` thus reached `(*io.PipeWriter).Write`, which runs
a `select`. A build method that only formats an error then took a rejection,
and 2 of the 3 examples stopped building for a few hours.

The 2 names came out, `math/big` stayed, and
`TestSyncAcceptsAnErrorFormatWithHTTP` holds the shape. `docs/parity.md` names
the real fix: the group of an interface method needs the interface type of the
call site in its key.

**The text property, measured.** A whole tree rebuild of 200 nodes with a text
of 20 characters costs 61.4 microseconds, 0.37% of the frame. The normal case
costs 0.58 microseconds. The read of the optional fields cost 4 times the text
at that hour, and `GoString.fromString` ran 3 to 6 times the cost of
`toDartString`. The work of the same day brought the first to 3 times.
`flutter/bench` holds the harness.

**The reconciler of A10 loses.** It costs 1.82 times the plain walk in the case
that it exists for, and 150 times the signal graph. Decision 249.

**The 16 programs that prove the day.** `make compare PROG=x` runs each 1 with
the go command and with the Dart output, and it diffs the 2.

| Program | What it proves |
|---|---|
| `zipuse` | `archive/zip` writes and reads an archive of 2 files. |
| `taruse` | `archive/tar` writes and reads a header of the PAX format. |
| `lzwuse` | `compress/lzw` round trips both bit orders. |
| `bzip2use` | `compress/bzip2` reads a stream that the `bzip2` command wrote. |
| `htmltpl` | `html/template` escapes each context, and it blocks a `javascript:` URL. |
| `xmluse` | `encoding/xml` marshals and unmarshals an attribute, a comment, a byte slice, and a nested struct. |
| `biguse` | `math/big` holds an `Int`, a `Float` of 200 bits, and a `Rat`. |
| `strconvquote` | The 13 new names of `strconv`, and a sweep of the whole rune range. |
| `osextra` | Each new name of `os` and `os/exec`, against a temporary directory. |
| `httptestextra` | `CloseClientConnections`, `Server.TLS`, and `NewTestServer`. |
| `tlsvalues` | Each constant of `crypto/tls`, each `String` result, and each fallback branch. |
| `dartdecl` | A `//gart:dart` declaration of 6 Dart symbols, with a twin file for the go command. |
| `dartobj` | A `dart.Object` that carries a Dart value through Go, over a JSON round trip. |
| `blankfunc` | 2 functions of the blank name in 1 package. |
| `geneq` | `==` and a map key of a generic struct. |
| `fmthex` | 21 verbs of `fmt` against 7 values, over each flag. |

### 2026-09-03, the thirteenth session

The session took the 9 open tasks of the "what is next" list, and each 1 is
done. 6 subagents ran in parallel on disjoint directories, and 5 more defects
came out of that work.

**The method group of an interface call, the 1 blocker.** The recorded fix was
correct and it was not sufficient. The chain took 3 causes, and only the first
stood in the plan.

1. The key of a group held the name and the signature alone. It now holds the
   interface of the call site, thus a concrete method joins only a group whose
   interface its receiver implements. Each test that cannot answer gives true,
   thus the group never drops a target that a call can reach.
2. A type assertion narrows the interface further. A value of `x.(I)`
   implements `I` and the interface of `x`. `writeMultiple` of `math/big`
   shows the shape: it asserts `io.ByteWriter` on a `fmt.State`, and the wide
   group held each `Write` of the program. A type switch clause takes the same
   rule. A pass before the main walk keeps the pair for each local name that no
   other statement writes.
3. 3 files of `math/big` needed a patch. `intconv.go` and `ftoa.go` wrote the
   message of an unknown verb with `fmt.Fprintf`, which takes an `io.Writer`,
   thus the group widened at the parameter. `natconv.go` guarded the divisor
   cache with a `sync.Mutex`, and `Mutex.Lock` of the overlay waits on a
   channel. The critical section holds `nat` arithmetic alone, thus 1 isolate
   never finds that cache held.

`crypto/rand.Int` and `crypto/rand.Prime` return, decision 250 records the
return, and the 3 Flutter examples each pass the check with `math/big` linked.

**The surface, 97 standard packages to 103, and 84 full to 91.**

- `crypto/x509` gives 106 of 261 names. It parses a DER or a PEM certificate
  and it holds a `CertPool`. Chain verification stays out, because `dart:io`
  owns the trust decision and a `Verify` that always gives nil is a security
  fault. 4 packages entered as native for it: `unicode/utf16`, `encoding/pem`,
  `encoding/asn1`, and `crypto/x509/pkix`.
- `encoding/gob` gives each exported name. 4 things blocked it, and `reflect`
  answers each 1. `Kind` is a `uint` now, with the 27 numbers of Go, because
  `encode.go` holds an array literal that indexes by a kind. `Type.PkgPath`
  enters. `PointerTo` reads the pointer descriptor that the emitter writes, in
  place of a synthetic 1 with an empty method table.
- `testing` goes from 92 of 145 to 132. `TB` holds the 7 names of the real
  `TB`, and the benchmark path now cancels its context in front of the
  cleanups.
- `net` goes from 127 of 346 to 142, and `net/http` from 223 of 392 to 240.
  `TCPConn`, `ListenConfig`, `Protocols`, `ResponseController`, and `Hijacker`
  enter. Each name that stays out needs a socket option, a file descriptor, or
  a raw socket, and `dart:io` exposes none of the 3.
- `crypto/rand` and `net/http/httptest` each reached 100 percent.
  `Server.Certificate` reads, and the overlay carries the pair of
  `net/http/internal/testcert`, thus the fields match the fields of Go.

**The web target works.** The decision of 2026-09-01 let a program that reaches
a file or a socket fail in the Dart compile. That rule made the target
unreachable: the Go package `os` names each function of the io shim, thus a
program that only reads `os.Getenv` failed with 36 missing names, and
`net/http` added 30 more. The 4 web files now declare each name of their native
form, and each unsupported operation reports the cause of Go under `GOOS=js`.
Decision 251 records the change, and `TestWebIO` reads the exact text under
`dart compile js` and under `dart compile wasm`.

**2 measured wins.**

- `GoString.fromString` adopts the buffer of the encoder. `_encode` gave a
  `Uint8List` under the static type `List<int>`, thus `fromBytes` copied every
  byte a second time. The measurement gave 27% to 32%.
- The generated host reads a field by its position, and not by a map. `GoNode`
  held 1 `Map<String, FieldDesc>` for each node kind, and the map took 7.1
  nanoseconds of the 15.9 that 1 optional read cost. The optional fields of a
  200 node frame fall from 38.2 microseconds to 20.3, and the whole frame from
  56.9 to 40.0.

**6 defects that the parallel work found.**

- A promoted member of an unexported embedded type reached no other package.
  The emitter gave it a private Dart name, thus `$net._conn$Close` failed in
  the library of the caller. The privacy of a method now follows the method
  name alone, and an embedded field takes a public name.
- The fix of that defect made a second 1. A Go struct that embeds the builtin
  `int` wrote a Dart field of the name `int`, and that name hides the Dart type
  inside its own class. `compos.go` of the Go test directory holds the shape,
  and the suite caught it. `int`, `double`, `bool`, and `num` joined the
  reserved set of the name pass.
- The address of an index wrote the index with no `$ix` conversion. `GoInt`
  implements `int` on each other target, thus the fault showed itself on
  dart2js alone.
- `gart test` reused a stale Dart package configuration. The freshness check
  read the package file of the output alone, and the output depends on the
  runtime by path, thus a new dependency of the runtime never reached the
  configuration.
- `docs/flutter-options.md` held 233 line references, and a check of the 20
  that a pass mapped on 2026-09-01 found 18 of them wrong 2 days later. The
  file is retired, and each claim that still stands moved into
  `docs/flutter-design.md`. 6 more claims of it are false today.
- The test of the curated surface named `encoding/gob` as its example of a
  package outside the surface. It named `database/sql` until 2026-09-03, and
  that package entered the surface too. `internal/verify` names `plugin` now,
  and `internal/driver/reject_test.go` names `os/user`.

**4 gaps that the work named and did not close.** `reflect.Addr().Interface()`
gives no typed pointer. A pointer to a type parameter loses its method table. A
`reflect.New` of a type that is not a struct gives an untyped box. And
`reflect.Type.String` holds the alias of the source, thus it gives `[]byte`
where Go gives `[]uint8`. The fourteenth session closed each 1.

### 2026-09-03, the fourteenth session

The session took the 3 tasks of the "what is next" list. A fourth task came out
of the work, and each 1 is done. 1 subagent held the reflect work while the
main session held the web work and the Flutter work.

**The web target reaches a real server.** A browser holds no socket, thus
decision 251 left each dial with the cause `operation not supported` and no web
program reached a network. `net/http` now takes the fetch API of the host when
`runtime.GOOS` gives `js`, which is the answer that Go takes under `GOOS=js`. 1
build of the Go code serves each target here, thus the branch stands at run
time and not behind a build tag.

3 shims carry the path. `FetchStart` sends the request and waits for the head
of the answer. `FetchRead` takes 1 chunk of the `ReadableStream` of the body,
thus the program holds no whole body in memory. `FetchClose` cancels the
reader. The host owns 3 answers that a socket owns in Go: it follows a
redirect, it decodes the gzip form, and it picks the protocol. Go gives the
same 3 answers under `GOOS=js`. Decision 252.

`TestWebFetch` runs 1 program against a real server 3 times: with the go
command, with the dart2js build, and with the dart2wasm build. The 3 outputs
are equal. The program reads a status, a header, a body, a request header, a
request body, a body of 100000 bytes through a buffer of 512 bytes, a 404, and
a deadline.

**2 defects that the web work found.**

- A blocking shim must put the Go form of an integer into the frame of the
  wait. The compiler wraps the direct result of a shim, and the frame answer
  skips that wrapper. The 2 forms are equal on each target except dart2js,
  where `GoInt` is a class of its own, and no web shim blocked before this
  change. The rule now stands next to `$await` of `sched.dart`.
- A timeout of a future of an extension type reads the type of the object
  itself. The widening stood at the call, thus dart2wasm rejected the
  `onTimeout` of the deadline helper at run time. The source future now takes a
  `then` that widens it.

`webio` also took the fetch path, thus its node run needed the browser global
`self`, which dart2js reads to schedule a microtask and node does not hold.
`runNodeJS` of `internal/driver/e2e_test.go` writes that 1 line in front of
each dart2js build.

**The 4 reflect gaps of the thirteenth session.** Each 1 gave a wrong answer at
run time, and none was a missing name.

`Value.Addr().Interface()` built a cell of `Object?`, thus the cast of the
emitted code threw. `reflect.New` wrote a `Box` of `Object?`, thus a write of
the result into a typed pointer slot threw and a gob decode into `**int`
panicked. The runtime now reads the kind of the value and it builds the cell of
the Dart type that the kind names.

15 kinds answer from the kind alone. A slice, a map, a channel, a function, and
a pointer each read the Dart type of the element, and that name lives only in
the emitted code. `withType` of `TypeDesc` closes the 5: stage 5 writes a
generic function that names the Dart type, and the runtime calls it with a
builder of a cell. Decision 254.

`encoding/asn1` reads its own type switch over `v.Addr().Interface()` again,
thus the patch of that package now holds 1 change alone: it drops
`internal/saferio`, which reads `unsafe.Sizeof`.
`testdata/programs/asn1parse` reads each of the 7 cases of that switch.

A pointer to a type parameter took 2 fixes. `descriptorOf` wrote the
description inline with an empty method map, and `$ptrDesc` now reads the
description of the type argument through the dictionary. The interface value of
the address carried the cell, thus a method of the pointer reached a `Box`
receiver.

`Type.String` held the alias of the source. `basicText` names the element type,
thus `[]byte` reads `[]uint8` and `rune` reads `int32`. `kindOf` reads it too,
because a named type over an alias gave a kind that the overlay maps to no
`reflect.Kind`.

**The Flutter measure, and a correction of decision 249.** That decision left 1
condition open: A10 earns its cost only when the Flutter work of an unchanged
node exceeds 202 nanoseconds. The Dart harness builds no Flutter widget, thus
it could not read the number.

`flutter/bench/flutter` reads it: **645 nanoseconds**, thus the condition holds
by a factor of 3.2. The first reason of decision 249 falls. With the Flutter
work in the sum, a frame of 200 nodes where nothing changed costs 178.3
microseconds under A5 and 93.0 under A10, thus **A5 costs 1.92 times A10**.
Decision 249 read the opposite ratio, because it counted the Go build and the
Dart walk alone.

The second reason stands and it decides the item. A `ui.Watch` rebuilds 1
boundary in 1.23 microseconds, thus the signal graph beats the floor of A10 by
76 times. A10 stays out. Decision 253.

The measurement needs a profile build. A widget test runs in debug mode, where
the Dart VM interprets the code and each assert of the framework runs, and the
debug number is 10.5 times the profile number. The loop drives
`BuildOwner.buildScope`, because a pump of a real binding waits 8.33
milliseconds for the next frame of the screen.

**A Flutter application builds for the web.** The 3 examples reached macOS,
iOS, and Android on 2026-09-03, and a web build failed on `GoInt` and on 2
functions of the io shim. Decision 251 closed the io shim, and this session
closed the last 8 errors: each bridge wrote `GoInt(x)` and read an element of a
`Slice<GoInt>` as an `int`. `GoInt` erases on the Dart VM and on wasm, thus
only dart2js rejected the code. `$gi` and `$ix` hold on each target.

`flutter build web` now writes `build/web` for the counter, for `apilist`, and
for the form. `apilist` reaches its API through the fetch client. The web
runner of an example comes from `flutter/example/platform.sh <app> web`,
because git holds no such directory.

### 2026-09-03, the fifteenth session

The session took the whole backlog of 43 tasks, and each 1 is closed. 8
subagents ran on disjoint directories, and the main session held the Flutter
graph, the value semantics, the panic texts, the format verbs, and the docs.
The work also found 23 defects that no task named, and it fixed 20 of them.

**The surface went from 91 full packages of 103 to 112 full of 123.** 17 import
paths entered: `database/sql`, `database/sql/driver`, `uuid`, `net/mail`,
`net/smtp`, `net/rpc`, `net/rpc/jsonrpc`, `net/http/cgi`, `go/constant`,
`go/types`, `go/build`, `go/doc`, `go/version`, `hash/maphash`, and 5 vendored
packages of `x/net` and `x/text`. 13 internal packages joined them.

| Package | Before | After |
|---|---|---|
| `reflect` | 130 of 188 | **173 of 188** |
| `net/http` | 240 of 392 | **341 of 392** |
| `net` | 142 of 346 | **218 of 346** |
| `runtime` | 31 of 114 | **43 of 114** |
| `encoding/json` | 59 of 92 | **67 of 92** |
| `crypto/x509` | 106 of 261 | **115 of 261** |
| `testing` | 132 of 145 | **145 of 145, full** |
| `go/types` | outside | **440 of 440, full** |
| `database/sql` | outside | **155 of 155, full** |

**The Flutter node set went from 427 types to 509, and the dropped
constructors from 257 to 174.** The set of 509 costs an application 26,752
bytes less than the set of 427, because the tree shaker drops each type that
the application does not name. `flutter/nodescan` replaced the regular
expression of `usedNodes`, thus the list of the used node types is exact.

**11 defects of the value semantics and of the scheduler.** Each 1 gave a
silent wrong answer or a hang, and each 1 is fixed:

- A range over a map of struct values shared the value and the key with the
  map. Go printed 1 and Dart printed 99.
- `append` shared the element object when the slice grew, and the spread form
  copied no element.
- A `return` inside an `if`, a `switch`, or a `for` ran no deferred call when
  the chain could block. The connection pool of `database/sql` thus leaked
  every connection.
- A method with a pointer receiver lost each write when an argument of the call
  blocked. Each cookie of `net/http` carried an empty value.
- A write to an element of an array wrote into a copy, thus the parser of Go
  lost each type argument of a composite literal.
- A struct result of a call that blocks was shared across the turns of a loop.
- A map with an interface key lost a pointer key after a write to the target.
- A package variable that a field of the receiver shadows read the field.
- A frame class of a local function stood 2 times at package level.
- A local function declaration reached the wrong Dart scope, and a second 1
  reached no other state.
- `runtime.Gosched` never gave the Dart event loop a turn, thus a program that
  spins on it while it waits for a timer hung for ever.

**9 defects of the library.** `encoding/json` read no `MarshalText` and no
`UnmarshalText`, and it crashed on a pointer field to a type that is not a
struct. `strconv.ParseFloat` rejected `Inf`, an underscore, and each
hexadecimal float, and `ParseInt` reported no value out of range. `fmt` read no
star width, no explicit argument index, no precision of a whole number, no
verb `U`, and no escape of a quoted rune, and it sorted the entries of a map by
their text. `strings.ToUpper` kept an invalid byte, and `EqualFold` compared 2
lower cases. `os.WriteFile` dropped the permission bits of a create, and a
relative `Cmd.Path` ignored `Cmd.Dir`. 8 texts of a run time panic differed
from Go.

**3 defects stay open**, and `docs/support.md` holds each 1: the seed corpus of
a fuzz target, the verb `%#v`, and the structural comparison of a pointer
inside an interface.

Read decisions 255 to 310 for the whole account.

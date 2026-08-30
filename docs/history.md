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


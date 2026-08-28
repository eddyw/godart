# Parity

This file lists each place where the Dart output differs from Go. A gap here is
a known difference. A difference that this file does not list is a bug.

## The reflect subset

The compiler writes a type descriptor for each named type. The descriptor gives
the name, the kind, the method table, the struct fields, and the struct tags.
The package `fmt` reads it.

| Feature | State |
|---|---|
| Type identity, kind, and name | Present |
| Struct field name, tag, and value | Present |
| Method table by name | Present |
| Element type and key type of a container | Present |
| `reflect.Value.Set` on a struct field | The write replaces the reference. It does not copy the fields. |
| A descriptor of a generic type | Missing. A value of a generic type does not reach an interface. |
| `reflect.DeepEqual` | Missing |
| A method value through reflect | Missing |
| The size and the alignment of a type | Missing. The target has no memory layout. |

## The runtime

| Item | Go | godart |
|---|---|---|
| Map order | Random for each run | The order of the writes |
| `select` with 2 ready cases | Random | The first case in source order |
| Deadlock | The runtime reports it and exits | The event loop ends and the program exits |
| A pointer value in `%v` | An address | `&` and the value, or `<func>` and `<chan>` |
| `%p` | An address | Not supported |
| A goroutine on a second thread | Yes | No. All Go code runs in 1 isolate. |
| `recover` | Present | Outside the subset |
| `unsafe` and cgo | Present | Outside the subset |

## The subset of the language

| Item | State |
|---|---|
| A pointer to a value that is not a struct | Rejected |
| `new` of a type that is not a struct | Rejected |
| A goto label inside a nested block | Rejected |
| A conversion between 2 struct types | Rejected |
| `import _ "unsafe"` | Present. A blank import performs no operation. |
| A method value of a named type that is not a struct | Rejected |
| A range loop over a function | Rejected |
| A complex number and the builtins `complex`, `real`, and `imag` | Rejected |
| A type declaration inside a function, of a struct type | Present. It takes the class of its shape. |
| A type declaration inside a function, of another type | Rejected |
| An unnamed struct type in an exported declaration | Rejected. Read the section below. |


## The standard library surface

The compiler reaches a standard library package 2 ways. An overlay replaces the
real source with a small file that holds the same behavior. A native package
reads the real Go source, because that source already stays inside the subset.

| Package | Form | Note |
|---|---|---|
| `cmp` | Native | The order of a mixed constraint reads the dictionary. |
| `container/list` | Native | |
| `errors` | Overlay | |
| `iter` | Overlay | `Seq` and `Seq2` only. `Pull` and `Pull2` need a coroutine of the Go runtime. |
| `maps` | Overlay | |
| `internal/bytealg` | Overlay | The search operations that `path` and `strings` reach. |
| `io` | Native | |
| `os` | Overlay | The file access and the process state. Read the section below. |
| `path` | Native | |
| `math` | Native, with 7 patches | 6 patches replace an assembly file, and 1 replaces `unsafe.go`. |
| `math/bits` | Native, with 1 patch | The patch replaces `bits_errors.go`, which needs `go:linkname`. |
| `fmt` | Overlay | |
| `slices` | Overlay | The sort is a stable merge sort. |
| `sort` | Overlay | |
| `strconv` | Overlay | |
| `strings` | Overlay | |
| `sync` | Overlay | |
| `syscall/dart` | Intrinsic | The compiler replaces each call. |
| `testing` | Overlay | |
| `time` | Overlay | |
| `unicode` | Native | |
| `unicode/utf8` | Overlay | The real source needs an index of a mixed type parameter. |

Each other standard library package stays outside the surface. Stage 2 rejects
an import of it with a source position.

A patch replaces 1 file of a native package by its base name. Each other file
keeps the real Go source. A patch that matches no file stays out, because a
build tag of the architecture selects the file set of a package.

The 3 causes that block the next candidate:

| Package | Cause |
|---|---|
| `bytes` | A `recover` call in `buffer.go`, a `goto` label outside the body of a function, and a method value of a named type that is not a struct. |

`bytes` needs 1 patch of `buffer.go` and 2 compiler changes. Each other file of
it already compiles.

## A pointer to a value that is not a struct

`*int`, `*string`, `*[]byte`, and each other pointer now stays in the subset. The
Dart form is a `Box<T>`, and the box holds 1 field with the value.

| Item | godart |
|---|---|
| `&v` of a local variable | Present |
| `&v` of a parameter | Present. The body holds the box. |
| `&v` of a named result | Present |
| `&v` of a package variable | Present |
| `&v` of a receiver that is not a struct | Present |
| `new(T)` | Present |
| `*p` read and write | Present |
| `*p += x` and `*p++` | Present |
| `p == nil` | Present |
| A pointer to a pointer | Present |
| The identity of 2 pointers | Present |
| `&v` of a struct field | Rejected by stage 2 |
| `&v` of an element of a slice, an array, or a map | Rejected by stage 2 |
| `&v` of the result of a call | Rejected by stage 2 |
| `&v` in a declaration of multiple values | Rejected by stage 5 |

The compiler writes a box only for a variable whose address the code takes, thus
a program that takes no address holds no box. A field, an element, and a call
result each live inside another object, and the compiler writes no box for one.

A pointer to a struct and to an array holds no box. The Dart object of a struct
and the Dart List of an array already hold a reference.

## A call that fills each argument

`f(g())` where `g` gives more than 1 result passes each result as its own
argument. The output writes a closure that takes the record of the results, thus
`g` runs 1 time.

## The file access

`os` reaches the file system through `syscall/dart`, and the Dart side uses
`dart:io`. Each operation is synchronous. Go blocks the goroutine at the same
point, thus the block analysis of stage 3 stays simple.

| Item | Go | godart |
|---|---|---|
| `ReadFile`, `WriteFile` | Present | Equal. The permission argument reaches no operation. |
| `Open`, `Create`, `OpenFile` | Present | Equal, for a file. |
| `Open` of a directory | Gives a file | Fails. Dart gives no handle of a directory. |
| `File.Read`, `Write`, `WriteString` | Present | Equal |
| `File.Seek`, `Close`, `Sync` | Present | Equal |
| `Stat`, `Remove`, `RemoveAll`, `Rename` | Present | Equal |
| `Mkdir`, `MkdirAll` | Present | Equal. The permission argument reaches no operation. |
| `FileInfo.ModTime` | Gives a `time.Time` | `ModNanos` gives an integer. `os` holds no import of `time`. |
| `ReadDir` | Gives a slice of `DirEntry` | `ReadNames` gives the names. |
| `Stdin`, `Stdout`, `Stderr` | Present | Equal. A read of `Stdin` stops at a newline. |
| `Getenv`, `LookupEnv`, `Environ` | Present | Equal |
| `Getwd`, `Chdir`, `TempDir` | Present | Equal |
| `Args`, `Exit` | Present | Equal. `Args[0]` is the text `gart`. |
| `IsNotExist`, `ErrNotExist`, `PathError` | Present | Equal |
| `FileMode` | Holds the permission bits | The type exists. No operation reads it. |
| `Chmod`, `Chown`, `Symlink`, `Link` | Present | Missing. Dart gives no such call. |
| `os.Pipe`, `os.StartProcess` | Present | Missing |

`fmt` grows `Fprint`, `Fprintln`, and `Fprintf`. The parameter holds the
interface `fmt.Writer` and not `io.Writer`, because an import of `io` would make
a cycle. A caller never names the type, thus a Go source compiles the same way
with the go command and with gart.

## A range loop over a function

`for k, v := range seq` accepts a function of the form
`func(yield func(K, V) bool)`. The compiler writes the body as a Dart closure
and it passes the closure to the function:

```go
seq(func(k K, v V) bool { BODY; return true })
```

| Item | godart |
|---|---|
| 0, 1, or 2 loop variables | Present |
| `break` of the loop | Present. It gives `return false`. |
| `continue` of the loop | Present. It gives `return true`. |
| A nested loop with its own `break` | Present |
| A nested switch with its own `break` | Present |
| A nested sequence | Present |
| `return` inside the body | Present. A flag carries the result out. |
| A labelled `break` or `continue` that leaves the body | Rejected by stage 2 |
| `goto` out of the body | Rejected by stage 2 |

A `return` inside the body writes each value into a temporary variable, it sets a
flag, and it gives false to the sequence. The code after the call of the sequence
reads the flag. A nested sequence carries the flag out 1 level for each call.

A labelled `break` or `continue` that leaves the body needs 1 flag for each
target and a check at each level between the closure and the target. The flag
also needs a reset when the target loop iterates. The compiler rejects the form
instead.

## The core type of a type parameter

Go gives a type parameter a core type when each term of its type set shares 1
underlying type. `S ~[]E` holds the core type `[]E`, thus each operation of a
slice applies to a value of the type S.

The compiler reads the core type for each operation: an index, a slice
expression, a range loop, `len`, `cap`, `make`, `append`, `copy`, `delete`, a
nil comparison, and a field read.

The Dart type of such a parameter is the Dart type of the core type, and not a
Dart type parameter. `S ~[]E` gives `Slice<E>`. A parameter whose core type is a
basic type keeps a Dart type parameter, because `~int` and `~string` give 2
different Dart types.

## A type parameter with a mixed constraint

A constraint that lists `int | string` holds 2 Dart types. `int` and `GoString`
have no common Dart operator, thus the compiler reads each operation from the
dictionary of the type parameter.

| Operation | godart |
|---|---|
| An assignment, an equality, a call | Present |
| An order comparison | Present. It reads `Dict.order`. |
| An index, a slice expression, a range loop | Rejected by stage 2 |
| `len` and `cap` | Rejected by stage 2 |

A constraint whose terms give 1 Dart type keeps each operator. `int | int64`
gives `int` 2 times, thus `+` and `<` work directly.

## A short variable declaration with a mixed left side

`a, b := f()` may hold a name that already exists. Go declares only the new
names, and it writes the value to each other name. The compiler holds the same
rule.

The left side of `:=` accepts only a name, thus a field and an index never
appear there.

## The order of a package variable

Go gives a package variable its value after each variable that its initializer
reads, also through a call of a function of the package. The declaration order
does not control it. The compiler reads `types.Info.InitOrder`, thus the output
holds the same order.

A variable with no initializer takes its zero value first.

## An unnamed struct type

`struct{ ... }` with no `type` declaration reaches the output as a Dart class.
The name comes from a hash of the text of the type, thus 2 equal Go types give 1
class. The class holds the fields, `$copy`, `$assign`, and the equality
operations when Go holds the type comparable.

| Item | Go | godart |
|---|---|---|
| A local variable and a table | Present | Equal |
| A composite literal | Present | Equal |
| A value in an interface | Present | Equal |
| A type assertion and a type switch | Present | Equal |
| A map key | Present | Equal |
| An exported declaration | Present | Rejected. Read the cause below. |

A Dart class holds a nominal identity, and Go gives an unnamed struct type a
structural identity. The compiler writes 1 class into each package that uses the
type, thus 2 packages give 2 classes that Dart holds apart. Stage 2 rejects an
unnamed struct type in an exported declaration of a package. A declaration of a
named type solves it.

The rule reads the exported names only. A local variable, an unexported
function, and an unexported field all accept an unnamed struct type.

## The test command

`gart test` compiles the test variants of each package and writes 1 entry point
for each test binary. The output holds the format of `go test`.

| Item | Go | godart |
|---|---|---|
| The source position of a log line | `file.go:12: text` | Only the text. The overlay reads no position. |
| `t.Parallel` | Runs the tests in parallel | Does nothing. All Go code runs in 1 isolate. |
| `t.FailNow`, `t.SkipNow` | `runtime.Goexit` | A throw. Each deferred function still runs, and the subset has no `recover`. |
| `-run` | A regular expression | Plain text. gart keeps a test whose name holds the text. |
| A duplicate subtest name | The second name takes `#01` | The 2 names stay equal |
| `TestMain` | Runs in place of the entry point | Equal. `testing.M` holds `Run`. |
| A benchmark | `go test` with no -bench flag runs none | Equal. `gart test -bench <text>` runs them. |
| `b.RunParallel` | Measures a parallel throughput | Stops the benchmark with an error. All Go code runs in 1 isolate. |
| `b.ReportAllocs` | Adds allocs/op and B/op | Does nothing. The Dart VM gives no count of allocations for 1 operation. |
| The clock of a benchmark | A monotonic nanosecond clock | `Stopwatch` of Dart. It also ticks each nanosecond. |
| A benchmark build | The normal test binary | An ahead of time build. The warm up cost of the Dart VM hides the result. |
| A fuzz target | Runs the seed corpus | Reported as a skipped test |
| An example with an output comment | Runs and compares the output | Reported as a skipped test |
| `t.TempDir`, `t.Setenv`, `t.Context` | Present | Missing. The overlay needs the file access. |
| `-bench`, `-benchtime`, `-benchcount` | Present | Present. `-benchtime 100x` becomes `-benchcount 100`. |
| `-benchmem` | Present | Missing. Read the row of `b.ReportAllocs`. |
| `-short`, `-count`, `-timeout`, `-cover` | Present | Missing |
| A panic inside a test | Reports the test and ends the binary | Equal. A panic passes through the harness. |

`testing.B` holds the state of a benchmark. The search of the count of
iterations follows `go test`: the first run takes 1 iteration, and each next
count comes from the time of the last run. The result line holds the format of
`go test`, thus `benchstat` reads the output. The suffix of the name is the
count of isolates, thus it is always 1.

The number measures the Dart output, and not Go. A comparison with the number of
`go test` says nothing. A comparison of 2 gart runs finds a regression, and that
is the purpose.

A `TestMain` function replaces the entry point of a test binary. The compiler
writes a call of `testing.MainWith`, and that function builds the `testing.M`
value and calls `TestMain`. Go passes the result of `m.Run` to `os.Exit` when
`TestMain` returns, and the compiler holds the same rule.

`testing.F` exists, thus a fuzz body type checks. Each method of it stops the
program, and no path reaches such a method.

### The cost of the async model

`testdata/tests/async` measures a call chain of 20 frames. The chain reads a
channel 1 time in its last frame, thus stage 3 marks each frame async.

| Chain | Go | godart |
|---|---|---|
| Synchronous | 8.5 ns/op | 28.9 ns/op |
| Asynchronous | 51.5 ns/op | 3900 ns/op |

An async frame of Dart costs about 180 nanoseconds, because each `await`
schedules a microtask. The block analysis keeps a synchronous path for a
function that cannot block, thus only a chain that can block pays the cost.

A measurement of 2 Dart forms of the same chain of 20 frames, on an ahead of
time build:

| Dart form | Cost |
|---|---|
| `Future<int> f() async` with `await` at each frame | 2400 ns/op |
| `FutureOr<int> f()` with a test of `is Future` and `.then` | 107 ns/op |

The second form is a factor of 22 faster. The compiler does not write it, because
such a form needs continuation passing style: each statement after a call becomes
a closure. The `async` form of Dart writes that transform, and a hand written
form must write it too.

A narrow case needs no such transform. A function whose body is
`return f(x) + 1` holds 1 call and 1 expression, thus `.then` fits directly. The
compiler writes no such case yet.

## The standard library

The verifier rejects each package outside the curated surface.

| Package | Content |
|---|---|
| `errors` | `New`, `Is`, `Unwrap` |
| `fmt` | `Print`, `Printf`, `Println`, `Sprint`, `Sprintf`, `Sprintln`, `Errorf` |
| `sort` | `Slice`, `SliceStable`, `Ints`, `Strings`, `Float64s`, `Search`, `SearchInts` |
| `strconv` | `Itoa`, `Atoi`, `FormatInt`, `ParseInt`, `FormatFloat`, `ParseFloat`, `FormatBool`, `ParseBool`, `Quote` |
| `strings` | The common operations and `Builder` |
| `sync` | `Mutex`, `RWMutex`, `WaitGroup`, `Once` |
| `testing` | `T`, `TB`, `Verbose`, `Short`. Read the section "The test command". |
| `time` | `Duration`, `Sleep`, `Now`, `Since`, and the operations of `Time` |
| `syscall/dart` | The interop. The compiler replaces each call. |

## The isolate worker

`dart.Worker` runs a pure function in a second isolate. The 2 isolates share no
memory, thus the function must not read a package variable and must not write
one. The input and the result cross the boundary as bytes.

Write 2 files with a build tag, thus the same Go source runs in both builds:

```go
//go:build dart

func runWorker(f func([]byte) []byte, input []byte) []byte {
	return dart.Worker(f, input)
}
```

```go
//go:build !dart

func runWorker(f func([]byte) []byte, input []byte) []byte {
	done := make(chan []byte, 1)
	go func() { done <- f(input) }()
	return <-done
}
```

`testdata/programs/worker` holds this pattern.

Gaps of the standard library surface:

| Item | State |
|---|---|
| `sync.RWMutex` | A read lock is equal to a write lock. The single isolate gives no parallel read. |
| `sync/atomic` | Missing. The single isolate makes a plain operation safe. |
| `time.Time` format and parse | Missing |
| `time.Timer` and `time.Ticker` | Missing |
| `strconv.Unquote` | Missing |
| `strings.Map`, `strings.Title` | Missing |
| `sort.Sort` with an interface | Missing. Use `sort.Slice`. |
| `encoding/json` | Missing |
| `io`, `os`, `net` | Missing. The interop package needs the file and the network calls. |
| `regexp` | Missing. Thus `-run` reads plain text. |

The verb set of `fmt`: `%v`, `%+v`, `%T`, `%d`, `%s`, `%q`, `%t`, `%c`, `%b`,
`%o`, `%x`, `%X`, `%f`, `%e`, `%g`, and `%%`. The width and the precision work.
The flag `-` works.

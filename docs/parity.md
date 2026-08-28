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
| A method value of a named type that is not a struct | Rejected |
| A range loop over a function | Rejected |
| A complex number and the builtins `complex`, `real`, and `imag` | Rejected |
| A type declaration inside a function | Rejected |
| An unnamed struct type | Rejected. A table driven test needs a named row type. |
| A `TestMain` function | Rejected. gart writes the entry point of a test binary. |

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
| `TestMain` | Runs in place of the entry point | Rejected by stage 2 |
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

`testing.F` exists, thus a fuzz body type checks. Each method of it stops the
program, and no path reaches such a method.

### The cost of the async model

`testdata/tests/async` measures a call chain of 20 frames. The chain reads a
channel 1 time in its last frame, thus stage 3 marks each frame async.

| Chain | Go | godart |
|---|---|---|
| Synchronous | 8.5 ns/op | 28.9 ns/op |
| Asynchronous | 51.5 ns/op | 3900 ns/op |

An async frame of Dart costs about 200 nanoseconds, because each `await`
schedules a microtask. The block analysis keeps a synchronous path for a
function that cannot block, thus only a chain that can block pays the cost.

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

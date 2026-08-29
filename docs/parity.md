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
| Method table by name and signature | Present. 2 methods of 1 name and 2 signatures stay apart. |
| Element type and key type of a container | Present |
| `reflect.Value.Set` on a struct field | The write replaces the reference. It does not copy the fields. |
| A descriptor of an instance of a generic type | Present. The description stands at the point of the use, and the name holds each type argument. |
| `reflect.DeepEqual` | Present. The walk reads the type descriptor. |
| `reflect.Type` | A struct with a handle of a descriptor of the runtime, and not an interface. Thus `Type` compares with `==` and holds no nil value. |
| `reflect.TypeOf(nil)` | Gives a Type with an empty text and an empty kind. Go gives nil. |
| `reflect.ValueOf`, `Value.Interface`, `Value.Elem`, `Value.Set`, `reflect.Zero`, `Type.Elem` | Present |
| `Value.Elem` of a value that is not a pointer | Panics. Go gives the value inside an interface. |
| Each other method of `reflect.Value` | Missing |
| A method value through reflect | Missing |
| The size and the alignment of a type | Missing. The target has no memory layout. |

## The runtime

| Item | Go | godart |
|---|---|---|
| Map order | Random for each run | The order of the writes |
| `select` with 2 ready cases | Random | The first case in source order |
| Deadlock | The runtime reports it and exits | Equal, with no stack trace |
| A pointer value in `%v` | An address | `&` and the value, or `<func>` and `<chan>` |
| `%p` | An address | Not supported |
| A goroutine on a second thread | Yes | No. All Go code runs in 1 isolate. |
| `recover` | Present | Outside the subset |
| `unsafe` and cgo | Present | Outside the subset |
| `runtime.GOMAXPROCS`, `runtime.NumCPU` | The count of processors | Always 1. All Go code runs in 1 isolate. |
| `runtime.NumGoroutine` | The count of goroutines | Always 1. The runtime library holds no such count. |
| `runtime.GOOS` | A constant of the build | A variable. Dart names the system at the start of the program. |
| `runtime.GOARCH` | A constant of the build | Always `dart64`. A Dart int holds 64 bits on each target. |

## The subset of the language

| Item | State |
|---|---|
| A pointer to a value that is not a struct | Present. It holds a box. |
| `new` of a type that is not a struct | Present |
| A goto label inside a nested block | Present |
| A conversion between 2 struct types | Rejected |
| `import _ "unsafe"` | Present. A blank import performs no operation. |
| A method value of a named type that is not a struct | Present. It writes a closure. |
| A range loop over a function | Present. Read the section below. |
| A deferred call that can block | Present. The chain of the deferred calls holds states of its own. |
| A complex number and the builtins `complex`, `real`, and `imag` | Rejected |
| A type declaration inside a function, of a struct type | Present. It takes the class of its shape. |
| A type declaration inside a function, of another type | Rejected |
| An unnamed struct type with an unexported field, in an exported declaration | Rejected. Read the section below. |


## The standard library surface

The compiler reaches a standard library package 2 ways. An overlay replaces the
real source with a small file that holds the same behavior. A native package
reads the real Go source, because that source already stays inside the subset.

| Package | Form | Note |
|---|---|---|
| `bufio` | Native | The real source needs no patch. |
| `bytes` | Native, with 1 patch | The patch replaces `buffer.go`, which holds a `recover` call. |
| `cmp` | Native | The order of a mixed constraint reads the dictionary. |
| `container/list` | Native | |
| `encoding/base64` | Native | The real source needs no patch. |
| `internal/byteorder` | Native | `encoding/base64` reads it. |
| `context` | Native | The real source needs no patch. |
| `errors` | Overlay | `New`, `Is`, `As`, `AsType`, `Unwrap`, and `Join`. |
| `flag` | Overlay | The real source reads `reflect` and `runtime.Caller`. |
| `reflect` | Overlay | `DeepEqual`, `TypeOf`, `ValueOf`, `Zero`, `Type.String`, `Type.Kind`, `Type.Comparable`, `Type.Elem`, `Value.Elem`, `Value.Set`, and `Value.Interface`. |
| `internal/reflectlite` | Overlay | `TypeOf`, `Type.String`, and `Type.Comparable`. The package `context` reads it. |
| `iter` | Overlay | `Seq` and `Seq2` only. `Pull` and `Pull2` need a coroutine of the Go runtime. |
| `maps` | Overlay | |
| `encoding/json` | Overlay | `Marshal` only. Read the section below. |
| `internal/bytealg` | Overlay | The search operations that `path` and `strings` reach. |
| `io` | Native | |
| `net` | Overlay | TCP, UDP, and a Unix socket. No TLS. Read the section below. |
| `os` | Overlay | The file access, the temporary names, and the process state. Read the section below. |
| `path` | Native | |
| `runtime` | Overlay | `GOMAXPROCS`, `NumCPU`, `NumGoroutine`, `GOOS`, and `GOARCH`. |
| `math` | Native, with 7 patches | 6 patches replace an assembly file, and 1 replaces `unsafe.go`. |
| `math/bits` | Native, with 1 patch | The patch replaces `bits_errors.go`, which needs `go:linkname`. |
| `fmt` | Overlay | |
| `slices` | Overlay | The sort is a stable merge sort. |
| `sort` | Overlay | `Sort`, `Stable`, `Slice`, `SliceStable`, `Search`, `Find`, and the typed forms. |
| `strconv` | Overlay | The integer form, the float form, the bool form, and the quote form. `ParseInt` and `ParseUint` each read a base of 0. |
| `strings` | Overlay | |
| `sync` | Overlay | |
| `syscall/dart` | Intrinsic | The compiler replaces each call. |
| `testing` | Overlay | |
| `sync/atomic` | Overlay | `Value` and `Int32` only. Read the section below. |
| `time` | Overlay | The clock, the timer, the ticker, and the calendar. Read the section below. |
| `unicode` | Native | |
| `unicode/utf8` | Overlay | The real source needs an index of a mixed type parameter. |

Each other standard library package stays outside the surface. Stage 2 rejects
an import of it with a source position.

A patch replaces 1 file of a native package by its base name. Each other file
keeps the real Go source. A patch that matches no file stays out, because a
build tag of the architecture selects the file set of a package.

The next candidate needs a survey of the rejections of stage 2 over the real
source of the package.

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

## The JSON form of a value

`encoding/json` writes the JSON form of a value with no code generator and no
general `reflect`.

The compiler writes a type descriptor for each named type. That descriptor holds
the name, the struct tag, a read, and a write of each field, thus the Dart side
walks it.

| Item | Go | godart |
|---|---|---|
| `Marshal` | Present | Equal |
| `MarshalIndent` | Present | Equal, with an empty prefix. |
| A json tag, `-`, and `omitempty` | Present | Equal |
| A nil slice and a nil map | Give `null` | Equal |
| A whole float | Gives `0`, not `0.0` | Equal |
| A nested struct and a pointer | Present | Equal |
| A map key of the type string or int | Present | Equal |
| The order of a map member | Sorted by the key | Equal |
| A byte slice | Gives base64 text | Gives a list of numbers |
| `Unmarshal` | Present | Present. Read the table below. |
| `Decoder` and `Encoder` | Present | Missing |
| `Marshaler` and `Unmarshaler` | Present | Missing |

`Unmarshal` reads the same descriptor. It writes each field through
`FieldDesc.set`, and it builds a nested value through `TypeDesc.zero` and
`TypeDesc.make`.

| Item of Unmarshal | Go | godart |
|---|---|---|
| A struct, a slice, an array, a map, a pointer | Present | Equal |
| A name that matches only in case | Matches | Equal |
| A member with no field | Stays out | Equal |
| A value of the type `any` | Gives bool, float64, string, []any, map[string]any | Equal |
| A type that does not fit the JSON value | Gives an error | Gives an error with another text |
| A broken text | Gives a `SyntaxError` | Gives an error with another text |
| `UnmarshalTypeError` and its fields | Present | Missing. Each cause is a `SyntaxError`. |
| A number with a fraction into an integer field | Gives an error | Gives an error |

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
| `Setenv`, `Unsetenv` | Writes the environment of the process | Writes a map of the overlay. The Dart VM gives no write to the environment of the process, and the subset starts no other process. |
| `Getwd`, `Chdir`, `TempDir` | Present | Equal |
| `CreateTemp`, `MkdirTemp` | A random name | Equal, with a counter and the clock in place of the random part. |
| `File.ReadAt` | Present | Equal. The call reads the position first and writes it again. |
| `Args`, `Exit` | Present | Equal. `Args[0]` is the text `gart`. |
| `IsNotExist`, `ErrNotExist`, `PathError` | Present | Equal |
| `FileMode` | Holds the permission bits | The type exists. No operation reads it. |
| `Chmod`, `Chown`, `Symlink`, `Link` | Present | Missing. Dart gives no such call. |
| `os.Pipe`, `os.StartProcess` | Present | Missing |

`fmt` holds `Stringer` and `GoStringer`. The formatter reads the method table of
the type descriptor, thus a value with a `String` method writes that text for
the verb `%v` and for the verb `%s`.

`fmt` grows `Fprint`, `Fprintln`, and `Fprintf`. The parameter holds the
interface `fmt.Writer` and not `io.Writer`, because an import of `io` would make
a cycle. A caller never names the type, thus a Go source compiles the same way
with the go command and with gart.

## The network access

`net` reaches TCP, UDP, and a Unix socket through `syscall/dart`. The Dart side
uses `Socket`, `ServerSocket`, and `RawDatagramSocket` of `dart:io`. A program
reads either the `net` package or the interop package.

A dial, a read, a write, a listen, and an accept each wait for a Dart future.
The runtime function `$await` writes a frame of the wait, thus the caller
becomes a state machine and stage 3 marks it. The 2 close operations wait for
nothing.

| Item | Go | godart |
|---|---|---|
| A dial of a TCP address | `net.Dial` | `dart.DialTCP` gives a handle and a cause. |
| A read | `Conn.Read` gives `io.EOF` at the end | `dart.ConnRead` gives 0 with an empty cause. |
| A write | `Conn.Write` | `dart.ConnWrite` waits for the flush of the socket. |
| A close of a connection | `Conn.Close` | `dart.ConnClose` closes both directions. |
| A listener | `net.Listen` | `dart.ListenTCP` gives a handle and the port that the system took. |
| An accept | `Listener.Accept` | `dart.Accept` gives a handle of a connection. |
| A close of a listener | `Listener.Close` | `dart.ListenClose`. Each connection of it stays open, and a Unix listener removes its file. |
| A dial and a listen of a Unix socket | `net.Dial`, `net.Listen` | `dart.DialUnix` and `dart.ListenUnix`. Each other operation reads the table of the connections. |
| A datagram socket | `net.ListenPacket`, `net.Dial` | `dart.ListenUDP` and `dart.DialUDP`, then `dart.PacketRead` and `dart.PacketWrite`. |
| A deadline | `Conn.SetReadDeadline` | `dart.ConnDeadline` and `dart.PacketDeadline` write 2 absolute times. |
| The text of a failure | `dial tcp 127.0.0.1:0: connect: ...` | The message of the Dart `SocketException`. |
| An address of the host name | Present | Only a numeric address and a host name that Dart resolves. |

The `net` package holds this surface:

| Item | Go | godart |
|---|---|---|
| `Dial` | Each network | `tcp`, `udp`, and `unix`. Each other name gives an error. |
| `Listen` | Each network | `tcp` and `unix`. A port of 0 gives a free port. |
| `ListenPacket` | Each network | `udp` only. |
| `Conn` | 8 methods | `Read`, `Write`, `Close`, and the 3 deadline methods. `LocalAddr` and `RemoteAddr` stay out. |
| `PacketConn` | 7 methods | `ReadFrom`, `WriteTo`, `Close`, `LocalAddr`, and the 3 deadline methods. |
| `Listener` | `Accept`, `Close`, `Addr` | Equal |
| `Addr` | `Network` and `String` | Equal |
| `Error` | `Timeout` and `Temporary` | `Timeout` only. `Temporary` is deprecated in Go. |
| `Dialer` | 12 fields | `Timeout`, `Dial`, and `DialContext`. The dial runs on a goroutine, and the select takes the first of the dial and the context. |
| `TCPAddr`, `UDPAddr`, `UnixAddr`, `IP`, `Resolver` | Present | Missing. A type assertion of an `Addr` gives a compile error, and not a wrong value. |
| An empty host of `Dial` | The local system | `127.0.0.1` |
| An empty host of `Listen` | Each interface | `0.0.0.0` |
| `SplitHostPort` | Accepts an IPv6 host in brackets | Only a host with no bracket. |
| `JoinHostPort` | Adds brackets for an IPv6 host | It joins the 2 parts with a colon. |
| TLS | `crypto/tls` | Missing. `SecureSocket` of Dart holds the whole handshake, and the surface holds no `crypto` package. |
| A deadline that a read already waits on | Ends that read | Reaches the next read only. The runtime applies the limit at the start of an operation. |
| A write that a deadline ended | Writes nothing | The bytes stay in the buffer of the socket. |
| A connected UDP socket | The system drops a datagram of another source | The runtime drops it. Dart gives no connect of a datagram socket. |
| The text of a failure | `dial tcp 127.0.0.1:0: connect: ...` | `dial tcp <address>: ` and the message of the Dart `SocketException`. |
| The text of a deadline | `read tcp <local>-><remote>: i/o timeout` | `read tcp: i/o timeout` |

## The clock and the timer

`time` reads the clock of `dart:io` and the scheduler of the runtime. A value of
`Time` holds a count of nanoseconds after the start of the Unix epoch and a
location. The overlay holds no zone database of its own, thus it gives 2
locations: `UTC` and `Local`. `Local` reads the offset and the name of the zone
from Dart at each instant.

| Item | Go | godart |
|---|---|---|
| `Now`, `Since`, `Until`, `Sleep` | Present | Equal |
| `Time.Unix`, `UnixNano`, `Sub`, `Add`, `Before`, `After`, `IsZero` | Present | Equal |
| `Duration` and its 5 read methods | Present | Equal |
| `Duration.String` | `1h2m3s` | Equal |
| `AfterFunc` | Runs f on a new goroutine | Equal. A Dart timer holds 1 wake source of the scheduler. |
| `Timer.Stop` | Reports false after the run | Equal |
| `NewTimer`, `Timer.C`, `Timer.Reset`, `After` | Present | Equal. The channel holds 1 value, thus a write of the timer never blocks. |
| `NewTicker`, `Ticker.C`, `Ticker.Stop`, `Ticker.Reset`, `Tick` | Present | Equal. Each pass starts the timer of the next pass. |
| `Time.String` | `2006-01-02 15:04:05.999999999 -0700 MST` | Equal, with no reading of the monotonic clock. Read below. |
| `Time.Format` | Present | Equal. The scanner holds each chunk of a layout, and the 19 layout constants are present. |
| `Date`, `Month`, `Weekday`, and the name of each | Present | Equal |
| `Time.In`, `Time.Location` | Present | Present |
| `UTC`, `Local` | 2 of the locations of the zone database | The only 2 locations. `FixedZone` and `LoadLocation` stay out. |
| `Time.Year`, `Month`, `Day`, `Hour`, `Weekday` | Present | Missing. `Format` reads the calendar, and no exported method does. |
| `Time.Parse`, `ParseDuration`, `Truncate`, `Round`, `Equal` | Present | Missing |
| The zero value of `Time` | January 1 of the year 1 | The start of the Unix epoch. A Time holds 1 count of nanoseconds, thus the range is the year 1678 to the year 2262. |

A Go `Time` of `time.Now` also holds a reading of the monotonic clock, and
`String` writes it as ` m=+1.234567890`. A `Time` of the overlay holds no such
reading, thus the text ends with the name of the zone.

`time.Date` with the location `Local` reads the offset of the system 2 times.
The Dart runtime names the offset of an instant, and the argument of `Date` is a
reading of a wall clock. The first lookup reads that wall clock as if it were
UTC, and the second one reads the instant that the first offset gives. A wall
clock reading inside 1 offset of a transition of the zone can take the offset of
the other side.

Go reads the zone of an instant from its own copy of the zone database, and Dart
reads the zone from the platform. The 2 agree for each date after 1901. A date
before that year reads a historical offset in Go, for example `+0805 LMT`, and
Dart gives the modern offset of the zone.

`sync/atomic` holds `Value` with `Load`, `Store`, and `Swap`, and `Int32` with
`Load`, `Store`, and `Add`. All Go code runs in 1 isolate, and a goroutine gives
up control only at a block operation. Thus each operation is a plain read or a
plain write.

| Item | Go | godart |
|---|---|---|
| `Value.Load`, `Store`, `Swap`, `CompareAndSwap` | Present | Equal, with no check of the type. |
| `Int32`, `Int64`, `Uint32`, `Uint64`, `Bool` | Present | Equal. Each one holds `Load`, `Store`, `Swap`, and `CompareAndSwap`. |
| A `Store` of a second concrete type | Panics | It writes the value. |
| `Pointer[T]` | Present | Equal |
| The function form, for example `AddInt64` | Present | Equal for `Add`, `Load`, `Store`, `Swap`, and `CompareAndSwap`. |

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
| A labelled `break` or `continue` that leaves the body | Present. A flag carries the branch out. |
| `goto` out of the body | Rejected by stage 4 |

A `return` inside the body writes each value into a temporary variable, it sets a
flag, and it gives false to the sequence. The code after the call of the sequence
reads the flag. A nested sequence carries the flag out 1 level for each call.

A labelled `break` or `continue` that leaves the body writes the flag of its
target and gives false to the sequence. The code after the call reads the flag:
it gives false again while another such body holds it, and it runs the real
branch at the level of the target. The flag takes 0 again at that point, because
the target construct runs its next pass.

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

A conversion of a value of a type parameter to an interface reads `Dict.desc`.
Thus a type switch over `any(v)` gives the real type argument:

```go
func kind[T string | []byte](s T) int {
	switch v := any(s).(type) {
	case string:
		return len(v)
	case []byte:
		return -len(v)
	}
	return 0
}
```

A value of an instance of a generic named type reaches an interface. The package
holds a variable for the description of each other named type, and an instance
reaches the output only through its use, thus the description stands at that
point. The name of the description holds each type argument, and a type
assertion compares that name.

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
The name comes from a hash of the text of the type and of the library of each
field type, thus 2 equal Go types give 1 class. The class holds the fields,
`$copy`, `$assign`, and the equality operations when Go holds the type
comparable.

The class stands in `lib/gart_anon/<name>.dart`. The content reads the shape
only, thus each package that holds the shape writes that file with equal bytes,
and 2 packages share 1 class.

| Item | Go | godart |
|---|---|---|
| A local variable and a table | Present | Equal |
| A composite literal | Present | Equal |
| A value in an interface | Present | Equal |
| A type assertion and a type switch | Present | Equal |
| A map key | Present | Equal |
| An exported declaration, each field exported | Present | Equal |
| An exported declaration, 1 field unexported | Present | Rejected. Read the cause below. |

Go gives an unexported field the identity of the package that declares it, and
Dart holds a name with a leading underscore private to 1 library. Thus the class
of such a shape stays inside 1 package, and 2 packages give 2 classes that Dart
holds apart. Stage 2 rejects such a shape in an exported declaration. A
declaration of a named type solves it, and so does an exported name for each
field.

The rule reads the exported names only. A local variable, an unexported
function, and an unexported field all accept an unnamed struct type with an
unexported field.

## The test command

`gart test` compiles the test variants of each package and writes 1 entry point
for each test binary. The output holds the format of `go test`.

| Item | Go | godart |
|---|---|---|
| The source position of a log line | `file.go:12: text` | Equal. Stage 4 writes the position in front of the call. |
| The position of a log line of a helper that calls `t.Helper` | The position of the caller | The position of the call inside the helper. |
| The position of a deferred log line after `Fatal` | The position of the runtime of Go, for example `panic.go:694` | No position. |
| `t.Parallel` | Runs the tests in parallel | Does nothing. All Go code runs in 1 isolate. |
| `t.FailNow`, `t.SkipNow` | `runtime.Goexit` | A throw. Each deferred function still runs, and the subset has no `recover`. |
| `-run` | A regular expression | Plain text. gart keeps a test whose name holds the text. |
| A duplicate subtest name | The second name takes `#01` | The 2 names stay equal |
| `TestMain` | Runs in place of the entry point | Equal. `testing.M` holds `Run`. |
| A benchmark | `go test` with no -bench flag runs none | Equal. `gart test -bench <text>` runs them. |
| `b.RunParallel` | Measures a parallel throughput | Stops the benchmark with an error. All Go code runs in 1 isolate. |
| `b.ReportAllocs` | Adds allocs/op and B/op | Does nothing. The Dart VM gives no count of allocations for 1 operation. |
| `testing.AllocsPerRun` | Gives the average count of allocations | Skips the test with the cause. A result of 0 would make such a test pass. |
| `b.Loop` | Runs the body 1 time and raises the count itself | Equal. The harness calls a body that reads `Loop` only 1 time. |
| The clock of a benchmark | A monotonic nanosecond clock | `Stopwatch` of Dart. It also ticks each nanosecond. |
| A benchmark build | The normal test binary | An ahead of time build. The warm up cost of the Dart VM hides the result. |
| A fuzz target | Runs the seed corpus | Reported as a skipped test |
| An example with an output comment | Runs and compares the output | Reported as a skipped test |
| `t.TempDir` | A new directory for the test | Equal. The cleanup of the test removes it. |
| `t.Setenv` | Writes the environment for the test | Equal. It writes the map of the `os` overlay, and the cleanup writes the value of the start again. |
| `t.Context` | A context that a cancel closes in front of the cleanups | Equal |
| `testing.TB` | Holds `Setenv` and `Context` | Holds neither. `B` and `F` give no such method. |
| `testing.Benchmark` | Gives a BenchmarkResult | Equal. The field T holds a count of nanoseconds, and not a `time.Duration`. |
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

`gart test` also reads the real test files of a package of GOROOT. Such a
package belongs to no module, and the driver gives it the module `std`. The
command needs no flag for it:

```bash
cd $(go env GOROOT)/src/path && gart test -o /tmp/x -runtime <path> .
```

A test file of GOROOT holds code outside the subset in many packages. `recover`,
`unsafe`, and the address of a byte each appear, and each one stays outside the
subset. Thus a package reaches a full run only when each of its test files stays
inside the subset.

### The block model

A function that can block gives up control with a plain return. It writes each
local into a frame object, and each caller on the stack does the same. Thus the
stack unwinds to the scheduler, and the scheduler runs the chain again when the
operation is ready. The body of such a function becomes a flat state machine: a
loop with a switch on a state variable.

The model holds no `Future` and no `await`. A call that gives up no control
costs 1 state assignment and 2 tests.

`testdata/tests/async` measures a call chain of 20 frames. The chain reads a
buffered channel 1 time in its last frame, thus no frame really waits and stage
3 still marks each frame.

| Chain of 20 frames | Go | godart |
|---|---|---|
| Synchronous | 8.5 ns/op | 26 ns/op |
| Asynchronous | 51.5 ns/op | 189 ns/op |

The `async` form of Dart cost 3900 ns/op for the same chain, because each
`await` schedules a microtask. The unwind model is a factor of 21 faster.

The block analysis of stage 3 keeps a synchronous path for a function that
cannot block, thus only a chain that can block pays the cost.

These rules follow from the model:

- A machine whose deferred calls can block holds the chain of those calls in 3
  states of its own. A return statement writes each result into the frame and
  jumps to the chain, and the chain runs 1 call for each pass. Such a call
  gives up control exactly as each other call does. The try block of the panic
  stands inside the loop of the machine, thus a panic also reaches the chain,
  and the frame keeps the panic until the last call ends.
- A local that a closure reads and that the code writes takes a box. A resume
  reads each local from the frame into a new call, thus the closure and the
  frame must share 1 value. A struct and an array need no box, because an
  assignment to such a variable writes the fields of the same object.
- The emitted Dart of a function that can block no longer follows the shape of
  its Go source.

### The end of the program

Package main runs as the first goroutine. Go ends the program when that
goroutine returns, and the runtime holds the same rule.

The runtime reports a deadlock when no goroutine can run and no timer waits:
`fatal error: all goroutines are asleep - deadlock!` on the error output, with
the status code 2. Go writes a stack trace of each goroutine, and the runtime
writes only the line.

## The standard library

The verifier rejects each package outside the curated surface.

| Package | Content |
|---|---|
| `errors` | `New`, `Is`, `Unwrap` |
| `fmt` | `Print`, `Printf`, `Println`, `Sprint`, `Sprintf`, `Sprintln`, `Errorf` |
| `sort` | `Slice`, `SliceStable`, `Ints`, `Strings`, `Float64s`, `Search`, `SearchInts` |
| `runtime` | `GOMAXPROCS`, `NumCPU`, `NumGoroutine`, `GOOS`, `GOARCH`. The scheduler lives in the Dart runtime library. |
| `strconv` | `Itoa`, `Atoi`, `AppendInt`, `FormatInt`, `ParseInt`, `FormatFloat`, `ParseFloat`, `FormatBool`, `ParseBool`, `Quote` |
| `strings` | The common operations and `Builder` |
| `sync` | `Mutex`, `RWMutex`, `WaitGroup`, `Once` |
| `testing` | `T`, `TB`, `B`, `F`, `Verbose`, `Short`, `AllocsPerRun`. Read the section "The test command". |
| `context` | Each function of the package. The real Go source compiles. |
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

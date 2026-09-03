# Parity

This file lists each place where the Dart output differs from Go. A gap here is
a known difference. A difference that this file does not list is a bug.

`docs/support.md` holds the count of the names that each package of the surface
holds. This file holds the behavior.

## At a glance

The state of 2026-09-03. Each row names the section that holds the detail.

| Area | The difference |
|---|---|
| A goroutine, a channel, and a `select` | Present. A function that blocks unwinds each frame to the scheduler, and the output holds no `Future` and no `await`. |
| A parallel run | Out. All Go code runs in 1 isolate. `GOMAXPROCS` and `NumCPU` give 1, and a goroutine with no block operation never gives up control. |
| The order of a map | The order of the writes. Go gives a random order. |
| A `select` with 2 ready cases | The first case in source order. Go takes a random one. |
| `sync.RWMutex` | A read lock is equal to a write lock. `sync.Map` is a plain map, and it needs no lock. |
| A pointer | Present, also to a value that is not a struct. `&*p` and a conversion between 2 struct types stay out. |
| `unsafe` and cgo | Outside the subset. |
| `reflect` | A subset over a type descriptor. `Type` is a struct with a handle, and not an interface. |
| An address | None. `%p` stays out, and a program counter is an index of a table of the runtime. |
| An integer | 64 bits on each target. The JS target holds the pair form of GopherJS, and `reflect` and `json` read 53 bits there. |
| A `Time` | 1 count of nanoseconds, thus the range is the year 1678 to the year 2262, and it holds no monotonic value. |
| The zone of a date before 1901 | The modern offset of the zone. Dart names the zone, and Go reads its own database. |
| The file access | Synchronous, through `dart:io`. No handle of a directory, and no file descriptor. `Chmod`, `Chown`, and `Chtimes` reach a path, and `Stat` still reports a fixed mode. |
| The process access | Present through `Process` of `dart:io`. No signal to a child, and no process group. `Getuid` and its family give -1. |
| The network access | TCP, UDP, and a Unix socket. `IP`, `IPMask`, `IPNet`, `ParseCIDR`, `TCPAddr`, `UDPAddr`, `IPAddr`, `UnixAddr`, `TCPListener`, and `UnixListener` are present. The overlay holds a resolver of the DNS since 2026-09-02, thus the whole `Lookup` family reads. No raw socket, no file descriptor, and no socket option. |
| HTTP | 1.1 and 2, for the client and the server. A cookie jar, a proxy, a multipart form, and the file server family (`FileServer`, `FileServerFS`, `FS`, `Dir`) are present. |
| TLS | The secure socket of Dart holds the handshake. No cipher choice, and no root pool. `http.ListenAndServeTLS` and the `tls.Dialer` read since 2026-09-04. `crypto/x509` parses a certificate, a request, and a revocation list, and it checks no signature. |
| JSON | The client surface is present: `Marshaler`, `Unmarshaler`, `TextMarshaler`, `TextUnmarshaler`, `Decoder`, `Encoder`, `RawMessage`, `Number`, `Token`, `Delim`, and `UseNumber`. The text of a syntax error of `Token` differs. |
| `gart test` | The format of `go test`, with `t.Parallel`, `-cover`, `-short`, and `-count`. It runs each example, and it runs the seed corpus of each fuzz target. No `-fuzz` flag, thus it generates no input. |
| A benchmark | Present, with the format of `go test`. No count of allocations. |
| A stack trace | The module form of each file, with no offset. |
| The garbage collector | `runtime.GC` gives up control only, and a cleanup follows the collector of Dart. |
| The web target | `dart compile js` and `dart compile wasm` each build any program. `net/http` reaches a server through the fetch API. A file or a socket operation gives the error that Go gives under `GOOS=js`. |

## The receiver of a method

The Dart form of a pointer to a struct is the object reference, and that type
accepts null. Go runs a method with a nil pointer receiver, and Dart calls no
member of null.

Thus a method with a pointer receiver on a struct type holds its body in a top
level function. The class holds a member that calls that function. The method
table of the type also calls the top level function, thus a nil pointer inside
an interface reaches the method.

A method of a generic type takes the same form. The top level function names the
type parameters of the receiver type first, and it takes the dictionary of each
one. Thus a nil receiver of such a method also runs.

## The name of a method

The method table of a type descriptor holds 1 key for each method. The key names
the method, each parameter type, and each result type. An alias gives the name
of the type behind it, because 2 packages can hold 2 alias names of 1 type.

The walk replaces an alias at the top level, and inside a pointer, a slice, an
array, a map, and a channel. An alias inside a struct, inside a signature, or
inside a type argument stays, thus 2 such methods reach 2 keys.

An unexported field of a type of another package reaches no accessor of its
descriptor. A Dart library holds each name that starts with an underscore. The
description of an instance of a generic type stands at the point of the use.
That point can lie outside the package of the type. `%+v` of such a value writes
`<nil>` for the field.

## The reflect subset

The compiler writes a type descriptor for each named type. The descriptor gives
the name, the kind, the method table, the struct fields, and the struct tags.
The package `fmt` reads it.

| Feature | State |
|---|---|
| Type identity, kind, and name | Present |
| Struct field name, tag, and value | Present |
| Method table by name and signature | Present. 2 methods of 1 name and 2 signatures stay apart. |
| An unexported method of a type of another package | The method table holds a placeholder that panics. A Dart library reaches no name that starts with an underscore. |
| Element type and key type of a container | Present |
| `reflect.Value.Set` on a struct field | The write replaces the reference. It does not copy the fields. |
| A descriptor of an instance of a generic type | Present. The description stands at the point of the use, and the name holds each type argument. |
| `reflect.DeepEqual` | Present. The walk reads the type descriptor. |
| `Type.String` | Present. The text follows `reflect`, and not `go/types`. An unnamed struct type and an unnamed interface type each hold 1 space after the keyword. An alias gives the name of the element type since 2026-09-03, thus `[]byte` reads `[]uint8` and `rune` reads `int32`. |
| `reflect.Type` | A struct with a handle of a descriptor of the runtime, and not an interface. Thus `Type` compares with `==` and holds no nil value. |
| `reflect.TypeOf(nil)` | Gives a Type with an empty text and an empty kind. Go gives nil. |
| `reflect.ValueOf`, `Value.Interface`, `Value.Elem`, `Value.Set`, `reflect.Zero`, `Type.Elem` | Present |
| `Value.Kind`, `Type`, `Len`, `Index`, `NumField`, `Field`, `CanSet`, `IsValid` | Present |
| `Value.Call` | Present. A function that gives up control stays out, because the call reads no scheduler. |
| `Value.SetMapIndex`, `Value.SetZero`, `reflect.MakeMap`, `reflect.TypeAssert` | Present |
| `Type.NumIn`, `In`, `NumOut`, `Out`, `Key` | Present |
| `Type.IsVariadic` | Always false. The description of a function type holds no such flag. |
| `Value.Int`, `Uint`, `Float`, `Bool`, `String`, and each `Set` form of them | Present |
| `reflect.New`, `reflect.MakeSlice`, `reflect.Indirect` | Present |
| `Type.NumField`, `Type.Field`, `Type.Len`, `Type.Name`, `StructTag.Get` | Present |
| `reflect.Kind` | Equal since 2026-09-03. The type is a `uint` with the 27 constants of Go under `iota`, and each number matches Go. `String` gives the text of Go, and an array that indexes by a kind thus compiles. |
| `Type.PkgPath` | Present since 2026-09-03. A named type gives the import path of its package, and each other type gives an empty text. |
| `Value.Elem` of a value that is not a pointer | Panics. Go gives the value inside an interface. |
| `Type.Size`, `Type.Align`, `Type.FieldAlign`, `StructField.Offset` | Present. The result follows the rules of a 64-bit machine, exactly as the compiler of Go computes them. |
| `Value.UnsafeAddr` | Not present. Go gives the address of the value, and Dart gives no address. `Value.Addr` builds a new cell at each call for a field of a scalar type, thus 2 reads of 1 field would give 2 numbers. |
| `reflect.SliceHeader`, `reflect.StringHeader` | Present since 2026-09-04. Each field holds a plain number, thus a read gives the value that the program wrote. The subset rejects the import of `unsafe`, thus no conversion of a slice or of a string reaches a header. |
| `Value.Complex` and `Value.SetComplex` | Not present. `Complex64` and `Complex128` stand in the kind set, and no `Value` method reads such a value. |
| `Type.NumMethod`, `Type.Method`, `Value.NumMethod`, `Value.Method` | Present. The order follows the name, exactly as in Go. `Method` gives the name and the index only. |
| `Value.IsNil`, `Value.MapIndex` | Present |
| `Value.MapKeys` | Present. The order follows the insertion, and Go gives a random order. |
| `Value.Addr().Interface()` | A typed pointer since 2026-09-03, thus a type assertion of it reads. The description of a slice, a map, a channel, a function, and a pointer carries the Dart type of a value, because the kind alone does not name it. |
| `reflect.New` of a type that is not a struct | A typed cell since 2026-09-03, thus a write of it into a typed pointer slot reads. |
| A pointer to a type parameter, inside a generic function | The method table stands since 2026-09-03. The description of the pointer reads the type argument through the dictionary, thus `Type.Implements` of it answers as Go does. |

## A type that reflect builds

`SliceOf`, `ArrayOf`, `MapOf`, and `PointerTo` each give a `Type` that no value
of the program carries.

The name of the type goes through the table of the runtime first. A type that
the program already holds keeps its own descriptor, thus `MakeSlice`,
`MakeMap`, `New`, and `Zero` of it each give a value with the Dart type of the
emitted code, and a type assertion of that value works.

A type that the program never holds gives a descriptor that the runtime builds.
Such a descriptor carries an untyped container: a `List<Object?>` for a slice
and for an array, and a `GoMap<Object?, Object?>` for a map. The Dart type of a
Go type lives only in the emitted code, thus the runtime cannot build the typed
form.

| Item | Go | godart |
|---|---|---|
| `SliceOf`, `ArrayOf`, `MapOf`, `PointerTo` | Present | Equal. `_withType` threads through the 3 synthetic builders since 2026-09-03, thus the container holds the Dart type of the element. |
| `MakeSlice`, `MakeMap`, `New`, `Zero` of such a type | A value of the type | Equal. A struct descriptor and each named descriptor carry a factory of the Dart type since 2026-09-03. |
| A type assertion of such a value to its Go type | The value | Equal since 2026-09-03. |
| `MakeFunc` | Present | Equal for each function type that the program names. The descriptor of a signature carries a bind factory that builds a closure of the exact Dart type. A type that the program never names holds no factory, and `MakeFunc` of it panics with that cause. |
| `FuncOf` | Present | Present. A signature that the program names 1 time gives its own description, thus the type equals the type of a value of it and `MakeFunc` of it works. A signature that no code names gives the shape and the text of Go, and `MakeFunc` of it panics with the cause, where Go builds a value. Read decision 325. |
| `StructOf` | Present | Not present. It builds a type that no emitted code names, thus no factory exists for it. |
| `Method.Func` | Present | Not present. The field needs a factory of the Dart type of each method, with the receiver in front. The emitter must write 1 for each exported method of each type of a method table, and that costs 2.7% of the Dart source of a program. Read decision 325. |
| `Method.Func` | A func value whose first parameter is the receiver | Not present. `_withReceiver` builds that type with no bind factory, thus `MakeFunc` of it panics. |
| A function that gives up control | Runs inside `Value.Call` and inside a `MakeFunc` body | Rejected with a clear cause. |
| A method value, from `Method`, `MethodByName`, or `Value.Method` | A function value of the method | Equal. The bind factory of the signature gives a closure of the exact Dart type, thus an assertion to a function type with a result reads. Decision 228 held the old limit. |
| `Value.Pointer`, `Value.UnsafePointer` | The machine address | The identity of the object. 2 reads of the same object give the same number, and the number names no memory, thus `runtime.FuncForPC` of it gives nil. |
| `Type.ChanDir` | The direction of the channel type | Equal. The type descriptor carries the direction since 2026-09-03. |
| `Type.AssignableTo`, `ConvertibleTo`, `Implements` | The rule of the specification | The name of the type, the method set of an interface, and the kind of a number each answer. A conversion between 2 named struct types of the same shape gives false. |

`fixedbugs/issue22605` and `fixedbugs/issue37716` of the test directory of Go
witnessed the limit until 2026-09-03. Each 1 builds a type that the program
never names, and it then asserts the value to the Go type. The 2 pass now, and
`internal/driver/gotest_test.go` holds no entry for either.

## The runtime

| Item | Go | godart |
|---|---|---|
| Map order | Random for each run | The order of the writes |
| `select` with 2 ready cases | Random | The first case in source order |
| Deadlock | The runtime reports it, writes the stack of each goroutine, and exits | Equal. The report walks the frame chain of each live goroutine. Go also writes the arguments, an offset, and a created-by line, and the chain holds none of them. The goroutine numbers can differ. |
| A pointer value in `%v` | An address | `&` and the value, or `<func>` and `<chan>` |
| `%p` | An address | Not supported |
| A goroutine on a second thread | Yes | No. All Go code runs in 1 isolate. |
| `recover` | Present | Present. A deferred call of the panicking function reads the value, and each other call gives nil. A panic inside a deferred call replaces the chain panic. |
| `unsafe` and cgo | Present | Outside the subset |
| `runtime.GOMAXPROCS`, `runtime.NumCPU` | The count of processors | Always 1. All Go code runs in 1 isolate. |
| `runtime.NumGoroutine` | The count of goroutines | Always 1. The runtime library holds no such count. |
| `runtime.GOOS` | A constant of the build | A variable. Dart names the system at the start of the program. |
| `runtime.GOARCH` | A constant of the build | Always `dart64`. A Dart int holds 64 bits on each target. |
| A stack trace of a panic | The path of the build machine in each frame, and an offset | The module form: the package path and the base name, as a build with `-trimpath` gives. No offset. The text of a line directive passes verbatim, and a relative directive resolves against the directory of the source, exactly as in Go. |
| A frame of a goroutine that waits | Present | Present in the deadlock report, from the frame chain of the scheduler. A panic trace still holds the frames of the running goroutine only. |
| `runtime.Caller`, `runtime.Callers`, `runtime.CallersFrames`, `runtime.FuncForPC` | The program counters of the stack | Equal, with the module form of each file. A program counter is an index of a table of the runtime, and not an address. |
| The frame of a deferred call | The line of the return, or the closing brace at a fall or a panic | Equal, except 2 forms: a return inside a defer chain that can block gives the brace, and the panic path of Go holds a frame of gopanic that the chain does not hold. |
| `runtime.AddCleanup` | Present | Present, with a Dart `Finalizer`. The time of the run follows the collector of Dart. |
| `runtime.GC` | Runs a collection and waits for it | Allocates a block of garbage and gives up control. Dart holds no call that forces a collection, thus a cleanup can stay for a later call. |
| The stack of a goroutine | Grows without a fixed limit | The Dart VM holds a fixed stack, thus a deep recursion overflows. `stack.go` of the Go repository is such a program. |
| `runtime.GOROOT` | The root of the Go tree | An empty text. The Dart output holds no such tree. |
| `runtime.GoroutineProfile`, `runtime.StackRecord` | 1 record for each live goroutine | Present since 2026-09-04. The scheduler gives the frames of each goroutine that waits, and a program counter is an index of a table of the runtime. Go stops the world for the read, and 1 isolate needs no stop. |
| `runtime.Pinner` | The collector moves the object no more, and it frees it no more | Present since 2026-09-04. The Pinner keeps a reference, thus the collector of Dart frees no pinned object. The subset holds no address, thus no program reads the position of an object. Go ends the program when a Pinner reaches the collector with a pin, and gart writes no such report. |
| `runtime.CPUProfile` | Panics with `CPUProfile no longer available` | Equal since 2026-09-04. Go removed the operation. |
| `runtime.SetDefaultGOMAXPROCS` | Writes the default value again | Present since 2026-09-04. All Go code runs in 1 isolate, thus the value is already 1 and the call needs no operation. |
| `runtime.MemStats`, `runtime.ReadMemStats` | The numbers of the heap | Not present. The Dart VM reports no number of its heap inside the process. Read `docs/support.md`. |
| The memory profile, the block profile, the mutex profile, and the trace | Present | Not present. The Dart VM counts no allocation, no block, and no contention, and it writes no event of the tracer of Go. |
| `runtime.SetFinalizer` | The finalizer takes the object itself | Not present. A Dart `Finalizer` must hold no reference of the object that it watches. `runtime.AddCleanup` maps to the Dart form. |
| `runtime.Breakpoint` | A trap of the machine | Not present. `debugger` of `dart:developer` waits for a client of the service with no report, and Go ends the program with SIGTRAP. |
| `debug.SetMaxStack` | Writes the limit of the runtime | Keeps the value only. |
| `filepath.EvalSymlinks` | Walks each element itself | Asks the system for the resolved path. A relative argument takes `Rel` against the resolved working directory, thus the result stays relative in each usual case. The error of a missing path holds a different text. |
| A panic of a run time error that recover reads | The value implements `error`, and `Error()` gives the text | Equal in the text. The value carries the description `runtime.errorString` in each case, and Go also holds `runtime.plainError` and `*runtime.TypeAssertionError`. A type switch on the recovered value thus reads 1 type where Go reads 3. |
| The text of a run time panic | Read the list of Go | Equal. A write to a nil map and the 3 faults of a channel write no prefix, a slice bound report names the bound that failed and writes the unsigned value of a bound of an unsigned type, and a failed assertion names the static interface type and the first missing method. Read decision 293 and decision 294. |
| `fmt.Errorf` | `*errors.errorString`, `*fmt.wrapError`, or `*fmt.wrapErrors` | Equal. Read decision 295. |
| A star width and a star precision, `%*d` and `%.*f` | Present | Equal since 2026-09-03. A value of a wrong type gives `%!(BADWIDTH)`, and a precision below 0 gives `%!(BADPREC)`. |
| An explicit argument index, `%[2]d` | Present | Equal since 2026-09-03. An index that names no argument gives `%!d(BADINDEX)`. |
| A precision on a string verb, `%.2s` | Cuts the text to 2 runes | Equal since 2026-09-03. |
| The verb `%#v` | The Go syntax form, `main.point{X:3, Y:4}` | Not present. It writes the form of `%v`. |

## The subset of the language

| Item | State |
|---|---|
| A pointer to a value that is not a struct | Present. It holds a box. |
| `new` of a type that is not a struct | Present |
| A goto label inside a nested block | Present |
| A conversion between 2 struct types | Rejected |
| A conversion of a slice to an array, as in `[4]byte(s)` | Present. It panics for a slice that is too short. |
| The builtin `clear` of a slice | Present |
| A method with a nil pointer receiver | Present. Such a method holds its body in a top level function. |
| A method with a pointer receiver on a generic type | The method stays a member of its class, thus a nil receiver panics. |
| A function declaration with no body | Present. Stage 2 rejects a read of it, and the Dart form panics. |
| `import _ "unsafe"` | Present. A blank import performs no operation. |
| A method value of a named type that is not a struct | Present. It writes a closure. |
| A method value of an interface value, for example `r.Read` | Present. The closure holds the value, and each call reads the method table. |
| A range loop over a function | Present. Read the section below. |
| A deferred call that can block | Present. The chain of the deferred calls holds states of its own. |
| A complex number and the builtins `complex`, `real`, and `imag` | Present. Each operation of `complex64` rounds the 2 parts to 32 bits. The division is the division of the Go runtime. `fmt` writes the `(1+2i)` form for the verbs `v`, `f`, `e`, and `g`. |
| A type declaration inside a function, of a struct type | Present. It takes the class of its shape. |
| A type declaration inside a function, of another type | Present. It takes the Dart form of its underlying type. |
| A method expression, for example `(*T).M` | Present. The value is a closure whose first parameter is the receiver. |
| A generic function as a value, for example `equal[int]` | Present. The value is a closure that gives the dictionary of each type argument. |
| An unnamed struct type with an unexported field, in an exported declaration | Rejected. Read the section below. |


## The standard library surface

The compiler reaches a standard library package 2 ways. An overlay replaces the
real source with a small file that holds the same behavior. A native package
reads the real Go source, because that source already stays inside the subset.

| Package | Form | Note |
|---|---|---|
| `bufio` | Native | The real source needs no patch. |
| `bytes` | Native | The real source needs no patch. |
| `cmp` | Native | The order of a mixed constraint reads the dictionary. |
| `container/list` | Native | |
| `encoding/base64` | Native | The real source needs no patch. |
| `regexp` | Native | The real source needs no patch. |
| `regexp/syntax` | Native | The real source needs no patch. |
| `encoding` | Native | The package holds interfaces only. |
| `compress/flate` | Native | The real source needs no patch. The output holds the same bytes as Go. |
| `compress/gzip` | Native | The real source needs no patch. |
| `compress/zlib` | Native | The real source needs no patch. |
| `internal/testhash` | Native | The tests of the hash packages of the Go repository read it. |
| `embed` | Overlay | The source of Go, with a constructor and with `strings` in place of 2 internal packages. Read the note below. |
| `math/cmplx` | Native | The real source needs no patch. It imports `math` and `math/bits` alone. |
| `crypto/subtle` | Overlay | The 8 names of Go, each in its portable form. Read the note below. |
| `encoding/binary` | Overlay | The source of Go, with no complex case and with a plain map in place of `sync.Map`. |
| `encoding/gob` | Overlay | The source of Go. The wire bytes are the real ones, and the type ids of the bootstrap keep the numbers 1 to 23. The speed helpers, `internal/saferio`, `sync.Pool`, and `atomic.Pointer` each stay out, and the general path writes the same bytes. A decode into a pointer to a pointer whose element is not a struct panics. |
| `encoding/asn1` | Patch | The source of Go. `asn1.go` drops `internal/saferio`, which holds `unsafe.Sizeof`. `marshal.go` writes a composite literal in place of a conversion between 2 struct types. |
| `encoding/pem`, `crypto/x509/pkix`, `unicode/utf16` | Native | Each name of Go. |
| `crypto/md5` | Overlay | The source of Go, with the generic block function and no FIPS mode. |
| `crypto/sha1` | Overlay | The source of Go, with the generic block function and no FIPS mode. |
| `crypto/sha256` | Overlay | The algorithm of `crypto/internal/fips140/sha256`, with the public names of `crypto/sha256`. |
| `crypto/sha512` | Overlay | The algorithm of `crypto/internal/fips140/sha512`, with the public names of `crypto/sha512`. |
| `crypto/hmac` | Overlay | `New` and `Equal`, from FIPS 198-1. |
| `crypto/rand` | Overlay | `Read`, `Reader`, and `Text`, over `Random.secure` of Dart. `Int` and `Prime` read `math/big`, thus the 2 stay out. |
| `mime`, `mime/quotedprintable`, `mime/multipart` | Native | A patch of `mime/multipart` replaces the `go:linkname` stub of `readMIMEHeader`. |
| `net/textproto`, `net/http/cookiejar` | Native | |
| `net/http` | Overlay | An HTTP/1.1 client and an HTTP/1.1 server. Read the section of the HTTP access. |
| `net/http/httptest` | Overlay | `NewRecorder`, `NewRequest`, `NewServer`, `NewUnstartedServer`, `NewTLSServer`, and `StartTLS`. |
| `testing/quick` | Overlay | The source of Go, with no complex case. |
| `crypto/tls` | Overlay | A client and a server over the secure socket of Dart. `Dial`, `DialWithDialer`, `Listen`, `X509KeyPair`, `Config`, `Certificate`, `Conn`, and `ConnectionState`. |
| `internal/byteorder` | Native | `encoding/base64` reads it. |
| `context` | Native | The real source needs no patch. |
| `errors` | Overlay | Each name of Go. `As` takes no pointer to an interface type. |
| `flag` | Overlay | Each name of Go. The real source reads `reflect` and `runtime.Caller`. |
| `reflect` | Overlay | 130 of the 188 names of Go. The section "The reflect subset" above holds each one. |
| `internal/reflectlite` | Overlay | `TypeOf`, `Type.String`, and `Type.Comparable`. The package `context` reads it. |
| `iter` | Overlay | `Seq`, `Seq2`, `Pull`, and `Pull2`. Read the section below. |
| `maps` | Overlay | |
| `encoding/json` | Overlay | Each name of the client surface. Read the section below. |
| `internal/bytealg` | Overlay | The search operations that `path` and `strings` reach. |
| `io` | Native |
| `io/fs` | Native | Each exported name, the `FS` interface family included. |
| `log` | Overlay | Each function and each method, with no `Llongfile` and no `Lshortfile`. Read below. |
| `path/filepath` | Overlay | The separator is always `/`. Read below. | |
| `net` | Overlay | TCP, UDP, a Unix socket, `IP`, `ParseIP`, and the `Lookup` family. No TLS. Read the 2 sections below. |
| `os` | Overlay | The file access, the environment, the identity of the process, and the process state. `Root`, `DirFS`, `Chmod`, and the signals each read. `File.Fd`, `NewFile`, and `StartProcess` stay out, because `dart:io` hides the descriptor. Read the section below. |
| `path` | Native | |
| `runtime` | Overlay | `GOMAXPROCS`, `NumCPU`, `NumGoroutine`, `GOROOT`, `GOOS`, `GOARCH`, `Caller`, `Callers`, `CallersFrames`, `FuncForPC`, `AddCleanup`, and `GC`. |
| `runtime/debug` | Overlay | Each name of Go. `ReadBuildInfo` gives no record, and `ReadGCStats` writes the zero value. |
| `math` | Native, with 7 patches | 6 patches replace an assembly file, and 1 replaces `unsafe.go`. |
| `math/bits` | Native, with 1 patch | The patch replaces `bits_errors.go`, which needs `go:linkname`. |
| `math/rand` | Native, with 1 patch | The patch replaces `rand.go`. `rng.go` holds the algorithm, thus a seeded sequence stays equal. |
| `math/rand/v2` | Native, with 1 patch | The patch replaces `rand.go`. `pcg.go` and `chacha8.go` hold the 2 algorithms. |
| `internal/chacha8rand` | Native, with 2 patches | The 2 patches replace the assembly of `block` and the 2 views of 1 memory block. |
| `container/heap` | Native | The real source needs no patch. |
| `container/ring` | Native | The real source needs no patch. |
| `encoding/ascii85` | Native | The real source needs no patch. |
| `encoding/csv` | Native | The real source needs no patch. |
| `encoding/hex` | Native | The real source needs no patch. |
| `hash` | Native | The 4 interfaces that each hash package holds. |
| `hash/adler32` | Native | The real source needs no patch. |
| `hash/crc32` | Native, with 5 patches | Each patch replaces the file of 1 architecture, thus the table of the portable path runs. |
| `hash/fnv` | Native | The real source needs no patch. |
| `net/netip` | Native, with 1 patch | The patch replaces `uint128.go`, which gives the address of a field. |
| `net/url` | Native | The real source needs no patch. |
| `unique` | Overlay | `Handle` and `Make`. The map keeps each value, because the subset holds no cleanup. |
| `internal/godebug` | Overlay | Each setting holds its default, thus `Value` gives an empty text. |
| `fmt` | Overlay | Each name of Go. |
| `slices` | Overlay | The sort is a stable merge sort. |
| `sort` | Overlay | Each name of Go. The order of `Slice` and of `Sort` is a quick sort, and it is not stable. |
| `strconv` | Overlay | The integer form, the float form, the bool form, and the quote form. `ParseInt` and `ParseUint` each read a base of 0. |
| `strings` | Overlay | Each name of Go. `NewReplacer` skips a pair whose old text is empty. |
| `sync` | Overlay | `Mutex`, `RWMutex`, `Map`, `Cond`, `Locker`, `WaitGroup`, `Once`, `OnceFunc`, `OnceValue`, `OnceValues`, and `Pool`. |
| `syscall/dart` | Intrinsic | The compiler replaces each call. |
| `testing` | Overlay | |
| `sync/atomic` | Overlay | `Value`, the 6 typed values, `Pointer`, and the function forms, with `And` and `Or`. The `unsafe.Pointer` functions stay out. Read the section below. |
| `time` | Overlay | The clock, the timer, the ticker, and the calendar. Read the section below. |
| `unicode` | Native | |
| `unicode/utf8` | Overlay | The real source needs an index of a mixed type parameter. Each test of the Go repository passes. |

Each other standard library package stays outside the surface. Stage 2 rejects
an import of it with a source position.

A patch replaces 1 file of a native package by its base name. Each other file
keeps the real Go source. A patch that matches no file stays out, because a
build tag of the architecture selects the file set of a package.

The next candidate needs a survey of the rejections of stage 2 over the real
source of the package.

## The packages of 2026-09-03

7 packages entered the surface on this day, and each 1 below names a behavior
that differs from Go.

### archive/tar

`FileInfoHeader` writes no owner name and no group name. The real
`stat_unix.go` reads `syscall.Stat_t` and `os/user`, and the surface holds
neither 1, thus the patch leaves `sysStat` nil. Each other field of the header
reads as Go writes it.

### math/big

The package reads the real Go source, and a 2 file patch replaces the 6
functions that assembly holds. Each 1 takes the pure Go body of `arith.go`,
which is the same body that the tag `math_big_pure_go` gives. The results are
equal, and the speed of a large multiplication is lower than the speed of Go on
the same machine.

`arith_amd64.go` reads `internal/cpu` to find the ADX instruction. The patch
gives the flag the value false, and no caller reads it on the pure Go path.

### encoding/xml

The package reads the real Go source. It needed 6 reflect names, and
`docs/support.md` names them. `Value.Bytes` of an array of bytes gives a copy,
because the runtime holds no address of an element. Go gives the array itself
when the value holds an address. A write through the result of that case thus
reaches no array here.

### crypto/rand

`Int` and `Prime` stay out, and the cause is the section below and not
`math/big`. The 2 read `math/big`, thus each program that reads `crypto/tls`
would carry that package. `crypto/rand` thus holds 3 of the 5 names of Go.

A program that wants a big random number imports `math/big` itself and writes
the 12 lines of `Int`. That program pays the cost below only when it also holds
a `//gart:sync` position.

## The method group of an interface call

The flow of stage 3 resolves an interface method call to each method of that
name and that signature in the program. It reads no interface type at the call
site, thus a method of a type that the interface rejects still joins the group.

1 chain shows the cost. `fmt.Formatter` names `Format(fmt.State, rune)`, and a
body of it calls `s.Write`. The group of `Write([]byte) (int, error)` holds
`(*io.PipeWriter).Write`, which runs a `select` and thus blocks. `io.PipeWriter`
implements no `fmt.State`, and the group takes it anyway.

The effect reaches the `//gart:sync` check alone. A program with 1
`fmt.Formatter` in it makes each call of `err.Error()` reach that `select`, thus
a build method that only formats an error takes a rejection. Each of the 3
Flutter examples that read `net/http` hit this on 2026-09-03, for the few hours
that `crypto/rand` carried `Int`.

`TestSyncAcceptsAnErrorFormatWithHTTP` of `internal/driver/sync_test.go` holds
the shape, thus the fault cannot return without a report.

**The fix.** The group needs the interface type of the call site as part of its
key, and a concrete method joins only a group whose interface its receiver
implements. The check stays sound, because a dynamic dispatch through an
interface reaches no type outside that interface. The work is a change of
`methodCell` and of the wiring in `internal/analyze/flow.go`, and it is open.

### crypto/tls

The overlay holds each value type, and `dart:io` owns the handshake. A `Config`
field that would change that handshake stays out, because such a field would
report support and change nothing. `Config.PreferServerCipherSuites` is the 1
field that stands, and Go documents it as ignored.

`Certificate` holds the DER chain and the parsed leaf since 2026-09-03, and
`X509KeyPair` fills the 2 as Go 1.23 does. `LoadX509KeyPair` reads the 2 PEM
files of the file system since 2026-09-04. `ConnectionState.PeerCertificates`
and `Config.RootCAs` each stay out, because the runtime holds no call that
reads the peer certificate of a secure socket.

| Item | Go | godart |
|---|---|---|
| `Dialer` | `NetDialer`, `Config`, `Dial`, `DialContext` | Equal. The dial runs on a goroutine, and the select takes the first of the dial, the context, and the cancel channel of the `net.Dialer`. |
| `DialWithDialer` | The timeout covers the dial and the handshake | Equal in the result. 1 call of the runtime opens the socket and runs the handshake, thus the limit covers the 2 together. |
| `NetDialer.Resolver` | Reads the host name, and the handshake keeps the name of the user | Equal. The dial takes the address that the resolver gave, and `ServerName` of the config wins over the host name. |
| `Conn.Handshake`, `Conn.HandshakeContext` | Runs the handshake | Each 1 gives nil. Dart runs the handshake inside the dial, thus each `Conn` of this package already holds a complete handshake. Go also gives nil at that point. |
| `ConnectionState.NegotiatedProtocolIsMutual` | Always true | Equal. Go marks the field deprecated with the same rule. |

### crypto/x509

The overlay parses a certificate, a certificate request, and a revocation list,
and it reads each field of the 3. It builds no chain and it checks no
signature, because `dart:io` owns the trust decision. A `Verify` that always
gives nil is a security fault, thus the name is absent and a program that calls
it takes a compile error.

| Item | Go | godart |
|---|---|---|
| `ParseCertificate`, `ParseCertificates` | Present | Equal for each field that the overlay reads. |
| The fields of a `Certificate` | 40 fields | 39. `PublicKey` alone stays out, and each other field reads. The name constraint fields, `Policies`, `PolicyIdentifiers`, `PolicyMappings`, and the 6 policy constraint fields each read since 2026-09-04. |
| `OID`, `ParseOID`, `OIDFromInts`, `OIDFromASN1OID` | Present | Equal. The port reads the real Go source, and an arc above 64 bits keeps its value. |
| `ExtKeyUsage.OID` | Present | Equal |
| `CertificateRequest`, `ParseCertificateRequest` | Present | Present, less `PublicKey` and `CheckSignature`. The parse reads `encoding/asn1`, and it fills each other field. |
| `RevocationList`, `RevocationListEntry`, `ParseRevocationList`, `ParseCRL`, `ParseDERCRL` | Present | Present, less `CheckSignatureFrom`. |
| `SystemCertPool` | The root store of the platform | Present, and it always gives a `SystemRootsError`. The secure socket of Dart owns the store, and it exports no root certificate. |
| `SetFallbackRoots` | Names the roots of a platform with no store | Not present. Go reads the pool inside `Verify` alone, and the overlay holds no `Verify`. |
| `IsEncryptedPEMBlock` | Present | Equal. `DecryptPEMBlock` and `EncryptPEMBlock` stay out, because `crypto/des` and `crypto/aes` stand outside the surface. |
| `Certificate.PublicKey` | An `any` of 1 of 3 key types | Not present. `crypto/rsa`, `crypto/ecdsa`, and `crypto/ed25519` stand outside the surface, thus the field could hold nil alone. `PublicKeyAlgorithm` names the algorithm, and `RawSubjectPublicKeyInfo` holds the DER. |
| `Certificate.Verify`, `VerifyOptions`, and each verify error | Present | Not present. The trust store belongs to `dart:io`. |
| `Certificate.CheckSignature` and its 2 companions | Present | Not present. A check needs `crypto/rsa` and `crypto/ecdsa`. |
| `Certificate.VerifyHostname` | Reads the names and the name constraints | Present. It matches the subject alternative names alone. |
| `CreateCertificate` and each other name that signs | Present | Not present. Each 1 needs a private key. |
| `CertPool` | 8 methods | `NewCertPool`, `AddCert`, `AppendCertsFromPEM`, `Subjects`, `Equal`, and `Clone`. |
| The text of a parse error | The text of `cryptobyte` | The text of `encoding/asn1`. The overlay reads the DER with that package. |
| A critical extension of a policy | Go reads `nameConstraints`, `certificatePolicies`, `policyMappings`, `policyConstraints`, and `inhibitAnyPolicy` | Equal since 2026-09-04. The parse reads each of the 5, thus none of them lands in `UnhandledCriticalExtensions`. |
| A URI of a subject alternative name | `url.Parse`, a label check, and a reverse label walk of Go | `url.Parse` and the label check. 1 rare malformed URI passes here. |

`testdata/programs/x509use` reads a certificate that the program holds as a PEM
constant, and it prints each field. A second check parsed the 128 certificates
of `/etc/ssl/cert.pem` under the 2 commands, and each output was equal.
`testdata/programs/n2x509` proves the `OID` family, each policy field, the
certificate request, and the revocation list.

### net/http/httptest

`Close` does not wait for a request that stands open. A panic of a handler ends
the program, because the server of the overlay recovers no panic and holds no
`ErrAbortHandler`. With `EnableHTTP2`, `TLS.NextProtos` reads
`[h2 http/1.1]` where Go reads `[h2]`.

`Server.Certificate` reads since 2026-09-03. The overlay carries the pair of
`net/http/internal/testcert`, thus the fields of the certificate match the
fields that Go gives.

### os and os/exec

| Name | The difference |
|---|---|
| `Stat` after a `Chmod` | The mode of the result does not change. `dart:io` reports no permission bit. The write itself reaches the file, thus an exec of it behaves as in Go. |
| A create with a permission, `os.WriteFile(name, data, 0o755)` | The bits of the call without the mask | Equal. The chmod stands after the open, and the new interop `Umask` reads the mask. A mode of 0644 after the mask runs no command. |
| A relative `Cmd.Path` next to a `Cmd.Dir` | The child resolves it against Dir | Equal. The overlay joins the 2, because `Process.start` of Dart resolves against the parent. |
| `Chtimes` | The precision is 1 millisecond. `dart:io` names a `DateTime`. |
| `File.Chmod`, `File.Chown`, `File.Truncate` | Each 1 reaches the file by its path. Go reaches it by its descriptor, thus a rename between the 2 calls changes the target here. |
| `OpenFile` with `O_EXCL` | A check and an open, and not 1 operation of the system. 2 processes can pass the check together. |
| `File.SetDeadline` and its 2 companions | Each 1 gives `ErrNoDeadline`. A read of a file is 1 synchronous call that nothing interrupts. Go gives the same error for a file whose poller cannot hold a deadline. |
| `Chmod`, `Chown`, `Link` | Each 1 runs a command of the platform, because `dart:io` names no permission bit, no owner, and no hard link. |

### The fmt verbs

| Verb | The difference |
|---|---|
| `%#v` of a pointer that Go writes as an address | Go writes `(*int)(0xc000014048)` for a pointer to a value that is not a struct, and for each pointer below the top level. The runtime writes `&` and the value, because Dart gives no address. |
| `%#v` of a channel or of a function that is not nil | Go writes `(chan int)(0xc000014048)`. The runtime writes `<chan>` and `<func>`, because Dart gives no address. A nil channel and a nil function each take the form of Go. |
| `%#v` of a text with a rune above the ASCII range that Go does not print | Go writes the escape `\u0085`. The runtime writes the rune itself. The table of the printable categories of Unicode lives in `strconv`, and the runtime holds no such table. |
| `%#v` of an error of the package `errors` | Go writes `&errors.errorString{s:"x"}`. The overlay names the field `text`, thus the output holds `text` in place of `s`. |
| `%T` and `%#v` of an instance of a generic type | Go writes `main.pair[int,string]`. The compiler writes `main.pair[int, string]`, with a space after the comma. `reflectText` of `internal/emit/iface.go` reads `types.TypeString`, and that function writes the space. |
| `%#q` | It gives the plain form of `%q`. Go writes a backquote when the text holds none. |
| `%g` with a precision | The precision reaches no digit. `strconv.FormatFloat` passes the verb to the runtime, and `_float` of the runtime reads the precision for `f` and for `e` alone. |
| A verb that the type rejects, with a width | The width reaches no text. Go writes `%!s(int=       5)` for `% 8s` of 5, and the runtime writes `%!s(int=5)`. A program that reaches this row already holds a bug. |

`strconv.ParseFloat` stays loose. It reads `double.tryParse` of Dart, thus it
accepts text with a space around it, it accepts a bare `0x10`, and it reports
no `ErrRange`. `ParseComplex` holds a scanner of its own and it matches Go.

## The go:embed directive

A `//go:embed` directive works, and the variable takes the type `string`,
`[]byte`, or `embed.FS`. The value stands at the declaration, thus a package
level variable that reads it sees the files.

The compiler of Go writes the file list of an `embed.FS` into the data of the
program, and this compiler holds no directive of the linker. Thus stage 1 reads
each directive, it resolves the patterns against the directory of the package,
and it writes the files into the source of the declaration. The go command
rejects a directive on a variable that holds a value, thus the same rewrite
turns `//go:embed` into a plain comment.

2 differences stand.

`embed.New` is a name of the overlay, and the real package of Go holds none.
Stage 2 rejects a read of it in a file that holds no directive, thus a program
that compiles here also compiles under the go command.

A nested module stays inside the set. The go command drops each file of a
directory that holds a `go.mod` of its own, and the resolution of stage 1 reads
the directory tree alone. A pattern that reaches such a directory thus gives
more files here than under the go command.

The other rules follow the go command: a pattern that names no file gives a
rejection, a pattern with `..` or an absolute path gives a rejection, a name
that starts with a dot or with a low line needs the prefix `all:`, and a
variable of the type `string` or `[]byte` takes 1 file.

## The Lookup family of net

The overlay holds a resolver of its own. `dnsmsg.go` writes and reads the
message, and `lookup.go` holds the `Resolver` and each `Lookup` function. The
real source of Go reads the resolver of the platform through cgo, or its own
resolver over the vendored package `golang.org/x/net/dns/dnsmessage`, and each
one stands outside the surface.

The resolver reads /etc/hosts first, and then each server of /etc/resolv.conf
over UDP. A truncated answer takes a second exchange over TCP. `LookupPort`
reads /etc/services, and `parsePort` of Go holds the rule for a number.

5 differences stand.

- `Resolver.PreferGo` and `Resolver.StrictErrors` each read no field. This
  resolver is always the resolver of Go, and each error of a server reaches the
  caller.
- The order of the addresses of `LookupIP` follows the answer, and Go sorts by
  RFC 6724. A caller that needs 1 order sorts the result.
- `LookupMX` and `LookupSRV` each sort by the preference and by the priority
  alone. Go also shuffles the records of equal weight, and this resolver keeps
  the order of the answer, thus 2 runs give 1 order.
- The configuration of /etc/resolv.conf and the table of /etc/services each
  read 1 time for each process. Go reads the configuration again after 5
  seconds.
- A record type outside the 8 of `dnsmsg.go` reaches no `Lookup` function. The
  8 are A, AAAA, CNAME, NS, PTR, MX, TXT, and SRV.

## The constant time operations

`crypto/subtle` holds each exported name of Go, and 2 differences stand.

`XORBytes` of Go panics for a destination that overlaps `x` or `y` in part.
The check of Go reads the package `unsafe`, and the documentation of Go says
"may panic", thus the overlay holds no such check. A call with such an overlap
writes the bytes in order, and it gives no report.

`WithDataIndependentTiming` runs the function and adds nothing. The mode of the
processor that holds the timing of each instruction steady needs an
instruction of the machine, and the Dart target reaches none. Go behaves the
same way on a platform that holds no such mode. `dit_test.go` of Go reads
`internal/cpu` and `internal/runtime/sys`, thus `make std-test STD=crypto/subtle`
gives a rejection of stage 2. `testdata/programs/subtleuse` covers the 8 names
instead.

The other 6 functions each give the result of Go. The real source of Go 1.27
calls `boolToUint8` of `crypto/internal/constanttime`, and that function
panics, because the compiler of Go replaces it with an instruction of the
machine. The overlay holds the portable form of each 1.

## A pointer to a value that is not a struct

`*int`, `*string`, `*[]byte`, and each other pointer now stays in the subset.
The Dart form is a `$Ptr<T>`. A pointer to a variable holds a `Box<T>` with the
value, and a pointer to a field or to an element holds a `$Ref`.

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
| `&v` of a struct field | Present. The pointer holds the object and the index of the field. |
| `&v` of an element of a slice or an array | Present. The pointer holds the backing list and the index. |
| `&v` of a composite literal of a type that is not a struct | Present. The pointer holds a box of its own. |
| `&*p` | Rejected by stage 2. Read `p`. |
| `&v` in a declaration of multiple values | Rejected by stage 5 |

The compiler writes a box only for a variable whose address the code takes, thus
a program that takes no address holds no box. A field and an element live inside
another object, thus the pointer holds a `$Ref` of that object and the index. Go
compares 2 pointers with `==`, and `$Ref` compares the object and the index for
it.

A pointer to a struct and to an array holds no box. The Dart object of a struct
and the Dart List of an array already hold a reference.

A pointer to a type parameter holds 1 form for both: the Dart type is `Object?`,
`$deref` reads the value, and `$setDeref` writes it. The form of such a pointer
follows the type argument, because a pointer to a struct is the object itself.

A pointer inside an interface value follows the same rule. `$goEqual` and
`$goHash` of the runtime read the value: a `$Ptr` takes `==` and `hashCode`, and
each other pointer takes `identical` and `identityHashCode`. Thus a map with an
interface key that holds a pointer answers as Go does. 2 pointers to a value
with no size stay 2 locations here, and gc can give the 2 pointers 1 address.

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
| `MarshalIndent` | Present | Equal |
| A json tag, `-`, and `omitempty` | Present | Equal |
| A nil slice and a nil map | Give `null` | Equal |
| A whole float | Gives `0`, not `0.0` | Equal |
| A nested struct and a pointer | Present | Equal |
| A map key of the type string or int | Present | Equal |
| The order of a map member | Sorted by the key | Equal |
| A byte slice | Gives base64 text | Equal |
| `Unmarshal` | Present | Present. Read the table below. |
| `Decoder` and `Encoder` | Present | Equal. Read the table below for the text of a syntax error of `Token`. |
| `Marshaler` | Present | Equal. Read the section below. |
| `TextMarshaler` | The walk reads it when the type holds no `MarshalJSON` | Not present |
| A `time.Time` field | Gives `"2023-11-14T22:13:20Z"` | Equal |
| `<`, `>`, and `&` in a string | Give `\u003c`, `\u003e`, and `\u0026` | Give the character itself. A reader gives 1 value from the 2 forms. |

`Marshaler` and `Unmarshaler` reach the walk through the method table of the
type descriptor. A method with a pointer receiver stays out of the value method
set, thus the description of a value type names the description of its pointer
type. The compiler writes that link only when the pointer type holds a method
that the value type does not hold.

| Item of the 2 interfaces | Go | godart |
|---|---|---|
| A value receiver and a pointer receiver on `MarshalJSON` | Present | Equal |
| A pointer receiver on `UnmarshalJSON` | Present | Equal |
| A value receiver on `UnmarshalJSON` | The walk calls it with a copy, and the write is lost | The walk does not call it |
| The bytes of `MarshalJSON` | Go writes them into the output | The walk reads them into a tree and writes that tree again, thus the output holds the compact form |
| The bytes of `UnmarshalJSON` | Go gives the bytes of the input | The walk reads the whole input first, thus the method reads the compact form |
| A broken form from `MarshalJSON` | Gives an error | Gives an error with another text |
| A byte slice whose element type holds `MarshalJSON` | Gives a list | Equal |

`Unmarshal` reads the same descriptor. It writes each field through
`FieldDesc.set`, and it builds a nested value through `TypeDesc.zero` and
`TypeDesc.make`.

| Item of Unmarshal | Go | godart |
|---|---|---|
| A struct, a slice, an array, a map, a pointer | Present | Equal |
| A name that matches only in case | Matches | Equal |
| A member with no field | Stays out | Equal |
| A value of the type `any` | Gives bool, float64, string, []any, map[string]any | Equal |
| A type that does not fit the JSON value | Gives an `UnmarshalTypeError` | Equal, with the text of Go |
| A broken text | Gives a `SyntaxError` | Equal, with the text and the offset of Go |
| `UnmarshalTypeError` | Present | Present, with `Value`, `Struct`, and `Field`. Read the section below. |
| An RFC 3339 text into a `time.Time` | The `UnmarshalJSON` method reads it | Equal |
| A base64 text into a byte slice | Present | Equal |
| A number with a fraction into an integer field | Gives an error | Gives an error |

### The stream forms and the error types

A scanner in the Go half of the package walks 1 JSON value of a buffer. It
reports the end of the value and it reports a syntax error with the text and the
offset of Go. `Decoder`, `Valid`, `Compact`, and `Indent` each read it, and
`Unmarshal` reads it in front of the Dart walk.

| Item | Go | godart |
|---|---|---|
| `NewDecoder`, `Decode`, `More`, `Buffered`, `InputOffset` | Present | Equal |
| `Decoder.DisallowUnknownFields` | Present | Equal |
| `Decoder.Token` | Present | Present. A valid input gives the same tokens. The text and the offset of a syntax error differ, because Go 1.27 routes `encoding/json` through the v2 implementation and that implementation blames a different character. `GOEXPERIMENT=none` gives the text of the overlay. |
| `Decoder.UseNumber` | A number of an `any` gives a `Number` | Present. The parse of the runtime keeps the text of each literal, thus the `Number` holds the text of the input. |
| `NewEncoder`, `Encode`, `SetIndent` | Present | Equal |
| `Encoder.SetEscapeHTML` | Turns the escape of `<`, `>`, and `&` on and off | Present, and it changes nothing. The encoder never escapes the 3 characters. |
| `Valid`, `Compact`, `Indent` | Present | Equal |
| `RawMessage` | Present | Equal. The bytes hold the compact form of the input. |
| `Number` and its 3 methods | Present | Equal. A field of that type holds each digit of the input. |
| `SyntaxError` | `Offset` and the text | Equal |
| `UnmarshalTypeError` | `Value`, `Type`, `Offset`, `Struct`, `Field` | `Value`, `Struct`, and `Field`. Read below. |
| `InvalidUnmarshalError`, `UnsupportedTypeError`, `UnsupportedValueError`, `MarshalerError` | Present | Present, with the text of Go. Read below. |

Each error type of Go holds a field `Type` of the type `reflect.Type`. The walk
of the type descriptor runs in the Dart runtime library, and it reports the
name of the type as a text. `reflect` of the surface builds no `Type` from a
name, thus 3 of the 5 such fields cannot hold a value at all. The text of the
type stands in an unexported field, thus each `Error` method writes the text of
Go. `UnmarshalTypeError` also holds no `Offset`, and `MarshalerError.Err` holds
a new error with the text of the original one.

`InvalidUnmarshalError.Type` and `UnmarshalFieldError.Type` are the 2 that a
`reflect` import would fill, and a measure of 2026-09-04 kept the 2 out. The
import adds 348 KB of Dart source and 208 KB of the AOT binary to each program
that reads `encoding/json`. `docs/support.md` holds the numbers and the rule,
and decision 320 holds the trade.

The Dart walk gives a cause to the Go half through 1 string. A cause that starts
with a null byte holds a code and the parts of a typed error, and the Go half
builds the error of that code.

### The scan family of fmt

`Sscan`, `Fscan`, `Scan`, and each other member read the runes of an input. A
scanner of the overlay holds the state, and each error text matches Go.

| Item | Go | godart |
|---|---|---|
| `Sscan`, `Sscanln`, `Sscanf` | Present | Equal |
| `Fscan`, `Fscanln`, `Fscanf` | Present | Equal |
| `Scan`, `Scanln`, `Scanf` | Read `os.Stdin` | Equal. The reader calls the runtime, and not `os`. Read below. |
| `ScanState` and `Scanner` | Present | Equal |
| The verbs `%v %d %b %o %x %X %s %q %c %t %e %f %g` | Present | Equal |
| A width, for example `%3d` | Present | Equal |
| `%p` and a complex number | Present | Not present |
| The error of a broken input | Present | Equal, with the text of Go |

The scan family reads standard input through `syscall/dart`. An import of `os`
would put the whole file access and the clock of `time` into each program that
reads `fmt`, and the web target holds neither. A web build reads 0 bytes of
standard input, thus each scan of that input gives `io.EOF`.

`fmt` holds `Formatter`, `State`, and `FormatString` since 2026-09-02.
`Sprintf` calls the method `Format` of a value that holds 1, and the method
then owns the width, the precision, and each flag.
`testdata/programs/fmtformat` proves it.

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
| `FileInfo`, `DirEntry`, `FileMode`, `PathError` | Alias of the type of `io/fs` | Equal |
| `FileInfo.ModTime` | Gives a `time.Time` | Equal. `ModNanos` also gives the count of nanoseconds. |
| `FileInfo.Mode` | The permission bits of the system | A directory gives `0755`, and each other file gives `0644`. Dart names no permission bit. |
| `FileInfo.Sys` | The value of the system | Always nil |
| `ReadDir`, `File.ReadDir` | Gives a slice of `DirEntry` | Equal. `ReadNames` also gives the names. |
| `Lstat` | Follows no symbolic link | Equal. The size of a link is the length of its target, and the time of a link is 0, because `dart:io` gives no metadata of a link itself. |
| `Stdin`, `Stdout`, `Stderr` | Present | Equal. A read of `Stdin` stops at a newline. |
| `Getenv`, `LookupEnv`, `Environ` | Present | Equal |
| `Setenv`, `Unsetenv` | Writes the environment of the process | Writes a map of the overlay. The Dart VM gives no write to the environment of the process, and the subset starts no other process. |
| `Getwd`, `Chdir`, `TempDir` | Present | Equal |
| `CreateTemp`, `MkdirTemp` | A random name | Equal, with a counter and the clock in place of the random part. |
| `File.ReadAt`, `File.WriteAt` | Present | Equal. Each call reads the position first and writes it again. |
| `OpenFile` | Each flag of the system | 4 modes: a read, a write with `O_TRUNC`, a write at the end with `O_APPEND`, and a read and a write with no truncation. Each write mode also reads, exactly as `os.Create` of Go does. Dart holds no mode that reads and writes and stands at the start, thus the last one moves the position back. |
| `O_CREATE` | Fails when the file does not exist and the flag stays out | Each mode of a write makes the file. Dart holds no open that fails when the file does not exist. |
| `Args`, `Exit` | Present | Equal. `Args[0]` is the text `gart`. |
| `IsNotExist`, `ErrNotExist`, `PathError` | Present | Equal. `errors.Is` finds each sentinel. |
| The text of a `PathError` | The message of the system call, for example `no such file or directory` | The text of the sentinel, for example `file does not exist`. `causeError` maps the message of Dart to the sentinel of Go, thus the test of the error works and the text differs. |
| `FileMode` | Holds the permission bits | The type reads each bit. The argument of `Mkdir`, of `WriteFile`, and of `OpenFile` reaches no operation. |
| `Symlink` | Present | Equal. The error of a failure holds a different text. |
| `Chmod`, `Chown`, `Link` | Present | Not present. Dart gives no such call. |
| `os.Pipe`, `os.StartProcess` | Present | Not present |
| `os.EOF` | Not present | Present, and it names `io.EOF`. A program that reads the name compiles with gart and fails with the go command. |

`fmt` holds `Stringer` and `GoStringer`. The formatter reads the method table of
the type descriptor. Thus a value with a `String` method writes that text for
the verb `%v` and for the verb `%s`. A value with a `GoString` method writes
that text for the verb `%#v`, and the verb reads neither `String` nor `Error`,
exactly as in Go.

`fmt` grows `Fprint`, `Fprintln`, and `Fprintf`. The parameter holds the
interface `fmt.Writer` and not `io.Writer`, because an import of `io` would make
a cycle. A caller never names the type, thus a Go source compiles the same way
with the go command and with gart.

## The process access

`os/exec` starts a child over `Process` of `dart:io`, and the runtime holds
the child in a table by its process id. Decision 212 holds the shape.

| Item | Go | godart |
|---|---|---|
| `Command`, `Run`, `Start`, `Wait`, `Output`, `CombinedOutput` | Present | Equal |
| `StdinPipe`, `StdoutPipe`, `StderrPipe` | The pipe works before `Start` | A write or a read before `Start` gives an error. The pipe of the system opens at the start. |
| `Cmd.Dir`, `Cmd.Env` | Present | Equal |
| `Args[0]` differs from `Path` | The child sees `Args[0]` | The child sees `Path`. Dart takes the program and the arguments apart. |
| `LookPath` | Reads the execute bit | Finds a plain file. Dart names no permission bit. |
| `ExitError`, `os.ProcessState` | Present | The state holds the exit code and the signal only. `UserTime`, `SystemTime`, and `SysUsage` stay out, because `dart:io` reports no CPU time of a child. |
| `os.ProcessState.Sys`, `exec.ExitError.Sys` | A `syscall.WaitStatus` | Equal since 2026-09-04. The overlay writes the exit code and the number of the signal into the word of a Unix wait state. `CoreDump` and `Stopped` are always false, and `StopSignal` always gives -1, because `dart:io` reports no such state. |
| `os.Process.WithHandle` | A handle of the system on Linux 5.4 and on Windows | Gives `ErrNoHandle` and runs the function no time. `dart:io` names a child by its process id. Go gives the same error on a platform with no handle. |
| `os.Process.Kill`, `Wait`, `Release`, `Signal` | Present | Equal. `Signal` reads the 12 signal numbers that macOS and Linux agree on. |
| `CommandContext`, `Cmd.Cancel`, `Cmd.WaitDelay` | Present | Equal. |
| `Cmd.SysProcAttr`, `Cmd.ExtraFiles`, a process group | Present | Out. `dart:io` gives the child 3 streams, and it honors no attribute of the platform. |
| The error of `Start` for a bad program | `fs.PathError` inside `exec.Error` | The text of the Dart exception |

## The log package and the filepath package

`log` holds each function and each method of Go. `Lshortfile` names the base
name of the file and the line, exactly as Go does. `Llongfile` names the file
the way the source map holds it: the package path and the base name. Go names
the file the way the build machine holds it, thus the text before the base
name differs.

`path/filepath` holds the path operations. The separator is always `/`. Go
selects the separator with a build tag of the operating system, and the emitted
Dart names the system only at run time.

| Item | Go | godart |
|---|---|---|
| `Clean`, `Join`, `Split`, `Dir`, `Base`, `Ext` | Present | Equal |
| `IsAbs`, `IsLocal`, `ToSlash`, `FromSlash`, `SplitList`, `VolumeName` | Present | Equal |
| `Match`, `Abs`, `Rel`, `Glob`, `ErrBadPattern` | Present | Equal |
| `Separator`, `ListSeparator` | The separator of the system | Always `/` and `:` |
| `Walk`, `WalkDir`, `SkipDir`, `SkipAll` | Present | Equal |
| `Localize`, `HasPrefix` | Present | Equal |
| `EvalSymlinks` | Present | Present. The table of the runtime above holds the 2 gaps. |

## The web target

The compiler adds no web layer of its own. A web build calls the Dart
compiler, per the web decision of 2026-09-01. `dart compile wasm bin/main.dart`
compiles a pure program, and the output runs under node through the loader
that the Dart compiler writes. `TestWasm` of the driver proves it.

`dart compile js` also compiles a pure program since 2026-09-01. The Go
integer of the emitted code is the type GoInt, and the JS build selects the
pair form of GopherJS, thus the 64-bit model holds there too. `TestJS` of the
driver proves it under node, and `docs/js-integers.md` holds the build and
the gaps of the JS target: a mixed numeric type set, the 53-bit reads of
reflect and json, and the rounded nanosecond clock.

Since 2026-09-03 a program that reaches a file or a socket also compiles. The
4 web files of the runtime declare each name of their native form, and each
unsupported operation gives the cause of Go under `GOOS=js`. Read decision
251. `TestWebIO` of the driver runs the program of `testdata/web/webio` under
the 2 web compilers, and it reads the exact text of each cause.

A web program also reaches a real server since 2026-09-03. `net/http` takes
the fetch API of the host when `runtime.GOOS` gives `js`, and the body of an
answer rides the `ReadableStream` of that API. Read decision 252. A Flutter
application also builds for the web since the same day. `flutter build web`
writes `build/web` for each of the 3 examples, and `apilist` reaches its API
through that client.
`TestWebFetch` of the driver runs the program of `testdata/web/webfetch`
against a real server with the go command and with the 2 web compilers, and
the 3 outputs are equal. A socket and a server stay out of reach, because a
browser holds neither.

| Item | Go | godart |
|---|---|---|
| A pure program on wasm | Compiles with GOOS=js GOARCH=wasm | Compiles with `dart compile wasm`. The 64-bit integer model holds. |
| A pure program on dart2js | Compiles with GOOS=js | Compiles with `dart compile js`. The pair form of GoInt keeps the 64-bit model. |
| A program that reaches a file or a socket | Compiles, and the call fails at run time | Equal. The compile passes, and the call gives an error. |
| `os.Open`, and each other file operation | `open x: function not implemented` in a browser | Equal |
| `net.Dial`, `http.Get`, and each other socket operation | An error of the host | `operation not supported`, inside a `net.OpError` |
| `os.Getenv` and `os.Environ` on the web | Empty | Equal |
| `os.Getwd` on the web | `/` | Equal |
| `os.Getpid` on the web | 0 | Equal |
| `os.TempDir` on the web | `/tmp` | Equal |
| `runtime.GOOS` on the web | `js` | Equal |
| `print` and `fmt` on the web | Standard output and standard error | The console of the host, 1 line for each call, thus a write without a newline waits in a buffer |
| `os.Exit` on the web | Ends the process | Throws an error of the host, thus a status outside 0 ends node with a failure |
| An HTTP request from a browser | `fetch`, through the js syscall layer | Equal. `net/http` takes the fetch API when `runtime.GOOS` gives `js`. |
| `CheckRedirect` of a `Client` on the web | Never runs, because the host follows the redirect | Equal |
| `Response.Proto` on the web | Empty, because the host picks the protocol | Equal |
| A gzip answer on the web | The host decodes it, and `Uncompressed` marks it | Equal |
| Each field of `Transport` on the web | Without effect, because the host owns the connection | Equal. The deadline of the request is the exception. |
| A TCP or a UDP socket in a browser | Not present | Equal |
| An HTTP server in a browser | Not present | Equal |

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
| `Conn` | 8 methods | Equal. `LocalAddr` and `RemoteAddr` read the socket of the runtime, and each 1 gives a `*TCPAddr` or a `*UnixAddr`. |
| `PacketConn` | 7 methods | `ReadFrom`, `WriteTo`, `Close`, `LocalAddr`, and the 3 deadline methods. |
| `Listener` | `Accept`, `Close`, `Addr` | Equal |
| `Addr` | `Network` and `String` | Equal |
| `Error` | `Timeout` and `Temporary` | Equal. A deadline that passed reports true for each one. |
| `Dialer` | 12 fields, 6 methods | `Timeout`, `Deadline`, `Cancel`, and `Resolver`, with `Dial`, `DialContext`, `DialTCP`, `DialUDP`, and `DialUnix`. The dial runs on a goroutine, and the select takes the first of the dial, the context, and the cancel channel. `DialIP` and the 8 fields of a socket option stay out. Read `docs/support.md`. |
| `DialTimeout` | Present | Equal. It reads a `Dialer` with the timeout. |
| `TCPAddr`, `UDPAddr`, `IPAddr`, `UnixAddr`, `IP`, `IPMask`, `IPNet` | Present | Equal. `ParseCIDR`, `CIDRMask`, `IPv4Mask`, `IP.Mask`, and `IP.DefaultMask` are present. |
| `ResolveTCPAddr`, `ResolveUDPAddr`, `ResolveIPAddr`, `ResolveUnixAddr` | Present, with a name lookup | Present. A host that is not an IP address gives a parse error. A lookup gives up control, and a Resolve function of Go holds no context, thus `Resolver.LookupIP` reads a name. |
| `Resolver`, `LookupHost`, `LookupIP`, `Interfaces` | Present | Present since 2026-09-02. The overlay holds a resolver of its own. Read the section "The Lookup family of net". |
| `IP.IsPrivate`, `IsMulticast`, `IsInterfaceLocalMulticast`, `IsLinkLocalMulticast`, `IsLinkLocalUnicast`, `IsGlobalUnicast` | Present | Equal since 2026-09-04. Each 1 reads the bytes of the address alone. |
| `IPv4bcast`, `IPv4allsys`, `IPv4allrouter`, `IPv4zero`, `IPv6zero`, `IPv6unspecified`, `IPv6loopback`, and the 3 IPv6 groups | Present | Equal |
| `InvalidAddrError`, `ParseError.Timeout`, `ParseError.Temporary` | Present | Equal |
| `TCPListener`, `ListenTCP`, `UnixListener`, `ListenUnix` | Present | Present, less `File`, `SyscallConn`, `SetDeadline`, and `SetUnlinkOnClose`. `Listen` gives a `*TCPListener` for the network tcp and a `*UnixListener` for the network unix, exactly as in Go. |
| `UnixConn.ReadFrom`, `UnixConn.ReadFromUnix` | The name of the source of a datagram | Present. Dart opens a stream socket of the Unix domain alone, and the system reports an empty name for the source of such a socket, thus the address is always nil. Go behaves the same way. |
| `UnixConn.WriteTo`, `UnixConn.WriteToUnix` | A datagram of the Unix domain | Not present. Read `docs/support.md`. |
| `TCPConn` | 20 methods | 10 methods: the 8 of `Conn`, plus `ReadFrom` and `WriteTo`. `Dial`, `Dialer.DialContext`, and `Listener.Accept` each give 1 for the network `tcp`. |
| `TCPConn.CloseRead`, `CloseWrite` | A half close | Not present. `dart.ConnClose` maps to `Socket.destroy`, which ends the 2 directions. |
| `TCPConn.SetNoDelay`, `SetLinger`, `SetKeepAlive`, `SetKeepAlivePeriod`, `SetKeepAliveConfig`, `SetReadBuffer`, `SetWriteBuffer` | Each writes a socket option | Not present. `syscall/dart` holds no option intrinsic. |
| `TCPConn.SyscallConn`, `TCPConn.File`, `TCPConn.MultipathTCP` | Present | Not present. `dart:io` owns the socket and gives no file descriptor. |
| `DialTCP` | A local address and a remote address | Present. A local address that is not nil gives an error, because Dart binds no local address of a dial. |
| `ListenConfig` | 5 fields, with `Listen` and `ListenPacket` | `Listen` and `ListenPacket`. The bind runs on a goroutine, and the select takes the first of the bind and the context. `Control`, `KeepAlive`, `KeepAliveConfig`, and the 2 MPTCP names stay out, because each 1 needs a socket option or a `syscall.RawConn`. |
| `IPConn`, `DialIP`, `ListenIP` | A raw IP socket | Not present. `dart:io` opens no `SOCK_RAW` socket, and `ReadMsgIP` needs a control message that Dart never exposes. |
| An empty host of `Dial` | The local system | `127.0.0.1` |
| An empty host of `Listen` | Each interface | `0.0.0.0` |
| `SplitHostPort` | Accepts an IPv6 host in brackets | Only a host with no bracket. |
| `JoinHostPort` | Adds brackets for an IPv6 host | It joins the 2 parts with a colon. |
| TLS | `crypto/tls` | Present through the overlay of `crypto/tls`. `SecureSocket` and `SecureServerSocket` of Dart hold the whole handshake. |
| A deadline that a read already waits on | Ends that read | Reaches the next read only. The runtime applies the limit at the start of an operation. |
| A write that a deadline ended | Writes nothing | The bytes stay in the buffer of the socket. |
| A connected UDP socket | The system drops a datagram of another source | The runtime drops it. Dart gives no connect of a datagram socket. |
| The text of a failure | `dial tcp 127.0.0.1:0: connect: ...` | `dial tcp <address>: ` and the message of the Dart `SocketException`. |
| The text of a deadline | `read tcp <local>-><remote>: i/o timeout` | `read tcp: i/o timeout` |

`testdata/programs/n2net` proves the 6 address tests, each well-known address,
`ResolveIPAddr`, the 2 listener types, and the 3 dial methods of a `Dialer`.

## The HTTP access

The real `net/http` of Go holds 53000 lines. It reads a vendored HTTP/2 stack, a
cookie jar, a file transport, `mime`, `net/textproto`, and `compress/gzip`. The
overlay holds an HTTP/1.1 client and an HTTP/1.1 server over the `net` package,
with about 1500 lines. HTTP/2 rides the Dart package `http2` behind ALPN, and
decision 213 holds the shape: the transport offers h2 on an https dial as Go
does, 1 session carries each request of a host, and the server answers each
stream on a goroutine of its own. `httptest.Server` takes `EnableHTTP2`, as in
Go.

| Item | Go | godart |
|---|---|---|
| `Get`, `Head`, `Post`, `PostForm`, `Client.Do` | Present | Equal |
| `NewRequest`, `NewRequestWithContext` | Present | Equal |
| `Header` with `Get`, `Set`, `Add`, `Del`, `Values`, `Clone`, `Write` | Present | Equal |
| A redirect | Follows up to 10 | Equal. `CheckRedirect` and `ErrUseLastResponse` work. |
| `Client.Timeout` | Present | Present. It reaches the deadline of the connection. |
| `Handler`, `HandlerFunc`, `ResponseWriter`, `Flusher` | Present | Equal |
| `ServeMux`, `Handle`, `HandleFunc`, `StripPrefix` | Present | Equal. The routing tree, the conflict index, and the pattern reader of Go 1.22 each read the real Go source, thus the method, the `{id}` wildcard, the `{$}` anchor, the trailing slash redirect, and the status 405 each work. `Request.PathValue`, `SetPathValue`, and `Pattern` read. |
| `Server`, `ListenAndServe`, `Serve`, `Close`, `Shutdown` | Present | Equal. The server tracks each listener and each connection, thus `Shutdown` waits for each connection to reach the idle state. The 4 timeouts, `MaxHeaderBytes`, `MaxHeaderValueCount`, `ConnState`, `BaseContext`, and `ConnContext` each read. |
| `Error`, `NotFound`, `Redirect`, `StatusText` | Present | Equal |
| The cookie family | Present | Equal. `cookie.go` reads the real Go source with 1 change. `Cookie.Valid` cannot reject an `Expires` below the year 1601, because the `Time` of the overlay names no year below 1678. |
| A chunked body | Present | Equal, for a read and for a write. |
| Keep alive | Present | Present. The client holds a pool of 2 connections for each host. |
| A Close of a body that stands before its end | The connection stays open when the read loop reaches the end first | The connection ends. |
| HTTP/2 | The vendored stack of Go | Present over the Dart package http2, for the client and the server, behind ALPN. The request body reaches the wire as 1 piece, a push of the server and a trailer stay out, and `Request.TLS` stays nil. |
| `Accept-Encoding: gzip` | The client adds it and reads the answer | Equal. `DisableCompression` stops it, and `Response.Uncompressed` marks the answer. |
| The type of a body that the handler does not name | `DetectContentType` reads the first 512 bytes | `text/plain; charset=utf-8` for each body |
| A body that is larger than the buffer of the server | The chunked form | The overlay names the length of each body that a handler writes with no Flush. |
| `Request.RemoteAddr` | The address of the peer | Equal. The server reads `Conn.RemoteAddr` of the accepted connection. |
| `Transport` | A pool, a proxy, and a set of timeouts | Equal, less `TLSHandshakeTimeout` and `OnProxyConnectResponse`. `tls.Dial` of the overlay opens the socket and runs the handshake in 1 call, thus no clock measures the handshake alone. `Dial` and `DialContext` reach no https request, because the secure socket of Dart takes no socket that stands open. |
| `httptest` | Present | `NewRecorder`, `NewRequest`, `NewServer`, `NewUnstartedServer`, `NewTLSServer`, `StartTLS`, `Server.Client`, and `Server.Close`. `CloseClientConnections` and the wait of `Close` for each request that stands open stay out. `StartTLS` binds a fresh port, because the secure socket of Dart wraps no open listener, and the client of a TLS server drops the certificate check, because the surface holds no root pool. |
| `CookieJar` and `net/http/cookiejar` | Present | Equal. `Client.Jar` reads a jar before each request, and it writes each cookie of the answer. |
| `Transport.Proxy`, `ProxyURL`, `ProxyFromEnvironment` | Present | Equal. An http request holds the whole URL in the request line, and an https request rides a CONNECT tunnel. `socks5` stays out. |
| `Request.ParseForm`, `FormValue`, `PostFormValue` | Present | Equal |
| `Request.ParseMultipartForm`, `FormFile`, `MultipartReader` | Present | Equal, with `mime/multipart` |
| `FileServer`, `FileServerFS`, `FS`, `Dir`, `ServeFile`, `ServeFileFS`, `ServeContent`, `NewFileTransport` | Present | Equal |
| `TimeoutHandler`, `MaxBytesReader`, `MaxBytesHandler`, `RedirectHandler`, `AllowQuerySemicolons`, `NoBody`, and the 13 `Err` variables | Present | Equal |
| `ReadResponse`, `ParseHTTPVersion`, `Header.WriteSubset`, `Request.Write`, `Request.WriteProxy`, `Response.Write` | Present | Equal. The transport writes the request line, `Host`, `User-Agent`, the length, and then each other field in order, thus the bytes match. |
| A body of unknown length | The chunked form | Equal. `NewRequest` leaves `ContentLength` at 0, and a body next to a 0 takes the chunked form. |
| A redirect of the status 307 or 308 | Keeps the method and the body | Equal. The client reads `GetBody` and it writes `Request.Response`. |
| `Request.TLS`, `Response.TLS` | The whole state of the connection | Present since 2026-09-03. The `tls.ConnectionState` of the overlay holds `NegotiatedProtocol`, `ServerName`, and `HandshakeComplete`, thus each other member reads its zero value. |
| `Request.TransferEncoding`, `Response.TransferEncoding` | Each encoding of the body | Present. The overlay writes the 1 element `chunked`, or nothing. |
| `Request.Trailer` | The fields that follow the body of a request | Present in the shape, and always nil. A request that declares a `Trailer` field gives the status 501, thus no trailer is lost with no report. |
| `Response.Trailer` | The fields that follow the body of an answer | Present in the shape, and always nil. The transport of the overlay reads no trailer of an answer. |
| `TrailerPrefix` | Names a field of the answer that follows the body | Equal since 2026-09-04. The answer takes the chunked form, and each declared name and each name with the prefix follows the last chunk. An answer of HTTP/1.0 writes no trailer, exactly as in Go. |
| `CloseNotifier` | The `ResponseWriter` of Go holds it | Present as a type. No `ResponseWriter` of the overlay holds it, thus an assertion gives false and the handler must read that result. The runtime reports a close of the peer at the next read alone. Go marks the interface deprecated. |
| `Pusher`, `PushOptions` | The `ResponseWriter` of HTTP/2 holds `Pusher` | Present as a type. The HTTP/2 session of the runtime sends no push frame, thus an assertion gives false. |
| `Request.Cancel` | Ends a request that stands open | Not present. The transport ends a request through a deadline of the connection, and Go marks the field deprecated. |
| `Protocols`, with the 6 accessors and `String` | Present | Equal |
| `Server.Protocols`, `Transport.Protocols` | Present | Present since 2026-09-04. A nil field holds HTTP1 and HTTP2, and the set names the list of ALPN. A set that holds `UnencryptedHTTP2` gives an error, because no path of the runtime carries HTTP/2 over a plain socket. |
| `ListenAndServeTLS`, `Server.ListenAndServeTLS`, `Server.TLSConfig` | Present | Present since 2026-09-04. An empty `Addr` names the port 443, and the 2 file names hold the PEM chain and the PEM key. `tls.Listen` of the overlay opens the secure listener. |
| `ServeTLS`, `Server.ServeTLS` | Wrap a listener that stands open | Not present. The secure socket of Dart takes over no socket that another call opened. |
| `Server.TLSNextProto`, `Transport.TLSNextProto` | Name 1 handler for each protocol of ALPN | Present since 2026-09-04. The protocol of the map joins the list of ALPN, and a connection of that protocol reaches the handler. |
| `CrossOriginProtection`, `NewCrossOriginProtection` | Present | Equal. The check reads `Sec-Fetch-Site`, `Origin`, and `Host`, and it needs no socket. |
| `HTTP2Config`, `Server.HTTP2`, `Transport.HTTP2` | Tune the HTTP/2 session | Not present. The session lives in the Dart runtime library, and it exposes no window, no frame size, and no ping. |
| `ClientConn`, `Transport.NewClientConn` | 1 connection of the pool, with a count of the free streams | Not present. The pool of the overlay owns each connection, and it reports no such count. |
| `ResponseController`, `NewResponseController` | 5 methods | Equal. `Flush`, `Hijack`, `SetReadDeadline`, `SetWriteDeadline`, and `EnableFullDuplex` each reach the connection of the HTTP/1.1 server. |
| `Hijacker`, `ErrHijacked` | Present | Present for the HTTP/1.1 server. The answer of an HTTP/2 stream carries no connection of its own, thus it holds neither. |
| `ResponseController.EnableFullDuplex` | Lets a handler read the body while it writes | Present, and it changes nothing. The overlay server reads the rest of a body after the handler ends, thus each handler already holds the 2 directions. |

`testdata/programs/httpuse` runs a server and a client in 1 program, and the
output of `go run` and of the Dart build stays equal.
`testdata/programs/httpkeep` counts the connections of a listener, and it proves
that 5 requests in a row share 1 connection. `testdata/programs/h2serve` runs
an HTTP/2 server and its client, and the 2 sides print `HTTP/2.0`.
`testdata/programs/httpclient` runs a cookie jar, a multipart form, and a proxy
that carries a plain request and a CONNECT tunnel.
`testdata/programs/n2httptls` runs `Server.ListenAndServeTLS`, the `Dialer` of
`crypto/tls`, the cross origin protection, and a trailer of an answer.

An https request through a proxy takes 1 name that Go does not hold. Go writes
the CONNECT request itself and then wraps the connection with `tls.Client`. The
secure socket of Dart takes over a socket that no read touched, thus the runtime
holds the whole exchange, and `crypto/tls.DialProxy` names it. `tls.Client` and
`tls.Server` stay out for the same cause.

## The clock and the timer

`time` reads the clock of `dart:io` and the scheduler of the runtime. A value of
`Time` holds a count of nanoseconds after the start of the Unix epoch and a
location. `UTC` and `Local` are the 2 built locations, and `Local` reads the
offset and the name of the zone from Dart at each instant. `LoadLocation`
builds each other location from the zone database.

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
| `Time.String` | `2006-01-02 15:04:05.999999999 -0700 MST` | Equal, with no value of the monotonic clock. Read below. |
| `Time.Format` | Present | Equal. The scanner holds each chunk of a layout, and the 19 layout constants are present. |
| `Date`, `Month`, `Weekday`, and the name of each | Present | Equal |
| `Time.In`, `Time.Location` | Present | Present |
| `UTC`, `Local` | 2 of the locations of the zone database | Equal |
| `LoadLocation` | The variable ZONEINFO, the zone directories of the system, the embedded database of `time/tzdata`, and the zip file of the Go tree | Equal, except the zip file of the Go tree. The Dart output holds no Go tree. `os.Setenv("ZONEINFO", ...)` reaches only the map of the os overlay, thus the variable must come from the start of the process. |
| `LoadLocationFromTZData` | Present | Equal. The parser is the parser of Go. |
| `time/tzdata` | The embedded IANA database | Equal. The import grows the emitted program by the zip, about 450 KB. |
| A date after the last transition of a zone file | The extend rule of the file | Equal. The tzset chain of Go reads the rule. |
| `FixedZone` | Present | Present. `Parse` builds one for an offset that the location does not hold. |
| `Time.Year`, `Month`, `Day`, `Date`, `Hour`, `Minute`, `Second`, `Clock`, `Nanosecond`, `Weekday`, `YearDay` | Present | Equal |
| `Time.UTC`, `Time.Local`, `Time.Equal` | Present | Equal |
| `Time.Parse`, `ParseInLocation`, `ParseError` | Present | Equal. Read below. |
| `Time.MarshalJSON`, `MarshalText`, `AppendText`, and the 2 read methods | RFC 3339 with the nanosecond | Equal |
| `Time.MarshalBinary`, `AppendBinary`, `UnmarshalBinary` | The versioned form of Go | Equal. The bytes of the 2 runtimes match. |
| `Time.MarshalJSON` outside the year range 0 to 9999 | Gives an error | The range of a Time cannot reach it, thus the method reports no error. |
| `ParseDuration`, `Duration.Truncate`, `Duration.Round` | Present | Equal |
| `Time.Truncate`, `Time.Round` | Round against January 1 of the year 1 | Equal. The count of nanoseconds of that day does not fit an int64, thus the general case builds a value of 128 bits. |
| The zero value of `Time` | January 1 of the year 1 | The start of the Unix epoch. A Time holds 1 count of nanoseconds, thus the range is the year 1678 to the year 2262. |

`Parse` reads each chunk of the layout that `Format` writes, and it gives a
`*ParseError` with the text of Go. A layout that names no year gives the year 0,
and that year stands outside the range of the overlay. Such a value wraps. Give
the layout a year, or read the clock of the result only.

A Go `Time` of `time.Now` also holds a value of the monotonic clock, and
`String` writes it as ` m=+1.234567890`. A `Time` of the overlay holds no such
value, thus the text ends with the name of the zone.

`time.Date` with the location `Local` reads the offset of the system 2 times.
The Dart runtime names the offset of an instant, and the argument of `Date` is a
wall clock value. The first lookup reads that wall clock as if it were UTC, and
the second one reads the instant that the first offset gives. A wall clock value
inside 1 offset of a transition of the zone can take the offset of the other
side.

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
| `Int32`, `Int64`, `Uint32`, `Uint64`, `Uintptr`, `Bool` | Present | Equal. Each one holds `Load`, `Store`, `Swap`, and `CompareAndSwap`, and each integer type holds `Add`, `And`, and `Or`. |
| A `Store` of a second concrete type | Panics | It writes the value. |
| `Pointer[T]` | Present | Equal |
| The function form, for example `AddInt64` | Present | Equal for `Add`, `And`, `Or`, `Load`, `Store`, `Swap`, and `CompareAndSwap`. |
| `LoadPointer`, `StorePointer`, `SwapPointer`, `CompareAndSwapPointer` | Present | Not present. Each one takes an `unsafe.Pointer`, which stands outside the subset. `Pointer[T]` holds the same use. |

## The root of a directory

`os.Root` holds a directory, and each operation of it stays inside that
directory.

Go opens the directory 1 time and reaches each name under it with `openat`,
thus no path can leave the tree. `dart:io` holds no such call. The overlay
resolves each symbolic link of the path with `resolveSymbolicLinks` and it
compares the result with the root.

| Item | Go | godart |
|---|---|---|
| A path that holds `..` | Rejected | Rejected |
| A symbolic link that points outside the root | Rejected | Rejected |
| A rename of a directory of the path between the check and the operation | Rejected, because the handle of the directory holds the tree | **Not rejected.** The check is not atomic, thus such a race can reach a name outside the root. |
| `Chmod`, `Chown`, `Chtimes`, `Lchown`, `Link` | Present | Not present. `dart:io` holds no such call, and `os` holds no such function either. |

Do not read `Root` as a security boundary against a writer that can rename a
directory of the path while the program runs.

## The signal access

`os/signal` reads `ProcessSignal.watch` of dart:io. Each watch runs on the
event loop, thus a signal reaches the program between 2 goroutine steps, as it
does in Go.

| Item | Go | godart |
|---|---|---|
| `Notify`, `Stop`, `Reset`, `Ignore`, `Ignored`, `NotifyContext` | Present | Equal |
| The signals that a program can catch | Each signal of the platform | `SIGHUP`, `SIGINT`, `SIGQUIT`, `SIGABRT`, `SIGALRM`, and `SIGTERM`. dart:io watches no other signal, thus `Notify` of another signal keeps the channel and no event reaches it. |
| The package `syscall` | The whole system call interface | The `Signal` type and the 12 signal numbers that macOS and Linux agree on. A signal whose number differs between the 2, for example `SIGUSR1`, stays out. |
| `(*os.Process).Signal` | Present | Equal. It reads `Process.killPid` of dart:io, thus it reaches any process and not only a child. |
| `Reset` after `Ignore` | `Ignored` still reports true, because the disposition of the system stays | Equal |

## The Pull functions of iter

Go runs `Pull` on a coroutine of its runtime. The subset holds no coroutine,
thus the sequence runs on a goroutine of its own. 2 channels with no buffer hand
the control back and forth, and the 2 goroutines never run at the same time.

| Item | Go | godart |
|---|---|---|
| `Pull`, `Pull2` | Present | Equal |
| A `stop` before the end of the sequence | The sequence ends | Equal |
| A `next` after `stop` | The zero value and false | Equal |
| `yield` after a `stop` | Gives false | Equal |
| A panic of the sequence | `next` panics with the same value | Equal. A deferred call of the caller can recover it. |
| `runtime.Goexit` inside the sequence | `next` panics | Out of the subset |

The `yield` of `Pull` gives up control, thus each function that a sequence calls
can block. Stage 3 marks each `iter.Seq` closure of a program as such, and the
output of a program that imports `iter` grows. `slices` and `maps` each import
`iter`.

## A range loop over a function

`for k, v := range seq` accepts a function of the form `func(yield func(K, V)
bool)`. The compiler writes the body as a Dart closure and it passes the closure
to the function:

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

A `return` inside the body writes each value into a temporary variable, it sets
a flag, and it gives false to the sequence. The code after the call of the
sequence reads the flag. A nested sequence carries the flag out 1 level for each
call.

A labelled `break` or `continue` that leaves the body writes the flag of its
target and gives false to the sequence. The code after the call reads the flag.
It gives false again while another such body holds it, and it runs the real
branch at the level of the target. The flag takes 0 again at that point, because
the target construct runs its next pass.

## The core type of a type parameter

Go gives a type parameter a core type when each term of its type set shares 1
underlying type. `S ~[]E` holds the core type `[]E`, thus each operation of a
slice applies to a value of the type S.

The compiler reads the core type for each operation. The operations are an
index, a slice expression, a range loop, `len`, `cap`, `make`, `append`, `copy`,
`delete`, a nil comparison, and a field read.

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
| An arithmetic operator and a conversion | Rejected by stage 2 |

A constraint whose terms give 1 Dart type keeps each operator. `int | int64`
gives `int` 2 times, thus `+` and `<` work directly.

## A type parameter of integer types

A constraint that lists only integer types gives 1 Dart type, because each
integer type of Go maps to the Dart type `int`. Such a parameter holds no core
type, and it still keeps each arithmetic operator.

The width differs per type argument, thus the dictionary holds a mask.
`Dict.mask` keeps the value inside the width, and the compiler writes it after
each arithmetic operation and after each conversion to the parameter. The mask
of a type that fills the width of a Dart int is the identity.

| Type set | godart |
|---|---|
| Each term reads the signed operators (`~int8`, `~uint16`, `~int64`) | Each operator |
| Each term is unsigned and 64 bits (`~uint`, `~uint64`, `~uintptr`) | Each operator, with the unsigned helpers |
| The 2 forms together (`~int \| ~uint64`) | `/`, `%`, `>>`, a conversion to a float, and the builtin `print` are rejected by stage 2 |

Go reads the bit pattern of an unsigned type of 64 bits as unsigned, and Dart
reads the same pattern as a negative number. Thus a division, a remainder, and a
right shift take 2 Dart expressions, and no type argument selects one at compile
time. `math/rand/v2` holds such a type set in `N` and in `Rand.N`, and neither
one reads those 3 operators.

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
holds a variable for the description of each other named type. An instance
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
field type. Thus 2 equal Go types give 1 class. The class holds the fields,
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

Go gives an unexported field the identity of the package that declares it. Dart
holds a name that starts with an underscore private to 1 library. Thus the class
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
| The source position of a log line | `file.go:12: text` | Equal. The harness reads the stack through the source map. |
| The position of a log line of a helper that calls `t.Helper` | The position of the caller | Equal. |
| The position of a deferred log line after `Fatal` | The position of the runtime of Go, for example `panic.go:694` | No position. |
| `t.Parallel` | Runs the tests in parallel | Equal. Each test runs on a goroutine of its own, and the group wakes each paused test after the pass of the tests that are not parallel. The order of the report is the order of the ends, and the scheduler of 1 isolate makes it stable. |
| `t.FailNow`, `t.SkipNow` | `runtime.Goexit` | A throw. Each deferred function still runs, and the subset has no `recover`. |
| `-run` | A regular expression | Equal. A `\|` separates 2 alternatives, and a `/` separates the expression of each element of the name. |
| A filter that keeps no test | `testing: warning: no tests to run`, and ` [no tests to run]` after the package | Equal. The driver writes the mark, and the harness writes the warning. |
| A duplicate subtest name | The second name takes `#01` | The 2 names stay equal |
| `TestMain` | Runs in place of the entry point | Equal. `testing.M` holds `Run`. |
| A benchmark | `go test` with no -bench flag runs none | Equal. `gart test -bench <text>` runs them. |
| `b.RunParallel` | Measures a parallel throughput | Stops the benchmark with an error. All Go code runs in 1 isolate. |
| `b.ReportAllocs` | Adds allocs/op and B/op | Does nothing. The Dart VM gives no count of allocations for 1 operation. |
| `testing.AllocsPerRun` | Gives the average count of allocations | Skips the test with the cause. A result of 0 would make such a test pass. |
| `b.Loop` | Runs the body 1 time and raises the count itself | Equal. The harness calls a body that reads `Loop` only 1 time. |
| The clock of a benchmark | A monotonic nanosecond clock | `Stopwatch` of Dart. It also ticks each nanosecond. |
| A benchmark build | The normal test binary | An ahead of time build. The warm up cost of the Dart VM hides the result. |
| A fuzz target | Runs the seed corpus | Equal. Read decision 312. |
| An example with an output comment | Runs and compares the output | Equal. Read decision 287. |
| The working directory of a test | The directory of the package | Equal. The driver runs the binary there, thus a relative path reads the same file. |
| `t.TempDir` | A new directory for the test | Equal. The cleanup of the test removes it. |
| `t.Setenv` | Writes the environment for the test | Equal. It writes the map of the `os` overlay, and the cleanup writes the value of the start again. |
| `t.Context` | A context that a cancel closes in front of the cleanups | Equal |
| `testing.TB` | Holds `Setenv` and `Context` | Equal since 2026-09-03. `TB` holds the 7 names of the real `TB`, and `T`, `B`, and `F` each give them. |
| `t.Attr` | Writes a key and a value to the test log | Present. A key with a space, or a value with a newline, reports an error and writes no line, as Go does. |
| `t.Output` | Gives a writer of the test log | Present. Each line takes the prefix of the test, as Go writes it. |
| `t.ArtifactDir`, `t.Deadline` | Read a flag of `testing.Init` | Present in the shape. `-timeout` stays out, thus `Deadline` reports no deadline. |
| `b.Chdir`, `b.Context`, `b.Setenv`, `b.TempDir`, `b.Elapsed`, `b.SetParallelism` | Present | Equal. The benchmark path now cancels its context in front of the cleanups, as the test path does. |
| `BenchmarkResult.MemAllocs`, `MemBytes`, `Extra`, `MemString` | Present | Present. Each memory number reads 0 on a measured run, because the Dart VM counts no allocation. `Extra` reads a value that the benchmark wrote. |
| `testing.Init`, `MainStart`, `RunTests`, `RunBenchmarks`, `RunExamples` | Present | Present. `Init` registers the flags of the harness, thus a `TestMain` that parses `-test.v` works. `RunExamples` compares standard output, which the runtime captures. `RunTests` and `RunBenchmarks` run the list, and Go runs nothing there. |
| An example with an output comment | `go test` runs it and compares the output | Equal. The generated entry point names the list, and `M.Run` runs each example after the tests and in front of the benchmarks. |
| `testing.Benchmark` | Gives a BenchmarkResult | Equal. The field T holds a count of nanoseconds, and not a `time.Duration`. |
| `-bench`, `-benchtime`, `-benchcount` | Present | Present. `-benchtime 100x` becomes `-benchcount 100`. |
| `-benchmem` | Present | Not present. Read the row of `b.ReportAllocs`. |
| `-short`, `-count`, `-cover` | Present | Equal. `-cover` reads `go tool cover` of the tool chain, and it rejects a package that the overlay replaces, because the rewrite reads the file of the disk. |
| `-timeout` | Present | Not present |
| A fuzz target | `go test` runs each seed of `f.Add` and each file of `testdata/fuzz` as a subtest | Equal. A seed of `f.Add` takes the name `seed#<n>`, and a corpus file takes the name of the file. |
| `f.Add`, `f.Fuzz`, and the `TB` set of `F` | Present | Equal. `F` holds a `*T`, thus each log line, each cleanup, and each skip follow the path of a test. |
| The corpus file | The versioned text form of Go | Equal. The parser holds the grammar of `internal/fuzz/encoding.go`, and it reads each of the 15 types that `f.Add` accepts, `math.Float32frombits`, and `math.Float64frombits`. |
| `-fuzz` | Generates new inputs | Not present. `gart test` holds no such flag, thus the command rejects it with `flag provided but not defined: -fuzz` and runs nothing. |
| A fuzz body that gives up control | Runs | Outside the call. `dart.CallValue` reads no scheduler, exactly as `reflect.Value.Call` does. A body that reads a channel or that sleeps thus stays outside the subset. |
| The cause of a corpus file with a bad literal | `literal value required for primitive type` | The text of `strconv`, for example `strconv.ParseInt: parsing "abc": invalid syntax`. Both reject the file and fail the target. |
| The position of the line `returned without calling F.Fuzz, F.Fail, or F.Skip` | The position of the runtime of Go, for example `fuzz.go:666` | No position. Read the row of a deferred log line. |
| `t.Parallel` inside a seed run | Runs the seeds at the same time | The cooperative order of 1 isolate. Go can put 2 `=== CONT` lines next to each other and write a `=== NAME` line in front of a log line, and gart writes each run one after the other. |
| A panic inside a test | Reports the test and ends the binary | Equal. A panic passes through the harness. |

`testing.B` holds the state of a benchmark. The search of the count of
iterations follows `go test`. The first run takes 1 iteration, and each next
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

`testing.F` holds a `*T` and the seed corpus of 1 fuzz target. `F.Fuzz` reads
the type of the body through the type descriptor of the runtime, and it reads
the corpus files of `testdata/fuzz/<name>`. It then runs the body 1 time for
each input as a subtest, and `dart.CallValue` makes the call. Read decision
312.

`gart test` also reads the real test files of a package of GOROOT. Such a
package belongs to no module, and the driver gives it the module `std`. The
command needs no flag for it:

```bash
cd $(go env GOROOT)/src/path && gart test -o /tmp/x -runtime <path> .
```

14 packages of GOROOT run each of their tests, and `TestTestsOfGoroot` proves
each one. The 14 are `path`, `container/list`, `container/heap`,
`container/ring`, `unicode`, `encoding/hex`, `encoding/ascii85`, `encoding/csv`,
`hash/crc32`, `hash/fnv`, `hash/adler32`, `maps`, `io`, and `regexp/syntax`.

A test file of GOROOT holds code outside the subset in many packages. `recover`,
`unsafe`, and the address of a byte each appear, and each one stays outside the
subset. Thus a package reaches a full run only when each of its test files stays
inside the subset.

The cause that blocks each package of the survey of 2026-08-31:

| Package | Cause |
|---|---|
| `errors` | 1 call of `recover` in `wrap_test.go`. |
| `io` | Runs. Each test passes. |
| `maps` | Runs. Each test passes. |
| `math/rand/v2` | `Type.NumMethod`, `Type.Method`, `Value.Method`, and `os.Pipe`. |
| `bytes` | `internal/testenv`, `internal/asan`, and 7 calls of `recover`. |
| `regexp` | 5 rejections in `regexp/syntax/parse.go`: 1 call of `recover` and 4 addresses of a `[]rune` field. |

### The block model

A function that can block gives up control with a plain return. It writes each
local into a frame object, and each caller on the stack does the same. Thus the
stack unwinds to the scheduler, and the scheduler runs the chain again when the
operation is ready. The body of such a function becomes a flat state machine: a
loop with a switch on a state variable.

The model holds no `Future` and no `await`. A call that gives up no control
costs 1 state assignment and 2 tests.

`testdata/tests/async` measures a call chain of 20 frames. The first 2 chains
read a buffered channel 1 time in the last frame. Thus no frame really waits,
and stage 3 still marks each frame. The third chain reads an empty channel that
a second goroutine fills, thus each frame writes a frame object and gives up
control.

| Chain of 20 frames | Go | godart |
|---|---|---|
| Synchronous | 8.5 ns/op | 26 ns/op |
| Asynchronous | 51.5 ns/op | 189 ns/op |
| An unwind and a rewind of each frame | 138 ns/op | 540 ns/op |

The `async` form of Dart cost 3900 ns/op for the second chain, because each
`await` schedules a microtask. The unwind model is a factor of 21 faster.

A frame holds each local of its function, and not only the locals that live at
the block point. A hand edit of the frame class of `BlockChain` measured the
gain of a liveness analysis on 2026-08-29:

| Form of the frame of a chain of 20 frames | Cost |
|---|---|
| Each local, the form of the compiler | 538 to 558 ns/op |
| The live locals of the state, with a branch on the state | 562 ns/op |
| No local at all, the upper bound of the gain | 532 ns/op |

The upper bound stands inside the noise of the baseline. The branch that selects
the live set costs more than the 2 field writes that it saves. Thus the liveness
analysis stays out.

The block analysis of stage 3 keeps a synchronous path for a function that
cannot block. Thus only a chain that can block pays the cost.

These rules follow from the model:

- A machine whose deferred calls can block holds the chain of those calls in 3
  states of its own. A return statement writes each result into the frame and
  jumps to the chain, and the chain runs 1 call for each pass. Such a call gives
  up control exactly as each other call does. The try block of the panic stands
  inside the loop of the machine. Thus a panic also reaches the chain, and the
  frame keeps the panic until the last call ends.
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

Gaps of the standard library surface. `docs/support.md` holds the count of
each package and the whole list:

| Item | State |
|---|---|
| `sync.RWMutex` | A read lock is equal to a write lock. The single isolate gives no parallel read. |
| `strconv.Unquote`, `strconv.UnquoteChar` | Present. The source comes from Go. |
| `strings.Title` | Not present. Go deprecated it. |
| `sort.Sort` with an interface | Present. `sort.Reverse` and the 3 typed slice types stay out. |
| `crypto/tls` | A client and a server. Read the section of the network access. |

A list of bytes reads as text for the verbs `s`, `q`, `x`, and `X`, exactly as
in Go. A byte that holds no rune of UTF-8 keeps its value through each write.
`GoString.toDartString` gives such a byte 1 code unit of the range U+DC80 to
U+DCFF, and `GoString.fromString` gives the byte again. A Dart String holds
runes only, thus the format path needs that escape.

The verb set of `fmt`: `%v`, `%+v`, `%#v`, `%T`, `%d`, `%s`, `%q`, `%t`, `%c`,
`%b`, `%o`, `%x`, `%X`, `%f`, `%e`, `%g`, and `%%`. The width and the precision
work. The flags `-`, `0`, `+`, and `#` work. The flag `#` writes the prefix of
the base in front of a whole number. With the verb `v` it writes the Go syntax
form. Read decision 315.

A Dart double holds a `float32` with the precision of 64 bits. `%v` and `%g`
write the shortest decimal text that reads back as the same 32-bit value, thus
the text holds the digits that Go writes.

## The findings of the go test directory, 2026-08-31

- A line directive reaches the source map, and `runtime.Caller` gives the
  position of the directive. The work of 2026-09-01 closed this finding. 2
  edge forms stay out: a directive with no file, and the gopanic frame of the
  panic path.
- Dart gives no machine address, no program counter, and no allocation count.
  A test of `%p` arithmetic, of `runtime.FuncForPC`, or of
  `testing.AllocsPerRun` has no Dart form.
- 2 values with no size take 2 cells. gc can give them 1 address.
- A type that nests 30 levels or more erases at 2 depths, and the 2 Dart
  forms differ.
- The reflect identity of a local type inside a generic function drops the
  outer type argument. 2 instantiations of `F[A].T[B]` with 2 arguments for A
  compare equal when B stays equal.
- A goroutine with a busy loop and no block operation never yields. The other
  goroutines never run, exactly as GopherJS behaves.
- `reflect.Value.Elem` of a pointer to an interface stays open.

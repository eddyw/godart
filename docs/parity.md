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

The verb set of `fmt`: `%v`, `%+v`, `%T`, `%d`, `%s`, `%q`, `%t`, `%c`, `%b`,
`%o`, `%x`, `%X`, `%f`, `%e`, `%g`, and `%%`. The width and the precision work.
The flag `-` works.

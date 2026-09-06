# The report of the runtime stream

The stream holds the standard library overlay, `reflect`, `encoding/json`,
`fmt`, and the input and output runtime. It closed 7 tasks of its own list and
5 defects that the conformance stream handed over.

The branch is `runtime-work`. 4 commits hold the work:

| Commit | What it holds |
|---|---|
| `c0ad54b` | The tasks 1, 2, 3, and 4 |
| `a488bee` | The tasks 5 and 6 |
| `26e7516` | The task 7 |
| `0403ba0` | The 5 defects of the conformance stream |

Each path below is relative to the root of the repository.

## 1. A marshal of a type that declares MarshalJSON or MarshalText

**Closed.**

### The real cause

`runtime/goruntime/lib/src/reflect.dart`, `_pointerMethods`, line 663.

The compiler writes the description of `*T` only when the pointer holds a
method that `T` does not hold. Each method of `Num` in the witness reads a value
receiver, thus the 2 method sets are equal and no such description stands in the
output. `$ptrDesc` builds one at run time, and `_pointerMethods` built each
entry of the method table with `_looseFunc`.

`_looseFunc` gives a closure of the Dart type `(Object?) => Object?`. A Dart
function type is covariant in a result, thus the cast of the call site rejected
that closure. The report reads:

```
type '(Object?) => Object?' is not a subtype of
type '(Object?) => (Slice<int>, Iface?)' in type cast
```

`encoding/json/v2` asks `reflect` for the pointer to the type first, thus each
addressable value of a named base type took that path.

### The change

`runtime/goruntime/lib/src/reflect.dart`:

- `_typedResultFunc`, line 1114, builds a closure whose result type is the Dart
  type of the results of the signature. The description of each result names its
  Dart type through `withType`, thus the closure carries the exact type.
- `_arityFunc`, line 1179, gives the closure the count of the parameters. Each
  parameter takes `Object?`, which fits each function type of that arity.
- `_pointerMethods`, line 663, and `_typedFunc`, line 1096, each read the new
  builder.

A method of more than 4 results falls back to the old form. No package of the
surface holds such a method.

### The proof

- `testdata/programs/rtmarshaler` holds 26 shapes: a value receiver, a pointer
  receiver, a named integer, a named text, a named slice, a map key, and an
  interface value. `make compare PROG=rtmarshaler` reports the 2 outputs equal.
  Before the change the Dart output held the cast error above.
- The real suite: `dart run bin/encoding/json_test.dart -test.run=Marshaler`.
  Before, `TestMarshalerEscaping` ended the whole binary with the cast error.
  After, 4 tests pass: `TestMarshalerEscaping`,
  `TestTextMarshalerMapKeysAreSorted`, `TestNilMarshalerTextMapKey`, and
  `TestMarshalerError`.

## 2. reflect.Value.SetMapIndex copied 1 level

**Closed.**

### The real cause

`runtime/goruntime/lib/src/reflect.dart`, `$pointerSet`, line 139.

The old body held `field.set(target, field.get(stored))` for each field. A field
of a struct type holds an object in Dart, thus the target and the source shared
that object. `$fieldWrite`, line 327, and `$valueSetIndex`, line 388, held the
same defect. A pointer to an array panicked with `reflect: Set on a pointer to
[2]main.Num`.

`SetMapIndex` of `internal/overlay/std/reflect/reflect.go`, line 950, builds a
fresh cell and writes the value into it, thus the 1 level copy reached each
entry of a map.

### The change

- `_assignInto`, line 173, walks each level of a struct and of an array. It
  keeps the object of the target, because Go holds the address of a field of the
  target and a fresh object would leave that address behind.
- `_copyOf`, line 257, gives a fresh value of a struct or of an array.
- `_holdsValue`, line 161, names the 2 kinds that hold a value in Go and an
  object in Dart.
- `$pointerSet`, `$fieldWrite`, and `$valueSetIndex` each read `_assignInto`.
- `SetMapIndex` copies an array too now, and not a struct only.

### The proof

`testdata/programs/rtmapset2` writes 3 levels of struct plus an array of struct
through 1 cell, exactly as a decoder does. It also writes through a pointer,
through a field, through an element of a slice, and into a map of an array.
`make compare PROG=rtmapset2` reports the 2 outputs equal.

## 3. A reflect.Value carried no validity mark

**Closed.**

### The real cause

`internal/overlay/std/reflect/reflect.go`, line 760. `IsValid` read
`v.read() != nil`.

A nil interface, a nil pointer, and a nil map each read as nil in Dart. Go holds
each 1 valid. `MapIndex` of a key that the map does not hold thus gave the same
answer as an entry that holds a nil interface, and `Zero` of an interface type
reported false.

### The change

- The `Value` struct holds the field `ok`, line 681. Only the zero `Value` holds
  false, and each of the 40 constructors sets it.
- `IsValid` reads the mark.
- `MapIndex`, line 1002, reads the new call `dart.MapHas` in front of the read,
  thus a missing key gives the zero `Value` and an entry of a nil interface
  gives a valid one.
- `ValueOf(nil)`, line 686, gives the zero `Value`.
- `Elem` of a nil pointer and of a nil interface each give the zero `Value`,
  line 828, exactly as Go does.
- `SetMapIndex` reads the mark in place of a test of 4 fields.
- The new intrinsic: `MapHas` in `internal/overlay/std/syscall/dart/dart.go`
  line 240, `internal/emit/call.go` line 313, and `$mapHas` of
  `runtime/goruntime/lib/src/reflect.dart` line 1449.

The gob encoder read `IsValid` as the nil test of an interface. It reads
`Elem().IsValid()` now, because `IsNil` of the overlay answers the value inside
the interface and a typed nil pointer would read as a nil interface.
`internal/overlay/std/encoding/gob/encode.go`, line 512.

### The proof

`testdata/programs/rtvalid` writes the mark and the kind of 21 values: a map of
an interface element, a map of a pointer element, the zero value of 5 kinds, a
field of 5 kinds, an element of a slice of interface, `Elem` of a nil pointer,
`Elem` of a nil interface, an absent field, an absent method, `SetMapIndex` with
a nil interface, and a range over a map. `make compare PROG=rtvalid` reports the
2 outputs equal.

## 4. TestUnmarshalMaxDepth did not end

**Closed for the named cost. 1 deeper limit stays open.**

### The real cause

`internal/overlay/std/strings/strings.go`, `Repeat`, line 214. The old body
joined 1 part at a time:

```go
out := ""
for range count {
	out += s
}
```

Each step builds a fresh text of the whole length, thus the cost stands at the
square of the result. `TestUnmarshalMaxDepth` of `encoding/json` asks for
`strings.Repeat("{\"a\":", 3000000)`, which is 15 megabytes. That 1 call moves
about 2.25 times 10 to the power 13 bytes.

The decoder was never the cost. The test never reached it.

### The change

`Repeat` allocates the whole size 1 time and it doubles the text. The count of
the steps thus stands at the logarithm of the count of the parts.

### The measured numbers

A probe of `strings.Repeat` ahead of time:

| Case | Before | After |
|---|---|---|
| 1,000 bytes | 0 ms | 0 ms |
| 10,000 bytes | 5 ms | 0 ms |
| 100,000 bytes | 166 ms | 0 ms |
| 300,000 bytes | 2,923 ms | 0 ms |
| 3,000,000 bytes | did not end | 0 ms |
| `{"a":` 3,000,000 times | did not end | 1 ms |

`make std-test STD=encoding/json` did not end. A run of 14 minutes of wall time
and 31 minutes of processor time reached no result, and the arithmetic above
gives about 1.5 hours for the 1 call. The whole suite now ends in 8.1 seconds
and it reports 304 tests that pass.

### What stays open

`TestUnmarshalMaxDepth` still fails, and the cause is new and deeper. The
decoder of `encoding/json/v2/jsontext` is a recursive descent: `consumeValue`
calls `consumeArray` and `consumeArray` calls `consumeValue`. The Dart stack
holds about 1,000 levels of that pair, and the test asks for 9,999. The report
is `Stack Overflow` and the trace holds 2,752 frames.

A probe reads `{"a":` plus n open brackets: n of 1,000 decodes, and n of 1,100
overflows. The Dart VM holds no flag for the size of the stack, and each
goroutine of `runtime/goruntime/lib/src/sched.dart` runs on the stack of the
main isolate. The fix is outside the paths of this stream. Read the list of the
defects at the end.

## 5. fmt.Sprintf cost 8.8 times the same call in Go

**Closed.**

### The real cause

`internal/overlay/std/fmt/fmt.go`, `Sprintf`, line 105.

The loop built the text of each verb as a Go string, and it gave that text to
the bridge `dart.Format`. 4 costs stood there for each verb of each call:

1. The build of the verb text. `%.2f` took 1 substring and 2 concatenations.
2. The decode of that Go string into a Dart string.
3. The scan of that Dart string in the runtime: `endsWith` for the letter, 3
   calls of `contains` for a flag, and `indexOf` for the precision.
4. A `spec` object of Dart for each verb.

2 more costs stood next to them. `pad` walked the runes of the text under a
width of 0, and `fill` joined 1 rune at a time.

### The change

- The new intrinsic `FormatVerb` takes the letter, the bits of the flags, the
  width, and the precision as 4 numbers.
  `internal/overlay/std/syscall/dart/dart.go` line 35, `internal/emit/call.go`
  line 239, and `$goFormatVerb` of `runtime/goruntime/lib/src/format.dart` line
  83.
- `_Verb` of `runtime/goruntime/lib/src/format.dart`, line 27, holds the parsed
  form. Each function of the writer reads it in place of the text.
- `_parseVerb`, line 143, keeps the old bridge `$goFormat` for `strconv`,
  `fmt/format.go`, and `runtime/pinner.go`.
- The text of the flags stands only where a `Format` method, the verb `U`, or a
  report reads it. `flagText`, line 303.
- The `spec` object stands only in the path of a `Format` method.
- `pad`, line 573, gives a text back under a width of 0 with no walk.
- `fill`, line 640, reads `strings.Repeat`.

### The measured numbers

The measurement ran B, A, B with `./bench/run.sh` on a machine whose load
average stood near 7. B holds the whole change of the tasks 5 and 6 plus the
width of the defect A. A is the commit `c0ad54b`. Each number is the gart form,
in milliseconds.

| Case | B1 | A | B2 | A against B2 |
|---|---|---|---|---|
| `fmt.sprintf.100k` | 62.25 | 89.71 | 61.63 | 1.45 |

The 2 B runs stand 1.0 percent apart. The ratio against Go moves from 8.8 to
6.1.

`str.concat.5k` is the control that holds no call of `fmt` and no call of
`strings`: it gave 4.58, 4.32, and 4.39. Each other case of `bench/` moved by
less than the drift of the machine.

An earlier A, B, A run measured the bridge alone, with no move of the width and
with no cut of the `spec` object: 91.02, 60.12, and 89.84.

## 6. strings.Split and strings.Replace allocate too much

**Closed.**

### The real cause

`internal/overlay/std/strings/strings.go`.

- `Split`, line 271, built the result with `append` on an empty slice. A line of
  6 parts thus took 4 copies of the storage.
- `Replace`, line 385, built the result with a `Builder`, and `Builder.Grow`,
  line 798, was an empty body. A text of 45 bytes thus took 6 copies of the
  storage.
- Each call of `matchAt`, line 261, ran for each position of the text.
- `Index`, line 19, and `LastIndex`, line 37, compared 2 substrings. Each
  position of the text thus built 1 Go string, which is a view and a header.

### The change

- `Split` and `Replace` each take the count of the parts in front of the walk,
  thus the storage of the result takes 1 allocation.
- `Builder.Grow` reserves the room.
- `Split`, `SplitSeq`, `Count`, and `Replace` each read the first byte of the
  separator in front of the whole match.
- `Index` and `LastIndex` read the bytes through `matchAt`.

### The measured numbers

The same B, A, B run as the task 5.

| Case | B1 | A | B2 | A against B2 |
|---|---|---|---|---|
| `str.split.100k` | 36.02 | 47.85 | 36.49 | 1.31 |
| `str.replace.100k` | 38.73 | 47.54 | 38.65 | 1.23 |
| `str.join.100k` | 11.73 | 12.63 | 11.53 | 1.10 |

`str.join.100k` holds no change of this stream. It moved less than the drift.

### 1 trade that needs a decision

`strings.Join`, line 244, holds the same square cost as the old `Repeat`. A
build into 1 byte slice of the whole size joins 2,000 elements 42 times faster,
and it costs 17 percent more for the 6 elements of the case `str.join.100k` of
`bench/`. A probe ahead of time gives these numbers:

| Form | 6 elements, 100,000 times | 2,000 elements, 20 times |
|---|---|---|
| The concatenation of today | 13.0 ms | 37.5 ms |
| A `Builder` with `Grow` | 16.3 ms | 0.9 ms |
| A byte slice of the whole size | 15.2 ms | 0.9 ms |
| A pre-sized slice with `copy` | 23.3 ms | 1.3 ms |

`Join` keeps the concatenation, because the task list of this stream does not
hold it and the change would move a measured case of `bench/` by 17 percent. The
comment of `Join` names this report.

## 7. Per byte loops of the input and output runtime

**Closed.**

### The real cause

6 loops read and wrote 1 byte at a time:

| File | Function | What it moved |
|---|---|---|
| `file.dart` | `_bytes`, line 179 | The bytes of a slice for a write |
| `file.dart` | `$fileRead` | The bytes of a read into a slice |
| `net.dart` | `_bytes`, line 267, and `_take`, line 275 | A write and a read of a connection |
| `net.dart` | `$packetRead`, line 887 | A read of a datagram |
| `process.dart` | `_bytes`, line 55, and `_take`, line 61 | A write and a read of a pipe |
| `http2.dart` | `_h2Bytes`, line 110, and `_takeH2`, line 221 | A write and a read of a stream |
| `json.dart` | `_base64`, line 210 | The bytes of a base64 field |

### The change

- `file.dart`, line 96 and line 110: a read writes into the storage of the slice
  itself, and a write reads the range of that storage. Neither one needs a
  buffer.
- `net.dart`, `process.dart`, and `http2.dart`: each move reads `setRange`. A
  write of a connection, of a pipe, and of a stream keeps a copy, because the
  write is not synchronous and the caller may write the slice again.
- `json.dart`: the base64 field reads a view of the storage. The old form built
  a `List` of the length of the slice, and each element of such a list holds 8
  bytes in the Dart VM.

A store into a `Uint8List` holds the low 8 bits, exactly as the mask of each
loop did.

### The proof and the measured numbers

`testdata/programs/rtio` moves a block through each of the 6 paths and it writes
the length and the FNV-1a hash of the result. The pattern holds each value of a
byte, thus a copy that loses the high bit fails. `make compare PROG=rtio`
reports the 2 outputs equal.

The measurement ran A, B, A ahead of time with a probe. Each number is the lower
of 3 laps, in milliseconds.

| Case | A1 | B | A2 | B against A2 |
|---|---|---|---|---|
| A write of a file of 16 megabytes | 41.4 | 2.0 | 26.2 | 13.1 |
| A read of a file of 16 megabytes | 34.8 | 0.5 | 27.3 | 54.6 |
| 8 megabytes through `cat` | 74.7 | 10.3 | 34.2 | 3.3 |
| 8 megabytes through a TCP echo | 127.0 | 8.1 | 72.9 | 9.0 |
| A marshal of 4 megabytes to base64 | 34.2 | 31.4 | 32.3 | 1.03 |

The base64 case holds the loop at 1.4 milliseconds of 32. The encoder of Dart
takes 5.2 and the JSON writer takes 8.8, thus the loop never stood at the front
there. The change still drops an allocation of 32 megabytes.

## The 5 defects of the conformance stream

Each witness moved into `testdata/programs/` with its name unchanged, and
`make compare PROG=<name>` reports the 2 outputs equal for each 1.

### A. A width did not reach each element of a compound value

**Closed.** Witness `conffmtwidth`.

The cause: the width stood on the Go side. `pad` of
`internal/overlay/std/fmt/fmt.go` wrote it on the whole text of a value, thus
`%-8v` of `[]int{1, 2}` gave `[1 2]   `. Go writes the width on each leaf, and
the brackets take none, thus Go gives `[1        2       ]`.

The change: `FormatVerb` takes the width, and the runtime holds the pad. `_leaf`
of `runtime/goruntime/lib/src/format.dart`, line 292, writes the precision
digits and the width on 1 leaf text. `_pad`, line 299, `_zeroPad`, line 316,
`_precDigits`, line 375, and `_runeCount`, line 360, each moved from the Go side
to the runtime. A slice, an array, a map, and a struct give the verb to each
element, and no compound writes a width of its own. A list of bytes under the
verb `s`, `q`, `x`, or `X` stands as 1 text, thus the width covers the whole
text there. `_list`, line 1012.

### B. The verb %x of a float32 wrote the fraction 1 bit too low

**Closed.** Witness `confhexfloat32`.

The cause: `_hexFloat` of `runtime/goruntime/lib/src/format.dart`, line 1284,
held `m -= lead` and it wrote the fraction in 6 hexadecimal digits. 6 digits hold
24 bits and the fraction of a `float32` holds 23, thus each digit stood 1 place
too low.

The change: the single form writes `(m - lead) << 1`. The fraction of a
`float64` holds 52 bits, which fills 13 digits exactly, thus that path needs no
shift.

### C. The verb %w of a value that is not an error

**Closed.** Witness `conffmtwrapnil`.

The cause: `Errorf` of `internal/overlay/std/fmt/fmt.go`, line 471, replaced
each `%w` with `%v`.

The change: `Errorf` keeps the letter `w` for an operand that holds no error,
thus `Sprintf` writes the bad verb form of Go: `%!w(<nil>)` and
`%!w(string=a text)`.

### D. errors.As with a target of a struct type that is not a pointer

**Closed.** Witness `conferrorsasvalue`.

The cause: `$assignError` wrote the value through a `$Ptr` cell. The Dart form
of a pointer to a struct is the object itself, thus the test `box is $Ptr` failed
and the call reported false.

The change: such a target takes the copy of each field, through `_assignInto` of
the task 2. `$assignError` moved from `format.dart` to
`runtime/goruntime/lib/src/reflect.dart`, line 218, because that walk stands
there. `goruntime.dart` exports the 2 files, thus each call of the emitted code
reads the same name.

### E. errors.As with a pointer to an interface type

**Closed.** Witness `conferrorsasiface`.

The cause: `$assignError` panicked with `errors.As with a pointer to an
interface type is outside the subset`. `docs/parity.md` names the gap.

The change: the description of an interface type holds the key of each method of
it, under the placeholder `$ifaceMethod`. `$implements` of `iface.dart` reads
that list, thus the check of the method set has a source. The cell then takes
the interface value itself.

## The defects that stand outside the paths of this stream

Each 1 below stays as it is.

1. **The Dart stack limits the decoder of `encoding/json/v2`.**
   `$(go env GOROOT)/src/encoding/json/jsontext/decode.go`, `consumeValue` and
   `consumeArray`, recurse 1 level for each level of the document. The Dart VM
   overflows above 1,000 levels, and `TestUnmarshalMaxDepth` asks for 9,999.
   `runtime/goruntime/lib/src/sched.dart`, line 288, runs each goroutine on the
   stack of the main isolate, and the Dart VM holds no flag for the size of that
   stack. The whole test binary ends there, thus each test behind that 1 runs
   not.

2. **`-test.run` reads only the form with an equal sign.**
   `internal/overlay/std/testing/testing.go`, line 940, reads
   `-test.run=<pattern>`. The go command also accepts `-test.run <pattern>`, and
   that form passes the filter with no effect. A whole suite then runs where the
   caller asked for 1 test.

3. **`TestUnmarshalEmbeddedUnexported` of `encoding/json` fails.** 2 of the 9
   cases stand wrong. Go refuses a write through an embedded pointer to an
   unexported struct type, and gart writes the field. The report of gart reads
   `cannot unmarshal number into Go struct field S5.Q of type int64`, and Go
   reads `cannot set embedded pointer to unexported struct type`. The defect
   stood before this stream, and it belongs to the field walk of the decoder.

4. **`$substring` builds 2 view objects for 1 substring.**
   `runtime/goruntime/lib/src/string.dart`, line 187, calls `Uint8List.view` and
   then `asUnmodifiableView`. A scan that gives many parts thus allocates 3
   objects for each part. `str.split.100k` holds 7 such builds for each
   operation.

5. **`net.TCPConn` holds no `CloseWrite`.**
   `internal/overlay/std/net/tcpsock.go`, line 17, names the gap already. The
   witness `rtio` reads a fixed count in place of a half close.

## How to run each proof again

```sh
make build
make compare PROG=rtmarshaler        # the task 1
make compare PROG=rtmapset2          # the task 2
make compare PROG=rtvalid            # the task 3
make std-test STD=encoding/json      # the task 1 and the task 4
make compare PROG=rtio               # the task 7
make compare PROG=conffmtwidth       # the defect A
make compare PROG=confhexfloat32     # the defect B
make compare PROG=conffmtwrapnil     # the defect C
make compare PROG=conferrorsasvalue  # the defect D
make compare PROG=conferrorsasiface  # the defect E
./bench/run.sh                       # the task 5 and the task 6
```

`go test ./internal/driver -run 'TestPrograms/[a-f]' -count=1` and the 2 other
ranges run the whole set of the programs of `testdata/`. Each 1 passes.
`make dart-test` and `dart analyze --fatal-infos` each pass.

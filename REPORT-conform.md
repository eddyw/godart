# The conformance suite of the Go language

The suite covers the Go language itself. Each program stands in
`testdata/programs/` with the name prefix `conf`. `TestPrograms` of
`internal/driver` finds each directory with no registration. It runs the
program with `go run .` and with the Dart output of gart, and it compares the
2 outputs byte for byte.

Each program writes a table of results, and each line holds a label. 1 build
thus gives many assertions. The column "Lines" holds the count of the output
lines of the program, which is the count of the assertions.

The cause of the suite is the defect of 2026-09-06: `fmt.Printf("%.0f", 0.5)`
wrote 1, and real Go writes 0. No test caught it.

## The areas

| # | Area | Lines | State |
|---|---|---|---|
| 1 | `confconst` | 99 | Passes |
| 2 | `confint` | 146 | Passes |
| 3 | `confconv` | 123 | Passes |
| 4 | `conffloat` | 156 | Passes |
| 5 | `confstring` | 102 | Passes |
| 6 | `confslice` | 90 | Passes |
| 7 | `confmap` | 78 | Passes |
| 8 | `confstruct` | 120 | Passes |
| 9 | `confiface` | 89 | Passes |
| 10 | `confmethod` | 90 | Open |
| 11 | `confclosure` | | Open |
| 12 | `confdefer` | | Open |
| 13 | `confpanic` | | Open |
| 14 | `confchan` | | Open |
| 15 | `confgoroutine` | | Open |
| 16 | `confgeneric` | | Open |
| 17 | `confrange` | | Open |
| 18 | `confflow` | | Open |
| 19 | `confassign` | | Open |
| 20 | `confinit` | | Open |
| 21 | `conferrors` | | Open |
| 22 | `confsort` | | Open |
| 23 | `conftime` | | Open |
| 24 | `confembed` | | Open |

## The defects

### 1. min and max compare a uint64 as signed

Out of scope. `internal/emit/call.go` owns the fix.

The witness: `/tmp/gart-conformance/confminmaxu64`

```go
var u, v uint64 = 0, 1<<64 - 1
fmt.Println("min", min(u, v))
fmt.Println("max", max(u, v))
```

| | Output |
|---|---|
| Go | `min 0` / `max 18446744073709551615` |
| gart | `min 18446744073709551615` / `max 0` |

The cause: `minMax` of `internal/emit/call.go`, line 1457, writes a call of
`$minOf` or of `$maxOf` for each element type. The 2 helpers stand in
`runtime/goruntime/lib/src/string.dart`, line 418 and line 427, and each 1
compares a `GoInt` with the signed operator `<=` of Dart. A Go `uint64` holds
its high values as a negative Dart int, thus the order inverts. A narrow
unsigned type keeps a value that is not negative, thus only `uint64`, `uint`,
and `uintptr` show the defect.

The fix: `minMax` must write `$uminOf` and `$umaxOf` for an unsigned 64-bit
type, and `string.dart` must hold the 2 new helpers over `$ult`. The emitter
alone knows the signedness, thus no change of the runtime alone closes it.

### 2. A conversion of a float64 to a uint64 above 2 to the power 63

Out of scope. `internal/emit/call.go` owns the fix.

The witness: `/tmp/gart-conformance/conffloattouint`

```go
f := float64(1 << 63)
fmt.Println(uint64(f))
```

| | Output |
|---|---|
| Go | `9223372036854775808` |
| gart | `9223372036854775807` |

The value 2 to the power 63 stands inside the range of a `uint64`, thus Go
gives the exact result. The case is not implementation-defined.

The cause: `conv` of `internal/emit/call.go`, line 1555, writes
`$fromF64(x)` for each conversion of a float to an integer. `$fromF64` of
`runtime/goruntime/lib/src/goint_native.dart`, line 155, calls
`double.truncate()` of Dart, which clamps the result to the range of a signed
64-bit int.

The fix: `conv` must write a helper `$fromF64U` for an unsigned 64-bit target,
and `goint_native.dart` and `goint_js.dart` must hold that helper. The helper
subtracts 2 to the power 64 from a value of at least 2 to the power 63 and
then truncates, thus the bit pattern is right.

### 3. A width of a verb does not reach each element of a compound value

Out of scope. `internal/overlay/std/fmt/**` and
`runtime/goruntime/lib/src/format.dart` own the fix.

The witness: `/tmp/gart-conformance/conffmtwidth`

```go
fmt.Printf("[%-8v]\n", []int{1, 2})
```

| | Output |
|---|---|
| Go | `[[1        2       ]]` |
| gart | `[[1 2]]` |

Go gives the width to each element of a slice, of an array, of a map, and of a
struct, and not to the whole text. The rule reaches each key and each value of
a map too: `%-8v` of `map[string]int{"a": 1}` gives `map[a       :1       ]`.

The cause: the printer of gart writes the whole compound value first, and it
then pads that text.

### 4. The verb %x of a float32 writes the fraction 1 bit too low

Out of scope. `runtime/goruntime/lib/src/format.dart` owns the fix.

The witness: `/tmp/gart-conformance/confhexfloat32`

```go
fmt.Printf("%x\n", float32(math.MaxFloat32))
```

| | Output |
|---|---|
| Go | `0x1.fffffep+127` |
| gart | `0x1.7fffffp+127` |

The verb `%x` of a `float64` is right. Only the 32-bit form differs.

The cause: `_hexFloat` of `runtime/goruntime/lib/src/format.dart`, line 1070,
holds `m -= lead;` and then writes the fraction in 6 hexadecimal digits. The
fraction of a `float32` holds 23 bits, and 6 digits hold 24 bits. Go shifts
the fraction 1 place to the left to fill the 24 bits. The Dart form pads on
the left instead, thus each digit stands 1 place too low. The fraction of a
`float64` holds 52 bits, which fills 13 digits exactly, thus that path is
right.

The fix: `_hexFloat` must write `m = (m - lead) << 1;` for the single form.

### 5. A slice expression of a nil slice gave a slice that is not nil

**Fixed.** `runtime/goruntime/lib/src/slice.dart`, `Slice.range`.

```go
var a []int
fmt.Println(a[:0] == nil)
```

| | Output |
|---|---|
| Go | `true` |
| gart before | `false` |
| gart after | `true` |

The specification says that a valid slice expression of a nil slice gives a
nil slice. `Slice.range` called `$view`, which builds a header with the nil
flag false. A nil slice holds a capacity of 0, thus each bound of a valid
expression is 0 and the source header is already the right result. `range`
now gives the source header back for a nil slice, after the bound tests.

`testdata/programs/confslice` holds the witness in the section
"nil against empty".

## The cases that the specification leaves open

The suite does not test these, because Go itself gives no answer.

| Case | The reason |
|---|---|
| A conversion of a float outside the range of the target integer type | The specification calls the result implementation-dependent. gc gives 1 answer on arm64 and another on amd64. `int8(1e15)` gives -1 on arm64 and 0 with a plain truncation. |
| The capacity after an append that grows | The specification names no growth rule. `confslice` reads the length and the rule `cap >= len` only. |
| The order of a range over a map | Random in Go. Each output of `confmap` takes a sort first. |
| A pair that a write makes during a range over a map | The specification says that the loop can give it or skip it. |
| A `select` with 2 ready cases | Random in Go, and the first case in source order in gart. Read `docs/parity.md`. |

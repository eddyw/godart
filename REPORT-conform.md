# The conformance suite of the Go language

The suite covers the Go language itself. Each program stands in
`testdata/programs/` with the name prefix `conf`. `TestPrograms` of
`internal/driver` finds each directory with no registration. It runs the
program with `go run .` and with the Dart output of gart, and it compares the
2 outputs byte for byte.

Each program writes a table of results, and each line holds a label. 1 build
thus gives many assertions. The column "Lines" holds the count of the output
lines, which is the count of the assertions.

The cause of the suite is the defect of 2026-09-06: `fmt.Printf("%.0f", 0.5)`
wrote 1, and real Go writes 0. No test caught it. The suite closes that class
of hole for the language itself.

**The 24 programs write 2180 lines, and each 1 passes.**

## The areas

| # | Area | Lines | The cover | State |
|---|---|---|---|---|
| 1 | `confconst` | 99 | A constant expression, iota, a value above the range of an int64, a constant shift, a typed constant, a rune constant | Passes |
| 2 | `confint` | 146 | The arithmetic of each integer type, a wrap, a shift above the width, a shift of a negative value, a division with a negative operand, the most negative value, min and max | Passes |
| 3 | `confconv` | 123 | Each numeric conversion, a truncation toward zero, a wrap of the target width, a string against a byte slice and a rune slice, an integer to a string | Passes |
| 4 | `conffloat` | 156 | NaN, the 2 infinities, a negative zero, a denormal, each comparison, the verbs v, g, e, f, E, G, and the bit pattern | Passes |
| 5 | `confstring` | 102 | An index, a slice, a range over a byte that carries no rune, len against the count of runes, a concatenation, a comparison | Passes |
| 6 | `confslice` | 90 | An append that shares the storage and 1 that grows, the 3 index form, a copy of 2 ranges that overlap, a slice of an array, clear, nil against empty | Passes |
| 7 | `confmap` | 78 | A struct key, an array key, an interface key, an absent key, a delete during a range, the 2 result form, clear | Passes |
| 8 | `confstruct` | 120 | A comparison, an embedded field, a promoted method, an unnamed struct type, a field of a zero size, the size and the offset from reflect | Passes |
| 9 | `confiface` | 89 | Each form of a type assertion, a type switch, a nil interface against a typed nil, a value type against a pointer type, an embedded interface | Passes |
| 10 | `confmethod` | 90 | A value receiver against a pointer receiver, a method value, a method expression, a method of an embedded field, the 2 method sets | Passes |
| 11 | `confclosure` | 52 | A capture of a variable, the loop variable of Go 1.22, a closure inside a defer, a recursive closure | Passes |
| 12 | `confdefer` | 43 | The order of each call, the value of each argument at the defer, a write of a named result, a defer in a loop, recover inside and outside | Passes |
| 13 | `confpanic` | 96 | A nil map write, an index, a nil pointer, a division by zero, a failed assertion, a slice bound, and the text of each 1 | Passes |
| 14 | `confchan` | 64 | A buffered and an unbuffered channel, a closed channel, the 2 result form, a nil channel in a select, a default, a range, len and cap | Passes |
| 15 | `confgoroutine` | 68 | sync.WaitGroup, sync.Mutex, sync.RWMutex, sync.Once, sync.Map, sync/atomic, and an order that a channel fixes | Passes |
| 16 | `confgeneric` | 98 | A type parameter, a constraint with a method, a union constraint, the inference, a generic type with a method, any and comparable | Passes |
| 17 | `confrange` | 58 | A range over an integer, over a function of 1 and of 2 values, over a string, over a pointer to an array, over a channel | Passes |
| 18 | `confflow` | 67 | A labeled break and continue, goto, a switch with no condition, fallthrough, a type switch, a select with a default | Passes |
| 19 | `confassign` | 69 | The order of an assignment of many values, a swap, a read of the old value, the order of the operands, a value of a map | Passes |
| 20 | `confinit` | 39 | The order of a package variable, an init function, the order between 3 files of 1 package, a variable that reads another 1 | Passes |
| 21 | `conferrors` | 98 | errors.New, fmt.Errorf with %w, errors.Is, errors.As, errors.Join, a custom type, a nil error in an interface | Passes |
| 22 | `confsort` | 76 | sort.Slice, sort.SliceStable, slices.Sort, slices.SortFunc, slices.BinarySearch, maps.Keys, and the stability of each 1 | Passes |
| 23 | `conftime` | 183 | A duration, the arithmetic of a time, each layout, a parse, a comparison, with a fixed time | Passes |
| 24 | `confembed` | 76 | A struct that embeds a pointer, 1 that embeds an interface, 2 embedded names of 1 depth, a method that hides another | Passes |

## The defects

The suite found 17 defects. 10 stand inside the language core, and each 1 is
closed. 7 stand in the paths that another stream owns, and each 1 holds a
witness under `/tmp/gart-conformance/`.

| # | The defect | The owner | State |
|---|---|---|---|
| 1 | `min` and `max` compare a `uint64` as signed | `internal/emit/call.go` | Open |
| 2 | A conversion of a float64 to a uint64 above 2 to the power 63 | `internal/emit/call.go` | Open |
| 3 | A width of a verb does not reach each element of a compound value | `fmt` | Open |
| 4 | The verb `%x` of a float32 writes the fraction 1 bit too low | `format.dart` | Open |
| 5 | A slice expression of a nil slice gave a slice that is not nil | `slice.dart` | Fixed |
| 6 | `recover` gave the value again at a second call | `panic.dart` | Fixed |
| 7 | A failed assertion to an interface type named the import path | `iface.dart`, `internal/emit` | Fixed |
| 8 | `make` of a channel with a capacity below 0 built the channel | `chan.dart` | Fixed |
| 9 | `make` of a slice with a huge length ran the allocator out of memory | `slice.dart` | Fixed |
| 10 | A type parameter of a set of float types was rejected | `internal/emit`, `internal/verify` | Fixed |
| 11 | `errors.As` with a target of a struct type that is not a pointer | `format.dart` | Open |
| 12 | `errors.As` with a pointer to an interface type | `format.dart` | Open |
| 13 | The verb `%w` of a value that is not an error writes no bad verb form | `fmt` | Open |
| 14 | A range over an array read the array itself and not a copy | `internal/normalize` | Fixed |
| 15 | A short declaration wrote a name in front of the read of a later value | `internal/normalize` | Fixed |
| 16 | The text of an instance of a generic type held a space behind a comma | `internal/emit` | Fixed |
| 17 | `Duration.String` of the most negative value wrote `--ns` | `internal/overlay/std/time` | Fixed |

### 1. min and max compare a uint64 as signed

**Open.** `internal/emit/call.go`, line 1457, owns the fix.

The witness: `conf-blocked/confminmaxu64`

```go
var u, v uint64 = 0, 1<<64 - 1
fmt.Println("min", min(u, v))
fmt.Println("max", max(u, v))
```

| | Output |
|---|---|
| Go | `min 0` and `max 18446744073709551615` |
| gart | `min 18446744073709551615` and `max 0` |

The cause: `minMax` of `internal/emit/call.go`, line 1457, writes a call of
`$minOf` or of `$maxOf` for each element type. The 2 helpers stand in
`runtime/goruntime/lib/src/string.dart`, line 418 and line 427, and each 1
compares a `GoInt` with the signed operator `<=` of Dart. A Go `uint64` holds
its high values as a negative Dart int, thus the order inverts. A narrow
unsigned type keeps a value that is not negative, thus only `uint64`, `uint`,
and `uintptr` show the defect.

The fix: `minMax` must write `$uminOf` and `$umaxOf` for an unsigned 64-bit
type, and `string.dart` must hold the 2 new helpers over the unsigned compare.
The emitter alone knows the signedness, thus no change of the runtime alone
closes it.

`testdata/programs/confint` holds each other case of `min` and `max`.

### 2. A conversion of a float64 to a uint64 above 2 to the power 63

**Open.** `internal/emit/call.go`, line 1555, owns the fix.

The witness: `conf-blocked/conffloattouint`

```go
f := float64(1 << 63)
fmt.Println(uint64(f))
```

| | Output |
|---|---|
| Go | `9223372036854775808` |
| gart | `9223372036854775807` |

The value 2 to the power 63 stands inside the range of a `uint64`, thus Go
gives the exact result and the case is not implementation-defined.

The cause: `conv` of `internal/emit/call.go`, line 1555, writes `$fromF64(x)`
for each conversion of a float to an integer. `$fromF64` of
`runtime/goruntime/lib/src/goint_native.dart`, line 155, calls
`double.truncate()` of Dart, which clamps the result to the range of a signed
64-bit int.

The fix: `conv` must write a helper `$fromF64U` for an unsigned 64-bit target,
and `goint_native.dart` and `goint_js.dart` must hold that helper. The helper
subtracts 2 to the power 64 from a value of at least 2 to the power 63 and it
then truncates, thus the bit pattern is right.

### 3. A width of a verb does not reach each element of a compound value

**Open.** `internal/overlay/std/fmt/**` and
`runtime/goruntime/lib/src/format.dart` own the fix.

The witness: `conf-blocked/conffmtwidth`

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
pads that text after that.

### 4. The verb %x of a float32 writes the fraction 1 bit too low

**Open.** `runtime/goruntime/lib/src/format.dart`, line 1070, owns the fix.

The witness: `conf-blocked/confhexfloat32`

```go
fmt.Printf("%x\n", float32(math.MaxFloat32))
```

| | Output |
|---|---|
| Go | `0x1.fffffep+127` |
| gart | `0x1.7fffffp+127` |

The verb `%x` of a `float64` is right. Only the 32-bit form differs.

The cause: `_hexFloat` of `format.dart`, line 1070, holds `m -= lead;` and it
then writes the fraction in 6 hexadecimal digits. The fraction of a `float32`
holds 23 bits, and 6 digits hold 24 bits. Go shifts the fraction 1 place to
the left to fill the 24 bits, and the Dart form pads on the left, thus each
digit stands 1 place too low. The fraction of a `float64` holds 52 bits, which
fills 13 digits exactly, thus that path is right.

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
now gives the source header back for a nil slice, behind the bound tests.

The witness stays in `testdata/programs/confslice`, section "nil against
empty".

### 6. recover gave the value again at a second call

**Fixed.** `runtime/goruntime/lib/src/panic.dart`, `$recover`.

```go
func f() (out string) {
	defer func() {
		first := recover()
		second := recover()
		out = fmt.Sprint(first, " then ", second)
	}()
	panic("H")
}
```

| | Output |
|---|---|
| Go | `H then <nil>` |
| gart before | `H then H` |
| gart after | `H then <nil>` |

A recover consumes the panic. `gorecover` of Go marks the panic with
`p.recovered` and each later call reads that mark and gives nil. `$recover`
set the flag `$recovered` and it read no flag, thus each call gave the value.
`$recover` now reads `$recovered` first. The flag belongs to the chain of 1
function, thus a new panic inside a deferred call clears it and the calls that
remain read that new panic.

The witness stays in `testdata/programs/confdefer`, section "recover".

### 7. A failed assertion to an interface type named the import path

**Fixed.** `runtime/goruntime/lib/src/iface.dart` and
`internal/emit/iface.go`.

```go
var c any = point{1, 2}
_ = c.(shape)
```

| | Output |
|---|---|
| Go | `interface conversion: main.point is not main.shape: missing method Area` |
| gart before | `interface conversion: example.com/x.point is not example.com/x.shape: missing method Area` |
| gart after | equal to Go |

A type descriptor holds 2 names: `name` is the identity, with the import path,
and `text` is the name that Go writes, with the package name. `$assertIface`
read `name`, and the emitter passed the identity name of the target type. The
report now reads `desc.display`, which gives `text` and falls back to `name`
for a predeclared type, and the new `typeDisplayText` of the emitter writes
the reflect text of the target.

The witness stays in `testdata/programs/confpanic`, section "a type
assertion".

### 8. make of a channel with a capacity below 0 built the channel

**Fixed.** `runtime/goruntime/lib/src/chan.dart`, the constructor of `Chan`.

```go
n := -1
_ = make(chan int, n)
```

| | Output |
|---|---|
| Go | `panic: makechan: size out of range` |
| gart before | no panic |
| gart after | equal to Go |

The constructor now rejects a capacity below 0 with the plain text of Go,
which writes no prefix `runtime error: `.

The witness stays in `testdata/programs/confpanic`, section "a builtin".

### 9. make of a slice with a huge length ran the allocator out of memory

**Fixed.** `runtime/goruntime/lib/src/slice.dart`, `$makeSlice`.

```go
n := 1 << 62
_ = make([]int, n, n)
```

| | Output |
|---|---|
| Go | `panic: runtime error: makeslice: len out of range` |
| gart before | `Unhandled exception: Out of Memory`, and the program ended |
| gart after | equal to Go |

Go compares the length against `maxAlloc` divided by the size of the element.
`maxAlloc` holds 2 to the power 48 on a 64-bit machine, thus a slice of an
element of 8 bytes reaches 2 to the power 45 elements. The allocator of Dart
takes no element size, thus 1 bound stands for each element type. A count
between this bound and the bound of Go needs at least 32 terabytes, which no
machine holds, and Go reports "out of memory" there too.

The witness stays in `testdata/programs/confpanic`, section "a builtin".

### 10. A type parameter of a set of float types was rejected

**Fixed.** `internal/verify/typeparam.go`, `internal/emit/type.go`, and
`internal/emit/expr.go`.

```go
type floating interface{ ~float32 | ~float64 }

func sum[T floating](list []T) T {
	var total T
	for _, v := range list {
		total += v
	}
	return total
}
```

| | Output |
|---|---|
| Go | `4` for `[]float64{1.5, 2.5}` |
| gart before | `the operator += reads the type parameter T, and its constraint holds a type that needs a mask of its width` |
| gart after | equal to Go |

A set of integer types already erased to the Dart type `GoInt`, and the
dictionary of the type argument carried the mask of the width. A set of float
types held no such rule, and the emitter kept the Dart type parameter. The
dictionary of `float32` already holds the round of 32 bits and the dictionary
of `float64` holds the identity, thus the same rule fits. `dartType` now
gives `double` for such a set, `mixedForm` accepts it, `isFloat` reports it,
`narrow` reads the mask of the dictionary, and stage 2 rejects only a set that
mixes an integer type with a narrow float type.

A constraint that mixes a narrow integer type with a float type stays out. The
Dart bound of such a set is `num`, and no mask of a narrow width fits that
bound. `conf-blocked/confmixedparam` holds the witness, and
`golang.org/x/exp/constraints.Ordered` is such a set.

The witness stays in `testdata/programs/confgeneric`.

### 11. errors.As with a target of a struct type that is not a pointer

**Open.** `runtime/goruntime/lib/src/format.dart`, `$assignError`, owns the
fix.

The witness: `conf-blocked/conferrorsasvalue`

```go
type valueError struct{ Field string }

func (v valueError) Error() string { return "bad field " + v.Field }

var err error = valueError{"name"}
var target valueError
fmt.Println(errors.As(err, &target))
```

| | Output |
|---|---|
| Go | `true` and `name` |
| gart | `false` and an empty text |

The cause: `$assignError` writes the value through a `$Ptr` cell. The Dart
form of a pointer to a struct is the object itself and not a cell, thus the
test `box is $Ptr<Object?>` fails and the function reports false. A write must
copy the fields of the value into the object of the target there.

`errors.AsType` of Go 1.27 takes no pointer, and it finds the same value.

### 12. errors.As with a pointer to an interface type

**Open.** `runtime/goruntime/lib/src/format.dart`, `$assignError`, owns the
fix.

The witness: `conf-blocked/conferrorsasiface`

```go
err := fmt.Errorf("outer: %w", &codeError{500})
var target interface{ Error() string }
fmt.Println(errors.As(err, &target))
```

| | Output |
|---|---|
| Go | `true` and `code 500: boom` |
| gart | `panic: runtime error: errors.As with a pointer to an interface type is outside the subset` |

`docs/parity.md` already names the gap. The type descriptor of an interface
type holds no method name, thus the check of the method set has no source.
The fix needs the method key list of an interface descriptor.

### 13. The verb %w of a value that is not an error writes no bad verb form

**Open.** `internal/overlay/std/fmt/**` owns the fix.

The witness: `conf-blocked/conffmtwrapnil`

```go
fmt.Println(fmt.Errorf("nothing: %w", error(nil)))
fmt.Println(fmt.Errorf("a text: %w", "not an error"))
```

| | Output |
|---|---|
| Go | `nothing: %!w(<nil>)` and `a text: %!w(string=not an error)` |
| gart | `nothing: <nil>` and the second line differs too |

`%w` accepts an operand that holds the interface `error` only. Each other
operand writes the bad verb form of Go.

### 14. A range over an array read the array itself and not a copy

**Fixed.** `internal/normalize/range.go`, `rangeList`.

```go
arr := [4]int{1, 2, 3, 4}
got := []int{}
for i, v := range arr {
	if i == 0 {
		arr[1] = 99
	}
	got = append(got, v)
}
fmt.Println(got)
```

| | Output |
|---|---|
| Go | `[1 2 3 4]` |
| gart before | `[1 99 3 4]` |
| gart after | `[1 2 3 4]` |

Go reads a range expression of an array type as a value, thus the walk holds
a copy and a write inside the body does not reach a later pass. A slice and a
pointer to an array each share the storage, and neither one takes a copy. The
stage now writes `copyPoint` around the container when the core type is an
array and the loop reads the value variable. `copyPoint` gives a fresh value
back with no copy, thus the result of a call costs nothing.

The witness stays in `testdata/programs/confrange`, section "an array and a
slice".

### 15. A short declaration wrote a name in front of the read of a later value

**Fixed.** `internal/normalize/stmt.go`, `define`.

```go
n := 1
n, m := n+10, n+20
fmt.Println(n, m)
```

| | Output |
|---|---|
| Go | `11 21` |
| gart before | `11 31` |
| gart after | `11 21` |

The specification runs an assignment in 2 phases: it reads each operand of the
left side and each expression of the right side first, and it writes each
target after that, from left to right. The left side of a short variable
declaration can hold a name that already exists, and such a name takes an
assignment. The stage wrote that name at once, thus a value that stands behind
it read the new value. Each value now takes a temporary variable when the left
side holds such a name. A list of new names alone needs no such variable,
because no value can read one of them.

The witness stays in `testdata/programs/confassign`, section "an assignment of
many values".

### 16. The text of an instance of a generic type held a space behind a comma

**Fixed.** `internal/emit/iface.go`, `reflectText`.

```go
fmt.Printf("%T\n", pair[int, string]{1, "a"})
```

| | Output |
|---|---|
| Go | `main.pair[int,string]` |
| gart before | `main.pair[int, string]` |
| gart after | `main.pair[int,string]` |

`reflectText` fell back to `types.TypeString`, which writes 1 space behind
each comma of a type argument list. `reflect` of Go writes no space there.
`reflectText` now holds a case for a named type with type arguments.

The witness stays in `testdata/programs/confgeneric`, section "a generic
type".

### 17. Duration.String of the most negative value wrote --ns

**Fixed.** `internal/overlay/std/time/time.go`, `Duration.String`.

```go
fmt.Println(time.Duration(-1 << 63))
```

| | Output |
|---|---|
| Go | `-2562047h47m16.854775808s` |
| gart before | `--ns` |
| gart after | `-2562047h47m16.854775808s` |

The magnitude of the smallest `Duration` is 2 to the power 63, which does not
fit an `int64`. The old body held `u := int64(d)` and `u = -u`, which gives
the same negative value again. The body now reads a `uint64`, and the 2 new
helpers `uitoa` and `upart` write an unsigned value.

The witness stays in `testdata/programs/conftime`, section "a duration".

## The cases that the specification leaves open

The suite does not test these, because Go itself gives no answer.

| Case | The reason |
|---|---|
| A conversion of a float outside the range of the target integer type | The specification calls the result implementation-dependent. gc gives 1 answer on arm64 and another on amd64. `int8(1e15)` gives -1 on arm64, and a plain truncation gives 0. |
| The capacity after an append that grows | The specification names no growth rule. `confslice` reads the length and the rule `cap >= len` only. |
| The order of a range over a map | Random in Go. Each output of `confmap` takes a sort first. |
| A pair that a write makes during a range over a map | The specification says that the loop can give it, or the loop can skip it. |
| A `select` with 2 ready cases | Random in Go, and the first case in source order in gart. Read `docs/parity.md`. |
| The order of 2 equal elements after an unstable sort | `sort.Slice` and `slices.Sort` name no such order. `confsort` reads the order of 2 equal elements only behind a stable sort. |

## A trap of gc, and how the suite works around it

gc reads the index of a panic report from a register. An inlined body can hold
another value there, thus `s[100]` of a slice inside a closure reports
`index out of range [0] with length 3`. A call of a function that carries the
directive `//go:noinline` gives the true index in each case.

`testdata/programs/confpanic` thus routes each index and each slice bound
through such a function. Read the helpers `at`, `high`, `low`, `lowHigh`, and
`three` of that program.

## The known limits that the suite reads around

| Limit | Where |
|---|---|
| The zero value of a `Time` is the start of the Unix epoch, and the range is the year 1678 to the year 2262 | A layout with no year gives the year 0, and such a value wraps. `conftime` gives each layout a date. Read "The clock and the timer" of `docs/parity.md`. |
| `%v` of a value that holds a pointer writes an address in Go | gart holds no address. `confembed` prints the value behind the pointer. Read "The runtime" of `docs/parity.md`. |
| A recovered run time value carries 1 description in gart, and Go holds 3 | `confpanic` writes the text of the value and not its type. |

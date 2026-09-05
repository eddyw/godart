# bench

This directory compares the same work on 4 runtimes: Go, the Dart that gart
writes from the Go source, Dart, and bun. It holds 3 groups of cases:

1. **The document.** A JSON decode, a JSON encode, a scan, and a byte loop.
2. **The string.** A concatenation, a builder, a substring, and the 2
   crossings of the boundary between a Go string and a Dart string.
3. **The scalar.** An integer, a float64, a slice, a map, and a struct. Each
   case holds no string and no document, thus it gives the cost of the
   arithmetic and of the memory alone.

The group of the server stays out on purpose. The question of 2026-09-06 is
the cost of a client.

## Run it again

```sh
make build                 # from the root of the repository
./bench/run.sh             # about 1 minute
cat bench/out.csv
```

`DATA` names another document. The default is
`flutter/example/compass/assets/activities.json`, 1,995,467 bytes and 2,723
objects of 10 fields.

The Go runner and the Dart driver call the same functions of `go/cases.go`.
The Dart driver also holds a Dart form of each case, thus the 2 numbers of 1
line come from 1 process and 1 machine state. `js/bench.js` holds the same
cases for bun.

Each case runs for 300 milliseconds, and the number is the mean of the
operations of that window. An ahead of time build runs each program, and bun
runs its own JIT.

### The 3 cases that do not compare

- **`int.hash.1m`.** A Go int and a Dart int each hold 64 bits. A JS number
  holds 53 bits of an integer, thus the bun form reads a BigInt. That number
  measures a BigInt against a machine integer.
- **`str.builder.200k`.** bun joins an array of parts. JavaScript holds no
  builder of bytes.
- **`struct.alloc.1m`.** A Go struct is a value, and a Dart object and a JS
  object each stand on the heap. Go allocates nothing in that loop.

`slice.sort.100k` builds the same sequence on each runtime.
`Math.imul` gives bun the low 32 bits of the product, because a plain multiply
would lose the low bits above 2^53.

## The result of 2026-09-06, late

The load average stood at 9.9, thus each absolute number carries that load.
The ratios stand.

| Case | Go | Dart | gart | bun | gart/Dart | gart/Go |
|---|---|---|---|---|---|---|
| `bytes.count` | 514.04 us | 2.76 ms | **3.61 ms** | — | 1.3 | 7.0 |
| `json.scan` | 978.98 us | — | **13.20 ms** | — | — | 13.5 |
| `json.decode.any` | 6.11 ms | 5.03 ms | **97.82 ms** | 1.34 ms | 19.4 | 16.0 |
| `json.decode.typed` | 3.30 ms | 5.39 ms | **93.88 ms** | 1.35 ms | 17.4 | 28.5 |
| `json.encode.typed` | 1.54 ms | 7.83 ms | **50.25 ms** | 1.68 ms | 6.4 | 32.5 |
| `str.concat.5k` | 5.47 ms | 4.33 ms | **4.41 ms** | 12.62 us | 1.0 | 0.8 |
| `str.builder.200k` | 770.85 us | 2.92 ms | **4.13 ms** | 1.19 ms | 1.4 | 5.4 |
| `str.substr.100k` | 597.73 us | 5.74 ms | **4.47 ms** | 1.15 ms | 0.8 | 7.5 |
| `int.sum.1m` | 253.03 us | 303.04 us | **303.86 us** | 254.71 us | 1.0 | 1.2 |
| `int.hash.1m` | 1.01 ms | 1.00 ms | **1.02 ms** | 22.57 ms | 1.0 | 1.0 |
| `int.div.1m` | 1.01 ms | 1.02 ms | **599.25 us** | 1.10 ms | 0.6 | 0.6 |
| `int.parse.100k` | 416.59 us | 421.26 us | **2.89 ms** | 467.34 us | 6.9 | 6.9 |
| `int.format.100k` | 1.16 ms | 845.42 us | **21.97 ms** | 1.50 ms | 26.0 | 19.0 |
| `float.sum.1m` | 903.46 us | 685.42 us | **3.64 ms** | 660.82 us | 5.3 | 4.0 |
| `float.math.1m` | 752.26 us | 767.34 us | **143.99 ms** | 955.44 us | 187.7 | 191.4 |
| `float.parse.100k` | 1.70 ms | 1.29 ms | **8.35 ms** | 1.47 ms | 6.5 | 4.9 |
| `float.format.100k` | 3.72 ms | 7.14 ms | **25.47 ms** | 3.74 ms | 3.6 | 6.8 |
| `slice.append.1m` | 1.94 ms | 5.57 ms | **25.24 ms** | 2.21 ms | 4.5 | 13.0 |
| `slice.sort.100k` | 4.82 ms | 23.42 ms | **26.62 ms** | 10.56 ms | 1.1 | 5.5 |
| `map.string.100k` | 4.61 ms | 8.99 ms | **58.59 ms** | 8.41 ms | 6.5 | 12.7 |
| `map.int.100k` | 1.61 ms | 5.30 ms | **11.30 ms** | 2.06 ms | 2.1 | 7.0 |
| `struct.alloc.1m` | 703.95 us | 3.45 ms | **4.22 ms** | 345.17 us | 1.2 | 6.0 |

### What the scalar cases say

1. **The integer of gart costs what a Dart int costs, and it holds 64 bits.**
   `GoInt` is an extension type, thus the ahead of time build sees a plain
   `int` and the arithmetic reaches the machine. `int.sum` and `int.hash` each
   sit at 1.0 against Dart and against Go. `int.div` sits at 0.7, because Dart
   holds a faster path than the division of Go.
2. **bun loses the 64 bit integer by a factor of 22.6.** A JS number holds 53
   bits, thus `int.hash.1m` reads a BigInt and it costs 22.57 milliseconds
   against 1.02. This is the 1 case where the ahead of time build of Dart
   stands far ahead of the JIT of bun.
3. **A float64 loop costs 5 times the same loop in Dart.** `float.sum` holds
   no call and no allocation, thus the gap comes from the emitted form of the
   conversion of an integer to a float64.
4. **`float.math.1m` is a defect and not a floor.** Read the section below.
5. **The floor of the slice reaches each case that writes bytes.**
   `int.format`, `float.format`, `int.parse`, `float.parse`, and
   `slice.append` each sit 3.6 to 26 times above Dart, because each 1 writes
   into a `[]byte` and `Slice.array` holds a `List<T>` whose real shape
   varies.
6. **bun stands ahead of Go on `struct.alloc` and near Go on `slice.append`.**
   The JIT holds a hidden class for `Point`, and the nursery of the JS engine
   is faster than the allocator of Dart. This case gives no work to gart.

### The defect of `math.Sqrt`

`float.math.1m` costs 144 milliseconds under gart and 767 microseconds under
Dart. That is a factor of 188, and the cause is 1 file of GOROOT.

`math.Sqrt` calls `math.sqrt`, and the compiler of Go replaces that call with
1 machine instruction. gart holds no such rule, thus it compiles the portable
Go form: a loop of 54 steps that builds the result bit by bit with 64 bit
integer arithmetic, and 2 calls of `Float64bits` and `Float64frombits` around
it. 1 square root thus costs about 140 nanoseconds, and the whole step of
the Dart form costs 0.77.

`internal/overlay/std/math/` already holds 5 files that set `haveArchExp`,
`haveArchLog`, `haveArchFloor`, `haveArchCeil`, `haveArchTrunc`, `haveArchMax`,
and `haveArchMin` to false. Each 1 of those names takes the portable path too,
and `dart:math` holds the fast form of each 1.

## The memo of the decode

A Go string holds UTF-8 bytes and a Dart string holds UTF-16 code units, thus
`toDartString` decodes. Before 2026-09-06 it decoded at each call. It now holds
the result in a field of the `GoString`, because a Go string never changes.

The measurement ran A, B, A: the baseline, the memo, and the baseline again.
The machine drifted 4.5 percent between the 2 baselines, thus the comparison
reads the memo against the second baseline.

| Case | What it holds | Before | After | Factor |
|---|---|---|---|---|
| `cross.frame.500x20` | 500 labels of 20 bytes, 1 read each | 7.65 us | 780 ns | 9.8 |
| `cross.frame.200x500` | 200 labels of 500 bytes, 1 read each | 35.09 us | 327 ns | 107 |
| `cross.todart.100000` | 1 string of 100,000 bytes, read again | 31.72 us | 18 ns | 1,762 |
| `cross.todart.2000000` | 1 string of 2 megabytes, read again | 777.23 us | 18 ns | 43,179 |
| `cross.fresh.20` | a new string at each read | 39 ns | 40 ns | 0.97 |
| `cross.fresh.2000000` | a new string of 2 megabytes | 776.26 us | 766.24 us | 1.01 |

The `cross.fresh` cases give the cost of the field on a string that no code
reads 2 times. That cost stands at 1 to 3 percent, and each case that holds no
boundary moved by less than the drift of the machine.

The memo also holds a memory cost. A string that crosses 1 time holds the
decode for its whole life. A program that reads a document of 100 megabytes
into a string and gives that string to Dart thus needs the memory 2 times.

### The hash of a Go string

The same change reached `hashCode`. `Object.hashAll` builds an iterator of the
byte list, and a Go map calls the getter for each key. The hash now runs 1 time
for each `GoString`, and the mix is FNV-1a with a mask of 26 bits. The mask
holds the product under 2^50, thus a JS number carries it with no loss and the
2 targets give the same hash.

| Case | Before | After | Factor |
|---|---|---|---|
| `map.string.100k` | 76.24 ms | 58.59 ms | 1.30 |
| `json.decode.any` | 104.97 ms | 97.82 ms | 1.07 |
| `json.decode.typed` | 98.71 ms | 93.88 ms | 1.05 |
| `map.int.100k` | 11.58 ms | 11.30 ms | 1.03 |

`map.int.100k` holds no string key, thus it stands as the control.

**The first mix was the Jenkins form, and it crashed dart2wasm.** The type flow
analysis reports `type 'Call' is not a subtype of type 'Parameter' in type
cast`, and the whole wasm build stops. That is the defect of decision 323, and
that decision names a call that fills an optional parameter of a method of
`GoString`. The Jenkins mix holds no parameter at all, thus the rule of
decision 323 is narrower than the defect. A method of `GoString` reaches the
defect, and the body decides. `make test` is the only gate that finds it,
because `TestWebIO`, `TestWebFetch`, and `TestWasm` hold the wasm build.

## What GopherJS does

GopherJS meets the same problem, because a JavaScript string holds UTF-16 code
units too. It takes the opposite decision, and the decision is right for
JavaScript and wrong for Dart.

**A Go string is a JS string, and each code unit holds 1 byte.** A string of
`"héllo"` holds 6 code units, and the 2 code units of the character `é` each
carry 1 byte of the UTF-8 form. `s[i]` is thus `charCodeAt(i)` and `len(s)` is
`s.length`, and each 1 costs nothing.

**The boundary is free for ASCII.** `$externalize` reads `$isASCII(v)`, which
scans the code units, and it gives the same string back when each code unit
stands under 128. A string that holds no ASCII takes a loop of
`$decodeRune` and `String.fromCharCode`. `$internalize` does the same work in
the other direction with `$encodeRune`. **GopherJS holds no memo.** It repeats
the scan at each crossing.

**The trade.** A JS engine holds a rope and a sliced string, thus a
concatenation and a substring cost nothing in that form. Dart holds neither. A
substring of Dart copies, and the measurement above puts `str.substr.100k` at
5.91 milliseconds for Dart and 4.72 for gart, where the substring shares the
byte storage and copies nothing. The byte form of gart also gives `[]byte(s)`
and `append(b, s...)` a share with no loop, and GopherJS builds a `Uint8Array`
with a `charCodeAt` loop for each such conversion.

The memo above gives gart the win of GopherJS at the boundary, and it keeps the
win of the byte form inside the program.

## The earlier results

### The result of 2026-09-05

The machine held a Dart analysis server at 120 percent of 1 core, thus each
absolute number carries that load. The ratios stand.

| Case | Go | Dart | gart | bun |
|---|---|---|---|---|
| `bytes.count`, a byte loop over 2 megabytes | 0.51 ms | 2.63 ms | 5.16 ms | — |
| `json.scan`, the scanner of encoding/json | 0.97 ms | — | 16.4 ms | — |
| `json.decode.any`, into the generic form | 6.12 ms | 5.43 ms | **401.5 ms** | 1.28 ms |
| `json.decode.any.bytes`, the fused decoder | — | 2.32 ms | — | — |
| `json.decode.typed`, into a struct | 3.24 ms | 5.71 ms | **354.3 ms** | 1.36 ms |
| `json.encode.typed` | 1.54 ms | 7.86 ms | **379.6 ms** | 1.69 ms |
| `str.concat.5k`, 5,000 additions | 5.50 ms | 4.43 ms | 4.95 ms | 0.013 ms |
| `str.builder.200k`, 200,000 writes | 0.74 ms | 2.87 ms | **50.3 ms** | 1.08 ms |
| `str.substr.100k`, 100,000 substrings | 0.58 ms | 6.10 ms | **46.3 ms** | 1.12 ms |

#### The result of 2026-09-06, after the overlay work

The same benchmark on the same machine, after the 13 commits of the overlay
work. The load average stood at 10.9, thus each absolute number carries that
load, and the ratio against the run above stands.

| Case | Go | Dart | gart, before | gart, now | bun |
|---|---|---|---|---|---|
| `bytes.count` | 0.53 ms | 2.83 ms | 5.16 ms | **3.59 ms** | — |
| `json.scan` | 1.02 ms | — | 16.4 ms | **13.2 ms** | — |
| `json.decode.any` | 6.28 ms | 5.28 ms | 401.5 ms | **111.6 ms** | 1.34 ms |
| `json.decode.typed` | 3.41 ms | 5.52 ms | 354.3 ms | **101.9 ms** | 1.37 ms |
| `json.encode.typed` | 1.59 ms | 8.10 ms | 379.6 ms | **53.1 ms** | 1.72 ms |
| `str.concat.5k` | 7.59 ms | 4.57 ms | 4.95 ms | 4.71 ms | 0.013 ms |
| `str.builder.200k` | 0.75 ms | 2.99 ms | 50.3 ms | **4.12 ms** | 1.15 ms |
| `str.substr.100k` | 0.57 ms | 6.09 ms | 46.3 ms | **4.65 ms** | 1.11 ms |

The decode of JSON falls by a factor of 3.6, the encode by 7.2, the builder by
12.2, and the substring by 10. A substring now costs less than the `substring`
of Dart itself. The builder costs 1.4 times the `StringBuffer` of Dart, and 5.5
times the builder of Go.

**The largest cause was a string constant.** The emitter wrote
`GoString.fromString('...')` at each read of a Go string constant, thus the
UTF-8 encoder ran again at each evaluation. `reflect.kindByName` holds a switch
of 26 such constants, and the decoder calls it for each value: 32 percent of a
decode sat in that encoder. The emitter hoists each constant to a name of the
file now.

The floor stays. A byte read of a Go slice costs about 7 times the same read in
Go, because `Slice.array` holds a `List<T>` whose real shape varies, thus each
read takes a polymorphic dispatch. A change of the slice representation is the
next step, and no such change stands.

#### What the numbers say

1. **The decoder of `encoding/json` is the hot spot, and not the scanner.**
   The scan of 2 megabytes takes 16.4 milliseconds under gart, and the whole
   decode takes 401.5. The scan is thus 4 percent of the decode. The remainder
   goes to the build of the generic value.
2. **A byte loop under gart costs 2 times the same loop in Dart**, and 10
   times the loop in Go. That is the floor of each case above.
3. **`strings.Builder` costs 250 nanoseconds for each write of 10 bytes.**
   `StringBuffer` of Dart costs 14, and the builder of Go costs 3.6.
4. **A substring costs 466 nanoseconds under gart**, against 5.8 in Go, where
   the slice holds no copy.
5. **The naive `+=` loop is the 1 string case that gart holds.** It matches
   Dart, because both allocate a new string at each step. bun is 390 times
   faster, because the JavaScript engine holds a rope and it joins the parts
   1 time.

#### The boundary of a string

The Dart driver also measures the 2 crossings. A Go string holds UTF-8 bytes
and a Dart string holds UTF-16 code units, thus each crossing converts.

| Length | Go to Dart | Dart to Go |
|---|---|---|
| 20 bytes | 29 ns | 69 ns |
| 1,000 bytes | 330 ns | 1,913 ns |
| 100,000 bytes | 31.4 us | 208.0 us |

The read into Dart costs 0.31 nanoseconds for each byte, and the write into Go
costs 2.08. A screen of 500 labels of 20 characters thus pays 15 microseconds
of the 16,700 microsecond budget of 1 frame at 60 hertz.

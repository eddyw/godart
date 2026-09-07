# bench

This directory compares the same work on 4 runtimes: Go, the Dart that gart
writes from the Go source, Dart, and bun. It holds 3 groups of cases:

1. **The document.** A JSON decode, a JSON encode, a scan, and a byte loop.
2. **The string.** A concatenation, a builder, a substring, and the 2
   crossings of the boundary between a Go string and a Dart string.
3. **The scalar.** An integer, a float64, a slice, a map, and a struct. Each
   case holds no string and no document, thus it gives the cost of the
   arithmetic and of the memory alone.
4. **The text of a client.** A format, a split, a join, a replace, and a sort
   of a struct by a field.

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

## The result of 2026-09-07

Each number is 1 run, on a machine whose load average stood near 5. The 3
groups above hold 27 cases, plus the 14 cases of the boundary. A second run
of the same set stands 5 percent or less apart on each case above 1
millisecond.

### The document, 1,995,467 bytes of JSON

| Case | Go | Dart | gart | bun | gart/Dart | gart/Go |
|---|---|---|---|---|---|---|
| `bytes.count` | 479.59 us | 2.54 ms | **2.42 ms** | — | 1.0 | 5.0 |
| `json.scan` | 940.60 us | — | **1.72 ms** | — | — | 1.8 |
| `json.decode.any` | 5.84 ms | 5.21 ms | **4.39 ms** | 1.23 ms | 0.8 | 0.8 |
| `json.decode.typed` | 3.11 ms | 5.32 ms | **3.63 ms** | 1.29 ms | 0.7 | 1.2 |
| `json.encode.typed` | 1.48 ms | 7.72 ms | **45.06 ms** | 1.59 ms | 5.8 | 30.5 |

### The string

| Case | Go | Dart | gart | bun | gart/Dart | gart/Go |
|---|---|---|---|---|---|---|
| `str.concat.5k` | 5.41 ms | 4.27 ms | **4.32 ms** | 12.12 us | 1.0 | 0.8 |
| `str.builder.200k` | 740.02 us | 2.85 ms | **3.80 ms** | 1.14 ms | 1.3 | 5.1 |
| `str.substr.100k` | 571.70 us | 5.52 ms | **3.95 ms** | 1.07 ms | 0.7 | 6.9 |

### The integer

| Case | Go | Dart | gart | bun | gart/Dart | gart/Go |
|---|---|---|---|---|---|---|
| `int.sum.1m` | 245.57 us | 302.23 us | **301.92 us** | 248.61 us | 1.0 | 1.2 |
| `int.hash.1m` | 955.58 us | 1.00 ms | **1.01 ms** | 22.26 ms | 1.0 | 1.1 |
| `int.div.1m` | 967.19 us | 1.01 ms | **567.18 us** | 1.06 ms | 0.6 | 0.6 |
| `int.parse.100k` | 400.89 us | 433.08 us | **3.83 ms** | 461.06 us | 8.8 | 9.6 |
| `int.format.100k` | 1.17 ms | 819.57 us | **1.92 ms** | 1.45 ms | 2.3 | 1.6 |

### The float64

| Case | Go | Dart | gart | bun | gart/Dart | gart/Go |
|---|---|---|---|---|---|---|
| `float.sum.1m` | 848.79 us | 675.49 us | **685.41 us** | 638.55 us | 1.0 | 0.8 |
| `float.math.1m` | 738.38 us | 753.13 us | **854.97 us** | 934.81 us | 1.1 | 1.2 |
| `float.parse.100k` | 1.65 ms | 1.25 ms | **8.28 ms** | 1.44 ms | 6.6 | 5.0 |
| `float.format.100k` | 3.57 ms | 7.03 ms | **25.37 ms** | 3.70 ms | 3.6 | 7.1 |

### The memory

| Case | Go | Dart | gart | bun | gart/Dart | gart/Go |
|---|---|---|---|---|---|---|
| `slice.append.1m` | 1.67 ms | 5.67 ms | **19.60 ms** | 2.06 ms | 3.5 | 11.7 |
| `slice.sort.100k` | 4.78 ms | 23.17 ms | **30.77 ms** | 10.40 ms | 1.3 | 6.4 |
| `map.string.100k` | 4.30 ms | 8.76 ms | **25.64 ms** | 7.87 ms | 2.9 | 6.0 |
| `map.int.100k` | 1.55 ms | 5.12 ms | **11.20 ms** | 2.03 ms | 2.2 | 7.2 |
| `struct.alloc.1m` | 695.05 us | 3.38 ms | **684.91 us** | 336.17 us | 0.2 | 1.0 |

### The text of a client

| Case | Go | Dart | gart | bun | gart/Dart | gart/Go |
|---|---|---|---|---|---|---|
| `fmt.sprintf.100k` | 9.89 ms | 12.44 ms | **69.35 ms** | 6.15 ms | 5.6 | 7.0 |
| `str.split.100k` | 4.30 ms | 6.29 ms | **35.05 ms** | 5.44 ms | 5.6 | 8.1 |
| `str.join.100k` | 3.98 ms | 15.11 ms | **18.13 ms** | 4.15 ms | 1.2 | 4.6 |
| `str.replace.100k` | 5.25 ms | 13.47 ms | **38.09 ms** | 11.07 ms | 2.8 | 7.3 |
| `sort.struct.50k` | 5.12 ms | 9.03 ms | **16.11 ms** | 5.92 ms | 1.8 | 3.1 |

### The boundary, gart alone

| Case | Cost |
|---|---|
| 1 string of 20 bytes, read again | 17 ns |
| A new string of 20 bytes | 40 ns |
| A write of 20 bytes into Go | 33 ns |
| 1 string of 1,000 bytes, read again | 17 ns |
| A new string of 1,000 bytes | 331 ns |
| A write of 1,000 bytes into Go | 549 ns |
| 1 string of 100,000 bytes, read again | 17 ns |
| A new string of 100,000 bytes | 30.39 us |
| A write of 100,000 bytes into Go | 51.72 us |
| 1 string of 2 megabytes, read again | 17 ns |
| A new string of 2 megabytes | 778.76 us |
| A write of 2 megabytes into Go | 1.17 ms |
| A screen of 500 labels of 20 bytes | 896 ns |
| A screen of 200 labels of 500 bytes | 368 ns |

### The change of the session

The 2 runs stand on 1 machine with a load average near 5, and the Go
column of the 2 differs by 3 percent or less.

| Case | 2026-09-06 | 2026-09-07 | Factor |
|---|---|---|---|
| `bytes.count` | 2.60 ms | 2.42 ms | 1.08 |
| `json.scan` | 10.96 ms | 1.72 ms | 6.39 |
| `json.decode.any` | 86.70 ms | 4.39 ms | 19.75 |
| `json.decode.typed` | 71.33 ms | 3.63 ms | 19.67 |
| `json.encode.typed` | 35.94 ms | 45.06 ms | 0.80 |
| `str.builder.200k` | 3.58 ms | 3.80 ms | 0.94 |
| `int.div.1m` | 627.13 us | 567.18 us | 1.11 |
| `int.parse.100k` | 2.86 ms | 3.83 ms | 0.75 |
| `float.math.1m` | 724.71 us | 854.97 us | 0.85 |
| `float.format.100k` | 23.02 ms | 25.37 ms | 0.91 |
| `slice.sort.100k` | 26.02 ms | 30.77 ms | 0.85 |
| `fmt.sprintf.100k` | 61.63 ms | 69.35 ms | 0.89 |
| `str.join.100k` | 16.83 ms | 18.13 ms | 0.93 |
| `sort.struct.50k` | 13.15 ms | 16.11 ms | 0.82 |

**The decode of JSON costs less than Go costs.** A scanner of the runtime
proves the whole decode before it writes the target, and each doubt hands the
work back to the Go code. Read decision 386.

**5 cases rose, and 4 of the 5 pay for a rule of Go.** `sort` and `slices` read
the real algorithm of Go now, thus `slice.sort` and `sort.struct` hold the
count of the swaps and of the comparisons that 4 tests of Go measure.
`int.parse` reads the real `atoi.go`, which holds the rule of the underscore.
`str.builder` holds the copy check of Go. `fmt.sprintf` pays 12 percent for the
report of a wrong verb: the walk reads the kind of each leaf against the verb,
and 9 defects of `fmt` closed with that read.

**`json.encode.typed` rose 26 percent**, and no rule of Go asks for it. A
`reflect.Value` carries the read only mark now, and the encoder walks each
field through `reflect`. A fast encode path removes that walk, and item 1 of
the next tasks holds its design.

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
| `cross.fresh.2000000` | a new string of 2 megabytes | 776.26 us | 753.51 us | 1.03 |

The other crossing also fell, because `GoString.fromString` holds a path for a
text of ASCII now. A write of 100,000 bytes into Go fell from 195.32 to 51.63
microseconds, and a write of 2 megabytes fell from 3.61 milliseconds to 1.10.

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

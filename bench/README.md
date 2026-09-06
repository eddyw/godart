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

## The result of 2026-09-06, after the third block of the day

Each number is 1 run, on a machine whose load average stood near 7. The 3
groups above hold 27 cases, plus the 14 cases of the boundary.

| Case | Go | Dart | gart | bun | gart/Dart | gart/Go |
|---|---|---|---|---|---|---|
| `bytes.count` | 508.08 us | 2.62 ms | **2.60 ms** | — | 1.0 | 5.1 |
| `json.scan` | 988.16 us | — | **10.78 ms** | — | — | 10.9 |
| `json.decode.any` | 5.68 ms | 4.90 ms | **83.58 ms** | 1.16 ms | 17.1 | 14.7 |
| `json.decode.typed` | 3.05 ms | 5.09 ms | **69.43 ms** | 1.19 ms | 13.6 | 22.7 |
| `json.encode.typed` | 1.43 ms | 7.75 ms | **36.45 ms** | 1.51 ms | 4.7 | 25.4 |
| `str.concat.5k` | 5.62 ms | 3.82 ms | **4.03 ms** | 11.58 us | 1.1 | 0.7 |
| `str.builder.200k` | 734.07 us | 2.75 ms | **3.60 ms** | 1.04 ms | 1.3 | 4.9 |
| `str.substr.100k` | 556.84 us | 5.50 ms | **3.85 ms** | 1.01 ms | 0.7 | 6.9 |
| `int.sum.1m` | 253.26 us | 294.71 us | **293.39 us** | 234.04 us | 1.0 | 1.2 |
| `int.hash.1m` | 909.23 us | 922.78 us | **923.73 us** | 22.34 ms | 1.0 | 1.0 |
| `int.div.1m` | 908.22 us | 987.85 us | **552.87 us** | 1.07 ms | 0.6 | 0.6 |
| `int.parse.100k` | 404.03 us | 450.45 us | **2.80 ms** | 458.33 us | 6.2 | 6.9 |
| `int.format.100k` | 1.11 ms | 778.88 us | **1.82 ms** | 1.47 ms | 2.3 | 1.6 |
| `float.sum.1m` | 804.09 us | 663.23 us | **662.40 us** | 591.34 us | 1.0 | 0.8 |
| `float.math.1m` | 715.73 us | 760.80 us | **807.82 us** | 850.87 us | 1.1 | 1.1 |
| `float.parse.100k` | 1.58 ms | 1.26 ms | **8.19 ms** | 1.42 ms | 6.5 | 5.2 |
| `float.format.100k` | 3.54 ms | 6.86 ms | **23.04 ms** | 3.70 ms | 3.4 | 6.5 |
| `slice.append.1m` | 1.62 ms | 5.73 ms | **19.77 ms** | 1.91 ms | 3.5 | 12.2 |
| `slice.sort.100k` | 4.58 ms | 23.00 ms | **26.00 ms** | 10.37 ms | 1.1 | 5.7 |
| `map.string.100k` | 4.30 ms | 8.52 ms | **22.76 ms** | 7.41 ms | 2.7 | 5.3 |
| `map.int.100k` | 1.53 ms | 5.00 ms | **10.12 ms** | 1.89 ms | 2.0 | 6.6 |
| `struct.alloc.1m` | 663.67 us | 3.28 ms | **661.78 us** | 335.05 us | 0.2 | 1.0 |
| `fmt.sprintf.100k` | 10.42 ms | 12.32 ms | **61.19 ms** | 5.95 ms | 5.0 | 5.9 |
| `str.split.100k` | 4.49 ms | 6.18 ms | **35.70 ms** | 5.52 ms | 5.8 | 7.9 |
| `str.join.100k` | 3.99 ms | 14.99 ms | **16.62 ms** | 4.12 ms | 1.1 | 4.2 |
| `str.replace.100k` | 5.20 ms | 13.43 ms | **37.26 ms** | 11.08 ms | 2.8 | 7.2 |
| `sort.struct.50k` | 5.07 ms | 8.75 ms | **12.84 ms** | 5.90 ms | 1.5 | 2.5 |

### What the numbers say

1. **The integer of gart costs what a Dart int costs, and it holds 64 bits.**
   `GoInt` is an extension type, thus the ahead of time build sees a plain
   `int` and the arithmetic reaches the machine. `int.sum` and `int.hash` each
   sit at 1.0 against Dart and against Go, and `int.div` sits at 0.6.
2. **bun loses the 64 bit integer by a factor of 23.** A JS number holds 53
   bits, thus `int.hash.1m` reads a BigInt and it costs 22.42 milliseconds
   against 0.96. This is the case where the ahead of time build of Dart stands
   far ahead of the JIT of bun.
3. **4 cases now cost less than the same work in Dart.** `struct.alloc` sits at
   0.2, `int.div` at 0.6, and `str.substr` at 0.7. `bytes.count`, `int.sum`,
   `int.hash`, and `float.sum` each sit at 1.0.
4. **`str.join.100k` rose from 11.33 milliseconds to 16.62 on purpose.**
   `strings.Join` joined 1 element at a time, thus its cost stood at the square
   of the result: 2,000 elements took 37.5 milliseconds and they take 0.9 now.
   The case of the benchmark holds 6 elements, and a library must hold no
   square cost. Read decision 379.
4. **The decoder of `encoding/json` holds the largest gap, and it is flat.** A
   profile of 2026-09-06 puts the Go code of that package at 24 percent, the
   read of a slice at 9, and the layer of `reflect` at 8. No single change
   reaches it, thus a large win needs an overlay of that package that reads the
   decoder of Dart.
5. **`fmt.Sprintf` costs 8.8 times the same call in Go.** The overlay parses
   the format and it calls the bridge 1 time for each verb. Each such call
   boxes the value, it decodes the verb, and it parses the verb again.
6. **A scan of a text still allocates.** `str.split` gives a slice of
   substrings, and each substring is 1 Go string with a view of the storage.

### The 4 changes of the third block of 2026-09-06

| Case | Before | After | Factor |
|---|---|---|---|
| `fmt.sprintf.100k` | 88.65 ms | 61.19 ms | 1.45 |
| `str.split.100k` | 48.32 ms | 35.70 ms | 1.35 |
| `str.replace.100k` | 48.26 ms | 37.26 ms | 1.30 |
| `str.join.100k`, 2,000 elements | 37.5 ms | 0.9 ms | 42 |

The bridge of `fmt` takes the parsed verb now: the letter, the flags as bits,
the width, and the precision. It built a Go string of the verb before, and the
runtime parsed that text again with a switch and 3 scans. `Split` and `Replace`
count the parts before the build, and `Grow` of `Builder` held an empty body.

**4 paths of the input and output runtime hold no case of this benchmark.** A
probe of 2026-09-06 measured them ahead of time: a write of a file of 16
megabytes fell from 41.4 milliseconds to 2.0, a read from 34.8 to 0.5, a pipe
of 8 megabytes from 74.7 to 10.3, and a TCP echo of 8 megabytes from 127.0 to
8.1. 6 loops read and wrote 1 byte at a time before.

### The 6 changes of 2026-09-06

| Case | Before | After | Factor |
|---|---|---|---|
| `float.math.1m` | 143.99 ms | 711 us | 202 |
| `int.format.100k` | 21.97 ms | 1.90 ms | 11.6 |
| `struct.alloc.1m` | 3.90 ms | 679 us | 5.7 |
| `float.sum.1m` | 3.43 ms | 693 us | 4.9 |
| `str.split.100k` | 92.14 ms | 48.32 ms | 1.9 |
| `str.replace.100k` | 74.28 ms | 48.26 ms | 1.5 |
| `float.format.100k` | 35.06 ms | 23.57 ms | 1.5 |
| `json.encode.typed` | 52.23 ms | 36.20 ms | 1.4 |
| `json.decode.typed` | 98.71 ms | 70.94 ms | 1.4 |
| `map.string.100k` | 76.24 ms | 23.44 ms | 3.3 |
| `slice.append.1m` | 24.65 ms | 19.74 ms | 1.2 |

1. **`math.Sqrt` compiled to a loop of 54 steps.** The compiler of Go replaces
   `math.sqrt` with 1 machine instruction, and gart held no such rule. `Sqrt`,
   `Floor`, `Ceil`, `Trunc`, and `Abs` read the instruction of Dart now. Read
   decision 359.
2. **`float64(i)` compiled to `int.toDouble`,** a virtual call that the ahead
   of time compiler does not fold. The operator + of int reads a double
   operand and gives the same value. Read decision 360.
3. **`strconv.FormatInt` joined 1 digit at a time to the front.** Read
   decision 361.
4. **A byte slice holds its storage in a field of the type `Uint8List`.**
   `Slice.array` held a `List<T>` whose real class varies, thus each read took
   a dispatch over many targets.
5. **A Go string of ASCII skips the UTF-8 encoder,** and it holds that text as
   its own decode. The encoder held 9 percent of the cost of `fmt.Sprintf`.
6. **The scan of the `strings` overlay reads bytes.** `Index`, `Count`,
   `Split`, `SplitSeq`, and `Replace` built 1 Go string for each position of
   the text.

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

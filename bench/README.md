# bench

This directory compares the same work on 4 runtimes: Go, the Dart that gart
writes from the Go source, Dart, and bun. It answers 2 questions of
2026-09-05: what does the JSON of gart cost against `dart:convert`, and what
does string work cost against Dart, Go, and JavaScript.

## Run it again

```sh
make build                 # from the root of the repository
./bench/run.sh             # about 4 minutes
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

## The result of 2026-09-05

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

### The result of 2026-09-06, after the performance work

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

### What the numbers say

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

### The boundary of a string

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

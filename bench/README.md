# bench

This directory compares the same work on 4 runtimes: Go, the Dart that gart
writes from the Go source, Dart, and bun. It holds 4 groups of cases:

1. **The document.** A JSON decode, a JSON encode, a scan, and a byte loop.
2. **The string.** A concatenation, a builder, a substring, and the 2
   crossings of the boundary between a Go string and a Dart string.
3. **The scalar.** An integer, a float64, a slice, a map, and a struct. Each
   case holds no string and no document, thus it gives the cost of the
   arithmetic and of the memory alone.
4. **The text of a client.** A format, a split, a join, a replace, and a sort
   of a struct by a field.

The fixture measures client workloads. It does not measure a server.

## Run it again

```sh
make build                 # from the root of the repository
./bench/run.sh
cat bench/out.csv
```

`DATA` names another document. The default is
`bench/data/activities.json`, 1,921,945 bytes and 2,723 objects of 10 fields.
The fixture retains the original Compass data byte for byte. `data/LICENSE`
holds its source license.

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

## Results

Record the SDK versions, host, source revision, and machine load with each result.
Keep dated measurements with their validation artifacts. This guide defines
the harness and does not promise a fixed performance ratio.

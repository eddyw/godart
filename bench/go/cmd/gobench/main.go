// Command gobench runs each case on the Go tool chain. It writes 1 CSV line
// for each case: the name, the runtime, the nanoseconds of 1 operation, and
// the count of the operations.
package main

import (
	"fmt"
	"os"
	"time"

	"bench.dev/cases"
)

const budget = 300 * time.Millisecond

func measure(name string, f func() int) {
	f()
	start := time.Now()
	n := 0
	for time.Since(start) < budget {
		f()
		n++
	}
	each := time.Since(start).Nanoseconds() / int64(n)
	fmt.Printf("%s,go,%d,%d\n", name, each, n)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gobench <path of activities.json>")
		os.Exit(2)
	}
	cases.Load(os.Args[1])
	cases.DecodeTyped()

	measure("bytes.count", cases.CountBytes)
	measure("json.scan", cases.Scan)
	measure("json.decode.any", cases.DecodeAny)
	measure("json.decode.typed", cases.DecodeTyped)
	measure("json.encode.typed", cases.EncodeTyped)
	measure("str.concat.5k", func() int { return cases.Concat(5000) })
	measure("str.builder.200k", func() int { return cases.Builder(200000) })
	measure("str.substr.100k", func() int { return cases.Substr(100000) })

	measure("int.sum.1m", func() int { return cases.IntSum(1000000) })
	measure("int.hash.1m", func() int { return cases.IntHash(1000000) })
	measure("int.div.1m", func() int { return cases.IntDiv(1000000) })
	measure("int.parse.100k", func() int { return cases.IntParse(100000) })
	measure("int.format.100k", func() int { return cases.IntFormat(100000) })
	measure("float.sum.1m", func() int { return int(cases.FloatSum(1000000)) })
	measure("float.math.1m", func() int { return int(cases.FloatMath(1000000)) })
	measure("float.parse.100k", func() int { return cases.FloatParse(100000) })
	measure("float.format.100k", func() int { return cases.FloatFormat(100000) })
	measure("slice.append.1m", func() int { return cases.SliceAppend(1000000) })
	measure("slice.sort.100k", func() int { return cases.SliceSort(100000) })
	measure("map.string.100k", func() int { return cases.MapString(100000) })
	measure("map.int.100k", func() int { return cases.MapInt(100000) })
	measure("struct.alloc.1m", func() int { return int(cases.StructAlloc(1000000)) })
}

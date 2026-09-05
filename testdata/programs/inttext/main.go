// Command inttext writes the text of many integers in each base of 2 to 36.
// A run under Go and a run under gart must give the same text.
package main

import (
	"fmt"
	"math"
	"strconv"
)

func values() []int64 {
	out := []int64{
		0, 1, -1, 7, -7, 9, 10, 11, 35, 36, 37, 99, 100, 255, 256,
		999, 1000, 65535, 65536, 1 << 20, 1 << 31, -(1 << 31),
		1 << 53, 1<<53 + 1, 1 << 62,
		math.MaxInt64, math.MinInt64, math.MaxInt64 - 1, math.MinInt64 + 1,
		math.MaxInt32, math.MinInt32,
	}
	x := uint64(0x0123456789ABCDEF)
	for i := 0; i < 200; i++ {
		x = x*6364136223846793005 + 1442695040888963407
		out = append(out, int64(x))
	}
	return out
}

func main() {
	vs := values()
	for _, v := range vs {
		u := uint64(v)
		fmt.Println(strconv.Itoa(int(v)), strconv.FormatInt(v, 10),
			strconv.FormatUint(u, 10))
		for base := 2; base <= 36; base++ {
			fmt.Printf("%d %s %s\n", base,
				strconv.FormatInt(v, base), strconv.FormatUint(u, base))
		}
		fmt.Printf("%s|%s\n",
			strconv.AppendInt([]byte("x:"), v, 16),
			strconv.AppendUint([]byte("y:"), u, 16))
		fmt.Printf("%d %x %o %b %v\n", v, v, v, v, v)
		fmt.Printf("%d %x %o %b\n", u, u, u, u)
		fmt.Println(strconv.Quote(strconv.Itoa(int(v))))
	}
	// A base outside the range panics with the text of Go.
	for _, f := range []func(){
		func() { fmt.Println(strconv.FormatInt(7, 1)) },
		func() { fmt.Println(strconv.FormatUint(7, 1)) },
		func() { fmt.Println(strconv.FormatInt(7, 37)) },
		func() { fmt.Println(strconv.FormatUint(7, 37)) },
		func() { fmt.Println(strconv.AppendInt(nil, 7, 0)) },
		func() { fmt.Println(strconv.AppendUint(nil, 7, 40)) },
	} {
		func() {
			defer func() { fmt.Println("recovered:", recover()) }()
			f()
		}()
	}
}

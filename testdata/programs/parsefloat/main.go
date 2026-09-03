// parsefloat reads each shape of a float text that Go accepts.
package main

import (
	"fmt"
	"math"
	"strconv"
)

var texts = []string{
	"1.5", "-1.5", "+2", "0", ".5", "5.", "1e10", "1E-10", "1.5e+3",
	"Inf", "inf", "+Inf", "-Inf", "INF", "Infinity", "-infinity",
	"NaN", "nan", "NAN",
	"1_000.5", "1_0e1_0",
	"0x1p4", "0x1.8p1", "0X1P-2", "-0x1p0",
	"1e400", "-1e400", "1e-400",
	"", " 1", "1 ", "abc", "1.2.3", "0x1", "--1", "1e", "_1", "1__0",
}

func main() {
	for _, s := range texts {
		v, err := strconv.ParseFloat(s, 64)
		fmt.Printf("%q %v %v\n", s, v, err)
	}
	for _, s := range []string{"1.5", "3.4e39", "-3.4e39", "Inf"} {
		v, err := strconv.ParseFloat(s, 32)
		fmt.Printf("32 %q %v %v\n", s, v, err)
	}
	v, err := strconv.ParseFloat("NaN", 64)
	fmt.Println("nan is not equal to itself:", v != v, err)
	fmt.Println("the largest float:", math.MaxFloat64)
}

// mapprint reads the order that fmt gives to the entries of a map.
package main

import (
	"fmt"
	"math"
)

type key struct {
	A int
	B string
}

func main() {
	fmt.Println(map[int]string{0: "zero", 1: "one", 2: "two", 3: "three", 1000: "K", -5: "neg"})
	fmt.Println(map[uint64]int{0: 1, 18446744073709551615: 2, 5: 3})
	fmt.Println(map[string]int{"b": 1, "a": 2, "B": 3, "": 4, "aa": 5})
	fmt.Println(map[float64]int{2.5: 1, -1: 2, math.NaN(): 3, 0: 4, math.Inf(1): 5})
	fmt.Println(map[bool]int{true: 1, false: 2})
	fmt.Println(map[key]int{{2, "a"}: 1, {1, "z"}: 2, {1, "a"}: 3})
	fmt.Println(map[int8]int{-128: 1, 127: 2, 0: 3})
	fmt.Printf("%v %d\n", map[int]int{10: 1, 2: 2}, map[int]int{10: 1, 2: 2})
}

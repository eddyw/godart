package main

import "fmt"

type intType interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type plainType interface {
	~int8 | ~int16 | ~int32 | ~int64
}

type wideType interface {
	~uint | ~uint64 | ~uintptr
}

// below reads the type set of math/rand/v2. It holds a signed term and an
// unsigned term of 64 bits, thus the order comes from the dictionary.
func below[Int intType](n Int) Int {
	if n <= 0 {
		panic("bad")
	}
	return Int(uint64(n) - 1)
}

func arith[Int intType](a, b Int) []Int {
	return []Int{a + b, a - b, a * b, a & b, a | b, a ^ b, a &^ b, a << 3, -a, ^a}
}

func ratio[Int plainType](a, b Int) (Int, Int, Int) {
	return a / b, a % b, a >> 2
}

func wide[Int wideType](a, b Int) (Int, Int, Int, bool) {
	return a / b, a % b, a >> 2, a < b
}

type counter[Int intType] struct {
	n Int
}

func (c *counter[Int]) add(v Int) { c.n += v }

func tally[Int intType](vs []Int) Int {
	seen := map[Int]bool{}
	c := &counter[Int]{}
	for _, v := range vs {
		if !seen[v] {
			seen[v] = true
			c.add(v)
		}
	}
	return c.n
}

func main() {
	fmt.Println(below[int8](100), below[uint8](200), below[int](7), below[uint64](1))
	fmt.Println(arith[int8](100, 7))
	fmt.Println(arith[uint8](200, 7))
	fmt.Println(arith[uint64](3, 7))
	fmt.Println(arith[int32](70000, 3))
	fmt.Println(ratio[int8](-100, 7))
	fmt.Println(ratio[int64](-100, 7))
	fmt.Println(wide[uint64](18446744073709551615, 7))
	fmt.Println(wide[uint](40, 7))
	fmt.Println(tally[uint8]([]uint8{200, 200, 100}), tally[int]([]int{1, 2, 3}))
	fmt.Println(string(rune(65)), fmt.Sprint(below[uint16](300)))
}

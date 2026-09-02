package main

import (
	"crypto/subtle"
	"fmt"
)

func main() {
	fmt.Println(subtle.ConstantTimeCompare([]byte("secret"), []byte("secret")))
	fmt.Println(subtle.ConstantTimeCompare([]byte("secret"), []byte("secreT")))
	fmt.Println(subtle.ConstantTimeCompare([]byte("secret"), []byte("secrets")))
	fmt.Println(subtle.ConstantTimeCompare(nil, nil))

	fmt.Println(subtle.ConstantTimeSelect(1, 10, 20))
	fmt.Println(subtle.ConstantTimeSelect(0, 10, 20))
	fmt.Println(subtle.ConstantTimeSelect(1, -1, 0))

	for _, p := range [][2]uint8{{0, 0}, {1, 1}, {0, 1}, {255, 255}, {255, 0}} {
		fmt.Print(subtle.ConstantTimeByteEq(p[0], p[1]), " ")
	}
	fmt.Println()

	for _, p := range [][2]int32{{0, 0}, {1, 1}, {1, 2}, {-1, -1}, {-1, 1},
		{2147483647, 2147483647}, {-2147483648, 2147483647}} {
		fmt.Print(subtle.ConstantTimeEq(p[0], p[1]), " ")
	}
	fmt.Println()

	for _, p := range [][2]int{{0, 0}, {0, 1}, {1, 0}, {5, 5}, {5, 6}, {6, 5},
		{0, 2147483647}, {2147483647, 0}} {
		fmt.Print(subtle.ConstantTimeLessOrEq(p[0], p[1]), " ")
	}
	fmt.Println()

	x := []byte("aaaa")
	subtle.ConstantTimeCopy(0, x, []byte("bbbb"))
	fmt.Printf("%s\n", x)
	subtle.ConstantTimeCopy(1, x, []byte("bbbb"))
	fmt.Printf("%s\n", x)

	dst := make([]byte, 4)
	fmt.Println(subtle.XORBytes(dst, []byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}), dst)
	fmt.Println(subtle.XORBytes(dst, []byte{1, 2}, []byte{5, 6, 7, 8}), dst)
	fmt.Println(subtle.XORBytes(dst, nil, []byte{5}))

	fmt.Println(shortDst())
	fmt.Println(unequalCopy())

	sum := 0
	subtle.WithDataIndependentTiming(func() { sum = 7 })
	fmt.Println(sum)
}

// shortDst gives the panic text of a destination that holds too few bytes.
func shortDst() (text string) {
	defer func() { text = fmt.Sprint(recover()) }()
	subtle.XORBytes(make([]byte, 1), []byte{1, 2}, []byte{3, 4})
	return "no panic"
}

// unequalCopy gives the panic text of a copy of 2 slices of different length.
func unequalCopy() (text string) {
	defer func() { text = fmt.Sprint(recover()) }()
	subtle.ConstantTimeCopy(1, make([]byte, 2), make([]byte, 3))
	return "no panic"
}

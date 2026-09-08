// The witness of the scan family of fmt.
package main

import (
	"fmt"
	"strings"
)

func main() {
	var c complex128
	n, err := fmt.Sscanf("(1+2i)", "%v", &c)
	fmt.Println("complex", n, err, c)

	var u8 uint8
	n, err = fmt.Sscanf("255", "%q", &u8)
	fmt.Println("q of uint8", n, err)

	var a, b int
	n, err = fmt.Sscanf("1 2", "%d\n%d", &a, &b)
	fmt.Println("newline", n, err)

	var t bool
	n, err = fmt.Sscanf("maybe", "%t", &t)
	fmt.Println("bool", n, err)

	var u82 uint8
	n, err = fmt.Sscanf("300", "%d", &u82)
	fmt.Println("overflow", n, err)

	var i int
	n, err = fmt.Sscanf("1_000", "%v", &i)
	fmt.Println("underscore", n, err, i)

	n, err = fmt.Sscanf("5", "%z", &i)
	fmt.Println("bad verb", n, err)

	n, err = fmt.Sscanf("5", "%d", i)
	fmt.Println("no pointer", n, err)

	r := strings.NewReader("one\ntwo\n")
	var s1, s2 string
	n, err = fmt.Fscanln(r, &s1)
	fmt.Println("first line", n, err, s1)
	n, err = fmt.Fscanln(r, &s2)
	fmt.Println("second line", n, err, s2)

	var x, y int
	n, err = fmt.Sscan("3 4", &x, &y)
	fmt.Println("scan", n, err, x, y)

	var f float64
	n, err = fmt.Sscanf("2.5rest", "%f", &f)
	fmt.Println("float", n, err, f)

	var w1, w2 string
	n, err = fmt.Sscanf("a b", "%s %s", &w1, &w2)
	fmt.Println("words", n, err, w1, w2)

	var hex int
	n, err = fmt.Sscanf("ff", "%x", &hex)
	fmt.Println("hex", n, err, hex)

	var ch rune
	n, err = fmt.Sscanf("k", "%c", &ch)
	fmt.Println("rune", n, err, string(ch))
}

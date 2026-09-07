// The program proves the scan family of the package fmt. A second scan over 1
// reader finds the input that the first scan left, each error text follows Go,
// and a complex value reads its 2 float tokens.
package main

import (
	"fmt"
	"strings"
)

func main() {
	r := strings.NewReader("1 2\n3 4\nabc def\n")
	var a, b int
	n, err := fmt.Fscanln(r, &a, &b)
	fmt.Println("line1", n, a, b, err)
	n, err = fmt.Fscanln(r, &a, &b)
	fmt.Println("line2", n, a, b, err)
	var s, t string
	n, err = fmt.Fscanln(r, &s, &t)
	fmt.Println("line3", n, s, t, err)
	n, err = fmt.Fscanln(r, &s)
	fmt.Println("line4", n, err)

	r2 := strings.NewReader("5 6 7")
	var x int
	for i := 0; i < 4; i++ {
		n, err = fmt.Fscan(r2, &x)
		fmt.Println("scan", i, n, x, err)
	}
	var u uint8
	n, err = fmt.Sscanf("300", "%d", &u)
	fmt.Println(n, err)
	var i2 int
	n, err = fmt.Sscanf("1_000", "%v", &i2)
	fmt.Println(n, err, i2)
	var bo bool
	n, err = fmt.Sscanf("maybe", "%t", &bo)
	fmt.Println(n, err)
	n, err = fmt.Sscanf("5", "%z", &i2)
	fmt.Println(n, err)
	n, err = fmt.Sscanf("5", "%d", i2)
	fmt.Println(n, err)
	n, err = fmt.Sscanf("1 2", "%d\n%d", &a, &b)
	fmt.Println(n, err)
	var c complex128
	n, err = fmt.Sscanf("(1+2i)", "%v", &c)
	fmt.Println(n, err, c)
	n, err = fmt.Sscanf("255", "%q", &u)
	fmt.Println(n, err)

	n, err = fmt.Sscanf("(3.5-2e3i)", "%g", &c)
	fmt.Println(n, err, c)
	n, err = fmt.Sscanf("1+2i", "%v", &c)
	fmt.Println(n, err, c)
	var f32 float32
	n, err = fmt.Sscanf("0x1p-2", "%v", &f32)
	fmt.Println(n, err, f32)
	n, err = fmt.Sscan("0b101 0o17 0xff", &i2)
	fmt.Println(n, err, i2)
	var s1, s2 string
	r3 := strings.NewReader("a b\nc d\n")
	n, err = fmt.Fscan(r3, &s1, &s2)
	fmt.Println(n, err, s1, s2)
	n, err = fmt.Fscan(r3, &s1, &s2)
	fmt.Println(n, err, s1, s2)
	n, err = fmt.Sscanf("true false", "%t %t", &bo, &bo)
	fmt.Println(n, err)
	n, err = fmt.Sscanf("T F", "%t %t", &bo, &bo)
	fmt.Println(n, err)
	n, err = fmt.Sscanf("maybe", "%v", &bo)
	fmt.Println(n, err)
	n, err = fmt.Sscanf("1_0_0", "%v", &i2)
	fmt.Println(n, err, i2)
	n, err = fmt.Sscanf("1_000", "%d", &i2)
	fmt.Println(n, err, i2)
}

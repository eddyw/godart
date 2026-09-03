// Program ptreq shows that a pointer compares by the location that it holds.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type point struct {
	X int
	Y int
}

type holder struct {
	P *point
	N int
}

type counter struct {
	n int
}

func main() {
	p1 := &point{X: 1, Y: 2}
	p2 := &point{X: 1, Y: 2}
	p3 := p1

	fmt.Println("2 pointers to equal structs:", p1 == p2)
	fmt.Println("1 pointer with itself:", p1 == p3)
	fmt.Println("the values behind them:", *p1 == *p2)

	var a any = p1
	var b any = p2
	var c any = p3
	fmt.Println("in an interface, 2 locations:", a == b)
	fmt.Println("in an interface, 1 location:", a == c)

	byLocation := map[*point]string{}
	byLocation[p1] = "first"
	byLocation[p2] = "second"
	fmt.Println("map entries for 2 equal structs:", len(byLocation))
	fmt.Println("the entry of the first location:", byLocation[p1])

	h1 := holder{P: p1, N: 7}
	h2 := holder{P: p2, N: 7}
	h3 := holder{P: p3, N: 7}
	fmt.Println("a struct with a pointer field, 2 locations:", h1 == h2)
	fmt.Println("a struct with a pointer field, 1 location:", h1 == h3)

	var i1 any = h1
	var i2 any = h2
	var i3 any = h3
	fmt.Println("that struct in an interface, 2 locations:", i1 == i2)
	fmt.Println("that struct in an interface, 1 location:", i1 == i3)

	byHolder := map[holder]int{}
	byHolder[h1] = 1
	byHolder[h2] = 2
	byHolder[h3] = 3
	fmt.Println("map entries for 2 holders:", len(byHolder), byHolder[h1])

	n1 := 5
	n2 := 5
	q1 := &n1
	q2 := &n2
	q3 := &n1
	fmt.Println("2 pointers to equal ints:", q1 == q2)
	fmt.Println("2 pointers to 1 int:", q1 == q3)
	var v1 any = q1
	var v2 any = q2
	var v3 any = q3
	fmt.Println("in an interface, 2 int locations:", v1 == v2)
	fmt.Println("in an interface, 1 int location:", v1 == v3)

	c1 := counter{n: 3}
	c2 := counter{n: 3}
	f1 := &c1.n
	f2 := &c2.n
	f3 := &c1.n
	fmt.Println("2 pointers to equal fields:", f1 == f2)
	fmt.Println("2 pointers to 1 field:", f1 == f3)
	var w1 any = f1
	var w2 any = f2
	var w3 any = f3
	fmt.Println("in an interface, 2 field locations:", w1 == w2)
	fmt.Println("in an interface, 1 field location:", w1 == w3)

	arr := [3]int{1, 2, 3}
	e1 := &arr[1]
	e2 := &arr[2]
	e3 := &arr[1]
	fmt.Println("2 pointers to elements:", e1 == e2, e1 == e3)

	readStrings()
	readFile()
}

// readStrings reads a strings.Reader to the end. The error of the end is the
// value io.EOF.
func readStrings() {
	r := strings.NewReader("ab")
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	fmt.Println("the first read of a strings.Reader:", n, err == nil)
	n, err = r.Read(buf)
	fmt.Println("the second read of a strings.Reader:", n, err == io.EOF)
	_, _, err = r.ReadRune()
	fmt.Println("ReadRune at the end:", err == io.EOF)
	_, err = r.ReadByte()
	fmt.Println("ReadByte at the end:", err == io.EOF)
	var boxed any = err
	fmt.Println("the error in an interface:", boxed == any(io.EOF))
}

// readFile reads an os.File to the end. The error of the end is the value
// io.EOF.
func readFile() {
	dir, err := os.MkdirTemp("", "ptreq")
	if err != nil {
		fmt.Println("MkdirTemp:", err)
		return
	}
	defer os.RemoveAll(dir)
	path := dir + "/a.txt"
	if err := os.WriteFile(path, []byte("ab"), 0o644); err != nil {
		fmt.Println("WriteFile:", err)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("Open:", err)
		return
	}
	defer f.Close()
	buf := make([]byte, 4)
	n, err := f.Read(buf)
	fmt.Println("the first read of an os.File:", n, err == nil)
	n, err = f.Read(buf)
	fmt.Println("the second read of an os.File:", n, err == io.EOF)
	var boxed any = err
	fmt.Println("the error in an interface:", boxed == any(io.EOF))
}

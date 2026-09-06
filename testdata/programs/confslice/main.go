// The program proves the slice rules of Go: an append that keeps the storage
// against an append that builds new storage, the 3 index form, a copy of 2
// ranges that overlap, a slice of an array, clear, and the difference between
// a nil slice and an empty slice.
package main

import "fmt"

func row(label string, value any) {
	fmt.Printf("%-26s %v\n", label, value)
}

// shareOrGrow proves that an append inside the capacity writes the storage of
// the source, and that an append above the capacity leaves it alone.
func shareOrGrow() {
	base := make([]int, 5, 10)
	for i := range base {
		base[i] = i
	}
	head := base[:3]
	row("base", base)
	row("head", head)
	row("len cap of head", fmt.Sprint(len(head), cap(head)))
	grown := append(head, 99)
	row("after append", base)
	row("grown", grown)
	row("base[3]", base[3])

	small := make([]int, 3, 3)
	copy(small, []int{1, 2, 3})
	other := append(small, 4)
	other[0] = 100
	row("small after grow", small)
	row("other", other)

	// A slice of the full capacity shares each element.
	full := base[:cap(base)]
	full[9] = 42
	row("full", full)
	row("base after write", base)

	// An append of a slice of a slice.
	a := []int{1, 2, 3}
	b := append(a[:1], a[2:]...)
	row("a after append", a)
	row("b", b)
}

// threeIndex proves the 3 index form, which caps the capacity.
func threeIndex() {
	base := make([]int, 8, 16)
	for i := range base {
		base[i] = i
	}
	cut := base[2:5:6]
	row("cut", cut)
	row("len cap of cut", fmt.Sprint(len(cut), cap(cut)))
	cut = append(cut, 100)
	row("after 1 append", base)
	row("cut", cut)
	row("cap of cut", cap(cut))
	next := append(cut, 200)
	row("after 2 appends", base)
	row("next", next)
	row("cut still", cut)

	empty := base[3:3:3]
	row("empty", empty)
	row("len cap of empty", fmt.Sprint(len(empty), cap(empty)))
	row("empty is nil", empty == nil)
	row("append to empty", append(empty, 7))
	row("base after that", base)
}

// copies proves that copy takes the smaller length, and that it reads the
// source before it writes the target when the 2 ranges overlap.
func copies() {
	src := []int{1, 2, 3, 4, 5, 6}
	dst := make([]int, 3)
	row("n of copy", copy(dst, src))
	row("dst", dst)

	long := make([]int, 8)
	row("n of short src", copy(long, src))
	row("long", long)

	// The 2 ranges overlap. The result must read the whole source first.
	a := []int{1, 2, 3, 4, 5, 6}
	copy(a[1:], a[:5])
	row("shift right", a)
	b := []int{1, 2, 3, 4, 5, 6}
	copy(b[:5], b[1:])
	row("shift left", b)
	c := []int{1, 2, 3, 4, 5, 6}
	copy(c, c)
	row("copy of itself", c)

	// copy also reads a string into a byte slice.
	buf := make([]byte, 4)
	row("n of string copy", copy(buf, "abcdef"))
	row("buf", string(buf))
	short := make([]byte, 8)
	row("n of short target", copy(short, "abc"))
	row("short", short)

	var nilDst []int
	var nilSrc []int
	row("copy into nil", copy(nilDst, src))
	row("copy from nil", copy(dst, nilSrc))
}

// arrays proves a slice of an array and of a pointer to an array.
func arrays() {
	var arr [6]int
	for i := range arr {
		arr[i] = i * 10
	}
	s := arr[1:4]
	row("s", s)
	row("len cap of s", fmt.Sprint(len(s), cap(s)))
	s[0] = 999
	row("arr after write", arr)

	// An array is a value, thus a copy holds its own storage.
	copyOfArr := arr
	copyOfArr[0] = -1
	row("arr", arr)
	row("copy", copyOfArr)
	row("arr == copy", arr == copyOfArr)
	row("arr == arr", arr == arr)

	p := &arr
	ps := p[2:5]
	ps[0] = 777
	row("through pointer", arr)
	row("len cap of ps", fmt.Sprint(len(ps), cap(ps)))
	row("full slice", arr[:])

	// A conversion of a slice to an array.
	src := []int{1, 2, 3, 4}
	var out [4]int = [4]int(src)
	src[0] = 100
	row("array of slice", out)
	row("array of prefix", [2]int(src))
	pa := (*[4]int)(src)
	pa[1] = 55
	row("pointer to array", src)

	// A 2 dimension array.
	var grid [2][3]int
	grid[1][2] = 5
	row("grid", grid)
	row("grid row", grid[1])
	row("len of grid", len(grid))
	row("len of row", len(grid[0]))
}

// clears proves the builtin clear on a slice.
func clears() {
	s := []int{1, 2, 3}
	clear(s)
	row("cleared ints", s)
	t := []string{"a", "b"}
	clear(t)
	row("cleared strings", fmt.Sprintf("%q", t))
	type pt struct{ X, Y int }
	u := []pt{{1, 2}, {3, 4}}
	clear(u)
	row("cleared structs", u)
	var v []int
	clear(v)
	row("cleared nil", v)
	part := make([]int, 5)
	for i := range part {
		part[i] = i + 1
	}
	clear(part[1:3])
	row("cleared range", part)
	var f []float64 = []float64{1, 2}
	clear(f)
	row("cleared floats", f)
}

// nilOrEmpty proves the difference between a nil slice and an empty slice.
func nilOrEmpty() {
	var a []int
	b := []int{}
	c := make([]int, 0)
	d := make([]int, 0, 5)
	row("a nil", a == nil)
	row("b nil", b == nil)
	row("c nil", c == nil)
	row("d nil", d == nil)
	row("lens", fmt.Sprint(len(a), len(b), len(c), len(d)))
	row("caps", fmt.Sprint(cap(a), cap(b), cap(c), cap(d)))
	row("a printed", fmt.Sprintf("%v %#v", a, a))
	row("b printed", fmt.Sprintf("%v %#v", b, b))
	row("append to nil", append(a, 1))
	row("a after append", a)
	row("range over nil", rangeCount(a))
	row("slice of nil", a[:0])
	row("nil of nil is nil", a[:0] == nil)
	var e []int
	e = append(e, a...)
	row("append nil to nil", e == nil)
	f := append(b, c...)
	row("append empty", f == nil)
}

func rangeCount(s []int) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// growth proves the length after a chain of appends. The capacity of Go is
// implementation-defined, thus the program reads the length alone.
func growth() {
	var s []int
	lens := []int{}
	for i := 0; i < 20; i++ {
		s = append(s, i)
		lens = append(lens, len(s))
	}
	row("lens", lens)
	row("last", s[len(s)-1])
	row("cap at least len", cap(s) >= len(s))
	// An append of many values at 1 time.
	t := append([]int{1}, 2, 3, 4)
	row("append many", t)
	u := append([]int(nil), t...)
	row("append spread", u)
	u[0] = 9
	row("t after write", t)
	// An append of a string into a byte slice.
	bs := append([]byte("ab"), "cd"...)
	row("append text", string(bs))
}

func main() {
	fmt.Println("== the storage ==")
	shareOrGrow()
	fmt.Println("== the 3 index form ==")
	threeIndex()
	fmt.Println("== the copy ==")
	copies()
	fmt.Println("== the array ==")
	arrays()
	fmt.Println("== clear ==")
	clears()
	fmt.Println("== nil against empty ==")
	nilOrEmpty()
	fmt.Println("== the growth ==")
	growth()
}

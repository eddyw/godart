// The program writes the Go syntax form of the verb %#v for each kind that
// the runtime walks.
package main

import "fmt"

type point struct {
	X int
	Y int
}

type label struct {
	Name string
	tag  byte
}

type outer struct {
	P    point
	L    label
	List []int
	M    map[string]int
}

type kind int

type flags uint8

type kinds []kind

type holder struct {
	V   any
	E   error
	P   *point
	B   []byte
	A   [2]int
	Ok  bool
	F   float64
	U   uint16
	R   rune
	Set map[kind]bool
}

// note holds the method GoString, thus the verb %#v reads that method.
type note struct {
	Text string
}

func (n note) GoString() string { return "note(" + n.Text + ")" }

// shout holds the method String, and the verb %#v never reads that method.
type shout struct {
	Text string
}

func (s shout) String() string { return "SHOUT " + s.Text }

func main() {
	p := point{3, 4}
	fmt.Printf("struct     %#v\n", p)
	fmt.Printf("nested     %#v\n", outer{p, label{"a", 7}, []int{1, 2}, map[string]int{"b": 2, "a": 1}})
	fmt.Printf("ptrstruct  %#v\n", &p)
	fmt.Printf("slice      %#v\n", []int{1, 2, 3})
	fmt.Printf("nilslice   %#v\n", []int(nil))
	fmt.Printf("emptyslice %#v\n", []int{})
	fmt.Printf("bytes      %#v\n", []byte("hi"))
	fmt.Printf("nilbytes   %#v\n", []byte(nil))
	fmt.Printf("map        %#v\n", map[string]int{"b": 2, "a": 1})
	fmt.Printf("nilmap     %#v\n", map[string]int(nil))
	fmt.Printf("emptymap   %#v\n", map[string]int{})
	fmt.Printf("string     %#v\n", "hi\tthere\n")
	fmt.Printf("quoted     %#v\n", `a "b" \c`)
	fmt.Printf("kind       %#v\n", kind(3))
	fmt.Printf("kinds      %#v\n", kinds{1, 2})
	fmt.Printf("iface      %#v\n", any(p))
	fmt.Printf("nilptr     %#v\n", (*point)(nil))
	fmt.Printf("array      %#v\n", [3]int{1, 2, 3})
	fmt.Printf("bool       %#v\n", true)
	fmt.Printf("int        %#v\n", 7)
	fmt.Printf("negint     %#v\n", -7)
	fmt.Printf("uint       %#v\n", uint(255))
	fmt.Printf("float      %#v\n", 1.5)
	fmt.Printf("rune       %#v\n", 'a')
	fmt.Printf("holder     %#v\n", holder{
		V:   p,
		E:   nil,
		P:   nil,
		B:   []byte("hi"),
		A:   [2]int{5, 6},
		Ok:  true,
		F:   2.25,
		U:   9,
		R:   'z',
		Set: map[kind]bool{2: true, 1: false},
	})
	fmt.Printf("gostring   %#v\n", note{"one"})
	fmt.Printf("gostrptr   %#v\n", &note{"two"})
	fmt.Printf("stringer   %#v\n", shout{"loud"})
	fmt.Printf("slicestruct %#v\n", []point{{1, 2}, {3, 4}})
	fmt.Printf("sliceptr   %#v\n", []*point{nil})
	fmt.Printf("mapslice   %#v\n", map[string][]int{"a": {1, 2}})
	fmt.Printf("mapstruct  %#v\n", map[int]point{1: {1, 2}})
	fmt.Printf("flags      %#v\n", flags(3))
	var ch chan int
	var fn func()
	fmt.Printf("nilchan    %#v %v\n", ch, ch)
	fmt.Printf("nilfunc    %#v %v\n", fn, fn)
	fmt.Printf("plainv     %v %+v\n", p, p)
}

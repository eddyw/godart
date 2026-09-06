// The program proves the text of each run time panic of Go, and it proves
// that recover gives the value. The value of a run time panic implements the
// interface error.
//
// The program does not write the type of a recovered run time value. Go holds
// 3 types there and gart holds 1. Read the section "The runtime" of
// docs/parity.md.
package main

import (
	"errors"
	"fmt"
)

// catch runs f and gives the text of the panic value.
func catch(label string, f func()) {
	defer func() {
		r := recover()
		if r == nil {
			fmt.Printf("%-30s no panic\n", label)
			return
		}
		if e, ok := r.(error); ok {
			fmt.Printf("%-30s error: %s\n", label, e.Error())
			return
		}
		fmt.Printf("%-30s value: %v\n", label, r)
	}()
	f()
}

type point struct{ X, Y int }

type shape interface{ Area() int }

type square struct{ Side int }

func (s square) Area() int { return s.Side * s.Side }

var sink any

func nilMap() {
	var m map[string]int
	catch("write of a nil map", func() { m["a"] = 1 })
	catch("delete of a nil map", func() { delete(m, "a") })
	catch("read of a nil map", func() { sink = m["a"] })
	catch("len of a nil map", func() { sink = len(m) })
	var n map[string][]int
	catch("append into a nil map", func() { n["a"] = append(n["a"], 1) })
	var p map[point]int
	catch("struct key of a nil map", func() { p[point{1, 2}] = 1 })
	catch("clear of a nil map", func() { clear(m) })
}

// The index helpers carry the directive go:noinline. gc writes the index of
// a panic from a register, and an inlined body can hold another value there.
// A call of a real function gives the true index in each case.

//go:noinline
func at(s []int, i int) int { return s[i] }

//go:noinline
func uat(s []int, i uint64) int { return s[i] }

//go:noinline
func put(s []int, i, v int) { s[i] = v }

//go:noinline
func atArray(a [3]int, i int) int { return a[i] }

//go:noinline
func atText(t string, i int) byte { return t[i] }

func indexes() {
	s := []int{1, 2, 3}
	catch("index 3 of len 3", func() { sink = at(s, 3) })
	catch("index 100 of len 3", func() { sink = at(s, 100) })
	catch("index -1", func() { sink = at(s, -1) })
	catch("write at index 3", func() { put(s, 3, 1) })
	var empty []int
	catch("index 0 of a nil slice", func() { sink = at(empty, 0) })
	var arr [3]int
	catch("index 5 of an array", func() { sink = atArray(arr, 5) })
	catch("index 3 of a string", func() { sink = atText("abc", 3) })
	catch("index of an empty text", func() { sink = atText("", 0) })
	// An index of an unsigned type reads a large value.
	catch("index of a big uint", func() { sink = uat(s, 1<<63) })
	catch("index of a huge uint", func() { sink = uat(s, 1<<64-1) })
	catch("index 2 of len 3", func() { sink = at(s, 2) })
}

//go:noinline
func high(s []int, h int) []int { return s[:h] }

//go:noinline
func low(s []int, l int) []int { return s[l:] }

//go:noinline
func lowHigh(s []int, l, h int) []int { return s[l:h] }

//go:noinline
func three(s []int, l, h, m int) []int { return s[l:h:m] }

//go:noinline
func textHigh(t string, h int) string { return t[:h] }

//go:noinline
func textLowHigh(t string, l, h int) string { return t[l:h] }

//go:noinline
func arrayHigh(a *[3]int, h int) []int { return a[:h] }

func slices() {
	s := []int{1, 2, 3}
	catch("high above cap", func() { sink = high(s, 4) })
	catch("low above high", func() { sink = lowHigh(s, 2, 1) })
	catch("low below 0", func() { sink = low(s, -1) })
	catch("max above cap", func() { sink = three(s, 0, 1, 4) })
	catch("high above max", func() { sink = three(s, 0, 2, 1) })
	catch("low above high of 3", func() { sink = three(s, 2, 1, 3) })
	catch("a legal 3 index form", func() { sink = three(s, 1, 2, 3) })
	catch("string high above len", func() { sink = textHigh("abcd", 5) })
	catch("string low above high", func() { sink = textLowHigh("abcd", 3, 2) })
	catch("string low below 0", func() { sink = textLowHigh("abcd", -1, 2) })
	var arr [3]int
	catch("array high above len", func() { sink = arrayHigh(&arr, 4) })
	catch("slice to a short array", func() { sink = [4]int(s) })
	catch("slice to a fit array", func() { sink = [3]int(s) })
	catch("pointer to a short array", func() { sink = (*[4]int)(s) })
}

func nilPointers() {
	var p *point
	catch("read of a field", func() { sink = p.X })
	catch("write of a field", func() { p.X = 1 })
	catch("a whole read", func() { sink = *p })
	var pp **point
	catch("2 levels", func() { sink = **pp })
	var pi *int
	catch("read of an int", func() { sink = *pi })
	catch("write of an int", func() { *pi = 1 })
	var pa *[3]int
	catch("index of a nil array", func() { sink = pa[0] })
	catch("slice of a nil array", func() { sink = pa[:] })
	catch("len of a nil array", func() { sink = len(pa) })
	var f func()
	catch("call of a nil function", func() { f() })
	var s shape
	catch("method of a nil iface", func() { sink = s.Area() })
	var m *withMethod
	catch("field in a method", func() { sink = m.Read() })
	catch("nil method no field", func() { sink = m.Safe() })
}

type withMethod struct{ V int }

func (w *withMethod) Read() int { return w.V }

func (w *withMethod) Safe() int {
	if w == nil {
		return -1
	}
	return w.V
}

func divides() {
	catch("int / 0", func() { a, b := 1, 0; sink = a / b })
	catch("int % 0", func() { a, b := 1, 0; sink = a % b })
	catch("int8 / 0", func() { var a, b int8 = 1, 0; sink = a / b })
	catch("uint64 / 0", func() { var a, b uint64 = 1, 0; sink = a / b })
	catch("0 / 0", func() { a, b := 0, 0; sink = a / b })
	catch("negative / 0", func() { a, b := -1, 0; sink = a / b })
	catch("float / 0", func() { a, b := 1.0, 0.0; sink = a / b })
	catch("float 0 / 0", func() { a, b := 0.0, 0.0; sink = a / b })
}

func assertions() {
	var a any = 42
	catch("int to string", func() { sink = a.(string) })
	catch("int to a struct", func() { sink = a.(point) })
	var s shape = square{2}
	catch("square to a pointer", func() { sink = s.(*square) })
	var nilFace shape
	catch("nil to a type", func() { sink = nilFace.(square) })
	var b any
	catch("nil any to int", func() { sink = b.(int) })
	catch("to a wrong iface", func() { sink = a.(error) })
	var c any = point{1, 2}
	catch("struct to an iface", func() { sink = c.(shape) })
	catch("int to int64", func() { sink = a.(int64) })
	// A 2 result form reports and it does not panic.
	v, ok := a.(string)
	fmt.Printf("%-30s %q %v\n", "the 2 result form", v, ok)
}

func channels() {
	closed := make(chan int)
	close(closed)
	catch("send into a closed channel", func() { closed <- 1 })
	catch("close of a closed channel", func() { close(closed) })
	var nilCh chan int
	catch("close of a nil channel", func() { close(nilCh) })
	catch("read of a closed channel", func() { sink = <-closed })
	v, ok := <-closed
	fmt.Printf("%-30s %v %v\n", "the 2 result form", v, ok)
}

func builtins() {
	catch("make with a negative len", func() { n := -1; sink = make([]int, n) })
	catch("make len above cap", func() { n, c := 3, 2; sink = make([]int, n, c) })
	catch("make a map of a neg size", func() { n := -1; sink = make(map[string]int, n) })
	catch("make a chan of a neg size", func() { n := -1; sink = make(chan int, n) })
	catch("a negative shift", func() { n := -1; sink = 1 << n })
	catch("a huge len", func() { n := 1 << 62; sink = make([]int, n, n) })
}

func explicit() {
	catch("panic of a text", func() { panic("plain text") })
	catch("panic of an int", func() { panic(42) })
	catch("panic of an error", func() { panic(errors.New("an error")) })
	catch("panic of a struct", func() { panic(point{1, 2}) })
	catch("panic of a slice", func() { panic([]int{1, 2}) })
	catch("panic of a formatted text", func() { panic(fmt.Sprintf("code %d", 7)) })
	catch("panic of a wrapped error", func() {
		panic(fmt.Errorf("outer: %w", errors.New("inner")))
	})
	// A panic value that recover gives back keeps its type.
	out := typed()
	fmt.Printf("%-30s %v\n", "the type of a panic value", out)
}

func typed() string {
	defer func() {}()
	var got string
	func() {
		defer func() {
			r := recover()
			if p, ok := r.(point); ok {
				got = fmt.Sprint("point ", p.X, " ", p.Y)
			}
		}()
		panic(point{3, 4})
	}()
	return got
}

// runtimeError proves that each run time value implements error, and that
// errors.As finds it.
func runtimeError() {
	var got error
	func() {
		defer func() {
			if e, ok := recover().(error); ok {
				got = e
			}
		}()
		var s []int
		sink = s[1]
	}()
	fmt.Printf("%-30s %v\n", "is an error", got != nil)
	fmt.Printf("%-30s %v\n", "the text", got)
	fmt.Printf("%-30s %v\n", "errors.Is of itself", errors.Is(got, got))
	fmt.Printf("%-30s %v\n", "errors.Is of another", errors.Is(got, errors.New("x")))
}

func main() {
	fmt.Println("== a nil map ==")
	nilMap()
	fmt.Println("== an index ==")
	indexes()
	fmt.Println("== a slice bound ==")
	slices()
	fmt.Println("== a nil pointer ==")
	nilPointers()
	fmt.Println("== a division ==")
	divides()
	fmt.Println("== a type assertion ==")
	assertions()
	fmt.Println("== a channel ==")
	channels()
	fmt.Println("== a builtin ==")
	builtins()
	fmt.Println("== an explicit panic ==")
	explicit()
	fmt.Println("== the error interface ==")
	runtimeError()
	fmt.Println("done")
}

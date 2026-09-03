// panictext reads the text of each run time error that recover gives.
//
// The 2 forms of a type assertion stay out. Go names the static interface type
// in front of the value, and the runtime call of gart carries no such type.
// docs/parity.md holds that row.
package main

import "fmt"

func say(name string, f func()) {
	defer func() { fmt.Printf("%-22s %v\n", name, recover()) }()
	f()
}

type notComparable struct{ f func() }

type holder struct{ P notComparable }

type myError struct{}

func (*myError) Error() string { return "my" }

type shape interface{ area() int }

type square struct{}

func (square) area() int { return 1 }

type circle struct{}

func (circle) area() int { return 2 }

//go:noinline
func neg() int { return -1 }

//go:noinline
func nine() int { return 9 }

// bounds reads each shape of a slice bound report. Go names the bound that
// failed, thus each shape holds its own text.
func bounds() {
	s := make([]int, 2, 4)
	str := "abcd"
	a := [2]int{}
	say("low neg", func() { _ = s[neg():] })
	say("high neg", func() { _ = s[:neg()] })
	say("max neg", func() { _ = s[0:1:neg()] })
	say("high over cap", func() { _ = s[0:nine()] })
	say("low over high", func() { l := nine(); _ = s[l:1] })
	say("3 max over cap", func() { _ = s[0:1:nine()] })
	say("3 high over max", func() { h := nine(); _ = s[0:h:2] })
	say("3 low over high", func() { l := nine(); _ = s[l:1:2] })
	say("string high", func() { _ = str[0:nine()] })
	say("string low neg", func() { _ = str[neg():] })
	say("array high", func() { _ = a[0:nine()] })
	say("array 3 max", func() { _ = a[0:1:nine()] })
}

func main() {
	say("nil map", func() { var m map[string]int; m["x"] = 1 })
	say("close nil chan", func() { var c chan int; close(c) })
	say("close closed chan", func() { c := make(chan int); close(c); close(c) })
	say("send closed chan", func() { c := make(chan int, 1); close(c); c <- 1 })
	say("divide by zero", func() { a, b := 1, 0; _ = a / b })
	say("nil pointer", func() { var p *int; _ = *p })
	say("index", func() { s := []int{1}; i := 3; _ = s[i] })
	bounds()
	say("negative shift", func() { n := -1; _ = 1 << n })
	say("makeslice len", func() { n := -1; _ = make([]int, n) })
	say("makeslice cap", func() { n := -1; _ = make([]int, 0, n) })
	say("assert iface", func() { var i any = 1; _ = i.(fmt.Stringer) })
	say("assert type", func() { var i any = 1; _ = i.(string) })
	say("assert nil", func() { var i any; _ = i.(string) })
	say("assert named", func() {
		var e error = fmt.Errorf("x")
		_ = e.(*myError)
	})
	say("assert named nil", func() {
		var e error
		_ = e.(*myError)
	})
	say("assert own iface", func() {
		var s shape = square{}
		_ = s.(circle)
	})
	say("uncomparable", func() {
		var a any = notComparable{}
		var b any = notComparable{}
		_ = a == b
	})
	say("uncomparable field", func() {
		var a any = holder{}
		var b any = holder{}
		_ = a == b
	})
	say("plain panic", func() { panic("text") })
	say("error panic", func() { panic(fmt.Errorf("wrapped")) })
	say("no panic", func() {})
}

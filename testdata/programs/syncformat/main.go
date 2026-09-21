package main

import "fmt"

//gart:sync
type text func() string

type quiet struct{}

func (quiet) Error() string { return "quiet" }

type slow struct{}

func (slow) Error() string { <-make(chan int); return "" }

var empty error
var concrete error = quiet{}

func factory() error { return quiet{} }

var returned = factory()

func zero() string {
	var local any
	return fmt.Sprintf("%v %s", local, empty)
}
func known() string { return fmt.Sprint(concrete, "/", returned) }
func nilMethod() (out string) {
	defer func() {
		if recover() != nil {
			out = "nil method panic"
		}
	}()
	return empty.Error()
}
func main() {
	var z text = zero
	var k text = known
	var n text = nilMethod
	fmt.Println(z(), k(), n())
}

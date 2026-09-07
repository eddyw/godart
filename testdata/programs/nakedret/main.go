// The program proves a return with no value. Go declares a result variable for
// each named result, and a blank name also declares one. Such a return gives
// the value of each variable, and a deferred call can change a named one.
package main

import "fmt"

func named() (a int, b bool) {
	a = 3
	return
}

func blank() (_ int, _ bool) {
	return
}

func mixed(n int) (_ int, ok bool) {
	if n > 0 {
		ok = true
		return
	}
	return
}

func generic[T any]() (_ T, _ bool) {
	return
}

func blankOne() (_ string) {
	return
}

func afterDefer() (n int, _ string) {
	defer func() { n += 5 }()
	n = 1
	return
}

func blankDefer() (_ int, s string) {
	defer func() { s += "!" }()
	s = "text"
	return
}

func main() {
	fmt.Println(named())
	fmt.Println(blank())
	fmt.Println(mixed(1))
	fmt.Println(mixed(-1))
	fmt.Println(generic[string]())
	fmt.Println(generic[int]())
	fmt.Printf("%q\n", blankOne())
	fmt.Println(afterDefer())
	fmt.Println(blankDefer())
}

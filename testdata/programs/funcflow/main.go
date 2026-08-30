// The program proves the value flow of stage 3 on 4 paths: a struct field, an
// escape through any with an assertion back, a slice element, and a result of
// a function. Each closure blocks on a channel, thus a lost path hangs or
// gives a junk value.
package main

import "fmt"

type box struct {
	fn func(int) int
}

func main() {
	ch := make(chan int)
	go func() { ch <- 7 }()

	// Path 1: a struct field holds a closure that blocks.
	b := box{fn: func(i int) int { return i + <-ch }}
	fmt.Println(b.fn(1))

	// Path 2: the value escapes through any, and an assertion reads it back.
	go func() { ch <- 5 }()
	var x any = func(i int) int { return i + <-ch }
	f := x.(func(int) int)
	fmt.Println(f(2))

	// Path 3: a slice element.
	go func() { ch <- 3 }()
	fns := []func(int) int{func(i int) int { return i + <-ch }}
	fmt.Println(fns[0](4))

	// Path 4: a result of a function gives the closure.
	go func() { ch <- 1 }()
	g := makeAdder(ch)
	fmt.Println(g(6))
}

func makeAdder(ch chan int) func(int) int {
	return func(i int) int { return i + <-ch }
}

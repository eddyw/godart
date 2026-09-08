// Command syncdeferchain proves that the chain of deferred calls of 1 function
// holds the panic of that function alone.
//
// A function whose deferred calls can block runs its chain inside its state
// machine. The end of that chain read a variable of the library, thus a normal
// return inside the chain of a panicking caller threw the panic of the caller
// again. The real test suite of database/sql found the defect.
package main

import "fmt"

// note writes into a channel, thus a call of it can block.
func note(ch chan int, v int) { ch <- v }

// inner returns in the normal way, and its deferred call can block.
func inner(ch chan int, tag string) int {
	defer note(ch, len(tag))
	fmt.Println("inner", tag)
	return len(tag)
}

// innerRecover holds a recover of its own. The chain of the caller stands open
// while it runs, and its recover must give nil.
func innerRecover(ch chan int) string {
	out := "clean"
	defer note(ch, 1)
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprint("inner recovered ", r)
		}
	}()
	fmt.Println("inner recover body")
	return out
}

// innerPanic panics inside its own chain, and its own deferred call holds it.
func innerPanic(ch chan int) (out string) {
	defer note(ch, 2)
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprint("held ", r)
		}
	}()
	panic("inner boom")
}

// deep panics through 2 frames that each hold a chain that can block.
func deep(ch chan int) {
	defer note(ch, 3)
	panic("deep boom")
}

func plain(ch chan int) {
	defer func() { fmt.Println("recover", recover()) }()
	defer func() {
		fmt.Println("defer start")
		fmt.Println("inner gave", inner(ch, "one"))
		fmt.Println("inner recover gave", innerRecover(ch))
		fmt.Println("inner panic gave", innerPanic(ch))
		fmt.Println("defer end")
	}()
	panic("boom")
}

// normalPath runs the same chain with no panic of the caller.
func normalPath(ch chan int) {
	defer func() { fmt.Println("normal recover", recover()) }()
	defer func() {
		fmt.Println("normal defer start")
		fmt.Println("normal inner gave", inner(ch, "two"))
		fmt.Println("normal inner recover gave", innerRecover(ch))
		fmt.Println("normal defer end")
	}()
	fmt.Println("normal body")
}

// nested panics inside a deferred call, and the outer chain holds the new one.
func nested(ch chan int) {
	defer func() { fmt.Println("nested recover", recover()) }()
	defer func() {
		fmt.Println("nested defer start")
		deep(ch)
		fmt.Println("nested defer end")
	}()
	panic("first")
}

func main() {
	ch := make(chan int, 32)
	plain(ch)
	normalPath(ch)
	nested(ch)
	close(ch)
	sum := 0
	for v := range ch {
		sum += v
	}
	fmt.Println("notes", sum)
}

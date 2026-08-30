// The program proves the hoist of the receiver of a method call whose
// argument blocks. The method holds a pointer receiver, thus the hoisted
// variable must hold the address of the location, and not a copy of the
// value.
package main

import "fmt"

type list []int

func (l *list) add(v int) { *l = append(*l, v) }

type builder struct {
	ch   chan int
	errs list
}

func (b *builder) work() {
	b.errs.add(<-b.ch)
}

func main() {
	b := &builder{ch: make(chan int, 1)}
	b.ch <- 5
	b.work()
	fmt.Println(b.errs)
}

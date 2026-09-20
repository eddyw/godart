package main

import "fmt"

type Step[T any] interface{ Next() T }

type item struct {
	value int
	ready <-chan int
}

func (p *item) Next() *item                  { p.value = <-p.ready; return p }
func advance[T Step[T]](newValue func() T) T { return newValue().Next() }

type sequence int

var events chan int

func (sequence) Next() sequence { return sequence(<-events) }

func valueOnly[T Step[T]]() string {
	var zero T
	return fmt.Sprint(zero.Next())
}

func main() {
	ch := make(chan int)
	go func() { ch <- 7 }()
	p := advance(func() *item { return &item{ready: ch} })
	fmt.Println(p.value)
	events = make(chan int)
	go func() { events <- 9 }()
	fmt.Println(valueOnly[sequence]())
}

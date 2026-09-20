package main

import "fmt"

type Step[T any] interface{ Next() T }

type item struct {
	value int
	ready <-chan int
}

func (p *item) Next() *item { p.value = <-p.ready; return p }

type factory[T Step[T]] struct{ newValue func() T }

func (f *factory[T]) next() T { return f.newValue().Next() }

type sequence int

var events chan int

func (sequence) Next() sequence { return sequence(<-events) }

type emptyFactory[T Step[T]] struct{}

func (emptyFactory[T]) next() string {
	var zero T
	return fmt.Sprint(zero.Next())
}

func main() {
	ch := make(chan int)
	go func() { ch <- 11 }()
	f := &factory[*item]{newValue: func() *item { return &item{ready: ch} }}
	fmt.Println(f.next().value)
	go func() { ch <- 17 }()
	next := f.next
	fmt.Println(next().value)
	go func() { ch <- 19 }()
	fmt.Println((*factory[*item]).next(f).value)

	events = make(chan int)
	go func() { events <- 13 }()
	var source interface{ next() string } = emptyFactory[sequence]{}
	fmt.Println(source.next())
}

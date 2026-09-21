package main

import "fmt"

type item struct{ Value int }
type holder[T any] struct{ Value T }

var calls int

func pointer[T any](value T) *T         { return &value }
func pair[T any](value T) (*T, int)     { calls++; return &value, 7 }
func forward[T any](value T) (*T, int)  { return pair(value) }
func (h holder[T]) pair() (*T, int)     { return &h.Value, 8 }
func (h holder[T]) pointer() *T         { return &h.Value }
func wait[T any](ch <-chan T) (*T, int) { return pair(<-ch) }
func show(p *item, n int)               { fmt.Println(*p, n) }
func main() {
	p, n := pair(41)
	*p++
	fmt.Println(*p, n)
	q, n := pair(item{3})
	q.Value++
	fmt.Println(*q, n)
	a, n := forward([2]int{4, 5})
	a[0] = 6
	fmt.Println(*a, n)
	f := pair[item]
	show(f(item{9}))
	single := pointer[item]
	fmt.Println(*single(item{10}))
	h := holder[item]{item{11}}
	show(h.pair())
	method := h.pair
	show(method())
	expression := holder[item].pair
	show(expression(h))
	one := h.pointer
	fmt.Println(*one())
	ch := make(chan item, 1)
	ch <- item{12}
	show(wait(ch))
	fmt.Println(h.Value, calls)
}

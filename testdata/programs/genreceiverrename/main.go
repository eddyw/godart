package main

import "fmt"

type Integer[T ~int | ~int64 | ~uint64] struct{ Value T }

func (r Integer[U]) Wait(ch <-chan struct{}, next U) U { r.Value = next; <-ch; return r.Value }

type Float[T ~float32 | ~float64] struct{ Value T }

func (r Float[U]) Wait(ch <-chan struct{}, next U) U { r.Value = next; <-ch; return r.Value }

type Complex[T ~complex64 | ~complex128] struct{ Value T }

func (r Complex[U]) Wait(ch <-chan struct{}, next U) U { r.Value = next; <-ch; return r.Value }

type Any[T any] struct{ Value T }

func (r Any[U]) Wait(ch <-chan struct{}, next U) U { r.Value = next; <-ch; return r.Value }

func signal() <-chan struct{} { ch := make(chan struct{}); go func() { close(ch) }(); return ch }

func main() {
	i := Integer[int]{1}
	if i.Wait(signal(), 2) != 2 || i.Value != 1 {
		panic("integer receiver")
	}
	u := Integer[uint64]{3}
	if u.Wait(signal(), 4) != 4 || u.Value != 3 {
		panic("unsigned receiver")
	}
	f := Float[float32]{5}
	if f.Wait(signal(), 6) != 6 || f.Value != 5 {
		panic("float receiver")
	}
	c := Complex[complex128]{7 + 8i}
	if c.Wait(signal(), 9+10i) != 9+10i || c.Value != 7+8i {
		panic("complex receiver")
	}
	a := Any[string]{"before"}
	if a.Wait(signal(), "after") != "after" || a.Value != "before" {
		panic("generic receiver")
	}
	method := i.Wait
	if method(signal(), 11) != 11 || i.Value != 1 {
		panic("method value")
	}
	renamedMore()
	fmt.Println("ok")
}

func (r *Any[U]) Change(ch <-chan struct{}, next U) U {
	<-ch
	r.Value = next
	return r.Value
}
func (r Any[U]) Closure() func(<-chan struct{}) U {
	return func(ch <-chan struct{}) U { <-ch; return r.Value }
}

type Pair[A, B any] struct {
	First  A
	Second B
}

func (p Pair[B, A]) Swap(ch <-chan struct{}) Pair[A, B] {
	<-ch
	return Pair[A, B]{p.Second, p.First}
}

type Items[T any] []T

func (items Items[U]) First(ch <-chan struct{}) U { <-ch; return items[0] }
func (r Any[U]) Shadow(ch <-chan struct{}) U {
	T := 29
	<-ch
	fmt.Println(T)
	return r.Value
}
func renamedMore() {
	a := Any[string]{"a"}
	fmt.Println(a.Shadow(signal()))
	fmt.Println(a.Change(signal(), "b"), a.Value)
	fn := a.Change
	fmt.Println(fn(signal(), "c"), a.Value)
	expr := (*Any[string]).Change
	fmt.Println(expr(&a, signal(), "d"), a.Value)
	closure := a.Closure()
	a.Value = "after"
	fmt.Println(closure(signal()), a.Value)
	pair := Pair[int, string]{17, "value"}
	fmt.Println(pair.Swap(signal()))
	fmt.Println(Items[int]{23}.First(signal()))
}

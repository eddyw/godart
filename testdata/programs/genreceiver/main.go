package main

import "fmt"

type Integer[T ~int | ~int64 | ~uint64] struct{ Value T }

func (r Integer[T]) Wait(ch <-chan struct{}, next T) T { r.Value = next; <-ch; return r.Value }

type Float[T ~float32 | ~float64] struct{ Value T }

func (r Float[T]) Wait(ch <-chan struct{}, next T) T { r.Value = next; <-ch; return r.Value }

type Complex[T ~complex64 | ~complex128] struct{ Value T }

func (r Complex[T]) Wait(ch <-chan struct{}, next T) T { r.Value = next; <-ch; return r.Value }

type Any[T any] struct{ Value T }

func (r Any[T]) Wait(ch <-chan struct{}, next T) T { r.Value = next; <-ch; return r.Value }

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
	fmt.Println("ok")
}

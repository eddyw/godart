package main

import "runtime"

type pointer *int

func outcome(f func()) (result string) {
	defer func() {
		if value := recover(); value != nil {
			result = value.(string)
		}
	}()
	f()
	return "accepted"
}

func main() {
	var nilPointer *int
	println(outcome(func() { runtime.AddCleanup(nilPointer, func(int) {}, 0) }))
	p := new(int)
	println(outcome(func() { runtime.AddCleanup(p, func(*int) {}, p) }))
	println(outcome(func() { runtime.AddCleanup(p, func(pointer) {}, pointer(p)) }))
	println(outcome(func() { runtime.AddCleanup(p, func(any) {}, any(p)).Stop() }))
	a := new([2]int)
	println(outcome(func() { runtime.AddCleanup(&a[1], func(*int) {}, &a[1]) }))
	runtime.AddCleanup(p, func(*int) {}, new(int)).Stop()
	runtime.AddCleanup(p, func(pointer) {}, pointer(nil)).Stop()
	runtime.AddCleanup(p, func([]int) {}, []int{1, 2}).Stop()
	runtime.AddCleanup(p, func(map[int]int) {}, map[int]int{1: 2}).Stop()
	println("distinct, nil, slice, and map arguments accepted")
	runtime.KeepAlive(p)
	runtime.KeepAlive(a)
}

package main

import "fmt"

type array [3]int
type slice []int
type item struct{ Value int }

func replace[T ~[]E | ~[3]E, E any](x T, value E) (T, *E) {
	p := &x[0]
	x[0] = value
	return x, p
}
func stage[T ~[]int | ~[3]int](x, y T) T                   { x, x[0] = y, 9; return x }
func receive[T ~[]int | ~[3]int](x T, c <-chan int) T      { x[1] = <-c; return x }
func increment[T ~[]int | ~[3]int](x T, next func() int) T { x[next()] += 5; return x }
func at[T ~[]int | ~[3]int, I ~int | ~uint64](x T, i I) {
	defer func() { fmt.Println(recover()) }()
	fmt.Println(x[i])
}
func put[T ~[]int | ~[3]int, I ~int | ~uint64](x T, i I) {
	defer func() { fmt.Println(recover()) }()
	x[i] = 5
}
func address[T ~[]int | ~[3]int, I ~int | ~uint64](x T, i I) {
	defer func() { fmt.Println(recover()) }()
	fmt.Println(*&x[i])
}
func arrays[T ~[2]int | ~[3]int](x T) T { x[0] = 20; return x }
func main() {
	a := array{1, 2, 3}
	s := slice{1, 2, 3}
	a1, ap := replace(a, 7)
	s1, sp := replace(s, 7)
	fmt.Println(a, a1, *ap, s, s1, *sp, sp == &s[0])
	fmt.Println(stage(a, array{4, 5, 6}), a)
	fmt.Println(stage(s, slice{4, 5, 6}), s)
	c := make(chan int, 2)
	c <- 11
	c <- 12
	fmt.Println(receive(a, c), a, receive(s, c), s)
	n := 0
	next := func() int { n++; return 2 }
	fmt.Println(increment(a, next), increment(s, next), n)
	av, arp := replace([3]item{{1}, {2}, {3}}, item{7})
	sv, srp := replace([]item{{1}, {2}, {3}}, item{8})
	fmt.Println(av, *arp, sv, *srp)
	at(a, -1)
	at(a, uint64(1<<63))
	at(s, 3)
	at(slice(nil), 0)
	put(a, uint64(1<<63))
	put(s, -1)
	address(a, uint64(1<<63))
	address(s, -1)
	fmt.Println(arrays([2]int{1, 2}), arrays([3]int{3, 4, 5}))
	arp.Value = 14
	srp.Value = 15
	fmt.Println(av, *arp, sv, *srp)
}

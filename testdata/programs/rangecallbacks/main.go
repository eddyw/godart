package main

import "fmt"

type reader struct{ ready <-chan int }

func (r reader) next() int { return <-r.ready }

func main() {
	ready := make(chan int)
	r := reader{ready: ready}
	go func() {
		for i := 1; i <= 6; i++ {
			ready <- i
		}
	}()
	for _, next := range []func() int{r.next, r.next} {
		fmt.Println(next())
	}
	for _, next := range [...]func(reader) int{reader.next, reader.next} {
		fmt.Println(next(r))
	}
	for _, next := range map[string]func() int{"only": r.next} {
		fmt.Println(next())
	}
	calls := make(chan func() int, 1)
	calls <- r.next
	close(calls)
	for next := range calls {
		fmt.Println(next())
	}
}

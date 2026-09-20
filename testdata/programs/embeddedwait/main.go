package main

import "fmt"

type step interface{ Next() int }
type wrapper struct{ step }
type outer struct{ wrapper }
type reader struct{ ready <-chan int }

func (r reader) Next() int { return <-r.ready }

func main() {
	ready := make(chan int)
	go func() {
		ready <- 7
		ready <- 11
	}()
	var first step = wrapper{reader{ready}}
	fmt.Println(first.Next())
	var second step = outer{wrapper{reader{ready}}}
	fmt.Println(second.Next())
}

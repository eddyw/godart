package main

import (
	"fmt"
	"sync/atomic"
)

func delayedRecover(id int, ready chan<- bool, release <-chan struct{}, results []any, done chan<- bool) {
	defer func() {
		ready <- true
		for {
			select {
			case <-release:
				results[id] = recover()
				done <- true
				return
			default:
			}
		}
	}()
	panic(id + 1)
}

func alternate(id uint32, turn *uint32, done chan<- bool) {
	for i := 0; i < 3; i++ {
	again:
		if atomic.LoadUint32(turn) != id {
			goto again
		}
		atomic.StoreUint32(turn, 1-id)
	}
	done <- true
}

func main() {
	ready := make(chan bool, 2)
	release := make(chan struct{})
	done := make(chan bool, 2)
	results := make([]any, 2)
	for id := range results {
		go delayedRecover(id, ready, release, results, done)
	}
	<-ready
	<-ready
	close(release)
	<-done
	<-done
	fmt.Println(results)

	var turn uint32
	go alternate(0, &turn, done)
	go alternate(1, &turn, done)
	<-done
	<-done
	fmt.Println(atomic.LoadUint32(&turn))
}

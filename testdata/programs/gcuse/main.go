package main

import (
	"fmt"
	"runtime"
	"time"
)

type box struct{ data [128]int }

func main() {
	closed := make(chan struct{})
	func() {
		b := &box{}
		runtime.AddCleanup(b, func(ch chan struct{}) { close(ch) }, closed)
	}()
	runtime.GC()
	select {
	case <-closed:
		fmt.Println("the cleanup ran")
	case <-time.After(5 * time.Second):
		fmt.Println("timeout")
	}

	// A cleanup that Stop drops never runs.
	kept := &box{}
	c := runtime.AddCleanup(kept, func(s string) { fmt.Println("must not run", s) }, "x")
	c.Stop()
	runtime.GC()
	fmt.Println("kept", len(kept.data))
}

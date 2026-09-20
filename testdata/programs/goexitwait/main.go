package main

import (
	"fmt"
	"runtime"
)

func stop() {
	defer func() {
		runtime.Gosched()
		fmt.Println("defer", recover())
	}()
	runtime.Goexit()
}

func main() {
	done := make(chan struct{})
	go func() {
		defer close(done)
		stop()
		fmt.Println("unexpected return")
	}()
	<-done
	fmt.Println("done")
}

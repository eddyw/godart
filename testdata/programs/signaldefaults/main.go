package main

import (
	"fmt"
	"syscall"
	"time"
)

func main() {
	for _, sig := range []syscall.Signal{syscall.SIGUSR1, syscall.SIGUSR2} {
		if err := syscall.Kill(syscall.Getpid(), sig); err != nil {
			panic(err)
		}
	}
	time.Sleep(100 * time.Millisecond)
	fmt.Println("alive")
}

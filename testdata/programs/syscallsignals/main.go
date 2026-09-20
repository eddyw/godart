package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	pid := syscall.Getpid()
	fmt.Println(pid == os.Getpid(), syscall.Kill(pid, 0))
	fmt.Println(errors.Is(syscall.Kill(pid, -1), syscall.EINVAL))
	fmt.Println(errors.Is(syscall.Kill(1<<30, 0), syscall.ESRCH))
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGWINCH)
	defer signal.Stop(c)
	if err := syscall.Kill(pid, syscall.SIGWINCH); err != nil {
		panic(err)
	}
	receive(c)
	self, err := os.FindProcess(pid)
	if err != nil {
		panic(err)
	}
	if err := self.Signal(syscall.SIGWINCH); err != nil {
		panic(err)
	}
	receive(c)
}

func receive(c <-chan os.Signal) {
	select {
	case s := <-c:
		fmt.Println(s, int(s.(syscall.Signal)))
	case <-time.After(5 * time.Second):
		panic("signal timeout")
	}
}

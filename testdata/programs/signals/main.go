package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println(os.Interrupt, os.Kill)
	fmt.Println(syscall.SIGTERM, int(syscall.SIGTERM))
	fmt.Println(syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGALRM)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	self, err := os.FindProcess(os.Getpid())
	if err != nil {
		fmt.Println("find:", err)
		return
	}
	if err := self.Signal(syscall.SIGTERM); err != nil {
		fmt.Println("signal:", err)
		return
	}
	fmt.Println("got", <-c)

	signal.Ignore(syscall.SIGHUP)
	fmt.Println("ignored hup:", signal.Ignored(syscall.SIGHUP))
	signal.Reset(syscall.SIGHUP)
	fmt.Println("ignored hup:", signal.Ignored(syscall.SIGHUP))

	// The last watch stays open, thus the release of the runtime must cancel
	// its subscription. A live subscription keeps the Dart isolate alive.
	signal.Stop(c)
	fmt.Println("done")
}

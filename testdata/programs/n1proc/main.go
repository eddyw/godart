// The program proves the process names that the work of 2026-09-04 added:
// os.ProcessState.Sys, os/exec.ExitError.Sys, and os.Process.WithHandle.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	// A child that ends with a code carries that code in the wait state.
	err := exec.Command("sh", "-c", "exit 3").Run()
	var bad *exec.ExitError
	if !errors.As(err, &bad) {
		fmt.Println("no exit error:", err)
		return
	}
	status, ok := bad.Sys().(syscall.WaitStatus)
	fmt.Println("exit:", ok, status.Exited(), status.ExitStatus())
	fmt.Println("exit signal:", status.Signaled(), status.CoreDump(),
		status.Stopped(), status.Continued())

	// A child that a signal ends carries the number of the signal.
	err = exec.Command("sh", "-c", "kill -9 $$").Run()
	if !errors.As(err, &bad) {
		fmt.Println("no exit error:", err)
		return
	}
	status, ok = bad.Sys().(syscall.WaitStatus)
	fmt.Println("kill:", ok, status.Signaled(), status.Signal())
	fmt.Println("kill code:", status.Exited(), status.ExitStatus(), bad.ExitCode())
	fmt.Println("text:", bad.Error())

	// The state of a child that Wait gives holds the same word.
	cmd := exec.Command("sh", "-c", "exit 7")
	cmd.Run()
	state, _ := cmd.ProcessState.Sys().(syscall.WaitStatus)
	fmt.Println("wait:", state.ExitStatus(), cmd.ProcessState.Sys() != nil)

	// A handle of the system reaches only a platform that holds one.
	self, _ := os.FindProcess(os.Getpid())
	held := self.WithHandle(func(handle uintptr) {})
	fmt.Println("handle:", held == nil || errors.Is(held, os.ErrNoHandle))
}

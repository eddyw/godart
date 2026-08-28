package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

// TestRejectBlockingDefer proves that the build rejects a deferred call that
// can block.
//
// The deferred calls run at the return of a function. A state machine keeps no
// state at that point, thus such a call reaches no scheduler. Stage 4 holds the
// rule, because stage 2 reads no block analysis.
func TestRejectBlockingDefer(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module reject\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

func send(ch chan int) { ch <- 1 }

func main() {
	ch := make(chan int, 1)
	defer send(ch)
	println("body")
}
`)
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	err = driver.Build(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtime,
	})
	if err == nil {
		t.Fatal("the build passed, and the rule must reject the program")
	}
	const want = "a deferred call that can block is outside the subset"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("the message is %q, and it must hold %q", err.Error(), want)
	}
	if !strings.Contains(err.Error(), "main.go:7") {
		t.Fatalf("the message is %q, and it must hold the position", err.Error())
	}
}

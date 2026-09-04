//go:build dart

// Package synctest gives the bubble of Go for a test of code that waits.
//
// Test runs a function in a bubble. Each goroutine that the function starts
// joins the bubble, and the bubble holds a clock of its own. The clock starts
// at midnight UTC of 2000-01-01, and it moves only when each goroutine of the
// bubble is durably blocked. A sleep inside a bubble thus takes no real time.
//
// A goroutine is durably blocked when only another goroutine of the same
// bubble can wake it. A send or a receive on a channel of the bubble, a select
// statement over such channels, and time.Sleep each block a goroutine that
// way. sync.Cond.Wait and sync.WaitGroup.Wait do too, and a read of a socket
// does not.
package synctest

import (
	"internal/linkhook"
	"syscall/dart"
	"testing"
	"time"
)

// Test runs f in a new bubble, and it waits for the end of each goroutine of
// the bubble.
//
// Test must not run inside a bubble. The test fails when the goroutines of the
// bubble reach a deadlock.
func Test(t *testing.T, f func(*testing.T)) {
	if dart.InBubble() {
		panic("synctest.Run called from within a synctest bubble")
	}
	ok := false
	cause := dart.BubbleRun(func() {
		ok = linkhook.SynctestTest(t, func(v any) { f(v.(*testing.T)) })
	})
	if cause != "" {
		panic(deadlockError(cause))
	}
	if !ok {
		// Go marks the failure outside the bubble, thus the time of the test
		// holds the real clock.
		t.FailNow()
	}
}

// Wait blocks until each goroutine of the bubble of the caller, other than the
// goroutine of the caller, is durably blocked.
//
// Wait must run inside a bubble, and 2 goroutines of 1 bubble must not call it
// at the same time.
func Wait() {
	if !dart.InBubble() {
		panic("goroutine is not in a bubble")
	}
	if !dart.BubbleWait() {
		panic("wait already in progress")
	}
}

// Sleep blocks until the clock of the bubble moved d and each other goroutine
// of the bubble is durably blocked. It is equal to a call of time.Sleep and a
// call of Wait.
func Sleep(d time.Duration) {
	time.Sleep(d)
	Wait()
}

// deadlockError is the value of the panic of a bubble that makes no more
// progress. Go holds the same 2 texts in a type of its runtime.
type deadlockError string

func (e deadlockError) Error() string { return string(e) }

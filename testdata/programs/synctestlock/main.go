// Command synctestlock proves the rule of a lock inside a bubble of
// testing/synctest.
//
// A wait for a lock is not durable. A goroutine outside the bubble can give
// the lock back, thus the clock of the bubble stands still while the wait
// stands. sync.Cond.Wait and sync.WaitGroup.Wait keep the durable mark, thus
// the clock moves for each one.
package main

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// unlockLate gives the lock back after a wait of the real clock. The goroutine
// stands outside each bubble.
func unlockLate(mu *sync.Mutex) {
	time.Sleep(50 * time.Millisecond)
	mu.Unlock()
}

func bubble(outer *sync.Mutex) {
	start := time.Now()

	// A goroutine outside the bubble holds the lock and gives it back. The
	// clock of the bubble does not move while the wait stands.
	outer.Lock()
	fmt.Println("outer lock", time.Since(start))
	outer.Unlock()

	// A lock that a goroutine of the bubble gives back. That goroutine stands
	// on the run queue, thus the wait needs no move of the clock either.
	var mu sync.Mutex
	mu.Lock()
	go func() { mu.Unlock() }()
	mu.Lock()
	fmt.Println("inner lock", time.Since(start))
	mu.Unlock()

	// A WaitGroup keeps the durable mark, thus the clock moves 2 seconds.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		time.Sleep(2 * time.Second)
		wg.Done()
	}()
	wg.Wait()
	fmt.Println("wait group", time.Since(start))

	// A Cond keeps the durable mark too. The goroutine takes the lock, marks
	// the condition, and wakes the waiter.
	var cmu sync.Mutex
	cond := sync.NewCond(&cmu)
	ready := false
	go func() {
		time.Sleep(3 * time.Second)
		cmu.Lock()
		ready = true
		cond.Signal()
		cmu.Unlock()
	}()
	cmu.Lock()
	for !ready {
		cond.Wait()
	}
	cmu.Unlock()
	fmt.Println("cond", time.Since(start))
}

func main() {
	var outer sync.Mutex
	outer.Lock()
	go unlockLate(&outer)

	t := new(testing.T)
	synctest.Test(t, func(t *testing.T) { bubble(&outer) })
	fmt.Println("failed", t.Failed())
}

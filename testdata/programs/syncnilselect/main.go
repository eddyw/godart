// Command syncnilselect proves that a select case of a nil channel keeps the
// wait of a synctest bubble durable.
//
// context.Background gives a Done channel that is nil. The pool of database/sql
// waits in a select over that channel and over a channel of the bubble, and the
// clock of the bubble must still move.
package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// pool holds the shape of TestPoolExhaustOnCancel of database/sql: each worker
// waits in a select over the Done channel of a context with no cancel and over
// a channel of the bubble, and 1 goroutine sleeps on the clock of the bubble.
func pool(t *testing.T) {
	start := time.Now()
	done := make(chan bool)
	var wg sync.WaitGroup
	var mu sync.Mutex
	order := make([]string, 0, 4)

	background := context.Background()
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			select {
			case <-background.Done():
				mu.Lock()
				order = append(order, fmt.Sprint("cancel ", n))
				mu.Unlock()
			case <-done:
				mu.Lock()
				order = append(order, fmt.Sprint("done ", n))
				mu.Unlock()
			}
		}(i)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()

	wg.Wait()
	mu.Lock()
	count := len(order)
	mu.Unlock()
	fmt.Println("workers", count, "after", time.Since(start))
}

// timeout proves that a context with a deadline still ends the same select.
func timeout(t *testing.T) {
	start := time.Now()
	never := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	background := context.Background()
	out := make(chan string, 2)
	go func() {
		select {
		case <-background.Done():
			out <- "background"
		case <-ctx.Done():
			out <- "deadline"
		case <-never:
			out <- "never"
		}
	}()
	fmt.Println("timeout", <-out, "after", time.Since(start))
}

// allNil proves that a select over nil channels alone blocks for ever, exactly
// as it does in Go. The bubble reports the deadlock.
func allNil(t *testing.T) {
	var a, b chan int
	select {
	case <-a:
	case <-b:
	}
}

func main() {
	t := new(testing.T)
	synctest.Test(t, pool)
	synctest.Test(t, timeout)

	func() {
		defer func() { fmt.Println("all nil", recover()) }()
		synctest.Test(t, allNil)
	}()

	// The same select outside a bubble takes the ready case.
	var nilCh chan int
	ready := make(chan int, 1)
	ready <- 7
	select {
	case v := <-nilCh:
		fmt.Println("nil gave", v)
	case v := <-ready:
		fmt.Println("ready gave", v)
	}

	// A nil channel with a default case takes the default.
	select {
	case <-nilCh:
		fmt.Println("nil ran")
	default:
		fmt.Println("default ran")
	}
	fmt.Println("failed", t.Failed())
}

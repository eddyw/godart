package main

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"
)

// bubble runs each part of the witness inside 1 bubble.
func bubble(t *testing.T) {
	start := time.Now()
	fmt.Println("start", start.UTC().Format(time.RFC3339Nano))

	// A goroutine that ends. Wait holds the caller until it ended.
	done := false
	go func() { done = true }()
	synctest.Wait()
	fmt.Println("wait", done, time.Since(start))

	// A goroutine that sleeps and then writes into a channel. The clock of the
	// bubble moves 3 seconds, and the run takes no real time.
	ch := make(chan string)
	go func() {
		time.Sleep(3 * time.Second)
		ch <- "value"
	}()
	fmt.Println("receive", <-ch, time.Since(start))

	// A timer of the bubble.
	timer := time.NewTimer(5 * time.Second)
	at := <-timer.C
	fmt.Println("timer", at.Sub(start))
	fmt.Println("timer stop", timer.Stop())

	// A function that a timer runs on a goroutine of the bubble.
	hits := 0
	time.AfterFunc(2*time.Second, func() { hits++ })
	synctest.Sleep(2 * time.Second)
	fmt.Println("after func", hits, time.Since(start))

	// A ticker of the bubble.
	tick := time.NewTicker(time.Second)
	for i := 0; i < 3; i++ {
		<-tick.C
		fmt.Println("tick", i, time.Since(start))
	}
	tick.Stop()

	// A group of goroutines that each sleep. The clock moves 1 time for each
	// deadline, thus the order of the lines follows the durations.
	out := make(chan int)
	for i := 1; i <= 3; i++ {
		go func(n int) {
			time.Sleep(time.Duration(n) * time.Second)
			out <- n
		}(i)
	}
	for i := 0; i < 3; i++ {
		fmt.Println("worker", <-out, time.Since(start))
	}

	fmt.Println("end", time.Now().UTC().Format(time.RFC3339Nano))
}

func main() {
	t := new(testing.T)
	synctest.Test(t, bubble)
	fmt.Println("failed", t.Failed())

	// A second bubble starts a clock of its own.
	synctest.Test(t, func(t *testing.T) {
		fmt.Println("second", time.Now().UTC().Format(time.RFC3339Nano))
		time.Sleep(time.Hour)
		fmt.Println("second", time.Now().UTC().Format(time.RFC3339Nano))
	})

	// The clock outside a bubble is the real clock, thus the year is not 2000.
	fmt.Println("outside", time.Now().Year() > 2000)
}

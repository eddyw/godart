package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

func afterFunc() {
	done := make(chan string, 1)
	timer := time.AfterFunc(10*time.Millisecond, func() {
		done <- "the timer ran"
	})
	fmt.Println(<-done)
	fmt.Println("a stop after the run", timer.Stop())
}

func stop() {
	late := time.AfterFunc(time.Hour, func() {
		fmt.Println("this line never runs")
	})
	fmt.Println("a stop before the run", late.Stop())
	fmt.Println("a second stop", late.Stop())
}

func until() {
	ahead := time.Now().Add(time.Hour)
	fmt.Println("until is under an hour", time.Until(ahead) <= time.Hour)
	fmt.Println("until is over 59 minutes", time.Until(ahead) > 59*time.Minute)
	behind := time.Now().Add(-time.Minute)
	fmt.Println("until of the past is negative", time.Until(behind) < 0)
}

func atomics() {
	var count atomic.Int32
	fmt.Println("the zero value", count.Load())
	fmt.Println("add", count.Add(3), count.Add(-1))
	count.Store(7)
	fmt.Println("store", count.Load())

	var holder atomic.Value
	fmt.Println("an empty value", holder.Load())
	holder.Store("first")
	fmt.Println("swap", holder.Swap("second"), holder.Load())
}

func main() {
	afterFunc()
	stop()
	until()
	atomics()
}

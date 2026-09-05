// The program proves that Reset of a timer of AfterFunc runs the function
// again, and that Reset and Stop of a Timer that no constructor made each
// panic with the text of Go.
package main

import (
	"fmt"
	"time"
)

func main() {
	done := make(chan int, 4)
	n := 0
	t := time.AfterFunc(10*time.Millisecond, func() {
		n++
		done <- n
	})
	fmt.Println("the reset of a timer that runs:", t.Reset(20*time.Millisecond))
	fmt.Println("the first run:", <-done)
	fmt.Println("the reset after the run:", t.Reset(10*time.Millisecond))
	fmt.Println("the second run:", <-done)

	c := time.NewTimer(10 * time.Millisecond)
	fmt.Println("the reset of a timer of a channel:", c.Reset(20*time.Millisecond))
	<-c.C
	fmt.Println("the channel holds the time")

	fmt.Println("a reset of a zero timer:", zeroReset())
	fmt.Println("a stop of a zero timer:", zeroStop())
}

func zeroReset() (text string) {
	defer func() { text = fmt.Sprint(recover()) }()
	var t time.Timer
	t.Reset(time.Second)
	return "no panic"
}

func zeroStop() (text string) {
	defer func() { text = fmt.Sprint(recover()) }()
	var t time.Timer
	t.Stop()
	return "no panic"
}

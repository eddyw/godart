// Command synctesttimer proves the identity of a timer of a bubble of
// testing/synctest.
//
// A timer that ran holds no timer any more, thus a stop of it reports false
// and it reaches no later timer. A timer that stood armed at the end of its
// bubble keeps its state, thus a stop of it from inside another bubble reports
// true.
package main

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"
)

// fired holds a timer of the first bubble that ran.
var fired *time.Timer

// armed holds a timer of the first bubble that never ran.
var armed *time.Timer

func first() {
	start := time.Now()

	a := time.NewTimer(time.Second)
	<-a.C
	fmt.Println("a fired", time.Since(start))

	// b takes a handle of its own, thus the stop of a below reaches b never.
	b := time.NewTimer(2 * time.Second)
	fmt.Println("a stop", a.Stop())
	<-b.C
	fmt.Println("b fired", time.Since(start))

	// A reset of a timer that ran reports false and starts the timer again.
	fmt.Println("b reset", b.Reset(time.Second))
	<-b.C
	fmt.Println("b again", time.Since(start))

	fired = b
	armed = time.NewTimer(time.Hour)
}

// second reads the 2 timers of the first bubble from inside a second bubble.
// Go writes no report for such a call, because the goroutine of the caller
// stands inside a bubble.
func second() {
	fmt.Println("fired stop", fired.Stop())
	fmt.Println("armed stop", armed.Stop())
}

// outside proves the same rule for a timer of the real clock.
func outside() {
	c := time.NewTimer(time.Millisecond)
	<-c.C
	d := time.NewTimer(time.Millisecond)
	fmt.Println("c stop", c.Stop())
	<-d.C
	fmt.Println("d fired")
	fmt.Println("d stop", d.Stop())
}

func main() {
	t := new(testing.T)
	synctest.Test(t, func(t *testing.T) { first() })
	synctest.Test(t, func(t *testing.T) { second() })
	outside()
	fmt.Println("failed", t.Failed())
}

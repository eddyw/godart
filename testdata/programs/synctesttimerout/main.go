// Command synctesttimerout proves the report of a timer of a bubble of
// testing/synctest that a goroutine outside each bubble stops.
//
// The end of the bubble keeps the mark of the timer, thus the report stands
// after that end too. The program writes no line before the report, because
// the 2 runtimes write a trace of their own after it.
package main

import (
	"testing"
	"testing/synctest"
	"time"
)

var armed *time.Timer

func main() {
	t := new(testing.T)
	synctest.Test(t, func(t *testing.T) {
		armed = time.NewTimer(time.Hour)
	})
	armed.Stop()
}

package parallel

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// order holds each step of the run. Each write stands behind a channel
// operation or behind the pass of the runner, thus the list holds 1 order.
var order []string

// The 3 channels of the rendezvous of the 2 parallel tests. A run that holds
// the 2 tests 1 after the other waits for ever, thus a pass proves that the 2
// tests run at the same time.
var (
	toSecond = make(chan int)
	toFirst  = make(chan int)
	firstEnd = make(chan int)
)

// TestMain writes the order of the steps after the run. The order proves that
// each parallel test waits for the pass of the tests that are not parallel.
func TestMain(m *testing.M) {
	code := m.Run()
	fmt.Println("order:", strings.Join(order, " "))
	os.Exit(code)
}

func TestAaaSerial(t *testing.T) {
	order = append(order, "serial-1")
}

func TestBbbFirst(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { close(firstEnd) })
	order = append(order, "first-start")
	toSecond <- 1
	<-toFirst
	order = append(order, "first-end")
}

func TestCccSerial(t *testing.T) {
	order = append(order, "serial-2")
}

func TestDddSecond(t *testing.T) {
	t.Parallel()
	<-toSecond
	order = append(order, "second-middle")
	toFirst <- 1
	<-firstEnd
	order = append(order, "second-end")
}

func TestEeeSum(t *testing.T) {
	if got := Sum(2, 3); got != 5 {
		t.Errorf("Sum(2, 3) gives %d, and 5 is the answer", got)
	}
}

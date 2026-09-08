// Command syncreflectselect proves that a wait of the package reflect keeps
// its durable mark inside a synctest bubble.
//
// reflect.Select and reflect.Value.Recv each wrap the frame of the runtime in
// a frame of their own. Such a wrapper must give the mark of the wait that it
// holds, or the clock of the bubble stands still.
package main

import (
	"fmt"
	"reflect"
	"testing"
	"testing/synctest"
	"time"
)

// selectWait waits inside reflect.Select while a sleeper of the bubble holds
// the only value.
func selectWait(t *testing.T) {
	start := time.Now()
	ch := make(chan int)
	var nilCh chan int
	go func() {
		time.Sleep(2 * time.Second)
		ch <- 41
	}()
	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(nilCh)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)},
	}
	i, v, ok := reflect.Select(cases)
	fmt.Println("select", i, v.Int(), ok, "after", time.Since(start))
}

// recvWait waits inside reflect.Value.Recv.
func recvWait(t *testing.T) {
	start := time.Now()
	ch := make(chan string)
	go func() {
		time.Sleep(3 * time.Second)
		ch <- "late"
	}()
	v, ok := reflect.ValueOf(ch).Recv()
	fmt.Println("recv", v.String(), ok, "after", time.Since(start))
}

// sendWait waits inside a send case of reflect.Select.
func sendWait(t *testing.T) {
	start := time.Now()
	ch := make(chan int)
	go func() {
		time.Sleep(time.Second)
		fmt.Println("reader got", <-ch)
	}()
	cases := []reflect.SelectCase{
		{Dir: reflect.SelectSend, Chan: reflect.ValueOf(ch),
			Send: reflect.ValueOf(7)},
	}
	i, _, _ := reflect.Select(cases)
	fmt.Println("send", i, "after", time.Since(start))
}

func main() {
	t := new(testing.T)
	synctest.Test(t, selectWait)
	synctest.Test(t, recvWait)
	synctest.Test(t, sendWait)
	fmt.Println("failed", t.Failed())
}

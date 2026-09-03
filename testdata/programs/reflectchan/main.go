// The program reads the channel family and Select of the package reflect. The
// runtime holds real channels, thus a send and a receive give up control.
package main

import (
	"fmt"
	"reflect"
	"time"
)

func main() {
	// ChanOf and MakeChan build a channel that a type assertion reads.
	ct := reflect.ChanOf(reflect.BothDir, reflect.TypeOf(0))
	fmt.Println(ct, ct.Kind(), ct.Elem(), ct.ChanDir())
	cv := reflect.MakeChan(ct, 2)
	ch := cv.Interface().(chan int)
	ch <- 1
	fmt.Println(cv.Len(), cv.Cap())

	// Send, Recv, and Close of a Value.
	cv.Send(reflect.ValueOf(2))
	got, ok := cv.Recv()
	fmt.Println(got.Int(), ok)
	got, ok = cv.Recv()
	fmt.Println(got.Int(), ok)

	// TryRecv on an empty channel gives no valid Value.
	x, ok := cv.TryRecv()
	fmt.Println(x.IsValid(), ok)

	// TrySend fills the buffer, and the third send finds no room.
	fmt.Println(cv.TrySend(reflect.ValueOf(7)), cv.TrySend(reflect.ValueOf(8)),
		cv.TrySend(reflect.ValueOf(9)))
	x, ok = cv.TryRecv()
	fmt.Println(x.Int(), ok)

	// A closed channel gives the zero value and a false flag.
	dc := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, reflect.TypeOf("")), 0)
	dc.Close()
	x, ok = dc.TryRecv()
	fmt.Println(x.IsValid(), x.String() == "", ok)
	y, ok := dc.Recv()
	fmt.Println(y.String() == "", ok)

	// The direction of a channel type reads from the descriptor.
	var recv <-chan int
	var send chan<- int
	fmt.Println(reflect.TypeOf(recv).ChanDir(), reflect.TypeOf(send).ChanDir())
	fmt.Println(reflect.ChanOf(reflect.RecvDir, reflect.TypeOf(0)),
		reflect.ChanOf(reflect.SendDir, reflect.TypeOf(0)))

	// A send of a Value gives up control until a receiver takes the value.
	un := make(chan int)
	go func() {
		time.Sleep(5 * time.Millisecond)
		fmt.Println("read", <-un)
	}()
	reflect.ValueOf(un).Send(reflect.ValueOf(11))

	// A receive of a Value gives up control until a sender writes.
	go func() {
		time.Sleep(5 * time.Millisecond)
		un <- 12
	}()
	z, ok := reflect.ValueOf(un).Recv()
	fmt.Println("recv", z.Int(), ok)

	// Select with a default takes the default when no case is ready.
	a := make(chan int, 1)
	b := make(chan string, 1)
	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(a)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(b)},
		{Dir: reflect.SelectDefault},
	}
	at, v, ok := reflect.Select(cases)
	fmt.Println("default", at, v.IsValid(), ok)

	// Select takes the case that is ready.
	b <- "text"
	at, v, ok = reflect.Select(cases)
	fmt.Println("ready", at, v.String(), ok)

	// A send case writes into the channel.
	sendCases := []reflect.SelectCase{
		{Dir: reflect.SelectSend, Chan: reflect.ValueOf(a), Send: reflect.ValueOf(21)},
		{Dir: reflect.SelectDefault},
	}
	at, _, _ = reflect.Select(sendCases)
	fmt.Println("send", at, <-a)

	// Select gives up control until a channel is ready.
	go func() {
		time.Sleep(5 * time.Millisecond)
		a <- 31
	}()
	at, v, ok = reflect.Select([]reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(a)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(b)},
	})
	fmt.Println("wait", at, v.Int(), ok)

	// A case whose Chan holds the zero Value never runs.
	c := make(chan int, 1)
	c <- 41
	at, v, ok = reflect.Select([]reflect.SelectCase{
		{Dir: reflect.SelectRecv},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(c)},
	})
	fmt.Println("skip", at, v.Int(), ok)

	// The zero value of a channel type reads as nil.
	zv := reflect.Zero(ct)
	fmt.Println("zero", zv.IsNil(), zv.Len(), zv.Cap())
	var nilChan chan int
	fmt.Println("nil", reflect.ValueOf(nilChan).IsNil())

	// A closed channel gives a false flag inside a select.
	close(b)
	at, v, ok = reflect.Select([]reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(b)},
	})
	fmt.Println("closed", at, v.String() == "", ok)
}

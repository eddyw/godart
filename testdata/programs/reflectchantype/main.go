package main

import (
	"fmt"
	"reflect"
)

type Receiver <-chan int

func same(actual reflect.Type, value any) {
	want := reflect.TypeOf(value)
	fmt.Println(actual == want, actual.String(), want.String())
}

func main() {
	recv := reflect.ChanOf(reflect.RecvDir, reflect.TypeOf(0))
	same(reflect.ChanOf(reflect.BothDir, recv), (chan (<-chan int))(nil))
	same(reflect.ChanOf(reflect.SendDir, recv), (chan<- <-chan int)(nil))
	same(reflect.ChanOf(reflect.RecvDir, recv), (<-chan <-chan int)(nil))
	same(reflect.ChanOf(reflect.BothDir, reflect.TypeOf(Receiver(nil))), chan Receiver(nil))
	record := reflect.TypeOf(struct{ Value int }{})
	inner := reflect.ChanOf(reflect.RecvDir, record)
	same(reflect.ChanOf(reflect.BothDir, inner), (chan (<-chan struct{ Value int }))(nil))
	same(reflect.StructOf([]reflect.StructField{{Name: "Value", Type: reflect.ChanOf(reflect.BothDir, recv)}}), struct{ Value chan (<-chan int) }{})
}

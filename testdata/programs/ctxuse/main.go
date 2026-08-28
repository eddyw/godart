package main

import (
	"context"
	"fmt"
	"time"
)

type key struct{ name string }

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Millisecond)
		cancel()
	}()
	<-ctx.Done()
	fmt.Println("cancel:", ctx.Err())

	timeout, stop := context.WithTimeout(context.Background(), time.Millisecond)
	<-timeout.Done()
	fmt.Println("timeout:", timeout.Err())
	stop()

	deadline, stop2 := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
	when, ok := deadline.Deadline()
	fmt.Println("deadline:", ok, when.After(time.Now()), deadline.Err())
	stop2()
	fmt.Println("after the stop:", deadline.Err())

	value := context.WithValue(context.Background(), key{name: "user"}, 42)
	fmt.Println("value:", value.Value(key{name: "user"}), value.Value(key{name: "other"}))

	inner := context.WithValue(value, key{name: "job"}, "build")
	fmt.Println("nested:", inner.Value(key{name: "user"}), inner.Value(key{name: "job"}))

	background := context.Background()
	fmt.Println("background:", background.Err(), background.Value(key{name: "user"}))
	fmt.Println("todo:", context.TODO().Err())

	cause, stop3 := context.WithCancelCause(context.Background())
	stop3(fmt.Errorf("the caller stopped it"))
	<-cause.Done()
	fmt.Println("cause:", cause.Err(), context.Cause(cause))

	fmt.Println("canceled:", context.Canceled)
	fmt.Println("exceeded:", context.DeadlineExceeded)
}

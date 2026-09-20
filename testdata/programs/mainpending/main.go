package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	ready := make(chan bool)
	go func() {
		ready <- true
		time.Sleep(time.Hour)
		panic("sleep resumed after main")
	}()
	<-ready
	time.AfterFunc(time.Hour, func() { panic("timer ran after main") })
	runtime.Gosched()
	fmt.Println("main returned")
}

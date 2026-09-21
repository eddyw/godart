package main

import (
	"fmt"
	"sync"
)

//gart:sync
type Builder func() int

var calls int
var cached = sync.OnceValue(func() int { calls++; return 42 })
var build Builder = func() int { return cached() }
var pair = sync.OnceValues(func() (int, bool) { return 7, true })
var pairBuild Builder = func() int { value, _ := pair(); return value }
var action = sync.OnceFunc(func() { calls++ })
var actionBuild Builder = func() int { action(); return calls }
var start = make(chan struct{})
var release = make(chan struct{})
var waitCalls int
var waiting = sync.OnceValue(func() int {
	waitCalls++
	close(start)
	<-release
	return 23
})
var panicCalls int
var failing = sync.OnceValue(func() int { panicCalls++; panic("cached panic") })

func caught() (out any) {
	defer func() { out = recover() }()
	failing()
	return nil
}

func main() {
	fmt.Println(build(), build(), pairBuild(), pairBuild())
	fmt.Println(actionBuild(), actionBuild(), calls)
	first, second := make(chan int, 1), make(chan int, 1)
	go func() { first <- waiting() }()
	<-start
	go func() { second <- waiting() }()
	close(release)
	fmt.Println(<-first, <-second, waitCalls)
	fmt.Println(caught(), caught(), panicCalls)
}

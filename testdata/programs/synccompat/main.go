package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

func caught(f func()) {
	defer func() { fmt.Println("panic", recover()) }()
	f()
}

func main() {
	var value atomic.Value
	fmt.Println("empty", value.Load(), value.CompareAndSwap(1, 2))
	fmt.Println("initial", value.CompareAndSwap(nil, 1), value.Load())
	value.Store(2)
	fmt.Println("swap", value.Swap(3), value.CompareAndSwap(3, 4), value.Load())
	caught(func() { value.Store(nil) })
	caught(func() { value.Store("bad") })
	caught(func() { value.Swap(nil) })
	caught(func() { value.Swap("bad") })
	caught(func() { value.CompareAndSwap(4, nil) })
	caught(func() { value.CompareAndSwap(4, "bad") })
	caught(func() { value.CompareAndSwap("bad", "bad") })
	fmt.Println("unchanged", value.Load())
	var typed atomic.Value
	var ptr *int
	typed.Store(ptr)
	fmt.Println("typed nil", typed.Load() == nil, typed.CompareAndSwap(ptr, new(int)))
	var slice atomic.Value
	slice.Store([]int{1})
	fmt.Println("slice", slice.Swap([]int{2}))
	var fresh atomic.Value
	fmt.Println("fresh swap", fresh.Swap("first"), fresh.Load())
	var rw sync.RWMutex
	rw.RLock()
	shared := rw.TryRLock()
	fmt.Println("shared", shared, "exclusive", rw.TryLock())
	if !shared {
		panic("a second reader cannot acquire the lock")
	}
	rw.RUnlock()
	started := make(chan bool)
	order := make(chan string, 2)
	release := make(chan bool)
	go func() {
		close(started)
		rw.Lock()
		order <- "writer"
		<-release
		rw.Unlock()
	}()
	<-started
	for rw.TryRLock() {
		rw.RUnlock()
		runtime.Gosched()
	}
	go func() {
		rw.RLock()
		order <- "reader"
		rw.RUnlock()
	}()
	rw.RUnlock()
	fmt.Println(<-order)
	close(release)
	fmt.Println(<-order)
	rw.Lock()
	fmt.Println("writer excludes", rw.TryRLock(), rw.TryLock())
	rw.Unlock()
	locker := rw.RLocker()
	locker.Lock()
	fmt.Println("rlocker", rw.TryRLock())
	rw.RUnlock()
	locker.Unlock()
	fmt.Println("free", rw.TryLock())
	rw.Unlock()
}

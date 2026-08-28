package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	var n64 atomic.Int64
	n64.Store(5)
	fmt.Println("int64:", n64.Add(3), n64.Load(), n64.Swap(1), n64.CompareAndSwap(1, 9), n64.Load())

	var flag atomic.Bool
	fmt.Println("bool:", flag.Load(), flag.Swap(true), flag.CompareAndSwap(true, false), flag.Load())

	var u32 atomic.Uint32
	u32.Store(7)
	fmt.Println("uint32:", u32.Add(1), u32.Load())

	var counter int64
	atomic.AddInt64(&counter, 4)
	atomic.StoreInt64(&counter, atomic.LoadInt64(&counter)+1)
	fmt.Println("function form:", counter)
	// The read of a variable and a call in 1 statement hold no order in Go,
	// thus the swap stands in a statement of its own.
	swapped := atomic.CompareAndSwapInt64(&counter, 5, 6)
	fmt.Println("compare and swap:", swapped, counter)

	// GOMAXPROCS, NumCPU, and GOARCH each hold a value of the target. Go reads
	// the machine, and gart writes 1 isolate and 1 architecture.
	fmt.Println("procs:", runtime.GOMAXPROCS(0) > 0, runtime.NumCPU() > 0)
	fmt.Println("os:", runtime.GOOS != "", runtime.GOARCH != "")

	start := time.Now()
	<-time.After(2 * time.Millisecond)
	fmt.Println("after:", time.Since(start) >= 2*time.Millisecond)

	timer := time.NewTimer(time.Millisecond)
	<-timer.C
	fmt.Println("timer:", timer.Stop())

	ticker := time.NewTicker(time.Millisecond)
	count := 0
	for range ticker.C {
		count++
		if count == 3 {
			ticker.Stop()
			break
		}
	}
	fmt.Println("ticker:", count)

	dir, err := os.MkdirTemp("", "gartcase*")
	fmt.Println("mkdirtemp:", err == nil, strings.Contains(dir, "gartcase"))
	f, err := os.CreateTemp(dir, "file*.txt")
	fmt.Println("createtemp:", err == nil, strings.HasSuffix(f.Name(), ".txt"))
	if _, err := f.WriteString("hello world"); err != nil {
		fmt.Println("write:", err)
	}
	f.Close()

	f2, _ := os.Open(f.Name())
	head := make([]byte, 5)
	n, err := f2.Read(head)
	fmt.Println("read:", n, string(head[:n]), err == nil)
	tail := make([]byte, 5)
	n, err = f2.ReadAt(tail, 6)
	fmt.Println("readat:", n, string(tail[:n]), err == nil)
	rest := make([]byte, 6)
	n, _ = f2.Read(rest)
	fmt.Println("the position stays:", n, string(rest[:n]))
	f2.Close()
	os.RemoveAll(dir)

	r := strings.NewReader("abcdef")
	part := make([]byte, 3)
	n, err = r.ReadAt(part, 2)
	fmt.Println("reader:", n, string(part[:n]), err == nil)
}

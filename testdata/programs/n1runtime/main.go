// The program proves the runtime names that the work of 2026-09-04 added:
// GoroutineProfile, StackRecord, Pinner, and SetDefaultGOMAXPROCS.
package main

import (
	"fmt"
	"runtime"
	"strings"
)

type node struct{ n int }

func main() {
	runtime.SetDefaultGOMAXPROCS()
	fmt.Println("procs:", runtime.GOMAXPROCS(0) >= 1)

	// A pin keeps the object alive, and Unpin drops the set.
	var pin runtime.Pinner
	p := &node{n: 7}
	pin.Pin(p)
	count := 3
	pin.Pin(&count)
	fmt.Println("pinned:", p.n, count)
	pin.Unpin()
	fmt.Println("unpinned:", p.n)

	// A short list reports the count and writes no record.
	var none []runtime.StackRecord
	n, ok := runtime.GoroutineProfile(none)
	fmt.Println("short:", n >= 1, ok)

	// 2 goroutines wait on a channel, thus the profile holds 3 records.
	done := make(chan int)
	hold := make(chan int)
	for i := 0; i < 2; i++ {
		go func() {
			<-hold
			done <- 1
		}()
	}
	runtime.Gosched()

	records := make([]runtime.StackRecord, 8)
	n, ok = runtime.GoroutineProfile(records)
	fmt.Println("count:", n, ok)
	fmt.Println("frames:", len(records[0].Stack()) > 0)
	fmt.Println("main:", strings.Contains(nameOf(records[0]), "main.main"))

	close(hold)
	<-done
	<-done
	fmt.Println("done")
}

// nameOf gives the name of each frame of a record, joined by a space.
func nameOf(r runtime.StackRecord) string {
	var out []string
	for _, pc := range r.Stack() {
		f := runtime.FuncForPC(pc)
		if f == nil {
			continue
		}
		out = append(out, f.Name())
	}
	return strings.Join(out, " ")
}

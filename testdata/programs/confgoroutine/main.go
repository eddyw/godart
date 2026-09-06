// The program proves the concurrency tools of Go: sync.WaitGroup, sync.Mutex,
// sync.RWMutex, sync.Once, sync.Map, sync/atomic, and the order that a chain
// of channels makes deterministic.
//
// The program writes no result that a schedule can change. Each count takes a
// lock, and each list takes a sort or a channel that fixes the order.
package main

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
)

func row(label string, value any) {
	fmt.Printf("%-34s %v\n", label, value)
}

func waitGroups() {
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			total += i
			mu.Unlock()
		}()
	}
	wg.Wait()
	row("the sum of 1 to 10", total)

	// A WaitGroup that a second Wait reads gives at once.
	wg.Wait()
	row("a second Wait", "gave")

	// Add of a count above 1.
	var wg2 sync.WaitGroup
	results := make([]int, 5)
	wg2.Add(5)
	for i := range results {
		go func() {
			defer wg2.Done()
			results[i] = i * i
		}()
	}
	wg2.Wait()
	row("each square", results)

	// A nested WaitGroup.
	var outer sync.WaitGroup
	count := int64(0)
	outer.Add(2)
	for g := 0; g < 2; g++ {
		go func() {
			defer outer.Done()
			var inner sync.WaitGroup
			inner.Add(3)
			for k := 0; k < 3; k++ {
				go func() {
					defer inner.Done()
					atomic.AddInt64(&count, 1)
				}()
			}
			inner.Wait()
		}()
	}
	outer.Wait()
	row("the nested count", count)

	// The Go method of a WaitGroup, which Go 1.25 added.
	var wg3 sync.WaitGroup
	n := int64(0)
	for i := 0; i < 4; i++ {
		wg3.Go(func() { atomic.AddInt64(&n, 2) })
	}
	wg3.Wait()
	row("the Go method", n)
}

func mutexes() {
	var mu sync.Mutex
	shared := 0
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			shared++
		}()
	}
	wg.Wait()
	row("100 increments", shared)

	// TryLock reports whether the lock was free.
	row("TryLock of a free lock", mu.TryLock())
	row("TryLock of a held lock", mu.TryLock())
	mu.Unlock()
	row("TryLock after Unlock", mu.TryLock())
	mu.Unlock()

	// A RWMutex holds a read lock and a write lock.
	var rw sync.RWMutex
	value := 0
	var wg2 sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			rw.Lock()
			value++
			rw.Unlock()
		}()
	}
	wg2.Wait()
	row("the write lock", value)
	rw.RLock()
	got := value
	rw.RUnlock()
	row("the read lock", got)
	row("TryRLock", rw.TryRLock())
	rw.RUnlock()
	row("TryLock of a RWMutex", rw.TryLock())
	rw.Unlock()
	// RLocker gives a Locker of the read side.
	l := rw.RLocker()
	l.Lock()
	l.Unlock()
	row("RLocker", "held and gave back")

	// A Mutex inside a struct.
	type safe struct {
		mu sync.Mutex
		m  map[string]int
	}
	s := safe{m: map[string]int{}}
	var wg3 sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg3.Add(1)
		go func() {
			defer wg3.Done()
			s.mu.Lock()
			s.m["k"]++
			s.mu.Unlock()
		}()
	}
	wg3.Wait()
	row("a map behind a lock", s.m["k"])
}

func onces() {
	var once sync.Once
	n := 0
	for i := 0; i < 5; i++ {
		once.Do(func() { n++ })
	}
	row("Do 5 times", n)

	// A Once across goroutines.
	var once2 sync.Once
	count := int64(0)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			once2.Do(func() { atomic.AddInt64(&count, 1) })
		}()
	}
	wg.Wait()
	row("Do in 10 goroutines", count)

	// OnceFunc, OnceValue, and OnceValues.
	calls := 0
	f := sync.OnceFunc(func() { calls++ })
	f()
	f()
	f()
	row("OnceFunc", calls)
	vcalls := 0
	g := sync.OnceValue(func() int { vcalls++; return 42 })
	row("OnceValue first", g())
	row("OnceValue second", g())
	row("OnceValue calls", vcalls)
	h := sync.OnceValues(func() (int, string) { return 1, "a" })
	a, b := h()
	c, d := h()
	row("OnceValues", fmt.Sprint(a, b, " ", c, d))

	// A Once that a panic reaches still counts as done.
	var once3 sync.Once
	tries := 0
	func() {
		defer func() { recover() }()
		once3.Do(func() { tries++; panic("in Do") })
	}()
	once3.Do(func() { tries++ })
	row("a panic inside Do", tries)
}

func atomics() {
	var i32 atomic.Int32
	i32.Store(5)
	row("Int32 Load", i32.Load())
	row("Int32 Add", i32.Add(3))
	row("Int32 Swap", i32.Swap(100))
	row("Int32 CAS true", i32.CompareAndSwap(100, 7))
	row("Int32 CAS false", i32.CompareAndSwap(100, 9))
	row("Int32 now", i32.Load())

	var i64 atomic.Int64
	i64.Add(1 << 40)
	row("Int64", i64.Load())
	var u32 atomic.Uint32
	u32.Store(1)
	u32.Add(^uint32(0))
	row("Uint32 wrap", u32.Load())
	var u64 atomic.Uint64
	u64.Store(1 << 63)
	row("Uint64", u64.Load())
	var b atomic.Bool
	row("Bool zero", b.Load())
	b.Store(true)
	row("Bool", b.Load())
	row("Bool CAS", b.CompareAndSwap(true, false))
	var v atomic.Value
	row("Value zero", v.Load())
	v.Store("text")
	row("Value", v.Load())
	row("Value Swap", v.Swap("other"))
	row("Value now", v.Load())
	var p atomic.Pointer[int]
	row("Pointer zero", p.Load() == nil)
	n := 5
	p.Store(&n)
	row("Pointer", *p.Load())

	// The function form.
	var raw int64 = 1
	row("AddInt64", atomic.AddInt64(&raw, 2))
	row("LoadInt64", atomic.LoadInt64(&raw))
	atomic.StoreInt64(&raw, 9)
	row("StoreInt64", raw)
	row("SwapInt64", atomic.SwapInt64(&raw, 10))
	row("CompareAndSwapInt64", atomic.CompareAndSwapInt64(&raw, 10, 11))
	row("raw now", raw)
	var rawU uint64 = 1
	row("AddUint64", atomic.AddUint64(&rawU, 2))
	var rawU32 uint32 = 0xF0
	row("AndUint32", atomic.AndUint32(&rawU32, 0x30))
	row("OrUint32", atomic.OrUint32(&rawU32, 0x0F))
	row("rawU32 now", rawU32)

	// A counter that many goroutines raise.
	var counter atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}
	wg.Wait()
	row("50 goroutines", counter.Load())
}

func syncMaps() {
	var m sync.Map
	m.Store("a", 1)
	m.Store("b", 2)
	v, ok := m.Load("a")
	row("Load", fmt.Sprint(v, " ", ok))
	v, ok = m.Load("z")
	row("Load of a gap", fmt.Sprint(v, " ", ok))
	v, loaded := m.LoadOrStore("c", 3)
	row("LoadOrStore new", fmt.Sprint(v, " ", loaded))
	v, loaded = m.LoadOrStore("c", 4)
	row("LoadOrStore old", fmt.Sprint(v, " ", loaded))
	v, loaded = m.LoadAndDelete("b")
	row("LoadAndDelete", fmt.Sprint(v, " ", loaded))
	m.Delete("a")
	keys := []string{}
	m.Range(func(k, _ any) bool {
		keys = append(keys, k.(string))
		return true
	})
	sort.Strings(keys)
	row("Range", keys)
	// A Range that stops early reads at least 1 key.
	seen := 0
	m.Range(func(_, _ any) bool { seen++; return false })
	row("Range that stops", seen)
}

// order proves the order that a chain of channels makes deterministic. Each
// goroutine waits for the value of the goroutine in front of it.
func order() {
	steps := []string{}
	const n = 5
	chans := make([]chan struct{}, n+1)
	for i := range chans {
		chans[i] = make(chan struct{})
	}
	done := make(chan struct{})
	for i := 0; i < n; i++ {
		go func() {
			<-chans[i]
			steps = append(steps, fmt.Sprint("step ", i))
			close(chans[i+1])
		}()
	}
	go func() {
		<-chans[n]
		close(done)
	}()
	close(chans[0])
	<-done
	row("the chain", steps)

	// A ping pong of 2 goroutines.
	ping := make(chan int)
	pong := make(chan int)
	log := []int{}
	go func() {
		for v := range ping {
			pong <- v + 1
		}
		close(pong)
	}()
	v := 0
	for i := 0; i < 4; i++ {
		ping <- v
		v = <-pong
		log = append(log, v)
	}
	close(ping)
	row("the ping pong", log)

	// A worker that gives its result on a channel of its own.
	results := make([]chan int, 3)
	for i := range results {
		results[i] = make(chan int, 1)
		go func() { results[i] <- i * 100 }()
	}
	out := []int{}
	for _, r := range results {
		out = append(out, <-r)
	}
	row("each result in order", out)
}

func main() {
	fmt.Println("== a WaitGroup ==")
	waitGroups()
	fmt.Println("== a Mutex ==")
	mutexes()
	fmt.Println("== a Once ==")
	onces()
	fmt.Println("== an atomic value ==")
	atomics()
	fmt.Println("== a sync.Map ==")
	syncMaps()
	fmt.Println("== the order ==")
	order()
}

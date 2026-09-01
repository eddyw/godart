//go:build dart

// Package sync gives the lock types of Go.
//
// All Go code runs in 1 Dart isolate, and a goroutine gives up control only at
// a block operation. Thus a lock needs a flag and a waiter queue, and not an
// atomic operation.
package sync

// A Mutex is a lock. The zero value is an open lock.
type Mutex struct {
	held    bool
	waiters []chan bool
}

// Lock takes the lock. It waits while another goroutine holds it.
func (m *Mutex) Lock() {
	for m.held {
		w := make(chan bool)
		m.waiters = append(m.waiters, w)
		<-w
	}
	m.held = true
}

// Unlock gives the lock back.
func (m *Mutex) Unlock() {
	if !m.held {
		panic("sync: unlock of unlocked mutex")
	}
	m.held = false
	if len(m.waiters) == 0 {
		return
	}
	w := m.waiters[0]
	m.waiters = m.waiters[1:]
	close(w)
}

// TryLock takes the lock when the lock is open.
func (m *Mutex) TryLock() bool {
	if m.held {
		return false
	}
	m.held = true
	return true
}

// A RWMutex is a lock for a reader and for a writer. The single isolate gives
// no parallel read, thus a read lock is equal to a write lock.
type RWMutex struct {
	mu Mutex
}

func (rw *RWMutex) Lock()          { rw.mu.Lock() }
func (rw *RWMutex) Unlock()        { rw.mu.Unlock() }
func (rw *RWMutex) RLock()         { rw.mu.Lock() }
func (rw *RWMutex) RUnlock()       { rw.mu.Unlock() }
func (rw *RWMutex) TryLock() bool  { return rw.mu.TryLock() }
func (rw *RWMutex) TryRLock() bool { return rw.mu.TryLock() }

// RLocker gives a Locker that takes the read lock.
func (rw *RWMutex) RLocker() Locker { return rlocker{rw} }

type rlocker struct{ rw *RWMutex }

func (r rlocker) Lock()   { r.rw.RLock() }
func (r rlocker) Unlock() { r.rw.RUnlock() }

// A Locker holds a lock and gives it back.
type Locker interface {
	Lock()
	Unlock()
}

// A Cond holds a queue of goroutines that wait for a change of a condition.
// Each goroutine of the queue holds the lock L while it tests the condition.
type Cond struct {
	// L holds the lock of the condition.
	L Locker

	waiters []chan bool
}

// NewCond gives a Cond with the lock l.
func NewCond(l Locker) *Cond { return &Cond{L: l} }

// Wait gives up the lock and waits for a Signal or a Broadcast. It takes the
// lock again before it gives control back.
func (c *Cond) Wait() {
	w := make(chan bool)
	c.waiters = append(c.waiters, w)
	c.L.Unlock()
	<-w
	c.L.Lock()
}

// Signal wakes 1 goroutine of the queue.
func (c *Cond) Signal() {
	if len(c.waiters) == 0 {
		return
	}
	w := c.waiters[0]
	c.waiters = c.waiters[1:]
	close(w)
}

// Broadcast wakes each goroutine of the queue.
func (c *Cond) Broadcast() {
	for _, w := range c.waiters {
		close(w)
	}
	c.waiters = nil
}

// A WaitGroup waits for a group of goroutines.
type WaitGroup struct {
	count   int
	waiters []chan bool
}

// Add changes the count of the group.
func (wg *WaitGroup) Add(delta int) {
	wg.count += delta
	if wg.count < 0 {
		panic("sync: negative WaitGroup counter")
	}
	if wg.count > 0 {
		return
	}
	for _, w := range wg.waiters {
		close(w)
	}
	wg.waiters = nil
}

// Done takes 1 away from the count.
func (wg *WaitGroup) Done() {
	wg.Add(-1)
}

// Go adds 1 to the count and runs f on a new goroutine. The count drops by 1
// after f gives control back.
func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}

// Wait waits until the count reaches 0.
func (wg *WaitGroup) Wait() {
	if wg.count == 0 {
		return
	}
	w := make(chan bool)
	wg.waiters = append(wg.waiters, w)
	<-w
}

// A Once runs a function 1 time.
type Once struct {
	done bool
}

// Do runs f when no earlier call of Do ran a function.
func (o *Once) Do(f func()) {
	if o.done {
		return
	}
	o.done = true
	f()
}

// OnceFunc gives a function that runs f 1 time. Each later call performs no
// operation.
func OnceFunc(f func()) func() {
	var once Once
	return func() { once.Do(f) }
}

// OnceValue gives a function that runs f 1 time and holds its result. Each
// later call gives the held result again.
func OnceValue[T any](f func() T) func() T {
	var once Once
	var out T
	return func() T {
		once.Do(func() { out = f() })
		return out
	}
}

// OnceValues gives a function that runs f 1 time and holds its 2 results.
func OnceValues[T1, T2 any](f func() (T1, T2)) func() (T1, T2) {
	var once Once
	var a T1
	var b T2
	return func() (T1, T2) {
		once.Do(func() { a, b = f() })
		return a, b
	}
}

// Pool holds a set of values that a caller reuses.
//
// All Go code runs in 1 Dart isolate, thus the pool needs no lock. Go may drop
// a held value at any time, and this pool holds each value until a Get takes
// it.
type Pool struct {
	// New builds a value when the pool holds none. A nil field gives a nil
	// result from Get.
	New func() any

	held []any
}

// Get takes a value out of the pool. It builds a value when the pool holds
// none.
func (p *Pool) Get() any {
	if n := len(p.held); n > 0 {
		out := p.held[n-1]
		p.held = p.held[:n-1]
		return out
	}
	if p.New == nil {
		return nil
	}
	return p.New()
}

// Put gives a value back to the pool.
func (p *Pool) Put(v any) {
	if v == nil {
		return
	}
	p.held = append(p.held, v)
}

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

func (rw *RWMutex) Lock()    { rw.mu.Lock() }
func (rw *RWMutex) Unlock()  { rw.mu.Unlock() }
func (rw *RWMutex) RLock()   { rw.mu.Lock() }
func (rw *RWMutex) RUnlock() { rw.mu.Unlock() }

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

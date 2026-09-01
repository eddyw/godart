package ui

import "slices"

// This file holds the reactive graph. A Signal holds a value. A read of a
// signal inside a tracked scope records a dependency, and a write of the signal
// marks each dependent scope dirty.
//
// The graph runs on 1 goroutine. gart puts all Go code in 1 Dart isolate, thus
// that rule costs nothing there. A plain Go program must keep the same rule: a
// background goroutine gives its result to the goroutine that owns the graph.

// scope is a tracked unit of work. A Watch of the host holds one, a Computed
// holds one, and an Effect holds one.
type scope struct {
	id    int
	run   func()
	deps  []*dep
	alive bool
}

// dep holds the scopes that read 1 value.
type dep struct {
	subs []int
}

var (
	scopes  []*scope
	freeIDs []int
	current *scope
	depth   int
	pending []int
)

func newScope(run func()) *scope {
	s := &scope{run: run, alive: true}
	if n := len(freeIDs); n > 0 {
		s.id = freeIDs[n-1]
		freeIDs = freeIDs[:n-1]
		scopes[s.id] = s
		return s
	}
	s.id = len(scopes)
	scopes = append(scopes, s)
	return s
}

func scopeByID(id int) *scope {
	if id < 0 || id >= len(scopes) {
		return nil
	}
	return scopes[id]
}

func (s *scope) drop() {
	s.alive = false
	s.clear()
	s.run = nil
	scopes[s.id] = nil
	freeIDs = append(freeIDs, s.id)
}

// clear removes the scope from each value that it read. The next run records
// the dependencies again.
func (s *scope) clear() {
	for _, d := range s.deps {
		out := d.subs[:0]
		for _, id := range d.subs {
			if id != s.id {
				out = append(out, id)
			}
		}
		d.subs = out
	}
	s.deps = nil
}

// enter runs fn with the scope as the tracked one, and it gives the previous
// scope back afterwards.
func (s *scope) enter(fn func()) {
	prev := current
	current = s
	s.clear()
	fn()
	current = prev
}

func (d *dep) track() {
	if current == nil {
		return
	}
	if slices.Contains(d.subs, current.id) {
		return
	}
	d.subs = append(d.subs, current.id)
	current.deps = append(current.deps, d)
}

func (d *dep) notify() {
	if len(d.subs) == 0 {
		return
	}
	ids := make([]int, len(d.subs))
	copy(ids, d.subs)
	depth++
	for _, id := range ids {
		mark(id)
	}
	depth--
	if depth == 0 {
		flush()
	}
}

func mark(id int) {
	s := scopeByID(id)
	if s == nil || !s.alive {
		return
	}
	if slices.Contains(pending, id) {
		return
	}
	pending = append(pending, id)
}

func flush() {
	for len(pending) > 0 {
		ids := pending
		pending = nil
		for _, id := range ids {
			s := scopeByID(id)
			if s != nil && s.alive && s.run != nil {
				s.run()
			}
		}
	}
}

// noCopy makes `go vet` report a copy of the value that holds it.
//
// The check copylocks reads the 2 methods below. A copy of a reactive value
// splits its subscriber list in a way that no error names, thus the check
// gives the only report of that mistake.
type noCopy struct{}

func (*noCopy) Lock() {}

func (*noCopy) Unlock() {}

// Signal holds 1 value. The zero value is ready, and it holds the zero value of
// T.
//
// A Signal must live behind a pointer. A copy of the struct takes a dep of its
// own that holds the subscribers of the moment, and the 2 lists then drift
// apart. `go vet` reports each copy.
type Signal[T any] struct {
	_ noCopy
	d dep
	v T
}

// NewSignal gives a signal that holds v.
func NewSignal[T any](v T) *Signal[T] { return &Signal[T]{v: v} }

// Get gives the value, and it records a dependency in the current scope.
func (s *Signal[T]) Get() T {
	s.d.track()
	return s.v
}

// Peek gives the value and records no dependency.
func (s *Signal[T]) Peek() T { return s.v }

// Set writes the value and marks each dependent scope dirty. It marks the
// scopes also when the new value equals the old one, because Go holds no
// equality operation for each type.
func (s *Signal[T]) Set(v T) {
	s.v = v
	s.d.notify()
}

// Update writes the result of f over the current value. It records no
// dependency for the read.
func (s *Signal[T]) Update(f func(T) T) { s.Set(f(s.v)) }

// Computed holds a value that comes from other signals. It runs its function
// again only after a read that follows a change.
type Computed[T any] struct {
	_     noCopy
	d     dep
	fn    func() T
	v     T
	valid bool
	s     *scope
}

// NewComputed gives a value that fn derives from other signals.
func NewComputed[T any](fn func() T) *Computed[T] {
	c := &Computed[T]{fn: fn}
	c.s = newScope(func() {
		c.valid = false
		c.d.notify()
	})
	return c
}

// Get gives the value. It runs the function again when a source changed.
func (c *Computed[T]) Get() T {
	if !c.valid {
		c.s.enter(func() { c.v = c.fn() })
		c.valid = true
	}
	c.d.track()
	return c.v
}

// Peek gives the value and records no dependency.
func (c *Computed[T]) Peek() T {
	if !c.valid {
		c.s.enter(func() { c.v = c.fn() })
		c.valid = true
	}
	return c.v
}

// Drop releases the computed value. A later Get panics.
func (c *Computed[T]) Drop() { c.s.drop() }

// Effect runs fn, and it runs fn again after a change of any signal that fn
// read. The result stops the effect.
func Effect(fn func()) func() {
	var s *scope
	s = newScope(func() { s.enter(fn) })
	s.run()
	return func() {
		if s.alive {
			s.drop()
		}
	}
}

// Batch runs fn and holds each notification until fn returns. A scope that 2
// writes touch runs 1 time.
func Batch(fn func()) {
	depth++
	fn()
	depth--
	if depth == 0 {
		flush()
	}
}

// Untracked runs fn with no dependency record.
func Untracked(fn func()) {
	prev := current
	current = nil
	fn()
	current = prev
}

// Tracked reports whether a scope records dependencies now.
func Tracked() bool { return current != nil }

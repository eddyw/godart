package ui

// This file holds the 3 shapes that a Go application takes for its state. Each
// one stands over the signal graph, and each one runs under `go test` with no
// Flutter and no Dart.
//
//   - Store gives the shape of ChangeNotifier: a method writes a field, and it
//     calls Notify.
//   - Reducer gives the shape of Redux and of bloc: a pure function takes the
//     state and 1 action, and it gives the next state.
//   - Command runs work that can block, and it holds Busy and Err in signals.

// mailbox carries work from a background goroutine to the goroutine that owns
// the graph. The graph runs on 1 goroutine, thus a goroutine must not write a
// signal of its own.
var mailbox = make(chan func(), 64)

// Post gives fn to the goroutine that owns the graph. Pump runs it.
func Post(fn func()) { mailbox <- fn }

// Pump runs each posted function and gives the count. The host calls it after
// each turn of the scheduler, and a test calls it after Wait.
func Pump() int {
	n := 0
	for {
		select {
		case fn := <-mailbox:
			fn()
			n++
		default:
			return n
		}
	}
}

// Store holds a subscriber list and no value. Embed it in a state struct to get
// the shape of ChangeNotifier.
//
//	type Cart struct {
//		ui.Store
//		items []Item
//	}
//
//	func (c *Cart) Add(i Item) { c.items = append(c.items, i); c.Notify() }
//
// A build that reads the state must call Track first.
type Store struct {
	_ noCopy
	d dep
}

// Track records a dependency of the current boundary on this store.
func (s *Store) Track() { s.d.track() }

// Notify marks each boundary that called Track.
func (s *Store) Notify() { s.d.notify() }

// Reducer holds 1 state value, and it applies an action with a pure function.
// The function takes no pointer, thus a Go test reads it as a table.
type Reducer[S any, A any] struct {
	_    noCopy
	sig  Signal[S]
	fn   func(S, A) S
	seen []A
}

// NewReducer gives a reducer over the start state.
func NewReducer[S any, A any](start S, fn func(S, A) S) *Reducer[S, A] {
	r := &Reducer[S, A]{fn: fn}
	r.sig.v = start
	return r
}

// State gives the current state, and it records a dependency.
func (r *Reducer[S, A]) State() S { return r.sig.Get() }

// Peek gives the current state and records no dependency.
func (r *Reducer[S, A]) Peek() S { return r.sig.Peek() }

// Do applies 1 action.
func (r *Reducer[S, A]) Do(a A) {
	r.seen = append(r.seen, a)
	r.sig.Set(r.fn(r.sig.v, a))
}

// Actions gives each action that Do took. A test reads it, and it names the
// order.
func (r *Reducer[S, A]) Actions() []A { return r.seen }

// Command runs 1 unit of work that can block. A build reads Busy and Err, and
// a build never waits.
type Command struct {
	_    noCopy
	busy Signal[bool]
	err  Signal[error]
	done Signal[bool]
	work func() error
	wait chan struct{}
}

// NewCommand gives a command over work. The work runs on a goroutine of its
// own, thus it may block.
func NewCommand(work func() error) *Command {
	return &Command{work: work}
}

// Busy reports whether the work runs now, and it records a dependency.
func (c *Command) Busy() bool { return c.busy.Get() }

// Err gives the error of the last run, and it records a dependency.
func (c *Command) Err() error { return c.err.Get() }

// Done reports whether a run finished, and it records a dependency.
func (c *Command) Done() bool { return c.done.Get() }

// Run starts the work. A second call during a run does nothing.
func (c *Command) Run() {
	if c.busy.Peek() {
		return
	}
	Batch(func() {
		c.busy.Set(true)
		c.err.Set(nil)
		c.done.Set(false)
	})
	work := c.work
	wait := make(chan struct{})
	c.wait = wait
	go func() {
		err := work()
		Post(func() { c.settle(err) })
		close(wait)
	}()
}

func (c *Command) settle(err error) {
	Batch(func() {
		c.err.Set(err)
		c.busy.Set(false)
		c.done.Set(true)
	})
}

// Wait blocks until the work finishes, and it applies the result. A test calls
// it. A build must never call it.
func (c *Command) Wait() {
	if c.wait == nil {
		return
	}
	<-c.wait
	c.wait = nil
	Pump()
}

// Command1 runs 1 unit of work that takes an argument.
type Command1[A any] struct {
	_    noCopy
	cmd  Command
	work func(A) error
}

// NewCommand1 gives a command over work that takes an argument.
func NewCommand1[A any](work func(A) error) *Command1[A] {
	return &Command1[A]{work: work}
}

// Busy reports whether the work runs now, and it records a dependency.
func (c *Command1[A]) Busy() bool { return c.cmd.Busy() }

// Err gives the error of the last run, and it records a dependency.
func (c *Command1[A]) Err() error { return c.cmd.Err() }

// Done reports whether a run finished, and it records a dependency.
func (c *Command1[A]) Done() bool { return c.cmd.Done() }

// Run starts the work with the argument.
func (c *Command1[A]) Run(a A) {
	work := c.work
	c.cmd.work = func() error { return work(a) }
	c.cmd.Run()
}

// Wait blocks until the work finishes, and it applies the result.
func (c *Command1[A]) Wait() { c.cmd.Wait() }

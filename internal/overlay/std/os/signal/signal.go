//go:build dart

// Package signal delivers an operating system signal to a Go channel.
//
// The real source of Go reaches the runtime, and the surface holds no such
// access. Each watch below reads ProcessSignal.watch of dart:io through the
// interop package, and 1 goroutine forwards each event to the channels.
//
// Signal delivery uses the signal watches that the Dart runtime supports.

package signal

import (
	"os"
	"syscall"
	"syscall/dart"
)

// watch holds 1 open watch of the runtime, and the channels that read it.
type watch struct {
	id    int
	users map[chan<- os.Signal]bool
}

var (
	watches = map[int]*watch{}
	// all holds each channel that Notify took with no signal.
	all = map[chan<- os.Signal]bool{}
	// ignored marks a signal that Ignore turned off.
	ignored = map[int]bool{}
)

// The signals that the runtime can watch. Each other signal gives no event.
var catchable = []syscall.Signal{
	syscall.SIGHUP,
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGUSR1,
	syscall.SIGUSR2,
	syscall.SIGWINCH,
}

func number(sig os.Signal) int {
	if s, ok := sig.(syscall.Signal); ok {
		return int(s)
	}
	return 0
}

// start opens a watch of 1 signal, and it starts the goroutine that forwards
// each event. A second call for the same signal reuses the open watch.
func start(n int) *watch {
	if w := watches[n]; w != nil {
		return w
	}
	id, cause := dart.SignalWatch(n)
	if cause != "" {
		return nil
	}
	w := &watch{id: id, users: map[chan<- os.Signal]bool{}}
	watches[n] = w
	go forward(w)
	return w
}

// forward reads each event of 1 watch and sends it to each channel. A send
// never blocks, as it does not in Go.
func forward(w *watch) {
	for {
		n, cause := dart.SignalWait(w.id)
		if cause != "" || n == 0 {
			return
		}
		if ignored[n] {
			continue
		}
		sig := syscall.Signal(n)
		for c := range w.users {
			select {
			case c <- sig:
			default:
			}
		}
		for c := range all {
			select {
			case c <- sig:
			default:
			}
		}
	}
}

// Notify sends each named signal to the channel. With no signal it sends each
// signal that the platform can catch.
//
// The send never blocks, thus the channel needs a buffer of 1 or more.
func Notify(c chan<- os.Signal, sig ...os.Signal) {
	if c == nil {
		panic("os/signal: Notify using nil channel")
	}
	if len(sig) == 0 {
		all[c] = true
		for _, s := range catchable {
			start(int(s))
		}
		return
	}
	for _, s := range sig {
		n := number(s)
		if n == 0 {
			continue
		}
		delete(ignored, n)
		if w := start(n); w != nil {
			w.users[c] = true
		}
	}
}

// Stop takes the channel out of each watch.
func Stop(c chan<- os.Signal) {
	delete(all, c)
	for n, w := range watches {
		delete(w.users, c)
		if len(w.users) == 0 && len(all) == 0 && !ignored[n] {
			reset(n)
		}
	}
}

// Reset takes each named signal out of each watch. With no signal it resets
// each signal that Notify or Ignore named.
//
// Reset does not undo Ignore, as it does not in Go: the disposition of the
// system stays, thus Ignored still reports true. Notify takes the signal back.
func Reset(sig ...os.Signal) {
	if len(sig) == 0 {
		for n := range watches {
			reset(n)
		}
		clear(all)
		return
	}
	for _, s := range sig {
		if n := number(s); n != 0 {
			reset(n)
		}
	}
}

func reset(n int) {
	w := watches[n]
	if w == nil {
		return
	}
	delete(watches, n)
	dart.SignalStop(w.id)
}

// Ignore turns each named signal off. With no signal it ignores each signal
// that Notify or Ignore named.
func Ignore(sig ...os.Signal) {
	if len(sig) == 0 {
		for n := range watches {
			ignored[n] = true
		}
		return
	}
	for _, s := range sig {
		n := number(s)
		if n == 0 {
			continue
		}
		ignored[n] = true
		start(n)
	}
}

// Ignored reports whether Ignore turned the signal off.
func Ignored(sig os.Signal) bool { return ignored[number(sig)] }

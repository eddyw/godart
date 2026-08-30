//go:build dart

// Package iter holds the 2 iterator types of the standard library package iter,
// and the 2 functions that turn a sequence into a pair of functions.
package iter

// Seq is a sequence of 1 value for each step.
type Seq[V any] func(yield func(V) bool)

// Seq2 is a sequence of 2 values for each step.
type Seq2[K, V any] func(yield func(K, V) bool)

// Pull turns a sequence into a pair of functions. next gives the next value of
// the sequence, and stop ends it. The caller must call stop after the last
// call of next.
//
// The Go runtime holds a coroutine for this work. The subset holds no
// coroutine, thus the sequence runs on a goroutine of its own. 2 channels with
// no buffer hand the control back and forth, and the 2 goroutines never run at
// the same time.
func Pull[V any](seq Seq[V]) (next func() (V, bool), stop func()) {
	resume := make(chan bool)
	values := make(chan V)
	done := make(chan bool)
	started := false
	over := false

	body := func() {
		stopped := false
		seq(func(v V) bool {
			if stopped {
				return false
			}
			values <- v
			if <-resume {
				return true
			}
			stopped = true
			return false
		})
		done <- true
	}

	next = func() (V, bool) {
		var zero V
		if over {
			return zero, false
		}
		if started {
			resume <- true
		} else {
			started = true
			go body()
		}
		select {
		case v := <-values:
			return v, true
		case <-done:
			over = true
			return zero, false
		}
	}

	stop = func() {
		if over {
			return
		}
		over = true
		if !started {
			return
		}
		// The goroutine of the sequence stands at the read of resume, because
		// the last call of next took its value.
		resume <- false
		<-done
	}
	return next, stop
}

// Pull2 turns a sequence of 2 values into a pair of functions. Read Pull.
func Pull2[K, V any](seq Seq2[K, V]) (next func() (K, V, bool), stop func()) {
	resume := make(chan bool)
	values := make(chan pair[K, V])
	done := make(chan bool)
	started := false
	over := false

	body := func() {
		stopped := false
		seq(func(k K, v V) bool {
			if stopped {
				return false
			}
			values <- pair[K, V]{k, v}
			if <-resume {
				return true
			}
			stopped = true
			return false
		})
		done <- true
	}

	next = func() (K, V, bool) {
		var zeroK K
		var zeroV V
		if over {
			return zeroK, zeroV, false
		}
		if started {
			resume <- true
		} else {
			started = true
			go body()
		}
		select {
		case p := <-values:
			return p.k, p.v, true
		case <-done:
			over = true
			return zeroK, zeroV, false
		}
	}

	stop = func() {
		if over {
			return
		}
		over = true
		if !started {
			return
		}
		resume <- false
		<-done
	}
	return next, stop
}

// pair holds the 2 values of 1 step of a Seq2.
type pair[K, V any] struct {
	k K
	v V
}

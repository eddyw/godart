//go:build dart

// Package iter provides push and pull iterators.
package iter

import "syscall/dart"

type Seq[V any] func(yield func(V) bool)
type Seq2[K, V any] func(yield func(K, V) bool)

// Pull converts a push iterator into next and stop functions.
func Pull[V any](seq Seq[V]) (next func() (V, bool), stop func()) {
	resume := make(chan struct{})
	yielded := make(chan struct{})
	var value V
	var ok, done, yieldNext, seqDone, exited bool
	var panicValue any
	go func() {
		<-resume
		if done {
			close(yielded)
			return
		}
		defer func() {
			panicValue = recover()
			exited = panicValue == nil && !seqDone
			done = true
			close(yielded)
		}()
		seq(func(v V) bool {
			if done {
				return false
			}
			if !yieldNext {
				panic("iter.Pull: yield called again before next")
			}
			yieldNext = false
			value, ok = v, true
			yielded <- struct{}{}
			<-resume
			return !done
		})
		var zero V
		value, ok = zero, false
		seqDone = true
	}()
	propagate := func() {
		if exited {
			dart.Goexit()
		}
		if panicValue != nil {
			panic(panicValue)
		}
	}
	next = func() (V, bool) {
		if done {
			var zero V
			return zero, false
		}
		if yieldNext {
			panic("iter.Pull: next called again before yield")
		}
		yieldNext = true
		resume <- struct{}{}
		<-yielded
		propagate()
		return value, ok
	}
	stop = func() {
		if done {
			return
		}
		done = true
		resume <- struct{}{}
		<-yielded
		propagate()
	}
	return next, stop
}

// Pull2 converts a push iterator into next and stop functions.
func Pull2[K, V any](seq Seq2[K, V]) (next func() (K, V, bool), stop func()) {
	resume := make(chan struct{})
	yielded := make(chan struct{})
	var key K
	var value V
	var ok, done, yieldNext, seqDone, exited bool
	var panicValue any
	go func() {
		<-resume
		if done {
			close(yielded)
			return
		}
		defer func() {
			panicValue = recover()
			exited = panicValue == nil && !seqDone
			done = true
			close(yielded)
		}()
		seq(func(k K, v V) bool {
			if done {
				return false
			}
			if !yieldNext {
				panic("iter.Pull2: yield called again before next")
			}
			yieldNext = false
			key, value, ok = k, v, true
			yielded <- struct{}{}
			<-resume
			return !done
		})
		var zeroK K
		var zeroV V
		key, value, ok = zeroK, zeroV, false
		seqDone = true
	}()
	propagate := func() {
		if exited {
			dart.Goexit()
		}
		if panicValue != nil {
			panic(panicValue)
		}
	}
	next = func() (K, V, bool) {
		if done {
			var zeroK K
			var zeroV V
			return zeroK, zeroV, false
		}
		if yieldNext {
			panic("iter.Pull2: next called again before yield")
		}
		yieldNext = true
		resume <- struct{}{}
		<-yielded
		propagate()
		return key, value, ok
	}
	stop = func() {
		if done {
			return
		}
		done = true
		resume <- struct{}{}
		<-yielded
		propagate()
	}
	return next, stop
}

//go:build dart

package testing

import (
	"internal/linkhook"
	"syscall/dart"
)

func init() {
	linkhook.SynctestTest = func(t any, f func(any)) bool {
		return runSynctest(t.(*T), f)
	}
}

// runSynctest runs the body of a synctest bubble under a T of its own, and it
// reports whether that T passed.
//
// The package testing/synctest reaches this function through linkhook. The
// call already stands on the first goroutine of the bubble, thus the body
// needs no goroutine of its own.
//
// Go gives the body a T with the name of the test that holds the bubble, and
// it writes no result line for that T. Each cleanup of the body runs inside
// the bubble, and each log line and each failure reach the outer test.
func runSynctest(t *T, f func(any)) bool {
	sub := &T{name: t.name, level: t.level, parent: t, synctest: true}
	id := dart.GoID()
	outer := current[id]
	current[id] = sub
	dart.Catch(func() { f(sub) })
	sub.stopping = false
	sub.out.flush()
	if sub.cancel != nil {
		sub.cancel()
	}
	sub.runCleanups()
	current[id] = outer
	// Go writes no result line for the T of a bubble, and a skip of it leaves
	// the outer test in the state PASS. The log lines and the failure pass up.
	t.logs = append(t.logs, sub.logs...)
	if sub.failed {
		t.failed = true
	}
	return !sub.failed
}

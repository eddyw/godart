//go:build dart

package debug

import (
	"os"
	"runtime"
	"time"
)

// Stack gives the stack trace of the goroutine that calls it.
//
// A function that gives up control leaves the Dart stack, thus the trace of a
// goroutine that waits holds the frames of the current state only.
// docs/parity.md holds that gap.
func Stack() []byte {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}

// PrintStack writes the stack trace of the goroutine that calls it to standard
// error.
func PrintStack() {
	os.Stderr.Write(Stack())
}

// FreeOSMemory gives up control after a collection. The collector of Dart owns
// the memory of the process, thus no memory reaches the operating system.
func FreeOSMemory() { runtime.GC() }

// gcPercent holds the last value of SetGCPercent. Go starts at 100.
var gcPercent = 100

// SetGCPercent writes the target of the collector and gives the value before
// the write. The collector of Dart reads no such target.
func SetGCPercent(percent int) int {
	old := gcPercent
	gcPercent = percent
	return old
}

// memoryLimit holds the last value of SetMemoryLimit.
var memoryLimit int64 = 1<<63 - 1

// SetMemoryLimit writes the limit of the memory and gives the value before the
// write. A negative value reads the limit and writes none.
func SetMemoryLimit(limit int64) int64 {
	old := memoryLimit
	if limit >= 0 {
		memoryLimit = limit
	}
	return old
}

// maxThreads holds the last value of SetMaxThreads.
var maxThreads = 10000

// SetMaxThreads writes the limit of the threads and gives the value before the
// write. All Go code runs in 1 isolate, thus the value reaches no operation.
func SetMaxThreads(threads int) int {
	old := maxThreads
	maxThreads = threads
	return old
}

// panicOnFault holds the last value of SetPanicOnFault.
var panicOnFault = false

// SetPanicOnFault writes the flag of a fault of an address and gives the value
// before the write. The subset holds no address, thus no fault occurs.
func SetPanicOnFault(enabled bool) bool {
	old := panicOnFault
	panicOnFault = enabled
	return old
}

// traceback holds the last value of SetTraceback.
var traceback = "single"

// SetTraceback writes the amount of the trace that a crash writes.
func SetTraceback(level string) { traceback = level }

// SetCrashOutput names a file that takes the trace of a crash. The Dart runtime
// writes each trace to standard error, thus the call reports ErrUnsupported for
// a file that is not nil.
func SetCrashOutput(f *os.File, opts CrashOptions) error {
	if f == nil {
		return nil
	}
	return errUnsupported
}

// CrashOptions holds the settings of SetCrashOutput. Go holds no field in it.
type CrashOptions struct{}

// WriteHeapDump writes a dump of the heap to a file descriptor. The Dart
// runtime writes no such dump, thus the call panics.
func WriteHeapDump(fd uintptr) {
	panic("runtime/debug: WriteHeapDump is outside the subset")
}

var errUnsupported = newError("unsupported operation")

func newError(text string) error { return &debugError{text} }

type debugError struct{ text string }

func (e *debugError) Error() string { return e.text }

// GCStats holds the numbers of the collector. The collector of Dart names none
// of them, thus ReadGCStats writes the zero value.
type GCStats struct {
	LastGC         time.Time
	NumGC          int64
	PauseTotal     time.Duration
	Pause          []time.Duration
	PauseEnd       []time.Time
	PauseQuantiles []time.Duration
}

// ReadGCStats writes the numbers of the collector. The collector of Dart names
// none of them, thus each field keeps the zero value.
func ReadGCStats(stats *GCStats) {}

// ReadBuildInfo gives no native Go build record for a Dart program.
func ReadBuildInfo() (*BuildInfo, bool) { return nil, false }

//go:build dart

package debug

import (
	"os"
	"runtime"
	"syscall/dart"
	"time"
)

// Stack gives the stack trace of the goroutine that calls it.
//
// A function that gives up control leaves the Dart stack, thus the trace of a
// goroutine that waits holds the frames of the current state only.
// docs/parity.md holds that gap.
func Stack() []byte {
	out := []byte("goroutine 1 [running]:\n")
	// Frame 0 is this function, thus the skip starts at 1.
	for _, frame := range dart.Stack(1) {
		name, file, line := splitFrame(frame)
		out = append(out, name...)
		out = append(out, "()\n\t"...)
		out = append(out, file...)
		out = append(out, ':')
		out = append(out, line...)
		out = append(out, '\n')
	}
	return out
}

// splitFrame reads 1 item of the stack of the runtime. The item holds
// "function|file|line".
func splitFrame(frame string) (string, string, string) {
	first := -1
	second := -1
	for i := 0; i < len(frame); i++ {
		if frame[i] != '|' {
			continue
		}
		if first < 0 {
			first = i
			continue
		}
		second = i
		break
	}
	if first < 0 || second < 0 {
		return frame, "", "0"
	}
	return frame[:first], frame[first+1 : second], frame[second+1:]
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

// ParseBuildInfo reads the text of a build record.
func ParseBuildInfo(data string) (*BuildInfo, error) {
	return nil, errUnsupported
}

// ReadBuildInfo gives the build record of the program. The Dart output holds no
// such record, thus the flag is false.
func ReadBuildInfo() (*BuildInfo, bool) { return nil, false }

// BuildInfo holds the build record of a program.
type BuildInfo struct {
	GoVersion string
	Path      string
	Main      Module
	Deps      []*Module
	Settings  []BuildSetting
}

// String gives the text of the record.
func (bi *BuildInfo) String() string { return "" }

// Module names 1 module of a build record.
type Module struct {
	Path    string
	Version string
	Sum     string
	Replace *Module
}

// BuildSetting holds 1 setting of a build record.
type BuildSetting struct {
	Key   string
	Value string
}

//go:build dart

// Package runtime gives the subset of the standard runtime package that the
// gart compiler supports.
//
// The scheduler of gart lives in the Dart runtime library, thus the package
// holds no control of a goroutine and no control of the garbage collector.
package runtime

import (
	"strconv"
	"strings"
	"syscall/dart"
)

// GOMAXPROCS gives the count of goroutines that run at the same time, and it
// takes no new value.
//
// All Go code of gart runs in 1 isolate, thus the count is always 1.
func GOMAXPROCS(n int) int { return 1 }

// SetDefaultGOMAXPROCS writes the default value of GOMAXPROCS again.
//
// All Go code of gart runs in 1 isolate. Thus the default is 1, GOMAXPROCS
// already gives 1, and the call needs no operation.
func SetDefaultGOMAXPROCS() {}

// NumCPU gives the count of processors that the program uses. The value is
// always 1, exactly as GOMAXPROCS.
func NumCPU() int { return 1 }

// NumGoroutine gives the count of goroutines that did not return. The
// scheduler holds each one, and the count holds the goroutine that calls.
func NumGoroutine() int { return len(dart.Parked()) + 1 }

// GOROOT gives the process-start GOROOT environment value.
// The Dart output contains no fallback Go SDK path.
func GOROOT() string {
	root, _ := dart.Getenv("GOROOT")
	return root
}

// GOOS names the operating system that runs the program. The value comes from
// Dart at the start of the program, because the output of 1 build runs on each
// system that the Dart VM holds.
var GOOS = dart.OS()

// GOARCH names the virtual 64-bit layout of Go values on all Dart targets.
const GOARCH = "dart64"

// Compiler names the tool chain that built the program. Go gives "gc" for its
// own tool chain, thus gart gives its own name.
const Compiler = "gart"

// Version gives the version of the tool chain that built the program.
//
// Go gives a text such as "go1.27.0". gart is no Go tool chain, thus the text
// names gart and its own number. internal/driver holds the same number, and
// TestVersionMatchesTheDriver compares the 2.
func Version() string { return "gart0.1.0-dev" }

// Gosched gives up control and lets each other goroutine run.
//
// The goroutine stands behind each goroutine of the run queue, and the
// scheduler runs it again after them.
func Gosched() { dart.Gosched() }

// Goexit ends the goroutine that calls it. Each deferred call of the goroutine
// runs first, and no other goroutine stops.
//
// Go keeps the program alive when the goroutine of main ends this way. The
// program then reports a crash after the last other goroutine ends.
func Goexit() { dart.Goexit() }

// KeepAlive keeps a value reachable up to the point of the call.
//
// The value reaches a function call here, thus the collector of Dart cannot
// free it before that call.
func KeepAlive(x any) {
	keepAlive = x
	keepAlive = nil
}

// keepAlive holds the last value of KeepAlive. A write of a package variable
// is a use that the Dart compiler cannot drop.
var keepAlive any

// LockOSThread binds the goroutine to the thread that runs it.
//
// All Go code of gart runs in 1 isolate, and that isolate runs on 1 thread.
// The goroutine already holds that thread, thus the call needs no operation.
func LockOSThread() {}

// UnlockOSThread drops the bind of LockOSThread. The bind needs no operation
// here, thus the drop needs none either.
func UnlockOSThread() {}

// NumCgoCall gives the count of calls of cgo that the program made. The subset
// holds no cgo, thus the count is always 0.
func NumCgoCall() int64 { return 0 }

// A PanicNilError is the value of a recover of `panic(nil)`. Go 1.21 and each
// later version give this value, thus recover reports the panic.
type PanicNilError struct {
	// The field keeps the type incomparable, exactly as Go does. A program
	// then cannot compare 2 values of the type by mistake.
	_ [0]*PanicNilError
}

func (*PanicNilError) Error() string {
	return "runtime error: panic called with nil argument"
}

func (*PanicNilError) RuntimeError() {}

// init installs the builder of the value of `panic(nil)`. The runtime library
// holds no class of an emitted Go type, thus it reads this builder.
func init() { dart.SetPanicNil(func() any { return &PanicNilError{} }) }

// frames holds the position of each program counter, and index holds the
// counter of each position. A Dart stack trace holds no address, thus a program
// counter is an index of the table. The table holds 1 entry for each call site
// that a program reads.
var (
	frames []string
	index  = map[string]uintptr{}
)

func record(text string) uintptr {
	if pc, ok := index[text]; ok {
		return pc
	}
	frames = append(frames, text)
	pc := uintptr(len(frames))
	index[text] = pc
	return pc
}

func at(pc uintptr) (string, string, int) {
	if pc == 0 || int(pc) > len(frames) {
		return "", "", 0
	}
	name, rest, _ := strings.Cut(frames[pc-1], "|")
	file, text, _ := strings.Cut(rest, "|")
	line, _ := strconv.Atoi(text)
	return name, file, line
}

// Caller gives the position of the call, and the count of skip names the frame.
// The value 0 names the caller of Caller.
//
// The file holds the base name only, because the source map of the output holds
// no directory.
func Caller(skip int) (pc uintptr, file string, line int, ok bool) {
	stack := dart.Stack(skip + 1)
	if len(stack) == 0 {
		return 0, "", 0, false
	}
	pc = record(stack[0])
	_, file, line = at(pc)
	return pc, file, line, true
}

// Callers writes the program counter of each frame into pc, and it gives the
// count. The count of skip names the first frame, and the value 0 names Callers
// itself.
func Callers(skip int, pc []uintptr) int {
	stack := dart.Stack(skip)
	n := 0
	for _, frame := range stack {
		if n == len(pc) {
			break
		}
		pc[n] = record(frame)
		n++
	}
	return n
}

// Frame holds the position of 1 call.
type Frame struct {
	PC       uintptr
	Func     *Func
	Function string
	File     string
	Line     int
	Entry    uintptr
}

// Frames reads the position of each program counter of a list.
type Frames struct {
	list []uintptr
}

func CallersFrames(callers []uintptr) *Frames {
	return &Frames{list: callers}
}

// Next gives the next frame, and the flag reports whether a frame follows it.
func (f *Frames) Next() (Frame, bool) {
	if len(f.list) == 0 {
		return Frame{}, false
	}
	pc := f.list[0]
	f.list = f.list[1:]
	name, file, line := at(pc)
	return Frame{PC: pc, Func: FuncForPC(pc), Function: name, File: file,
		Line: line, Entry: pc}, len(f.list) > 0
}

// Func names a function of the program.
type Func struct {
	pc uintptr
}

// FuncForPC gives the function of a program counter, and it gives nil for a
// counter that no call gave.
func FuncForPC(pc uintptr) *Func {
	if pc == 0 || int(pc) > len(frames) {
		return nil
	}
	return &Func{pc: pc}
}

func (f *Func) Name() string {
	name, _, _ := at(f.pc)
	return name
}

func (f *Func) FileLine(pc uintptr) (string, int) {
	_, file, line := at(pc)
	return file, line
}

func (f *Func) Entry() uintptr { return f.pc }

// Stack writes the stack trace of the goroutine that calls it into buf, and it
// gives the count of bytes that the write took. A trace that does not fit
// gives the part that fits.
//
// The flag all asks for the trace of each other goroutine after the trace of
// this one. A function that gives up control leaves the Dart stack, thus the
// trace of a goroutine that waits holds the frames of its saved state.
func Stack(buf []byte, all bool) int {
	text := make([]byte, 0, 512)
	text = append(text, "goroutine "...)
	text = strconv.AppendInt(text, int64(dart.GoID()), 10)
	text = append(text, " [running]:\n"...)
	// Frame 0 is this function, thus the skip starts at 1.
	for _, frame := range dart.Stack(1) {
		text = appendFrame(text, frame)
	}
	if all {
		for _, block := range dart.Parked() {
			head, rest, _ := strings.Cut(block, "\n")
			id, reason, _ := strings.Cut(head, "|")
			text = append(text, "\ngoroutine "...)
			text = append(text, id...)
			text = append(text, " ["...)
			text = append(text, reason...)
			text = append(text, "]:\n"...)
			for rest != "" {
				var frame string
				frame, rest, _ = strings.Cut(rest, "\n")
				text = appendFrame(text, frame)
			}
		}
	}
	return copy(buf, text)
}

// appendFrame writes 1 frame of the runtime in the form of Go. The item holds
// "function|file|line".
func appendFrame(out []byte, frame string) []byte {
	name, rest, _ := strings.Cut(frame, "|")
	file, line, _ := strings.Cut(rest, "|")
	out = append(out, name...)
	out = append(out, "()\n\t"...)
	out = append(out, file...)
	out = append(out, ':')
	out = append(out, line...)
	return append(out, '\n')
}

// A Cleanup names a cleanup that AddCleanup attached.
type Cleanup struct{ handle int }

// Stop drops the cleanup. A cleanup that already ran gives no operation.
func (c Cleanup) Stop() { dart.StopCleanup(c.handle) }

// AddCleanup attaches a function that runs after the collector frees the value
// that the pointer names.
//
// Dart gives no way to force a collection, thus the time of the run follows the
// collector of Dart. GC forces no run. Read docs/parity.md.
func AddCleanup[T, S any](ptr *T, cleanup func(S), arg S) Cleanup {
	if ptr == nil {
		panic("runtime: AddCleanup: ptr is nil")
	}
	return Cleanup{handle: dart.AddCleanup(ptr, func() { cleanup(arg) })}
}

// GC provokes a collection and gives up control, thus each cleanup of a value
// that no code reaches runs.
//
// Dart holds no call that forces a collection. The runtime library allocates a
// block of garbage for it. The collector follows no rule of Go, thus a cleanup
// can stay for a later call. Read docs/parity.md.
func GC() { dart.GC() }

// An Error is a run time error of Go. A recover of a run time panic gives a
// value that holds this interface.
type Error interface {
	error
	// RuntimeError marks the type as a run time error.
	RuntimeError()
}

// A TypeAssertionError names a type assertion that failed.
type TypeAssertionError struct {
	text string
}

func (e *TypeAssertionError) RuntimeError() {}

func (e *TypeAssertionError) Error() string { return e.text }

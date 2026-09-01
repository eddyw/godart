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

// NumCPU gives the count of processors that the program uses. The value is
// always 1, exactly as GOMAXPROCS.
func NumCPU() int { return 1 }

// NumGoroutine gives the count of goroutines. The runtime library holds no such
// count, thus the value is always 1.
func NumGoroutine() int { return 1 }

// GOROOT gives the root of the Go tree. The Dart output holds no such tree,
// thus the text is empty. Go marks the function as deprecated.
func GOROOT() string { return "" }

// GOOS names the operating system that runs the program. The value comes from
// Dart at the start of the program, because the output of 1 build runs on each
// system that the Dart VM holds.
var GOOS = dart.OS()

// GOARCH names the architecture. A Dart int holds 64 bits on the VM and with
// AOT, and the compiler rejects the web target. Thus 1 value serves each
// machine.
const GOARCH = "dart64"

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

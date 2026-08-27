//go:build dart

// Package dart gives Go code access to the Dart runtime.
//
// The compiler replaces each call to this package with a call to the runtime
// library, thus no body below ever runs.
package dart

// Print writes text to standard output.
func Print(text string) {
	panic("syscall/dart needs the gart compiler")
}

// Format writes 1 value in the format of a Go verb.
//
// The verb holds the flags, the precision, and the letter, for example "v",
// "+v", or ".2f". The width belongs to the caller.
func Format(verb string, value any) string {
	panic("syscall/dart needs the gart compiler")
}

// Kind gives the kind of a value, in the words of the package reflect.
func Kind(value any) string {
	panic("syscall/dart needs the gart compiler")
}

// Upper gives the text in upper case, with the rules of Unicode.
func Upper(text string) string {
	panic("syscall/dart needs the gart compiler")
}

// Lower gives the text in lower case, with the rules of Unicode.
func Lower(text string) string {
	panic("syscall/dart needs the gart compiler")
}

// ParseFloat reads a float from text. The second result is false when the text
// holds no float.
func ParseFloat(text string) (float64, bool) {
	panic("syscall/dart needs the gart compiler")
}

// Sleep gives up control for a count of nanoseconds. The call blocks the
// goroutine, thus the caller becomes an async function.
func Sleep(nanoseconds int64) {
	panic("syscall/dart needs the gart compiler")
}

// NowNanos gives the count of nanoseconds since the start of the Unix epoch.
func NowNanos() int64 {
	panic("syscall/dart needs the gart compiler")
}

// Worker runs f in a new isolate and gives the result.
//
// The function must be a pure function of its input. It must not read a
// package variable, and it must not write one. In the Go build, the same API
// runs the function on a goroutine, thus the result stays equal.
//
// The call blocks the goroutine, thus the caller becomes an async function.
func Worker(f func([]byte) []byte, input []byte) []byte {
	panic("syscall/dart needs the gart compiler")
}

// SliceLen gives the length of a slice of an unknown element type.
func SliceLen(slice any) int {
	panic("syscall/dart needs the gart compiler")
}

// SliceSwap exchanges 2 elements of a slice of an unknown element type.
func SliceSwap(slice any, i, j int) {
	panic("syscall/dart needs the gart compiler")
}

// Stop stops the call chain of the goroutine. Each deferred function of the
// chain still runs.
//
// The package testing uses it for FailNow. Go stops such a chain with
// runtime.Goexit.
func Stop() {
	panic("syscall/dart needs the gart compiler")
}

// Catch calls f and holds a stop signal.
//
// It reports false when f called Stop, and true in each other case. A panic
// passes through. The call gives up control, thus the caller becomes an async
// function.
func Catch(f func()) bool {
	panic("syscall/dart needs the gart compiler")
}

// Exit ends the program with a status code.
func Exit(code int) {
	panic("syscall/dart needs the gart compiler")
}

// Args gives the arguments of the program, without the name of the program.
func Args() []string {
	panic("syscall/dart needs the gart compiler")
}

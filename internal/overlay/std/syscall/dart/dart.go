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

// SliceLen gives the length of a slice of an unknown element type.
func SliceLen(slice any) int {
	panic("syscall/dart needs the gart compiler")
}

// SliceSwap exchanges 2 elements of a slice of an unknown element type.
func SliceSwap(slice any, i, j int) {
	panic("syscall/dart needs the gart compiler")
}

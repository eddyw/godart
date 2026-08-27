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

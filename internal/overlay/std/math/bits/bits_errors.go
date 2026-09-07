//go:build dart

// This file patches the file of the same name of the package math/bits.
//
// The real file reaches 2 variables of the runtime package with go:linkname,
// and that directive needs the package unsafe. The 2 variables hold the error
// of an overflow and the error of a division by zero.
package bits

// A runtimeError is an error of the run time. Go gives the 2 values below the
// type runtime.errorString, thus a caller reads runtime.Error of them.
type runtimeError string

func (e runtimeError) RuntimeError() {}

func (e runtimeError) Error() string { return string(e) }

var overflowError error = runtimeError("runtime error: integer overflow")

var divideError error = runtimeError("runtime error: integer divide by zero")

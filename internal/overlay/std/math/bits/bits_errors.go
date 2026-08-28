//go:build dart

// This file patches the file of the same name of the package math/bits.
//
// The real file reaches 2 variables of the runtime package with go:linkname,
// and that directive needs the package unsafe. The 2 variables hold the error
// of an overflow and the error of a division by zero.
package bits

import "errors"

var overflowError error = errors.New("runtime error: integer overflow")

var divideError error = errors.New("runtime error: integer divide by zero")

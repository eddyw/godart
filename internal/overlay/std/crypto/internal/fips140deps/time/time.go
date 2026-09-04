//go:build dart

// This file patches the file time.go of the package
// crypto/internal/fips140deps/time.
//
// The real file reads the monotonic clock of the runtime through a go:linkname
// directive. This file reads the same clock through the package time.
package time

import "time"

// HighPrecisionNow gives a timestamp of high resolution. The unit, the epoch,
// and the resolution each stay unnamed. A program measures a difference with
// it, and never an absolute time.
func HighPrecisionNow() int64 {
	return time.Now().UnixNano()
}

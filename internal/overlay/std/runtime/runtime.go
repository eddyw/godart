//go:build dart

// Package runtime gives the subset of the standard runtime package that the
// gart compiler supports.
//
// The scheduler of gart lives in the Dart runtime library, thus the package
// holds no control of a goroutine and no control of the garbage collector.
package runtime

// GOMAXPROCS gives the count of goroutines that run at the same time, and it
// takes no new value.
//
// All Go code of gart runs in 1 isolate, thus the count is always 1.
func GOMAXPROCS(n int) int { return 1 }

//go:build dart

// Package runtime gives the subset of the standard runtime package that the
// gart compiler supports.
//
// The scheduler of gart lives in the Dart runtime library, thus the package
// holds no control of a goroutine and no control of the garbage collector.
package runtime

import "syscall/dart"

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

// GOOS names the operating system that runs the program. The value comes from
// Dart at the start of the program, because the output of 1 build runs on each
// system that the Dart VM holds.
var GOOS = dart.OS()

// GOARCH names the architecture. A Dart int holds 64 bits on the VM and with
// AOT, and the compiler rejects the web target. Thus 1 value serves each
// machine.
const GOARCH = "dart64"

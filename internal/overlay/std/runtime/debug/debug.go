//go:build dart

// Package debug gives the runtime knobs that the tests of the Go repository
// read. The Dart runtime holds no such knob, thus a write keeps the value
// only.
package debug

// maxStack holds the last value of SetMaxStack. Go starts with 1 GB on a
// 64-bit machine.
var maxStack = 1 << 30

// SetMaxStack writes the limit and gives the value before the write.
func SetMaxStack(n int) int {
	old := maxStack
	maxStack = n
	return old
}

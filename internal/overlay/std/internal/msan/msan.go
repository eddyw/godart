//go:build dart

// Package msan gives the flag of the memory sanitizer of Go. A build of Go
// sets the flag with -msan, and the Dart output holds no sanitizer, thus the
// flag stays off.
//
// The real package also holds Read, Write, Malloc, Free, and Move. Each 1
// takes an unsafe.Pointer and reaches the runtime of Go, thus the subset holds
// none of them.
package msan

// Enabled reports whether the memory sanitizer runs.
const Enabled = false

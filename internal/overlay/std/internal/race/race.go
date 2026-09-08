//go:build dart

// Package race gives the flag of the race detector of Go. A build of Go sets
// the flag with -race, and the Dart output holds no detector, thus the flag
// stays off.
//
// The real package also holds Acquire, Release, Read, Write, and 6 more. Each
// 1 takes an unsafe.Pointer and reaches the runtime of Go, thus the subset
// holds none of them.
package race

// Enabled reports whether the race detector runs.
const Enabled = false

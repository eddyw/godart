//go:build dart

package main

import "syscall/dart"

// runWorker starts f in a second isolate.
func runWorker(f func([]byte) []byte, input []byte) []byte {
	return dart.Worker(f, input)
}

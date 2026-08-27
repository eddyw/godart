//go:build !dart

package main

// runWorker starts f on a goroutine. The worker contract keeps the result
// equal to the result of the isolate.
func runWorker(f func([]byte) []byte, input []byte) []byte {
	done := make(chan []byte, 1)
	go func() { done <- f(input) }()
	return <-done
}

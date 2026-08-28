// Package chain measures the cost of the async model of gart.
//
// The block analysis of stage 3 marks each function that can block, and such a
// function becomes a Dart async function. AsyncChain touches a channel 1 time
// in its last frame, thus each frame of the chain becomes async. SyncChain does
// the same work with no channel. The difference of the 2 benchmarks is the cost
// of the async call.
package chain

// Depth is the count of frames of a chain.
const Depth = 20

// Sink holds a result. The Dart compiler removes code that no other code reads.
var Sink int

// SyncChain calls itself n times. No frame blocks, thus each frame stays
// synchronous.
func SyncChain(n int) int {
	if n == 0 {
		return 0
	}
	return n + SyncChain(n-1)
}

// AsyncChain calls itself n times. The last frame reads a channel, thus stage 3
// marks each frame of the chain.
func AsyncChain(n int) int {
	if n == 0 {
		done := make(chan int, 1)
		done <- 0
		return <-done
	}
	return n + AsyncChain(n-1)
}

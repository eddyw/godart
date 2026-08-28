//go:build dart

// Package iter holds the 2 iterator types of the standard library package iter.
//
// The compiler cannot read the real source. The functions Pull and Pull2 of
// that package need a coroutine of the Go runtime, and the Dart VM gives no
// such primitive. The 2 functions stay out.
package iter

// Seq is a sequence of 1 value for each step.
type Seq[V any] func(yield func(V) bool)

// Seq2 is a sequence of 2 values for each step.
type Seq2[K, V any] func(yield func(K, V) bool)

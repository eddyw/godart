//go:build dart

// This file patches the file hypot_asm.go of the package math.
//
// The real file declares each function with no body, because assembly holds
// the code. This file holds the pure Go form of the same package, thus the flag
// stays false and the caller takes the portable path.
//
// The go command compiles the real file only on 386 and on amd64. A host of
// another architecture holds no such file, and the load leaves this patch out.
package math

const haveArchHypot = false

func archHypot(p, q float64) float64 {
	panic("not implemented")
}

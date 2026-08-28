// Package fail holds a test that fails, thus the TestMain hook reads a status
// that is not 0.
package fail

func Double(v int) int { return v * 3 }

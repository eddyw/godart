// Package target holds a test, a benchmark, a fuzz target, and an example.
// gart runs each 1 that go test runs, and the 2 outputs are equal.
package target

// Length gives the count of bytes of the text.
func Length(text string) int { return len(text) }

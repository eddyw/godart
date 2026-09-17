//go:build dart

package cryptotest

import "testing"

// Dart cannot allocate protected native memory pages.
func BoundarySlices(t *testing.T, size int) (start, end []byte) {
	t.Helper()
	t.Skip("native memory page protection is unavailable")
	return nil, nil
}

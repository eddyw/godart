package driver_test

import "testing"

func TestCompilerConcurrencyOriginal(t *testing.T) {
	runCompilerAssertionFiles(t, "compiler-concurrent-slices", []string{"dupLoad_test.go", "slice_test.go", "deferNoReturn_test.go"})
}

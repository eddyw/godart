package driver_test

import "testing"

func TestCompilerCompoundReturns(t *testing.T) {
	runCompilerAssertionFiles(t, "compiler-compound-returns", []string{"compound_test.go", "namedReturn_test.go"})
}

package driver_test

import "testing"

func TestCompilerValuesControl(t *testing.T) {
	runCompilerAssertionFiles(t, "compiler-values-control", []string{"array_test.go", "append_test.go", "closure_test.go", "cmp_test.go", "short_test.go", "break_test.go", "divbyzero_test.go"})
}

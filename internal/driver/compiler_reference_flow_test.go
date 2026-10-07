package driver_test

import "testing"

func TestCompilerReferenceFlow(t *testing.T) {
	runCompilerAssertionFiles(t, "compiler-reference-flow", []string{"addressed_test.go", "loadstore_test.go", "map_test.go", "chan_test.go", "regalloc_test.go", "ctl_test.go", "string_test.go"})
}

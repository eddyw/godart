package driver_test

import "testing"

func TestCompilerArithmeticOriginal(t *testing.T) {
	runCompilerAssertionFiles(t, "compiler-arithmetic", []string{"arith_test.go", "arithBoundary_test.go", "arithConst_test.go", "cmpConst_test.go", "fp_test.go", "sqrtConst_test.go", "copy_test.go", "zero_test.go"})
}

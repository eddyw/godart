package driver_test

import (
	"bytes"
	"go/build"
	"path/filepath"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestKernelUpstreamParity(t *testing.T) {
	for _, suite := range []struct{ pkg, run string }{
		{"path", ""},
		{"io", ""},
		{"math/bits", ""},
		{"hash/crc32", ""},
		{"fmt", "^(TestErrorf|TestSprintf|TestScanf|TestScan)$"},
	} {
		pkg := suite.pkg
		t.Run(pkg, func(t *testing.T) {
			dir := filepath.Join(build.Default.GOROOT, "src", pkg)
			compiles, incremental := 0, 0
			opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: t.TempDir(), Runtime: runtimeDir(t), TrimPath: true,
				Trace: func(phase string, _ time.Duration) {
					if phase == "kernel-incremental" {
						incremental++
					}
					if phase == "kernel-compile" {
						compiles++
					}
				}}
			for iteration := range 3 {
				if iteration == 2 {
					opt.TrimPath = false
				}
				var output bytes.Buffer
				code, err := driver.Test(opt, driver.TestOptions{Verbose: true, Short: testing.Short(), Run: suite.run}, &output)
				if err != nil {
					t.Fatal(err)
				}
				if code != 0 {
					t.Log(output.String())
				}
				compareUpstream(t, "kernel/"+pkg, pkg, suite.run, "vm", dir, output.String(), code)
			}
			if compiles != 2 || incremental != 1 {
				t.Fatalf("compiled %d kernels, %d incremental, want 2 and 1", compiles, incremental)
			}
		})
	}
}

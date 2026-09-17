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
		{"fmt", "^(TestErrorf|TestSprintf|TestScanf|TestScan)$"},
	} {
		pkg := suite.pkg
		t.Run(pkg, func(t *testing.T) {
			dir := filepath.Join(build.Default.GOROOT, "src", pkg)
			compiles := 0
			opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: t.TempDir(), Runtime: runtimeDir(t), TrimPath: true,
				Trace: func(phase string, _ time.Duration) {
					if phase == "kernel-compile" {
						compiles++
					}
				}}
			for range 2 {
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
			if compiles != 1 {
				t.Fatalf("compiled %d kernels, want 1", compiles)
			}
		})
	}
}

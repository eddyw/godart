package driver_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestSourceFrontendExecution(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/frontend\n\ngo 1.27.0\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "os")
func main() { fmt.Println(os.Args[1]); os.Exit(7) }
`)
	opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: t.TempDir(), Runtime: runtimeDir(t)}
	for _, frontend := range []string{"source", "resident"} {
		t.Run(frontend, func(t *testing.T) {
			opt.Frontend = frontend
			var output bytes.Buffer
			code, err := driver.Run(opt, []string{"two words"}, &output)
			if err != nil || code != 7 || output.String() != "two words\n" {
				t.Fatalf("code %d, error %v, output %q", code, err, output.String())
			}
		})
	}
	writeFile(t, filepath.Join(dir, "main_test.go"), `package main
import "testing"
func TestSource(t *testing.T) { t.Log("source frontend") }
`)
	opt.Frontend = "source"
	opt.Trace = func(phase string, _ time.Duration) {
		if strings.HasPrefix(phase, "kernel-") {
			t.Errorf("source mode started %s", phase)
		}
	}
	var output bytes.Buffer
	code, err := driver.Test(opt, driver.TestOptions{Verbose: true}, &output)
	if err != nil || code != 0 || !strings.Contains(output.String(), "PASS: TestSource") {
		t.Fatalf("code %d, error %v, output %q", code, err, output.String())
	}
}

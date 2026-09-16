package driver_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestMonotonicClock(t *testing.T) {
	dir, err := filepath.Abs("../overlay/std/time")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := driver.Test(driver.Options{Dir: dir, Patterns: []string{"."}, Out: t.TempDir()}, driver.TestOptions{Verbose: true}, &out)
	if err != nil || code != 0 {
		t.Fatalf("time tests failed: %v, exit %d\n%s", err, code, out.String())
	}
	for _, name := range []string{"TestMonotonicWallChange", "TestMonotonicPreservation", "TestMonotonicBounds"} {
		if !strings.Contains(out.String(), "--- PASS: "+name) {
			t.Fatalf("missing %s\n%s", name, out.String())
		}
	}
}

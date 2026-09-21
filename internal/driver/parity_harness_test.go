package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestParityHarness(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/parityharness\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "parity_test.go"), `package parityharness
import "testing"
func TestPass(t *testing.T) {
 t.Logf("child output:\n--- PASS: TestPass (0.01s)\n    --- PASS: TestPass/kept (0.01s)")
 t.Run("kept", func(t *testing.T) {})
 t.Run("excluded", func(t *testing.T) {})
}
func TestFail(t *testing.T) { t.Fatal("intentional assertion failure") }
func TestSkip(t *testing.T) { t.Skip("intentional unknown skip") }
`)
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, name := range []string{"TestPass", "TestFail", "TestSkip"} {
		t.Run(name, func(t *testing.T) {
			native := programOutcome(t, dir, "go", "test", "-json", "-count=1", "-run=^"+name+"$", ".")
			gart := programOutcome(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v", "-test.run=^"+name+"$")
			n, err := parity.NativeResults(strings.NewReader(native.Text), native.Exit)
			if err != nil {
				t.Fatal(err)
			}
			g, err := parity.GartResults(strings.NewReader(gart.Text), gart.Exit)
			if err != nil {
				t.Fatal(err)
			}
			c := parity.Compare(n, g, []string{name}, nil)
			if (len(c.Problems) == 0) != (name == "TestPass") {
				t.Fatalf("unexpected acceptance: %+v", c)
			}
			if name == "TestFail" && (native.Exit == 0 || gart.Exit == 0) {
				t.Fatalf("failed assertion returned success: Go=%d Gart=%d", native.Exit, gart.Exit)
			}
			if name != "TestPass" {
				return
			}
			filtered := programOutcome(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v", "-test.run=^TestPass$", "-test.skip=^TestPass$/^excluded$")
			g, err = parity.GartResults(strings.NewReader(filtered.Text), filtered.Exit)
			if err != nil {
				t.Fatal(err)
			}
			if g.Outcomes["TestPass"] != "pass" || g.Outcomes["TestPass/kept"] != "pass" || g.Outcomes["TestPass/excluded"] != "" {
				t.Fatalf("subtest exclusion changed its parent or sibling: %+v", g)
			}
			if c := parity.Compare(n, g, []string{name}, nil); len(c.Problems) == 0 {
				t.Fatal("missing subtest passed the parity gate")
			}
		})
	}
}

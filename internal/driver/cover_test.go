package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/overlay"
)

// TestTestsCover proves that the -cover flag of gart test reports the same
// part of the statements as go test. The module cover holds a function that no
// test reaches, thus the number stays below 100.
func TestTestsCover(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/cover")
	if err != nil {
		t.Fatal(err)
	}
	want := goTest(t, dir, true, "-cover")
	got := gartTest(t, dir, driver.TestOptions{Verbose: true, Cover: true})
	if got != want {
		t.Errorf("the output differs\ngo test:\n%s\ngart test:\n%s", want, got)
	}
	if !strings.Contains(got, "coverage: 63.6% of statements") {
		t.Errorf("the output holds no real count of the statements:\n%s", got)
	}
}

// TestTestsCoverWithNoStatement proves that a package with no statement reports
// the words of go test. The module env holds such a package.
func TestTestsCoverWithNoStatement(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/env")
	if err != nil {
		t.Fatal(err)
	}
	want := goTest(t, dir, false, "-cover")
	got := gartTest(t, dir, driver.TestOptions{Cover: true})
	if got != want {
		t.Errorf("the output differs\ngo test:\n%s\ngart test:\n%s", want, got)
	}
	if !strings.Contains(got, "coverage: [no statements]") {
		t.Errorf("the output names no empty package:\n%s", got)
	}
}

// TestTestsShort proves that the -short flag of gart test reaches
// testing.Short. The module cover holds a test that skips itself in the short
// form.
func TestTestsShort(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/cover")
	if err != nil {
		t.Fatal(err)
	}
	want := goTest(t, dir, true, "-short")
	got := gartTest(t, dir, driver.TestOptions{Verbose: true, Short: true})
	if got != want {
		t.Errorf("the output differs\ngo test:\n%s\ngart test:\n%s", want, got)
	}
	if !strings.Contains(got, "--- SKIP: TestShort") {
		t.Errorf("the short form skipped no test:\n%s", got)
	}
}

// TestTestsCount proves that the -count flag of gart test runs the whole list
// again. A count of 0 runs no test, exactly as go test does.
func TestTestsCount(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/cover")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	for _, count := range []int{2, 0} {
		want := goTest(t, dir, true, "-count="+itoa(count))
		got := gartTestIn(t, dir, out, driver.TestOptions{Verbose: true, Count: count, CountGiven: true})
		if got != want {
			t.Errorf("the output of -count=%d differs\ngo test:\n%s\ngart test:\n%s",
				count, want, got)
		}
	}
}

// TestTestsFlagWithASpace proves that the harness of a test binary reads a flag
// whose value stands in the next argument, exactly as the flag package of Go
// does. gart test writes the form with the equal sign, and a caller that runs
// the binary itself can write either form.
func TestTestsFlagWithASpace(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/cover")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtimeDir(t),
		Tests:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("the module gave %d test binaries, and 1 is the answer", len(stats.Binaries))
	}
	run(t, out, "dart", "pub", "get")
	entry := filepath.Join(out, stats.Binaries[0].Entry)
	for _, c := range []struct {
		args []string
		runs int
	}{
		{[]string{"-test.v", "-test.run", "TestClassify"}, 1},
		{[]string{"-test.v", "-test.count", "2", "-test.run", "TestClassify"}, 2},
		{[]string{"-test.v", "--test.run", "TestClassify"}, 1},
	} {
		got := run(t, dir, "dart", append([]string{"run", entry}, c.args...)...)
		if n := strings.Count(got, "--- PASS: TestClassify"); n != c.runs {
			t.Errorf("%v ran TestClassify %d times, and %d is the answer:\n%s", c.args, n, c.runs, got)
		}
		if strings.Contains(got, "TestShort") {
			t.Errorf("%v ran TestShort, and the filter must drop it:\n%s", c.args, got)
		}
	}
}

// TestTestsCoverWithNoRun proves that the line of the package holds the
// coverage in front of the mark of an empty run, exactly as go test writes it.
func TestTestsCoverWithNoRun(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/cover")
	if err != nil {
		t.Fatal(err)
	}
	want := goTest(t, dir, false, "-cover", "-count=0")
	got := gartTest(t, dir, driver.TestOptions{
		Cover: true, Count: 0, CountGiven: true,
	})
	if got != want {
		t.Errorf("the output differs\ngo test:\n%s\ngart test:\n%s", want, got)
	}
}

// TestTestsRejectsANegativeCount proves that the driver reports a clear cause
// for a count below 0.
func TestTestsRejectsANegativeCount(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/cover")
	if err != nil {
		t.Fatal(err)
	}
	_, err = driver.Test(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
	}, driver.TestOptions{Count: -1, CountGiven: true}, &strings.Builder{})
	if err == nil || !strings.Contains(err.Error(), "-count") {
		t.Errorf("the driver gave the error %v, and a cause of -count is the answer", err)
	}
}

// TestVersionMatchesTheOverlay proves that runtime.Version of a program gives
// the version of the tool chain that cmd/gart prints.
func TestVersionMatchesTheOverlay(t *testing.T) {
	want := "\"gart" + driver.Version + "\""
	for _, p := range overlay.Packages() {
		if p.Path != "runtime" {
			continue
		}
		for _, f := range p.Files {
			if strings.Contains(string(f.Content), want) {
				return
			}
		}
	}
	t.Errorf("no file of the overlay of runtime holds %s", want)
}

// itoa writes a small count. The test reads no other number.
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	out := ""
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	return out
}

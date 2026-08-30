package driver_test

import (
	"bytes"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

// TestTests compiles the test binaries of each module of testdata/tests, runs
// them with the Dart VM, and compares the output with the output of go test.
//
// The comparison cuts each elapsed time away, because a run does not repeat it.
func TestTests(t *testing.T) {
	root, err := filepath.Abs("../../testdata/tests")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		// The module unsupported holds a fuzz target and an example. go test
		// runs them, and gart reports them as a skipped test. Thus the output
		// differs on purpose, and a separate test reads it.
		if !entry.IsDir() || entry.Name() == "unsupported" {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(root, entry.Name())
			for _, verbose := range []bool{false, true} {
				name := "plain"
				if verbose {
					name = "verbose"
				}
				t.Run(name, func(t *testing.T) {
					want := goTest(t, dir, verbose)
					got := gartTest(t, dir, driver.TestOptions{Verbose: verbose})
					if got != want {
						t.Errorf("the output differs\ngo test:\n%s\ngart test:\n%s", want, got)
					}
				})
			}
		})
	}
}

// TestTestsFilter proves that the -run flag of gart keeps the same tests as the
// -run flag of go test. The module table holds a test with 3 subtests, thus a
// pattern with a slash, with a bar, or with a group reaches each rule of the
// matcher.
func TestTestsFilter(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/table")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	for _, pattern := range []string{
		"TestClamp",
		"TestClamp/inside",
		"TestClamp/inside|TestSum",
		"TestClamp/(inside|above)",
		"TestClamp/[/a]",
		"^TestSum$",
		"TestNothing",
	} {
		t.Run(pattern, func(t *testing.T) {
			want := goTest(t, dir, true, "-run", pattern)
			got := gartTestIn(t, dir, out, driver.TestOptions{Verbose: true, Run: pattern})
			if got != want {
				t.Errorf("the output differs\ngo test:\n%s\ngart test:\n%s", want, got)
			}
		})
	}
}

// TestTestsFiltersTheSkipLine proves that the -run flag also reads a regular
// expression for the skip line of an unsupported function. The driver writes
// that line, and not the harness.
func TestTestsFiltersTheSkipLine(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/unsupported")
	if err != nil {
		t.Fatal(err)
	}
	got := gartTest(t, dir, driver.TestOptions{Run: "^FuzzLength$"})
	if !strings.Contains(got, "--- SKIP: FuzzLength") {
		t.Errorf("the output holds no skip line of FuzzLength:\n%s", got)
	}
	if strings.Contains(got, "ExampleLength") {
		t.Errorf("the filter kept ExampleLength:\n%s", got)
	}
}

// TestTestsReportsAnUnsupportedFunction proves that a fuzz target and an
// example reach the output as a skipped test. A benchmark stays out, because
// go test with no -bench flag also runs no benchmark.
func TestTestsReportsAnUnsupportedFunction(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/unsupported")
	if err != nil {
		t.Fatal(err)
	}
	got := gartTest(t, dir, driver.TestOptions{})
	for _, want := range []string{
		"--- SKIP: FuzzLength\n    gart runs no fuzz target",
		"--- SKIP: ExampleLength\n    gart runs no example",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the output holds no %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "BenchmarkLength") {
		t.Errorf("the output names a benchmark:\n%s", got)
	}
}

// TestTestsRunsTestMain proves that a TestMain function replaces the entry
// point of a test binary. Such a function runs the setup, the tests, and the
// teardown.
func TestTestsRunsTestMain(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/tm\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "tm.go"), "package tm\n\n// Value gives a number.\nfunc Value() int { return 1 }\n")
	writeFile(t, filepath.Join(dir, "tm_test.go"), `package tm

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	fmt.Println("before")
	code := m.Run()
	fmt.Println("after", code)
	os.Exit(code)
}

func TestValue(t *testing.T) {
	if Value() != 1 {
		t.Error("Value gives the wrong number")
	}
}
`)
	out := &bytes.Buffer{}
	code, err := driver.Test(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
	}, driver.TestOptions{Verbose: true}, out)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Errorf("got the exit code %d, want 0", code)
	}
	got := out.String()
	for _, want := range []string{"before", "=== RUN   TestValue", "PASS", "after 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("the output holds no %q:\n%s", want, got)
		}
	}
	if k, j := strings.Index(got, "before"), strings.Index(got, "after 0"); k > j {
		t.Errorf("the setup ran after the teardown:\n%s", got)
	}
}

// TestTestsSkipsAnAllocationTest proves that AllocsPerRun reports a skip with
// the cause. The Dart VM gives no count of allocations for 1 operation, and a
// result of 0 would make such a test pass.
func TestTestsSkipsAnAllocationTest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/alloc\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "alloc.go"),
		"package alloc\n\n// Grow gives a new slice.\nfunc Grow() []byte { return make([]byte, 8) }\n")
	writeFile(t, filepath.Join(dir, "alloc_test.go"), `package alloc

import "testing"

func TestAllocs(t *testing.T) {
	if n := testing.AllocsPerRun(10, func() { Grow() }); n > 0 {
		t.Errorf("Grow makes %v allocations, and 0 is the count", n)
	}
}
`)
	got := gartTest(t, dir, driver.TestOptions{Verbose: true})
	for _, want := range []string{"--- SKIP: TestAllocs", "gart counts no allocation"} {
		if !strings.Contains(got, want) {
			t.Errorf("the output holds no %q:\n%s", want, got)
		}
	}
}

// TestTestsOfGoroot proves that the test command runs the real test files of a
// package of GOROOT. Such a package belongs to no module, and the driver gives
// it the module std.
//
// The test reads the Go installation of the machine. A new Go release can add a
// test file that falls outside the subset, and the assertion then fails.
func TestTestsOfGoroot(t *testing.T) {
	cases := []struct {
		pkg string
		// run holds the filter of -run. An empty filter runs each test.
		run   string
		wants []string
	}{
		{"path", "", []string{"--- PASS: TestClean ", "--- SKIP: TestCleanMallocs "}},
		{"container/list", "", []string{"--- PASS: TestList ", "--- PASS: TestZeroList "}},
		{"container/heap", "", []string{"--- PASS: TestInit0 ", "--- PASS: TestRemove2 "}},
		{"container/ring", "", []string{"--- PASS: TestNew ", "--- PASS: TestLink1 "}},
		{"unicode", "", []string{"--- PASS: TestLetterOptimizations ", "--- PASS: TestSpecialCaseNoMapping "}},
		{"encoding/hex", "", []string{"--- PASS: TestEncode ", "--- PASS: TestEncoderDecoder "}},
		{"encoding/ascii85", "", []string{"--- PASS: TestEncode ", "--- PASS: TestEncoderBuffering "}},
		{"encoding/csv", "", []string{"--- PASS: TestRead ", "--- PASS: TestWrite "}},
		{"hash/crc32", "", []string{"--- PASS: TestGolden ", "--- PASS: TestSimple "}},
		{"hash/fnv", "", []string{"--- PASS: TestGolden32 ", "--- PASS: TestGolden64 "}},
		{"hash/adler32", "", []string{"--- PASS: TestGolden ", "--- PASS: TestGoldenMarshal "}},
		{"maps", "", []string{"--- PASS: TestEqualFunc ", "--- PASS: TestCloneLarge "}},
		{"io", "", []string{
			"--- PASS: TestCopy ", "--- PASS: TestMultiReaderFreesExhaustedReaders "}},
		{"internal/chacha8rand", "", []string{
			"--- PASS: TestOutput ", "--- PASS: TestMarshal "}},
		{"unicode/utf8", "", []string{
			"--- PASS: TestFullRune ", "--- PASS: TestDecodeInvalidSequence "}},
		// The tests of zlib read testdata through a relative path, thus the
		// binary must run in the directory of the package.
		{"compress/zlib", "", []string{
			"--- PASS: TestWriterBig ", "--- PASS: TestWriterDict "}},
		// TestParseInvalidRegexps reads a limit of a resource, and the patch of
		// parse.go holds no recover for it. Read docs/parity.md.
		{"regexp/syntax", "Test[^P]|TestParse[SFLMN]", []string{
			"--- PASS: TestParseSimple ", "--- PASS: TestSimplify "}},
		{"math/rand/v2", "", []string{
			"--- PASS: TestRegress ", "--- PASS: TestChaCha8Read "}},
		{"compress/gzip", "", []string{
			"--- PASS: TestRoundTrip ", "--- PASS: TestConcat "}},
		// The whole run of the flate tests takes 8 minutes, thus the row
		// keeps a fast set. TestDeflateInflateString proves the size limit
		// of the encoder at each level.
		{"compress/flate", "TestDeflate$|TestDeflateInflate$|TestDeflateInflateString$|TestReaderDict$|TestWriterDict$|TestBulkHash4$",
			[]string{"--- PASS: TestDeflateInflateString ", "--- PASS: TestWriterDict "}},
	}
	for _, c := range cases {
		t.Run(c.pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(build.Default.GOROOT, "src", filepath.FromSlash(c.pkg))
			if _, err := os.Stat(dir); err != nil {
				t.Skipf("the machine holds no source of the package %s: %v", c.pkg, err)
			}
			out := &bytes.Buffer{}
			code, err := driver.Test(driver.Options{
				Dir:      dir,
				Patterns: []string{"."},
				Out:      t.TempDir(),
				Runtime:  runtimeDir(t),
				Format:   true,
			}, driver.TestOptions{Verbose: true, Run: c.run}, out)
			if err != nil {
				t.Fatalf("the test build failed:\n%v", err)
			}
			got := out.String()
			if code != 0 {
				t.Errorf("got the exit code %d, want 0:\n%s", code, got)
			}
			for _, want := range c.wants {
				if !strings.Contains(got, want) {
					t.Errorf("the output holds no %q:\n%s", want, got)
				}
			}
		})
	}
}

// TestTestsOfANestedPackage proves that the entry point of a test binary lands
// in the directory of the import path of the package under test. The name of
// the entry point holds only the base of that path.
func TestTestsOfANestedPackage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nested\n\ngo 1.27\n")
	pkg := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(pkg, 0o777); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pkg, "b.go"),
		"package b\n\n// Value gives a number.\nfunc Value() int { return 2 }\n")
	writeFile(t, filepath.Join(pkg, "b_test.go"), `package b

import "testing"

func TestValue(t *testing.T) {
	if Value() != 2 {
		t.Error("Value gives the wrong number")
	}
}
`)
	got := gartTest(t, dir, driver.TestOptions{Verbose: true})
	if !strings.Contains(got, "--- PASS: TestValue") {
		t.Errorf("the output holds no result of TestValue:\n%s", got)
	}
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o666); err != nil {
		t.Fatal(err)
	}
}

func runtimeDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func goTest(t *testing.T, dir string, verbose bool, extra ...string) string {
	t.Helper()
	args := []string{"test", "-count=1"}
	if verbose {
		args = append(args, "-v")
	}
	args = append(args, extra...)
	cmd := exec.Command("go", append(args, "./...")...)
	cmd.Dir = dir
	text, _ := cmd.CombinedOutput()
	return normalize(string(text))
}

func gartTest(t *testing.T, dir string, options driver.TestOptions) string {
	t.Helper()
	return gartTestIn(t, dir, t.TempDir(), options)
}

// gartTestIn runs the tests of a module and writes the Dart package into out. A
// second call with the same directory reads the cache, thus a run of many
// filters compiles 1 time.
func gartTestIn(t *testing.T, dir, out string, options driver.TestOptions) string {
	t.Helper()
	text := &bytes.Buffer{}
	if _, err := driver.Test(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtimeDir(t),
		Format:   true,
	}, options, text); err != nil {
		t.Fatalf("the test build failed:\n%v", err)
	}
	return normalize(text.String())
}

var (
	elapsed  = regexp.MustCompile(`\(\d+\.\d\ds\)`)
	total    = regexp.MustCompile(`\t\d+\.\d+s`)
	position = regexp.MustCompile(`^(\s+)([\w./-]+\.go):\d+: `)
)

// normalize cuts each part of the output that a run does not repeat. The source
// position of a log line stays, because the compiler writes it.
func normalize(text string) string {
	text = elapsed.ReplaceAllString(text, "(0.00s)")
	text = total.ReplaceAllString(text, "\t0.000s")
	return cutRuntimePosition(text)
}

// cutRuntimePosition drops a source position that names a file outside the test
// files. A deferred log call after Fatal runs on the way out of the goroutine,
// and Go writes the position of its own runtime for such a line.
func cutRuntimePosition(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		m := position.FindStringSubmatch(line)
		if m == nil || strings.HasSuffix(m[2], "_test.go") {
			continue
		}
		lines[i] = position.ReplaceAllString(line, "$1")
	}
	return strings.Join(lines, "\n")
}

// TestRun proves that the run command compiles package main and runs it with
// the Dart VM. The program reads its arguments through the interop package.
func TestRun(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/programs/args")
	if err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	code, err := driver.Run(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
		Format:   true,
	}, []string{"one", "two"}, out)
	if err != nil {
		t.Fatalf("the run failed:\n%v", err)
	}
	if code != 0 {
		t.Errorf("got the exit code %d, want 0", code)
	}
	if got, want := out.String(), "count 2\ntext one,two\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRunReportsTheExitCode proves that a panic gives a status that is not 0.
func TestRunReportsTheExitCode(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/programs/panics")
	if err != nil {
		t.Fatal(err)
	}
	code, err := driver.Run(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
		Format:   true,
	}, nil, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("the run failed:\n%v", err)
	}
	if code == 0 {
		t.Error("got the exit code 0, want a code of a panic")
	}
}

// TestBench proves that the -bench flag runs the benchmarks and writes a line
// in the format of go test. benchstat reads that format.
func TestBench(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/basic")
	if err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	code, err := driver.Test(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
		Format:   true,
	}, driver.TestOptions{Benchmarks: true, BenchCount: 50}, out)
	if err != nil {
		t.Fatalf("the benchmark run failed:\n%v", err)
	}
	if code != 0 {
		t.Errorf("got the exit code %d, want 0", code)
	}
	got := out.String()
	for _, want := range []string{
		"goos: dart",
		"pkg: example.com/basic/calc",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the output holds no %q:\n%s", want, got)
		}
	}
	for _, name := range []string{
		"BenchmarkAdd", "BenchmarkDouble/one", "BenchmarkDouble/big", "BenchmarkLoop",
	} {
		if !benchLine(got, name) {
			t.Errorf("the output holds no result of %s:\n%s", name, got)
		}
	}
	// A benchmark that holds a subbenchmark reports no line of its own, exactly
	// as go test does.
	if benchLine(got, "BenchmarkDouble") {
		t.Errorf("the parent of a subbenchmark reported a line:\n%s", got)
	}
}

// TestBenchFilter proves that the -bench flag keeps only a benchmark whose name
// holds the text.
func TestBenchFilter(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/basic")
	if err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	if _, err := driver.Test(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
		Format:   true,
	}, driver.TestOptions{Benchmarks: true, Bench: "BenchmarkAdd", BenchCount: 10}, out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !benchLine(got, "BenchmarkAdd") {
		t.Errorf("the output holds no result of BenchmarkAdd:\n%s", got)
	}
	if strings.Contains(got, "BenchmarkDouble") {
		t.Errorf("the filter kept BenchmarkDouble:\n%s", got)
	}
}

// benchLine reports whether the output holds the result line of a benchmark.
// The line is <name>-<isolates>\t<count>\t<value> ns/op.
func benchLine(text, name string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, name+"-1\t") && strings.HasSuffix(line, " ns/op") {
			return true
		}
	}
	return false
}

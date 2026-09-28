package driver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
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
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(root, entry.Name())
			out := t.TempDir()
			for _, verbose := range []bool{false, true} {
				name := "plain"
				if verbose {
					name = "verbose"
				}
				t.Run(name, func(t *testing.T) {
					want := goTest(t, dir, verbose)
					got := gartTestIn(t, dir, out, driver.TestOptions{Verbose: verbose})
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

// TestTestsFiltersASeed proves that the -run flag reads the name of a seed of a
// fuzz target. A seed of f.Add takes the name seed#<n>, and a corpus file takes
// the name of the file.
func TestTestsFiltersASeed(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/fuzzseed")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	for _, pattern := range []string{
		"FuzzBytes",
		"FuzzBytes/seed#1",
		"FuzzText/plain|FuzzBytes/long",
		"^FuzzReport$/^high$",
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
		Trace:    traceBuild(t),
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

func TestTestsCustomFlags(t *testing.T) {
	for _, mode := range []string{"default", "main", "parsed"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/flags\n\ngo 1.27\n")
			source := `package flags
import (
 "flag"
 "testing"
)
var enabled = flag.Bool("enabled", false, "enable the fixture")
var label = flag.String("label", "", "fixture label")
func TestFlags(t *testing.T) {
 if !flag.Parsed() || !*enabled || *label != "two words" {
  t.Fatalf("parsed=%v enabled=%v label=%q", flag.Parsed(), *enabled, *label)
 }
}
`
			if mode != "default" {
				parse := ""
				if mode == "parsed" {
					parse = "flag.Parse()"
				}
				source += "func TestMain(m *testing.M) { " + parse + "; m.Run() }\n"
			}
			writeFile(t, filepath.Join(dir, "flags_test.go"), source)
			args := []string{"-test.run=^TestFlags$", "-enabled", "-label", "two words"}
			nativeArgs := append([]string{"test", "-count=1", "-args"}, args...)
			run(t, dir, "go", nativeArgs...)
			out := t.TempDir()
			got := gartTestIn(t, dir, out, driver.TestOptions{Args: args})
			if !strings.HasPrefix(got, "ok  ") {
				t.Fatalf("unexpected test output: %q", got)
			}
		})
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
	type stdCase struct {
		Package string `json:"package"`
		Run     string `json:"run"`
	}
	data, err := os.ReadFile("../parity/testdata/vm-suites.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []stdCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}

	type namedCase struct {
		name string
		stdCase
	}
	var runs []namedCase
	for _, c := range cases {
		if c.Package == "debug/buildinfo" && !testing.Short() {
			listing := run(t, ".", "go", "test", "-list", ".", c.Package)
			var others []string
			for _, name := range strings.Fields(listing) {
				if name != "TestReadFile" && (strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Example") || strings.HasPrefix(name, "Fuzz")) {
					others = append(others, regexp.QuoteMeta(name))
				}
			}
			runs = append(runs, namedCase{c.Package, stdCase{Package: c.Package, Run: "^(" + strings.Join(others, "|") + ")$"}})
			runs = append(runs, namedCase{c.Package + "-native-exe", stdCase{Package: c.Package, Run: "^TestReadFile$/^" + build.Default.GOOS + "_" + build.Default.GOARCH + "$/^(pie|exe)$"}})
			continue
		}
		if c.Package == "crypto/mldsa" && !testing.Short() {
			listing := run(t, ".", "go", "test", "-list", ".", c.Package)
			var others []string
			for _, name := range strings.Fields(listing) {
				if name != "TestAccumulated" && (strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Example") || strings.HasPrefix(name, "Fuzz")) {
					others = append(others, regexp.QuoteMeta(name))
				}
			}
			runs = append(runs, namedCase{c.Package, stdCase{Package: c.Package, Run: "^(" + strings.Join(others, "|") + ")$"}})
			for _, parameter := range []int{44, 65, 87} {
				runs = append(runs, namedCase{fmt.Sprintf("crypto/mldsa/accumulated-%d", parameter), stdCase{Package: c.Package, Run: fmt.Sprintf("^TestAccumulated$/^ML-DSA-%d$", parameter)}})
			}
			continue
		}
		if c.Package == "archive/zip" && !testing.Short() {
			large := []string{"TestOver65kFiles", "TestZip64", "TestZip64EdgeCase", "TestZip64DirectoryOffset", "TestZip64ManyRecords"}
			listing := run(t, ".", "go", "test", "-list", ".", c.Package)
			var others []string
			for _, name := range strings.Fields(listing) {
				if name != "TestZip64WriterCDGoldens" && name != "TestZip64LargeDirectory" && !slices.Contains(large, name) && (strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Example") || strings.HasPrefix(name, "Fuzz")) {
					others = append(others, regexp.QuoteMeta(name))
				}
			}
			runs = append(runs, namedCase{c.Package, stdCase{Package: c.Package, Run: "^(" + strings.Join(others, "|") + ")$"}})
			for _, name := range large {
				runs = append(runs, namedCase{c.Package + "-" + name, stdCase{Package: c.Package, Run: "^" + name + "$"}})
			}
			for i, name := range []string{"uint32max-1_NoZip64", "uint32max_HasZip64"} {
				runs = append(runs, namedCase{fmt.Sprintf("archive/zip-large-directory-%d", i), stdCase{Package: c.Package, Run: "^TestZip64LargeDirectory$/^" + name + "$"}})
			}
			fixtures, err := filepath.Glob(filepath.Join(build.Default.GOROOT, "src/archive/zip/testdata/zip64/*.zsparse"))
			if err != nil || len(fixtures) == 0 {
				t.Fatalf("upstream ZIP64 fixtures unavailable: %v", err)
			}
			groups := make([][]string, 3)
			for i, fixture := range fixtures {
				name := strings.TrimSuffix(filepath.Base(fixture), ".zsparse")
				groups[i%len(groups)] = append(groups[i%len(groups)], regexp.QuoteMeta(name))
			}
			for i, names := range groups {
				runs = append(runs, namedCase{fmt.Sprintf("archive/zip-goldens-%d", i), stdCase{Package: c.Package, Run: "^TestZip64WriterCDGoldens$/^(" + strings.Join(names, "|") + ")$"}})
			}
			continue
		}
		if c.Package != "compress/flate" || testing.Short() {
			runs = append(runs, namedCase{c.Package, c})
			continue
		}
		// Preserve all upstream cases. Only the independent reset subtests split.
		listing := run(t, ".", "go", "test", "-list", ".", "compress/flate")
		var others []string
		for _, name := range strings.Fields(listing) {
			if name != "TestWriter_Reset" && (strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Example") || strings.HasPrefix(name, "Fuzz")) {
				others = append(others, regexp.QuoteMeta(name))
			}
		}
		c.Run = "^(" + strings.Join(others, "|") + ")$"
		runs = append(runs, namedCase{c.Package, c})
		for i, levels := range []string{"[0157]", "[268]", "[349]"} {
			runs = append(runs, namedCase{fmt.Sprintf("compress/flate-reset-%d", i), stdCase{
				Package: c.Package, Run: "^TestWriter_Reset$/^level=" + levels + "$",
			}})
		}
	}
	var names []string
	for _, c := range runs {
		names = append(names, c.name)
	}
	selected := testShard(t, names)
	for _, c := range runs {
		if !selected[c.name] {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(build.Default.GOROOT, "src", filepath.FromSlash(c.Package))
			if _, err := os.Stat(dir); err != nil {
				t.Fatalf("the machine holds no source of the package %s: %v", c.Package, err)
			}
			outDir := t.TempDir()
			var inputs *certificateInputs
			options := driver.Options{Trace: traceBuild(t), Dir: dir, Patterns: []string{"."}, Out: outDir, Runtime: runtimeDir(t), Tests: true, TrimPath: true}
			if c.Package == "crypto/x509" {
				inputs = prepareX509SuiteInputs(t, dir)
				options.TestSources = inputs.TestSources
			}
			importer := prepareImporterInputs(t, c.Package)
			if importer != nil {
				options.TestSources = importer.TestSources
			}
			stats, err := driver.BuildWithStats(options)
			if err != nil {
				t.Logf("upstream test build failed: %v", err)
				compareUpstreamSDKInputs(t, c.name, c.Package, c.Run, "vm", dir, "the test build failed: "+err.Error(), 1, inputs, nil, importer, true, nil)
				return
			}
			if len(stats.Binaries) != 1 {
				t.Fatalf("got %d upstream test binaries", len(stats.Binaries))
			}
			prepareDart(t, outDir)
			if c.Package == "internal/godebug" || c.Package == "crypto/x509" && build.Default.GOOS == "linux" && !testing.Short() || c.Package == "net/http/cgi" || c.Package == "net/http/httptest" || (c.Package == "log" || c.Package == "log/slog") && !testing.Short() {
				prepareUpstreamSelfExec(t, outDir, stats.Binaries[0].Entry)
			}
			var images []nativeImage
			if c.Package == "debug/elf" || c.Package == "debug/gosym" {
				images = append(images, prepareUpstreamNativeImage(t, dir, outDir, stats.Binaries[0].Entry))
			}
			var env []string
			if c.Package == "go/build" || c.Package == "debug/buildinfo" || c.Package == "crypto/x509" || c.Package == "internal/testenv" || c.Package == "internal/godebug" || c.Package == "internal/godebugs" || importer != nil || c.Package == "go/importer" || c.Package == "go/internal/gccgoimporter" || c.Package == "internal/types/errors" {
				env = append(os.Environ(), "GOROOT="+build.Default.GOROOT)
			}
			args := append([]string{filepath.Join(outDir, stats.Binaries[0].Entry)}, upstreamTestArgs("-test.run="+c.Run, "-test.skip="+upstreamSkip(t, c.Package))...)
			args = append(args, upstreamExtraArgs(c.Package)...)
			text, execErr, finished := output(dir, env, upstreamLimit(), "dart", args...)
			if !finished {
				compareUpstreamSDKInputs(t, c.name, c.Package, c.Run, "vm", dir, string(text), 124, inputs, nil, importer, true, nil, images...)
				return
			}
			code := 0
			if execErr != nil {
				if e, ok := execErr.(*exec.ExitError); ok {
					code = e.ExitCode()
				} else {
					t.Fatal(execErr)
				}
			}
			got := string(text)
			t.Logf("upstream results:\n%s", got)
			compareUpstreamSDKInputs(t, c.name, c.Package, c.Run, "vm", dir, got, code, inputs, nil, importer, true, nil, images...)

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
	text, err, ok := output(dir, nil, runLimit, "go", append(args, "./...")...)
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	assertTestExit(t, code, string(text))
	if !ok {
		t.Fatalf("go %v did not end inside %v in %s", args, runLimit, dir)
	}
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
	code, err := driver.Test(driver.Options{
		Trace:    traceBuild(t),
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtimeDir(t),
		Format:   false,
	}, options, text)
	if err != nil {
		t.Fatalf("the test build failed:\n%v", err)
	}
	assertTestExit(t, code, text.String())
	return normalize(text.String())
}

func assertTestExit(t *testing.T, code int, text string) {
	t.Helper()
	want := 0
	if strings.HasSuffix(text, "FAIL\n") {
		want = 1
	}
	if code != want {
		t.Fatalf("test process exit %d disagrees with output, want %d:\n%s", code, want, text)
	}
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
		Trace:    traceBuild(t),
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
		Format:   false,
	}, []string{"one", "two"}, out, out)
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
		Trace:    traceBuild(t),
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
		Format:   false,
	}, nil, &bytes.Buffer{}, &bytes.Buffer{})
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
		Trace:    traceBuild(t),
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
		Format:   false,
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
		Trace:    traceBuild(t),
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
		Format:   false,
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

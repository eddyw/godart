package driver

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/eddyw/godart/internal/load"
	harness "github.com/eddyw/godart/internal/overlay/std/testing"
)

// mainEntry is the Dart entry point of package main.
const mainEntry = "bin/main.dart"

// Version is the version of the gart tool chain. cmd/gart prints it, and the
// overlay of the package runtime gives the same number to a program through
// runtime.Version. TestVersionMatchesTheOverlay compares the 2.
const Version = "0.1.0-dev"

// TestOptions holds the flags of the test command.
type TestOptions struct {
	// Verbose writes 1 line for each test.
	Verbose bool
	// Run keeps only a test whose name matches this regular expression.
	Run string
	// Benchmarks runs the benchmarks. Bench then keeps only a benchmark whose
	// name matches its regular expression.
	Benchmarks bool
	Bench      string
	// BenchTime is the length that 1 benchmark targets. It is 1 second when the
	// caller gives no value.
	BenchTime time.Duration
	// BenchCount fixes the count of iterations. It replaces the search when it
	// is above 0.
	BenchCount int
	// Short asks each test for its short form. testing.Short then gives true.
	Short bool
	// Count is the count of runs of the whole list of the tests. CountGiven
	// must hold true for the value to reach the run: go test runs no test for
	// a count of 0, and 0 is also the zero value of this field.
	Count      int
	CountGiven bool
	// Cover writes a counter into each statement of each package under test,
	// and the run then reports the part of the statements that it reached.
	Cover bool
}

// Run compiles the module and runs package main with the Dart VM. The result
// is the exit code of the program.
func Run(opt Options, args []string, out io.Writer) (int, error) {
	if _, err := BuildWithStats(opt); err != nil {
		return 1, err
	}
	if _, err := os.Stat(filepath.Join(opt.Out, mainEntry)); err != nil {
		return 1, fmt.Errorf("the module holds no package main")
	}
	if err := pubGet(opt.Out, opt.Runtime); err != nil {
		return 1, err
	}
	return dartRun(opt.Out, append([]string{mainEntry}, args...), out)
}

// Test compiles the test binaries and runs each one. The result is the exit
// code of the command.
//
// A test binary runs on the Dart VM. A benchmark run compiles the binary ahead
// of time first, because the VM starts a just in time compiler and its warm up
// cost hides the result.
func Test(opt Options, testOpt TestOptions, out io.Writer) (int, error) {
	if testOpt.CountGiven && testOpt.Count < 0 {
		return 1, fmt.Errorf("the flag -count takes a count of 0 or more")
	}
	opt.Tests = true
	opt.Cover = testOpt.Cover
	stats, err := BuildWithStats(opt)
	if err != nil {
		return 1, err
	}
	if err := pubGet(opt.Out, opt.Runtime); err != nil {
		return 1, err
	}
	for _, path := range stats.NoTests {
		fmt.Fprintf(out, "?   \t%s\t[no test files]\n", path)
	}
	failed := false
	for _, bin := range stats.Binaries {
		for _, name := range bin.Skips {
			if !matches(name, testOpt.Run) {
				continue
			}
			fmt.Fprintf(out, "--- SKIP: %s\n    %s\n", name, load.SkipCause(name))
		}
		if testOpt.Benchmarks {
			writeBenchHeader(out, bin.Package)
		}
		start := time.Now()
		text, code, err := runBinary(opt.Out, bin, testOpt)
		if err != nil {
			return 1, err
		}
		elapsed := seconds(time.Since(start))
		if code != 0 || testOpt.Verbose || testOpt.Benchmarks {
			fmt.Fprint(out, text)
		}
		if code != 0 {
			failed = true
			fmt.Fprintf(out, "FAIL\t%s\t%ss\n", bin.Package, elapsed)
			continue
		}
		// go test writes the coverage in front of the mark of an empty run.
		fmt.Fprintf(out, "ok  \t%s\t%ss%s%s\n", bin.Package, elapsed,
			coverSuffix(text), noRun(text))
	}
	if failed {
		fmt.Fprintln(out, "FAIL")
		return 1, nil
	}
	return 0, nil
}

// writeBenchHeader writes the configuration of a benchmark run. benchstat reads
// such a line, thus a comparison of 2 runs keeps the context.
func writeBenchHeader(out io.Writer, pkgPath string) {
	fmt.Fprintf(out, "goos: dart\n")
	fmt.Fprintf(out, "goarch: %s\n", runtime.GOARCH)
	fmt.Fprintf(out, "pkg: %s\n", pkgPath)
}

// runBinary runs 1 test binary and gives its output and its exit code.
//
// The binary runs in the directory of the package under test, exactly as the
// binary of go test does. Thus a test file reads a relative path such as
// testdata/x.txt. The entry point takes an absolute path for that reason.
func runBinary(out string, bin TestBinary, testOpt TestOptions) (string, int, error) {
	args := binaryArgs(testOpt)
	dir := bin.Dir
	if dir == "" {
		dir = out
	}
	if !testOpt.Benchmarks {
		entry, err := filepath.Abs(filepath.Join(out, bin.Entry))
		if err != nil {
			return "", 1, err
		}
		return dartOutput(dir, append([]string{"run", entry}, args...))
	}
	exe, err := compileExe(out, bin.Entry)
	if err != nil {
		return "", 1, err
	}
	return command(dir, exe, args)
}

// binaryArgs writes the flags that the harness of the binary reads.
func binaryArgs(testOpt TestOptions) []string {
	var args []string
	if testOpt.Verbose {
		args = append(args, "-test.v")
	}
	if testOpt.Run != "" {
		args = append(args, "-test.run="+testOpt.Run)
	}
	if testOpt.Short {
		args = append(args, "-test.short")
	}
	// A count of 0 runs no test, exactly as go test does. The value 1 is the
	// default of the harness, thus it needs no flag.
	if testOpt.CountGiven && testOpt.Count != 1 {
		args = append(args, "-test.count="+strconv.Itoa(testOpt.Count))
	}
	if testOpt.Benchmarks {
		args = append(args, "-test.bench="+testOpt.Bench)
		if testOpt.BenchCount > 0 {
			args = append(args, "-test.benchcount="+strconv.Itoa(testOpt.BenchCount))
		} else if testOpt.BenchTime > 0 {
			args = append(args, "-test.benchtime="+
				strconv.FormatInt(testOpt.BenchTime.Nanoseconds(), 10))
		}
	}
	return args
}

// compileExe builds an ahead of time binary of an entry point and gives its
// full path.
func compileExe(out, entry string) (string, error) {
	exe, err := filepath.Abs(filepath.Join(out, strings.TrimSuffix(entry, ".dart")+".exe"))
	if err != nil {
		return "", err
	}
	cmd := exec.Command("dart", "compile", "exe", entry, "-o", exe)
	cmd.Dir = out
	if text, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("dart compile exe %s: %v\n%s", entry, err, text)
	}
	return exe, nil
}

// noRun marks a package whose filter kept no test. go test reads the warning of
// the binary and writes the mark, thus the driver does the same.
func noRun(text string) string {
	if strings.Contains(text, harness.NoTestsToRun+"\n") {
		return " [no tests to run]"
	}
	return ""
}

// coverSuffix reads the coverage line of a binary and gives the part that go
// test writes after the time of the package.
func coverSuffix(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "coverage: ") {
			return "\t" + line
		}
	}
	return ""
}

// matches reads the filter of the -run flag and of the -bench flag. The harness
// of a test binary reads the same file.
func matches(name, pattern string) bool {
	return harness.MatchName(name, pattern, regexp.MatchString)
}

func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 64)
}

// pubGet resolves the dependencies of the output package. The Dart VM needs
// the package configuration, and the configuration holds the path of the
// runtime library.
func pubGet(out, runtime string) error {
	config := filepath.Join(out, ".dart_tool", "package_config.json")
	specs := []string{filepath.Join(out, "pubspec.yaml")}
	if runtime != "" {
		// The output package depends on the runtime by path, thus a new
		// dependency of the runtime must reach the configuration of the
		// output. The package file of the output does not change with it.
		specs = append(specs, filepath.Join(runtime, "pubspec.yaml"))
	}
	if fresh(config, specs) {
		return nil
	}
	cmd := exec.Command("dart", "pub", "get")
	cmd.Dir = out
	if text, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("dart pub get: %v\n%s", err, text)
	}
	return nil
}

// fresh reports whether the configuration is not older than each package file.
func fresh(config string, specs []string) bool {
	a, err := os.Stat(config)
	if err != nil {
		return false
	}
	for _, spec := range specs {
		b, err := os.Stat(spec)
		if err != nil {
			return false
		}
		if a.ModTime().Before(b.ModTime()) {
			return false
		}
	}
	return true
}

// dartRun runs the Dart VM and gives the exit code of the program.
func dartRun(dir string, args []string, out io.Writer) (int, error) {
	cmd := exec.Command("dart", append([]string{"run"}, args...)...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if code := exitCode(err); code >= 0 {
			return code, nil
		}
		return 1, err
	}
	return 0, nil
}

// dartOutput runs the Dart VM and holds the output. The test command writes
// the output only when the binary fails, exactly as go test does.
func dartOutput(dir string, args []string) (string, int, error) {
	return command(dir, "dart", args)
}

// command runs a program in a directory and holds its output.
func command(dir, name string, args []string) (string, int, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	text, err := cmd.CombinedOutput()
	if err != nil {
		if code := exitCode(err); code >= 0 {
			return string(text), code, nil
		}
		return string(text), 1, err
	}
	return string(text), 0, nil
}

// exitCode reads the status of a program that ended with a failure. The result
// is -1 when the command itself failed.
func exitCode(err error) int {
	if e, ok := err.(*exec.ExitError); ok {
		return e.ExitCode()
	}
	return -1
}

package driver

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eddyw/godart/internal/load"
)

// mainEntry is the Dart entry point of package main.
const mainEntry = "bin/main.dart"

// TestOptions holds the flags of the test command.
type TestOptions struct {
	// Verbose writes 1 line for each test.
	Verbose bool
	// Run keeps only a test whose name holds this text.
	Run string
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
	if err := pubGet(opt.Out); err != nil {
		return 1, err
	}
	return dartRun(opt.Out, append([]string{mainEntry}, args...), out)
}

// Test compiles the test binaries and runs each one with the Dart VM. The
// result is the exit code of the command.
func Test(opt Options, testOpt TestOptions, out io.Writer) (int, error) {
	opt.Tests = true
	stats, err := BuildWithStats(opt)
	if err != nil {
		return 1, err
	}
	if err := pubGet(opt.Out); err != nil {
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
		args := []string{bin.Entry}
		if testOpt.Verbose {
			args = append(args, "-test.v")
		}
		if testOpt.Run != "" {
			args = append(args, "-test.run="+testOpt.Run)
		}
		start := time.Now()
		text, code, err := dartOutput(opt.Out, args)
		if err != nil {
			return 1, err
		}
		elapsed := seconds(time.Since(start))
		if code != 0 || testOpt.Verbose {
			fmt.Fprint(out, text)
		}
		if code != 0 {
			failed = true
			fmt.Fprintf(out, "FAIL\t%s\t%ss\n", bin.Package, elapsed)
			continue
		}
		fmt.Fprintf(out, "ok  \t%s\t%ss\n", bin.Package, elapsed)
	}
	if failed {
		fmt.Fprintln(out, "FAIL")
		return 1, nil
	}
	return 0, nil
}

// matches reads the filter of the -run flag. The harness holds the same rule,
// because the subset has no regexp package.
func matches(name, pattern string) bool {
	return pattern == "" || strings.Contains(name, pattern)
}

func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 64)
}

// pubGet resolves the dependencies of the output package. The Dart VM needs
// the package configuration, and the configuration holds the path of the
// runtime library.
func pubGet(out string) error {
	config := filepath.Join(out, ".dart_tool", "package_config.json")
	spec := filepath.Join(out, "pubspec.yaml")
	if fresh(config, spec) {
		return nil
	}
	cmd := exec.Command("dart", "pub", "get")
	cmd.Dir = out
	if text, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("dart pub get: %v\n%s", err, text)
	}
	return nil
}

// fresh reports whether the configuration is not older than the package file.
func fresh(config, spec string) bool {
	a, err := os.Stat(config)
	if err != nil {
		return false
	}
	b, err := os.Stat(spec)
	if err != nil {
		return false
	}
	return !a.ModTime().Before(b.ModTime())
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
	cmd := exec.Command("dart", append([]string{"run"}, args...)...)
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

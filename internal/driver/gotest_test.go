package driver_test

import (
	"encoding/json"
	"go/build"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

// goTestSkips lists each file of the test directory of the Go repository that
// compiles and whose run differs from Go. Each entry names the cause. A fix
// removes its entry, and the file then joins the comparison.
var goTestSkips = map[string]string{
	// The Dart runtime exposes no equivalent machine operation.
	"fixedbugs/issue27201":  "runtime.Stack omits native instruction offsets, but this test requires a space after the source line before that offset",
	"peano":                 "the file is a torture test of the segmented stack of Go, and the Dart VM holds a fixed stack",
	"inline_literal":        "reflect.Value.Pointer gives the identity of the object, thus runtime.FuncForPC gives nil for each value and the loop of the file never ends",
	"stack":                 "the Dart VM holds a fixed stack, and the recursion overflows",
	"stackobj2":             "the Dart VM holds a fixed stack, and the recursion overflows",
	"fixedbugs/issue58300":  "reflect.Value.Pointer gives the identity of the object, and runtime.FuncForPC reads a program counter that the Dart VM gives not",
	"fixedbugs/issue58300b": "reflect.Value.Pointer gives the identity of the object, and runtime.FuncForPC reads a program counter that the Dart VM gives not",
	"fixedbugs/issue4667":   "the Dart VM gives no allocation count, and testing.AllocsPerRun reads one",
	"fixedbugs/issue4618":   "the Dart VM gives no allocation count, and testing.AllocsPerRun reads one",
	"fixedbugs/issue29735":  "runtime.FuncForPC walks the program counters, and the Dart VM gives none",
	"fixedbugs/issue29312":  "the composite literal nests 253 levels, and the Dart front end rejects a file with that many nested expressions",
	"fixedbugs/bug260":      "the test reads the machine address of a value, and Dart gives none",
	"fixedbugs/bug352":      "2 values with no size take 2 cells, and gc gives them 1 address",
	"fixedbugs/issue75764":  "the Dart VM holds a fixed stack, and the recursion overflows",
	"fixedbugs/issue29190":  "the test allocates a list of 4 billion elements, and the Dart VM runs out of memory",
	"fixedbugs/issue7550":   "the append of 2 slices of the maximum length must overflow the capacity, and the Dart build runs out of memory",

	// The compiler or runtime retains a known compatibility gap.
	"fixedbugs/issue5856":  "the panic path of Go holds a frame of gopanic, and the frame chain of the runtime holds none",
	"fixedbugs/issue19182": "a busy loop without a block operation never yields, and the other goroutine never runs",

	// The fixture requires a Go tree, a native architecture, or a different harness.
	"fixedbugs/issue10607": "the test exercises native Go linker flags and requires a companion source outside this single-file harness",
	"winbatch":             "the file reads a file of the tree of Go under runtime.GOROOT, and gart gives an empty text there",
	"nosplit":              "the file picks the frame size of the test from runtime.GOARCH, and gart reports dart64",
	"linkobj":              "the file writes the name of a temporary directory of the system, and the 2 runs take 2 names",
	"linkmain_run":         "the file writes the name of a temporary directory of the system, and the 2 runs take 2 names",
	"fixedbugs/issue21317": "the file writes the name of a temporary directory of the system, and the 2 runs take 2 names",
	"linkx_run":            "the file runs go run on a neighbor file that the run directory does not hold, and the go run wrapper of the comparison adds an exit line that dart run does not add",

	// The test exceeds the run budget.
	"fixedbugs/issue34395": "a global array of 100 MB fills element by element, and the run needs more than the cap",
}

// runWithin runs a program with a time limit. A file of the test directory
// can hold a loop that never ends under the block model, and the limit keeps
// the suite alive. WaitDelay closes the pipes when a child keeps them open.
//
// The 3 results are the output, whether the program failed, and whether the
// run ended inside the limit.
func runWithin(t *testing.T, dir string, limit time.Duration, name string, args ...string) (string, bool, bool) {
	t.Helper()
	start := time.Now()
	text, err, ok := output(dir, nil, limit, name, args...)
	t.Logf("command=%s %v duration=%.3fs", name, args, time.Since(start).Seconds())
	return cutTrace(string(text)), err != nil, ok
}

// cutExitStatus drops the line that `go run` writes when the program ends with
// a value that is not 0. `dart run` writes no such line, thus the comparison
// reads the exit status of the 2 runs on its own.
func cutExitStatus(text string) string {
	lines := strings.Split(text, "\n")
	n := len(lines)
	if n > 0 && strings.HasPrefix(lines[n-1], "exit status ") {
		return strings.TrimRight(strings.Join(lines[:n-1], "\n"), "\n")
	}
	return text
}

// TestRunFilesOfGoTest compiles each file of the test directory of the Go
// repository whose header is `// run`, runs the Dart output, and compares it
// with the output of go run. The directory proves the language, thus a file
// outside the subset skips.
func TestRunFilesOfGoTest(t *testing.T) {
	data, err := os.ReadFile("testdata/go-test-rejections.json")
	if err != nil {
		t.Fatal(err)
	}
	var rejections map[string][]string
	if err := json.Unmarshal(data, &rejections); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(build.Default.GOROOT, "test")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("the machine holds no test directory of Go: %v", err)
	}
	runtime := runtimeDir(t)
	type sourceCase struct {
		name, file, dir string
		source          []byte
	}
	var cases []sourceCase
	var names []string
	for _, dir := range []string{"", "typeparam", "fixedbugs"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Errorf("the directory %q of the test directory is gone: %v", dir, err)
			continue
		}
		for _, entry := range entries {
			base, ok := strings.CutSuffix(entry.Name(), ".go")
			if !ok {
				continue
			}
			name := path.Join(dir, base)
			source, err := os.ReadFile(filepath.Join(root, dir, entry.Name()))
			if err != nil || !strings.HasPrefix(string(source), "// run\n") {
				continue
			}
			cases = append(cases, sourceCase{name, entry.Name(), dir, source})
			names = append(names, name)
		}
	}
	selected := testShard(t, names)
	for _, c := range cases {
		if !selected[c.name] {
			continue
		}
		name, source := c.name, c.source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := build.Default
			ctx.BuildTags = append(slices.Clone(ctx.BuildTags), "dart")
			eligible, err := ctx.MatchFile(filepath.Join(root, c.dir), c.file)
			if err != nil {
				t.Fatal(err)
			}
			if !eligible {
				t.Skipf("upstream build constraints exclude %s/%s", ctx.GOOS, ctx.GOARCH)
			}
			if cause, ok := goTestSkips[name]; ok {
				t.Skip(cause)
			}
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/gotest\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, c.file), string(source))
			out := filepath.Join(dir, "out")
			err = driver.Build(driver.Options{
				Trace:    traceBuild(t),
				Dir:      dir,
				Patterns: []string{"./..."},
				Out:      out,
				Runtime:  runtime,
				Format:   false,
			})
			if want, known := rejections[name]; known {
				if err == nil {
					t.Fatal("the build now passes: remove its expected rejection")
				}
				got := strings.Split(strings.ReplaceAll(err.Error(), dir, "$DIR"), "\n")
				slices.Sort(got)
				if !slices.Equal(got, want) {
					t.Fatalf("the rejection changed\nwant:\n%s\ngot:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
				}
				t.Skipf("known rejection:\n%v", err)
			}
			if err != nil {
				t.Fatalf("unexpected build failure:\n%v", err)
			}
			want, wantFail, ran := runWithin(t, dir, runLimit, "go", "run", c.file)
			if !ran {
				t.Errorf("go run did not end within %v", runLimit)
				return
			}
			want = cutExitStatus(want)
			prepareDart(t, out)
			limit := 2 * time.Minute
			if name == "fixedbugs/issue13169" || name == "fixedbugs/issue78081" {
				// These upstream stress cases perform 1 million channel transfers
				// or 104 million recursive calls.
				limit = 5 * time.Minute
			}
			got, gotFail, ended := runWithin(t, out, limit, "dart", "--packages=.dart_tool/package_config.json", "bin/main.dart")
			if !ended {
				t.Errorf("the run did not end within %v", limit)
				return
			}
			if got != want {
				t.Errorf("the output differs\nGo:\n%s\nDart:\n%s", want, got)
			}
			if gotFail != wantFail {
				t.Errorf("the exit status differs: go failed %v, and dart failed %v",
					wantFail, gotFail)
			}
		})
	}
}

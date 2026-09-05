package driver_test

import (
	"go/build"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

// goTestSkips lists each file of the test directory of the Go repository that
// compiles and whose run differs from Go. Each entry names the cause. A fix
// removes its entry, and the file then joins the comparison.
var goTestSkips = map[string]string{
	"peano":          "the file is a torture test of the segmented stack of Go, and the Dart VM holds a fixed stack",
	"inline_literal": "reflect.Value.Pointer gives the identity of the object, thus runtime.FuncForPC gives nil for each value and the loop of the file never ends",
	"stack":          "the Dart VM holds a fixed stack, and the recursion overflows",
	"stackobj2":      "the Dart VM holds a fixed stack, and the recursion overflows",

	"fixedbugs/issue58300":  "reflect.Value.Pointer gives the identity of the object, and runtime.FuncForPC reads a program counter that the Dart VM gives not",
	"fixedbugs/issue58300b": "reflect.Value.Pointer gives the identity of the object, and runtime.FuncForPC reads a program counter that the Dart VM gives not",
	"fixedbugs/issue25897a": "the body of the reflect.MakeFunc value sends on a full channel, and a function that gives up control is outside MakeFunc, exactly as it is outside Value.Call",
	"fixedbugs/issue4667":   "the Dart VM gives no allocation count, and testing.AllocsPerRun reads one",
	"fixedbugs/issue4618":   "the Dart VM gives no allocation count, and testing.AllocsPerRun reads one",
	"fixedbugs/issue29735":  "runtime.FuncForPC walks the program counters, and the Dart VM gives none",
	"fixedbugs/issue5856":   "the panic path of Go holds a frame of gopanic, and the frame chain of the runtime holds none",
	"fixedbugs/issue29312":  "the composite literal nests 253 levels, and the Dart front end rejects a file with that many nested expressions",
	"fixedbugs/issue19182":  "a busy loop without a block operation never yields, and the other goroutine never runs",
	"typeparam/nested":      "a local type of a generic function now holds the type argument of that function in its own name, and the name of such a type inside a type argument of another instance still holds none, thus 2 instances that differ only there compare equal",
	"winbatch":              "the file reads a file of the tree of Go under runtime.GOROOT, and gart gives an empty text there",
	"nosplit":               "the file picks the frame size of the test from runtime.GOARCH, and gart reports dart64",
	"linkobj":               "the file writes the name of a temporary directory of the system, and the 2 runs take 2 names",
	"linkmain_run":          "the file writes the name of a temporary directory of the system, and the 2 runs take 2 names",
	"fixedbugs/issue21317":  "the file writes the name of a temporary directory of the system, and the 2 runs take 2 names",
	"linkx_run":             "the file runs go run on a neighbor file that the run directory does not hold, and the go run wrapper of the comparison adds an exit line that dart run does not add",

	"fixedbugs/bug260":     "the test reads the machine address of a value, and Dart gives none",
	"fixedbugs/bug352":     "2 values with no size take 2 cells, and gc gives them 1 address",
	"fixedbugs/issue75764": "the Dart VM holds a fixed stack, and the recursion overflows",
	"fixedbugs/issue34395": "a global array of 100 MB fills element by element, and the run needs more than the cap",

	"devirtualization_nil_panics": "the test reads the line of the panic site in the recovered stack, and the source map gives none",
	"fixedbugs/issue29190":        "the test allocates a list of 4 billion elements, and the Dart VM runs out of memory",
	"fixedbugs/issue7550":         "the append of 2 slices of the maximum length must overflow the capacity, and the Dart build runs out of memory",
	"fixedbugs/issue29504":        "the test reads the frames of the recovered stack, and the source map gives none of them",
	"fixedbugs/issue22083":        "the test reads the line of the panic site in the stack of a recover, and the frame chain of the runtime holds the current state only",
	"fixedbugs/issue33724":        "the test reads the name of the function of the panic site in the stack of a recover, and the frame chain of the runtime holds the current state only",
}

// runWithin runs a program with a time limit. A file of the test directory
// can hold a loop that never ends under the block model, and the limit keeps
// the suite alive. WaitDelay closes the pipes when a child keeps them open.
//
// The 3 results are the output, whether the program failed, and whether the
// run ended inside the limit.
func runWithin(t *testing.T, dir string, limit time.Duration, name string, args ...string) (string, bool, bool) {
	t.Helper()
	text, err, ok := output(dir, nil, limit, name, args...)
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
	root := filepath.Join(build.Default.GOROOT, "test")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("the machine holds no test directory of Go: %v", err)
	}
	runtime := runtimeDir(t)
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
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				if cause, ok := goTestSkips[name]; ok {
					t.Skip(cause)
				}
				dir := t.TempDir()
				writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/gotest\n\ngo 1.27\n")
				writeFile(t, filepath.Join(dir, "main.go"), string(source))
				out := filepath.Join(dir, "out")
				err := driver.Build(driver.Options{
					Dir:      dir,
					Patterns: []string{"./..."},
					Out:      out,
					Runtime:  runtime,
					Format:   false,
				})
				if err != nil {
					t.Skipf("outside the subset:\n%v", err)
				}
				want, wantFail, ran := runWithin(t, dir, runLimit, "go", "run", "main.go")
				if !ran {
					t.Errorf("go run did not end within %v", runLimit)
					return
				}
				want = cutExitStatus(want)
				run(t, out, "dart", "pub", "get")
				got, gotFail, ended := runWithin(t, out, 2*time.Minute, "dart", "run", "bin/main.dart")
				if !ended {
					t.Error("the run did not end within 2 minutes")
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
}

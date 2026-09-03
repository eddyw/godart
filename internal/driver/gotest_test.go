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
	"peano":          "a deref of a type that names itself has no Dart form",
	"inline_literal": "reflect.Value.Pointer gives the identity of the object, thus runtime.FuncForPC gives nil for each value and the loop of the file never ends",
	"stack":          "the Dart VM holds a fixed stack, and the recursion overflows",
	"stackobj2":      "the Dart VM holds a fixed stack, and the recursion overflows",

	"fixedbugs/issue4316":   "a deref of a type that names itself has no Dart form",
	"fixedbugs/issue58300":  "reflect.Value.Pointer gives the identity of the object, and runtime.FuncForPC reads a program counter that the Dart VM gives not",
	"fixedbugs/issue58300b": "reflect.Value.Pointer gives the identity of the object, and runtime.FuncForPC reads a program counter that the Dart VM gives not",
	"fixedbugs/issue25897a": "the body of the reflect.MakeFunc value sends on a full channel, and a function that gives up control is outside MakeFunc, exactly as it is outside Value.Call",
	"fixedbugs/issue4667":   "the Dart VM gives no allocation count, and testing.AllocsPerRun reads one",
	"fixedbugs/issue4618":   "the Dart VM gives no allocation count, and testing.AllocsPerRun reads one",
	"fixedbugs/issue29735":  "runtime.FuncForPC walks the program counters, and the Dart VM gives none",
	"fixedbugs/issue5856":   "the panic path of Go holds a frame of gopanic, and the frame chain of the runtime holds none",
	"fixedbugs/issue72063":  "the Y combinator gives a local generic type that names itself, and the Dart form of its result erases to dynamic",
	"fixedbugs/issue29312":  "the composite literal nests 253 levels, and the Dart front end rejects a file with that many nested expressions",
	"fixedbugs/issue19182":  "a busy loop without a block operation never yields, and the other goroutine never runs",
	"linkx_run":             "the file runs go run on a neighbor file that the run directory does not hold, and the go run wrapper of the comparison adds an exit line that dart run does not add",
	"typeparam/nested":      "a local type of a generic function whose type argument names an interface type reaches $ifaceMethod, and the dispatch calls that sentinel with a receiver",
	"typeparam/issue48137":  "the Y combinator gives a local generic type that names itself, and the Dart form of its result erases to dynamic",

	"fixedbugs/bug260":     "the test reads the machine address of a value, and Dart gives none",
	"fixedbugs/bug352":     "2 values with no size take 2 cells, and gc gives them 1 address",
	"fixedbugs/issue75764": "the Dart VM holds a fixed stack, and the recursion overflows",
	"fixedbugs/issue13169": "the run holds 100000 channel rounds with a deep stack, and it needs more than the cap",
	"fixedbugs/issue34395": "a global array of 100 MB fills element by element, and the run needs more than the cap",
	"fixedbugs/issue78081": "the run holds 1600000 rounds of a byte append, and it needs more than the cap",

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
func runWithin(t *testing.T, dir string, limit time.Duration, name string, args ...string) (string, bool) {
	t.Helper()
	text, _, ok := output(dir, nil, limit, name, args...)
	return cutTrace(string(text)), ok
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
				want := runProgram(t, dir, "go", "run", "main.go")
				run(t, out, "dart", "pub", "get")
				got, ended := runWithin(t, out, 2*time.Minute, "dart", "run", "bin/main.dart")
				if !ended {
					t.Error("the run did not end within 2 minutes")
					return
				}
				if got != want {
					t.Errorf("the output differs\nGo:\n%s\nDart:\n%s", want, got)
				}
			})
		}
	}
}

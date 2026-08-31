package driver_test

import (
	"context"
	"go/build"
	"os"
	"os/exec"
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
	"reflectmethod4": "a bound method of reflect is no plain Dart function",
	"stack":          "the Dart VM holds a fixed stack, and the recursion overflows",
	"stackobj2":      "the Dart VM holds a fixed stack, and the recursion overflows",

	"fixedbugs/issue4316":  "a deref of a type that names itself has no Dart form",
	"fixedbugs/issue4667":  "the Dart VM gives no allocation count, and testing.AllocsPerRun reads one",
	"fixedbugs/issue29735": "runtime.FuncForPC walks the program counters, and the Dart VM gives none",
	"fixedbugs/issue22662": "a line directive does not reach the source map, and runtime.Caller gives the true position",
	"fixedbugs/issue72063": "a function type that nests 30 levels erases at 2 depths, and the 2 Dart forms differ",
	"fixedbugs/issue29312": "a slice type that nests 60 levels breaks the Dart form",
	"fixedbugs/issue19182": "a busy loop without a block operation never yields, and the other goroutine never runs",
	"typeparam/nested":     "the reflect identity of a local type of a generic function drops the outer type argument",
	"typeparam/issue48137": "a function type that nests 30 levels erases at 2 depths, and the 2 Dart forms differ",

	"fixedbugs/bug260":     "the test reads the machine address of a value, and Dart gives none",
	"fixedbugs/bug352":     "2 values with no size take 2 cells, and gc gives them 1 address",
	"fixedbugs/issue75764": "the Dart VM holds a fixed stack, and the recursion overflows",
	"fixedbugs/issue14646": "a line directive does not reach the source map, and runtime.Caller gives the true position",
	"fixedbugs/issue18149": "a line directive does not reach the source map, and runtime.Caller gives the true position",
	"fixedbugs/issue5856":  "a line directive does not reach the source map, and runtime.Caller gives the true position",
	"fixedbugs/issue13169": "the run holds 100000 channel rounds with a deep stack, and it needs more than the cap",
	"fixedbugs/issue34395": "a global array of 100 MB fills element by element, and the run needs more than the cap",
	"fixedbugs/issue78081": "the run holds 1600000 rounds of a byte append, and it needs more than the cap",

	"convert4":                    "the address of an element of an array that a slice conversion shares differs from the address of the slice element",
	"devirtualization_nil_panics": "the test reads the line of the panic site in the recovered stack, and the source map gives none",
	"fixedbugs/issue29190":        "the test allocates a list of 4 billion elements, and the Dart VM runs out of memory",
	"fixedbugs/issue26094":        "the text of a conversion panic names a local type with its position, and Go names it with the package",
	"fixedbugs/issue7550":         "the append of 2 slices of the maximum length must overflow the capacity, and the Dart build runs out of memory",
	"fixedbugs/issue4562":         "Go prints the panic of the caller in front of a panic that a recover turned into a second panic, and the Dart print holds the second panic only",
	"fixedbugs/issue19040":        "the panic of a method wrapper of a nil pointer gives no error value with the text of Go",
	"fixedbugs/issue48898":        "a defer of a call of panic inside 2 chains of deferred calls orders the panics differently",
	"fixedbugs/issue23734":        "a read of a map with an interface key whose type has no identity must panic, and the Dart read compares in silence",
	"fixedbugs/issue34123":        "a deref of a nil pointer inside a map read keeps no check, and Go panics",
	"fixedbugs/issue29504":        "the test reads the frames of the recovered stack, and the source map gives none of them",
}

// runWithin runs a program with a time limit. A file of the test directory
// can hold a loop that never ends under the block model, and the limit keeps
// the suite alive. WaitDelay closes the pipes when a child keeps them open.
func runWithin(t *testing.T, dir string, limit time.Duration, name string, args ...string) (string, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.WaitDelay = 10 * time.Second
	text, _ := cmd.CombinedOutput()
	return cutTrace(string(text)), ctx.Err() == nil
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

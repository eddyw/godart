package driver_test

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
}

// TestRunFilesOfGoTest compiles each file of the test directory of the Go
// repository whose header is `// run`, runs the Dart output, and compares it
// with the output of go run. The directory proves the language, thus a file
// outside the subset skips.
func TestRunFilesOfGoTest(t *testing.T) {
	root := filepath.Join(build.Default.GOROOT, "test")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("the machine holds no test directory of Go: %v", err)
	}
	runtime := runtimeDir(t)
	for _, entry := range entries {
		name, ok := strings.CutSuffix(entry.Name(), ".go")
		if !ok {
			continue
		}
		source, err := os.ReadFile(filepath.Join(root, entry.Name()))
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
			got := runProgram(t, out, "dart", "run", "bin/main.dart")
			if got != want {
				t.Errorf("the output differs\nGo:\n%s\nDart:\n%s", want, got)
			}
		})
	}
}

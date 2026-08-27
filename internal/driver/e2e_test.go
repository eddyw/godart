package driver_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

// TestPrograms compiles each Go program of testdata and runs the Dart output.
// The output must be equal to the output of the Go program.
func TestPrograms(t *testing.T) {
	root, err := filepath.Abs("../../testdata/programs")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := filepath.Abs("../../runtime/goruntime")
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
			want := runProgram(t, dir, "go", "run", ".")
			out := t.TempDir()
			err := driver.Build(driver.Options{
				Dir:      dir,
				Patterns: []string{"./..."},
				Out:      out,
				Runtime:  runtime,
				Format:   true,
			})
			if err != nil {
				t.Fatalf("the build failed:\n%v", err)
			}
			run(t, out, "dart", "pub", "get")
			got := runProgram(t, out, "dart", "run", "bin/main.dart")
			if got != want {
				t.Errorf("the output differs\nGo:\n%s\nDart:\n%s", want, got)
			}
		})
	}
}

// runProgram runs a compiled program. A panic writes a stack trace, and the 2
// runtimes write a different trace, thus the test cuts the trace away.
func runProgram(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	text, _ := cmd.CombinedOutput()
	return cutTrace(string(text))
}

func cutTrace(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "goroutine ") {
			return strings.Join(lines[:i], "\n")
		}
	}
	return text
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	text, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed in %s: %v\n%s", name, args, dir, err, text)
	}
	return string(text)
}

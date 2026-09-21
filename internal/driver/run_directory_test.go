package driver_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRunPreservesWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "cmd", "app")
	if err := os.MkdirAll(app, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/run-directory\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "input.txt"), "source directory\n")
	writeFile(t, filepath.Join(app, "main.go"), `package main
import ("fmt"; "os")
func main() {
 data,err := os.ReadFile("input.txt")
 if err != nil { panic(err) }
 if err := os.WriteFile("result.txt",data,0600); err != nil { panic(err) }
 fmt.Print(string(data))
}
`)
	want := run(t, dir, "go", "run", "./cmd/app")
	for _, frontend := range []string{"source", "resident"} {
		t.Run(frontend, func(t *testing.T) {
			if err := os.Remove(filepath.Join(dir, "result.txt")); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			code, err := driver.Run(driver.Options{Dir: dir, Patterns: []string{"./cmd/app"}, Out: t.TempDir(), Runtime: runtimeDir(t), Frontend: frontend}, nil, &output)
			if err != nil || code != 0 || output.String() != want {
				t.Fatalf("run differs: exit %d, error %v, output %q; Go %q", code, err, output.String(), want)
			}
			data, err := os.ReadFile(filepath.Join(dir, "result.txt"))
			if err != nil || string(data) != want {
				t.Fatalf("relative output differs: %q, %v", data, err)
			}
		})
	}
}

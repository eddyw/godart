package driver_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestGoFileArguments(t *testing.T) {
	for _, module := range []bool{false, true} {
		name := "without-module"
		if module {
			name = "with-module"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if module {
				writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fileargs\n\ngo 1.27\n")
			}
			writeFile(t, filepath.Join(dir, "LICENSE"), "File program license\n")
			sourceDir := dir
			if module {
				sourceDir = filepath.Join(dir, "cmd")
				if err := os.Mkdir(sourceDir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(t, filepath.Join(sourceDir, "main.go"), "package main\nimport \"fmt\"\nfunc main() { fmt.Println(message()) }\n")
			helper := "package main\nfunc message() string { return \"file arguments\" }\n"
			if module {
				if err := os.Mkdir(filepath.Join(dir, "dep"), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(dir, "dep", "dep.go"), "package dep\nconst Message = \"file arguments\"\n")
				helper = "package main\nimport \"example.com/fileargs/dep\"\nfunc message() string { return dep.Message }\n"
			}
			writeFile(t, filepath.Join(sourceDir, "helper.go"), helper)
			writeFile(t, filepath.Join(sourceDir, "unselected.go"), "package main\nvar invalid = missing\n")
			patterns := []string{filepath.Join(sourceDir, "main.go"), filepath.Join(sourceDir, "helper.go")}
			want := programOutcome(t, dir, "go", append([]string{"run"}, patterns...)...)
			if want.Exit != 0 {
				t.Fatal(want.Text)
			}
			opt := driver.Options{Dir: dir, Patterns: patterns, Out: t.TempDir(), Runtime: runtimeDir(t)}
			var output bytes.Buffer
			code, err := driver.Run(opt, nil, &output)
			if err != nil || code != want.Exit || cutTrace(output.String()) != want.Text {
				t.Fatalf("file arguments: Go=%+v Gart exit=%d error=%v output=%s", want, code, err, output.String())
			}
			license, err := os.ReadFile(filepath.Join(opt.Out, "LICENSE"))
			if err != nil || !strings.Contains(string(license), "File program license") {
				t.Fatalf("source license missing: %v\n%s", err, license)
			}
		})
	}
}

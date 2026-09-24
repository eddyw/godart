package driver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestModulePackageNames(t *testing.T) {
	for _, name := range []string{"http2", "assert", "class", "goruntime", "true", "null", "of", "type", "async", "normal", "123abc", "hello-world"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/"+name+"\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, "main.go"), "package main\nimport \"example.com/"+name+"/value\"\nfunc main() { println(value.Message()) }\n")
			if err := os.Mkdir(filepath.Join(dir, "value"), 0755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "value/value.go"), "package value\nfunc Message() string { return \"ready\" }\n")
			out := t.TempDir()
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
				t.Fatal(err)
			}
			runProgram(t, out, "dart", "pub", "get")
			got := runProgram(t, out, "dart", filepath.Join(out, "bin/main.dart"))
			if got != "ready" {
				t.Fatalf("unexpected program output: %q", got)
			}
		})
	}
}

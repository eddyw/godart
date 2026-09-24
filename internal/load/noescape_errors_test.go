package load_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestNoescapeDiagnostics(t *testing.T) {
	original, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "test", "fixedbugs", "issue48097.go"))
	if err != nil {
		t.Fatal(err)
	}
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ name, source string }{
		{"upstream_noescape", string(original)},
		{"body", "package p\n//go:noescape\nfunc f() {}\n"},
		{"blank_line", "package p\n//go:noescape\n\nfunc f() {}\n"},
		{"duplicate", "package p\n//go:noescape\n//go:noescape\nfunc f() {}\n"},
		{"method", "package p\ntype T int\n//go:noescape\nfunc (T) f() {}\n"},
		{"generic", "package p\n//go:noescape\nfunc f[T any]() {}\n"},
		{"external", "package p\n//go:noescape\nfunc f()\n"},
		{"following", "package p\n//go:noescape\nfunc f()\nfunc g() {}\n"},
		{"ordinary_comment", "package p\n// go:noescape\nfunc f() {}\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			name := filepath.Join(dir, "input.go")
			for path, source := range map[string]string{
				"go.mod":   "module example.com/noescape\n\ngo 1.27\n",
				"input.go": tt.source,
				"LICENSE":  string(license),
			} {
				if err := os.WriteFile(filepath.Join(dir, path), []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("go", "tool", "compile", "-e", "-o", filepath.Join(dir, "input.o"), name)
			want, nativeErr := cmd.CombinedOutput()
			_, err := load.Load(dir, ".")
			if (nativeErr == nil) != (err == nil) {
				t.Fatalf("Go error: %v\n%s\nGart error: %v", nativeErr, want, err)
			}
			if err != nil && strings.TrimSpace(err.Error()) != strings.TrimSpace(string(want)) {
				t.Fatalf("Go:\n%s\nGart:\n%v", want, err)
			}
		})
	}
}

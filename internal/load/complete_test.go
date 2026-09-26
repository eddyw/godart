package load_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestCompleteDiagnostics(t *testing.T) {
	original, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "test", "fixedbugs", "issue48097.go"))
	if err != nil {
		t.Fatal(err)
	}
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	for _, complete := range []bool{false, true} {
		t.Run(strconv.FormatBool(complete), func(t *testing.T) {
			for _, tt := range []struct{ name, source string }{
				{"upstream", string(original)},
				{"function", "package p\nfunc F()\n"},
				{"method", "package p\ntype T int\nfunc (T) F()\n"},
				{"init", "package p\nfunc init()\n"},
				{"body", "package p\nfunc F() {}\n"},
				{"linkname", "package p\nimport _ \"unsafe\"\n//go:linkname F other.F\nfunc F()\n"},
				{"linkname_export", "package p\nimport _ \"unsafe\"\n//go:linkname F\nfunc F()\n"},
				{"wasmimport", "package p\n//go:wasmimport env F\nfunc F()\n"},
				{"wasmimport_blank_line", "package p\n//go:wasmimport env F\n\nfunc F()\n"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					dir := t.TempDir()
					name := filepath.Join(dir, "input.go")
					for path, source := range map[string]string{
						"go.mod":   "module example.com/complete\n\ngo 1.27\n",
						"input.go": tt.source,
						"LICENSE":  string(license),
					} {
						if err := os.WriteFile(filepath.Join(dir, path), []byte(source), 0600); err != nil {
							t.Fatal(err)
						}
					}
					args := []string{"tool", "compile", "-e", "-o", filepath.Join(dir, "input.o")}
					if complete {
						args = append(args, "-complete")
					}
					cmd := exec.Command("go", append(args, name)...)
					want, nativeErr := cmd.CombinedOutput()
					_, err := load.LoadWithOptions(load.Options{Dir: dir, Patterns: []string{"."}, Complete: complete})
					if (nativeErr == nil) != (err == nil) {
						t.Fatalf("Go error: %v\n%s\nGart error: %v", nativeErr, want, err)
					}
					if err != nil && strings.TrimSpace(err.Error()) != strings.TrimSpace(string(want)) {
						t.Fatalf("Go:\n%s\nGart:\n%v", want, err)
					}
				})
			}
		})
	}
}

func TestCompleteDartDeclaration(t *testing.T) {
	dir := t.TempDir()
	for path, source := range map[string]string{
		"go.mod":   "module example.com/complete\n\ngo 1.27\n",
		"input.go": "package p\n//gart:dart dart:math max\nfunc F(a, b float64) float64\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, path), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := load.LoadWithOptions(load.Options{Dir: dir, Patterns: []string{"."}, Complete: true}); err != nil {
		t.Fatal(err)
	}
}

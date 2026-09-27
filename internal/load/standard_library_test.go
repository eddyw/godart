package load_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestStandardLibraryDiagnostics(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		standard     bool
	}{
		{"unknown", "//go:unknown\nfunc F() {}\n", true},
		{"unknown outside standard library", "//go:unknown\nfunc F() {}\n", false},
		{"keepalive", "//go:uintptrkeepalive\nfunc F(p uintptr) {}\n", true},
		{"nosplit", "//go:uintptrkeepalive\n//go:nosplit\nfunc F(p uintptr) {}\n", true},
		{"escapes", "//go:uintptrkeepalive\n//go:uintptrescapes\nfunc F(p uintptr) {}\n", true},
		{"bodyless", "//go:uintptrkeepalive\nfunc F(p uintptr)\n", true},
		{"parse error before body error", "//go:unknown\nfunc A() {}\n//go:noescape\nfunc B() {}\n", true},
		{"noinline", "//go:noinline\nfunc F() {}\n", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			name := filepath.Join(dir, "input.go")
			for path, source := range map[string]string{
				"go.mod":   "module example.com/directives\n\ngo 1.27\n",
				"input.go": "package p\n" + tt.source,
			} {
				if err := os.WriteFile(filepath.Join(dir, path), []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{"tool", "compile", "-e", "-o", filepath.Join(dir, "input.o")}
			if tt.standard {
				args = append(args, "-std")
			}
			want, nativeErr := exec.Command("go", append(args, name)...).CombinedOutput()
			_, err := load.LoadWithOptions(load.Options{Dir: dir, Patterns: []string{"."}, StandardLibrary: tt.standard})
			if (nativeErr == nil) != (err == nil) {
				t.Fatalf("Go error: %v\n%s\nGart error: %v", nativeErr, want, err)
			}
			if err != nil && strings.TrimSpace(err.Error()) != strings.TrimSpace(string(want)) {
				t.Fatalf("Go:\n%s\nGart:\n%v", want, err)
			}
		})
	}
}

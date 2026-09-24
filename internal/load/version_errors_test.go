package load_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestVersionDiagnosticSourcePositions(t *testing.T) {
	for _, tt := range []struct{ name, language, source string }{
		{"module", "1.17", "package p\n//line mapped.go:100\nfunc f[T any](x T) {}\n"},
		{"file", "1.22", "//go:build go1.21\n\npackage p\n//line mapped.go:100\nfunc f() { for range 10 {} }\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, text := range map[string]string{"go.mod": "module example.com/version\n\ngo " + tt.language + "\n", "first.go": "package p\n", "version.go": tt.source} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("go", "tool", "compile", "-e", "-C", "-lang=go"+tt.language, "-o", "out.o", "first.go", "version.go")
			cmd.Dir = dir
			native, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("native accepted invalid source")
			}
			_, err = load.Load(dir, ".")
			if err == nil {
				t.Fatal("Gart accepted invalid source")
			}
			normalize := func(text string) string {
				lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(text, dir+string(filepath.Separator), "")), "\n")
				sort.Strings(lines)
				return strings.Join(lines, "\n")
			}
			if got, want := normalize(err.Error()), normalize(string(native)); got != want {
				t.Fatalf("Go:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}

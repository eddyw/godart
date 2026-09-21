package load_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestImportErrorsPreserveTypeErrors(t *testing.T) {
	for _, tt := range []struct{ name, diagnostic string }{
		{"/foo", "import path cannot be absolute path"},
		{"unicode//utf8", `non-canonical import path "unicode//utf8" (should be "unicode/utf8")`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range map[string]string{
				"go.mod":  "module example.com/import-errors\n\ngo 1.27\n",
				"main.go": "package main\nimport _ \"" + tt.name + "\"\nvar x = missing\n",
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := load.Load(dir, ".")
			if err == nil {
				t.Fatal("invalid source produced no diagnostic")
			}
			for _, want := range []string{tt.diagnostic, "undefined: missing"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("missing %q in %v", want, err)
				}
			}
		})
	}
}

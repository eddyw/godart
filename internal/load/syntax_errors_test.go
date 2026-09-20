package load_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestLexicalErrorsPreserveTypeErrors(t *testing.T) {
	for _, tt := range []struct {
		name, source, diagnostic string
	}{
		{"number", "var x = 0x", "hexadecimal literal has no digits"},
		{"string", `var x = "\q"`, "unknown escape"},
		{"tag", `type S struct { Field int "\q" }`, "unknown escape"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range map[string]string{
				"go.mod": "module example.com/lexical\n\ngo 1.27\n",
				"bad.go": "package lexical\n" + tt.source + "\nvar y = missing\n",
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := load.Load(dir, "./...")
			if err == nil {
				t.Fatal("invalid source produced no diagnostic")
			}
			for _, want := range []string{tt.diagnostic, "undefined: missing"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("missing %q in diagnostics:\n%v", want, err)
				}
			}
			if strings.Contains(err.Error(), "malformed constant") {
				t.Errorf("duplicate literal diagnostic:\n%v", err)
			}
		})
	}
}

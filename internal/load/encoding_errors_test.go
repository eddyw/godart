package load_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestEncodingDiagnostics(t *testing.T) {
	cases := map[string]string{
		"prefix":       "var \xc2A int\nvar y=A\nvar z=missing\n",
		"inside":       "var x\xc2y int\nvar z=xy\n",
		"suffix":       "var x\xc2 int\nvar z=x\n",
		"same-line":    "var x=\xc2missing\n",
		"NUL-name":     "var x\x00y int\nvar z=xy\n",
		"NUL-literal":  "var x=\"bad\x00\"\nvar z=missing\n",
		"UTF8-literal": "var x=\"bad\xff\"\nvar z=missing\n",
		"number":       "const x=1\xff2\nvar z=missing\n",
		"operator":     "var x=1<\xff<2\nvar z=missing\n",
		"comment":      "// bad\xff\nvar z=missing\n",
		"NUL-comment":  "// bad\x00\nvar z=missing\n",
		"directive":    "//line generated.go:100\nvar \xc2A int\nvar y=missing\n",
		"unicode":      "var 世\xc2界 int\nvar z=世界\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "main.go")
			if err := os.WriteFile(source, []byte("package lexical\n"+body), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/lexical\n\ngo 1.27\n"), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "tool", "compile", "-e", "-C", "-o", filepath.Join(dir, "out.o"), source)
			native, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("native accepted invalid source")
			}
			_, err = load.Load(dir, "./...")
			if err == nil {
				t.Fatal("Gart accepted invalid source")
			}
			normalize := func(s string) string {
				s = regexp.MustCompile(`(?m)^(.*?\.go:[0-9]+):[0-9]+:`).ReplaceAllString(s, "$1:")
				s = strings.ReplaceAll(s, dir+string(filepath.Separator), "")
				lines := strings.Split(strings.TrimSpace(s), "\n")
				sort.Strings(lines)
				return strings.Join(lines, "\n")
			}
			want, got := normalize(string(native)), normalize(err.Error())
			if got != want {
				t.Fatalf("Go:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}

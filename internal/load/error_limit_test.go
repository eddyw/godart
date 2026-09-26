package load_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestErrorLimitDiagnostics(t *testing.T) {
	original, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "test", "fixedbugs", "issue20298.go"))
	if err != nil {
		t.Fatal(err)
	}
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(t.TempDir(), "importcfg")
	imports, err := exec.Command("go", "list", "-export", "-f", "{{if .Export}}packagefile {{.ImportPath}}={{.Export}}{{end}}", "std").CombinedOutput()
	if err != nil {
		t.Fatalf("import configuration: %v\n%s", err, imports)
	}
	if err := os.WriteFile(config, imports, 0600); err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{"upstream": string(original)}
	for _, count := range []int{9, 10, 11} {
		source := "package p\n"
		for i := 0; i < count; i++ {
			source += fmt.Sprintf("var v%d = missing%d\n", i, i)
		}
		sources[fmt.Sprintf("undefined_%d", count)] = source
	}
	sources["secondary"] = "package p\n" + strings.Repeat("var v int\n", 12)
	sources["duplicate"] = "package p\nvar v = " + strings.Repeat("missing + ", 12) + "0\n"
	sources["syntax"] = "package p\n" + strings.Repeat("var _ = @\n", 12)
	sources["complete"] = "package p\n"
	for i := 0; i < 6; i++ {
		sources["complete"] += fmt.Sprintf("func f%d()\n//go:noescape\nfunc g%d() {}\n", i, i)
	}
	for name, source := range sources {
		for _, limit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/limit=%t", name, limit), func(t *testing.T) {
				dir := t.TempDir()
				file := filepath.Join(dir, "input.go")
				for path, text := range map[string]string{
					"go.mod":   "module example.com/errorlimit\n\ngo 1.27\n",
					"input.go": source,
					"LICENSE":  string(license),
				} {
					if err := os.WriteFile(filepath.Join(dir, path), []byte(text), 0600); err != nil {
						t.Fatal(err)
					}
				}
				flag := "-e"
				if limit {
					flag = "-e=0"
				}
				args := []string{"tool", "compile", flag, "-importcfg=" + config, "-o", filepath.Join(dir, "input.o")}
				if name == "complete" {
					args = append(args, "-complete")
				}
				cmd := exec.Command("go", append(args, file)...)
				want, nativeErr := cmd.CombinedOutput()
				_, err := load.LoadWithOptions(load.Options{Dir: dir, Patterns: []string{"."}, LimitErrors: limit, Complete: name == "complete"})
				if nativeErr == nil || err == nil {
					t.Fatalf("Go error: %v\n%s\nGart error: %v", nativeErr, want, err)
				}
				if strings.TrimSpace(err.Error()) != strings.TrimSpace(string(want)) {
					t.Fatalf("Go:\n%s\nGart:\n%v", want, err)
				}
			})
		}
	}
	t.Run("packages", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/errorlimit\n\ngo 1.27\n"), 0600); err != nil {
			t.Fatal(err)
		}
		var want []string
		for _, name := range []string{"a", "b"} {
			path := filepath.Join(dir, name)
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(path, "input.go")
			if err := os.WriteFile(file, []byte(sources["undefined_11"]), 0600); err != nil {
				t.Fatal(err)
			}
			text, err := exec.Command("go", "tool", "compile", "-e=0", "-o", filepath.Join(path, "input.o"), file).CombinedOutput()
			if err == nil {
				t.Fatal("Go accepted invalid source")
			}
			want = append(want, strings.TrimSpace(string(text)))
		}
		_, err := load.LoadWithOptions(load.Options{Dir: dir, Patterns: []string{"./..."}, LimitErrors: true})
		if err == nil || err.Error() != strings.Join(want, "\n") {
			t.Fatalf("Go:\n%s\nGart:\n%v", strings.Join(want, "\n"), err)
		}
	})
}

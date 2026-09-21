package driver_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinkedFunctionCache(t *testing.T) {
	source, out := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(source, "storage"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(source, "go.mod"), "module example.com/linkcache\n\ngo 1.27\n")
	writeFile(t, filepath.Join(source, "storage/value.go"), "package storage\nfunc number() int {return 8}\nfunc Read() int {return number()}\n")
	plain := `package main
import "example.com/linkcache/storage"
func main() {println(storage.Read())}
`
	linked := `package main
import ("example.com/linkcache/storage"; _ "unsafe")
//go:linkname number example.com/linkcache/storage.number
func number() int
func main() {println(number()+storage.Read())}
`
	for _, c := range []struct {
		source, want string
		cached       bool
	}{{plain, "8\n", false}, {plain, "8\n", true}, {linked, "16\n", false}, {linked, "16\n", true}, {plain, "8\n", false}} {
		writeFile(t, filepath.Join(source, "main.go"), c.source)
		stats, err := driver.BuildWithStats(driver.Options{Dir: source, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)})
		if err != nil {
			t.Fatal(err)
		}
		if got := slices.Contains(stats.Cached, "example.com/linkcache/storage"); got != c.cached {
			t.Fatalf("storage cached=%t, want %t", got, c.cached)
		}
		prepareDart(t, out)
		if got := run(t, out, "dart", "bin/main.dart"); got != c.want {
			t.Fatalf("got %q, want %q", got, c.want)
		}
	}
}

func TestLinkedFunctionDiagnostics(t *testing.T) {
	for _, c := range []struct{ name, lib, source, want string }{
		{"missing-package", "func Value() int {return 1}", `//go:linkname value absent.Value
func value() int
func main(){println(value())}`, "requires an imported target package"},
		{"missing-function", "func Value() int {return 1}", `//go:linkname value example.com/linkdiagnostic/lib.Missing
func value() int
func main(){println(value())}`, "requires a target function with the same type"},
		{"different-type", "func Value() int {return 1}", `//go:linkname value example.com/linkdiagnostic/lib.Value
func value() string
func main(){println(value())}`, "requires a target function with the same type"},
		{"variable-target", "var Value func() int", `//go:linkname value example.com/linkdiagnostic/lib.Value
func value() int
func main(){println(value())}`, "requires a target function with the same type"},
		{"body", "func Value() int {return 1}", `//go:linkname value example.com/linkdiagnostic/lib.Value
func value() int {return 2}
func main(){println(value())}`, "linked function aliases must have no body"},
		{"retargeted", `import _ "unsafe"
//go:linkname Value alternate.Value
func Value() int`, `//go:linkname value example.com/linkdiagnostic/lib.Value
func value() int
func main(){println(value())}`, "targets another linkname declaration"},
		{"unused", "func Value() int {return 1}", `//go:linkname value absent.Value
func value() int
func main(){}`, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "lib"), 0755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/linkdiagnostic\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, "lib/lib.go"), "package lib\n"+c.lib+"\n")
			writeFile(t, filepath.Join(dir, "main.go"), "package main\nimport (_ \"unsafe\"; _ \"example.com/linkdiagnostic/lib\")\n"+c.source+"\n")
			err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: t.TempDir(), Runtime: runtimeDir(t)})
			if c.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want %q", err, c.want)
			}
		})
	}
}

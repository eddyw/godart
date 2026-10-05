package driver_test

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestCompilerCompoundReturns(t *testing.T) {
	for _, name := range []string{"compound_test.go", "namedReturn_test.go"} {
		t.Run(name, func(t *testing.T) {
			const pkg = "cmd/compile/internal/test/testdata"
			sourcePath := "src/" + pkg + "/" + name
			source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), sourcePath))
			if err != nil {
				t.Fatal(err)
			}
			license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
			if err != nil {
				t.Fatal(err)
			}
			file, err := parser.ParseFile(token.NewFileSet(), sourcePath, source, 0)
			if err != nil {
				t.Fatal(err)
			}
			selection := parity.Package{Path: pkg}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
					selection.Tests = append(selection.Tests, parity.Test{Name: fn.Name.Name, File: name, Kind: "Test", Eligible: true})
				}
			}
			if len(selection.Tests) == 0 {
				t.Fatal("the original compiler assertion fixture has no tests")
			}
			inventory := parity.Inventory{
				SourceSHA256: fmt.Sprintf("%x", sha256.Sum256(source)),
				Fixtures: []parity.SourceFile{
					{Path: sourcePath, SHA256: fmt.Sprintf("%x", sha256.Sum256(source))},
					{Path: "LICENSE", SHA256: fmt.Sprintf("%x", sha256.Sum256(license))},
				},
				Packages: []parity.Package{selection},
			}
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/compilerassert\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, name), string(source))
			writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
			out := t.TempDir()
			stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
			if err != nil {
				t.Fatal(err)
			}
			if len(stats.Binaries) != 1 {
				t.Fatalf("got %d binaries", len(stats.Binaries))
			}
			prepareDart(t, out)
			args := upstreamTestArgs()
			for _, target := range []string{"vm", "aot", "js", "wasm"} {
				t.Run(target, func(t *testing.T) {
					var result programResult
					if target == "vm" {
						result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
					} else {
						result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
					}
					compareUpstreamInventory(t, "compiler-compound-returns/"+name, pkg, "", target, dir, result.Text, result.Exit, inventory, nil, nil, nil, true)
				})
			}

		})
	}
}

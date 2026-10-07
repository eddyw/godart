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

func runCompilerAssertionFiles(t *testing.T, group string, names []string) {
	const pkg = "cmd/compile/internal/test/testdata"
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/compilerassert\n\ngo 1.27\n")
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	selection := parity.Package{Path: pkg}
	inventory := parity.Inventory{Fixtures: []parity.SourceFile{{Path: "LICENSE", SHA256: fmt.Sprintf("%x", sha256.Sum256(license))}}}
	digest := sha256.New()
	// Upstream TestCode compiles these files into 1 test binary.
	// Keep the original files together to avoid repeated target compilation.
	for _, name := range names {
		sourcePath := "src/" + pkg + "/" + name
		source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), sourcePath))
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), sourcePath, source, 0)
		if err != nil {
			t.Fatal(err)
		}
		start := len(selection.Tests)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
				selection.Tests = append(selection.Tests, parity.Test{Name: fn.Name.Name, File: name, Kind: "Test", Eligible: true})
			}
		}
		if len(selection.Tests) == start {
			t.Fatal("the original compiler assertion fixture has no tests")
		}
		inventory.Fixtures = append(inventory.Fixtures, parity.SourceFile{Path: sourcePath, SHA256: fmt.Sprintf("%x", sha256.Sum256(source))})
		fmt.Fprintf(digest, "%s\x00%s\x00", sourcePath, source)
		writeFile(t, filepath.Join(dir, name), string(source))
	}
	inventory.SourceSHA256 = fmt.Sprintf("%x", digest.Sum(nil))
	inventory.Packages = []parity.Package{selection}
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
			compareUpstreamInventory(t, group, pkg, "", target, dir, result.Text, result.Exit, inventory, nil, nil, nil, true)
		})
	}
}

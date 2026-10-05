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

func TestCompilerPhiOriginal(t *testing.T) {
	const pkg = "cmd/compile/internal/test/testdata"
	const sourcePath = "src/" + pkg + "/phi_test.go"
	source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), sourcePath))
	if err != nil {
		t.Fatal(err)
	}
	const dependencyPath = "src/" + pkg + "/unsafe_test.go"
	dependency, err := os.ReadFile(filepath.Join(runtime.GOROOT(), dependencyPath))
	if err != nil {
		t.Fatal(err)
	}
	dependencyTokens := token.NewFileSet()
	dependencyFile, err := parser.ParseFile(dependencyTokens, dependencyPath, dependency, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sharedDeclaration *ast.GenDecl
	for _, decl := range dependencyFile.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR {
			continue
		}
		for _, spec := range group.Specs {
			value := spec.(*ast.ValueSpec)
			for _, name := range value.Names {
				if name.Name == "always" {
					sharedDeclaration = group
				}
			}
		}
	}
	if sharedDeclaration == nil {
		t.Fatal("the original compiler phi dependency has no always declaration")
	}
	packageStart := dependencyTokens.Position(dependencyFile.Package).Offset
	packageEnd := dependencyTokens.Position(dependencyFile.Name.End()).Offset
	declarationStart := dependencyTokens.Position(sharedDeclaration.Pos()).Offset
	declarationEnd := dependencyTokens.Position(sharedDeclaration.End()).Offset
	companion := string(dependency[:packageStart]) + string(dependency[packageStart:packageEnd]) + "\n\n" + string(dependency[declarationStart:declarationEnd]) + "\n"
	sourceDigest := sha256.New()
	sourceDigest.Write(source)
	sourceDigest.Write(dependency)
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
			selection.Tests = append(selection.Tests, parity.Test{Name: fn.Name.Name, File: "phi_test.go", Kind: "Test", Eligible: true})
		}
	}
	if len(selection.Tests) == 0 {
		t.Fatal("the original compiler assertion fixture has no tests")
	}
	inventory := parity.Inventory{
		SourceSHA256: fmt.Sprintf("%x", sourceDigest.Sum(nil)),
		Fixtures: []parity.SourceFile{
			{Path: sourcePath, SHA256: fmt.Sprintf("%x", sha256.Sum256(source))},
			{Path: dependencyPath, SHA256: fmt.Sprintf("%x", sha256.Sum256(dependency))},
			{Path: "LICENSE", SHA256: fmt.Sprintf("%x", sha256.Sum256(license))},
		},
		Packages: []parity.Package{selection},
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/compilerassert\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "phi_test.go"), string(source))
	writeFile(t, filepath.Join(dir, "phi_dependency.go"), companion)
	if err := os.Mkdir(filepath.Join(dir, "_sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "_sources", "unsafe_test.go"), string(dependency))
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
			compareUpstreamInventory(t, "compiler-phi", pkg, "", target, dir, result.Text, result.Exit, inventory, nil, nil, nil, true)
		})
	}
}

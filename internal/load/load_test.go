package load_test

import (
	"go/build"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

// TestLoadTestsReachesAVariantOfTheSurface proves that the overlay replaces the
// files of a package of the surface that the load holds only as a test variant.
//
// The real package testing imports io, thus a test build of io holds testing
// only as the variant testing [io.test]. A miss of that variant keeps the real
// files of testing, and the load then reports a redeclaration.
//
// The test files of io hold other gaps, thus the load can still report an error.
// The assertion reads only the redeclaration.
func TestLoadTestsReachesAVariantOfTheSurface(t *testing.T) {
	dir := filepath.Join(build.Default.GOROOT, "src", "io")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("the machine holds no source of the package io: %v", err)
	}
	_, err := load.LoadTests(dir, ".")
	if err == nil {
		return
	}
	for _, mark := range []string{"already declared", "redeclared in this block"} {
		if strings.Contains(err.Error(), mark) {
			t.Errorf("the overlay missed a test variant of the surface:\n%v", err)
		}
	}
}

func loadHello(t *testing.T) (pkgPath string, scope *types.Scope) {
	t.Helper()
	dir, err := filepath.Abs("testdata/hello")
	if err != nil {
		t.Fatal(err)
	}
	found, err := load.Load(dir, "./...")
	if err != nil {
		t.Fatal(err)
	}
	if len(found.Packages) != 1 {
		t.Fatalf("got %d packages, want 1", len(found.Packages))
	}
	p := found.Packages[0]
	if p.Types == nil || p.TypesInfo == nil {
		t.Fatal("the package holds no type information")
	}
	if len(p.Syntax) != 2 {
		t.Fatalf("got %d syntax trees, want 2", len(p.Syntax))
	}
	return p.PkgPath, p.Types.Scope()
}

func TestLoadGivesTypeInformation(t *testing.T) {
	pkgPath, scope := loadHello(t)
	if pkgPath != "example.com/hello" {
		t.Errorf("got package path %q, want %q", pkgPath, "example.com/hello")
	}
	fn, _ := scope.Lookup("greet").(*types.Func)
	if fn == nil {
		t.Fatal("the scope holds no function greet")
	}
	if got := fn.Signature().String(); got != "func(name string) string" {
		t.Errorf("got signature %q, want %q", got, "func(name string) string")
	}
}

func TestLoadAppliesTheDartBuildTag(t *testing.T) {
	_, scope := loadHello(t)
	c, _ := scope.Lookup("target").(*types.Const)
	if c == nil {
		t.Fatal("the scope holds no constant target")
	}
	if got := c.Val().String(); got != `"dart"` {
		t.Errorf("got target %s, want %q", got, "dart")
	}
}

func TestLoadReportsATypeError(t *testing.T) {
	dir, err := filepath.Abs("testdata/broken")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := load.Load(dir, "./..."); err == nil {
		t.Fatal("got no error for a package that does not type check")
	}
}

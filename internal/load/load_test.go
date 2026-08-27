package load_test

import (
	"go/types"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func loadHello(t *testing.T) (pkgPath string, scope *types.Scope) {
	t.Helper()
	dir, err := filepath.Abs("testdata/hello")
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := load.Load(dir, "./...")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	p := pkgs[0]
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

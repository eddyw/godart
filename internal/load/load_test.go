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

// TestADiffReachesASourceFileOfANativePackage proves that a `.patch` file of
// the overlay changes the real Go source of a native package.
//
// The compiler reads bytes from the real Go source, and
// bytes/buffer.go.patch removes the deferred call of recover of growSlice. The
// subset holds no recover, thus the load must give a body with no such call
// and with every other line of Go.
func TestADiffReachesASourceFileOfANativePackage(t *testing.T) {
	dir, err := filepath.Abs("testdata/hello")
	if err != nil {
		t.Fatal(err)
	}
	found, err := load.Load(dir, "./...")
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(build.Default.GOROOT, "src", "bytes", "buffer.go")
	src, ok := found.Sources[name]
	if !ok {
		t.Fatalf("the load holds no source of %s", name)
	}
	text := string(src)
	if !strings.Contains(text, "func growSlice(b []byte, n int) []byte {") {
		t.Error("the patched file holds no growSlice")
	}
	if strings.Contains(text, "if recover() != nil {") {
		t.Error("the patched file still holds the deferred call of recover")
	}
	if !strings.Contains(text, "func (b *Buffer) ReadFrom(r io.Reader) (n int64, err error) {") {
		t.Error("the patched file lost a line that the diff does not name")
	}
}

// TestAnOverlayFileWithANewNameKeepsItsOwnPath proves that an overlay file
// whose base name matches no file of the real package takes a path of its own.
//
// Such a file took the free slot of an unrelated real file before, thus each
// report of the compiler named a source that holds no such line. A person who
// read the position then looked at code of another subject.
//
// strings/extra.go of the overlay holds the name, and the real package strings
// holds no file of it.
func TestAnOverlayFileWithANewNameKeepsItsOwnPath(t *testing.T) {
	dir, err := filepath.Abs("testdata/hello")
	if err != nil {
		t.Fatal(err)
	}
	found, err := load.Load(dir, "./...")
	if err != nil {
		t.Fatal(err)
	}
	stringsDir := filepath.Join(build.Default.GOROOT, "src", "strings")
	name := filepath.Join(stringsDir, "extra.go")
	src, ok := found.Sources[name]
	if !ok {
		t.Fatalf("the load holds no source of %s", name)
	}
	if !strings.Contains(string(src), "package strings") {
		t.Errorf("the source of %s holds no package clause of strings", name)
	}
	// No other file of the package holds that content, thus a position of it
	// names extra.go alone.
	for other, content := range found.Sources {
		if other == name || filepath.Dir(other) != stringsDir {
			continue
		}
		if string(content) == string(src) {
			t.Errorf("the source of extra.go also stands at %s", other)
		}
	}
	// The real file of the same directory that no overlay file replaces holds
	// the empty form, and its path stays free of foreign code.
	real := filepath.Join(stringsDir, "builder.go")
	if content, ok := found.Sources[real]; ok && strings.Contains(string(content), "func ") {
		t.Errorf("%s holds a body, want the empty form or the real file", real)
	}
}

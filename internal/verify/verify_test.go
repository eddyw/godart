package verify_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/verify"
)

// run type checks src and returns the result of the verifier.
func run(t *testing.T, src string) error {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "a.go", src, 0)
	if err != nil {
		t.Fatalf("the source does not parse: %v", err)
	}
	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("example.com/a", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("the source does not type check: %v", err)
	}
	return verify.Package(fset, []*ast.File{file}, info)
}

const head = "package a\n\ntype point struct{ x, y int }\n\n"

func accept(t *testing.T, name, body string) {
	t.Helper()
	acceptFile(t, name, head+body)
}

func acceptFile(t *testing.T, name, src string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		if err := run(t, src); err != nil {
			t.Errorf("got a rejection, want none:\n%v", err)
		}
	})
}

func reject(t *testing.T, name, want, body string) {
	t.Helper()
	rejectFile(t, name, want, head+body)
}

func rejectFile(t *testing.T, name, want, src string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		err := run(t, src)
		if err == nil {
			t.Fatal("got no rejection, want one")
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("got %q, want a message that holds %q", err.Error(), want)
		}
		if !strings.Contains(err.Error(), "a.go:") {
			t.Errorf("got %q, want a source position", err.Error())
		}
	})
}

func TestImports(t *testing.T) {
	rejectFile(t, "unsafe", "rejects the package unsafe", `package a

import "unsafe"

var _ = unsafe.Sizeof(0)
`)
	rejectFile(t, "curated surface", "outside the supported surface", `package a

import "os"

var _ = os.Args
`)
	accept(t, "no import", "var _ = 1\n")
}

func TestRecover(t *testing.T) {
	reject(t, "call", "rejects a call to recover", `func f() { _ = recover() }`)
	accept(t, "shadowed name", `func f() { recover := 1; _ = recover }`)
	accept(t, "panic", `func f() { panic("stop") }`)
}

func TestAddress(t *testing.T) {
	accept(t, "struct variable", `func f() { var p point; _ = &p }`)
	accept(t, "struct literal", `func f() { _ = &point{} }`)
	accept(t, "struct field of a struct", `
type line struct{ a point }

func f() { var l line; _ = &l.a }
`)
	accept(t, "struct element of a slice", `func f(s []point) { _ = &s[0] }`)
	reject(t, "int variable", "the address of int", `func f() { var n int; _ = &n }`)
	reject(t, "int field", "the address of int", `func f() { var p point; _ = &p.x }`)
	reject(t, "int element", "the address of int", `func f(s []int) { _ = &s[0] }`)
	reject(t, "array element", "the address of int", `func f() { var a [2]int; _ = &a[0] }`)
	reject(t, "string variable", "the address of string", `func f() { var s string; _ = &s }`)
}

func TestNew(t *testing.T) {
	accept(t, "struct type", `func f() { _ = new(point) }`)
	reject(t, "int type", "rejects new(int)", `func f() { _ = new(int) }`)
	reject(t, "expression", "rejects new(int)", `func f() { _ = new(1 + 2) }`)
	reject(t, "slice type", "rejects new([]int)", `func f() { _ = new([]int) }`)
}

func TestComplex(t *testing.T) {
	reject(t, "variable", "rejects the type complex128", `func f() { var c complex128; _ = c }`)
	reject(t, "builtin", "rejects the builtin complex", `func f() { _ = complex(1.0, 2.0) }`)
	reject(t, "real", "rejects the builtin real", `func f(c complex64) { _ = real(c) }`)
	accept(t, "float", `func f() { var x float64; _ = x }`)
}

func TestReportsEachRejection(t *testing.T) {
	err := run(t, head+`func f() {
	var n int
	_ = &n
	_ = new(int)
}`)
	if err == nil {
		t.Fatal("got no rejection, want 2")
	}
	if got := strings.Count(err.Error(), "a.go:"); got != 2 {
		t.Errorf("got %d rejections, want 2:\n%v", got, err)
	}
}

// runNamed type checks src under a file name and gives the result of the
// verifier. The rule for TestMain reads the file name.
func runNamed(t *testing.T, name, src string) error {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		t.Fatalf("the source does not parse: %v", err)
	}
	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("example.com/a", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("the source does not type check: %v", err)
	}
	return verify.Package(fset, []*ast.File{file}, info)
}

func TestTestMain(t *testing.T) {
	const src = `package a

type m struct{}

func (v *m) Run() int { return 0 }

func TestMain(v *m) { _ = v.Run() }
`
	err := runNamed(t, "a_test.go", src)
	if err == nil {
		t.Fatal("got no rejection of TestMain, want one")
	}
	if !strings.Contains(err.Error(), "rejects a TestMain function") {
		t.Errorf("got %q, want a message about TestMain", err.Error())
	}
	if !strings.Contains(err.Error(), "a_test.go:") {
		t.Errorf("got %q, want a source position", err.Error())
	}
	if err := runNamed(t, "a.go", src); err != nil {
		t.Errorf("got a rejection outside a test file, want none:\n%v", err)
	}
	if err := runNamed(t, "a_test.go", `package a

func TestAdd(v *int) { _ = v }
`); err != nil {
		t.Errorf("got a rejection of a test function, want none:\n%v", err)
	}
}

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

import "net/http"

var _ = http.StatusOK
`)
	acceptFile(t, "a blank import of unsafe", `package a

import _ "unsafe"
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
	accept(t, "int variable", `func f() { var n int; _ = &n }`)
	accept(t, "string variable", `func f() { var s string; _ = &s }`)
	accept(t, "slice variable", `func f() { var s []int; _ = &s }`)
	accept(t, "parameter", `func f(n int) { _ = &n }`)
	accept(t, "package variable", `
var total int

func f() { _ = &total }
`)
	reject(t, "int field", "the address of int", `func f() { var p point; _ = &p.x }`)
	reject(t, "int element", "the address of int", `func f(s []int) { _ = &s[0] }`)
	reject(t, "array element", "the address of int", `func f() { var a [2]int; _ = &a[0] }`)
	reject(t, "a call result", "the address of int",
		`func g() int { return 1 }

func f() { _ = &[]int{g()}[0] }`)
}

func TestNew(t *testing.T) {
	accept(t, "struct type", `func f() { _ = new(point) }`)
	accept(t, "int type", `func f() { _ = new(int) }`)
	accept(t, "expression", `func f() { _ = new(1 + 2) }`)
	accept(t, "slice type", `func f() { _ = new([]int) }`)
}

func TestComplex(t *testing.T) {
	reject(t, "variable", "rejects the type complex128", `func f() { var c complex128; _ = c }`)
	reject(t, "builtin", "rejects the builtin complex", `func f() { _ = complex(1.0, 2.0) }`)
	reject(t, "real", "rejects the builtin real", `func f(c complex64) { _ = real(c) }`)
	accept(t, "float", `func f() { var x float64; _ = x }`)
}

func TestReportsEachRejection(t *testing.T) {
	err := run(t, head+`func f(s []int) {
	var p point
	_ = &p.x
	_ = &s[0]
}`)
	if err == nil {
		t.Fatal("got no rejection, want 2")
	}
	if got := strings.Count(err.Error(), "a.go:"); got != 2 {
		t.Errorf("got %d rejections, want 2:\n%v", got, err)
	}
}

// TestAnonStructCrossesAPackage proves that stage 2 rejects an unnamed struct
// type in an exported declaration. Such a type gives 1 Dart class in each
// package that uses it, and Dart holds the 2 classes apart.
func TestAnonStructCrossesAPackage(t *testing.T) {
	const want = "unnamed struct type to another package"
	rejectFile(t, "an exported result", want, `package a

func Make() struct{ N int } { return struct{ N int }{N: 1} }
`)
	rejectFile(t, "an exported parameter", want, `package a

func Take(v struct{ N int }) int { return v.N }
`)
	rejectFile(t, "an exported variable", want, `package a

var Value struct{ N int }
`)
	rejectFile(t, "an exported field", want, `package a

type Box struct {
	Inner struct{ N int }
}
`)
	rejectFile(t, "a slice of an unnamed struct", want, `package a

func List() []struct{ N int } { return nil }
`)
	rejectFile(t, "an exported method", want, `package a

type Box struct{ n int }

func (b Box) Get() struct{ N int } { return struct{ N int }{N: b.n} }
`)
	acceptFile(t, "an unexported function", `package a

func make1() struct{ N int } { return struct{ N int }{N: 1} }
`)
	acceptFile(t, "a local variable", `package a

func F() int {
	v := struct{ n int }{n: 1}
	return v.n
}
`)
	acceptFile(t, "an unexported field", `package a

type Box struct {
	inner struct{ N int }
}
`)
	acceptFile(t, "an unexported method", `package a

type Box struct{ n int }

func (b Box) get() struct{ N int } { return struct{ N int }{N: b.n} }
`)
	acceptFile(t, "a method of an unexported type", `package a

type box struct{ n int }

func (b box) Get() struct{ N int } { return struct{ N int }{N: b.n} }
`)
}

// TestRangeFuncBranch proves that stage 2 accepts a range loop over a function,
// and that a branch which leaves the body of one is rejected. Such a body is a
// Dart closure, and a Dart closure cannot leave a loop outside itself.
func TestRangeFuncBranch(t *testing.T) {
	const head = `package a

func seq(yield func(int) bool) {}

func f() int {
`
	acceptFile(t, "a plain loop", head+`	for v := range seq {
		_ = v
	}
	return 0
}
`)
	acceptFile(t, "a break of the loop", head+`	for v := range seq {
		if v == 1 {
			break
		}
	}
	return 0
}
`)
	acceptFile(t, "a continue of the loop", head+`	for v := range seq {
		if v == 1 {
			continue
		}
	}
	return 0
}
`)
	acceptFile(t, "a break of a nested loop", head+`	for range seq {
		for i := 0; i < 2; i++ {
			break
		}
	}
	return 0
}
`)
}

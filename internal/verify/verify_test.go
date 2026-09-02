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
	return verify.Package(fset, []*ast.File{file}, info, nil)
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

import "encoding/gob"

var _ = gob.Register
`)
	acceptFile(t, "a blank import of unsafe", `package a

import _ "unsafe"
`)
	accept(t, "no import", "var _ = 1\n")
}

func TestRecover(t *testing.T) {
	accept(t, "call", `func f() { _ = recover() }`)
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
	accept(t, "int field", `func f() { var p point; _ = &p.x }`)
	accept(t, "int element", `func f(s []int) { _ = &s[0] }`)
	accept(t, "array element", `func f() { var a [2]int; _ = &a[0] }`)
	accept(t, "an element of a call result",
		`func g() []int { return nil }

func f() { _ = &g()[0] }`)
	accept(t, "a composite literal", `func f() { _ = &[]int{1} }`)
	reject(t, "a value behind a pointer", "the address of int",
		`func f(p *int) { _ = &*p }`)
}

func TestNew(t *testing.T) {
	accept(t, "struct type", `func f() { _ = new(point) }`)
	accept(t, "int type", `func f() { _ = new(int) }`)
	accept(t, "expression", `func f() { _ = new(1 + 2) }`)
	accept(t, "slice type", `func f() { _ = new([]int) }`)
}

func TestComplex(t *testing.T) {
	accept(t, "variable", `func f() { var c complex128; _ = c }`)
	accept(t, "builtin", `func f() { _ = complex(1.0, 2.0) }`)
	accept(t, "real", `func f(c complex64) { _ = real(c) }`)
	accept(t, "float", `func f() { var x float64; _ = x }`)
}

func TestReportsEachRejection(t *testing.T) {
	err := run(t, head+`func f(p *int, q *string) {
	_ = &*p
	_ = &*q
}`)
	if err == nil {
		t.Fatal("got no rejection, want 2")
	}
	if got := strings.Count(err.Error(), "a.go:"); got != 2 {
		t.Errorf("got %d rejections, want 2:\n%v", got, err)
	}
}

// TestAddressOfPackageVariable proves that stage 2 accepts the address of an
// unexported package variable and rejects the address of an exported one. Such
// a variable holds a box, and the box does not cross a package boundary.
func TestAddressOfPackageVariable(t *testing.T) {
	acceptFile(t, "an unexported package variable", `package a

var key int

func Key() *int { return &key }
`)
	rejectFile(t, "an exported package variable", "exported package variable", `package a

var Key int

func Ref() *int { return &Key }
`)
	acceptFile(t, "an exported struct variable", `package a

type T struct{ n int }

var Value T

func Ref() *T { return &Value }
`)
	acceptFile(t, "an exported variable of a main package", `package main

var Key int

func main() { ref(&Key) }

func ref(p *int) { *p = 1 }
`)
	acceptFile(t, "an exported variable of a test package", `package a_test

var Sink int

func Ref() *int { return &Sink }
`)
}

// TestAnonStructCrossesAPackage proves that stage 2 rejects an unnamed struct
// type with an unexported field in an exported declaration. Go gives such a
// field the identity of its package, and Dart holds a name with a leading
// underscore private to 1 library, thus 2 packages give 2 classes.
func TestAnonStructCrossesAPackage(t *testing.T) {
	const want = "unnamed struct type with an unexported field"
	rejectFile(t, "an exported result", want, `package a

func Make() struct{ n int } { return struct{ n int }{n: 1} }
`)
	rejectFile(t, "an exported parameter", want, `package a

func Take(v struct{ n int }) int { return v.n }
`)
	rejectFile(t, "an exported variable", want, `package a

var Value struct{ n int }
`)
	rejectFile(t, "an exported field", want, `package a

type Box struct {
	Inner struct{ n int }
}
`)
	rejectFile(t, "a slice of an unnamed struct", want, `package a

func List() []struct{ n int } { return nil }
`)
	rejectFile(t, "an exported method", want, `package a

type Box struct{ n int }

func (b Box) Get() struct{ n int } { return struct{ n int }{n: b.n} }
`)
	rejectFile(t, "a nested unnamed struct", want, `package a

func Make() struct{ Inner struct{ n int } } {
	return struct{ Inner struct{ n int } }{}
}
`)
	acceptFile(t, "an exported field of a shape", `package a

func Make() struct{ N int } { return struct{ N int }{N: 1} }
`)
	acceptFile(t, "an exported variable of a shape", `package a

var Value struct{ N int }
`)
	acceptFile(t, "an empty shape", `package a

func Done() <-chan struct{} { return nil }
`)
	acceptFile(t, "an unexported function", `package a

func make1() struct{ n int } { return struct{ n int }{n: 1} }
`)
	acceptFile(t, "a local variable", `package a

func F() int {
	v := struct{ n int }{n: 1}
	return v.n
}
`)
	acceptFile(t, "an unexported field", `package a

type Box struct {
	inner struct{ n int }
}
`)
	acceptFile(t, "an unexported method", `package a

type Box struct{ n int }

func (b Box) get() struct{ n int } { return struct{ n int }{n: b.n} }
`)
	acceptFile(t, "a method of an unexported type", `package a

type box struct{ n int }

func (b box) Get() struct{ n int } { return struct{ n int }{n: b.n} }
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

// TestTypeParamForm proves that an operation on a type parameter with more than
// 1 Dart form reaches a rejection, and that a type set of integer types stays
// inside the subset.
func TestTypeParamForm(t *testing.T) {
	const head = `package a

type mixType interface{ ~int | ~string }

type floatType interface{ ~float32 | ~float64 }

type intType interface {
	~int | ~int8 | ~uint8 | ~uint64
}

type plainType interface{ ~int8 | ~int16 }

type wideType interface{ ~int | ~float64 }

`
	rejectFile(t, "the operator + on 2 Dart types", "more than 1 Dart type",
		head+"func f[T mixType](a, b T) T { return a + b }\n")
	rejectFile(t, "the operator + on 2 float widths", "a mask of its width",
		head+"func f[T floatType](a, b T) T { return a + b }\n")
	rejectFile(t, "the operator += on 2 float widths", "a mask of its width",
		head+"func f[T floatType](a, b T) T { a += b; return a }\n")
	rejectFile(t, "the operator ++ on 2 float widths", "a mask of its width",
		head+"func f[T floatType](a T) T { a++; return a }\n")
	rejectFile(t, "the unary - on 2 float widths", "a mask of its width",
		head+"func f[T floatType](a T) T { return -a }\n")
	rejectFile(t, "a conversion to 2 float widths", "more than 1 Dart type",
		head+"func f[T floatType](a int) T { return T(a) }\n")

	rejectFile(t, "a division of mixed sign", "an unsigned type of 64 bits",
		head+"func f[T intType](a, b T) T { return a / b }\n")
	rejectFile(t, "a remainder of mixed sign", "an unsigned type of 64 bits",
		head+"func f[T intType](a, b T) T { return a % b }\n")
	rejectFile(t, "a right shift of mixed sign", "an unsigned type of 64 bits",
		head+"func f[T intType](a T) T { return a >> 2 }\n")
	rejectFile(t, "a shift assignment of mixed sign", "an unsigned type of 64 bits",
		head+"func f[T intType](a T) T { a >>= 2; return a }\n")
	rejectFile(t, "a conversion of mixed sign to a float", "an unsigned type of 64 bits",
		head+"func f[T intType](a T) float64 { return float64(a) }\n")
	rejectFile(t, "the builtin println of mixed sign", "an unsigned type of 64 bits",
		head+"func f[T intType](a T) { println(a) }\n")

	acceptFile(t, "the operator + on integer types",
		head+"func f[T intType](a, b T) T { return a + b }\n")
	acceptFile(t, "a shift left on integer types",
		head+"func f[T intType](a T) T { return a << 2 }\n")
	acceptFile(t, "a conversion between integer types",
		head+"func f[T intType](a T) T { return T(uint64(a) + 1) }\n")
	acceptFile(t, "a comparison of integer types",
		head+"func f[T intType](a, b T) bool { return a < b }\n")
	acceptFile(t, "a division of 1 sign form",
		head+"func f[T plainType](a, b T) T { return a / b }\n")
	acceptFile(t, "a comparison of 2 Dart types",
		head+"func f[T mixType](a, b T) bool { return a < b }\n")
	acceptFile(t, "the operator + on an int and a float64",
		head+"func f[T wideType](a, b T) T { return a + b }\n")
	acceptFile(t, "a conversion of 2 Dart types to an interface",
		head+"func f[T mixType](a T) any { return any(a) }\n")

	rejectFile(t, "a division of an int and a float64", "the Dart operator / always gives a double",
		head+"func f[T wideType](a, b T) T { return a / b }\n")
}

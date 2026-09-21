package verify_test

import (
	"go/ast"
	"go/constant"
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
	return runFiles(t, map[string]string{"a.go": src})
}

func runFiles(t *testing.T, sources map[string]string) error {
	t.Helper()
	fset := token.NewFileSet()
	// The comments reach the tree, because a directive of gart stands in a
	// comment. The loader of stage 1 reads them the same way.
	var files []*ast.File
	for name, src := range sources {
		file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		if err != nil {
			t.Fatalf("the source does not parse: %v", err)
		}
		files = append(files, file)
	}
	info := &types.Info{
		Types:     map[ast.Expr]types.TypeAndValue{},
		Defs:      map[*ast.Ident]types.Object{},
		Uses:      map[*ast.Ident]types.Object{},
		Implicits: map[ast.Node]types.Object{},
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("example.com/a", fset, files, info); err != nil {
		t.Fatalf("the source does not type check: %v", err)
	}
	return verify.Package(fset, files, info, nil, nil)
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

var _ = unsafe.Pointer(uintptr(1))
`)
	rejectFile(t, "curated surface", "outside the supported surface", `package a

import "plugin"

var _ = plugin.Open
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

func TestBodylessConstantBranches(t *testing.T) {
	const prefix = "package a\nfunc external()\n"
	for name, body := range map[string]string{
		"false if":       "if false { external() }",
		"true else":      "if true {} else { external() }",
		"false loop":     "for ; false; external() { external() }",
		"switch number":  "switch 1 { case 2: external(); case 1: }",
		"switch string":  "switch `a` { case `b`: external() }",
		"switch complex": "switch 1i { case -1i: external() }",
		"switch default": "switch 1 { case 2: external(); default: }",
		"dead default":   "switch 1 { default: external(); case 1: }",
		"fallthrough":    "switch 1 { case 1: fallthrough; case 2:; case 3: external() }",
	} {
		acceptFile(t, name, prefix+"func f() { "+body+" }")
	}
	for name, body := range map[string]string{
		"true if":             "if true { external() }",
		"false else":          "if false {} else { external() }",
		"if initializer":      "if external(); false {}",
		"loop initializer":    "for external(); false; {}",
		"live default":        "switch 1 { default: external(); case 2: }",
		"live fallthrough":    "switch 1 { case 1: fallthrough; case 2: external() }",
		"labeled fallthrough": "switch 1 { case 1: goto L; L: fallthrough; case 2: external() }",
		"trailing empty":      "switch 1 { case 1: fallthrough; ; case 2: external() }",
		"unknown tag":         "switch x { case 1: external() }",
		"unknown case":        "switch 1 { case x: external() }",
	} {
		rejectFile(t, name, "holds no body", prefix+"func f(x int) { "+body+" }")
	}
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
	accept(t, "a value behind a pointer",
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
	err := run(t, `package main
import "unsafe"
import "plugin"
func main() { _ = unsafe.Pointer(uintptr(1)); _ = plugin.Open }`)
	if err == nil {
		t.Fatal("got no rejection, want 2")
	}
	if got := strings.Count(err.Error(), "a.go:"); got != 2 {
		t.Errorf("got %d rejections, want 2:\n%v", got, err)
	}
}

func TestAddressOfPackageVariable(t *testing.T) {
	acceptFile(t, "an unexported package variable", `package a

var key int

func Key() *int { return &key }
`)
	acceptFile(t, "an exported package variable", `package a

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

func TestAnonStructCrossesAPackage(t *testing.T) {
	acceptFile(t, "an exported table of an external test package", `package a_test

var Cases = []struct{ in, out string }{{"a", "b"}}
`)
	acceptFile(t, "an exported variable of package main", `package main

var Value struct{ n int }

func main() {}
`)
	acceptFile(t, "an exported result", `package a

func Make() struct{ n int } { return struct{ n int }{n: 1} }
`)
	acceptFile(t, "an exported parameter", `package a

func Take(v struct{ n int }) int { return v.n }
`)
	acceptFile(t, "an exported variable", `package a

var Value struct{ n int }
`)
	acceptFile(t, "an exported field", `package a

type Box struct {
	Inner struct{ n int }
}
`)
	acceptFile(t, "a slice of an unnamed struct", `package a

func List() []struct{ n int } { return nil }
`)
	acceptFile(t, "an exported method", `package a

type Box struct{ n int }

func (b Box) Get() struct{ n int } { return struct{ n int }{n: b.n} }
`)
	acceptFile(t, "a nested unnamed struct", `package a

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

func TestTypeParamForm(t *testing.T) {
	const head = `package a

type mixType interface{ ~int | ~string }

type floatType interface{ ~float32 | ~float64 }

type intType interface {
	~int | ~int8 | ~uint8 | ~uint64
}

type plainType interface{ ~int8 | ~int16 }

type wideType interface{ ~int | ~float64 }

type narrowMix interface{ ~int | ~float32 }

`
	acceptFile(t, "the operator + on numbers and strings",
		head+"func f[T mixType](a, b T) T { return a + b }\n")
	acceptFile(t, "the operator += on numbers and strings",
		head+"func f[T mixType](a, b T) T { a += b; return a }\n")
	// A type set of float types gives the Dart type double, and the
	// dictionary of the type argument holds the mask of the width: float32
	// rounds to 32 bits, and float64 holds the identity. Thus each operator
	// of a float applies.
	acceptFile(t, "the operator + on 2 float widths",
		head+"func f[T floatType](a, b T) T { return a + b }\n")
	acceptFile(t, "the operator += on 2 float widths",
		head+"func f[T floatType](a, b T) T { a += b; return a }\n")
	acceptFile(t, "the operator ++ on 2 float widths",
		head+"func f[T floatType](a T) T { a++; return a }\n")
	acceptFile(t, "the unary - on 2 float widths",
		head+"func f[T floatType](a T) T { return -a }\n")
	acceptFile(t, "a division on 2 float widths",
		head+"func f[T floatType](a, b T) T { return a / b }\n")
	acceptFile(t, "a conversion to 2 float widths",
		head+"func f[T floatType](a int) T { return T(a) }\n")
	// A set that mixes an integer type with a float type holds the bound num,
	// and the emitter casts each result to the parameter in front of the
	// mask, thus a narrow term keeps each operator too.
	acceptFile(t, "the operator + on an int and a narrow float",
		head+"func f[T narrowMix](a, b T) T { return a + b }\n")
	acceptFile(t, "the unary - on an int and a narrow float",
		head+"func f[T narrowMix](a T) T { return -a }\n")
	acceptFile(t, "a division of an int and a narrow float",
		head+"func f[T narrowMix](a, b T) T { return a / b }\n")

	acceptFile(t, "a division of mixed sign",
		head+"func f[T intType](a, b T) T { return a / b }\n")
	acceptFile(t, "a remainder of mixed sign",
		head+"func f[T intType](a, b T) T { return a % b }\n")
	acceptFile(t, "a right shift of mixed sign",
		head+"func f[T intType](a T) T { return a >> 2 }\n")
	acceptFile(t, "a shift assignment of mixed sign",
		head+"func f[T intType](a T) T { a >>= 2; return a }\n")
	acceptFile(t, "a conversion of mixed sign to a float",
		head+"func f[T intType](a T) float64 { return float64(a) }\n")
	acceptFile(t, "the builtin println of mixed sign",
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

	acceptFile(t, "a division of an int and a float64",
		head+"func f[T wideType](a, b T) T { return a / b }\n")
}

// TestDartDecl proves the value boundary of an external Dart declaration in
// both directions.
//
// A declaration of the runtime library of gart carries each type of the
// subset. The compiler writes the Dart form of each Go type, and the runtime
// function names that form in its own signature, thus the Dart compiler proves
// the pair.
//
// Each other library holds the 5 forms of the boundary: bool, int, float64,
// string, and dart.Object. A foreign signature of Object? or of dynamic takes
// a wrong value with no report, thus the verifier names the position of the
// parameter or of the result.
func TestDartDecl(t *testing.T) {
	const runtime = "package:goruntime/goruntime.dart"
	const other = "dart:convert"

	acceptFile(t, "the runtime library carries a slice", `package a

//gart:dart `+runtime+` $jsonMarshal
func jsonFastMarshal(v any) []byte
`)
	acceptFile(t, "the runtime library carries a struct", `package a

type point struct{ x, y int }

//gart:dart `+runtime+` $shift
func shift(p point, by []int) map[string]point
`)
	acceptFile(t, "another library carries the 4 basic forms", `package a

//gart:dart `+other+` encode
func encode(flag bool, count int, ratio float64, name string) string
`)
	rejectFile(t, "another library rejects a slice parameter",
		"the parameter v of jsonFastMarshal holds the type any", `package a

//gart:dart `+other+` jsonEncode
func jsonFastMarshal(v any) []byte
`)
	rejectFile(t, "another library rejects a slice result",
		"the result of jsonFastMarshal holds the type []byte", `package a

//gart:dart `+other+` jsonEncode
func jsonFastMarshal(name string) []byte
`)
	rejectFile(t, "the runtime library gives 1 value",
		"carries //gart:dart and 2 results", `package a

//gart:dart `+runtime+` $split
func split(name string) ([]byte, bool)
`)
	rejectFile(t, "the runtime library takes no variadic parameter",
		"carries //gart:dart and a variadic parameter", `package a

//gart:dart `+runtime+` $join
func join(parts ...string) string
`)
	rejectFile(t, "only a function carries the directive",
		"carries //gart:dart, and only a function carries it", `package a

type point struct{ x, y int }

//gart:dart `+runtime+` $shift
func (p point) shift() int
`)
}

func TestUnsafeLayout(t *testing.T) {
	acceptFile(t, "metadata", `package a
import "unsafe"
var x struct { A byte; B int64 }
const Size = unsafe.Sizeof(x)
const Align = unsafe.Alignof(x)
const Offset = unsafe.Offsetof(x.B)
const PointerSize = unsafe.Sizeof(unsafe.Pointer(nil))
`)
	acceptFile(t, "alias", `package a
import u "unsafe"
const Size = u.Sizeof(1)
`)
	acceptFile(t, "dot", `package a
import . "unsafe"
const Size = Sizeof(1)
`)
	acceptFile(t, "dynamic metadata", `package a
import "unsafe"
func f[T any](x T) uintptr { return unsafe.Sizeof(x) }
`)
	rejectFile(t, "mixed operations", "rejects the package unsafe", `package a
import "unsafe"
const Size = unsafe.Sizeof(1)
var Pointer = unsafe.Pointer(uintptr(1))
`)
}

func TestNilUnsafePointer(t *testing.T) {
	const prefix = "package a\nimport \"unsafe\"\n"
	for name, body := range map[string]string{
		"declaration": "var p unsafe.Pointer",
		"nil":         "var p = unsafe.Pointer(nil)",
		"zero":        "func f() unsafe.Pointer { return unsafe.Pointer(uintptr(0)) }",
		"named":       "type P unsafe.Pointer; var p P; var q = unsafe.Pointer(p); var _ = any(q)",
	} {
		acceptFile(t, name, prefix+body)
	}
	for name, body := range map[string]string{
		"nonzero":               "var p = unsafe.Pointer(uintptr(1))",
		"variable":              "func f(n uintptr) unsafe.Pointer { return unsafe.Pointer(n) }",
		"address":               "var n int; var p = unsafe.Pointer(&n)",
		"integer":               "func f(p unsafe.Pointer) uintptr { return uintptr(p) }",
		"typed pointer":         "func f(p unsafe.Pointer) *int { return (*int)(p) }",
		"arithmetic":            "func f(p unsafe.Pointer) unsafe.Pointer { return unsafe.Add(p, 1) }",
		"generic address":       "func f[T ~unsafe.Pointer](n uintptr) T { return T(n) }",
		"mixed generic address": "func f[T ~unsafe.Pointer | ~uintptr](n uintptr) T { return T(n) }",
	} {
		rejectFile(t, name, "rejects the package unsafe", prefix+body)
	}
}

func TestUnsafePointerAcrossFiles(t *testing.T) {
	for name, body := range map[string]string{
		"address":       "func f(n uintptr) P { return P(n) }",
		"integer":       "func f(p P) uintptr { return uintptr(p) }",
		"typed pointer": "func f(p P) *int { return (*int)(p) }",
	} {
		t.Run(name, func(t *testing.T) {
			err := runFiles(t, map[string]string{
				"a.go": "package a\nimport \"unsafe\"\ntype P unsafe.Pointer",
				"b.go": "package a\n" + body,
			})
			if err == nil || !strings.Contains(err.Error(), "b.go:") || !strings.Contains(err.Error(), "unsafe conversion") {
				t.Fatalf("expected conversion rejection in b.go, got %v", err)
			}
		})
	}
}

func TestChannelElementSize(t *testing.T) {
	accept(t, "largest element", "var c chan [65535]byte")
	accept(t, "slice element", "var c chan []byte")
	accept(t, "generic slice element", "func f[T any]() { var c chan []T; _ = c }")
	accept(t, "generic local element", "func f[T any]() { type elem struct{ value T }; var c chan elem; _ = c }")
	reject(t, "element boundary", "channel element type too large", "var c chan [65536]byte")
	reject(t, "padded element", "channel element type too large", "var c chan struct { x [65529]byte; y uint64 }")
}

func TestArrayAddressSpace(t *testing.T) {
	accept(t, "below boundary", "type T [(1<<50)-1]byte")
	accept(t, "zero size", "type T [1<<50]struct{}")
	accept(t, "generic element", "type T[P any] [1<<50]P")
	reject(t, "boundary", "larger than address space", "type T [1<<50]byte")
	reject(t, "nested array", "larger than address space", "type T [2][1<<49]byte")
	reject(t, "overflow", "larger than address space", "type T [1<<62]uint64")
}

func TestMapElementSize(t *testing.T) {
	accept(t, "below boundary", "type T [(1<<31)-1]byte; var m map[int]T")
	accept(t, "generic element", "type M[T any] map[int]T")
	reject(t, "element boundary", "map element type too large", "type T [1<<31]byte; var m map[int]T")
	reject(t, "key boundary", "map key type too large", "type T [1<<31]byte; var m map[T]int")
}

type resolvedImporter struct{ pkg *types.Package }

func (i resolvedImporter) Import(string) (*types.Package, error) { return i.pkg, nil }

func TestResolvedImportPaths(t *testing.T) {
	for _, path := range []string{"vendor/example.com/dep", "example.com/dep", "vendor/golang.org/x/net/http/httpguts"} {
		for _, form := range []struct{ name, value string }{{"", "dep.Value"}, {"named", "named.Value"}, {".", "Value"}, {"_", "1"}} {
			t.Run(path+"/"+form.name, func(t *testing.T) {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, "import.go", "package a\nimport "+form.name+" \"example.com/dep\"\nvar _ = "+form.value, 0)
				if err != nil {
					t.Fatal(err)
				}
				pkg := types.NewPackage(path, "dep")
				pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, "Value", types.Typ[types.Int], constant.MakeInt64(1)))
				pkg.MarkComplete()
				info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Implicits: map[ast.Node]types.Object{}}
				config := types.Config{Importer: resolvedImporter{pkg}}
				if _, err := config.Check("example.com/a", fset, []*ast.File{file}, info); err != nil {
					t.Fatal(err)
				}
				err = verify.Package(fset, []*ast.File{file}, info, nil, nil)
				if path == "vendor/example.com/dep" {
					if err == nil || !strings.Contains(err.Error(), "import.go:2:") || !strings.Contains(err.Error(), path) {
						t.Fatalf("expected resolved import rejection, got %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

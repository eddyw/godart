package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"testing"
)

func apiScope(t *testing.T, source string) *types.Scope {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "api.go", "package api\n"+source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := new(types.Config).Check("example.com/api", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg.Scope()
}

func TestAPIContracts(t *testing.T) {
	native := apiScope(t, `
type Type interface { Fields() int; Methods() string }
type Value struct { Public int; private int }
func (Value) Read() int { return 0 }
func Apply(input int) (result string) { return "" }
func Generic[T comparable](input T) T { return input }
const Limit = 3
`)
	gart := apiScope(t, `
type Type interface { Fields() string }
type Value struct { Public string; hidden bool }
func (*Value) Read() int { return 0 }
func Apply(x int) string { return "" }
func Generic[T comparable](x T) T { return x }
const Limit = 4
`)
	if !slices.Contains(names(native), "Type.Methods") {
		t.Fatal("interface method absent")
	}
	report := compareAPI("api", native, gart)
	got := map[string]string{}
	for _, symbol := range report.Symbols {
		got[symbol.Name] = symbol.Status
	}
	for name, want := range map[string]string{
		"Type.Methods": "missing", "Type.Fields": "different", "Value": "match",
		"Value.Public": "different", "Value.Read": "different", "Apply": "match", "Limit": "different", "Generic": "match",
	} {
		if got[name] != want {
			t.Errorf("%s: got %s, want %s", name, got[name], want)
		}
	}
}

func TestSignatureParameterNames(t *testing.T) {
	native := apiScope(t, `
func All[Map ~map[K]V, K comparable, V any](m Map) func(yield func(key K, value V) bool) { return nil }
func Callback(f func(value int) (result string)) {}
func Variadic(f ...func(value int)) {}
func Changed[T ~int | ~string](value T) {}
func Distinct[A, B any](a A, b B) A { return a }
`)
	gart := apiScope(t, `
func All[M ~map[A]B, A comparable, B any](m M) func(func(A, B) bool) { return nil }
func Callback(f func(int) string) {}
func Variadic(f ...func(int)) {}
func Changed[T ~int | ~float64](value T) {}
func Distinct[A, B any](a A, b B) B { return b }
`)
	for _, symbol := range compareAPI("api", native, gart).Symbols {
		want := "match"
		if symbol.Name == "Changed" || symbol.Name == "Distinct" {
			want = "different"
		}
		if symbol.Status != want {
			t.Errorf("%s: got %s, want %s\nnative: %s\ngart: %s", symbol.Name, symbol.Status, want, symbol.Native, symbol.Gart)
		}
	}
}

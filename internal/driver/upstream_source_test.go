package driver_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"testing"
)

func TestSelectedUpstreamMethods(t *testing.T) {
	const source = `package sample

type Plain int
func (Plain) Value() int { return 1 }
func (*Plain) pointer() int { return 2 }

type Generic[T any] struct{ value T }
func (g Generic[T]) Value() T { return g.value }
func (g *Generic[T]) pointer() T { return g.value }

type Pair[A, B any] struct{ first A; second B }
func (p Pair[A, B]) Values() (A, B) { return p.first, p.second }
func (p *Pair[A, B]) pointer() (A, B) { return p.first, p.second }

type Unselected struct{}
func (Unselected) Other() {}

func verify() {
	var _ interface { Value() int; pointer() int } = (*Plain)(nil)
	var _ interface { Value() string; pointer() string } = (*Generic[string])(nil)
	var _ interface { Values() (int, string); pointer() (int, string) } = (*Pair[int, string])(nil)
}
`
	path := filepath.Join(t.TempDir(), "fixture.go")
	writeFile(t, path, source)
	selected := selectedUpstreamSource(t, path, []string{"Plain", "Generic", "Pair", "verify"})
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, selected, 0)
	if err != nil {
		t.Fatal(err)
	}
	var config types.Config
	pkg, err := config.Check("example.com/sample", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Scope().Lookup("Unselected") != nil {
		t.Fatal("the adapter retained an unrelated type")
	}
}

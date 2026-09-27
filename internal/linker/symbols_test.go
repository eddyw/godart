package linker

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/analyze"
	"golang.org/x/tools/go/packages"
)

type packageImporter map[string]*types.Package

func (p packageImporter) Import(path string) (*types.Package, error) {
	return p[path], nil
}

func TestStandardLibraryFunctionAlias(t *testing.T) {
	for _, test := range []struct {
		name               string
		mismatch, excluded bool
		want               string
	}{
		{name: "same-signature"},
		{name: "different-signature", mismatch: true, want: "requires a target function with the same type"},
		{name: "excluded-target", excluded: true, want: "requires an imported target package"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			imports := packageImporter{"unsafe": types.Unsafe}
			parse := func(path, source string) *packages.Package {
				t.Helper()
				file, err := parser.ParseFile(fset, path+".go", source, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				info := &types.Info{
					Types: map[ast.Expr]types.TypeAndValue{},
					Defs:  map[*ast.Ident]types.Object{},
					Uses:  map[*ast.Ident]types.Object{},
				}
				conf := types.Config{Importer: imports}
				pkg, err := conf.Check(path, fset, []*ast.File{file}, info)
				if err != nil {
					t.Fatal(err)
				}
				imports[path] = pkg
				return &packages.Package{ID: path, PkgPath: path, Fset: fset, Types: pkg,
					TypesInfo: info, Syntax: []*ast.File{file}, Imports: map[string]*packages.Package{}}
			}
			provider := parse("provider", `package provider
func receive(ch chan int) int { return <-ch }
`)
			result := "int"
			if test.mismatch {
				result = "string"
			}
			consumer := parse("consumer", `package consumer
import (_ "provider"; _ "unsafe")
//go:linkname receive provider.receive
func receive(ch chan int) `+result+`
func Call(ch chan int) { receive(ch) }
func Value() func(chan int) `+result+` { return receive }
`)
			consumer.Imports["provider"] = provider
			pkgs := []*packages.Package{consumer}
			if !test.excluded {
				pkgs = append(pkgs, provider)
			}
			links, err := Resolve(pkgs)
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("got %v, want %q", err, test.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var call *ast.CallExpr
			ast.Inspect(consumer.Syntax[0], func(node ast.Node) bool {
				if expr, ok := node.(*ast.CallExpr); ok {
					call = expr
				}
				return true
			})
			target := provider.Types.Scope().Lookup("receive")
			uses := 0
			for id, obj := range consumer.TypesInfo.Uses {
				if id.Name == "receive" {
					uses++
					if obj != target {
						t.Fatal("the call or function value retains the bodyless alias")
					}
				}
			}
			if uses != 2 || !links.Targets[target] || len(links.Imports[consumer.Types]) != 1 {
				t.Fatal("the private target has incomplete references or visibility")
			}
			blocks := analyze.Module([]*packages.Package{provider, consumer})
			if !blocks.CallAsync(call) {
				t.Fatal("the linked call loses the target channel receive effect")
			}
		})
	}
}

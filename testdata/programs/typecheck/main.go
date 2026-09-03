// The program reads a Go source with go/parser and checks it with go/types.
// The source stands in a string, thus the program reads no file of the
// machine.
package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
)

const src = `package p

type Pair[K comparable, V any] struct {
	Key K
	Val V
}

func (p Pair[K, V]) String() string { return "pair" }

type Number interface {
	~int | ~float64
}

func Sum[T Number](xs []T) T {
	var total T
	for _, x := range xs {
		total += x
	}
	return total
}

const Limit = 1 << 20

func Build(n int) (Pair[string, int], error) {
	p := Pair[string, int]{Key: "a", Val: n * 2}
	f := Sum([]float64{1.5, 2.5})
	_ = f
	return p, nil
}
`

func main() {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		fmt.Println("parse:", err)
		return
	}
	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
	}
	conf := types.Config{}
	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		fmt.Println("check:", err)
		return
	}

	fmt.Println("package:", pkg.Name())
	names := pkg.Scope().Names()
	sort.Strings(names)
	for _, name := range names {
		obj := pkg.Scope().Lookup(name)
		fmt.Printf("%s %s = %s\n", objKind(obj), name, obj.Type())
	}

	limit := pkg.Scope().Lookup("Limit").(*types.Const)
	fmt.Println("Limit value:", limit.Val(), constant.Sign(limit.Val()))

	pair := pkg.Scope().Lookup("Pair").Type().(*types.Named)
	fmt.Println("Pair params:", pair.TypeParams().Len(),
		"methods:", pair.NumMethods())
	fields := pair.Underlying().(*types.Struct)
	for i := 0; i < fields.NumFields(); i++ {
		fmt.Println("  field", fields.Field(i).Name(), fields.Field(i).Type())
	}

	sum := pkg.Scope().Lookup("Sum").(*types.Func)
	sig := sum.Type().(*types.Signature)
	fmt.Println("Sum param:", sig.Params().At(0).Type(),
		"result:", sig.Results().At(0).Type())

	for _, expr := range []string{
		"Sum([]int{1, 2, 3})",
		"Build(3)",
		"Limit + 1",
		"Pair[string, int]{}",
		"[]byte(\"x\")",
	} {
		tv, err := types.Eval(fset, pkg, file.End(), expr)
		if err != nil {
			fmt.Printf("eval %s: %v\n", expr, err)
			continue
		}
		fmt.Printf("eval %s: %s value=%v\n", expr, tv.Type, tv.Value)
	}

	_, err = conf.Check("bad", fset, []*ast.File{mustParse(fset,
		"package bad\nfunc f() int { return \"x\" }\n")}, nil)
	fmt.Println("bad package:", err)

	fmt.Println("assignable:", types.AssignableTo(types.Typ[types.Int],
		types.Typ[types.Int64]))
	fmt.Println("identical:", types.Identical(types.Typ[types.String],
		types.Typ[types.String]))
	fmt.Println("universe error:", types.Universe.Lookup("error").Type())
}

func mustParse(fset *token.FileSet, src string) *ast.File {
	file, err := parser.ParseFile(fset, "bad.go", src, 0)
	if err != nil {
		panic(err)
	}
	return file
}

func objKind(obj types.Object) string {
	switch obj.(type) {
	case *types.Const:
		return "const"
	case *types.TypeName:
		return "type"
	case *types.Func:
		return "func"
	case *types.Var:
		return "var"
	}
	return "other"
}

// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2013 The Go Authors. All rights reserved.
// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in internal/load/LICENSE-Go.

package verify

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/eddyw/godart/internal/typeset"
)

// NoInterface holds method origins excluded by the fieldtrack experiment.
// Digest also invalidates descriptors of types that embed an imported receiver.
type NoInterface struct {
	Methods map[*types.Func]bool
	Digest  string
}

// CheckNoInterface supplements go/types, which does not read this compiler pragma.
// Call only when the selected SDK enables fieldtrack.
func CheckNoInterface(pkgs []*packages.Package) (NoInterface, error) {
	result := NoInterface{Methods: map[*types.Func]bool{}}
	names := map[string]bool{}
	var errs []error
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			// A blank line does not detach a compiler pragma from its declaration.
			for _, group := range file.Comments {
				for _, comment := range group.List {
					verb, _, _ := strings.Cut(comment.Text, " ")
					if verb != "//go:nointerface" {
						continue
					}
					var function *ast.FuncDecl
					if comment.Pos() > file.Name.End() {
						for _, decl := range file.Decls {
							if comment.Pos() < decl.End() {
								if comment.End() < decl.Pos() {
									function, _ = decl.(*ast.FuncDecl)
								}
								break
							}
						}
					}
					if function == nil {
						errs = append(errs, &Error{Pos: pkg.Fset.Position(comment.Pos() + 2), Msg: "misplaced compiler directive"})
						continue
					}
					if function.Recv == nil {
						continue
					}
					method := pkg.TypesInfo.Defs[function.Name].(*types.Func).Origin()
					result.Methods[method] = true
					names[method.FullName()] = true
				}
			}
		}
	}
	keys := make([]string, 0, len(names))
	for name := range names {
		keys = append(keys, name)
	}
	slices.Sort(keys)
	result.Digest = fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(keys, "\x00"))))
	for _, pkg := range pkgs {
		v := noInterfaceVerifier{pkg: pkg, methods: result.Methods}
		for _, file := range pkg.Syntax {
			v.walk(file, nil)
		}
		errs = append(errs, v.errs...)
	}
	return result, errors.Join(errs...)
}

type noInterfaceVerifier struct {
	pkg     *packages.Package
	methods map[*types.Func]bool
	errs    []error
}

func (v *noInterfaceVerifier) assignment(pos token.Pos, actual, want types.Type, constraint bool) {
	iface, ok := want.Underlying().(*types.Interface)
	if !ok || iface.NumMethods() == 0 {
		return
	}
	if _, ok := actual.Underlying().(*types.Interface); ok {
		return
	}
	set := types.NewMethodSet(actual)
	for i := 0; i < iface.NumMethods(); i++ {
		required := iface.Method(i)
		method := set.Lookup(required.Pkg(), required.Name())
		if method == nil || !v.methods[method.Obj().(*types.Func).Origin()] {
			continue
		}
		verb := "implement"
		if constraint {
			verb = "satisfy"
		}
		v.errs = append(v.errs, &Error{Pos: v.pkg.Fset.Position(pos), Msg: fmt.Sprintf("%s does not %s %s (%s method is marked 'nointerface')", types.TypeString(actual, types.RelativeTo(v.pkg.Types)), verb, types.TypeString(want, types.RelativeTo(v.pkg.Types)), required.Name())})
		return
	}
}

func (v *noInterfaceVerifier) expression(expr ast.Expr, want types.Type) {
	v.assignment(expr.Pos(), v.pkg.TypesInfo.TypeOf(expr), want, false)
}

func (v *noInterfaceVerifier) values(exprs []ast.Expr, destinations []types.Type) {
	if len(exprs) == 1 {
		if tuple, ok := v.pkg.TypesInfo.TypeOf(exprs[0]).(*types.Tuple); ok {
			for i, want := range destinations {
				v.assignment(exprs[0].Pos(), tuple.At(i).Type(), want, false)
			}
			return
		}
	}
	for i, expr := range exprs {
		v.expression(expr, destinations[i])
	}
}

func tupleTypes(tuple *types.Tuple) []types.Type {
	result := make([]types.Type, tuple.Len())
	for i := range result {
		result[i] = tuple.At(i).Type()
	}
	return result
}

func (v *noInterfaceVerifier) walk(node ast.Node, results *types.Tuple) {
	info := v.pkg.TypesInfo
	ast.Inspect(node, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			v.walk(n.Type, nil)
			if n.Recv != nil {
				v.walk(n.Recv, nil)
			}
			if n.Body != nil {
				v.walk(n.Body, info.Defs[n.Name].Type().(*types.Signature).Results())
			}
			return false
		case *ast.FuncLit:
			v.walk(n.Type, nil)
			v.walk(n.Body, info.TypeOf(n).(*types.Signature).Results())
			return false
		case *ast.Ident:
			if instance, ok := info.Instances[n]; ok {
				var params *types.TypeParamList
				switch original := info.Uses[n].Type().(type) {
				case *types.Signature:
					params = original.TypeParams()
				case *types.Named:
					params = original.TypeParams()
				case *types.Alias:
					params = original.TypeParams()
				}
				for i := 0; i < instance.TypeArgs.Len(); i++ {
					v.assignment(n.Pos(), instance.TypeArgs.At(i), params.At(i).Constraint(), true)
				}
			}
		case *ast.ValueSpec:
			if len(n.Values) > 0 {
				destinations := make([]types.Type, len(n.Names))
				for i, name := range n.Names {
					destinations[i] = info.Defs[name].Type()
				}
				v.values(n.Values, destinations)
			}
		case *ast.AssignStmt:
			if n.Tok == token.ASSIGN || n.Tok == token.DEFINE {
				destinations := make([]types.Type, len(n.Lhs))
				for i, lhs := range n.Lhs {
					destinations[i] = info.TypeOf(lhs)
				}
				// A blank identifier has no destination type.
				if len(n.Rhs) == 1 {
					if tuple, ok := info.TypeOf(n.Rhs[0]).(*types.Tuple); ok {
						for i, want := range destinations {
							if want != nil {
								v.assignment(n.Rhs[0].Pos(), tuple.At(i).Type(), want, false)
							}
						}
						break
					}
				}
				for i, rhs := range n.Rhs {
					if destinations[i] != nil {
						v.expression(rhs, destinations[i])
					}
				}
			}
		case *ast.ReturnStmt:
			if len(n.Results) > 0 {
				v.values(n.Results, tupleTypes(results))
			}
		case *ast.CallExpr:
			if info.Types[n.Fun].IsType() {
				if len(n.Args) > 0 {
					v.expression(n.Args[0], info.TypeOf(n.Fun))
				}
				break
			}
			var builtin *types.Builtin
			switch fun := ast.Unparen(n.Fun).(type) {
			case *ast.Ident:
				builtin, _ = info.Uses[fun].(*types.Builtin)
			case *ast.SelectorExpr:
				builtin, _ = info.Uses[fun.Sel].(*types.Builtin)
			}
			if builtin != nil {
				if builtin.Name() == "append" && !n.Ellipsis.IsValid() {
					elem := typeset.Core(info.TypeOf(n.Args[0])).(*types.Slice).Elem()
					for _, expr := range n.Args[1:] {
						v.expression(expr, elem)
					}
				}
				break
			}

			sig := typeset.Core(info.TypeOf(n.Fun)).(*types.Signature)
			destinations := tupleTypes(sig.Params())
			if sig.Variadic() && !n.Ellipsis.IsValid() {
				fixed := len(destinations) - 1
				elem := destinations[fixed].(*types.Slice).Elem()
				count := len(n.Args)
				if count == 1 {
					if tuple, ok := info.TypeOf(n.Args[0]).(*types.Tuple); ok {
						count = tuple.Len()
					}
				}
				destinations = destinations[:fixed]
				for len(destinations) < count {
					destinations = append(destinations, elem)
				}
			}
			v.values(n.Args, destinations)
		case *ast.SendStmt:
			v.expression(n.Value, typeset.Core(info.TypeOf(n.Chan)).(*types.Chan).Elem())
		case *ast.CompositeLit:
			core := typeset.Core(info.TypeOf(n))
			if pointer, ok := core.(*types.Pointer); ok {
				core = typeset.Core(pointer.Elem())
			}
			switch typ := core.(type) {
			case *types.Array:
				v.elements(n.Elts, typ.Elem())
			case *types.Slice:
				v.elements(n.Elts, typ.Elem())
			case *types.Map:
				for _, elt := range n.Elts {
					pair := elt.(*ast.KeyValueExpr)
					v.expression(pair.Key, typ.Key())
					v.expression(pair.Value, typ.Elem())
				}
			case *types.Struct:
				for i, elt := range n.Elts {
					if pair, ok := elt.(*ast.KeyValueExpr); ok {
						v.expression(pair.Value, info.Uses[pair.Key.(*ast.Ident)].Type())
					} else {
						v.expression(elt, typ.Field(i).Type())
					}
				}
			}
		case *ast.BinaryExpr:
			if n.Op == token.EQL || n.Op == token.NEQ {
				v.expression(n.X, info.TypeOf(n.Y))
				v.expression(n.Y, info.TypeOf(n.X))
			}
		case *ast.SwitchStmt:
			if n.Tag != nil {
				for _, clause := range n.Body.List {
					for _, expr := range clause.(*ast.CaseClause).List {
						v.expression(expr, info.TypeOf(n.Tag))
					}
				}
			}
		case *ast.TypeAssertExpr:
			if n.Type != nil {
				v.assignment(n.Pos(), info.TypeOf(n.Type), info.TypeOf(n.X), false)
			}
		case *ast.TypeSwitchStmt:
			var assertion *ast.TypeAssertExpr
			switch assign := n.Assign.(type) {
			case *ast.ExprStmt:
				assertion = assign.X.(*ast.TypeAssertExpr)
			case *ast.AssignStmt:
				assertion = assign.Rhs[0].(*ast.TypeAssertExpr)
			}
			for _, clause := range n.Body.List {
				for _, expr := range clause.(*ast.CaseClause).List {
					v.assignment(expr.Pos(), info.TypeOf(expr), info.TypeOf(assertion.X), false)
				}
			}
		case *ast.RangeStmt:
			if n.Tok == token.ASSIGN {
				values := noInterfaceRangeTypes(info.TypeOf(n.X))
				for i, lhs := range []ast.Expr{n.Key, n.Value} {
					if lhs != nil && info.TypeOf(lhs) != nil {
						v.assignment(lhs.Pos(), values[i], info.TypeOf(lhs), false)
					}
				}
			}
		}
		return true
	})
}

func (v *noInterfaceVerifier) elements(elements []ast.Expr, want types.Type) {
	for _, expr := range elements {
		if pair, ok := expr.(*ast.KeyValueExpr); ok {
			expr = pair.Value
		}
		v.expression(expr, want)
	}
}

func noInterfaceRangeTypes(t types.Type) []types.Type {
	core := typeset.Core(t)
	if pointer, ok := core.(*types.Pointer); ok {
		core = typeset.Core(pointer.Elem())
	}
	switch typ := core.(type) {
	case *types.Array:
		return []types.Type{types.Typ[types.Int], typ.Elem()}
	case *types.Slice:
		return []types.Type{types.Typ[types.Int], typ.Elem()}
	case *types.Map:
		return []types.Type{typ.Key(), typ.Elem()}
	case *types.Chan:
		return []types.Type{typ.Elem()}
	case *types.Signature:
		return tupleTypes(typeset.Core(typ.Params().At(0).Type()).(*types.Signature).Params())
	case *types.Basic:
		if typ.Info()&types.IsString != 0 {
			return []types.Type{types.Typ[types.Int], types.Typ[types.Rune]}
		}
	}
	return []types.Type{t}
}

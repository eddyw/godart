package analyze

import (
	"go/ast"
	"go/token"
	"go/types"
)

type interfaceOrigin struct {
	expr  ast.Expr
	info  *types.Info
	index int
	zero  bool
}

// An immutable interface keeps the concrete type of its initializer.
// Reassignment, an address escape, or an unknown factory result drops the proof.
func (a *analyzer) collectOrigins(f *ast.File) {
	info := a.pkg.TypesInfo
	keep := func(id *ast.Ident, values []ast.Expr, count, index int) {
		origin := interfaceOrigin{info: info, zero: len(values) == 0}
		if len(values) == count {
			origin.expr = values[index]
		} else if len(values) == 1 {
			origin.expr, origin.index = values[0], index
		}
		v, _ := info.Defs[id].(*types.Var)
		if v != nil && isInterface(v.Type()) {
			if _, parameter := types.Unalias(v.Type()).(*types.TypeParam); parameter {
				// A type parameter can have a non-interface zero value.
				origin.zero = false
			}
			a.interfaceOrigins[v] = origin
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if a.unevaluated(n) {
			return false
		}
		switch n := n.(type) {
		case *ast.ValueSpec:
			for i, id := range n.Names {
				keep(id, n.Values, len(n.Names), i)
			}
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok {
					continue
				}
				if n.Tok == token.DEFINE {
					keep(id, n.Rhs, len(n.Lhs), i)
				}
				if v, ok := info.Uses[id].(*types.Var); ok {
					a.originMutable[v] = true
				}
			}
		case *ast.UnaryExpr:
			if n.Op == token.AND {
				if id, ok := unparen(n.X).(*ast.Ident); ok {
					if v, ok := info.Uses[id].(*types.Var); ok {
						a.originMutable[v] = true
					}
				}
			}
		case *ast.RangeStmt:
			for _, x := range []ast.Expr{n.Key, n.Value} {
				if id, ok := x.(*ast.Ident); ok {
					if v, ok := info.Uses[id].(*types.Var); ok {
						a.originMutable[v] = true
					}
				}
			}
		case *ast.FuncDecl:
			fn, _ := info.Defs[n.Name].(*types.Func)
			if fn == nil || n.Body == nil {
				break
			}
			sig := fn.Type().(*types.Signature)
			for i := range sig.Results().Len() {
				result := sig.Results().At(i)
				if result.Name() != "" || !isInterface(result.Type()) {
					continue
				}
				var returns []interfaceOrigin
				ast.Inspect(n.Body, func(x ast.Node) bool {
					if _, ok := x.(*ast.FuncLit); ok {
						return false
					}
					if r, ok := x.(*ast.ReturnStmt); ok {
						origin := interfaceOrigin{info: info}
						if len(r.Results) == sig.Results().Len() {
							origin.expr = r.Results[i]
						} else if len(r.Results) == 1 {
							origin.expr, origin.index = r.Results[0], i
						}
						returns = append(returns, origin)
					}
					return true
				})
				a.interfaceReturns[retCell{fn, i}] = returns
			}
		}
		return true
	})
}

func (a *analyzer) concreteInterfaces(x ast.Expr, info *types.Info, seen map[types.Object]bool) ([]types.Type, bool) {
	return a.concreteOrigin(interfaceOrigin{expr: x, info: info}, seen)
}

func (a *analyzer) concreteOrigin(origin interfaceOrigin, seen map[types.Object]bool) ([]types.Type, bool) {
	x, info := origin.expr, origin.info
	if x == nil {
		return nil, origin.zero
	}
	t := info.TypeOf(x)
	if tuple, ok := t.(*types.Tuple); ok {
		t = tuple.At(origin.index).Type()
	}
	if t == nil {
		return nil, false
	}
	if !isInterface(t) {
		return []types.Type{t}, true
	}
	x = unparen(x)
	if id, ok := x.(*ast.Ident); ok {
		v, _ := info.Uses[id].(*types.Var)
		if v == nil || seen[v] || a.originMutable[v] {
			return nil, false
		}
		origin, ok := a.interfaceOrigins[v]
		if !ok {
			return nil, false
		}
		seen[v] = true
		defer delete(seen, v)
		return a.concreteOrigin(origin, seen)
	}
	call, ok := x.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	var fn *types.Func
	switch fun := unparen(call.Fun).(type) {
	case *ast.Ident:
		fn, _ = info.Uses[fun].(*types.Func)
	case *ast.SelectorExpr:
		if info.Selections[fun] == nil {
			fn, _ = info.Uses[fun.Sel].(*types.Func)
		}
	}
	if fn == nil || seen[fn] || len(a.interfaceReturns[retCell{fn, origin.index}]) == 0 {
		return nil, false
	}
	seen[fn] = true
	defer delete(seen, fn)
	var out []types.Type
	for _, origin := range a.interfaceReturns[retCell{fn, origin.index}] {
		types, ok := a.concreteOrigin(origin, seen)
		if !ok {
			return nil, false
		}
		out = append(out, types...)
	}
	return out, true
}

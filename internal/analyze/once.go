package analyze

import (
	"go/ast"
	"go/types"
)

func calledVariable(expr ast.Expr, info *types.Info) *types.Var {
	switch expr := unparen(expr).(type) {
	case *ast.Ident:
		variable, _ := info.Uses[expr].(*types.Var)
		return variable
	case *ast.SelectorExpr:
		if info.Selections[expr] == nil {
			variable, _ := info.Uses[expr.Sel].(*types.Var)
			return variable
		}
	}
	return nil
}

// Each Once factory owns its lock. A callback without suspension cannot leave
// that lock held for another goroutine. The caller also suppresses polling.
func (a *analyzer) recordOnce(name *ast.Ident, value ast.Expr) {
	variable, _ := a.pkg.TypesInfo.Defs[name].(*types.Var)
	if variable == nil || a.originMutable[variable] {
		return
	}
	call, ok := unparen(value).(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return
	}
	expr := unparen(call.Fun)
	switch indexed := expr.(type) {
	case *ast.IndexExpr:
		expr = unparen(indexed.X)
	case *ast.IndexListExpr:
		expr = unparen(indexed.X)
	}
	var object types.Object
	switch expr := expr.(type) {
	case *ast.Ident:
		object = a.pkg.TypesInfo.Uses[expr]
	case *ast.SelectorExpr:
		if a.pkg.TypesInfo.Selections[expr] == nil {
			object = a.pkg.TypesInfo.Uses[expr.Sel]
		}
	}
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil || function.Pkg().Path() != "sync" {
		return
	}
	switch function.Name() {
	case "OnceFunc", "OnceValue", "OnceValues":
		a.onceSources[variable] = a.flowExpr(call.Args[0])
	}
}

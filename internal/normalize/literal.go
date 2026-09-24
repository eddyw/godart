package normalize

import (
	"go/ast"
	"go/types"

	"github.com/eddyw/godart/internal/typeset"
)

func literalOrdinals(node ast.Node, info *types.Info) map[ast.Node]int {
	called := map[*ast.SelectorExpr]bool{}
	ast.Inspect(node, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := unparenExpr(call.Fun).(*ast.SelectorExpr); ok {
				called[selector] = true
			}
		}
		return true
	})
	out := map[ast.Node]int{}
	var walk func(ast.Node)
	walk = func(node ast.Node) {
		next, rangeNext := 0, 0
		var visit func(ast.Node) bool
		visit = func(node ast.Node) bool {
			if selector, ok := node.(*ast.SelectorExpr); ok {
				if selection := info.Selections[selector]; selection != nil && typeset.HoldsParam(selection.Recv()) {
					_, parameter := types.Unalias(selection.Recv()).(*types.TypeParam)
					expression := selection.Kind() == types.MethodExpr && !parameter
					bound := selection.Kind() == types.MethodVal && !called[selector]
					if expression || bound {
						next++
						out[selector] = next
					}
				}
			}
			if literal, ok := node.(*ast.FuncLit); ok {
				next++
				out[literal] = next
				walk(literal.Body)
				return false
			}
			if loop, ok := node.(*ast.RangeStmt); ok {
				if _, function := typeset.Core(info.TypeOf(loop.X)).(*types.Signature); function {
					ast.Inspect(loop.X, visit)
					rangeNext++
					out[loop] = rangeNext
					walk(loop.Body)
					return false
				}
			}
			return true
		}
		ast.Inspect(node, visit)
	}
	walk(node)
	return out
}

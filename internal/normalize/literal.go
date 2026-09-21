package normalize

import (
	"go/ast"
	"go/types"

	"github.com/eddyw/godart/internal/typeset"
)

func literalOrdinals(node ast.Node, info *types.Info) map[ast.Node]int {
	out := map[ast.Node]int{}
	var walk func(ast.Node)
	walk = func(node ast.Node) {
		next, rangeNext := 0, 0
		var visit func(ast.Node) bool
		visit = func(node ast.Node) bool {
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

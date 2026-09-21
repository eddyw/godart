package verify

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

func deadBodylessReads(files []*ast.File, info *types.Info) map[*ast.Ident]bool {
	dead := map[*ast.Ident]bool{}
	mark := func(node ast.Node) {
		ast.Inspect(node, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				dead[id] = true
			}
			return true
		})
	}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.IfStmt:
				if value := info.Types[n.Cond].Value; value != nil {
					if constant.BoolVal(value) {
						if n.Else != nil {
							mark(n.Else)
						}
					} else {
						mark(n.Body)
					}
				}
			case *ast.ForStmt:
				if value := info.Types[n.Cond].Value; value != nil && !constant.BoolVal(value) {
					mark(n.Body)
					if n.Post != nil {
						mark(n.Post)
					}
				}
			case *ast.SwitchStmt:
				selected, known := constantCase(n, info)
				if !known {
					break
				}
				fallthroughCase := false
				for i, statement := range n.Body.List {
					clause := statement.(*ast.CaseClause)
					if i != selected && !fallthroughCase {
						for _, statement := range clause.Body {
							mark(statement)
						}
						continue
					}
					fallthroughCase = false
					for lastIndex := len(clause.Body) - 1; lastIndex >= 0; lastIndex-- {
						lastStatement := clause.Body[lastIndex]
						if _, empty := lastStatement.(*ast.EmptyStmt); empty {
							continue
						}
						for {
							label, ok := lastStatement.(*ast.LabeledStmt)
							if !ok {
								break
							}
							lastStatement = label.Stmt
						}
						last, ok := lastStatement.(*ast.BranchStmt)
						fallthroughCase = ok && last.Tok == token.FALLTHROUGH
						break
					}
				}
			}
			return true
		})
	}
	return dead
}

func constantCase(statement *ast.SwitchStmt, info *types.Info) (int, bool) {
	tag := constant.MakeBool(true)
	if statement.Tag != nil {
		tag = info.Types[statement.Tag].Value
		if tag == nil {
			return 0, false
		}
	}
	selected, fallback := -1, -1
	for i, body := range statement.Body.List {
		clause := body.(*ast.CaseClause)
		if clause.List == nil {
			fallback = i
		}
		for _, expression := range clause.List {
			value := info.Types[expression].Value
			if value == nil {
				return 0, false
			}
			if selected < 0 && constant.Compare(tag, token.EQL, value) {
				selected = i
			}
		}
	}
	if selected < 0 {
		selected = fallback
	}
	return selected, true
}

package load

import (
	"bytes"
	"go/ast"
	"go/scanner"
	"go/token"
	"strings"
)

func misplacedDirectives(fset *token.FileSet, file *ast.File, src []byte) scanner.ErrorList {
	var errs scanner.ErrorList
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if !strings.HasPrefix(comment.Text, "//go:") {
				continue
			}
			pos := fset.PositionFor(comment.Pos(), false)
			line := bytes.LastIndexByte(src[:pos.Offset], '\n') + 1
			allowed := len(bytes.TrimSpace(src[line:pos.Offset])) == 0
			if allowed {
				verb, _, _ := strings.Cut(comment.Text, " ")
				switch verb {
				case "//go:build":
					allowed = comment.End() < file.Package
				case "//go:noescape", "//go:norace", "//go:nosplit", "//go:noinline",
					"//go:nocheckptr", "//go:systemstack", "//go:nowritebarrier",
					"//go:nowritebarrierrec", "//go:yeswritebarrierrec",
					"//go:cgo_unsafe_args", "//go:uintptrkeepalive",
					"//go:uintptrescapes", "//go:registerparams":
					allowed = false
					if comment.Pos() > file.Name.End() {
						for _, decl := range file.Decls {
							if comment.Pos() < decl.End() {
								_, function := decl.(*ast.FuncDecl)
								allowed = function && comment.End() < decl.Pos()
								break
							}
						}
					}
				}
			}
			if !allowed {
				errs.Add(fset.Position(comment.Pos()+2), "misplaced compiler directive")
			}
		}
	}
	return errs
}

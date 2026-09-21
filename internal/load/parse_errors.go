package load

import (
	"bytes"
	"go/ast"
	"go/scanner"
	"go/token"
	"os"

	"golang.org/x/tools/go/packages"

	syntax "github.com/eddyw/godart/internal/gosyntax"
)

func compilerParseErrors(pkg *packages.Package, sources map[string][]byte) (scanner.ErrorList, error) {
	if !pkg.IllTyped {
		return nil, nil
	}
	for _, err := range pkg.Errors {
		if err.Kind == packages.ParseError {
			return nil, nil
		}
	}
	var diagnostics scanner.ErrorList
	for _, file := range pkg.Syntax {
		needsParse := false
		ast.Inspect(file, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.BranchStmt, *ast.LabeledStmt, *ast.TypeSwitchStmt:
				needsParse = true
			}
			return !needsParse
		})
		if !needsParse {
			continue
		}
		name := pkg.Fset.PositionFor(file.Pos(), false).Filename
		source, ok := sources[name]
		if !ok {
			var err error
			source, err = os.ReadFile(name)
			if err != nil {
				return nil, err
			}
		}
		syntax.Parse(syntax.NewFileBase(name), bytes.NewReader(source), func(err error) {
			e := err.(syntax.Error)
			diagnostics.Add(token.Position{Filename: e.Pos.RelFilename(), Line: int(e.Pos.RelLine()), Column: int(e.Pos.RelCol())}, e.Msg)
		}, nil, syntax.CheckBranches)
	}
	return diagnostics, nil
}

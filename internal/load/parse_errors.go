package load

import (
	"bytes"
	"go/ast"
	"go/scanner"
	"go/token"
	"os"
	"strings"

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
	// A NUL in the import header prevents discovery from returning GoFiles.
	for _, err := range pkg.Errors {
		if err.Kind != packages.ListError || !strings.HasPrefix(err.Msg, "read ") || !strings.HasSuffix(err.Msg, ": unexpected NUL in input") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(err.Msg, "read "), ": unexpected NUL in input")
		source, ok := sources[name]
		if !ok {
			var err error
			source, err = os.ReadFile(name)
			if err != nil {
				return nil, err
			}
		}
		diagnostics = append(diagnostics, compilerSyntaxErrors(token.NewFileSet(), name, source, nil)...)
	}
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

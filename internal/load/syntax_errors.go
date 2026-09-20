// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-Go file.

package load

import (
	"bytes"
	"go/ast"
	"go/scanner"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"

	syntax "github.com/eddyw/godart/internal/gosyntax"
)

// Go stops before type analysis after a syntax error and reports only the
// first syntax error on each line. Other parser diagnostics permit analysis.
func compilerSyntaxErrors(fset *token.FileSet, name string, src []byte, file *ast.File) scanner.ErrorList {
	var diagnostics scanner.ErrorList
	var lastSyntax token.Position
	hasSyntaxError := false
	parsed, _ := syntax.Parse(syntax.NewFileBase(name), bytes.NewReader(src), func(err error) {
		e := err.(syntax.Error)
		pos := token.Position{
			Filename: e.Pos.RelFilename(),
			Line:     int(e.Pos.RelLine()),
			Column:   int(e.Pos.RelCol()),
		}
		if strings.HasPrefix(e.Msg, "syntax error") {
			hasSyntaxError = true
			if pos.Filename == lastSyntax.Filename && pos.Line == lastSyntax.Line {
				return
			}
			lastSyntax = pos
		}
		diagnostics.Add(pos, e.Msg)
	}, nil, syntax.CheckBranches)
	if !hasSyntaxError {
		// The public AST discards some invalid statements. Keep its parser
		// diagnostics unless a lexical error requires the compiler wording.
		lexicalError := false
		var tokens scanner.Scanner
		tokens.Init(token.NewFileSet().AddFile(name, -1, len(src)), src, func(token.Position, string) {
			lexicalError = true
		}, 0)
		for {
			_, kind, _ := tokens.Scan()
			if kind == token.EOF {
				break
			}
		}
		if !lexicalError {
			return nil
		}
	}
	if !hasSyntaxError && parsed != nil && file != nil {
		bad := make(map[token.Position]bool)
		syntax.Inspect(parsed, func(n syntax.Node) bool {
			if literal, ok := n.(*syntax.BasicLit); ok && literal.Bad {
				pos := literal.Pos()
				bad[token.Position{Filename: pos.RelFilename(), Line: int(pos.RelLine()), Column: int(pos.RelCol())}] = true
			}
			return true
		})
		astutil.Apply(file, func(c *astutil.Cursor) bool {
			if literal, ok := c.Node().(*ast.BasicLit); ok {
				pos := fset.Position(literal.Pos())
				pos.Offset = 0
				// Tags and import paths require *ast.BasicLit, not ast.Expr.
				if bad[pos] && c.Name() != "Tag" && c.Name() != "Path" {
					c.Replace(&ast.BadExpr{From: literal.Pos(), To: literal.End()})
				}
			}
			return true
		}, nil)
	}
	return diagnostics
}

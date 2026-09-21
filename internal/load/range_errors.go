// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-Go file.

package load

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	syntax "github.com/eddyw/godart/internal/gosyntax"
)

type rangeCut struct {
	start, end  int
	loop, extra token.Pos
}

// The public parser discards range statements with more than 2 variables.
// Retain the first 2 for type analysis, which determines the legal arity.
func repairRangeClauses(fset *token.FileSet, name string, src []byte, file *ast.File, parseErr error, extras *sync.Map) (*ast.File, error) {
	list, ok := parseErr.(scanner.ErrorList)
	if !ok || len(list) == 0 || file == nil {
		return file, parseErr
	}
	for _, err := range list {
		if err.Msg != "expected at most 2 expressions" {
			return file, parseErr
		}
	}
	parsed, err := syntax.Parse(syntax.NewFileBase(name), bytes.NewReader(src), nil, nil, syntax.CheckBranches)
	if err != nil {
		return file, parseErr
	}
	original := fset.File(file.Pos())
	offset := func(pos syntax.Pos) int {
		return original.Offset(original.LineStart(int(pos.Line()))) + int(pos.Col()) - 1
	}
	var cuts []rangeCut
	syntax.Inspect(parsed, func(n syntax.Node) bool {
		loop, ok := n.(*syntax.ForStmt)
		if !ok {
			return true
		}
		clause, ok := loop.Init.(*syntax.RangeClause)
		if !ok {
			return true
		}
		lhs, ok := clause.Lhs.(*syntax.ListExpr)
		if !ok || len(lhs.ElemList) < 3 {
			return true
		}
		cuts = append(cuts, rangeCut{
			start: offset(syntax.EndPos(lhs.ElemList[1])),
			end:   offset(syntax.EndPos(lhs.ElemList[len(lhs.ElemList)-1])),
			loop:  original.Pos(offset(loop.Pos())),
			extra: original.Pos(offset(syntax.StartPos(lhs.ElemList[2]))),
		})
		return true
	})
	if len(cuts) == 0 {
		return file, parseErr
	}
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].start < cuts[j].start })
	var compact []byte
	kept := cuts[:0]
	cursor := 0
	for _, cut := range cuts {
		if cut.start < cursor {
			continue
		}
		compact = append(compact, src[cursor:cut.start]...)
		cursor = cut.end
		kept = append(kept, cut)
	}
	cuts = kept
	compact = append(compact, src[cursor:]...)
	repaired, err := parser.ParseFile(fset, name, compact, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return file, parseErr
	}
	changed := fset.File(repaired.Pos())
	positionType := reflect.TypeFor[token.Pos]()
	// All AST token positions must refer to the original source, including
	// comments, multiline expressions, and positions after a removed span.
	ast.Inspect(repaired, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		node := reflect.ValueOf(n).Elem()
		for i := 0; i < node.NumField(); i++ {
			field := node.Field(i)
			if field.Type() != positionType {
				continue
			}
			pos := token.Pos(field.Int())
			if pos < token.Pos(changed.Base()) || pos > token.Pos(changed.Base()+changed.Size()) {
				continue
			}
			off := changed.Offset(pos)
			for _, cut := range cuts {
				if off < cut.start {
					break
				}
				off += cut.end - cut.start
			}
			field.SetInt(int64(original.Pos(off)))
		}
		return true
	})
	for _, cut := range cuts {
		extras.Store(cut.loop, cut.extra)
	}
	return repaired, nil
}

func rangeClauseErrors(pkg *packages.Package, extras *sync.Map) scanner.ErrorList {
	var diagnostics scanner.ErrorList
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			stmt, ok := n.(*ast.RangeStmt)
			if !ok {
				return true
			}
			extra, ok := extras.Load(stmt.For)
			if !ok {
				return true
			}
			typ := pkg.TypesInfo.TypeOf(stmt.X)
			if typ == nil || typ == types.Typ[types.Invalid] {
				return true
			}
			for _, err := range pkg.TypeErrors {
				if (err.Pos == stmt.X.Pos() || err.Pos == stmt.Key.Pos() || err.Pos == stmt.Value.Pos()) &&
					(strings.HasPrefix(err.Msg, "cannot range over ") || strings.HasPrefix(err.Msg, "range over ")) {
					return true
				}
			}
			diagnostics.Add(pkg.Fset.Position(extra.(token.Pos)), "range clause permits at most two iteration variables")
			return true
		})
	}
	return diagnostics
}

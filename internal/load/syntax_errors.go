// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-Go file.

package load

import (
	"bytes"
	"go/scanner"
	"go/token"
	"strings"

	syntax "github.com/eddyw/godart/internal/gosyntax"
)

// Go stops before type analysis after a syntax error and reports only the
// first syntax error on each line. Other parser diagnostics permit analysis.
func compilerSyntaxErrors(name string, src []byte) scanner.ErrorList {
	var diagnostics scanner.ErrorList
	var lastSyntax token.Position
	hasSyntaxError := false
	syntax.Parse(syntax.NewFileBase(name), bytes.NewReader(src), func(err error) {
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
		return nil
	}
	return diagnostics
}

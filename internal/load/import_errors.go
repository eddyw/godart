// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-Go file.

package load

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func compilerImportDiagnostic(pkg *packages.Package, diagnostic packages.Error) packages.Error {
	if diagnostic.Kind != packages.TypeError || !strings.HasPrefix(diagnostic.Msg, "could not import ") {
		return diagnostic
	}
	for _, file := range pkg.Syntax {
		for _, spec := range file.Imports {
			if pkg.Fset.Position(spec.Path.Pos()).String() != diagnostic.Pos {
				continue
			}
			name, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return diagnostic
			}
			var reason string
			switch {
			case strings.HasPrefix(name, "/"):
				reason = "import path cannot be absolute path"
			case name == ".", name == "..", strings.HasPrefix(name, "./"), strings.HasPrefix(name, "../"):
				return diagnostic
			default:
				if clean := path.Clean(name); clean != name {
					reason = fmt.Sprintf("non-canonical import path %q (should be %q)", name, clean)
				}
			}
			if reason != "" {
				diagnostic.Msg = fmt.Sprintf("could not import %s (%s)", name, reason)
			}
			return diagnostic
		}
	}
	return diagnostic
}

// The compiler already reports malformed import literals. Discovery and type
// analysis repeat that diagnostic at the literal start.
func redundantImportError(pkg *packages.Package, diagnostic packages.Error) bool {
	if diagnostic.Kind != packages.ListError && diagnostic.Kind != packages.TypeError {
		return false
	}
	for _, file := range pkg.Syntax {
		for _, spec := range file.Imports {
			if _, err := strconv.Unquote(spec.Path.Value); err == nil {
				continue
			}
			pos := pkg.Fset.Position(spec.Path.Pos()).String()
			if diagnostic.Pos == pos || diagnostic.Pos == filepath.Base(pos) {
				return true
			}
		}
	}
	return false
}

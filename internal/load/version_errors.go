// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-Go file.

package load

import (
	"fmt"
	"go/ast"
	"go/version"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

var versionErrorRx = regexp.MustCompile(`requires go[0-9]+\.[0-9]+ or later`)

func compilerVersionDiagnostic(pkg *packages.Package, diagnostic packages.Error) packages.Error {
	if diagnostic.Kind != packages.TypeError || !versionErrorRx.MatchString(diagnostic.Msg) {
		return diagnostic
	}
	for _, file := range pkg.Syntax {
		match := strings.HasPrefix(diagnostic.Pos, pkg.Fset.File(file.Pos()).Name()+":")
		if !match {
			ast.Inspect(file, func(node ast.Node) bool {
				if node != nil && pkg.Fset.Position(node.Pos()).String() == diagnostic.Pos {
					match = true
				}
				return !match
			})
		}
		if !match {
			continue
		}
		fileVersion := pkg.TypesInfo.FileVersions[file]
		if file.GoVersion != "" && file.GoVersion == fileVersion {
			diagnostic.Msg = fmt.Sprintf("%s (file declares //go:build %s)", diagnostic.Msg, fileVersion)
		} else {
			diagnostic.Msg = fmt.Sprintf("%s (-lang was set to %s; check go.mod)", diagnostic.Msg, version.Lang(pkg.Types.GoVersion()))
		}
		return diagnostic
	}
	return diagnostic
}

// Copyright 2016 The Go Authors. All rights reserved.
// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-Go file.

package load

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"go/version"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func directiveContracts(p *packages.Package) []error {
	var errs []error
	seen := make(map[types.Object]bool)
	noescape := make(map[*ast.FuncDecl]bool)
	reject := func(pos token.Pos, message string) {
		errs = append(errs, fmt.Errorf("%s: %s", p.Fset.Position(pos), message))
	}
	for _, file := range p.Syntax {
		unsafe := false
		for _, spec := range file.Imports {
			if path, _ := strconv.Unquote(spec.Path.Value); path == "unsafe" {
				unsafe = true
			}
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				verb, args, hasArgs := strings.Cut(comment.Text, " ")
				switch verb {
				case "//go:noescape":
					for _, decl := range file.Decls {
						if comment.Pos() >= decl.End() {
							continue
						}
						if fn, ok := decl.(*ast.FuncDecl); ok && comment.End() < fn.Pos() && fn.Body != nil && !noescape[fn] {
							pos := fn.Name.Pos()
							if fn.Recv != nil {
								pos = fn.Recv.Pos()
							}
							reject(pos, "can only use //go:noescape with external func implementations")
							noescape[fn] = true
						}
						break
					}
				case "//go:systemstack", "//go:nowritebarrier", "//go:nowritebarrierrec", "//go:yeswritebarrierrec":
					if p.PkgPath != "runtime" {
						reject(comment.Pos()+2, verb+" only allowed in runtime")
					}
				case "//go:cgo_ldflag":
					args = strings.TrimSpace(args)
					_, err := strconv.Unquote(args)
					if err != nil || !strings.HasPrefix(args, `"`) {
						reject(comment.Pos()+2, `usage: //go:cgo_ldflag "arg"`)
					} else {
						reject(comment.Pos()+2, "//go:cgo_ldflag requires cgo, which is outside the supported surface")
					}
				case "//go:linkname", "//go:linknamestd":
					if !hasArgs {
						continue
					}
					fields := strings.Fields(args)
					if len(fields) < 1 || len(fields) > 2 {
						reject(comment.Pos()+2, "usage: "+verb+" localname [linkname]")
						continue
					}
					if !unsafe {
						reject(comment.Pos()+2, verb+` only allowed in Go files that import "unsafe"`)
						continue
					}
					if len(fields) == 2 && strings.Contains(fields[1], "[") && strings.Contains(fields[1], "]") {
						reject(comment.Pos()+2, verb+" reference of an instantiation is not allowed")
						continue
					}
					obj := p.Types.Scope().Lookup(fields[0])
					switch obj.(type) {
					case *types.Func, *types.Var:
						if seen[obj] {
							reject(comment.Pos()+2, "duplicate "+verb+" for "+fields[0])
						}
						seen[obj] = true
					default:
						if p.Types.GoVersion() == "" || version.Compare(p.Types.GoVersion(), "go1.18") >= 0 {
							reject(comment.Pos()+2, verb+" must refer to declared function or variable")
						}
					}
				}
			}
		}
	}
	return errs
}

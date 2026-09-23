// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-Go file.

package emit

import (
	"fmt"
	"go/types"
)

// Assertion diagnostics also use the package of an unnamed struct's private
// fields. Types with method metadata use the receiver package first.
func assertionPackage(t types.Type) string {
	t = types.Unalias(t)
	var pkg *types.Package
	if named, ok := t.(*types.Named); ok {
		pkg = named.Obj().Pkg()
	} else if !isInterfaceType(t) && types.NewMethodSet(t).Len() > 0 {
		if pointer, ok := t.(*types.Pointer); ok {
			if named, ok := types.Unalias(pointer.Elem()).(*types.Named); ok {
				pkg = named.Obj().Pkg()
			}
		}
	} else if structure, ok := t.(*types.Struct); ok {
		for i := 0; i < structure.NumFields(); i++ {
			field := structure.Field(i)
			if !field.Exported() {
				pkg = field.Pkg()
				break
			}
		}
	}
	if pkg == nil {
		return ""
	}
	if pkg.Name() == "main" {
		return "main"
	}
	return pkg.Path()
}

func assertionPackagePart(t types.Type) string {
	if pkg := assertionPackage(t); pkg != "" {
		return ", assertionPkg: " + dartString(pkg)
	}
	return ""
}

func methodPackagePart(t types.Type) string {
	t = types.Unalias(t)
	if _, named := t.(*types.Named); !named {
		switch u := t.(type) {
		case *types.Pointer:
			t = types.Unalias(u.Elem())
		case *types.Array:
			t = types.Unalias(u.Elem())
		case *types.Slice:
			t = types.Unalias(u.Elem())
		case *types.Chan:
			t = types.Unalias(u.Elem())
		}
	}
	if named, ok := t.(*types.Named); ok && named.Obj().Pkg() != nil {
		return fmt.Sprintf(", methodPkg: %s", dartString(named.Obj().Pkg().Path()))
	}
	return ""
}

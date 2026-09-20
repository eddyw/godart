// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-Go file.

package load

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func interfaceCycles(p *packages.Package) []error {
	var errs []error
	var finder cycleFinder
	for _, file := range p.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			iface, ok := node.(*ast.InterfaceType)
			if !ok {
				return true
			}
			if finder.hasCycle(types.Unalias(p.TypesInfo.TypeOf(iface)).(*types.Interface)) {
				errs = append(errs, fmt.Errorf("%s: invalid recursive type: anonymous interface refers to itself", p.Fset.Position(iface.Pos())))
				for typ := range finder.cyclic {
					finder.cyclic[typ] = false
				}
			}
			return false
		})
	}
	return errs
}

type cycleFinder struct {
	cyclic map[*types.Interface]bool
}

// hasCycle reports whether typ is part of an anonymous interface cycle.
func (f *cycleFinder) hasCycle(typ *types.Interface) bool {
	// We use Method instead of ExplicitMethod to implicitly expand any
	// embedded interfaces. Then we just need to walk any anonymous
	// types, keeping track of *types.Interface types we visit along
	// the way.
	for i := 0; i < typ.NumMethods(); i++ {
		if f.visit(typ.Method(i).Type()) {
			return true
		}
	}
	return false
}

// visit recursively walks typ0 to check any referenced interface types.
func (f *cycleFinder) visit(typ0 types.Type) bool {
	for { // loop for tail recursion
		switch typ := types.Unalias(typ0).(type) {
		default:
			panic(fmt.Sprintf("unexpected type: %T", typ))

		case *types.Basic, *types.Named, *types.TypeParam:
			return false // named types cannot be part of an anonymous cycle
		case *types.Pointer:
			typ0 = typ.Elem()
		case *types.Array:
			typ0 = typ.Elem()
		case *types.Chan:
			typ0 = typ.Elem()
		case *types.Map:
			if f.visit(typ.Key()) {
				return true
			}
			typ0 = typ.Elem()
		case *types.Slice:
			typ0 = typ.Elem()

		case *types.Struct:
			for i := 0; i < typ.NumFields(); i++ {
				if f.visit(typ.Field(i).Type()) {
					return true
				}
			}
			return false

		case *types.Interface:
			// The empty interface (e.g., "any") cannot be part of a cycle.
			if typ.NumExplicitMethods() == 0 && typ.NumEmbeddeds() == 0 {
				return false
			}

			// Allocate only for a non-empty anonymous interface.
			if x, ok := f.cyclic[typ]; ok {
				return x
			}
			if f.cyclic == nil {
				f.cyclic = make(map[*types.Interface]bool)
			}
			f.cyclic[typ] = true
			if f.hasCycle(typ) {
				return true
			}
			f.cyclic[typ] = false
			return false

		case *types.Signature:
			return f.visit(typ.Params()) || f.visit(typ.Results())
		case *types.Tuple:
			for i := 0; i < typ.Len(); i++ {
				if f.visit(typ.At(i).Type()) {
					return true
				}
			}
			return false
		}
	}
}

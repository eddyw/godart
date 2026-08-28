// Package typeset reads the type set of a Go constraint.
//
// A type parameter holds an interface with a list of terms. Go gives such a
// parameter a core type when each term shares 1 underlying type, and an
// operation of the core type then applies to the parameter.
package typeset

import "go/types"

// maxDepth bounds the walk of an embedded interface. A cycle is not possible in
// a program that the type checker accepted, and the bound holds against a
// mistake in the compiler.
const maxDepth = 8

// Terms reads the types of the type set of a constraint. It reports whether the
// walk found a closed set.
//
// An interface with a method only holds no closed set, thus the result is
// false. An empty list with a true result means the constraint is `any`.
func Terms(t types.Type) ([]types.Type, bool) {
	iface, ok := types.Unalias(t).Underlying().(*types.Interface)
	if !ok {
		return nil, false
	}
	var out []types.Type
	if !walk(iface, &out, 0) {
		return nil, false
	}
	return out, true
}

func walk(iface *types.Interface, out *[]types.Type, depth int) bool {
	if depth > maxDepth {
		return false
	}
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		switch t := types.Unalias(iface.EmbeddedType(i)).(type) {
		case *types.Union:
			for k := 0; k < t.Len(); k++ {
				term := t.Term(k)
				inner, ok := types.Unalias(term.Type()).Underlying().(*types.Interface)
				if ok && !inner.IsMethodSet() {
					if !walk(inner, out, depth+1) {
						return false
					}
					continue
				}
				*out = append(*out, term.Type())
			}
		case *types.Interface:
			if !walk(t, out, depth+1) {
				return false
			}
		default:
			*out = append(*out, t)
		}
	}
	return true
}

// Core gives the core type of a type. The result is the underlying type for a
// type that is not a type parameter.
//
// A type parameter holds a core type when each term of its type set shares 1
// underlying type. The result is nil when the parameter holds no such type.
func Core(t types.Type) types.Type {
	tp, ok := types.Unalias(t).(*types.TypeParam)
	if !ok {
		return types.Unalias(t).Underlying()
	}
	terms, ok := Terms(tp.Constraint())
	if !ok || len(terms) == 0 {
		return nil
	}
	first := terms[0].Underlying()
	for _, term := range terms[1:] {
		if !types.Identical(first, term.Underlying()) {
			return nil
		}
	}
	return first
}

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
		default:
			// An embedded interface intersects the type set, and a method
			// adds no term. A named interface walks the same as a literal
			// one.
			if inner, ok := types.Unalias(t).Underlying().(*types.Interface); ok {
				if !walk(inner, out, depth+1) {
					return false
				}
				continue
			}
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

// Set describes the type set of a type parameter. The compiler reads it to
// choose the Dart type of the parameter and the Dart form of each operation.
type Set struct {
	// Numeric reports that each term is an integer type or a float type. Such
	// a set holds 1 Dart bound, thus a Dart operator applies to it.
	Numeric bool

	// Integer reports that each term is an integer type. Each such term gives
	// the Dart type int.
	Integer bool

	// Float reports that each term is a float type.
	Float bool

	// Wide reports that a term is an unsigned integer type of 64 bits. Go
	// reads the bit pattern of such a type as unsigned, and Dart reads the
	// same pattern as a negative number.
	Wide bool

	// Plain reports that a term is an integer type that Dart holds with the
	// sign of Go. The signed operators of Dart give the Go result for it.
	Plain bool
}

// MixedSign reports that the type set holds an unsigned integer type of 64 bits
// and another integer type. A division, a remainder, and a right shift each
// read the sign, thus those 3 operations hold 2 Dart forms.
func (s Set) MixedSign() bool { return s.Integer && s.Wide && s.Plain }

// Unsigned64 reports that each term is an unsigned integer type of 64 bits.
func (s Set) Unsigned64() bool { return s.Integer && s.Wide && !s.Plain }

// Describe reads the type set of a type parameter. It reports false for a type
// that is not a type parameter, and for a constraint with no closed type set.
func Describe(t types.Type) (Set, bool) {
	tp, ok := types.Unalias(t).(*types.TypeParam)
	if !ok {
		return Set{}, false
	}
	terms, ok := Terms(tp.Constraint())
	if !ok || len(terms) == 0 {
		return Set{}, false
	}
	out := Set{Numeric: true, Integer: true, Float: true}
	for _, term := range terms {
		b, ok := types.Unalias(term).Underlying().(*types.Basic)
		if !ok {
			return Set{}, true
		}
		switch {
		case b.Info()&types.IsInteger != 0:
			out.Float = false
		case b.Info()&types.IsFloat != 0:
			out.Integer = false
		default:
			return Set{}, true
		}
		switch b.Kind() {
		case types.Uint, types.Uint64, types.Uintptr:
			out.Wide = true
		case types.Float32, types.Float64:
		default:
			out.Plain = true
		}
	}
	return out, true
}

// HoldsParam reports whether a type reads a type parameter anywhere in its
// structure.
func HoldsParam(t types.Type) bool {
	found := false
	var walk func(types.Type)
	seen := map[types.Type]bool{}
	walk = func(t types.Type) {
		if t == nil || found || seen[t] {
			return
		}
		seen[t] = true
		switch t := types.Unalias(t).(type) {
		case *types.TypeParam:
			found = true
		case *types.Slice:
			walk(t.Elem())
		case *types.Array:
			walk(t.Elem())
		case *types.Pointer:
			walk(t.Elem())
		case *types.Chan:
			walk(t.Elem())
		case *types.Map:
			walk(t.Key())
			walk(t.Elem())
		case *types.Named:
			for i := 0; i < t.TypeArgs().Len(); i++ {
				walk(t.TypeArgs().At(i))
			}
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				walk(t.Field(i).Type())
			}
		case *types.Interface:
			for i := 0; i < t.NumMethods(); i++ {
				walk(t.Method(i).Type())
			}
		case *types.Signature:
			for i := 0; i < t.Params().Len(); i++ {
				walk(t.Params().At(i).Type())
			}
			for i := 0; i < t.Results().Len(); i++ {
				walk(t.Results().At(i).Type())
			}
		}
	}
	walk(t)
	return found
}

// BytesOrText reports whether each term of the type set of a parameter is a
// byte slice or a string.
//
// Such a set holds 2 Dart types, thus no Dart member fits both. The dictionary
// of the parameter carries the length, the index, the slice, and the 2
// conversions. The standard library declares such a parameter in
// encoding/json/jsontext and in encoding/json/v2.
func BytesOrText(t types.Type) bool {
	tp, ok := types.Unalias(t).(*types.TypeParam)
	if !ok {
		return false
	}
	terms, ok := Terms(tp.Constraint())
	if !ok || len(terms) == 0 {
		return false
	}
	for _, term := range terms {
		if !bytesOrTextTerm(term.Underlying()) {
			return false
		}
	}
	return true
}

func bytesOrTextTerm(u types.Type) bool {
	if b, ok := u.(*types.Basic); ok {
		return b.Kind() == types.String
	}
	s, ok := u.(*types.Slice)
	if !ok {
		return false
	}
	b, ok := types.Unalias(s.Elem()).Underlying().(*types.Basic)
	return ok && b.Kind() == types.Uint8
}

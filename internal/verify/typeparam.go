package verify

import (
	"go/ast"
	"go/token"
	"go/types"
)

// checkTypeParamOp rejects an operation on a value whose type is a type
// parameter with more than 1 Dart form.
//
// The compiler gives a type parameter 1 Dart type. A constraint that lists
// `string | []byte` holds 2 Dart types, thus no Dart operator fits both. Such a
// type parameter still accepts an assignment, an equality, and a call.
func (v *verifier) checkTypeParamOp(n ast.Node) {
	switch n := n.(type) {
	case *ast.IndexExpr:
		v.rejectMixed(n.Pos(), v.typeOf(n.X), "an index")
	case *ast.SliceExpr:
		v.rejectMixed(n.Pos(), v.typeOf(n.X), "a slice expression")
	case *ast.RangeStmt:
		v.rejectMixed(n.Pos(), v.typeOf(n.X), "a range loop")
	}
}

// checkTypeParamBuiltin rejects len and cap on such a type parameter.
func (v *verifier) checkTypeParamBuiltin(e *ast.CallExpr, fn string) {
	if fn != "len" && fn != "cap" || len(e.Args) != 1 {
		return
	}
	v.rejectMixed(e.Pos(), v.typeOf(e.Args[0]), "the builtin "+fn)
}

func (v *verifier) rejectMixed(pos token.Pos, t types.Type, what string) {
	tp, ok := types.Unalias(t).(*types.TypeParam)
	if !ok || !mixedDartForm(tp) {
		return
	}
	v.reject(pos, "the compiler gives a type parameter 1 Dart type",
		"%s reads the type parameter %s, and its constraint holds more than 1 Dart type",
		what, tp.Obj().Name())
}

// mixedDartForm reports whether the type set of a type parameter holds more
// than 1 Dart form.
//
// 2 terms give 1 Dart form when their underlying types are identical. The check
// stays strict, thus it rejects a case that the emitter cannot write.
func mixedDartForm(tp *types.TypeParam) bool {
	iface, ok := types.Unalias(tp.Constraint()).Underlying().(*types.Interface)
	if !ok {
		return false
	}
	var terms []types.Type
	if !collectTerms(iface, &terms, 0) || len(terms) < 2 {
		return false
	}
	first := terms[0].Underlying()
	for _, t := range terms[1:] {
		if !types.Identical(first, t.Underlying()) {
			return true
		}
	}
	return false
}

// collectTerms reads the types of the type set of a constraint. It reports
// whether the walk found a closed set.
func collectTerms(iface *types.Interface, out *[]types.Type, depth int) bool {
	if depth > 8 {
		return false
	}
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		switch t := types.Unalias(iface.EmbeddedType(i)).(type) {
		case *types.Union:
			for k := 0; k < t.Len(); k++ {
				term := t.Term(k)
				if inner, ok := types.Unalias(term.Type()).Underlying().(*types.Interface); ok &&
					!inner.IsMethodSet() {
					if !collectTerms(inner, out, depth+1) {
						return false
					}
					continue
				}
				*out = append(*out, term.Type())
			}
		case *types.Interface:
			if !collectTerms(t, out, depth+1) {
				return false
			}
		default:
			*out = append(*out, t)
		}
	}
	return true
}

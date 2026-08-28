package verify

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/eddyw/godart/internal/typeset"
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
// A type parameter with a core type gives 1 Dart type, thus each operation of
// the core type applies. A parameter with no core type holds more than 1 Dart
// type, and no Dart operator fits each one.
func mixedDartForm(tp *types.TypeParam) bool {
	return typeset.Core(tp) == nil
}

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
// type parameter still accepts an assignment, an equality, a comparison, and a
// call.
func (v *verifier) checkTypeParamOp(n ast.Node) {
	switch n := n.(type) {
	case *ast.IndexExpr:
		v.rejectMixed(n.Pos(), v.typeOf(n.X), "an index")
	case *ast.SliceExpr:
		v.rejectMixed(n.Pos(), v.typeOf(n.X), "a slice expression")
	case *ast.RangeStmt:
		v.rejectMixed(n.Pos(), v.typeOf(n.X), "a range loop")
	case *ast.BinaryExpr:
		v.checkTypeParamArith(n)
	case *ast.UnaryExpr:
		switch n.Op {
		case token.SUB, token.XOR:
			v.rejectArith(n.Pos(), v.typeOf(n.X), n.Op, "the operator "+n.Op.String())
		}
	case *ast.IncDecStmt:
		v.rejectArith(n.Pos(), v.typeOf(n.X), token.ADD, "the operator "+n.Tok.String())
	case *ast.AssignStmt:
		v.checkTypeParamAssign(n)
	case *ast.CallExpr:
		v.checkTypeParamConv(n)
	}
}

// checkTypeParamBuiltin rejects a builtin that reads such a type parameter.
func (v *verifier) checkTypeParamBuiltin(e *ast.CallExpr, fn string) {
	switch fn {
	case "len", "cap":
		if len(e.Args) == 1 {
			v.rejectMixed(e.Pos(), v.typeOf(e.Args[0]), "the builtin "+fn)
		}
	case "print", "println":
		// The text of an unsigned value of 64 bits differs from the text of
		// the same bit pattern as a signed value.
		for _, a := range e.Args {
			v.rejectMixedSign(a.Pos(), v.typeOf(a), "the builtin "+fn)
		}
	}
}

// arithOps names each binary operator that reads the width of its type.
var arithOps = map[token.Token]bool{
	token.ADD: true, token.SUB: true, token.MUL: true, token.QUO: true,
	token.REM: true, token.AND: true, token.OR: true, token.XOR: true,
	token.AND_NOT: true, token.SHL: true, token.SHR: true,
}

func (v *verifier) checkTypeParamArith(e *ast.BinaryExpr) {
	if !arithOps[e.Op] {
		return
	}
	t := v.typeOf(e.X)
	// The right operand of a shift holds its own type. Each other operator
	// gives the 2 operands 1 type, and an untyped constant takes it.
	if e.Op != token.SHL && e.Op != token.SHR {
		if _, ok := types.Unalias(t).(*types.TypeParam); !ok {
			t = v.typeOf(e.Y)
		}
	}
	v.rejectArith(e.Pos(), t, e.Op, "the operator "+e.Op.String())
}

func (v *verifier) checkTypeParamAssign(s *ast.AssignStmt) {
	if s.Tok == token.ASSIGN || s.Tok == token.DEFINE || len(s.Lhs) != 1 {
		return
	}
	op := s.Tok + (token.ADD - token.ADD_ASSIGN)
	v.rejectArith(s.Pos(), v.typeOf(s.Lhs[0]), op, "the operator "+s.Tok.String())
}

// rejectArith rejects an arithmetic operation that holds no single Dart form.
//
// A type set of integer types keeps each operator, because the dictionary holds
// the mask of the width. A set of float types keeps the same rule: the
// dictionary of float32 holds the round of 32 bits, and the dictionary of
// float64 holds the identity. A set that mixes an integer type with a float
// type keeps each operator that Go allows on both, and a division of such a
// set falls outside the subset, because the Dart operator `/` always gives a
// double. Such a mixed set holds 1 Dart bound of `num`, and no mask of a
// narrow width fits that bound, thus a narrow term of a mixed set stays out.
func (v *verifier) rejectArith(pos token.Pos, t types.Type, op token.Token, what string) {
	set, ok := typeset.Describe(t)
	if !ok || typeset.Core(t) != nil {
		return
	}
	tp := types.Unalias(t).(*types.TypeParam)
	switch {
	case !set.Numeric:
		v.reject(pos, "the compiler gives a type parameter 1 Dart type",
			"%s reads the type parameter %s, and its constraint holds more than 1 Dart type",
			what, tp.Obj().Name())
	case !set.Integer && !set.Float && set.Narrow:
		v.reject(pos, "the dictionary holds a mask for a set of integer types or of float types",
			"%s reads the type parameter %s, and its constraint mixes an integer type with a float type that needs a mask of its width",
			what, tp.Obj().Name())
	case op == token.QUO && !set.Integer && !set.Float:
		v.reject(pos, "the Dart operator / always gives a double",
			"%s reads the type parameter %s, and its constraint holds an integer type and a float type",
			what, tp.Obj().Name())
	}
	switch op {
	case token.QUO, token.REM, token.SHR:
		v.rejectMixedSign(pos, t, what)
	}
}

// checkTypeParamConv rejects a conversion that reads a type parameter with more
// than 1 Dart form.
//
// A conversion to an interface stays inside the subset. A conversion of a type
// set of mixed sign to a float type falls outside it. Dart holds the bit pattern of an unsigned value of 64 bits as a
// negative number, thus the 2 sign forms give 2 Dart expressions.
func (v *verifier) checkTypeParamConv(e *ast.CallExpr) {
	tv, ok := v.info.Types[e.Fun]
	if !ok || !tv.IsType() || len(e.Args) != 1 {
		return
	}
	to, from := v.typeOf(e), v.typeOf(e.Args[0])
	// A conversion to an interface reads the description of the dictionary,
	// thus each type set reaches an interface. The underlying type of a type
	// parameter is its constraint, thus the test names the parameter first.
	if _, param := types.Unalias(to).(*types.TypeParam); !param && types.IsInterface(to) {
		return
	}
	v.rejectMixed(e.Pos(), to, "a conversion")
	v.rejectMixed(e.Pos(), from, "a conversion")
	if isFloat(to) {
		v.rejectMixedSign(e.Pos(), from, "a conversion to a float type")
	}
}

func (v *verifier) rejectMixed(pos token.Pos, t types.Type, what string) {
	tp, ok := types.Unalias(t).(*types.TypeParam)
	if !ok || !mixedDartForm(tp) {
		return
	}
	// A type set of a byte slice and a string reads the length, the index, the
	// slice, and the 2 conversions from its dictionary.
	if typeset.BytesOrText(tp) {
		return
	}
	v.reject(pos, "the compiler gives a type parameter 1 Dart type",
		"%s reads the type parameter %s, and its constraint holds more than 1 Dart type",
		what, tp.Obj().Name())
}

// rejectMixedSign rejects an operation that reads the sign of a type set that
// holds an unsigned type of 64 bits and another integer type.
func (v *verifier) rejectMixedSign(pos token.Pos, t types.Type, what string) {
	if set, ok := typeset.Describe(t); !ok || !set.MixedSign() {
		return
	}
	tp := types.Unalias(t).(*types.TypeParam)
	v.reject(pos, "an unsigned type of 64 bits fills the width of a Dart int",
		"%s reads the sign of the type parameter %s, and its constraint holds "+
			"an unsigned type of 64 bits and another integer type",
		what, tp.Obj().Name())
}

// mixedDartForm reports whether the type set of a type parameter holds more
// than 1 Dart form.
//
// A type parameter with a core type gives 1 Dart type, thus each operation of
// the core type applies. A type set of integer types also gives 1 Dart type,
// and the dictionary holds the mask of the width. A type set of float types
// gives the Dart type double, and the dictionary of float32 holds the round of
// 32 bits. Each other parameter with no core type holds more than 1 Dart type.
func mixedDartForm(tp *types.TypeParam) bool {
	if typeset.Core(tp) != nil {
		return false
	}
	set, ok := typeset.Describe(tp)
	return !ok || (!set.Integer && !set.Float)
}

package normalize

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
)

// copyPoint inserts a struct copy. Go gives a struct and an array value
// semantics, thus each copy point needs a fresh value.
func (b *builder) copyPoint(e ir.Expr) ir.Expr {
	if e == nil || !needsCopy(e.GoType()) || isFresh(e) {
		return e
	}
	return &ir.Copy{X: e, Type: e.GoType()}
}

func needsCopy(t types.Type) bool {
	if t == nil {
		return false
	}
	switch typeset.Core(t).(type) {
	case *types.Struct, *types.Array:
		return true
	}
	return false
}

// isFresh reports whether the expression already builds a new value.
func isFresh(e ir.Expr) bool {
	switch e.(type) {
	case *ir.Composite, *ir.Call, *ir.Copy, *ir.Zero, *ir.Builtin, *ir.Conv:
		return true
	}
	return false
}

func (b *builder) tempVar(pos token.Pos, role string, t types.Type) *ir.Var {
	return &ir.Var{Name: b.temp(pos, role), Type: t}
}

// lvalue prepares a location for a write. The result holds the statements that
// evaluate the operands, a read of the location, and a builder for the write.
func (b *builder) lvalue(e ast.Expr) (pre []ir.Stmt, get ir.Expr, set func(ir.Expr) ir.Stmt) {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return b.lvalue(e.X)
	case *ast.Ident:
		x := b.expr(e)
		return nil, x, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: x, Rhs: v} }
	case *ast.SelectorExpr:
		x := b.expr(e)
		field, ok := x.(*ir.Field)
		if !ok {
			return nil, x, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: x, Rhs: v} }
		}
		pre, base := b.hoist(field.X, e.X.Pos(), "b")
		read := &ir.Field{X: base, Name: field.Name, Type: field.Type}
		return pre, read, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: read, Rhs: v} }
	case *ast.IndexExpr:
		x := b.expr(e)
		idx, ok := x.(*ir.Index)
		if !ok {
			return nil, x, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: x, Rhs: v} }
		}
		pre1, base := b.hoist(idx.X, e.X.Pos(), "b")
		pre2, key := b.hoist(idx.Index, e.Index.Pos(), "k")
		read := &ir.Index{X: base, Index: key, Type: idx.Type}
		return append(pre1, pre2...), read, func(v ir.Expr) ir.Stmt {
			return &ir.Assign{Lhs: read, Rhs: v}
		}
	case *ast.StarExpr:
		x := b.expr(e)
		return nil, x, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: x, Rhs: v} }
	}
	x := b.expr(e)
	return nil, x, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: x, Rhs: v} }
}

// forceHoist puts an expression into a temporary variable in each case. A
// defer statement reads its operands at that moment, thus a later write to a
// variable does not change the deferred call.
func (b *builder) forceHoist(e ir.Expr, pos token.Pos, role string) ([]ir.Stmt, ir.Expr) {
	if e == nil {
		return nil, nil
	}
	v := b.tempVar(pos, role, e.GoType())
	return []ir.Stmt{&ir.Declare{Vars: []*ir.Var{v}, Init: e}}, &ir.Ident{Name: v.Name, Type: v.Type}
}

// hoist puts an expression into a temporary variable when a second read of the
// expression would run its side effects again.
func (b *builder) hoist(e ir.Expr, pos token.Pos, role string) ([]ir.Stmt, ir.Expr) {
	switch e.(type) {
	case *ir.Ident, *ir.Const:
		return nil, e
	}
	v := b.tempVar(pos, role, e.GoType())
	return []ir.Stmt{&ir.Declare{Vars: []*ir.Var{v}, Init: e}}, &ir.Ident{Name: v.Name, Type: v.Type}
}

// multiValue builds an expression that gives more than 1 value. A map read, a
// type assertion, and a channel receive each have a form with 2 values.
func (b *builder) multiValue(e ast.Expr) ir.Expr {
	if x, ok := unparenExpr(e).(*ast.UnaryExpr); ok && x.Op == token.ARROW {
		// The type of the expression is already a pair in this context, thus
		// the element type comes from the channel.
		elem := chanElem(b.typeOf(x.X))
		return &ir.Recv{
			Chan:  b.expr(x.X),
			Comma: true,
			Type:  types.NewTuple(anonVar(elem), anonVar(types.Typ[types.Bool])),
		}
	}
	if x, ok := unparenExpr(e).(*ast.TypeAssertExpr); ok {
		assert := b.typeOf(x.Type)
		return &ir.TypeAssert{
			X:      b.expr(x.X),
			Assert: assert,
			Comma:  true,
			Type:   types.NewTuple(anonVar(assert), anonVar(types.Typ[types.Bool])),
		}
	}
	if x, ok := unparenExpr(e).(*ast.IndexExpr); ok {
		if m := mapType(b.typeOf(x.X)); m != nil {
			return &ir.MapRead{
				X:    b.expr(x.X),
				Key:  b.convert(b.expr(x.Index), m.Key()),
				Type: types.NewTuple(anonVar(m.Elem()), anonVar(types.Typ[types.Bool])),
			}
		}
	}
	return b.expr(e)
}

func unparenExpr(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

// isIfaceType reports whether a value of the type holds an interface pair.
//
// A type parameter holds no such pair. Its underlying type is the constraint,
// and the constraint is an interface.
func isIfaceType(t types.Type) bool {
	if _, ok := types.Unalias(t).(*types.TypeParam); ok {
		return false
	}
	_, ok := types.Unalias(t).Underlying().(*types.Interface)
	return ok
}

func mapType(t types.Type) *types.Map {
	m, _ := typeset.Core(t).(*types.Map)
	return m
}

func anonVar(t types.Type) *types.Var {
	return types.NewVar(token.NoPos, nil, "", t)
}

func (b *builder) exprAs(e ast.Expr, want types.Type) ir.Expr {
	return b.convert(b.expr(e), want)
}

// convert makes an implicit Go conversion explicit.
func (b *builder) convert(x ir.Expr, want types.Type) ir.Expr {
	if x == nil || want == nil {
		return x
	}
	got := x.GoType()
	if got == nil {
		return x
	}
	if basic, ok := types.Unalias(got).(*types.Basic); ok {
		if basic.Kind() == types.UntypedNil {
			return &ir.Zero{Type: want}
		}
		if basic.Info()&types.IsUntyped != 0 {
			if c, ok := x.(*ir.Const); ok {
				return &ir.Const{Value: c.Value, Type: want}
			}
			return &ir.Conv{X: x, From: got, Type: want}
		}
	}
	if types.Identical(got, want) {
		return x
	}
	if isIfaceType(want) {
		if !isIfaceType(got) {
			return &ir.Conv{X: x, From: got, Type: want}
		}
	}
	return x
}

func (b *builder) expr(e ast.Expr) ir.Expr {
	if e == nil {
		return nil
	}
	if tv, ok := b.info.Types[e]; ok && tv.Value != nil {
		return &ir.Const{Value: tv.Value, Type: tv.Type}
	}
	switch e := e.(type) {
	case *ast.ParenExpr:
		return b.expr(e.X)
	case *ast.Ident:
		return b.ident(e)
	case *ast.BinaryExpr:
		return b.binary(e)
	case *ast.UnaryExpr:
		return b.unary(e)
	case *ast.StarExpr:
		return &ir.Unary{Op: token.MUL, X: b.expr(e.X), Type: b.typeOf(e)}
	case *ast.CallExpr:
		return b.call(e)
	case *ast.SelectorExpr:
		return b.selector(e)
	case *ast.IndexExpr:
		return b.index(e)
	case *ast.IndexListExpr:
		return b.expr(e.X)
	case *ast.SliceExpr:
		return b.slice(e)
	case *ast.CompositeLit:
		return b.composite(e)
	case *ast.FuncLit:
		return b.funcLit(e)
	case *ast.TypeAssertExpr:
		return &ir.TypeAssert{X: b.expr(e.X), Assert: b.typeOf(e.Type), Type: b.typeOf(e)}
	}
	b.fail(e.Pos(), "the expression %T is outside the subset", e)
	return nil
}

func (b *builder) ident(e *ast.Ident) ir.Expr {
	obj := b.info.ObjectOf(e)
	if obj == nil {
		if e.Name == "nil" {
			return &ir.Zero{Type: b.typeOf(e)}
		}
		b.fail(e.Pos(), "the identifier %s has no object", e.Name)
		return nil
	}
	if _, ok := obj.(*types.Nil); ok {
		return &ir.Zero{Type: b.typeOf(e)}
	}
	if n, ok := b.names[obj]; ok {
		return &ir.Ident{Name: n, Obj: obj, Type: obj.Type(), Boxed: b.boxed(obj)}
	}
	return &ir.Ident{Name: obj.Name(), Obj: obj, Type: obj.Type(), Boxed: b.boxed(obj)}
}

func (b *builder) binary(e *ast.BinaryExpr) ir.Expr {
	t := b.typeOf(e)
	x := b.expr(e.X)
	y := b.expr(e.Y)
	if x == nil || y == nil {
		return nil
	}
	switch e.Op {
	case token.SHL, token.SHR:
		x = b.convert(x, b.typeOf(e))
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		x, y = b.balance(x, y)
	default:
		x = b.convert(x, t)
		y = b.convert(y, t)
	}
	return &ir.Binary{Op: e.Op, X: x, Y: y, Type: t}
}

// balance gives 2 operands of a comparison the same type. Go allows an untyped
// operand, and the other operand then gives the type.
func (b *builder) balance(x, y ir.Expr) (ir.Expr, ir.Expr) {
	if isUntyped(x.GoType()) && !isUntyped(y.GoType()) {
		return b.convert(x, y.GoType()), y
	}
	if isUntyped(y.GoType()) && !isUntyped(x.GoType()) {
		return x, b.convert(y, x.GoType())
	}
	// Go asks that 1 operand is assignable to the type of the other. A
	// concrete operand of a comparison with an interface becomes an interface
	// value, because Go compares the type and the value.
	if isIfaceType(x.GoType()) && !isIfaceType(y.GoType()) {
		return x, b.convert(y, x.GoType())
	}
	if isIfaceType(y.GoType()) && !isIfaceType(x.GoType()) {
		return b.convert(x, y.GoType()), y
	}
	return x, y
}

func isUntyped(t types.Type) bool {
	basic, ok := types.Unalias(t).(*types.Basic)
	return ok && basic.Info()&types.IsUntyped != 0
}

func (b *builder) unary(e *ast.UnaryExpr) ir.Expr {
	if e.Op == token.ARROW {
		return &ir.Recv{Chan: b.expr(e.X), Type: b.typeOf(e)}
	}
	x := b.expr(e.X)
	if x == nil {
		return nil
	}
	return &ir.Unary{Op: e.Op, X: x, Type: b.typeOf(e)}
}

func (b *builder) selector(e *ast.SelectorExpr) ir.Expr {
	sel, ok := b.info.Selections[e]
	if !ok {
		return b.ident(e.Sel)
	}
	switch sel.Kind() {
	case types.FieldVal:
		x := b.expr(e.X)
		if x == nil {
			return nil
		}
		return b.fieldPath(x, sel)
	case types.MethodVal:
		x := b.expr(e.X)
		if x == nil {
			return nil
		}
		fn, _ := sel.Obj().(*types.Func)
		return &ir.Method{Recv: b.embeddedPath(x, sel), Obj: fn, Type: sel.Type()}
	}
	b.fail(e.Pos(), "a method expression is not supported yet")
	return nil
}

// embeddedPath walks the embedded fields that lead to a promoted method. Dart
// gives no method promotion, thus the receiver becomes the embedded value.
func (b *builder) embeddedPath(x ir.Expr, sel *types.Selection) ir.Expr {
	index := sel.Index()
	t := x.GoType()
	for _, i := range index[:len(index)-1] {
		st := structOf(t)
		if st == nil {
			b.fail(token.NoPos, "the method path leaves a struct type")
			return x
		}
		f := st.Field(i)
		x = &ir.Field{X: x, Name: f.Name(), Type: f.Type()}
		t = f.Type()
	}
	return x
}

// fieldPath walks the embedded fields that lead to the selected field.
func (b *builder) fieldPath(x ir.Expr, sel *types.Selection) ir.Expr {
	t := x.GoType()
	for _, i := range sel.Index() {
		st := structOf(t)
		if st == nil {
			b.fail(token.NoPos, "the field path leaves a struct type")
			return nil
		}
		f := st.Field(i)
		x = &ir.Field{X: x, Name: f.Name(), Type: f.Type()}
		t = f.Type()
	}
	return x
}

func structOf(t types.Type) *types.Struct {
	u := typeset.Core(t)
	if p, ok := u.(*types.Pointer); ok {
		u = typeset.Core(p.Elem())
	}
	st, _ := u.(*types.Struct)
	return st
}

func (b *builder) index(e *ast.IndexExpr) ir.Expr {
	if tv, ok := b.info.Types[e.Index]; ok && tv.IsType() {
		// The expression names an instance of a generic function or of a
		// generic type. The instance reads as the name itself, and the type
		// arguments reach the call through the info of the type checker.
		return b.expr(e.X)
	}
	x := b.expr(e.X)
	if x == nil {
		return nil
	}
	key := b.expr(e.Index)
	if m, ok := typeset.Core(x.GoType()).(*types.Map); ok {
		key = b.convert(key, m.Key())
	} else {
		key = b.convert(key, types.Typ[types.Int])
	}
	return &ir.Index{X: x, Index: key, Type: b.typeOf(e)}
}

func (b *builder) slice(e *ast.SliceExpr) ir.Expr {
	out := &ir.Slice{X: b.expr(e.X), Type: b.typeOf(e)}
	if e.Low != nil {
		out.Low = b.convert(b.expr(e.Low), types.Typ[types.Int])
	}
	if e.High != nil {
		out.High = b.convert(b.expr(e.High), types.Typ[types.Int])
	}
	if e.Max != nil {
		out.Max = b.convert(b.expr(e.Max), types.Typ[types.Int])
	}
	return out
}

func (b *builder) funcLit(e *ast.FuncLit) ir.Expr {
	sig, _ := b.typeOf(e).(*types.Signature)
	if sig == nil {
		return nil
	}
	out := &ir.FuncLit{Type: sig, Async: b.blocks.Blocks(e)}
	prev := b.fn
	prevPending := b.pending
	b.pending = nil
	inner := &ir.Func{Params: b.varsOf(sig.Params()), Results: b.results(sig)}
	b.fn = inner
	out.Params = inner.Params
	out.Results = inner.Results
	out.Body = b.block(e.Body)
	out.Defers = inner.Defers
	b.fn = prev
	b.pending = prevPending
	if !out.Async {
		return out
	}
	out.Machine = b.flatten(out.Body, out.Params, nil, true, out.Defers,
		b.frameClass(e.Pos()), e.Pos())
	addResults(out.Machine, out.Results)
	local := b.litName(e.Pos())
	b.pending = append(b.pending, &ir.LocalFunc{Name: local, Lit: out})
	return &ir.Ident{Name: local, Type: sig}
}

func zeroConst(t types.Type) *ir.Const {
	return &ir.Const{Value: constant.MakeInt64(0), Type: t}
}

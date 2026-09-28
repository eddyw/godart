package normalize

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"path/filepath"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
)

// copyPoint inserts a struct copy. Go gives a struct and an array value
// semantics, thus each copy point needs a fresh value.
func (b *builder) copyPoint(e ir.Expr) ir.Expr {
	if e == nil || !needsCopy(e.GoType()) || ir.Fresh(e) {
		return e
	}
	return &ir.Copy{X: e, Type: e.GoType()}
}

func needsCopy(t types.Type) bool {
	if t == nil {
		return false
	}
	if _, ok := types.Unalias(t).(*types.TypeParam); ok {
		return true
	}
	switch typeset.Core(t).(type) {
	case *types.Struct, *types.Array:
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
		if mr, ok := x.(*ir.MapRead); ok {
			pre1, base := b.hoist(mr.X, e.X.Pos(), "b")
			pre2, key := b.hoist(mr.Key, e.Index.Pos(), "k")
			read := &ir.MapRead{X: base, Key: key, Type: mr.Type}
			return append(pre1, pre2...), read, func(v ir.Expr) ir.Stmt {
				return &ir.Assign{Lhs: read, Rhs: v}
			}
		}
		idx, ok := x.(*ir.Index)
		if !ok {
			return nil, x, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: x, Rhs: v} }
		}
		pre1, base := b.hoist(idx.X, e.X.Pos(), "b")
		pre2, key := b.hoist(idx.Index, e.Index.Pos(), "k")
		read := &ir.Index{Pos: idx.Pos, X: base, Index: key, Type: idx.Type}
		return append(pre1, pre2...), read, func(v ir.Expr) ir.Stmt {
			return &ir.Assign{Lhs: read, Rhs: v}
		}
	case *ast.StarExpr:
		x := b.expr(e)
		un, ok := x.(*ir.Unary)
		if !ok || un.Op != token.MUL {
			return nil, x, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: x, Rhs: v} }
		}
		pre, ptr := b.hoist(un.X, e.X.Pos(), "p")
		read := &ir.Unary{Op: un.Op, X: ptr, Type: un.Type}
		return pre, read, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: read, Rhs: v} }
	}
	x := b.expr(e)
	return nil, x, func(v ir.Expr) ir.Stmt { return &ir.Assign{Lhs: x, Rhs: v} }
}

// lvalueStaged prepares a location of a parallel assignment. Go evaluates the
// operands of each location before the first write, thus the key of an index
// and the pointer of a deref each land in a variable, and a write to a name
// on the left does not move a location on the right of it. The base of a
// field keeps its name, because a variable is stable storage.
func (b *builder) lvalueStaged(e ast.Expr) (pre []ir.Stmt, set func(ir.Expr) ir.Stmt) {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return b.lvalueStaged(e.X)
	case *ast.IndexExpr:
		x := b.expr(e)
		switch x := x.(type) {
		case *ir.Index:
			// A parallel assignment stages the slice value itself, thus a
			// write of the variable in the same statement misses it. An
			// array writes through the variable, and the name stays.
			hoistBase := b.hoist
			if _, ok := typeset.Core(x.X.GoType()).(*types.Slice); ok || typeset.ArrayOrSlice(x.X.GoType()) {
				hoistBase = b.forceHoist
			}
			pre1, base := hoistBase(x.X, e.X.Pos(), "b")
			index := x.Index
			if _, ok := typeset.Core(x.X.GoType()).(*types.Map); ok {
				index = b.copyPoint(index)
			}
			pre2, key := b.forceHoist(index, e.Index.Pos(), "k")
			read := &ir.Index{Pos: x.Pos, X: base, Index: key, Type: x.Type}
			return append(pre1, pre2...), func(v ir.Expr) ir.Stmt {
				return &ir.Assign{Lhs: read, Rhs: v}
			}
		case *ir.MapRead:
			pre1, base := b.forceHoist(x.X, e.X.Pos(), "b")
			pre2, key := b.forceHoist(x.Key, e.Index.Pos(), "k")
			read := &ir.MapRead{X: base, Key: key, Type: x.Type}
			return append(pre1, pre2...), func(v ir.Expr) ir.Stmt {
				return &ir.Assign{Lhs: read, Rhs: v}
			}
		}
	case *ast.SelectorExpr:
		x := b.expr(e)
		if field, ok := x.(*ir.Field); ok {
			// A field of a pointer stages the pointer value itself, thus a
			// write of the variable in the same statement misses it. A field
			// of a struct variable writes through the variable, and the name
			// stays.
			hoistBase := b.hoist
			if _, ok := typeset.Core(b.typeOf(e.X)).(*types.Pointer); ok {
				hoistBase = b.forceHoist
			}
			pre, base := hoistBase(field.X, e.X.Pos(), "b")
			read := &ir.Field{X: base, Name: field.Name, Type: field.Type}
			return pre, func(v ir.Expr) ir.Stmt {
				return &ir.Assign{Lhs: read, Rhs: v}
			}
		}
	case *ast.StarExpr:
		x := b.expr(e)
		if un, ok := x.(*ir.Unary); ok {
			pre, p := b.forceHoist(un.X, e.X.Pos(), "p")
			read := &ir.Unary{Op: un.Op, X: p, Type: un.Type}
			return pre, func(v ir.Expr) ir.Stmt {
				return &ir.Assign{Lhs: read, Rhs: v}
			}
		}
	}
	pre, _, set = b.lvalue(e)
	return pre, set
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
	if b.blocks.SyncType(want) && !b.blocks.SyncType(got) {
		return &ir.Conv{X: x, From: got, Type: want, Sync: true}
	}
	if types.Identical(got, want) {
		return x
	}
	if _, fromStruct := typeset.Core(got).(*types.Struct); fromStruct {
		if _, toStruct := typeset.Core(want).(*types.Struct); toStruct {
			return &ir.Conv{X: x, From: got, Type: want}
		}
	}
	for _, typ := range []types.Type{got, want} {
		if _, parameter := types.Unalias(typ).(*types.TypeParam); parameter {
			if _, pointer := typeset.Core(typ).(*types.Pointer); pointer {
				return &ir.Conv{X: x, From: got, Type: want}
			}
		}
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
		return b.methodInst(e, b.expr(e.X))
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
	name := obj.Name()
	if n, ok := b.names[obj]; ok {
		name = n
	}
	id := &ir.Ident{Name: name, Obj: obj, Type: obj.Type(), Boxed: b.boxed(obj)}
	if targs, sig := b.funcInstArgs(e, obj); targs != nil {
		return &ir.FuncInst{Fun: id, TypeArgs: targs, Type: sig}
	}
	return id
}

// funcInstArgs gives the type arguments and the instantiated signature of an
// instance of a generic function. It gives nil for any other name. A call
// unwraps the node again, thus only a value of such an instance keeps it.
func (b *builder) funcInstArgs(e *ast.Ident, obj types.Object) ([]types.Type, *types.Signature) {
	fn, ok := obj.(*types.Func)
	if !ok {
		return nil, nil
	}
	decl, ok := fn.Origin().Type().(*types.Signature)
	if !ok || decl.Recv() != nil || decl.TypeParams().Len() == 0 {
		return nil, nil
	}
	inst, ok := b.info.Instances[e]
	if !ok || inst.TypeArgs == nil || inst.TypeArgs.Len() != decl.TypeParams().Len() {
		return nil, nil
	}
	sig, ok := inst.Type.(*types.Signature)
	if !ok {
		return nil, nil
	}
	out := make([]types.Type, inst.TypeArgs.Len())
	for i := range out {
		out[i] = inst.TypeArgs.At(i)
	}
	return out, sig
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
		out := b.fieldPath(x, sel)
		for at, ok := out.(*ir.Field); ok; at, ok = at.X.(*ir.Field) {
			at.Pos = e.Sel.Pos()
		}
		return out
	case types.MethodVal:
		x := b.expr(e.X)
		if x == nil {
			return nil
		}
		fn, _ := sel.Obj().(*types.Func)
		return &ir.Method{Recv: b.embeddedPath(x, sel), Obj: fn, Type: sel.Type(), Pos: e.Sel.Pos(), Ordinal: b.literalOrdinals[e]}
	}
	if sel.Kind() == types.MethodExpr {
		fn, _ := sel.Obj().(*types.Func)
		return &ir.MethodExpr{Obj: fn, Recv: sel.Recv(), Type: sel.Type(), Pos: e.Sel.Pos(), Ordinal: b.literalOrdinals[e]}
	}
	b.fail(e.Pos(), "the selector %s has no Dart form", e.Sel.Name)
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

// methodInst fills the type arguments of an instance of a generic method, and
// the type becomes the instantiated signature. Any other instance passes
// through, because a call reads its own type arguments.
func (b *builder) methodInst(e ast.Expr, x ir.Expr) ir.Expr {
	targs := b.typeArgs(e)
	if len(targs) == 0 {
		return x
	}
	switch m := x.(type) {
	case *ir.Method:
		m.TypeArgs = targs
		m.Type = b.typeOf(e)
	case *ir.MethodExpr:
		m.TypeArgs = targs
		m.Type = b.typeOf(e)
	}
	return x
}

func (b *builder) index(e *ast.IndexExpr) ir.Expr {
	if tv, ok := b.info.Types[e.Index]; ok && tv.IsType() {
		// The expression names an instance of a generic function, of a
		// generic type, or of a generic method. A method instance keeps its
		// type arguments, and a call reads them from the type checker.
		return b.methodInst(e, b.expr(e.X))
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
	return &ir.Index{Pos: e.Lbrack, X: x, Index: key, Type: b.typeOf(e)}
}

func (b *builder) slice(e *ast.SliceExpr) ir.Expr {
	out := &ir.Slice{Pos: e.Lbrack, X: b.expr(e.X), Type: b.typeOf(e)}
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
	out := &ir.FuncLit{Pos: e.Pos(), Ordinal: b.literalOrdinals[e], Type: sig, Async: b.blocks.Blocks(e), End: e.Body.Rbrace}
	prev := b.fn
	prevPending := b.pending
	prevGoto := b.gotoLabels
	prevLabelYields, prevYield, prevYieldGotos := b.labelYields, b.currentYield, b.yieldGotos
	prevLoops, prevRetFlag, prevRetVars, prevBranchFlags := b.loops, b.retFlag, b.retVars, b.branchFlags
	b.gotoLabels = gotoTargets(e.Body)
	b.labelYields, b.currentYield, b.yieldGotos = b.rangeLabels(e.Body), nil, nil
	b.loops, b.retFlag, b.retVars, b.branchFlags = nil, "", nil, nil
	labels := b.gotoLabels
	b.pending = nil
	// The body of a literal is a function of its own, thus the order pass runs
	// over it also inside the header of a loop.
	prevNoOrder := b.noOrder
	b.noOrder = 0
	inner := &ir.Func{Params: b.varsOf(sig.Params()), Results: b.results(sig)}
	b.fn = inner
	out.Params = inner.Params
	out.Results = inner.Results
	out.Body = b.block(e.Body)
	out.Body.List = append(b.returnState(), out.Body.List...)
	out.Defers = inner.Defers
	out.Captured = b.litCaptures(e)
	if b.litIndependent(e) {
		position := b.fset.PositionFor(e.Pos(), false)
		out.StaticName = fmt.Sprintf("$literal%x_%d_%d", filepath.Base(position.Filename), position.Line, position.Column)
	}
	b.fn = prev
	b.pending = prevPending
	b.gotoLabels = prevGoto
	b.labelYields, b.currentYield, b.yieldGotos = prevLabelYields, prevYield, prevYieldGotos
	b.loops, b.retFlag, b.retVars, b.branchFlags = prevLoops, prevRetFlag, prevRetVars, prevBranchFlags
	b.noOrder = prevNoOrder
	if !out.Async && len(labels) == 0 {
		return out
	}
	out.Machine = b.flatten(out.Body, out.Params, nil, out.Async, out.Defers,
		inner.DeferBlocks, out.Results, b.frameClass(e.Pos()), e.Pos())
	addResults(out.Machine, out.Results)
	if !out.Async {
		return out
	}
	local := b.litName(e.Pos())
	b.pending = append(b.pending, &ir.LocalFunc{Name: local, Lit: out})
	return &ir.Ident{Name: local, Type: sig}
}

func zeroConst(t types.Type) *ir.Const {
	return &ir.Const{Value: constant.MakeInt64(0), Type: t}
}

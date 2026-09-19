package normalize

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
)

// rangeStmt rewrites a range loop for the type of its container.
//
// A type parameter reads its core type. Go gives such a parameter a core type
// when each term of its type set shares 1 underlying type, and a range loop
// then reads that type.
func (b *builder) rangeStmt(s *ast.RangeStmt, label string) []ir.Stmt {
	var declarations []ir.Stmt
	if b.sharedLoopVars && s.Tok == token.DEFINE {
		for _, target := range []ast.Expr{s.Key, s.Value} {
			if target == nil || isBlank(target) {
				continue
			}
			obj := b.info.Defs[target.(*ast.Ident)].(*types.Var)
			v := b.declareLocal(obj, obj.Name())
			declarations = append(declarations, &ir.Declare{Vars: []*ir.Var{v}, Init: &ir.Zero{Type: obj.Type()}})
		}
	}
	return append(declarations, b.rangeLoop(s, label)...)
}

func (b *builder) rangeLoop(s *ast.RangeStmt, label string) []ir.Stmt {
	t := typeset.Core(b.typeOf(s.X))
	if p, ok := t.(*types.Pointer); ok {
		if _, isArray := typeset.Core(p.Elem()).(*types.Array); isArray {
			t = typeset.Core(p.Elem())
		}
	}
	if sig, ok := t.(*types.Signature); ok {
		return b.rangeFunc(s, label, sig)
	}
	b.pushLoop(loopPlain, label)
	defer b.popLoop()
	switch t := t.(type) {
	case *types.Slice, *types.Array:
		return b.rangeList(s, label)
	case *types.Basic:
		if t.Info()&types.IsString != 0 {
			return b.rangeString(s, label)
		}
		if t.Info()&types.IsInteger != 0 {
			return b.rangeInt(s, label)
		}
	case *types.Map:
		return b.rangeMap(s, label)
	case *types.Chan:
		return b.rangeChan(s, label, t.Elem())
	}
	b.fail(s.Pos(), "a range loop over %s is outside the subset", b.typeOf(s.X))
	return nil
}

// lenIsConst reports whether the length of the range expression is constant.
// A call and a channel receive each make it variable, and Go then evaluates
// the expression 1 time.
func lenIsConst(e ast.Expr) bool {
	ok := true
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			ok = false
		case *ast.UnaryExpr:
			if n.Op == token.ARROW {
				ok = false
			}
		}
		return ok
	})
	return ok
}

func intConst(n int64) *ir.Const {
	return &ir.Const{Value: constant.MakeInt64(n), Type: types.Typ[types.Int]}
}

func (b *builder) intVar(pos token.Pos, role string) (*ir.Var, *ir.Ident) {
	v := b.tempVar(pos, role, types.Typ[types.Int])
	return v, &ir.Ident{Name: v.Name, Type: types.Typ[types.Int]}
}

// bind declares per-iteration variables or assigns the shared loop variables.
func (b *builder) bind(s *ast.RangeStmt, target ast.Expr, value ir.Expr) []ir.Stmt {
	if target == nil || isBlank(target) {
		return nil
	}
	if s.Tok == token.DEFINE {
		if b.sharedLoopVars {
			return []ir.Stmt{&ir.Assign{Lhs: b.expr(target), Rhs: b.copyPoint(b.convert(value, b.typeOf(target)))}}
		}
		id, _ := target.(*ast.Ident)
		obj, _ := b.info.Defs[id].(*types.Var)
		if obj == nil {
			return nil
		}
		v := b.declareLocal(obj, obj.Name())
		return []ir.Stmt{&ir.Declare{Vars: []*ir.Var{v}, Init: b.convert(value, obj.Type())}}
	}
	pre, _, set := b.lvalue(target)
	return append(pre, set(b.copyPoint(b.convert(value, b.typeOf(target)))))
}

// bindBoth writes the key and the value of 1 pass of a range loop. Go
// evaluates the operands of the 2 locations first, thus the location of the
// value reads the key before the write of the key. `for i, x[i] = range y`
// writes the element that the old key names.
func (b *builder) bindBoth(s *ast.RangeStmt, kv, vv ir.Expr) []ir.Stmt {
	if s.Value == nil || isBlank(s.Value) || s.Tok == token.DEFINE {
		out := b.bind(s, s.Key, kv)
		if s.Value != nil && vv != nil {
			out = append(out, b.bind(s, s.Value, vv)...)
		}
		return out
	}
	pre, set := b.lvalueStaged(s.Value)
	out := append(pre, b.bind(s, s.Key, kv)...)
	return append(out, set(b.copyPoint(b.convert(vv, b.typeOf(s.Value)))))
}

func (b *builder) rangeList(s *ast.RangeStmt, label string) []ir.Stmt {
	// Go reads the range expression 0 times when at most 1 iteration
	// variable stands and the length is constant. A nil pointer to an array
	// ranges by the length of the type.
	if (s.Value == nil || isBlank(s.Value)) && lenIsConst(s.X) {
		u := typeset.Core(b.typeOf(s.X))
		if ptr, ok := u.(*types.Pointer); ok {
			u = typeset.Core(ptr.Elem())
		}
		if a, ok := u.(*types.Array); ok {
			iv, i := b.intVar(s.Pos(), "i")
			body := &ir.Block{}
			body.List = append(body.List, b.bindBoth(s, i, nil)...)
			body.List = append(body.List, b.block(s.Body).List...)
			return []ir.Stmt{&ir.For{
				Init:  &ir.Declare{Vars: []*ir.Var{iv}, Init: intConst(0)},
				Cond:  &ir.Binary{Op: token.LSS, X: i, Y: intConst(a.Len()), Type: types.Typ[types.Bool]},
				Post:  &ir.Assign{Lhs: i, Rhs: &ir.Binary{Op: token.ADD, X: i, Y: intConst(1), Type: types.Typ[types.Int]}},
				Body:  body,
				Label: label,
			}}
		}
	}
	x := b.expr(s.X)
	// Go reads a range expression of an array type as a value, thus the walk
	// takes a copy. A write of the array inside the body then does not reach
	// the value of a later pass. A slice and a pointer to an array each share
	// the storage, and neither one takes a copy. copyPoint gives a fresh
	// value back with no copy, thus the result of a call costs nothing.
	if s.Value != nil && !isBlank(s.Value) {
		if _, ok := typeset.Core(b.typeOf(s.X)).(*types.Array); ok {
			x = b.copyPoint(x)
		}
	}
	pre, container := b.forceHoist(x, s.X.Pos(), "range")
	iv, i := b.intVar(s.Pos(), "i")
	lv, length := b.intVar(s.Pos(), "n")
	pre = append(pre,
		&ir.Declare{Vars: []*ir.Var{lv}, Init: &ir.Builtin{Name: "len", Args: []ir.Expr{container}, Type: types.Typ[types.Int]}})
	elem := elemType(container.GoType())
	body := &ir.Block{}
	body.List = append(body.List, b.bindBoth(s, i, b.copyPoint(&ir.Index{X: container, Index: i, Type: elem}))...)
	body.List = append(body.List, b.block(s.Body).List...)
	loop := &ir.For{
		Init:  &ir.Declare{Vars: []*ir.Var{iv}, Init: intConst(0)},
		Cond:  &ir.Binary{Op: token.LSS, X: i, Y: length, Type: types.Typ[types.Bool]},
		Post:  &ir.Assign{Lhs: i, Rhs: &ir.Binary{Op: token.ADD, X: i, Y: intConst(1), Type: types.Typ[types.Int]}},
		Body:  body,
		Label: label,
	}
	return append(pre, loop)
}

func (b *builder) rangeInt(s *ast.RangeStmt, label string) []ir.Stmt {
	pre, limit := b.hoist(b.convert(b.expr(s.X), types.Typ[types.Int]), s.X.Pos(), "n")
	iv, i := b.intVar(s.Pos(), "i")
	body := &ir.Block{}
	body.List = append(body.List, b.bind(s, s.Key, i)...)
	body.List = append(body.List, b.block(s.Body).List...)
	loop := &ir.For{
		Init:  &ir.Declare{Vars: []*ir.Var{iv}, Init: intConst(0)},
		Cond:  &ir.Binary{Op: token.LSS, X: i, Y: limit, Type: types.Typ[types.Bool]},
		Post:  &ir.Assign{Lhs: i, Rhs: &ir.Binary{Op: token.ADD, X: i, Y: intConst(1), Type: types.Typ[types.Int]}},
		Body:  body,
		Label: label,
	}
	return append(pre, loop)
}

// rangeString walks the runes of a string. The size of the last rune moves the
// index, thus a continue statement also moves it.
func (b *builder) rangeString(s *ast.RangeStmt, label string) []ir.Stmt {
	pre, str := b.forceHoist(b.expr(s.X), s.X.Pos(), "range")
	iv, i := b.intVar(s.Pos(), "i")
	sv, size := b.intVar(s.Pos(), "size")
	rv := b.tempVar(s.Pos(), "rune", types.Typ[types.Rune])
	runeRead := &ir.Ident{Name: rv.Name, Type: types.Typ[types.Rune]}
	pre = append(pre,
		&ir.Declare{Vars: []*ir.Var{iv}, Init: intConst(0)},
		&ir.Declare{Vars: []*ir.Var{sv}, Init: intConst(0)},
		&ir.Declare{Vars: []*ir.Var{rv}, Init: intConst(0)})
	body := &ir.Block{}
	body.List = append(body.List, &ir.AssignMulti{
		Lhs: []ir.Expr{runeRead, size},
		Rhs: &ir.Builtin{Name: "decoderune", Args: []ir.Expr{str, i}, Type: types.Typ[types.Int]},
	})
	body.List = append(body.List, b.bindBoth(s, i, runeRead)...)
	body.List = append(body.List, b.block(s.Body).List...)
	loop := &ir.For{
		Cond: &ir.Binary{Op: token.LSS, X: i,
			Y:    &ir.Builtin{Name: "len", Args: []ir.Expr{str}, Type: types.Typ[types.Int]},
			Type: types.Typ[types.Bool]},
		Post:  &ir.Assign{Lhs: i, Rhs: &ir.Binary{Op: token.ADD, X: i, Y: size, Type: types.Typ[types.Int]}},
		Body:  body,
		Label: label,
	}
	return append(pre, loop)
}

// rangeMap walks the entries of a map with the iterator of the runtime. The
// iterator jumps over an entry that the body removes, and it starts in
// constant time, thus n loops over a map that shrinks stay linear. The value
// reads the entry at the visit, thus a NaN key needs no snapshot.
func (b *builder) rangeMap(s *ast.RangeStmt, label string) []ir.Stmt {
	m := mapType(b.typeOf(s.X))
	pre, container := b.forceHoist(b.expr(s.X), s.X.Pos(), "range")
	itType := &ir.MapIterType{Map: m}
	itv := b.tempVar(s.Pos(), "it", itType)
	it := &ir.Ident{Name: itv.Name, Type: itType}
	pre = append(pre, &ir.Declare{Vars: []*ir.Var{itv},
		Init: &ir.Builtin{Name: "mapiter", Args: []ir.Expr{container}, Type: itType}})
	key := b.copyPoint(&ir.Builtin{Name: "iterkey", Args: []ir.Expr{it}, Type: m.Key()})
	value := b.copyPoint(&ir.Builtin{Name: "iterval", Args: []ir.Expr{it}, Type: m.Elem()})
	body := &ir.Block{}
	body.List = append(body.List, b.bindBoth(s, key, value)...)
	body.List = append(body.List, b.block(s.Body).List...)
	loop := &ir.For{
		Cond:  &ir.Builtin{Name: "iternext", Args: []ir.Expr{it}, Type: types.Typ[types.Bool]},
		Body:  body,
		Label: label,
	}
	return append(pre, loop)
}

// rangeChan reads a channel until the channel closes.
func (b *builder) rangeChan(s *ast.RangeStmt, label string, elem types.Type) []ir.Stmt {
	pre, container := b.forceHoist(b.expr(s.X), s.X.Pos(), "range")
	value := b.tempVar(s.Pos(), "v", elem)
	okVar := b.tempVar(s.Pos(), "ok", types.Typ[types.Bool])
	ok := &ir.Ident{Name: okVar.Name, Type: types.Typ[types.Bool]}
	body := &ir.Block{}
	body.List = append(body.List, &ir.Declare{
		Vars: []*ir.Var{value, okVar},
		Init: &ir.Recv{Chan: container, Comma: true,
			Type: types.NewTuple(anonVar(elem), anonVar(types.Typ[types.Bool]))},
	})
	body.List = append(body.List, &ir.If{
		Cond: &ir.Unary{Op: token.NOT, X: ok, Type: types.Typ[types.Bool]},
		Then: &ir.Block{List: []ir.Stmt{&ir.Break{Label: label}}},
	})
	body.List = append(body.List, b.bind(s, s.Key,
		b.copyPoint(&ir.Ident{Name: value.Name, Type: elem}))...)
	body.List = append(body.List, b.block(s.Body).List...)
	return append(pre, &ir.For{Body: body, Label: label})
}

func elemType(t types.Type) types.Type {
	switch u := typeset.Core(t).(type) {
	case *types.Slice:
		return u.Elem()
	case *types.Array:
		return u.Elem()
	case *types.Pointer:
		if a, ok := typeset.Core(u.Elem()).(*types.Array); ok {
			return a.Elem()
		}
	case *types.Map:
		return u.Elem()
	case *types.Basic:
		return types.Typ[types.Byte]
	}
	return nil
}

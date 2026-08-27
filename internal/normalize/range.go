package normalize

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/eddyw/godart/internal/ir"
)

// rangeStmt rewrites a range loop for the type of its container.
func (b *builder) rangeStmt(s *ast.RangeStmt, label string) []ir.Stmt {
	t := types.Unalias(b.typeOf(s.X)).Underlying()
	if p, ok := t.(*types.Pointer); ok {
		if _, isArray := types.Unalias(p.Elem()).Underlying().(*types.Array); isArray {
			t = types.Unalias(p.Elem()).Underlying()
		}
	}
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
	}
	b.fail(s.Pos(), "a range loop over %s is outside the subset", b.typeOf(s.X))
	return nil
}

func intConst(n int64) *ir.Const {
	return &ir.Const{Value: constant.MakeInt64(n), Type: types.Typ[types.Int]}
}

func (b *builder) intVar(pos token.Pos, role string) (*ir.Var, *ir.Ident) {
	v := b.tempVar(pos, role, types.Typ[types.Int])
	return v, &ir.Ident{Name: v.Name, Type: types.Typ[types.Int]}
}

// bind writes the key or the value of a range loop into its variable. The
// variable belongs to 1 iteration, thus the stage declares it in the body.
func (b *builder) bind(s *ast.RangeStmt, target ast.Expr, value ir.Expr) []ir.Stmt {
	if target == nil || isBlank(target) {
		return nil
	}
	if s.Tok == token.DEFINE {
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

func (b *builder) rangeList(s *ast.RangeStmt, label string) []ir.Stmt {
	pre, container := b.hoist(b.expr(s.X), s.X.Pos(), "range")
	iv, i := b.intVar(s.Pos(), "i")
	lv, length := b.intVar(s.Pos(), "n")
	pre = append(pre,
		&ir.Declare{Vars: []*ir.Var{lv}, Init: &ir.Builtin{Name: "len", Args: []ir.Expr{container}, Type: types.Typ[types.Int]}})
	elem := elemType(container.GoType())
	body := &ir.Block{}
	body.List = append(body.List, b.bind(s, s.Key, i)...)
	body.List = append(body.List, b.bind(s, s.Value, b.copyPoint(&ir.Index{X: container, Index: i, Type: elem}))...)
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
	pre, str := b.hoist(b.expr(s.X), s.X.Pos(), "range")
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
	body.List = append(body.List, b.bind(s, s.Key, i)...)
	body.List = append(body.List, b.bind(s, s.Value, runeRead)...)
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

// rangeMap walks the keys of a map. Go does not give a key that the body
// removes, thus the loop reads the map again for each key.
func (b *builder) rangeMap(s *ast.RangeStmt, label string) []ir.Stmt {
	m := mapType(b.typeOf(s.X))
	pre, container := b.hoist(b.expr(s.X), s.X.Pos(), "range")
	keyList := types.NewSlice(m.Key())
	kv := b.tempVar(s.Pos(), "keys", keyList)
	keys := &ir.Ident{Name: kv.Name, Type: keyList}
	pre = append(pre, &ir.Declare{Vars: []*ir.Var{kv},
		Init: &ir.Builtin{Name: "mapkeys", Args: []ir.Expr{container}, Type: keyList}})
	iv, i := b.intVar(s.Pos(), "i")
	key := &ir.Index{X: keys, Index: i, Type: m.Key()}
	body := &ir.Block{}
	body.List = append(body.List, &ir.If{
		Cond: &ir.Unary{Op: token.NOT, Type: types.Typ[types.Bool],
			X: &ir.Builtin{Name: "maphas", Args: []ir.Expr{container, key}, Type: types.Typ[types.Bool]}},
		Then: &ir.Block{List: []ir.Stmt{&ir.Continue{}}},
	})
	body.List = append(body.List, b.bind(s, s.Key, key)...)
	body.List = append(body.List, b.bind(s, s.Value,
		b.copyPoint(&ir.Index{X: container, Index: key, Type: m.Elem()}))...)
	body.List = append(body.List, b.block(s.Body).List...)
	loop := &ir.For{
		Init: &ir.Declare{Vars: []*ir.Var{iv}, Init: intConst(0)},
		Cond: &ir.Binary{Op: token.LSS, X: i,
			Y:    &ir.Builtin{Name: "len", Args: []ir.Expr{keys}, Type: types.Typ[types.Int]},
			Type: types.Typ[types.Bool]},
		Post:  &ir.Assign{Lhs: i, Rhs: &ir.Binary{Op: token.ADD, X: i, Y: intConst(1), Type: types.Typ[types.Int]}},
		Body:  body,
		Label: label,
	}
	return append(pre, loop)
}

func elemType(t types.Type) types.Type {
	switch u := types.Unalias(t).Underlying().(type) {
	case *types.Slice:
		return u.Elem()
	case *types.Array:
		return u.Elem()
	case *types.Pointer:
		if a, ok := types.Unalias(u.Elem()).Underlying().(*types.Array); ok {
			return a.Elem()
		}
	case *types.Map:
		return u.Elem()
	case *types.Basic:
		return types.Typ[types.Byte]
	}
	return nil
}

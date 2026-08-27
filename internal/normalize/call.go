package normalize

import (
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/eddyw/godart/internal/ir"
)

func (b *builder) call(e *ast.CallExpr) ir.Expr {
	if tv, ok := b.info.Types[e.Fun]; ok && tv.IsType() {
		if len(e.Args) != 1 {
			b.fail(e.Pos(), "a conversion takes 1 argument")
			return nil
		}
		x := b.expr(e.Args[0])
		if x == nil {
			return nil
		}
		return &ir.Conv{X: x, From: b.typeOf(e.Args[0]), Type: tv.Type}
	}
	if id := calleeIdent(e.Fun); id != nil {
		if bi, ok := b.info.Uses[id].(*types.Builtin); ok {
			return b.builtin(e, bi.Name())
		}
	}
	fun := b.expr(e.Fun)
	if fun == nil {
		return nil
	}
	sig := signatureOf(b.typeOf(e.Fun))
	return &ir.Call{
		Fun:      fun,
		Args:     b.args(e, sig),
		Type:     b.typeOf(e),
		Spread:   e.Ellipsis.IsValid(),
		Variadic: sig != nil && sig.Variadic(),
	}
}

func calleeIdent(e ast.Expr) *ast.Ident {
	for {
		switch x := e.(type) {
		case *ast.ParenExpr:
			e = x.X
		case *ast.Ident:
			return x
		default:
			return nil
		}
	}
}

func signatureOf(t types.Type) *types.Signature {
	sig, _ := types.Unalias(t).Underlying().(*types.Signature)
	return sig
}

func (b *builder) args(e *ast.CallExpr, sig *types.Signature) []ir.Expr {
	out := make([]ir.Expr, len(e.Args))
	for i, a := range e.Args {
		out[i] = b.copyPoint(b.exprAs(a, b.paramType(sig, i, e.Ellipsis.IsValid())))
	}
	return out
}

// paramType gives the type of the parameter that takes argument i. The last
// parameter of a variadic function takes the element type, except when the call
// spreads a slice.
func (b *builder) paramType(sig *types.Signature, i int, spread bool) types.Type {
	if sig == nil {
		return nil
	}
	params := sig.Params()
	last := params.Len() - 1
	if sig.Variadic() && i >= last && !spread {
		slice, ok := types.Unalias(params.At(last).Type()).Underlying().(*types.Slice)
		if ok {
			return slice.Elem()
		}
	}
	if i >= params.Len() {
		return nil
	}
	return params.At(i).Type()
}

func (b *builder) builtin(e *ast.CallExpr, fn string) ir.Expr {
	t := b.typeOf(e)
	switch fn {
	case "make", "new":
		out := &ir.Builtin{Name: fn, Type: t}
		for _, a := range e.Args[1:] {
			out.Args = append(out.Args, b.convert(b.expr(a), types.Typ[types.Int]))
		}
		if fn == "new" && len(e.Args) == 1 {
			if tv, ok := b.info.Types[e.Args[0]]; ok && !tv.IsType() {
				out.Args = append(out.Args, b.expr(e.Args[0]))
			}
		}
		return out
	case "append":
		out := &ir.Builtin{Name: fn, Type: t, Spread: e.Ellipsis.IsValid()}
		slice, _ := types.Unalias(t).Underlying().(*types.Slice)
		out.Args = append(out.Args, b.expr(e.Args[0]))
		for _, a := range e.Args[1:] {
			if out.Spread {
				out.Args = append(out.Args, b.expr(a))
				continue
			}
			out.Args = append(out.Args, b.copyPoint(b.exprAs(a, slice.Elem())))
		}
		return out
	case "len", "cap", "copy", "delete", "clear", "panic", "print", "println",
		"min", "max", "close", "complex", "real", "imag":
		out := &ir.Builtin{Name: fn, Type: t}
		for _, a := range e.Args {
			out.Args = append(out.Args, b.expr(a))
		}
		return out
	}
	b.fail(e.Pos(), "the builtin %s is outside the subset", fn)
	return nil
}

func (b *builder) composite(e *ast.CompositeLit) ir.Expr {
	t := b.typeOf(e)
	switch u := types.Unalias(t).Underlying().(type) {
	case *types.Struct:
		return b.structLit(e, t, u)
	case *types.Slice:
		return &ir.Composite{Elems: b.listLit(e, u.Elem()), Type: t}
	case *types.Array:
		return &ir.Composite{Elems: b.listLit(e, u.Elem()), Type: t}
	case *types.Map:
		out := &ir.Composite{Type: t}
		for _, el := range e.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				b.fail(el.Pos(), "a map literal needs a key for each element")
				continue
			}
			out.Keys = append(out.Keys, b.copyPoint(b.exprAs(kv.Key, u.Key())))
			out.Elems = append(out.Elems, b.copyPoint(b.exprAs(kv.Value, u.Elem())))
		}
		return out
	}
	b.fail(e.Pos(), "a literal of type %s is outside the subset", t)
	return nil
}

// listLit fills the elements of an array literal or of a slice literal. A key
// gives the index of the element, and a gap holds the zero value.
func (b *builder) listLit(e *ast.CompositeLit, elem types.Type) []ir.Expr {
	var out []ir.Expr
	next := 0
	for _, el := range e.Elts {
		if kv, ok := el.(*ast.KeyValueExpr); ok {
			tv, ok := b.info.Types[kv.Key]
			if !ok || tv.Value == nil {
				b.fail(kv.Pos(), "an index in a literal must be a constant")
				continue
			}
			i, _ := constantInt(tv)
			next = i
			el = kv.Value
		}
		for len(out) < next {
			out = append(out, &ir.Zero{Type: elem})
		}
		value := b.copyPoint(b.exprAs(el, elem))
		if len(out) == next {
			out = append(out, value)
		} else {
			out[next] = value
		}
		next++
	}
	return out
}

func (b *builder) structLit(e *ast.CompositeLit, t types.Type, st *types.Struct) ir.Expr {
	out := &ir.Composite{Type: t}
	if len(e.Elts) > 0 {
		if _, keyed := e.Elts[0].(*ast.KeyValueExpr); keyed {
			return b.keyedStructLit(e, t, st)
		}
	}
	for i, el := range e.Elts {
		if i >= st.NumFields() {
			break
		}
		out.Elems = append(out.Elems, b.copyPoint(b.exprAs(el, st.Field(i).Type())))
	}
	for i := len(out.Elems); i < st.NumFields(); i++ {
		out.Elems = append(out.Elems, &ir.Zero{Type: st.Field(i).Type()})
	}
	return out
}

// keyedStructLit fills a struct literal that names its fields. Go 1.27 allows a
// key that selects a field of an embedded struct, thus the key holds a path.
func (b *builder) keyedStructLit(e *ast.CompositeLit, t types.Type, st *types.Struct) ir.Expr {
	type entry struct {
		path  []int
		value ast.Expr
	}
	var entries []entry
	for _, el := range e.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			b.fail(el.Pos(), "a keyed literal needs a key for each element")
			continue
		}
		path := b.fieldPathOf(t, kv.Key)
		if path == nil {
			continue
		}
		entries = append(entries, entry{path: path, value: kv.Value})
	}
	var build func(st *types.Struct, structType types.Type, entries []entry) ir.Expr
	build = func(st *types.Struct, structType types.Type, entries []entry) ir.Expr {
		out := &ir.Composite{Type: structType}
		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			var nested []entry
			var direct ast.Expr
			for _, en := range entries {
				if en.path[0] != i {
					continue
				}
				if len(en.path) == 1 {
					direct = en.value
					continue
				}
				nested = append(nested, entry{path: en.path[1:], value: en.value})
			}
			switch {
			case direct != nil:
				out.Elems = append(out.Elems, b.copyPoint(b.exprAs(direct, field.Type())))
			case len(nested) > 0:
				inner := structOf(field.Type())
				if inner == nil {
					b.fail(e.Pos(), "the key path leaves a struct type")
					out.Elems = append(out.Elems, &ir.Zero{Type: field.Type()})
					continue
				}
				out.Elems = append(out.Elems, build(inner, field.Type(), nested))
			default:
				out.Elems = append(out.Elems, &ir.Zero{Type: field.Type()})
			}
		}
		return out
	}
	return build(st, t, entries)
}

// fieldPathOf resolves the key of a struct literal to a field index path. The
// key is a field name, or a selector that names a field of an embedded struct.
func (b *builder) fieldPathOf(t types.Type, key ast.Expr) []int {
	var names []string
	for {
		switch k := key.(type) {
		case *ast.Ident:
			names = append([]string{k.Name}, names...)
			key = nil
		case *ast.SelectorExpr:
			names = append([]string{k.Sel.Name}, names...)
			key = k.X
			continue
		default:
			b.fail(key.Pos(), "the key of a struct literal must be a field selector")
			return nil
		}
		break
	}
	var path []int
	current := t
	for _, n := range names {
		obj, index, _ := types.LookupFieldOrMethod(current, true, b.pkg, n)
		field, ok := obj.(*types.Var)
		if !ok || !field.IsField() {
			b.fail(key.Pos(), "the struct has no field %s", n)
			return nil
		}
		path = append(path, index...)
		current = field.Type()
	}
	return path
}

func constantInt(tv types.TypeAndValue) (int, bool) {
	if tv.Value == nil {
		return 0, false
	}
	n, ok := constant.Int64Val(constant.ToInt(tv.Value))
	return int(n), ok
}

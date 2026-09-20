package normalize

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
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
		from := b.typeOf(e.Args[0])
		// A conversion to the same type gives a copy of the value. Go accepts
		// such a conversion, and an alias of a struct type makes one. A struct
		// and an array need a real copy, because Go gives each one value
		// semantics. Stage 5 holds no form for a conversion between 2 struct
		// classes, thus the copy must stand here.
		if needsCopy(tv.Type) && types.Identical(from, tv.Type) {
			return &ir.Copy{X: x, Type: tv.Type}
		}
		return &ir.Conv{X: x, From: from, Type: tv.Type}
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
	if inst, ok := fun.(*ir.FuncInst); ok {
		fun = inst.Fun
	}
	sig := signatureOf(b.typeOf(e.Fun))
	out := &ir.Call{
		Pos:      e.Fun.Pos(),
		Fun:      fun,
		Type:     b.typeOf(e),
		Async:    b.blocks.CallAsync(e),
		Spread:   e.Ellipsis.IsValid(),
		Variadic: sig != nil && sig.Variadic(),
		TypeArgs: b.typeArgs(e.Fun),
	}
	if selector, ok := unparenExpr(e.Fun).(*ast.SelectorExpr); ok {
		out.Pos = selector.Sel.Pos()
	}
	if !b.spreadResults(out, e, sig) {
		out.Args = b.args(e, sig)
	}
	return out
}

// logMethods lists each method of the testing package that writes a log line.
// Go reads the source position of such a call from the stack, and the harness
// of the overlay reads no stack.
// typeArgs reads the type arguments of a call to a generic function. The type
// checker records them, also when the source gives none and inference finds
// them.
func (b *builder) typeArgs(fun ast.Expr) []types.Type {
	id := funcIdent(fun)
	if id == nil {
		return nil
	}
	inst, ok := b.info.Instances[id]
	if !ok || inst.TypeArgs == nil {
		return nil
	}
	// A method of a generic type also has an instance. Only the type
	// parameters of the function itself need a type argument here, thus the
	// list keeps the last ones.
	want := 0
	if fn, ok := b.info.Uses[id].(*types.Func); ok {
		if sig, ok := fn.Origin().Type().(*types.Signature); ok {
			want = sig.TypeParams().Len()
		}
	} else if _, ok := b.info.Uses[id].(*types.TypeName); ok {
		want = inst.TypeArgs.Len()
	}
	if want == 0 || want > inst.TypeArgs.Len() {
		return nil
	}
	out := make([]types.Type, want)
	start := inst.TypeArgs.Len() - want
	for i := range out {
		out[i] = inst.TypeArgs.At(start + i)
	}
	return out
}

// funcIdent finds the name of the called function.
func funcIdent(e ast.Expr) *ast.Ident {
	for {
		switch x := e.(type) {
		case *ast.ParenExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.IndexListExpr:
			e = x.X
		case *ast.SelectorExpr:
			return x.Sel
		case *ast.Ident:
			return x
		default:
			return nil
		}
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
	sig, _ := typeset.Core(t).(*types.Signature)
	return sig
}

func (b *builder) args(e *ast.CallExpr, sig *types.Signature) []ir.Expr {
	out := make([]ir.Expr, len(e.Args))
	for i, a := range e.Args {
		out[i] = b.copyPoint(b.exprAs(a, b.paramType(sig, i, e.Ellipsis.IsValid())))
	}
	return out
}

// spreadResults fills the arguments of a call whose only argument is a call
// with multiple results. Go passes each result as its own argument.
//
// The emitter writes a closure that takes the record of the results, thus the
// inner call runs 1 time. Each argument reads 1 field of the record.
func (b *builder) spreadResults(out *ir.Call, e *ast.CallExpr, sig *types.Signature) bool {
	if len(e.Args) != 1 || e.Ellipsis.IsValid() {
		return false
	}
	inner, ok := unparenExpr(e.Args[0]).(*ast.CallExpr)
	if !ok {
		return false
	}
	tuple, ok := b.typeOf(inner).(*types.Tuple)
	if !ok || tuple.Len() < 2 {
		return false
	}
	recv := b.temp(e.Pos(), "sr")
	args := make([]ir.Expr, tuple.Len())
	for i := range args {
		want := b.paramType(sig, i, e.Ellipsis.IsValid())
		if want == nil {
			want = tuple.At(i).Type()
		}
		read := &ir.Ident{
			Name: recv + ".$" + strconv.Itoa(i+1),
			Type: tuple.At(i).Type(),
		}
		args[i] = b.convert(read, want)
	}
	out.Args = args
	out.Results = b.expr(inner)
	out.ResultsVar = recv
	return true
}

// spreadBuiltin fills the arguments of a builtin whose only argument is a call
// with multiple results, for example copy(f()). The form follows spreadResults.
func (b *builder) spreadBuiltin(out *ir.Builtin, e *ast.CallExpr) bool {
	if len(e.Args) != 1 || e.Ellipsis.IsValid() {
		return false
	}
	inner, ok := unparenExpr(e.Args[0]).(*ast.CallExpr)
	if !ok {
		return false
	}
	tuple, ok := b.typeOf(inner).(*types.Tuple)
	if !ok || tuple.Len() < 2 {
		return false
	}
	recv := b.temp(e.Pos(), "sr")
	args := make([]ir.Expr, tuple.Len())
	for i := range args {
		args[i] = &ir.Ident{
			Name: recv + ".$" + strconv.Itoa(i+1),
			Type: tuple.At(i).Type(),
		}
	}
	out.Args = args
	out.Results = b.expr(inner)
	out.ResultsVar = recv
	return true
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
		slice, ok := typeset.Core(params.At(last).Type()).(*types.Slice)
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
		slice, _ := typeset.Core(t).(*types.Slice)
		if b.spreadBuiltin(out, e) {
			for i := 1; i < len(out.Args); i++ {
				out.Args[i] = b.copyPoint(b.convert(out.Args[i], slice.Elem()))
			}
			return out
		}
		out.Args = append(out.Args, b.expr(e.Args[0]))
		for _, a := range e.Args[1:] {
			if out.Spread {
				out.Args = append(out.Args, b.expr(a))
				continue
			}
			out.Args = append(out.Args, b.copyPoint(b.exprAs(a, slice.Elem())))
		}
		return out
	case "panic":
		return &ir.Builtin{Name: fn, Type: t,
			Args: []ir.Expr{b.convert(b.expr(e.Args[0]), anyType())}}
	case "len", "cap", "copy", "delete", "clear", "print", "println",
		"min", "max", "close", "complex", "real", "imag", "recover":
		out := &ir.Builtin{Name: fn, Type: t}
		if b.spreadBuiltin(out, e) {
			return out
		}
		for _, a := range e.Args {
			out.Args = append(out.Args, b.expr(a))
		}
		// A key of a map with interface keys takes the type of the key, and
		// each other type of key needs the conversion to it.
		if fn == "delete" {
			if m, ok := typeset.Core(b.typeOf(e.Args[0])).(*types.Map); ok {
				out.Args[1] = b.convert(out.Args[1], m.Key())
			}
		}
		return out
	}
	b.fail(e.Pos(), "the builtin %s is outside the subset", fn)
	return nil
}

func (b *builder) composite(e *ast.CompositeLit) ir.Expr {
	return b.compositeOf(e, b.typeOf(e))
}

func (b *builder) compositeOf(e *ast.CompositeLit, t types.Type) ir.Expr {
	switch u := typeset.Core(t).(type) {
	case *types.Struct:
		return b.structLit(e, t, u)
	case *types.Slice:
		return b.listLit(e, u.Elem(), t)
	case *types.Array:
		return b.listLit(e, u.Elem(), t)
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
	case *types.Pointer:
		// An element of a literal of a container of pointers omits the
		// address operator. The literal builds the element, and the address
		// wraps it, exactly as the written form does.
		inner := b.compositeOf(e, u.Elem())
		if inner == nil {
			return nil
		}
		return &ir.Unary{Op: token.AND, X: inner, Type: t}
	}
	b.fail(e.Pos(), "a literal of type %s is outside the subset", t)
	return nil
}

// listLit fills the elements of an array literal or of a slice literal. A key
// gives the index of the element, and a gap holds the zero value.
func (b *builder) listLit(e *ast.CompositeLit, elem types.Type, t types.Type) ir.Expr {
	values := make([]ir.Expr, len(e.Elts))
	slots := make([]int, len(e.Elts))
	pure := true
	ordered := true
	next := 0
	size := 0
	for k, el := range e.Elts {
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
		values[k] = b.copyPoint(b.exprAs(el, elem))
		slots[k] = next
		pure = pure && pureExpr(values[k])
		if next < size {
			ordered = false
		}
		if next+1 > size {
			size = next + 1
		}
		next++
	}
	if array, ok := typeset.Core(t).(*types.Array); ok {
		size = int(array.Len())
	}
	if size > len(values) {
		return &ir.Composite{Type: t, Elems: values, Indices: slots, Length: size}
	}
	out := &ir.Composite{Type: t}
	// Go evaluates the values in the order of the source, and the slots take
	// them by the index. The record holds the source order.
	if !pure && !ordered {
		elems, vars := padRecord(append([]ir.Expr{}, values...))
		tuple := types.NewTuple(vars...)
		recv := b.temp(e.Pos(), "cl")
		out.Results = &ir.Record{Elems: elems, Type: tuple}
		out.ResultsVar = recv
		for i := range values {
			values[i] = &ir.RecordField{
				X:     &ir.Ident{Name: recv, Type: tuple},
				Index: i,
				Type:  tuple.At(i).Type(),
			}
		}
	}
	out.Elems = make([]ir.Expr, size)
	for i := range out.Elems {
		out.Elems[i] = &ir.Zero{Type: elem}
	}
	for k := range values {
		if values[k] != nil {
			out.Elems[slots[k]] = values[k]
		}
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
	// Go evaluates the values in the order of the source, and the fields take
	// them by the keys. The record holds the source order.
	values := make([]ir.Expr, len(entries))
	paths := make([][]int, len(entries))
	pure := true
	for i, en := range entries {
		paths[i] = en.path
		values[i] = b.copyPoint(b.exprAs(en.value, leafFieldType(st, en.path)))
		pure = pure && pureExpr(values[i])
	}
	if !pure && len(entries) > 1 {
		elems, vars := padRecord(values)
		tuple := types.NewTuple(vars...)
		recv := b.temp(e.Pos(), "cl")
		rec := &ir.Record{Elems: elems, Type: tuple}
		values = make([]ir.Expr, len(entries))
		for i := range values {
			values[i] = &ir.RecordField{
				X:     &ir.Ident{Name: recv, Type: tuple},
				Index: i,
				Type:  tuple.At(i).Type(),
			}
		}
		out := b.buildKeyed(st, t, paths, values, e.Pos())
		if c, ok := out.(*ir.Composite); ok {
			c.Results = rec
			c.ResultsVar = recv
		}
		return out
	}
	return b.buildKeyed(st, t, paths, values, e.Pos())
}

// buildKeyed places each staged value at its field path.
func (b *builder) buildKeyed(st *types.Struct, t types.Type,
	paths [][]int, values []ir.Expr, pos token.Pos) ir.Expr {
	type slot struct {
		path []int
		ir   ir.Expr
	}
	slots := make([]slot, len(paths))
	for i := range paths {
		slots[i] = slot{path: paths[i], ir: values[i]}
	}
	var build func(st *types.Struct, structType types.Type, slots []slot) ir.Expr
	build = func(st *types.Struct, structType types.Type, slots []slot) ir.Expr {
		out := &ir.Composite{Type: structType}
		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			var nested []slot
			var direct ir.Expr
			for _, en := range slots {
				if en.path[0] != i {
					continue
				}
				if len(en.path) == 1 {
					direct = en.ir
					continue
				}
				nested = append(nested, slot{path: en.path[1:], ir: en.ir})
			}
			switch {
			case direct != nil:
				out.Elems = append(out.Elems, direct)
			case len(nested) > 0:
				inner := structOf(field.Type())
				if inner == nil {
					b.fail(pos, "the key path leaves a struct type")
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
	return build(st, t, slots)
}

// leafFieldType walks a field path to its type.
func leafFieldType(st *types.Struct, path []int) types.Type {
	t := st.Field(path[0]).Type()
	for _, i := range path[1:] {
		inner := structOf(t)
		if inner == nil {
			return t
		}
		t = inner.Field(i).Type()
	}
	return t
}

// pureExpr reports whether a second read of an expression gives the same
// value with no effect.
func pureExpr(e ir.Expr) bool {
	switch e.(type) {
	case *ir.Const, *ir.Zero, *ir.Ident:
		return true
	}
	return false
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
	current := typeset.Core(t)
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

func anyType() types.Type {
	return types.Universe.Lookup("any").Type()
}

func constantInt(tv types.TypeAndValue) (int, bool) {
	if tv.Value == nil {
		return 0, false
	}
	n, ok := constant.Int64Val(constant.ToInt(tv.Value))
	return int(n), ok
}

// wasmRecordFields names the field count of a record that dart2wasm rejects.
//
// dart2wasm writes 1 class for each shape of a record, and the hashCode of
// that class calls Object.hash with the shape and with each field. Object.hash
// holds 20 positional parameters, thus a record of 20 fields gives 21
// arguments. The type flow analysis of the Dart VM then stops with `type
// 'Call' is not a subtype of type 'Parameter' in type cast`. A record of 19
// fields passes, and a record of 21 fields passes too, because dart2wasm reads
// Object.hashAll above 20 fields.
const wasmRecordFields = 20

// padRecord builds the variable of each element of an ordering record, and it
// adds 1 element to a record of exactly 20 fields.
//
// No code reads the added element. The shape then stands outside the defect of
// dart2wasm above, and the ahead of time build of the Dart VM holds 1 more
// word for such a record.
func padRecord(values []ir.Expr) ([]ir.Expr, []*types.Var) {
	if len(values) == wasmRecordFields {
		values = append(values, &ir.Zero{Type: types.Typ[types.Bool]})
	}
	vars := make([]*types.Var, len(values))
	for i, v := range values {
		vars[i] = anonVar(v.GoType())
	}
	return values, vars
}

package emit

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
)

func isInterfaceType(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Interface)
	return ok
}

// writeDescriptors writes the run time description of each named type of the
// package. An interface value holds such a description and the value.
func (e *emitter) writeDescriptors() {
	scope := e.pkg.Scope()
	for _, n := range scope.Names() {
		obj, ok := scope.Lookup(n).(*types.TypeName)
		if !ok || obj.IsAlias() {
			continue
		}
		named, ok := types.Unalias(obj.Type()).(*types.Named)
		if !ok || isInterfaceType(named) {
			continue
		}
		e.descriptor(named, false)
		e.descriptor(named, true)
	}
}

func (e *emitter) descriptor(named *types.Named, pointer bool) {
	t := types.Type(named)
	variable := "$desc$" + name.Package(named.Obj().Name())
	if pointer {
		t = types.NewPointer(named)
		variable = "$descP$" + name.Package(named.Obj().Name())
	}
	set := types.NewMethodSet(t)
	entries := make([]string, 0, set.Len())
	for i := 0; i < set.Len(); i++ {
		entries = append(entries, e.adapter(named, set.At(i)))
	}
	sort.Strings(entries)
	e.line("final TypeDesc %s = TypeDesc('%s', <String, Function>{%s});",
		variable, types.TypeString(t, nil), strings.Join(entries, ", "))
}

// adapter writes 1 entry of a method table. The entry takes the value of the
// interface, thus each entry has the same shape of receiver.
func (e *emitter) adapter(named *types.Named, sel *types.Selection) string {
	fn, ok := sel.Obj().(*types.Func)
	if !ok {
		return ""
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return ""
	}
	params := make([]string, sig.Params().Len())
	args := make([]string, sig.Params().Len())
	for i := range params {
		args[i] = fmt.Sprintf("$a%d", i)
		params[i] = e.dartType(sig.Params().At(i).Type()) + " " + args[i]
	}
	recv := fmt.Sprintf("($r as %s)", e.dartType(named))
	index := sel.Index()
	current := types.Type(named)
	for _, i := range index[:len(index)-1] {
		st := structOf(current)
		if st == nil {
			e.fail("the method path of %s leaves a struct type", named)
			return ""
		}
		field := st.Field(i)
		recv += "." + name.Package(field.Name())
		current = field.Type()
	}
	call := fmt.Sprintf("%s.%s(%s)", recv, methodName(fn), strings.Join(args, ", "))
	if !isStructType(current) {
		owner := receiverTypeName(current)
		if owner == nil {
			e.fail("the method %s has no named receiver type", fn.Name())
			return ""
		}
		all := append([]string{recv}, args...)
		call = e.qualify(owner, methodFuncName(owner.Name(), fn.Name())) +
			"(" + strings.Join(all, ", ") + ")"
	}
	head := "Object? $r"
	if len(params) > 0 {
		head += ", " + strings.Join(params, ", ")
	}
	return fmt.Sprintf("'%s': (%s) => %s", fn.Name(), head, call)
}

func structOf(t types.Type) *types.Struct {
	u := types.Unalias(t).Underlying()
	if p, ok := u.(*types.Pointer); ok {
		u = types.Unalias(p.Elem()).Underlying()
	}
	st, _ := u.(*types.Struct)
	return st
}

// descriptorOf gives the Dart expression that describes a Go type.
func (e *emitter) descriptorOf(t types.Type) string {
	u := types.Unalias(t)
	if p, ok := u.(*types.Pointer); ok {
		if named, ok := types.Unalias(p.Elem()).(*types.Named); ok {
			return e.qualify(named.Obj(), "$descP$"+name.Package(named.Obj().Name()))
		}
	}
	if named, ok := u.(*types.Named); ok {
		return e.qualify(named.Obj(), "$desc$"+name.Package(named.Obj().Name()))
	}
	return fmt.Sprintf("const TypeDesc('%s', <String, Function>{})", types.TypeString(t, nil))
}

// interfaceMethods lists the method names of an interface type, in order.
func interfaceMethods(t types.Type) []string {
	iface, ok := types.Unalias(t).Underlying().(*types.Interface)
	if !ok {
		return nil
	}
	out := make([]string, iface.NumMethods())
	for i := range out {
		out[i] = iface.Method(i).Name()
	}
	sort.Strings(out)
	return out
}

func methodNameList(names []string) string {
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = "'" + n + "'"
	}
	return "<String>[" + strings.Join(parts, ", ") + "]"
}

// box puts a concrete value into an interface value. Go copies a struct at
// this point, because the interface holds a value and not a location.
func (e *emitter) box(x ir.Expr, from types.Type) string {
	text := e.expr(x)
	switch {
	case isStructType(from):
		text = e.deref(x) + ".$copy()"
	case isArrayType(from):
		text = e.copyArray(text, from)
	}
	return fmt.Sprintf("Iface(%s, %s)", e.descriptorOf(from), text)
}

// interfaceCall calls a method through an interface value. The call reads the
// method table of the descriptor.
func (e *emitter) interfaceCall(m *ir.Method, args []string) string {
	sig, ok := m.Obj.Type().(*types.Signature)
	if !ok {
		e.fail("the method %s has no signature", m.Obj.Name())
		return "null"
	}
	result := e.resultType(sig)
	if isPure(m.Recv) {
		recv := e.expr(m.Recv)
		params := []string{"Object?"}
		for i := 0; i < sig.Params().Len(); i++ {
			params = append(params, e.dartType(sig.Params().At(i).Type()))
		}
		all := append([]string{recv + "!.value"}, args...)
		return fmt.Sprintf("($method(%s, '%s') as %s Function(%s))(%s)",
			recv, m.Obj.Name(), result, strings.Join(params, ", "), strings.Join(all, ", "))
	}
	call := fmt.Sprintf("$call(%s, '%s', <Object?>[%s])",
		e.expr(m.Recv), m.Obj.Name(), strings.Join(args, ", "))
	if result == "void" {
		return call
	}
	return fmt.Sprintf("(%s as %s)", call, result)
}

// isPure reports whether a second read of the expression gives the same value
// and runs no side effect.
func isPure(x ir.Expr) bool {
	switch x := x.(type) {
	case *ir.Ident, *ir.Const, *ir.Zero, *ir.Blank:
		return true
	case *ir.Field:
		return isPure(x.X)
	}
	return false
}

// typeTest writes the test of a type assertion and of a type switch.
func (e *emitter) typeTest(x ir.Expr, target types.Type) string {
	if isInterfaceType(target) {
		return fmt.Sprintf("$implements(%s, %s)", e.expr(x), methodNameList(interfaceMethods(target)))
	}
	return fmt.Sprintf("$isType(%s, '%s')", e.expr(x), types.TypeString(target, nil))
}

func (e *emitter) typeAssert(x *ir.TypeAssert) string {
	if !x.Comma {
		if isInterfaceType(x.Assert) {
			return fmt.Sprintf("$assertIface(%s, %s, '%s')",
				e.expr(x.X), methodNameList(interfaceMethods(x.Assert)),
				types.TypeString(x.Assert, nil))
		}
		return fmt.Sprintf("($assertType(%s, '%s') as %s)",
			e.expr(x.X), types.TypeString(x.Assert, nil), e.dartType(x.Assert))
	}
	if isInterfaceType(x.Assert) {
		return fmt.Sprintf("(() { final $x = %s; if ($implements($x, %s)) return ($x, true); return (null, false); })()",
			e.expr(x.X), methodNameList(interfaceMethods(x.Assert)))
	}
	return fmt.Sprintf("(() { final $x = %s; if ($isType($x, '%s')) return ($x!.value as %s, true); return (%s, false); })()",
		e.expr(x.X), types.TypeString(x.Assert, nil), e.dartType(x.Assert), e.zero(x.Assert))
}

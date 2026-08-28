package emit

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
)

// isInterfaceType reports whether a value of the type holds an interface pair.
//
// A type parameter holds no such pair. Its underlying type is the constraint,
// and the constraint is an interface, thus the check reads the core type first.
func isInterfaceType(t types.Type) bool {
	if _, ok := types.Unalias(t).(*types.TypeParam); ok {
		return false
	}
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
		// A generic type needs 1 description for each instance, thus the
		// package holds no variable for it. Read instanceDesc.
		if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
			continue
		}
		e.descriptor(named, false)
		e.descriptor(named, true)
	}
}

// isInstance reports whether the named type is an instance of a generic type.
func isInstance(named *types.Named) bool {
	return named.TypeArgs() != nil && named.TypeArgs().Len() > 0
}

// instanceDesc writes the description of 1 instance of a generic named type.
//
// The package holds a variable for the description of each other named type. An
// instance reaches the output only through its use, thus the description stands
// at that point. The name holds each type argument, and a type assertion
// compares that name.
func (e *emitter) instanceDesc(named *types.Named, pointer bool) string {
	t := types.Type(named)
	if pointer {
		t = types.NewPointer(named)
	}
	return fmt.Sprintf("TypeDesc('%s', '%s', <String, Function>{%s}, text: '%s'%s, zero: () => %s)",
		typeName(t), kindOf(t), strings.Join(e.methodTable(named, t), ", "),
		types.TypeString(types.Unalias(t), shortQualifier), e.descParts(t), e.zero(t))
}

// methodTable writes the adapter of each method of the type, in the order of
// the name.
func (e *emitter) methodTable(named *types.Named, t types.Type) []string {
	_, pointer := types.Unalias(t).(*types.Pointer)
	set := types.NewMethodSet(t)
	entries := make([]string, 0, set.Len())
	for i := 0; i < set.Len(); i++ {
		if entry := e.adapter(named, set.At(i), pointer); entry != "" {
			entries = append(entries, entry)
		}
	}
	sort.Strings(entries)
	return entries
}

func (e *emitter) descriptor(named *types.Named, pointer bool) {
	t := types.Type(named)
	variable := "$desc$" + name.Package(named.Obj().Name())
	if pointer {
		t = types.NewPointer(named)
		variable = "$descP$" + name.Package(named.Obj().Name())
	}
	e.line("final TypeDesc %s = TypeDesc('%s', '%s', <String, Function>{%s}, text: '%s'%s, zero: () => %s);",
		variable, types.TypeString(t, nil), kindOf(t),
		strings.Join(e.methodTable(named, t), ", "), types.TypeString(t, shortQualifier),
		e.descParts(t), e.zero(t))
}

// descParts writes the fields of a struct type, and the element type and the
// key type of a container type.
func (e *emitter) descParts(t types.Type) string {
	out := e.descShape(t)
	// The package internal/reflectlite reads the flag. Go compares 2 values of
	// each type except a slice, a map, a function, and a type that holds one.
	if !types.Comparable(t) {
		out += ", comparable: false"
	}
	return out
}

// descShape writes the element type, the key type, and the fields.
func (e *emitter) descShape(t types.Type) string {
	u := types.Unalias(t).Underlying()
	if p, ok := u.(*types.Pointer); ok {
		return ", elem: () => " + e.descriptorOf(p.Elem())
	}
	switch u := u.(type) {
	case *types.Struct:
		return e.fieldDescs(t, u)
	case *types.Slice:
		return fmt.Sprintf(", elem: () => %s, make: ($n) => $makeSlice<%s>($n, $n, %s)",
			e.descriptorOf(u.Elem()), e.dartType(u.Elem()), e.alloc(u.Elem()))
	case *types.Array:
		return ", elem: () => " + e.descriptorOf(u.Elem())
	case *types.Chan:
		return ", elem: () => " + e.descriptorOf(u.Elem())
	case *types.Map:
		return fmt.Sprintf(", key: () => %s, elem: () => %s, make: ($n) => GoMap<%s, %s>(() => %s)",
			e.descriptorOf(u.Key()), e.descriptorOf(u.Elem()),
			e.dartType(u.Key()), e.dartType(u.Elem()), e.zero(u.Elem()))
	}
	return ""
}

// fieldDescs writes the description of each struct field. The reflect subset
// and the fmt package read the name, the tag, and the value.
func (e *emitter) fieldDescs(t types.Type, st *types.Struct) string {
	class := e.dartType(t)
	parts := make([]string, st.NumFields())
	for i := range parts {
		field := st.Field(i)
		dart := name.Package(field.Name())
		parts[i] = fmt.Sprintf(
			"FieldDesc('%s', %s, %t, () => %s, ($r) => ($r as %s).%s, ($r, $v) => ($r as %s).%s = $v as %s)",
			field.Name(), dartString(st.Tag(i)), field.Exported(),
			e.descriptorOf(field.Type()), class, dart, class, dart,
			e.dartType(field.Type()))
	}
	return ", fields: <FieldDesc>[" + strings.Join(parts, ", ") + "]"
}

// shortQualifier writes the package name of a type, and not the import path.
// The Go verb %v and the Go verb %T both use this form.
func shortQualifier(p *types.Package) string { return p.Name() }

// kindOf gives the kind of a type, in the words of the Go package reflect.
func kindOf(t types.Type) string {
	u := types.Unalias(t).Underlying()
	switch u := u.(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.Bool, types.UntypedBool:
			return "bool"
		case types.String, types.UntypedString:
			return "string"
		case types.UnsafePointer:
			return "unsafe.Pointer"
		}
		return u.String()
	case *types.Slice:
		return "slice"
	case *types.Array:
		return "array"
	case *types.Map:
		return "map"
	case *types.Pointer:
		return "ptr"
	case *types.Struct:
		return "struct"
	case *types.Chan:
		return "chan"
	case *types.Signature:
		return "func"
	case *types.Interface:
		return "interface"
	}
	return "invalid"
}

func dartString(text string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range text {
		switch r {
		case '\'':
			b.WriteString(`\'`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '$':
			b.WriteString(`\$`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// adapter writes 1 entry of a method table. The entry takes the value of the
// interface, thus each entry has the same shape of receiver.
//
// pointer marks the table of a pointer to the type. Such a value holds a box
// when the base type is not a struct and not an array, thus a method with a
// value receiver reads the field of the box.
func (e *emitter) adapter(named *types.Named, sel *types.Selection, pointer bool) string {
	fn, ok := sel.Obj().(*types.Func)
	if !ok {
		return ""
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return ""
	}
	if sig.TypeParams() != nil && sig.TypeParams().Len() > 0 {
		// Go forbids a type parameter on an interface method, thus a generic
		// method never reaches a method table.
		return ""
	}
	params := make([]string, sig.Params().Len())
	args := make([]string, sig.Params().Len())
	for i := range params {
		args[i] = fmt.Sprintf("$a%d", i)
		params[i] = e.dartType(sig.Params().At(i).Type()) + " " + args[i]
	}
	recv := fmt.Sprintf("($r as %s)", e.dartType(named))
	if pointer && needsBox(named) {
		recv = fmt.Sprintf("($r as %s)", e.dartType(types.NewPointer(named)))
		if _, ok := types.Unalias(sig.Recv().Type()).(*types.Pointer); !ok {
			recv += "!.v"
		}
	}
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
		if isPointerType(current) && structOf(current) != nil {
			// A pointer to a struct is the Dart object reference, and that
			// type accepts null.
			recv += "!"
		}
	}
	call := fmt.Sprintf("%s.%s(%s)", recv, methodName(fn), strings.Join(args, ", "))
	switch {
	case isInterfaceType(current):
		// An embedded interface field holds an interface pair. The call reads
		// the method table of that pair.
		call = fmt.Sprintf("$call(%s, %q, <Object?>[%s])",
			recv, fn.Name(), strings.Join(args, ", "))
		if cast := e.resultCast(sig); cast != "" {
			call = "(" + call + " as " + cast + ")"
		}
	case structOf(current) == nil:
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

// resultCast gives the Dart type of the result of a method. The dispatch of an
// interface gives Object?, thus the caller needs a cast.
func (e *emitter) resultCast(sig *types.Signature) string {
	switch sig.Results().Len() {
	case 0:
		return ""
	case 1:
		return e.dartType(sig.Results().At(0).Type())
	}
	return e.recordType(sig.Results())
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
	// A type parameter reads its description from its dictionary. The caller
	// gives 1 dictionary for each type argument.
	if tp, ok := u.(*types.TypeParam); ok {
		return e.dictOf(tp) + ".desc"
	}
	if p, ok := u.(*types.Pointer); ok {
		if named, ok := types.Unalias(p.Elem()).(*types.Named); ok {
			if isInstance(named) {
				return e.instanceDesc(named, true)
			}
			return e.qualify(named.Obj(), "$descP$"+name.Package(named.Obj().Name()))
		}
	}
	if named, ok := u.(*types.Named); ok {
		// An interface type holds no descriptor of its own, and a predeclared
		// type belongs to no package.
		if !isInterfaceType(named) && named.Obj().Pkg() != nil {
			if isInstance(named) {
				return e.instanceDesc(named, false)
			}
			if named.TypeParams() == nil || named.TypeParams().Len() == 0 {
				return e.qualify(named.Obj(), "$desc$"+name.Package(named.Obj().Name()))
			}
		}
	}
	if b, ok := u.(*types.Basic); ok {
		if text := basicDesc(b); text != "" {
			return text
		}
	}
	return fmt.Sprintf("TypeDesc('%s', '%s', const <String, Function>{}, text: '%s'%s, zero: () => %s)",
		typeName(t), kindOf(t), types.TypeString(types.Unalias(t), shortQualifier),
		e.descParts(t), e.zero(t))
}

// basicDesc names the description of a predeclared type. The runtime library
// holds 1 constant for each one.
func basicDesc(b *types.Basic) string {
	switch b.Kind() {
	case types.Bool, types.UntypedBool:
		return "$typeBool"
	case types.String, types.UntypedString:
		return "$typeString"
	case types.Int:
		return "$typeInt"
	case types.Int8:
		return "$typeInt8"
	case types.Int16:
		return "$typeInt16"
	case types.Int32:
		return "$typeInt32"
	case types.Int64:
		return "$typeInt64"
	case types.Uint:
		return "$typeUint"
	case types.Uint8:
		return "$typeUint8"
	case types.Uint16:
		return "$typeUint16"
	case types.Uint32:
		return "$typeUint32"
	case types.Uint64:
		return "$typeUint64"
	case types.Uintptr:
		return "$typeUintptr"
	case types.Float32:
		return "$typeFloat32"
	case types.Float64, types.UntypedFloat:
		return "$typeFloat64"
	}
	return ""
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

// typeName gives the name that a descriptor holds and that a type test reads.
// An alias carries the name of its own declaration, thus the name comes from
// the type behind it.
func typeName(t types.Type) string {
	return types.TypeString(types.Unalias(t), nil)
}

// typeTest writes the test of a type assertion and of a type switch.
func (e *emitter) typeTest(x ir.Expr, target types.Type) string {
	if isInterfaceType(target) {
		return fmt.Sprintf("$implements(%s, %s)", e.expr(x), methodNameList(interfaceMethods(target)))
	}
	return fmt.Sprintf("$isType(%s, '%s')", e.expr(x), typeName(target))
}

func (e *emitter) typeAssert(x *ir.TypeAssert) string {
	if !x.Comma {
		if isInterfaceType(x.Assert) {
			return fmt.Sprintf("$assertIface(%s, %s, '%s')",
				e.expr(x.X), methodNameList(interfaceMethods(x.Assert)),
				typeName(x.Assert))
		}
		return fmt.Sprintf("($assertType(%s, '%s') as %s)",
			e.expr(x.X), typeName(x.Assert), e.dartType(x.Assert))
	}
	if isInterfaceType(x.Assert) {
		return fmt.Sprintf("(() { final $x = %s; if ($implements($x, %s)) return ($x, true); return (null, false); })()",
			e.expr(x.X), methodNameList(interfaceMethods(x.Assert)))
	}
	return fmt.Sprintf("(() { final $x = %s; if ($isType($x, '%s')) return ($x!.value as %s, true); return (%s, false); })()",
		e.expr(x.X), typeName(x.Assert), e.dartType(x.Assert), e.zero(x.Assert))
}

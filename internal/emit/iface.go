package emit

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
	"github.com/eddyw/godart/internal/typeset"
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
	e.scopeDescriptors(e.pkg.Scope())
}

// scopeDescriptors writes the description of each named type of a scope and of
// each scope inside it. A type of a function scope also reaches an interface,
// thus it also needs a description.
func (e *emitter) scopeDescriptors(scope *types.Scope) {
	for _, n := range scope.Names() {
		obj, ok := scope.Lookup(n).(*types.TypeName)
		if !ok || obj.IsAlias() {
			continue
		}
		// A dot import puts each exported name of another package into the
		// scope of the file. The package of that name writes its description.
		if obj.Pkg() != e.pkg {
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
	for i := 0; i < scope.NumChildren(); i++ {
		e.scopeDescriptors(scope.Child(i))
	}
}

// descName gives the name of the variable that holds the description of a named
// type. A type of a function scope holds its position in the name, because 2
// functions can declare 1 name.
func (e *emitter) descName(obj *types.TypeName, pointer bool) string {
	out := "$desc$"
	if pointer {
		out = "$descP$"
	}
	out += name.Package(obj.Name())
	if obj.Parent() == nil || obj.Parent() == obj.Pkg().Scope() || e.cfg.Fset == nil {
		return out
	}
	p := e.cfg.Fset.Position(obj.Pos())
	return fmt.Sprintf("%s$%d_%d", out, p.Line, p.Column)
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
	// The description of an instance with a free type parameter can reach
	// itself through a field or a method, and no variable can hold it. The
	// second visit gives a shallow description, thus the text ends.
	key := e.typeName(t)
	if e.descBusy == nil {
		e.descBusy = map[string]bool{}
	}
	if e.descBusy[key] {
		return fmt.Sprintf("TypeDesc('%s', '%s', const <String, Function>{}, text: '%s', zero: () => %s)",
			dartText(key), kindOf(t), dartText(reflectText(types.Unalias(t))), e.zero(t))
	}
	e.descBusy[key] = true
	defer delete(e.descBusy, key)
	return fmt.Sprintf("TypeDesc(%s, '%s', <String, Function>{%s}, text: %s%s, zero: () => %s)",
		e.typeNameText(t), kindOf(t), strings.Join(e.methodTable(named, t), ", "),
		e.reflectTextExpr(t), e.methodTypeTable(t)+e.descParts(t), e.zero(t))
}

// methodTable writes the adapter of each method of the type, in the order of
// the name.
// methodTable writes the adapter of each method of the method set of a type.
// The element type is the named type or the unnamed struct type behind t, and t
// holds the pointer form when the table belongs to a pointer.
func (e *emitter) methodTable(elem types.Type, t types.Type) []string {
	_, pointer := types.Unalias(t).(*types.Pointer)
	set := types.NewMethodSet(t)
	entries := make([]string, 0, set.Len())
	for i := 0; i < set.Len(); i++ {
		if entry := e.adapter(elem, set.At(i), pointer); entry != "" {
			entries = append(entries, entry)
		}
	}
	sort.Strings(entries)
	return entries
}

func (e *emitter) descriptor(named *types.Named, pointer bool) {
	t := types.Type(named)
	variable := e.descName(named.Obj(), false)
	if pointer {
		t = types.NewPointer(named)
		variable = e.descName(named.Obj(), true)
	}
	e.line("final TypeDesc %s = TypeDesc('%s', '%s', <String, Function>{%s}, text: '%s'%s, zero: () => %s);",
		variable, dartText(e.typeName(t)), kindOf(t),
		strings.Join(e.methodTable(named, t), ", "), dartText(reflectText(t)),
		e.methodTypeTable(t)+e.descParts(t), e.zero(t))
}

// methodTypeTable writes the function type of each exported method of the
// method set of a type. The package reflect reads it: Value.Method gives a
// func value, and Value.Call of it needs the parameter types and the result
// types.
func (e *emitter) methodTypeTable(t types.Type) string {
	set := types.NewMethodSet(t)
	entries := make([]string, 0, set.Len())
	for i := 0; i < set.Len(); i++ {
		fn, ok := set.At(i).Obj().(*types.Func)
		if !ok || !fn.Exported() {
			continue
		}
		if o, ok := fn.Type().(*types.Signature); ok &&
			o.TypeParams() != nil && o.TypeParams().Len() > 0 {
			continue
		}
		sig, ok := set.At(i).Type().(*types.Signature)
		if !ok {
			continue
		}
		entries = append(entries, fmt.Sprintf("%s: () => %s",
			e.methodKeyText(fn.Name(), sig), e.descriptorOf(sig)))
	}
	if len(entries) == 0 {
		return ""
	}
	sort.Strings(entries)
	return ", methodTypes: <String, TypeDesc Function()>{" +
		strings.Join(entries, ", ") + "}"
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
		return fmt.Sprintf(", elem: () => %s, length: %d", e.descriptorOf(u.Elem()), u.Len())
	case *types.Chan:
		return ", elem: () => " + e.descriptorOf(u.Elem())
	case *types.Map:
		return fmt.Sprintf(", key: () => %s, elem: () => %s, make: ($n) => %s",
			e.descriptorOf(u.Key()), e.descriptorOf(u.Elem()), e.mapNew(u))
	case *types.Signature:
		return e.signatureShape(u)
	}
	return ""
}

// signatureShape writes the parameter types and the result types of a function
// type. The package reflect reads them, and Value.Call needs them.
func (e *emitter) signatureShape(sig *types.Signature) string {
	params := make([]string, sig.Params().Len())
	for i := range params {
		params[i] = "() => " + e.descriptorOf(sig.Params().At(i).Type())
	}
	results := make([]string, sig.Results().Len())
	for i := range results {
		results[i] = "() => " + e.descriptorOf(sig.Results().At(i).Type())
	}
	return fmt.Sprintf(", params: <TypeDesc Function()>[%s], results: <TypeDesc Function()>[%s]",
		strings.Join(params, ", "), strings.Join(results, ", "))
}

// fieldDescs writes the description of each struct field. The reflect subset
// and the fmt package read the name, the tag, and the value.
func (e *emitter) fieldDescs(t types.Type, st *types.Struct) string {
	class := e.dartType(t)
	parts := make([]string, st.NumFields())
	for i := range parts {
		field := st.Field(i)
		parts[i] = fmt.Sprintf("FieldDesc('%s', %s, %t, () => %s, %s)",
			field.Name(), dartString(st.Tag(i)), field.Exported(),
			e.descriptorOf(field.Type()), e.fieldAccess(field, i, class))
	}
	return ", fields: <FieldDesc>[" + strings.Join(parts, ", ") + "]"
}

// fieldAccess writes the getter and the setter of 1 field.
//
// An unexported field of a type of another package reaches no accessor. A Dart
// library holds each name that starts with an underscore, thus another library
// cannot name it. The description of an instance of a generic type stands at
// the point of the use, and that point can lie outside the package of the type.
func (e *emitter) fieldAccess(field *types.Var, i int, class string) string {
	if !field.Exported() && field.Pkg() != nil && field.Pkg() != e.pkg {
		return "$noRead, $noWrite"
	}
	dart := structFieldName(field, i)
	return fmt.Sprintf("($r) => ($r as %s).%s, ($r, $v) => ($r as %s).%s = $v as %s",
		class, dart, class, dart, e.dartType(field.Type()))
}

// dartText escapes a text for a Dart string literal that single quotes hold.
// The tag of a struct field reaches the text of a type, and such a tag holds a
// quotation mark and a backslash.
func dartText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '\'', '$':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// reflectText writes the text of a type in the form that reflect.Type.String
// gives.
//
// `go/types` writes an unnamed struct type and an unnamed interface type with
// no space after the keyword, and `reflect` writes 1 space. Each other form is
// equal, thus the function falls back to `types.TypeString`.
func reflectText(t types.Type) string {
	switch t := types.Unalias(t).(type) {
	case *types.Struct:
		if t.NumFields() == 0 {
			return "struct {}"
		}
		parts := make([]string, t.NumFields())
		for i := range parts {
			field := t.Field(i)
			text := reflectText(field.Type())
			if !field.Embedded() {
				text = field.Name() + " " + text
			}
			if tag := t.Tag(i); tag != "" {
				text += " " + strconv.Quote(tag)
			}
			parts[i] = text
		}
		return "struct { " + strings.Join(parts, "; ") + " }"
	case *types.Interface:
		if t.NumMethods() == 0 {
			return "interface {}"
		}
		parts := make([]string, t.NumMethods())
		for i := range parts {
			m := t.Method(i)
			parts[i] = m.Name() + strings.TrimPrefix(reflectText(m.Type()), "func")
		}
		return "interface { " + strings.Join(parts, "; ") + " }"
	case *types.Pointer:
		return "*" + reflectText(t.Elem())
	case *types.Slice:
		return "[]" + reflectText(t.Elem())
	case *types.Array:
		return "[" + strconv.FormatInt(t.Len(), 10) + "]" + reflectText(t.Elem())
	case *types.Map:
		return "map[" + reflectText(t.Key()) + "]" + reflectText(t.Elem())
	case *types.Chan:
		switch t.Dir() {
		case types.SendOnly:
			return "chan<- " + reflectText(t.Elem())
		case types.RecvOnly:
			return "<-chan " + reflectText(t.Elem())
		}
		return "chan " + reflectText(t.Elem())
	case *types.Signature:
		return signatureText(t)
	}
	return types.TypeString(t, shortQualifier)
}

// signatureText writes a function type in the form of reflect.
func signatureText(sig *types.Signature) string {
	params := make([]string, sig.Params().Len())
	for i := range params {
		params[i] = reflectText(sig.Params().At(i).Type())
	}
	if sig.Variadic() && len(params) > 0 {
		last := params[len(params)-1]
		params[len(params)-1] = "..." + strings.TrimPrefix(last, "[]")
	}
	out := "func(" + strings.Join(params, ", ") + ")"
	results := make([]string, sig.Results().Len())
	for i := range results {
		results[i] = reflectText(sig.Results().At(i).Type())
	}
	switch len(results) {
	case 0:
		return out
	case 1:
		return out + " " + results[0]
	}
	return out + " (" + strings.Join(results, ", ") + ")"
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
func (e *emitter) adapter(elem types.Type, sel *types.Selection, pointer bool) string {
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
	// The selection carries the signature of the instance, and the key of an
	// instance holds the type arguments, not the parameters of the origin.
	selSig, _ := sel.Type().(*types.Signature)
	if selSig == nil {
		selSig = sig
	}
	params := make([]string, selSig.Params().Len())
	args := make([]string, selSig.Params().Len())
	for i := range params {
		args[i] = fmt.Sprintf("$a%d", i)
		params[i] = e.dartType(selSig.Params().At(i).Type()) + " " + args[i]
	}
	recv := fmt.Sprintf("($r as %s)", e.dartType(elem))
	if pointer && needsBox(elem) {
		recv = fmt.Sprintf("($r as %s)", e.dartType(types.NewPointer(elem)))
		if _, ok := types.Unalias(sig.Recv().Type()).(*types.Pointer); !ok {
			recv += "!.v"
		}
	}
	// The Dart form of a pointer to a struct accepts null, thus the cast must
	// accept it too. Go runs a method with a nil pointer receiver.
	if pointer && !needsBox(elem) && len(sel.Index()) == 1 && topLevelMethod(elem, fn) {
		recv = fmt.Sprintf("($r as %s)", e.dartType(types.NewPointer(elem)))
	}
	index := sel.Index()
	current := elem
	for _, i := range index[:len(index)-1] {
		st := structOf(current)
		if st == nil {
			e.fail("the method path of %s leaves a struct type", elem)
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
	case fn.Pkg() != nil && fn.Pkg() != e.pkg && !fn.Exported():
		// A Dart library holds each name that starts with an underscore, thus
		// another library reaches no unexported method of a type of this
		// package. Such a method reaches the table of a type that embeds the
		// type, and only the package of the method can call it.
		call = fmt.Sprintf("$noMethod(%q)", fn.Pkg().Path()+"."+fn.Name())
	case isInterfaceType(current):
		// An embedded interface field holds an interface pair. The call reads
		// the method table of that pair.
		call = fmt.Sprintf("$call(%s, %s, <Object?>[%s])",
			recv, e.methodKeyText(fn.Name(), selSig), strings.Join(args, ", "))
		if cast := e.resultCast(selSig); cast != "" {
			call = "(" + call + " as " + cast + ")"
		}
	case structOf(current) == nil || topLevelMethod(current, fn):
		owner := receiverTypeName(current)
		if owner == nil {
			e.fail("the method %s has no named receiver type", fn.Name())
			return ""
		}
		dicts, typeArgs := e.recvTypeArgs(current)
		all := append(append([]string{recv}, dicts...), args...)
		call = e.qualify(owner, methodFuncName(owner.Name(), fn.Name())) +
			joinTypeArgs(typeArgs, "") + "(" + strings.Join(all, ", ") + ")"
	}
	head := "Object? $r"
	if len(params) > 0 {
		head += ", " + strings.Join(params, ", ")
	}
	return fmt.Sprintf("%s: (%s) => %s", e.methodKeyText(fn.Name(), selSig), head, call)
}

// topLevelMethod reports whether the body of the method stands in a top level
// function. A method with a pointer receiver on a struct type takes that form.
func topLevelMethod(recv types.Type, fn *types.Func) bool {
	obj := receiverTypeName(recv)
	return obj != nil && pointerMethod(fn)
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
				return e.instDescVar(named, true)
			}
			// The package holds no variable for an interface type and for a
			// predeclared type, thus a pointer to one takes its own
			// description.
			if !isInterfaceType(named) && named.Obj().Pkg() != nil &&
				(named.TypeParams() == nil || named.TypeParams().Len() == 0) {
				return e.qualify(named.Obj(), e.descName(named.Obj(), true))
			}
		}
	}
	if named, ok := u.(*types.Named); ok {
		// An interface type holds no descriptor of its own, and a predeclared
		// type belongs to no package.
		if !isInterfaceType(named) && named.Obj().Pkg() != nil {
			if isInstance(named) {
				return e.instDescVar(named, false)
			}
			if named.TypeParams() == nil || named.TypeParams().Len() == 0 {
				return e.qualify(named.Obj(), e.descName(named.Obj(), false))
			}
		}
	}
	if b, ok := u.(*types.Basic); ok {
		if text := basicDesc(b); text != "" {
			return text
		}
	}
	// An unnamed struct type holds the promoted method of each embedded field,
	// thus its description also needs a method table. An interface type holds
	// the key of each method that it needs, because a type assertion to a type
	// parameter reads that set. A slice, a map, and a channel hold no method.
	table := "const <String, Function>{}"
	methodTypes := ""
	switch {
	case isStructUnder(t):
		if entries := e.methodTable(types.Unalias(t), t); len(entries) > 0 {
			table = "<String, Function>{" + strings.Join(entries, ", ") + "}"
			methodTypes = e.methodTypeTable(t)
		}
	case isInterfaceType(t):
		if keys := interfaceMethods(t); len(keys) > 0 {
			parts := make([]string, len(keys))
			for i, k := range keys {
				parts[i] = fmt.Sprintf("%q: $ifaceMethod", k)
			}
			table = "<String, Function>{" + strings.Join(parts, ", ") + "}"
		}
	}
	return fmt.Sprintf("TypeDesc(%s, '%s', %s, text: %s%s, zero: () => %s)",
		e.typeNameText(t), kindOf(t), table, e.reflectTextExpr(t),
		methodTypes+e.descParts(t), e.zero(t))
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
		m := iface.Method(i)
		sig, _ := m.Type().(*types.Signature)
		out[i] = methodKey(m.Name(), sig)
	}
	sort.Strings(out)
	return out
}

// methodKey names 1 entry of a method table.
//
// The key holds the name and the signature. Go compares both, thus an
// interface with the method `Unwrap() error` and an interface with the method
// `Unwrap() []error` stay apart.
func methodKey(name string, sig *types.Signature) string {
	if sig == nil {
		return name
	}
	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('(')
	for i := 0; i < sig.Params().Len(); i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		if sig.Variadic() && i == sig.Params().Len()-1 {
			b.WriteString("...")
			if s, ok := types.Unalias(sig.Params().At(i).Type()).(*types.Slice); ok {
				b.WriteString(keyType(s.Elem()))
				continue
			}
		}
		b.WriteString(keyType(sig.Params().At(i).Type()))
	}
	b.WriteByte(')')
	for i := 0; i < sig.Results().Len(); i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(keyType(sig.Results().At(i).Type()))
	}
	return b.String()
}

// keyType names a type inside a method key.
//
// An alias gives the name of the type behind it. 2 packages can hold 2 alias
// names of 1 type, and Go holds the 2 methods equal, thus the 2 keys must stay
// equal.
func keyType(t types.Type) string {
	return types.TypeString(unaliasDeep(t), nil)
}

// unaliasDeep replaces each alias of a type, and of the element of a container
// of it, with the type behind it.
//
// An alias inside a struct, inside a signature, or inside a type argument
// stays. docs/parity.md holds that gap.
func unaliasDeep(t types.Type) types.Type {
	switch u := types.Unalias(t).(type) {
	case *types.Pointer:
		return types.NewPointer(unaliasDeep(u.Elem()))
	case *types.Slice:
		return types.NewSlice(unaliasDeep(u.Elem()))
	case *types.Array:
		return types.NewArray(unaliasDeep(u.Elem()), u.Len())
	case *types.Map:
		return types.NewMap(unaliasDeep(u.Key()), unaliasDeep(u.Elem()))
	case *types.Chan:
		return types.NewChan(u.Dir(), unaliasDeep(u.Elem()))
	default:
		return u
	}
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
	// The argument of a type parameter can be an interface type, and a value
	// of such a parameter already holds an interface value.
	if _, ok := types.Unalias(from).(*types.TypeParam); ok {
		return fmt.Sprintf("$boxDesc(%s, %s)", e.descriptorOf(from), text)
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
		return fmt.Sprintf("($method(%s, %s) as %s Function(%s))(%s)",
			recv, e.methodKeyText(m.Obj.Name(), sig), result,
			strings.Join(params, ", "), strings.Join(all, ", "))
	}
	call := fmt.Sprintf("$call(%s, %s, <Object?>[%s])",
		e.expr(m.Recv), e.methodKeyText(m.Obj.Name(), sig), strings.Join(args, ", "))
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
func (e *emitter) typeName(t types.Type) string {
	text := types.TypeString(stripNames(types.Unalias(t)), nil)
	// 2 local types of 2 functions can share 1 printed name, and the
	// identity of a type reads the name. The position keeps them apart, and
	// the display text stays short.
	if named, ok := types.Unalias(t).(*types.Named); ok &&
		isLocalType(named.Obj()) && e.cfg.Fset != nil {
		p := e.cfg.Fset.Position(named.Obj().Pos())
		text += fmt.Sprintf("·%d_%d", p.Line, p.Column)
	}
	return text
}

// stripNames gives the type with no parameter name inside a function type.
//
// A type assertion compares the name of a descriptor, and 2 texts of 1
// function type must stay equal. The text of a declared signature holds the
// parameter names, and the text of an assertion target holds none.
func stripNames(t types.Type) types.Type {
	switch u := types.Unalias(t).(type) {
	case *types.Basic:
		// byte and uint8 name 1 type, and so do rune and int32. The test of
		// a type compares the name, thus the name comes from the kind.
		return types.Typ[u.Kind()]
	case *types.Signature:
		params := make([]*types.Var, u.Params().Len())
		for i := range params {
			params[i] = types.NewVar(token.NoPos, nil, "", stripNames(u.Params().At(i).Type()))
		}
		results := make([]*types.Var, u.Results().Len())
		for i := range results {
			results[i] = types.NewVar(token.NoPos, nil, "", stripNames(u.Results().At(i).Type()))
		}
		return types.NewSignatureType(nil, nil, nil,
			types.NewTuple(params...), types.NewTuple(results...), u.Variadic())
	case *types.Pointer:
		return types.NewPointer(stripNames(u.Elem()))
	case *types.Slice:
		return types.NewSlice(stripNames(u.Elem()))
	case *types.Array:
		return types.NewArray(stripNames(u.Elem()), u.Len())
	case *types.Map:
		return types.NewMap(stripNames(u.Key()), stripNames(u.Elem()))
	case *types.Chan:
		return types.NewChan(u.Dir(), stripNames(u.Elem()))
	}
	return t
}

// isStructUnder reports whether the underlying type is a struct.
func isStructUnder(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Struct)
	return ok
}

// typeTest writes the test of a type assertion and of a type switch.
func (e *emitter) typeTest(x ir.Expr, target types.Type) string {
	if isInterfaceType(target) {
		return fmt.Sprintf("$implements(%s, %s)", e.expr(x), e.methodKeyListText(target))
	}
	// A type parameter holds no kind of its own. An interface type argument
	// needs the test of the method set, thus the test reads the description.
	if tp, ok := types.Unalias(target).(*types.TypeParam); ok {
		return fmt.Sprintf("$isDictType(%s, %s.desc)", e.expr(x), e.dictOf(tp))
	}
	return fmt.Sprintf("$isType(%s, %s)", e.expr(x), e.typeNameText(target))
}

// typeNameText writes the name that a type test compares. A type parameter
// holds no name of its own, thus the name comes from its dictionary. A type
// that reads a parameter interpolates that name at run time.
func (e *emitter) typeNameText(t types.Type) string {
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		return e.dictOf(tp) + ".desc.name"
	}
	if !typeset.HoldsParam(t) {
		return "'" + dartText(e.typeName(t)) + "'"
	}
	var reads []string
	sub := e.substParams(t, &reads)
	return interpolate(types.TypeString(stripNames(sub), nil), reads)
}

// interpolate writes the Dart string whose marker parts read a name at run
// time.
func interpolate(text string, reads []string) string {
	parts := strings.Split(text, "\x01")
	var b strings.Builder
	b.WriteString("'")
	for i, part := range parts {
		if i%2 == 0 {
			b.WriteString(dartText(part))
			continue
		}
		var n int
		fmt.Sscanf(part, "%d", &n)
		b.WriteString("${" + reads[n] + "}")
	}
	b.WriteString("'")
	return b.String()
}

// methodKeyText writes the Dart expression of the key of 1 method. A key that
// reads a type parameter interpolates its name at run time.
func (e *emitter) methodKeyText(name string, sig *types.Signature) string {
	if sig == nil || !typeset.HoldsParam(sig) {
		return "'" + dartText(methodKey(name, sig)) + "'"
	}
	var reads []string
	sub, ok := e.substParams(sig, &reads).(*types.Signature)
	if !ok {
		return "'" + dartText(methodKey(name, sig)) + "'"
	}
	return interpolate(methodKey(name, sub), reads)
}

// methodKeyListText writes the Dart list of the key of each method of an
// interface type.
func (e *emitter) methodKeyListText(t types.Type) string {
	iface, ok := types.Unalias(t).Underlying().(*types.Interface)
	if !ok {
		return "<String>[]"
	}
	parts := make([]string, iface.NumMethods())
	for i := range parts {
		m := iface.Method(i)
		sig, _ := m.Type().(*types.Signature)
		parts[i] = e.methodKeyText(m.Name(), sig)
	}
	sort.Strings(parts)
	return "<String>[" + strings.Join(parts, ", ") + "]"
}

// reflectTextExpr writes the Dart expression of the reflect text of a type.
// A type parameter reads the text of its dictionary at run time.
func (e *emitter) reflectTextExpr(t types.Type) string {
	if !typeset.HoldsParam(t) {
		return "'" + dartText(reflectText(types.Unalias(t))) + "'"
	}
	var reads []string
	sub := e.substParams(t, &reads)
	for i := range reads {
		reads[i] = strings.Replace(reads[i], ".desc.name", ".desc.display", 1)
	}
	return interpolate(reflectText(types.Unalias(sub)), reads)
}

// substParams replaces each type parameter with a marker type, thus the name
// text splits at each parameter. The list holds the run time read of each
// marker, in order.
func (e *emitter) substParams(t types.Type, reads *[]string) types.Type {
	switch u := types.Unalias(t).(type) {
	case *types.TypeParam:
		marker := fmt.Sprintf("\x01%d\x01", len(*reads))
		*reads = append(*reads, e.dictOf(u)+".desc.name")
		obj := types.NewTypeName(token.NoPos, nil, marker, nil)
		return types.NewNamed(obj, types.Typ[types.Int], nil)
	case *types.Pointer:
		return types.NewPointer(e.substParams(u.Elem(), reads))
	case *types.Slice:
		return types.NewSlice(e.substParams(u.Elem(), reads))
	case *types.Array:
		return types.NewArray(e.substParams(u.Elem(), reads), u.Len())
	case *types.Map:
		return types.NewMap(e.substParams(u.Key(), reads), e.substParams(u.Elem(), reads))
	case *types.Chan:
		return types.NewChan(u.Dir(), e.substParams(u.Elem(), reads))
	case *types.Signature:
		params := make([]*types.Var, u.Params().Len())
		for i := range params {
			params[i] = types.NewVar(token.NoPos, nil, "", e.substParams(u.Params().At(i).Type(), reads))
		}
		results := make([]*types.Var, u.Results().Len())
		for i := range results {
			results[i] = types.NewVar(token.NoPos, nil, "", e.substParams(u.Results().At(i).Type(), reads))
		}
		return types.NewSignatureType(nil, nil, nil,
			types.NewTuple(params...), types.NewTuple(results...), u.Variadic())
	case *types.Struct:
		fields := make([]*types.Var, u.NumFields())
		tags := make([]string, u.NumFields())
		for i := range fields {
			f := u.Field(i)
			fields[i] = types.NewField(token.NoPos, f.Pkg(), f.Name(),
				e.substParams(f.Type(), reads), f.Embedded())
			tags[i] = u.Tag(i)
		}
		return types.NewStruct(fields, tags)
	case *types.Named:
		if u.TypeArgs().Len() > 0 {
			args := make([]types.Type, u.TypeArgs().Len())
			for i := range args {
				args[i] = e.substParams(u.TypeArgs().At(i), reads)
			}
			if inst, err := types.Instantiate(nil, u.Origin(), args, false); err == nil {
				return inst
			}
		}
		return u
	}
	return t
}

func (e *emitter) typeAssert(x *ir.TypeAssert) string {
	tp, isParam := types.Unalias(x.Assert).(*types.TypeParam)
	if !x.Comma {
		if isInterfaceType(x.Assert) {
			return fmt.Sprintf("$assertIface(%s, %s, %s)",
				e.expr(x.X), e.methodKeyListText(x.Assert),
				e.typeNameText(x.Assert))
		}
		if isParam {
			return fmt.Sprintf("$assertDict(%s, %s)", e.expr(x.X), e.dictOf(tp))
		}
		return fmt.Sprintf("($assertType(%s, %s) as %s)",
			e.expr(x.X), e.typeNameText(x.Assert), e.dartType(x.Assert))
	}
	if isInterfaceType(x.Assert) {
		return fmt.Sprintf("(() { final $x = %s; if ($implements($x, %s)) return ($x, true); return (null, false); })()",
			e.expr(x.X), e.methodKeyListText(x.Assert))
	}
	if isParam {
		return fmt.Sprintf("(() { final $x = %s; if ($isDictType($x, %s.desc)) return ($dictValue<%s>($x, %s), true); return (%s.zero(), false); })()",
			e.expr(x.X), e.dictOf(tp), e.dartType(tp), e.dictOf(tp), e.dictOf(tp))
	}
	return fmt.Sprintf("(() { final $x = %s; if ($isType($x, %s)) return ($x!.value as %s, true); return (%s, false); })()",
		e.expr(x.X), e.typeNameText(x.Assert), e.dartType(x.Assert), e.zero(x.Assert))
}

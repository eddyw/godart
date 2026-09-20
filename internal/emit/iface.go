package emit

import (
	"crypto/sha256"
	"encoding/hex"
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
		// The name of a local type of a generic function reads the dictionary
		// of that function, thus the description stands at the point of the
		// use.
		if len(outerTypeParams(named)) > 0 {
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
		e.reflectTextExpr(t),
		e.methodTypeTable(t)+e.descParts(t)+e.pointerPart(named, pointer),
		e.zero(t))
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
		e.methodTypeTable(t)+e.descParts(t)+e.pointerPart(named, pointer),
		e.zero(t))
}

// pointerPart links the 2 descriptions of a type whose pointer type holds
// a method that the value type does not hold.
//
// A method with a pointer receiver stays out of the value method set.
// encoding/json reads such a method, because Unmarshal writes through a
// pointer. The value description names the pointer description, and the pointer
// description builds a pointer to a fresh value.
func (e *emitter) pointerPart(t types.Type, pointer bool) string {
	if types.NewMethodSet(types.NewPointer(t)).Len() <=
		types.NewMethodSet(t).Len() {
		return ""
	}
	if !pointer {
		return ", ptr: () => " + e.descriptorOf(types.NewPointer(t))
	}
	return ", newPtr: () => " + e.freshPointer(t)
}

// freshPointer writes a pointer to the zero value of a type. The Dart form of a
// pointer to a struct and of a pointer to an array is the value itself, and
// each other pointer holds a box.
func (e *emitter) freshPointer(t types.Type) string {
	switch types.Unalias(t).Underlying().(type) {
	case *types.Struct, *types.Array:
		return e.zero(t)
	}
	return fmt.Sprintf("Box<%s>(%s)", e.dartType(t), e.zero(t))
}

// Interface method metadata includes unexported signatures for reflection.
func (e *emitter) methodTypeTable(t types.Type) string {
	set := types.NewMethodSet(t)
	entries := make([]string, 0, set.Len())
	for i := 0; i < set.Len(); i++ {
		fn, ok := set.At(i).Obj().(*types.Func)
		if !ok || !fn.Exported() && !isInterfaceType(t) {
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
			e.methodKeyText(fn.Id(), sig), e.descriptorOf(sig)))
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
	out := e.descShape(t) + withTypePart(e, t)
	if _, ok := types.Unalias(t).Underlying().(*types.Struct); ok {
		class := e.dartType(t)
		if class != "Object?" {
			out += fmt.Sprintf(", assign: ($d, $s) => ($d as %s).$assign($s as %s)", class, class)
		}
	}
	// The package internal/reflectlite reads the flag. Go compares 2 values of
	// each type except a slice, a map, a function, and a type that holds one.
	if !types.Comparable(t) {
		out += ", comparable: false"
	}
	return out
}

// withTypePart writes the factory that names the Dart type of a value of the
// type. The package reflect builds a typed cell through it, thus Value.Addr
// and reflect.New each give a pointer that a type assertion reads.
//
// A slice, a map, a channel, a function, and a pointer each name the Dart type
// of the element, and an array names it too. A struct names the class that the
// emitter writes. None of these 7 names lives outside the emitted code, thus
// each 1 carries the factory. The kind alone names the Dart type of each other
// type: a bool gives bool, a string gives GoString, and each integer kind gives
// GoInt.
func withTypePart(e *emitter, t types.Type) string {
	switch types.Unalias(t).Underlying().(type) {
	case *types.Slice, *types.Map, *types.Chan, *types.Signature,
		*types.Pointer, *types.Array, *types.Struct:
		dart := e.dartType(t)
		// A cyclic type and a type of a Dart object each erase to the top type
		// of Dart, and the kind already gives that answer.
		if dart == "Object?" {
			return ""
		}
		return fmt.Sprintf(
			", withType: <$R>($R Function<$T>() $f) => $f<%s>()", dart)
	}
	return ""
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
		return fmt.Sprintf(", elem: () => %s, dir: %d",
			e.descriptorOf(u.Elem()), chanDir(u))
	case *types.Map:
		return fmt.Sprintf(", key: () => %s, elem: () => %s, make: ($n) => %s",
			e.descriptorOf(u.Key()), e.descriptorOf(u.Elem()), e.mapNew(u))
	case *types.Signature:
		return e.signatureShape(u)
	}
	return ""
}

// chanDir names the direction of a channel type in the numbers of the package
// reflect: 1 for a receive, 2 for a send, and 3 for both.
func chanDir(c *types.Chan) int {
	switch c.Dir() {
	case types.RecvOnly:
		return 1
	case types.SendOnly:
		return 2
	}
	return 3
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
	variadic := ""
	if sig.Variadic() {
		variadic = ", variadic: true"
	}
	return fmt.Sprintf(", params: <TypeDesc Function()>[%s], results: <TypeDesc Function()>[%s]%s%s",
		strings.Join(params, ", "), strings.Join(results, ", "), variadic,
		e.bindPart(sig))
}

// bindPart writes the factory that builds a closure of the Dart type of a
// function type.
//
// The body of the factory reads a list of the raw arguments and gives a list of
// the raw results. Only the emitted code names the Dart type of a parameter and
// of a result, thus the package reflect cannot write such a closure of its own.
// A closure of Object? parameters gives a dynamic result, and Dart rejects that
// value at a location of a function type with a result.
//
// reflect.Value.Method, reflect.Value.MethodByName, and reflect.MakeFunc each
// read the factory.
func (e *emitter) bindPart(sig *types.Signature) string {
	params := make([]string, sig.Params().Len())
	args := make([]string, sig.Params().Len())
	for i := range params {
		args[i] = fmt.Sprintf("$a%d", i)
		params[i] = e.dartType(sig.Params().At(i).Type()) + " " + args[i]
	}
	call := fmt.Sprintf("$f(<Object?>[%s])", strings.Join(args, ", "))
	rs := sig.Results()
	if rs.Len() == 0 {
		return fmt.Sprintf(", bind: (List<Object?> Function(List<Object?>) $f) => (%s) => %s",
			strings.Join(params, ", "), call)
	}
	parts := make([]string, rs.Len())
	zeros := make([]string, rs.Len())
	for i := range parts {
		parts[i] = fmt.Sprintf("$o[%d] as %s", i, e.dartType(rs.At(i).Type()))
		zeros[i] = e.zero(rs.At(i).Type())
	}
	result, zero := parts[0], zeros[0]
	resultType := e.dartType(rs.At(0).Type())
	if rs.Len() > 1 {
		result, zero = "("+strings.Join(parts, ", ")+")", "("+strings.Join(zeros, ", ")+")"
		resultType = e.recordType(rs)
	}
	return fmt.Sprintf(", bind: (List<Object?> Function(List<Object?>) $f) => (%s) => $resumeReflect<%s>(%s, (Object? $v) { final $o = $v as List<Object?>; return %s; }, %s)",
		strings.Join(params, ", "), resultType, call, result, zero)

}

// fieldDescs writes the description of each struct field. The reflect subset
// and the fmt package read the name, the tag, and the value.
func (e *emitter) fieldDescs(t types.Type, st *types.Struct) string {
	class := e.dartType(t)
	// The class of an instance of a generic type writes each field with the
	// type that the declaration holds. The accessor reads the substituted type,
	// thus a function value needs the wrapper. Read funcAdapt.
	decl := st
	if named, ok := types.Unalias(t).(*types.Named); ok {
		if ref := e.structRef(named); ref.TypeArgs().Len() > 0 {
			if o, ok := ref.Origin().Underlying().(*types.Struct); ok {
				decl = o
			}
		}
	}
	parts := make([]string, st.NumFields())
	for i := range parts {
		field := st.Field(i)
		embedded := ""
		if field.Embedded() {
			embedded = ", embedded: true"
		}
		if !field.Exported() && field.Pkg() != nil {
			// An unnamed struct type holds no package of its own, and the
			// package reflect reads the package of each unexported field. Go
			// names the package of a command main, and the emitter names it
			// with the path of the module.
			path := field.Pkg().Path()
			if field.Pkg().Name() == "main" {
				path = "main"
			}
			embedded += ", pkg: " + dartString(path)
		}
		declType := field.Type()
		if i < decl.NumFields() {
			declType = decl.Field(i).Type()
		}
		parts[i] = fmt.Sprintf("FieldDesc('%s', %s, %t, () => %s, %s%s)",
			field.Name(), dartString(st.Tag(i)), field.Exported(),
			e.descriptorOf(field.Type()),
			e.fieldAccess(t, field, i, class, declType), embedded)
	}
	return ", fields: <FieldDesc>[" + strings.Join(parts, ", ") + "]"
}

// fieldAccess writes the getter and the setter of 1 field.
//
// Generic instance descriptors can stand outside the declaring package.
// Generated accessors expose their private Dart fields to those descriptors.
func (e *emitter) fieldAccess(owner types.Type, field *types.Var, i int,
	class string, declType types.Type) string {
	dart := structFieldName(field, i)
	if field.Name() != "_" {
		dart = e.fieldDartName(owner, field.Name())
	}
	if !field.Exported() && field.Pkg() != nil && field.Pkg() != e.pkg {
		if named, ok := types.Unalias(owner).(*types.Named); ok && named.TypeArgs().Len() > 0 {
			dart = fmt.Sprintf("$field%d", i)
		} else {
			return "$noRead, $noWrite"
		}
	}
	read := fmt.Sprintf("($r as %s).%s", class, dart)
	write := fmt.Sprintf("$v as %s", e.dartType(field.Type()))
	return fmt.Sprintf("($r) => %s, ($r, $v) => ($r as %s).%s = %s",
		e.funcAdapt(read, declType, field.Type()), class, dart,
		e.funcAdapt(write, field.Type(), declType))
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
	case *types.Basic:
		return basicText(t)
	case *types.Named:
		// reflect writes the type arguments of an instance with no space
		// behind a comma: `main.pair[int,string]`. go/types writes 1 space
		// there, thus the fallback below does not fit an instance.
		if args := t.TypeArgs(); args != nil && args.Len() > 0 {
			parts := make([]string, args.Len())
			for i := range parts {
				parts[i] = reflectText(args.At(i))
			}
			name := t.Obj().Name()
			if pkg := t.Obj().Pkg(); pkg != nil {
				name = shortQualifier(pkg) + "." + name
			}
			return name + "[" + strings.Join(parts, ",") + "]"
		}
	}
	return types.TypeString(t, shortQualifier)
}

// basicText names a predeclared type in the form that reflect gives.
//
// byte and uint8 name 1 type, and so do rune and int32. Go prints the name of
// the element type, thus uint8 and not byte. stripNames holds the same rule
// for the identity of a type.
func basicText(b *types.Basic) string {
	if b.Kind() == types.UnsafePointer {
		return b.String()
	}
	return types.Typ[b.Kind()].Name()
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
		return basicText(u)
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
	// 1 Dart function serves each instance of a generic declaration. The
	// parameter of the target thus reads the origin, and a function value of a
	// narrower Dart type needs the wrapper. Read funcAdapt.
	orig := signatureOf(fn.Origin().Type())
	params := make([]string, selSig.Params().Len())
	args := make([]string, selSig.Params().Len())
	for i := range params {
		name := fmt.Sprintf("$a%d", i)
		from := selSig.Params().At(i).Type()
		params[i] = e.dartType(from) + " " + name
		args[i] = name
		if orig != nil && i < orig.Params().Len() {
			args[i] = e.funcAdapt(name, from, orig.Params().At(i).Type())
		}
	}
	recv := fmt.Sprintf("($r as %s)", e.dartType(elem))
	_, wantPtr := types.Unalias(sig.Recv().Type()).(*types.Pointer)
	if pointer && !needsBox(elem) {
		// Go stops when a value method runs with a nil pointer receiver, and
		// the Dart cast of null gives a cast error.
		recv = fmt.Sprintf("($r == null ? %s : ($r as %s))",
			e.wrapPanic(elem, fn), e.dartType(elem))
	}
	if pointer && needsBox(elem) {
		ptr := e.dartType(types.NewPointer(elem))
		if wantPtr {
			recv = fmt.Sprintf("($r as %s)", ptr)
		} else {
			recv = fmt.Sprintf("($r == null ? %s : ($r as %s)!.v)",
				e.wrapPanic(elem, fn), ptr)
		}
	}
	// The Dart form of a pointer to a struct accepts null, thus the cast must
	// accept it too. Go runs a method with a nil pointer receiver.
	if pointer && !needsBox(elem) && len(sel.Index()) == 1 && topLevelMethod(elem, fn) {
		recv = fmt.Sprintf("($r as %s)", e.dartType(types.NewPointer(elem)))
	}
	index := sel.Index()
	current := elem
	// The last field of the path holds the receiver, and each field before it
	// only carries the walk. A pointer to a struct in the middle of the walk
	// needs the object behind it.
	path := index[:len(index)-1]
	for n, i := range path {
		st := structOf(current)
		if st == nil {
			e.fail("the method path of %s leaves a struct type", elem)
			return ""
		}
		field := st.Field(i)
		parent, holder := recv, current
		recv += "." + e.fieldDartName(current, field.Name())
		current = field.Type()
		if n == len(path)-1 {
			recv = e.fitReceiver(recv, parent, holder, field.Name(), current, sig)
			break
		}
		if isPointerType(current) && structOf(current) != nil {
			recv = nilRead(recv, false)
		}
	}
	call := fmt.Sprintf("%s.%s(%s)", recv, e.memberName(fn), strings.Join(args, ", "))
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
			recv, e.methodKeyText(fn.Id(), selSig), strings.Join(args, ", "))
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
	return fmt.Sprintf("%s: (%s) => %s", e.methodKeyText(fn.Id(), selSig), head, call)
}

// wrapPanic writes the panic of a method wrapper that a nil pointer receiver
// reached. Go names the value method with its package, and the pointer type
// with the name of the type alone.
func (e *emitter) wrapPanic(elem types.Type, fn *types.Func) string {
	name := reflectText(elem)
	short := name
	if i := strings.LastIndex(name, "."); i >= 0 {
		short = name[i+1:]
	}
	return fmt.Sprintf("$panicWrap(%s, %s, %s)",
		dartString(name), dartString(fn.Name()), dartString(short))
}

// nilRead gives the value behind a pointer of a method path. Go panics with
// the nil dereference error, and the bare null check of Dart gives an error of
// Dart. The text holds a field path of the receiver, thus it runs no effect.
func nilRead(text string, box bool) string {
	if box {
		return "(" + text + " == null ? $panicNil() : " + text + "!.v)"
	}
	return "(" + text + " == null ? $panicNil() : " + text + "!)"
}

// fitReceiver adjusts the last field of a method path to the receiver of the
// method. An embedded field and the receiver can differ in form: Go reads
// through a pointer field for a value receiver, and it takes the address of a
// value field for a pointer receiver.
//
// A pointer to a struct is the Dart object itself, thus a pointer receiver on
// a field of such a struct needs no address.
func (e *emitter) fitReceiver(recv, parent string, holder types.Type,
	fieldName string, field types.Type, sig *types.Signature) string {
	_, wantPtr := types.Unalias(sig.Recv().Type()).(*types.Pointer)
	p, isPtr := typeset.Core(field).(*types.Pointer)
	if isPtr && !wantPtr {
		return nilRead(recv, needsBox(p.Elem()))
	}
	if !isPtr && wantPtr && needsBox(field) {
		owner := holder
		if hp, ok := typeset.Core(owner).(*types.Pointer); ok {
			owner = hp.Elem()
		}
		name := e.fieldDartName(holder, fieldName)
		return fmt.Sprintf(
			"$Ref<%s, %s>(%s, %d, ($r, $i) => $r.%s, ($r, $i, $v) => $r.%s = $v)",
			e.dartType(owner), e.dartType(field), parent,
			fieldIndex(owner, fieldName), name, name)
	}
	return recv
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
	for i, held := range e.inlineDescTypes {
		if types.Identical(t, held) {
			return fmt.Sprintf("$inlineDesc%d", i)
		}
	}
	u := types.Unalias(t)
	// A type parameter reads its description from its dictionary. The caller
	// gives 1 dictionary for each type argument.
	if tp, ok := u.(*types.TypeParam); ok {
		return e.dictOf(tp) + ".desc"
	}
	if isDartObjectPointer(u) {
		return e.groundDescVar(t)
	}
	if named, ok := u.(*types.Named); ok && isDartObject(named.Obj()) {
		return e.groundDescVar(t)
	}
	if p, ok := u.(*types.Pointer); ok {
		// A generic function holds no variable for the description of a
		// pointer to its type parameter. The runtime reads the description of
		// the type argument, thus the method table of the pointer stays, and
		// Implements of it answers as Go does.
		if tp, ok := types.Unalias(p.Elem()).(*types.TypeParam); ok {
			return "$ptrDesc(" + e.dictOf(tp) + ".desc)"
		}
		if named, ok := types.Unalias(p.Elem()).(*types.Named); ok {
			if isInstance(named) {
				return e.instDescVar(named, true)
			}
			if len(outerTypeParams(named)) > 0 {
				return e.instanceDesc(named, true)
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
			if len(outerTypeParams(named)) > 0 {
				return e.instanceDesc(named, false)
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
	// The library holds 1 variable for the description of a type with no type
	// parameter, thus 2 uses of 1 type write the text once.
	if !holdsTypeParam(t) {
		if key := e.groundDescVar(t); key != "" {
			return key
		}
	}
	return e.inlineDescriptor(t)
}

// groundDescVar names the variable of the library that holds the description of
// a type with no type parameter, and it registers the type on the first read.
//
// The name is empty when the description reads a dictionary of the enclosing
// function, because such a text stands in no variable of the library. It is
// also empty when 2 type names meet at 1 key, because 1 variable serves 1 type.
func (e *emitter) groundDescVar(t types.Type) string {
	name := e.typeName(t)
	sum := sha256.Sum256([]byte(name))
	key := e.nilPrefix + "$desc" + hex.EncodeToString(sum[:4])
	if held, ok := e.instDescs[key]; ok {
		if e.typeName(held) != name {
			return ""
		}
		return key
	}
	before := e.dictReads
	// Reserve the descriptor before its pointer descriptor refers back to it.
	e.instDescs[key] = t
	text := e.inlineDescriptor(t)
	if e.dictReads != before {
		delete(e.instDescs, key)
		return ""
	}
	e.descTexts[key] = text
	return key
}

// inlineDescriptor writes the description of a type that holds no name of its
// own.
//
// An unnamed struct type holds the promoted method of each embedded field,
// thus its description also needs a method table. A pointer to such a struct
// holds the same set, and each pointer method of it. An interface type holds
// the key of each method that it needs, because a type assertion to a type
// parameter reads that set. A slice, a map, and a channel hold no method.
func (e *emitter) inlineDescriptor(t types.Type) string {
	u := types.Unalias(t)
	self := ""
	_, structure := u.(*types.Struct)
	if holdsTypeParam(t) && (isInterfaceType(t) || structure &&
		types.NewMethodSet(types.NewPointer(t)).Len() > types.NewMethodSet(t).Len()) {
		// A recursive generic descriptor needs a local reference.
		self = fmt.Sprintf("$inlineDesc%d", len(e.inlineDescTypes))
		e.inlineDescTypes = append(e.inlineDescTypes, t)
		defer func() { e.inlineDescTypes = e.inlineDescTypes[:len(e.inlineDescTypes)-1] }()
	}
	table := "const <String, Function>{}"
	methodTypes := ""
	pointerParts := ""
	owner := u
	_, pointer := owner.(*types.Pointer)
	if p, ok := owner.(*types.Pointer); ok {
		owner = types.Unalias(p.Elem())
	}
	switch {
	case isStructUnder(owner):
		pointerParts = e.pointerPart(owner, pointer)
		if entries := e.methodTable(owner, t); len(entries) > 0 {
			table = "<String, Function>{" + strings.Join(entries, ", ") + "}"
			methodTypes = e.methodTypeTable(t)
		}
	case isInterfaceType(t):
		methodTypes = e.methodTypeTable(t)
		iface := types.Unalias(t).Underlying().(*types.Interface)
		if iface.NumMethods() > 0 {
			parts := make([]string, iface.NumMethods())
			for i := range parts {
				method := iface.Method(i)
				sig := method.Type().(*types.Signature)
				parts[i] = e.methodKeyText(method.Id(), sig) + ": $ifaceMethod"
			}
			table = "<String, Function>{" + strings.Join(parts, ", ") + "}"
		}
	}
	text := fmt.Sprintf("TypeDesc(%s, '%s', %s, text: %s%s, zero: () => %s)",
		e.typeNameText(t), kindOf(t), table, e.reflectTextExpr(t),
		methodTypes+e.descParts(t)+pointerParts, e.zero(t))
	if self != "" {
		return fmt.Sprintf("(() { late final TypeDesc %s; %s = %s; return %s; })()", self, self, text, self)
	}
	return text
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
	case types.Complex64:
		return "$typeComplex64"
	case types.Complex128, types.UntypedComplex:
		return "$typeComplex128"
	}
	return ""
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
	case *types.Basic:
		// byte and uint8 name 1 type, and so do rune and int32. go/types holds
		// 2 objects with 2 texts, thus the key comes from the kind. A method
		// that spells uint8 must satisfy an interface that spells byte.
		return types.Typ[u.Kind()]
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
//
// A fresh expression needs no copy. Such an expression builds a value that no
// other expression names yet, thus the box holds the only reference. A tree of
// composite literals, for example a widget tree, then costs 1 object for each
// node in place of 2. ir.Fresh holds the rule, and stage 4 reads the same 1.
func (e *emitter) box(x ir.Expr, from types.Type) string {
	text := e.expr(x)
	fresh := ir.Fresh(x)
	switch {
	case isStructType(from) && !fresh:
		text = e.copyOf(e.deref(x), from)
	case isArrayType(from) && !fresh:
		text = e.copyArray(text, from)
	}
	// The argument of a type parameter can be an interface type, and a value
	// of such a parameter already holds an interface value.
	if _, ok := types.Unalias(from).(*types.TypeParam); ok {
		return fmt.Sprintf("$boxDesc(%s, %s)", e.descriptorOf(from), text)
	}
	// A generic function writes a cell for the address of a value. The Dart
	// form of a pointer to a struct and of a pointer to an array is the value
	// itself. The interface value thus reads the cell out.
	if p, ok := types.Unalias(from).(*types.Pointer); ok {
		if tp, ok := types.Unalias(p.Elem()).(*types.TypeParam); ok {
			return fmt.Sprintf("Iface(%s, $ptrValue(%s.desc, %s))",
				e.descriptorOf(from), e.dictOf(tp), text)
		}
	}
	return fmt.Sprintf("Iface(%s, %s)", e.descriptorOf(from), text)
}

// paramsHoldParam reports whether a parameter of a method reads a type
// parameter.
//
// The method table holds 1 adapter for each instance, and the Dart type of
// such a parameter is the type of the argument there. A call inside a generic
// body writes Object? for the same parameter, and a Dart function parameter is
// contravariant. The 2 forms thus fail the cast, and the call takes the
// dynamic path.
func paramsHoldParam(sig *types.Signature) bool {
	for i := 0; i < sig.Params().Len(); i++ {
		if typeset.HoldsParam(sig.Params().At(i).Type()) {
			return true
		}
	}
	return false
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
	if isPure(m.Recv) && !paramsHoldParam(sig) {
		recv := e.expr(m.Recv)
		params := []string{"Object?"}
		for i := 0; i < sig.Params().Len(); i++ {
			params = append(params, e.dartType(sig.Params().At(i).Type()))
		}
		all := append([]string{recv + "!.value"}, args...)
		return fmt.Sprintf("($method(%s, %s) as %s Function(%s))(%s)",
			recv, e.methodKeyText(m.Obj.Id(), sig), result,
			strings.Join(params, ", "), strings.Join(all, ", "))
	}
	call := fmt.Sprintf("$call(%s, %s, <Object?>[%s])",
		e.expr(m.Recv), e.methodKeyText(m.Obj.Id(), sig), strings.Join(args, ", "))
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
	// A type that mentions an unnamed struct type needs the exact text:
	// types.TypeString writes an embedded field as its type alone, and it
	// writes an alias by its name, thus 2 shapes that Go holds apart can give
	// 1 text. The name of a descriptor is the identity of the type at run
	// time, and a pointer to a shape must agree with the shape itself.
	if mentionsStruct(t, map[types.Type]bool{}) {
		return typeText(stripNames(types.Unalias(t))) + e.localMarks(t)
	}
	return types.TypeString(stripNames(types.Unalias(t)), nil) + e.localMarks(t)
}

// localMarks names the position of each local type that t mentions.
//
// 2 local types of 2 functions can share 1 printed name, and the identity of a
// type reads the name. A derived type of such a type holds the same printed
// name as the derived type of a package level type of that name, thus the walk
// reaches every level. The package encoding/json_test holds `type Animal int`
// at the top and `type Animal struct{...}` inside an example, and `[]Animal`
// of the example took the descriptor of `[]Animal` of the integer.
//
// The walk of a named type reads its type arguments alone. Go reads the name of
// such a type as its identity, thus the fields of the type behind it stay out.
func (e *emitter) localMarks(t types.Type) string {
	if e.cfg.Fset == nil {
		return ""
	}
	var out strings.Builder
	seen := map[types.Type]bool{}
	var walk func(types.Type)
	walk = func(t types.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
		switch u := types.Unalias(t).(type) {
		case *types.Named:
			if isLocalType(u.Obj()) {
				p := e.cfg.Fset.Position(u.Obj().Pos())
				fmt.Fprintf(&out, "·%d_%d", p.Line, p.Column)
			}
			if args := u.TypeArgs(); args != nil {
				for i := range args.Len() {
					walk(args.At(i))
				}
			}
		case *types.Pointer:
			walk(u.Elem())
		case *types.Slice:
			walk(u.Elem())
		case *types.Array:
			walk(u.Elem())
		case *types.Chan:
			walk(u.Elem())
		case *types.Map:
			walk(u.Key())
			walk(u.Elem())
		case *types.Signature:
			walk(u.Params())
			walk(u.Results())
		case *types.Tuple:
			for i := range u.Len() {
				walk(u.At(i).Type())
			}
		case *types.Struct:
			for i := range u.NumFields() {
				walk(u.Field(i).Type())
			}
		}
	}
	walk(t)
	return out.String()
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
	case *types.Named:
		// An alias inside a type argument also names its target. `List[any]`
		// and `List[interface{}]` name 1 Go type, thus the 2 texts must agree.
		if args := u.TypeArgs(); args != nil && args.Len() > 0 {
			list := make([]types.Type, args.Len())
			same := true
			for i := range list {
				list[i] = stripNames(args.At(i))
				same = same && list[i] == args.At(i)
			}
			if !same {
				if inst, err := types.Instantiate(nil, u.Origin(), list, false); err == nil {
					return inst
				}
			}
		}
		return u
	}
	return types.Unalias(t)
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
	var text string
	if !typeset.HoldsParam(t) && !hasOuterType(t) {
		text = "'" + dartText(e.typeName(t)) + "'"
	} else {
		var reads []string
		sub := e.substParams(t, &reads, false)
		text = interpolate(e.typeName(sub), reads)
	}
	return spliceOuter(text, e.outerReads(t, ".desc.name"))
}

// typeDisplayText writes the text that a report of a failed assertion names.
// Go names a type with the name of its package, and not with the import path
// of it, thus the text follows reflect. typeNameText holds the identity name,
// which a type test compares.
func (e *emitter) typeDisplayText(t types.Type) string {
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		return e.dictOf(tp) + ".desc.text"
	}
	if !typeset.HoldsParam(t) && !hasOuterType(t) {
		return "'" + dartText(reflectText(t)) + "'"
	}
	// A type that reads a type parameter keeps the identity name. Such a name
	// holds the import path, and no program of the suite reaches it.
	return e.typeNameText(t)
}

// outerReads gives the run time read of the description of each type argument
// of the generic function that declares a local type.
//
// Go names such a type after the arguments of the function and after its own
// arguments. `type T[B any]` inside `func F[A any]` gives `main.T[A;B]`, and a
// local type with no parameter of its own gives `main.S[A]`. 1 Dart function
// serves each instance of F, thus the name reads the dictionary.
func (e *emitter) outerReads(t types.Type, field string) []string {
	params := outerTypeParams(namedOf(t))
	if len(params) == 0 {
		return nil
	}
	out := make([]string, len(params))
	for i, tp := range params {
		out[i] = "${" + e.dictOf(tp) + field + "}"
	}
	return out
}

func hasOuterType(t types.Type) bool {
	switch u := types.Unalias(t).(type) {
	case *types.Named:
		if len(outerTypeParams(u)) > 0 {
			return true
		}
		for i := range u.TypeArgs().Len() {
			if hasOuterType(u.TypeArgs().At(i)) {
				return true
			}
		}
	case *types.Pointer:
		return hasOuterType(u.Elem())
	case *types.Slice:
		return hasOuterType(u.Elem())
	case *types.Array:
		return hasOuterType(u.Elem())
	case *types.Map:
		return hasOuterType(u.Key()) || hasOuterType(u.Elem())
	case *types.Chan:
		return hasOuterType(u.Elem())
	case *types.Struct:
		for i := range u.NumFields() {
			if hasOuterType(u.Field(i).Type()) {
				return true
			}
		}
	case *types.Signature:
		for _, tuple := range []*types.Tuple{u.Params(), u.Results()} {
			for i := range tuple.Len() {
				if hasOuterType(tuple.At(i).Type()) {
					return true
				}
			}
		}
	}
	return false
}

// Go retains the declaration number of a local type inside a type argument.
func localTypeNumber(obj *types.TypeName) int {
	count := 0
	var walk func(*types.Scope)
	walk = func(scope *types.Scope) {
		for _, name := range scope.Names() {
			tn, ok := scope.Lookup(name).(*types.TypeName)
			if !ok || tn.IsAlias() || !isLocalType(tn) || tn.Pos() > obj.Pos() {
				continue
			}
			if _, ok := tn.Type().(*types.Named); ok {
				count++
			}
		}
		for i := range scope.NumChildren() {
			walk(scope.Child(i))
		}
	}
	walk(obj.Pkg().Scope())
	return count
}

// outerTypeParams gives the type parameters of the generic function that
// declares a local type, in the order of the declaration.
func outerTypeParams(named *types.Named) []*types.TypeParam {
	if named == nil || !isLocalType(named.Obj()) {
		return nil
	}
	var out []*types.TypeParam
	for sc := named.Obj().Parent(); sc != nil && sc.Parent() != nil; sc = sc.Parent() {
		for _, n := range sc.Names() {
			tn, _ := sc.Lookup(n).(*types.TypeName)
			if tn == nil {
				continue
			}
			if tp, ok := types.Unalias(tn.Type()).(*types.TypeParam); ok {
				out = append(out, tp)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index() < out[j].Index() })
	return out
}

// spliceOuter puts the type arguments of the enclosing generic function into
// the Dart text of a type name.
func spliceOuter(text string, outer []string) string {
	if len(outer) == 0 {
		return text
	}
	list := strings.Join(outer, ",")
	if i := strings.Index(text, "["); i >= 0 {
		return text[:i+1] + list + ";" + text[i+1:]
	}
	if i := strings.Index(text, "\u00b7"); i >= 0 {
		return text[:i] + "[" + list + "]" + text[i:]
	}
	return text[:len(text)-1] + "[" + list + "]'"
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
	if sig == nil || (!typeset.HoldsParam(sig) && !hasOuterType(sig)) {
		return "'" + dartText(methodKey(name, sig)) + "'"
	}
	var reads []string
	sub, ok := e.substParams(sig, &reads, false).(*types.Signature)
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
		parts[i] = e.methodKeyText(m.Id(), sig)
	}
	sort.Strings(parts)
	return "<String>[" + strings.Join(parts, ", ") + "]"
}

// reflectTextExpr writes the Dart expression of the reflect text of a type.
// A type parameter reads the text of its dictionary at run time.
func (e *emitter) reflectTextExpr(t types.Type) string {
	var text string
	if !typeset.HoldsParam(t) && !hasOuterType(t) {
		text = "'" + dartText(reflectText(types.Unalias(t))) + "'"
	} else {
		var reads []string
		sub := e.substParams(t, &reads, true)
		text = interpolate(reflectText(types.Unalias(sub)), reads)
	}
	return spliceOuter(text, e.outerReads(t, ".desc.display"))
}

// substParams replaces each type parameter with a marker type, thus the name
// text splits at each parameter. The list holds the run time read of each
// marker, in order.
func (e *emitter) substParams(t types.Type, reads *[]string, display bool) types.Type {
	switch u := types.Unalias(t).(type) {
	case *types.TypeParam:
		marker := fmt.Sprintf("\x01%d\x01", len(*reads))
		field := ".desc.name"
		if display {
			field = ".desc.display"
		}
		*reads = append(*reads, e.dictOf(u)+field)
		obj := types.NewTypeName(token.NoPos, nil, marker, nil)
		return types.NewNamed(obj, types.Typ[types.Int], nil)
	case *types.Pointer:
		return types.NewPointer(e.substParams(u.Elem(), reads, display))
	case *types.Slice:
		return types.NewSlice(e.substParams(u.Elem(), reads, display))
	case *types.Array:
		return types.NewArray(e.substParams(u.Elem(), reads, display), u.Len())
	case *types.Map:
		return types.NewMap(e.substParams(u.Key(), reads, display), e.substParams(u.Elem(), reads, display))
	case *types.Chan:
		return types.NewChan(u.Dir(), e.substParams(u.Elem(), reads, display))
	case *types.Signature:
		params := make([]*types.Var, u.Params().Len())
		for i := range params {
			params[i] = types.NewVar(token.NoPos, nil, "", e.substParams(u.Params().At(i).Type(), reads, display))
		}
		results := make([]*types.Var, u.Results().Len())
		for i := range results {
			results[i] = types.NewVar(token.NoPos, nil, "", e.substParams(u.Results().At(i).Type(), reads, display))
		}
		return types.NewSignatureType(nil, nil, nil,
			types.NewTuple(params...), types.NewTuple(results...), u.Variadic())
	case *types.Struct:
		fields := make([]*types.Var, u.NumFields())
		tags := make([]string, u.NumFields())
		for i := range fields {
			f := u.Field(i)
			fields[i] = types.NewField(token.NoPos, f.Pkg(), f.Name(),
				e.substParams(f.Type(), reads, display), f.Embedded())
			tags[i] = u.Tag(i)
		}
		return types.NewStruct(fields, tags)
	case *types.Interface:
		methods := make([]*types.Func, u.NumExplicitMethods())
		for i := range methods {
			method := u.ExplicitMethod(i)
			sig := e.substParams(method.Type(), reads, display).(*types.Signature)
			methods[i] = types.NewFunc(token.NoPos, method.Pkg(), method.Name(), sig)
		}
		embedded := make([]types.Type, u.NumEmbeddeds())
		for i := range embedded {
			embedded[i] = e.substParams(u.EmbeddedType(i), reads, display)
		}
		return types.NewInterfaceType(methods, embedded).Complete()
	case *types.Named:
		if u.TypeArgs().Len() > 0 {
			args := make([]types.Type, u.TypeArgs().Len())
			for i := range args {
				arg := u.TypeArgs().At(i)
				if hasOuterType(arg) {
					marker := fmt.Sprintf("\x01%d\x01", len(*reads))
					text := e.typeNameText(arg)
					if display {
						text = e.reflectTextExpr(arg)
						if n, ok := types.Unalias(arg).(*types.Named); ok && isLocalType(n.Obj()) {
							text += fmt.Sprintf(" + '·%d'", localTypeNumber(n.Obj()))
						}
					}
					*reads = append(*reads, text)
					args[i] = types.NewNamed(types.NewTypeName(token.NoPos, nil, marker, nil), types.Typ[types.Int], nil)
				} else {
					args[i] = e.substParams(arg, reads, display)
				}
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
				e.typeDisplayText(x.Assert))
		}
		if isParam {
			return fmt.Sprintf("$assertDict(%s, %s, %s)", e.expr(x.X), e.dictOf(tp), e.reflectTextExpr(x.X.GoType()))
		}
		// $assertType gives Object?, thus a target of that Dart type needs no
		// cast. A pointer to a type parameter is such a target.
		//
		// The report of a failed assertion names the static type of the
		// operand, exactly as Go does: "interface {} is int, not string".
		static := dartString(reflectText(x.X.GoType()))
		want := dartString(reflectText(x.Assert))
		if dart := e.dartType(x.Assert); dart != "Object?" {
			return fmt.Sprintf("($assertType(%s, %s, %s, %s) as %s)",
				e.expr(x.X), e.typeNameText(x.Assert), static, want, dart)
		}
		return fmt.Sprintf("$assertType(%s, %s, %s, %s)",
			e.expr(x.X), e.typeNameText(x.Assert), static, want)
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

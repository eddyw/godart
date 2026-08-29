package emit

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/name"
	"github.com/eddyw/godart/internal/typeset"
)

// dartType gives the Dart type that holds a value of the Go type.
func (e *emitter) dartType(t types.Type) string {
	t = types.Unalias(t)
	switch t := t.(type) {
	case *types.Basic:
		return basicType(t)
	case *types.Named:
		if st, ok := t.Underlying().(*types.Struct); ok {
			if isLocalType(t.Obj()) {
				return e.anonClass(st)
			}
			return e.className(t.Obj()) + e.typeArgList(t.TypeArgs())
		}
		return e.dartType(t.Underlying())
	case *types.TypeParam:
		// A type parameter whose core type is a composite holds 1 Dart type.
		// `S ~[]E` gives `Slice<E>`, thus the parameter S adds nothing.
		//
		// A basic core type keeps the parameter. `~int` and `~string` give 2
		// Dart types, and the bound of the parameter holds what they share.
		if core := typeset.Core(t); core != nil && !isBasicType(core) {
			return e.dartType(core)
		}
		// A type set of integer types gives 1 Dart type. The dictionary holds
		// the mask of the width and the order of the sign.
		if set, ok := typeset.Describe(t); ok && set.Integer {
			return "int"
		}
		return name.Local(t.Obj().Name())
	case *types.Pointer:
		if needsBox(t.Elem()) {
			return "Box<" + e.dartType(t.Elem()) + ">?"
		}
		return e.dartType(t.Elem()) + "?"
	case *types.Slice:
		return "Slice<" + e.dartType(t.Elem()) + ">"
	case *types.Array:
		return "List<" + e.dartType(t.Elem()) + ">"
	case *types.Map:
		return "GoMap<" + e.dartType(t.Key()) + ", " + e.dartType(t.Elem()) + ">"
	case *types.Chan:
		return "Chan<" + e.dartType(t.Elem()) + ">"
	case *types.Signature:
		return e.funcType(t) + "?"
	case *types.Tuple:
		return e.recordType(t)
	case *types.Interface:
		return "Iface?"
	case *types.Struct:
		return e.anonClass(t)
	}
	e.fail("the type %s is outside the subset", t)
	return "Object?"
}

func basicType(t *types.Basic) string {
	switch t.Kind() {
	case types.Bool, types.UntypedBool:
		return "bool"
	case types.String, types.UntypedString:
		return "GoString"
	case types.Float32, types.Float64, types.UntypedFloat:
		return "double"
	case types.UnsafePointer:
		return "Object?"
	}
	if t.Info()&types.IsInteger != 0 {
		return "int"
	}
	return "Object?"
}

// funcType gives the Dart type of a function value.
func (e *emitter) funcType(sig *types.Signature) string {
	params := make([]string, sig.Params().Len())
	for i := range params {
		params[i] = e.dartType(sig.Params().At(i).Type())
	}
	return e.resultType(sig) + " Function(" + strings.Join(params, ", ") + ")"
}

// resultType gives the Dart type that holds the results. A function with more
// than 1 result returns a record.
func (e *emitter) resultType(sig *types.Signature) string {
	rs := sig.Results()
	switch rs.Len() {
	case 0:
		return "void"
	case 1:
		return e.dartType(rs.At(0).Type())
	}
	return e.recordType(rs)
}

func (e *emitter) recordType(t *types.Tuple) string {
	parts := make([]string, t.Len())
	for i := range parts {
		parts[i] = e.dartType(t.At(i).Type())
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// zero gives the Dart text of the zero value of a Go type.
func (e *emitter) zero(t types.Type) string {
	u := types.Unalias(t)
	if tp, ok := u.(*types.TypeParam); ok {
		return e.dictOf(tp) + ".zero()"
	}
	if named, ok := u.(*types.Named); ok {
		if st, isStruct := named.Underlying().(*types.Struct); isStruct {
			if isLocalType(named.Obj()) {
				return e.anonClass(st) + ".$zero()"
			}
			return e.className(named.Obj()) + e.typeArgList(named.TypeArgs()) +
				".$zero(" + strings.Join(e.dictArgs(named.TypeArgs()), ", ") + ")"
		}
		u = named.Underlying()
	}
	if st, ok := u.(*types.Struct); ok {
		return e.anonClass(st) + ".$zero()"
	}
	switch u := u.Underlying().(type) {
	case *types.Tuple:
		// A channel read gives the value and a flag. The pair is a Dart record.
		parts := make([]string, u.Len())
		for i := range parts {
			parts[i] = e.zero(u.At(i).Type())
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case *types.Basic:
		switch {
		case u.Kind() == types.Bool || u.Kind() == types.UntypedBool:
			return "false"
		case u.Kind() == types.String || u.Kind() == types.UntypedString:
			return "GoString.empty"
		case u.Info()&types.IsFloat != 0:
			return "0.0"
		case u.Info()&types.IsInteger != 0:
			return "0"
		}
		return "null"
	case *types.Slice:
		return "Slice<" + e.dartType(u.Elem()) + ">.nil()"
	case *types.Array:
		return e.makeArray(u)
	case *types.Map:
		return fmt.Sprintf("GoMap<%s, %s>.nil(() => %s)",
			e.dartType(u.Key()), e.dartType(u.Elem()), e.zero(u.Elem()))
	case *types.Chan:
		if !holdsTypeParam(u.Elem()) {
			return e.nilChan(u.Elem())
		}
		return fmt.Sprintf("Chan<%s>.nil(() => %s)", e.dartType(u.Elem()), e.zero(u.Elem()))
	case *types.Pointer, *types.Signature, *types.Interface:
		return "null"
	}
	e.fail("the type %s has no zero value in the subset", t)
	return "null"
}

func (e *emitter) makeArray(a *types.Array) string {
	return fmt.Sprintf("$makeArray<%s>(%d, () => %s)",
		e.dartType(a.Elem()), a.Len(), e.zero(a.Elem()))
}

// typeParamBound gives the Dart bound of a type parameter.
//
// A Go constraint lists a set of types, and Dart has no such form. The bound
// takes the Dart type that holds each type of the set, thus an operator of the
// set still applies.
func (e *emitter) typeParamBound(tp *types.TypeParam) string {
	terms, ok := typeset.Terms(tp.Constraint())
	if !ok || len(terms) == 0 {
		return ""
	}
	dart := ""
	numeric := true
	for _, t := range terms {
		text := e.dartType(t)
		if text != "int" && text != "double" {
			numeric = false
		}
		if dart == "" {
			dart = text
			continue
		}
		if dart != text {
			if numeric {
				dart = "num"
				continue
			}
			return ""
		}
	}
	if dart == "Object?" {
		return ""
	}
	return dart
}

// mixedForm reports whether the type set of a type parameter holds more than 1
// Dart type. Such a type parameter reads each operation from its dictionary,
// because no Dart operator fits each type argument.
func (e *emitter) mixedForm(tp *types.TypeParam) bool {
	if typeset.Core(tp) != nil {
		return false
	}
	set, ok := typeset.Describe(tp)
	return !ok || !set.Integer
}

// typeArgList writes the type arguments of an instance of a generic type.
func (e *emitter) typeArgList(args *types.TypeList) string {
	if args == nil || args.Len() == 0 {
		return ""
	}
	parts := make([]string, args.Len())
	for i := range parts {
		parts[i] = e.dartType(args.At(i))
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

// dictArgs writes the dictionary of each type argument.
func (e *emitter) dictArgs(args *types.TypeList) []string {
	if args == nil {
		return nil
	}
	out := make([]string, args.Len())
	for i := range out {
		out[i] = e.dictOf(args.At(i))
	}
	return out
}

// dictOf gives the run time data of 1 type. A generic function reads the zero
// value, the equality, and the hash of a type argument from it.
func (e *emitter) dictOf(t types.Type) string {
	u := types.Unalias(t)
	if tp, ok := u.(*types.TypeParam); ok {
		return "$d" + name.Local(tp.Obj().Name())
	}
	if b, ok := u.Underlying().(*types.Basic); ok {
		// A named type takes the dictionary of its underlying type only when
		// the description stays equal. A named type holds its own description.
		if _, named := u.(*types.Named); !named {
			if text := basicDict(b); text != "" {
				return text
			}
		}
	}
	dart := e.dartType(t)
	return fmt.Sprintf("Dict<%s>(() => %s, (a, b) => %s, (v) => %s, %s, %s, %s, %s)",
		dart, e.zero(t), e.equalText("a", "b", t), e.hashText("v", t), e.alloc(t),
		e.orderText(t), e.maskText(t), e.descriptorOf(t))
}

// equalText writes the Go equality of 2 values of a type.
func (e *emitter) equalText(a, b string, t types.Type) string {
	if isArrayType(t) {
		return fmt.Sprintf("$equalArray(%s, %s)", a, b)
	}
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		return fmt.Sprintf("%s.equal(%s, %s)", e.dictOf(tp), a, b)
	}
	if isReference(t) {
		return fmt.Sprintf("identical(%s, %s)", a, b)
	}
	return a + " == " + b
}

func (e *emitter) hashText(v string, t types.Type) string {
	if isArrayType(t) {
		return fmt.Sprintf("Object.hashAll(%s)", v)
	}
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		return fmt.Sprintf("%s.hash(%s)", e.dictOf(tp), v)
	}
	if isReference(t) {
		return fmt.Sprintf("identityHashCode(%s)", v)
	}
	return v + ".hashCode"
}

// isReference reports whether Go compares 2 values of the type by identity.
//
// A pointer, a channel, and a function value hold a reference. The Dart form of
// a pointer to a struct is the object of the struct, thus the == operator of
// Dart would read the fields. Go reads the address.
func isReference(t types.Type) bool {
	switch types.Unalias(t).Underlying().(type) {
	case *types.Pointer, *types.Chan, *types.Signature:
		return true
	}
	return false
}

// alloc gives a Dart function that builds the backing array of a slice. A
// numeric element type takes a typed data list.
func (e *emitter) alloc(elem types.Type) string {
	if tp, ok := types.Unalias(elem).(*types.TypeParam); ok {
		return e.dictOf(tp) + ".alloc"
	}
	if list := typedList(elem); list != "" {
		return fmt.Sprintf("(n) => %s(n)", list)
	}
	return fmt.Sprintf("(n) => List<%s>.generate(n, (_) => %s)",
		e.dartType(elem), e.zero(elem))
}

// typedList gives the typed data list of a numeric element type. Such a list
// holds the values without a box, thus it saves memory and time.
func typedList(t types.Type) string {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	if !ok {
		return ""
	}
	switch b.Kind() {
	case types.Int8:
		return "Int8List"
	case types.Int16:
		return "Int16List"
	case types.Int32:
		return "Int32List"
	case types.Int, types.Int64:
		return "Int64List"
	case types.Uint8:
		return "Uint8List"
	case types.Uint16:
		return "Uint16List"
	case types.Uint32:
		return "Uint32List"
	case types.Uint, types.Uint64, types.Uintptr:
		return "Uint64List"
	case types.Float32:
		return "Float32List"
	case types.Float64:
		return "Float64List"
	}
	return ""
}

func (e *emitter) className(obj *types.TypeName) string {
	return e.qualify(obj, name.Type(obj.Name()))
}

// qualify adds the import prefix when the object belongs to another package.
func (e *emitter) qualify(obj types.Object, dartName string) string {
	if obj.Pkg() == nil || obj.Pkg() == e.pkg {
		return dartName
	}
	prefix, ok := e.prefixes[obj.Pkg().Path()]
	if !ok {
		e.fail("the package %s has no import prefix", obj.Pkg().Path())
		return dartName
	}
	return prefix + "." + dartName
}

// genericType reports whether the named type holds type parameters. A method of
// such a type stays a member of its class, because a top level function would
// need the type parameters and the dictionaries of the receiver type.
func genericType(obj *types.TypeName) bool {
	named, ok := types.Unalias(obj.Type()).(*types.Named)
	return ok && named.TypeParams() != nil && named.TypeParams().Len() > 0
}

// pointerMethod reports whether the declaration of the method holds a pointer
// receiver. Such a method of a struct type holds its body in a top level
// function, because Go runs it with a nil receiver.
func pointerMethod(obj *types.Func) bool {
	sig, ok := obj.Type().(*types.Signature)
	return ok && sig.Recv() != nil && isPointerType(sig.Recv().Type())
}

// isUnsigned64 reports whether the type needs the unsigned helpers. A narrower
// unsigned type keeps a value that fits in a Dart int with no sign change.
func isUnsigned64(t types.Type) bool {
	if set, ok := typeset.Describe(t); ok && set.Unsigned64() {
		return true
	}
	b := basicOf(t)
	if b == nil {
		return false
	}
	switch b.Kind() {
	case types.Uint, types.Uint64, types.Uintptr:
		return true
	}
	return false
}

func isUnsigned(t types.Type) bool {
	if set, ok := typeset.Describe(t); ok && set.Unsigned64() {
		return true
	}
	b := basicOf(t)
	return b != nil && b.Info()&types.IsUnsigned != 0
}

func isInteger(t types.Type) bool {
	if set, ok := typeset.Describe(t); ok && set.Integer {
		return true
	}
	b := basicOf(t)
	return b != nil && b.Info()&types.IsInteger != 0
}

func isFloat(t types.Type) bool {
	b := basicOf(t)
	return b != nil && b.Info()&types.IsFloat != 0
}

func isString(t types.Type) bool {
	b := basicOf(t)
	return b != nil && b.Info()&types.IsString != 0
}

// basicOf gives the basic type behind a Go type. A type parameter gives the
// basic type of its type set, and only when each type of the set agrees.
func basicOf(t types.Type) *types.Basic {
	u := types.Unalias(t)
	if tp, ok := u.(*types.TypeParam); ok {
		terms, ok := typeset.Terms(tp.Constraint())
		if !ok || len(terms) == 0 {
			return nil
		}
		var out *types.Basic
		for _, term := range terms {
			b, ok := types.Unalias(term).Underlying().(*types.Basic)
			if !ok {
				return nil
			}
			if out == nil {
				out = b
				continue
			}
			if out.Kind() != b.Kind() {
				return nil
			}
		}
		return out
	}
	b, _ := u.Underlying().(*types.Basic)
	return b
}

// Each predicate below reads the core type. A type parameter whose type set
// shares 1 underlying type holds that type as its core type, thus an operation
// of the core type applies to the parameter.

// needsBox reports whether a pointer to the type needs a box.
//
// The Dart form of a struct and of an array already holds a reference, thus a
// pointer to one is the object itself. Each other type needs a box.
func needsBox(t types.Type) bool {
	return !isStructType(t) && !isArrayType(t)
}

// isLocalType reports whether a function declares the type.
//
// The output holds no class of such a type at package level. A local struct
// type takes the class of its shape, exactly as an unnamed struct type does.
func isLocalType(obj *types.TypeName) bool {
	return obj.Pkg() != nil && obj.Parent() != obj.Pkg().Scope()
}

func isBasicType(t types.Type) bool {
	_, ok := types.Unalias(t).(*types.Basic)
	return ok
}

func isStructType(t types.Type) bool {
	_, ok := typeset.Core(t).(*types.Struct)
	return ok
}

func isArrayType(t types.Type) bool {
	_, ok := typeset.Core(t).(*types.Array)
	return ok
}

func isMapType(t types.Type) bool {
	_, ok := typeset.Core(t).(*types.Map)
	return ok
}

func isChanType(t types.Type) bool {
	_, ok := typeset.Core(t).(*types.Chan)
	return ok
}

func isSliceType(t types.Type) bool {
	_, ok := typeset.Core(t).(*types.Slice)
	return ok
}

func isPointerType(t types.Type) bool {
	_, ok := typeset.Core(t).(*types.Pointer)
	return ok
}

// maskText gives the mask function of a dictionary. A type that needs no mask
// takes the identity.
func (e *emitter) maskText(t types.Type) string {
	if m := mask(t); m != "" {
		return m
	}
	return "$identity<" + e.dartType(t) + ">"
}

// mask gives the runtime function that keeps a value inside a narrow type.
func mask(t types.Type) string {
	if _, ok := types.Unalias(t).(*types.TypeParam); ok {
		return ""
	}
	b := basicOf(t)
	if b == nil {
		return ""
	}
	switch b.Kind() {
	case types.Int8:
		return "$i8"
	case types.Int16:
		return "$i16"
	case types.Int32:
		return "$i32"
	case types.Uint8:
		return "$u8"
	case types.Uint16:
		return "$u16"
	case types.Uint32:
		return "$u32"
	case types.Float32:
		return "$f32"
	}
	return ""
}

// holdsTypeParam reports whether a type reads a type parameter. A value of such
// a type takes no variable at package level.
func holdsTypeParam(t types.Type) bool {
	found := false
	var walk func(types.Type)
	seen := map[types.Type]bool{}
	walk = func(t types.Type) {
		if t == nil || found || seen[t] {
			return
		}
		seen[t] = true
		switch t := types.Unalias(t).(type) {
		case *types.TypeParam:
			found = true
		case *types.Slice:
			walk(t.Elem())
		case *types.Array:
			walk(t.Elem())
		case *types.Pointer:
			walk(t.Elem())
		case *types.Chan:
			walk(t.Elem())
		case *types.Map:
			walk(t.Key())
			walk(t.Elem())
		case *types.Named:
			for i := 0; i < t.TypeArgs().Len(); i++ {
				walk(t.TypeArgs().At(i))
			}
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				walk(t.Field(i).Type())
			}
		case *types.Signature:
			for i := 0; i < t.Params().Len(); i++ {
				walk(t.Params().At(i).Type())
			}
			for i := 0; i < t.Results().Len(); i++ {
				walk(t.Results().At(i).Type())
			}
		}
	}
	walk(t)
	return found
}

// basicDict names the dictionary of a predeclared type. The runtime library
// holds 1 value for each one.
func basicDict(b *types.Basic) string {
	switch b.Kind() {
	case types.Bool, types.UntypedBool:
		return "$dictBool"
	case types.String, types.UntypedString:
		return "$dictString"
	case types.Int:
		return "$dictInt"
	case types.Int8:
		return "$dictInt8"
	case types.Int16:
		return "$dictInt16"
	case types.Int32:
		return "$dictInt32"
	case types.Int64:
		return "$dictInt64"
	case types.Uint:
		return "$dictUint"
	case types.Uint8:
		return "$dictUint8"
	case types.Uint16:
		return "$dictUint16"
	case types.Uint32:
		return "$dictUint32"
	case types.Uint64:
		return "$dictUint64"
	case types.Uintptr:
		return "$dictUintptr"
	case types.Float32:
		return "$dictFloat32"
	case types.Float64, types.UntypedFloat:
		return "$dictFloat64"
	}
	return ""
}

// orderText gives the order function of a dictionary. Go holds no order for
// each type, thus a type outside the ordered set stops the program.
func (e *emitter) orderText(t types.Type) string {
	dart := e.dartType(t)
	u := types.Unalias(t).Underlying()
	b, ok := u.(*types.Basic)
	if !ok {
		return "$unordered<" + dart + ">"
	}
	switch {
	case b.Info()&types.IsString != 0:
		return "$compare"
	case isUnsigned64(t):
		return "(a, b) => $ult(a, b) ? -1 : (a == b ? 0 : 1)"
	case b.Info()&(types.IsInteger|types.IsFloat) != 0:
		return "(a, b) => a < b ? -1 : (a == b ? 0 : 1)"
	}
	return "$unordered<" + dart + ">"
}

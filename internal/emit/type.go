package emit

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
	"github.com/eddyw/godart/internal/typeset"
)

// dartType gives the Dart type that holds a value of the Go type.
func (e *emitter) dartType(t types.Type) string {
	// The guard caps a cycle through a function type. Each use of such a type
	// starts at its name, thus each use gives 1 text.
	if e.typeDepth > 64 {
		return "Object?"
	}
	e.typeDepth++
	defer func() { e.typeDepth-- }()
	t = types.Unalias(t)
	switch t := t.(type) {
	case *types.Basic:
		return basicType(t)
	case *types.Named:
		// A Dart object holds no Go form, thus the type erases to the top type
		// of Dart. Only a //gart:dart declaration gives such a value.
		if isDartObject(t.Obj()) {
			return "Object?"
		}
		if st, ok := t.Underlying().(*types.Struct); ok {
			ref := e.structRef(t)
			if isLocalType(ref.Obj()) {
				return e.anonType(st)
			}
			return e.className(ref.Obj()) + e.typeArgList(ref.TypeArgs())
		}
		// A type that names itself through a pointer, a slice, or a map has
		// no finite Dart form. Each use of such a type takes Object?, thus
		// the 2 sides of an assignment keep 1 form.
		if e.cyclic(t.Obj()) {
			return "Object?"
		}
		return e.dartType(t.Underlying())
	case *types.TypeParam:
		if arg := e.fieldTypeArgs[t]; arg != nil {
			// Type arguments already belong to the instance scope.
			previous := e.fieldTypeArgs
			e.fieldTypeArgs = nil
			defer func() { e.fieldTypeArgs = previous }()
			return e.dartType(arg)
		}
		// A composite core can supply the Dart representation directly.
		// `S ~[]E` gives `Slice<E>`.
		//
		// A basic core type keeps the parameter. `~int` and `~string` give 2
		// Dart types, and the bound of the parameter holds what they share.
		if core := typeset.Core(t); core != nil && !isBasicType(core) {
			if p, ok := core.(*types.Pointer); ok {
				if _, ok := types.Unalias(p.Elem()).(*types.TypeParam); ok {
					// Containers must retain the substituted pointer type.
					return e.typeParamName(t)
				}
			}
			// The core struct of a constraint is a shape of the compiler, and
			// the class of a type argument of the shape is not the class of
			// the shape. Each operation on such a parameter reads its
			// dictionary, thus the value takes no Dart type.
			if _, ok := core.(*types.Struct); ok {
				return "Object?"
			}
			// A constraint that names its own type parameter, for example
			// `~func() T` on the parameter T, has no finite Dart form. The
			// depth guard wrote 33 levels of it, and the 2 sides of a call
			// then held 2 forms.
			if mentionsParam(core, t, map[*types.TypeName]bool{}) {
				return "Object?"
			}
			return e.dartType(core)
		}
		// The dictionary preserves concrete precision within each representation.
		if set, ok := typeset.Describe(t); ok {
			if set.Integer {
				return "GoInt"
			}
			if set.Float {
				if b := basicOf(t); b != nil {
					return basicType(b)
				}
				return e.typeParamName(t)
			}
			if set.Complex {
				return "GoComplex"
			}
		}
		return e.typeParamName(t)
	case *types.Pointer:
		// The Dart form of a pointer to a type parameter follows the type
		// argument: a pointer to a struct is the object itself, and each other
		// pointer holds a cell. Thus such a pointer takes 1 form that holds
		// both, and $deref reads it.
		if _, ok := types.Unalias(t.Elem()).(*types.TypeParam); ok {
			return "Object?"
		}
		// A pointer to a type that names itself takes the form of that type.
		// `type Number *Number` names 1 Go type, thus the 2 spellings must
		// give 1 Dart form or an assignment of the 2 fails.
		if named, ok := types.Unalias(t.Elem()).(*types.Named); ok && e.cyclic(named.Obj()) {
			return "Object?"
		}
		if needsBox(t.Elem()) {
			return "$Ptr<" + e.dartType(t.Elem()) + ">?"
		}
		// The cap of the depth gives Object?, which already accepts null. A
		// second question mark is no Dart type.
		elem := e.dartType(t.Elem())
		if strings.HasSuffix(elem, "?") {
			return elem
		}
		return elem + "?"
	case *types.Slice:
		depth := 0
		for nested := types.Type(t); ; {
			sl, ok := nested.Underlying().(*types.Slice)
			if !ok {
				break
			}
			depth++
			if depth > deepSliceLevels {
				return "Slice"
			}
			nested = sl.Elem()
		}
		return "Slice<" + e.dartType(t.Elem()) + ">"
	case *types.Array:
		return "List<" + e.dartType(t.Elem()) + ">"
	case *types.Map:
		return "GoMap<" + e.dartType(t.Key()) + ", " + e.dartType(t.Elem()) + ">"
	case *types.Chan:
		return "Chan<" + e.dartType(t.Elem()) + ">"
	case *types.Signature:
		// A signature that names a cyclic type holds no finite Dart form
		// either. `func(*lexer) stateFn` is identical to the underlying type
		// of stateFn, thus the 2 must take 1 form.
		if e.mentionsCyclic(t, map[*types.TypeName]bool{}) {
			return "Object?"
		}
		return e.funcType(t) + "?"
	case *types.Tuple:
		return e.recordType(t)
	case *types.Interface:
		return "Iface?"
	case *types.Struct:
		return e.anonType(t)
	case *ir.MapIterType:
		return "GoMapIter<" + e.dartType(t.Map.Key()) + ", " + e.dartType(t.Map.Elem()) + ">"
	}
	e.fail("the type %s is outside the subset", t)
	return "Object?"
}

// Type parameter names stay outside the Go identifier namespace.
func (e *emitter) typeParamName(tp *types.TypeParam) string {
	if alias, ok := e.receiverParams[tp]; ok {
		return alias
	}
	if tp.Obj().Name() == "_" {
		return fmt.Sprintf("$T%d", tp.Index())
	}
	return "$tp$" + name.Local(tp.Obj().Name())
}

func basicType(t *types.Basic) string {
	switch t.Kind() {
	case types.Bool, types.UntypedBool:
		return "bool"
	case types.String, types.UntypedString:
		return "GoString"
	case types.Float32:
		return "GoFloat32"
	case types.Float64, types.UntypedFloat:
		return "double"
	case types.UnsafePointer:
		return "Object?"
	}
	if t.Info()&types.IsComplex != 0 {
		return "GoComplex"
	}
	if t.Info()&types.IsInteger != 0 {
		return "GoInt"
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
		if isDartObject(named.Obj()) {
			return "null"
		}
		if st, isStruct := named.Underlying().(*types.Struct); isStruct {
			ref := e.structRef(named)
			if isLocalType(ref.Obj()) {
				return e.anonZero(st)
			}
			return e.className(ref.Obj()) + e.typeArgList(ref.TypeArgs()) +
				".$zero(" + strings.Join(e.dictArgs(ref.TypeArgs()), ", ") + ")"
		}
		u = named.Underlying()
	}
	if st, ok := u.(*types.Struct); ok {
		return e.anonZero(st)
	}
	if _, ok := u.(*ir.MapIterType); ok {
		return "GoMapIter.empty()"
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
		case u.Kind() == types.Float32:
			return "GoFloat32.zero"
		case u.Info()&types.IsFloat != 0:
			return "0.0"
		case u.Kind() == types.Complex64:
			return "GoComplex32.zero"
		case u.Info()&types.IsComplex != 0:
			return "GoComplex.zero"
		case u.Info()&types.IsInteger != 0:
			return "GoInt.zero"
		}
		return "null"
	case *types.Slice:
		return "Slice<" + e.dartType(u.Elem()) + ">.nil()"
	case *types.Array:
		return e.makeArray(u)
	case *types.Map:
		return fmt.Sprintf("GoMap<%s, %s>.nil(%s%s)",
			e.dartType(u.Key()), e.dartType(u.Elem()), e.cycleZeroFunc(u.Elem()), e.mapKeyOptions(u))
	case *types.Chan:
		if !holdsTypeParam(u.Elem()) {
			return e.nilChan(u.Elem())
		}
		return fmt.Sprintf("Chan<%s>.nil(%s)", e.dartType(u.Elem()), e.cycleZeroFunc(u.Elem()))
	case *types.Pointer, *types.Signature, *types.Interface:
		return "null"
	}
	e.fail("the type %s has no zero value in the subset", t)
	return "null"
}

// zeroFunc gives a Dart function value that gives the zero value of a Go type.
// A map, a channel, and an array each take such a function.
//
// A tear off of a top level function or of a constructor stands in the place
// of a closure. Dart holds 1 such tear off for the life of the program, and a
// closure builds a fresh object at each read of the expression. A type with no
// tear off takes a top level function of the output file.
func (e *emitter) zeroFunc(t types.Type) string {
	u := types.Unalias(t)
	if tp, ok := u.(*types.TypeParam); ok {
		return e.dictOf(tp) + ".zero"
	}
	if named, ok := u.(*types.Named); ok {
		if st, isStruct := named.Underlying().(*types.Struct); isStruct {
			ref := e.structRef(named)
			if isLocalType(ref.Obj()) {
				return e.anonZeroFunc(st)
			}
			if ref.TypeArgs().Len() == 0 {
				return e.className(ref.Obj()) + ".$zero"
			}
			return e.zeroHoist(t)
		}
		u = named.Underlying()
	}
	if st, ok := u.(*types.Struct); ok {
		return e.anonZeroFunc(st)
	}
	switch u := u.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Kind() == types.Bool || u.Kind() == types.UntypedBool:
			return "$boolZero"
		case u.Kind() == types.String || u.Kind() == types.UntypedString:
			return "$stringZero"
		case u.Kind() == types.Float32:
			return "$float32Zero"
		case u.Info()&types.IsFloat != 0:
			return "$doubleZero"
		case u.Kind() == types.Complex64:
			return "$complex32Zero"
		case u.Info()&types.IsComplex != 0:
			return "$complexZero"
		case u.Info()&types.IsInteger != 0:
			return "$intZero"
		}
		return "$nullZero"
	case *types.Slice:
		return "Slice<" + e.dartType(u.Elem()) + ">.nil"
	case *types.Pointer, *types.Signature, *types.Interface:
		return "$nullZero"
	}
	return e.zeroHoist(t)
}

// anonZeroFunc gives the zero function of an unnamed struct shape. A shape
// with no dictionary takes the tear off of its zero constructor.
func (e *emitter) anonZeroFunc(st *types.Struct) string {
	if len(e.anonDicts(st)) == 0 {
		return e.anonType(st) + ".$zero"
	}
	return e.zeroHoist(st)
}

// zeroHoist writes a top level function that gives the zero value of a type,
// and gives its name. 2 types with 1 Dart form share 1 function.
//
// A zero value that reads the dictionary of a type parameter names a local of
// the enclosing function, thus it stays a closure.
func (e *emitter) zeroHoist(t types.Type) string {
	if e.lits == nil || holdsTypeParam(t) {
		return "() => " + e.zero(t)
	}
	reads := e.dictReads
	body := e.zero(t)
	if e.dictReads != reads {
		return "() => " + body
	}
	if e.lits.zeros == nil {
		e.lits.zeros = map[string]string{}
	}
	if name, ok := e.lits.zeros[body]; ok {
		return name
	}
	name := fmt.Sprintf("$zf%d_%d", e.lits.tag, e.lits.seq)
	e.lits.seq++
	e.lits.zeros[body] = name
	e.lits.decls = append(e.lits.decls,
		fmt.Sprintf("%s %s() => %s;", e.dartType(t), name, body))
	return name
}

// Anonymous storage takes 1 dictionary per field.
func (e *emitter) anonZero(st *types.Struct) string {
	return fmt.Sprintf("%s.$zero(%s)", e.anonType(st),
		strings.Join(e.anonDicts(st), ", "))
}

// cycleZeroFunc gives the zero function of an element type of a zero value.
// The Dart type of a type that names itself erases to Object?, thus null is
// its zero, and the text of the true zero has no end.
func (e *emitter) cycleZeroFunc(t types.Type) string {
	if n, ok := types.Unalias(t).(*types.Named); ok && e.cyclic(n.Obj()) {
		return "$nullZero"
	}
	return e.zeroFunc(t)
}

// coreDispatch reports whether a method call on a type parameter with this
// core type goes through the method table. A struct constraint holds no single
// Dart class for all its type arguments.
func coreDispatch(core types.Type) bool {
	if core == nil {
		return true
	}
	switch core := types.Unalias(core).(type) {
	case *types.Pointer:
		_, param := types.Unalias(core.Elem()).(*types.TypeParam)
		return param
	}

	return true
}

// chanExpr writes a channel expression for a member call. The Dart type of a
// type that names itself erases to Object?, thus the call needs the real
// channel type back.
func (e *emitter) chanExpr(x ir.Expr) string {
	text := e.expr(x)
	if e.dartType(x.GoType()) != "Object?" {
		return text
	}
	ch, ok := typeset.Core(x.GoType()).(*types.Chan)
	if !ok {
		return text
	}
	return fmt.Sprintf("(%s as Chan<%s>)", text, e.dartType(ch.Elem()))
}

func (e *emitter) makeArray(a *types.Array) string {
	return fmt.Sprintf("$makeArray<%s>(%d, %s)",
		e.dartType(a.Elem()), a.Len(), e.zeroFunc(a.Elem()))
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
	for _, t := range terms {
		// A struct term names a shape of the compiler, and no Dart bound
		// holds the class of each type argument of the shape.
		if _, ok := types.Unalias(t).(*types.Struct); ok {
			return ""
		}
		// A term that names the parameter itself, for example `~func() T` on
		// T, gives no finite bound. A type argument of such a parameter
		// erases to Object?, thus no bound of Dart accepts it.
		if mentionsParam(t, tp, map[*types.TypeName]bool{}) {
			return ""
		}
		text := e.dartType(t)
		if dart == "" {
			dart = text
			continue
		}
		if dart != text {
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
	return !ok || (!set.Integer && !set.Complex)
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
		// The name stands for a local of the enclosing function, thus a
		// description that holds it stands in no variable of the library.
		e.dictReads++
		return "$d" + e.typeParamName(tp)
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
	body := fmt.Sprintf("Dict<%s>(() => %s, (a, b) => %s, (v) => %s, %s, %s, %s, %s%s)",
		dart, e.zero(t), e.equalText("a", "b", t), e.hashText("v", t), e.alloc(t),
		e.orderText(t), e.maskText(t), e.descriptorOf(t), e.dictCopy(t)+dictText(t))
	if e.lits == nil {
		return body
	}
	if e.lits.dicts == nil {
		e.lits.dicts = map[string]string{}
	}
	params := dictionaryParams(t)
	declaration := e.typeParamDecl(params)
	key := declaration + body
	if value, ok := e.lits.dicts[key]; ok {
		return value
	}
	// Local dictionary closures can retain unrelated Go closure captures.
	name := fmt.Sprintf("$dict%d_%d", e.lits.tag, e.lits.seq)
	e.lits.seq++
	value := name
	if len(params) == 0 {
		e.lits.decls = append(e.lits.decls, fmt.Sprintf("final %s = %s;", name, body))
	} else {
		arguments := make([]string, len(params))
		for i, param := range params {
			arguments[i] = e.dictOf(param)
		}
		value += e.typeArgsOfParams(params) + "(" + strings.Join(arguments, ", ") + ")"
		e.lits.decls = append(e.lits.decls, fmt.Sprintf("Dict<%s> %s%s(%s) => %s;",
			dart, name, declaration, strings.Join(e.dictParams(params), ", "), body))
	}
	e.lits.dicts[key] = value
	return value
}

// dictText gives the byte operations of a dictionary. A type set that holds a
// byte slice and a string holds 2 Dart types, thus each such operation reads
// the dictionary.
func dictText(t types.Type) string {
	u := types.Unalias(t).Underlying()
	if isRuneSlice(t) {
		return ", text: (v) => $runesToString(v), fromText: (v) => $stringToRunes(v)"
	}
	if b, ok := u.(*types.Basic); ok && b.Kind() == types.String {
		return ", length: (v) => $gi(v.length), at: (v, i, [uns = false]) => v.$at(i, uns)," +
			" slice: (v, a, b, [au = false, bu = false]) => $substring(v, a, b, au, bu)," +
			" bytes: (v) => $stringToBytes(v), text: (v) => v," +
			" fromBytes: (v) => $bytesToString(v), fromText: (v) => v"
	}
	s, ok := u.(*types.Slice)
	if !ok {
		return ""
	}
	if b, ok := types.Unalias(s.Elem()).Underlying().(*types.Basic); !ok ||
		b.Kind() != types.Uint8 {
		return ""
	}
	return ", length: (v) => $gi(v.len), at: (v, i, [uns = false]) => v.$at(i, uns)," +
		" slice: (v, a, b, [au = false, bu = false]) => v.range(a, b, $gi(v.cap), false, false, au, bu)," +
		" bytes: (v) => v, text: (v) => $bytesToString(v)," +
		" fromBytes: (v) => v, fromText: (v) => $stringToBytes(v)"
}

// dictCopy gives the copy and the assign of a dictionary. A struct and an
// array hold value semantics, thus a copy point reads them from the
// dictionary of a type parameter.
func (e *emitter) dictCopy(t types.Type) string {
	switch {
	case isStructType(t):
		return ", copy: (v) => v.$copy(), assign: (d, s) { d.$assign(s); return d; }"
	case isArrayType(t):
		return fmt.Sprintf(", copy: (v) => %s, assign: (d, s) { %s; return d; }",
			e.copyOf("v", t), e.assignElem("d", "s", t))
	}
	return ""
}

// equalText writes the Go equality of 2 values of a type.
func (e *emitter) equalText(a, b string, t types.Type) string {
	if isDartObjectPointer(t) {
		return fmt.Sprintf("identical(%s, %s)", a, b)
	}
	if isArrayType(t) {
		return fmt.Sprintf("$goEqual(%s, %s, %s)", a, b, e.descriptorOf(t))
	}
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		return fmt.Sprintf("%s.equal(%s, %s)", e.dictOf(tp), a, b)
	}
	if _, ok := types.Unalias(t).Underlying().(*types.Pointer); ok {
		return fmt.Sprintf("$goEqual(%s, %s, %s)", a, b, e.descriptorOf(t))
	}
	if isReference(t) {
		return fmt.Sprintf("identical(%s, %s)", a, b)
	}
	return a + " == " + b
}

func (e *emitter) hashText(v string, t types.Type) string {
	if isDartObjectPointer(t) {
		return fmt.Sprintf("identityHashCode(%s)", v)
	}
	if isArrayType(t) {
		return fmt.Sprintf("$goHash(%s, %s)", v, e.descriptorOf(t))
	}
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		return fmt.Sprintf("%s.hash(%s)", e.dictOf(tp), v)
	}
	if _, ok := types.Unalias(t).Underlying().(*types.Pointer); ok {
		return fmt.Sprintf("$goHash(%s, %s)", v, e.descriptorOf(t))
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
	zero, known := zeroSized(elem)
	if zero {
		return fmt.Sprintf("(n) => $ZeroList<%s>(n, () => %s)", e.dartType(elem), e.zero(elem))
	}
	if !known {
		return fmt.Sprintf("(n) => $typeSize(%s) == GoInt.zero ? $ZeroList<%s>(n, () => %s) : List<%s>.generate(n, (_) => %s)", e.descriptorOf(elem), e.dartType(elem), e.zero(elem), e.dartType(elem), e.zero(elem))
	}
	if fn := goList(elem); fn != "" {
		return fn
	}
	if list := typedList(elem); list != "" {
		return fmt.Sprintf("(n) => %s(n)", list)
	}
	return fmt.Sprintf("(n) => List<%s>.generate(n, (_) => %s)",
		e.dartType(elem), e.zero(elem))
}

// goList gives the allocator of a Go integer element type. The native form
// keeps the dense list of the width, and the JS form gives a plain list of
// pairs, thus the caller cannot name the Dart class of the list.
func goList(t types.Type) string {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	if !ok || b.Info()&types.IsInteger == 0 {
		return ""
	}
	switch b.Kind() {
	case types.Int8:
		return "$goList8"
	case types.Int16:
		return "$goList16"
	case types.Int32:
		return "$goList32"
	case types.Uint8:
		return "$goListU8"
	case types.Uint16:
		return "$goListU16"
	case types.Uint32:
		return "$goListU32"
	case types.Uint, types.Uint64, types.Uintptr:
		return "$goListU64"
	}
	return "$goList64"
}

// typedList gives the typed data list of a float element type. Such a list
// holds the values without a box, thus it saves memory and time.
func typedList(t types.Type) string {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	if !ok {
		return ""
	}
	switch b.Kind() {
	case types.Float32:
		return "$Float32List"
	case types.Float64:
		return "Float64List"
	}
	return ""
}

func (e *emitter) className(obj *types.TypeName) string {
	rep := e.mergeRep(obj)
	return e.qualify(rep, name.Type(rep.Name()))
}

// mergeGroup gives each named struct type of 1 package that holds the struct
// object of obj. A group of 1 member gives no result.
//
// `type outer inner` gives outer the underlying type of inner, and go/types
// holds 1 struct object for the 2 names. `(*outer)(p)` then names the value
// that p names, and a write through the result reaches that value. The 2 types
// thus need 1 Dart class. A struct that a second declaration spells again
// holds a struct object of its own, and it keeps its own class.
//
// A group holds 1 package. The class of a group carries the methods of each
// member, and a group of 2 packages would put the methods of 1 package in the
// library of the other one.
//
// A generic type stays out. Each instance of such a type holds a struct object
// of its own.
func (e *emitter) mergeGroup(obj *types.TypeName) []*types.TypeName {
	pkg := obj.Pkg()
	if pkg == nil {
		return nil
	}
	st := structObject(obj)
	if st == nil {
		return nil
	}
	group := e.structNames(pkg)[st]
	if len(group) < 2 {
		return nil
	}
	return group
}

// structNames gives each named struct type of 1 package for each struct object
// that the package holds. The scope gives the names in order, thus the first
// member of a list is stable.
func (e *emitter) structNames(pkg *types.Package) map[*types.Struct][]*types.TypeName {
	if byStruct, ok := e.merges[pkg]; ok {
		return byStruct
	}
	byStruct := map[*types.Struct][]*types.TypeName{}
	scope := pkg.Scope()
	for _, n := range scope.Names() {
		if st, tn := packageStruct(scope, n); st != nil {
			byStruct[st] = append(byStruct[st], tn)
		}
	}
	e.merges[pkg] = byStruct
	return byStruct
}

// structObject gives the struct object of a named struct type that holds no
// type parameter and no type argument. It gives nil for each other type.
func structObject(obj *types.TypeName) *types.Struct {
	named, _ := types.Unalias(obj.Type()).(*types.Named)
	if named == nil || named.TypeParams().Len() > 0 || named.TypeArgs().Len() > 0 {
		return nil
	}
	st, _ := named.Underlying().(*types.Struct)
	return st
}

// foreignRep gives the named struct type of another package that owns the Dart
// class of the struct object of obj.
//
// `type X q.Y` gives X the struct object of q.Y, and go/types holds 1 struct
// object for the 2 names. A conversion between the 2 thus names 1 value, and
// the 2 types need 1 Dart form. X takes the form of q.Y, and it declares no
// class of its own.
//
// A method with a value receiver stands in the Dart class of its type, and the
// class of q.Y cannot hold a method of the package of X. Such a type thus
// keeps its own class, and a conversion keeps its rejection. The test reads
// each named type of the package of X that holds the object, because the
// merge group of 1 package needs 1 answer.
func (e *emitter) foreignRep(obj *types.TypeName) *types.TypeName {
	st := structObject(obj)
	if st == nil || st.NumFields() == 0 || obj.Pkg() == nil {
		return nil
	}
	home := st.Field(0).Pkg()
	if home == nil || home == obj.Pkg() || home.Path() == dartObjectPath {
		return nil
	}
	owners := e.structNames(home)[st]
	if len(owners) == 0 {
		return nil
	}
	for _, member := range e.structNames(obj.Pkg())[st] {
		if valueMethod(member) {
			return nil
		}
	}
	return owners[0]
}

// valueMethod reports whether a named type declares a method with a value
// receiver. The Dart class of the type holds the body of such a method.
func valueMethod(obj *types.TypeName) bool {
	named, _ := types.Unalias(obj.Type()).(*types.Named)
	if named == nil {
		return false
	}
	for i := range named.NumMethods() {
		if !pointerMethod(named.Method(i)) {
			return true
		}
	}
	return false
}

// structOwner gives the named type that owns the Dart class of a named struct
// type. It gives nil for a type that owns its own class.
func (e *emitter) structOwner(obj *types.TypeName) *types.Named {
	if inst := e.instanceRep(obj); inst != nil {
		return inst
	}
	if isLocalType(obj) {
		if owners := e.structNames(obj.Pkg())[structObject(obj)]; len(owners) > 0 {
			return e.structRef(owners[0].Type().(*types.Named))
		}
	}
	rep := e.foreignRep(obj)
	if rep == nil {
		return nil
	}
	named, _ := types.Unalias(rep.Type()).(*types.Named)
	return named
}

// packageStruct gives the struct object of a named struct type of a scope.
func packageStruct(scope *types.Scope, n string) (*types.Struct, *types.TypeName) {
	tn, _ := scope.Lookup(n).(*types.TypeName)
	if tn == nil || tn.IsAlias() {
		return nil, nil
	}
	named, _ := types.Unalias(tn.Type()).(*types.Named)
	if named == nil || named.TypeParams().Len() > 0 {
		return nil, nil
	}
	st, _ := named.Underlying().(*types.Struct)
	if st == nil {
		return nil, nil
	}
	return st, tn
}

// mergeRep gives the type name that owns the Dart class of a group. The scope
// gives each name in order, thus the first member is stable.
func (e *emitter) mergeRep(obj *types.TypeName) *types.TypeName {
	group := e.mergeGroup(obj)
	if len(group) == 0 {
		return obj
	}
	return group[0]
}

// memberName gives the name of a method inside its Dart class. A merged class
// holds the methods of each member of the group, thus 2 members that declare 1
// method name need 2 names.
func (e *emitter) memberName(fn *types.Func) string {
	base := name.Package(fn.Name())
	if !fn.Exported() {
		base = "$method$" + name.Public(fn.Name())
	}
	sig, _ := fn.Type().(*types.Signature)
	if sig == nil || sig.Recv() == nil {
		return base
	}
	obj := receiverTypeName(sig.Recv().Type())
	if obj == nil {
		return base
	}
	for _, other := range e.mergeGroup(obj) {
		if other != obj && declaresMethod(other, fn.Name()) {
			return base + "$" + name.Type(obj.Name())
		}
	}
	return base
}

// declaresMethod reports whether a named type declares a method of the name.
func declaresMethod(obj *types.TypeName, method string) bool {
	named, _ := types.Unalias(obj.Type()).(*types.Named)
	if named == nil {
		return false
	}
	for i := range named.NumMethods() {
		if named.Method(i).Name() == method {
			return true
		}
	}
	return false
}

// qualify adds the import prefix when the object belongs to another package.
func (e *emitter) qualify(obj types.Object, dartName string) string {
	if obj.Pkg() == nil || obj.Pkg() == e.pkg {
		return dartName
	}
	path := obj.Pkg().Path()
	prefix, ok := e.prefixes[path]
	if !ok {
		// A package that the Go source does not import can still reach the
		// output. A composite literal of a struct of another package writes the
		// zero value of each field, and a field type belongs to a third
		// package.
		prefix = e.addImport(path)
	}
	if prefix == "" {
		e.fail("the package %s has no import prefix", path)
		return dartName
	}
	return prefix + "." + dartName
}

// addImport gives a prefix to a package that the import list does not hold, and
// it records the import. The library file writes it.
func (e *emitter) addImport(path string) string {
	uri := e.cfg.URIs[path]
	if uri == "" {
		return ""
	}
	base := path
	if k := strings.LastIndex(base, "/"); k >= 0 {
		base = base[k+1:]
	}
	used := map[string]bool{}
	for _, p := range e.prefixes {
		used[p] = true
	}
	prefix := "$" + sanitize(base)
	for i := 1; used[prefix]; i++ {
		prefix = fmt.Sprintf("$%s$%d", sanitize(base), i)
	}
	e.prefixes[path] = prefix
	e.extra = append(e.extra, Import{Path: path, URI: uri})
	return prefix
}

// recvTypeParams gives the type parameters of a named receiver type. The top
// level function of a method of such a type names them, because the body reads
// the dictionary of each one.
func recvTypeParams(obj *types.TypeName) []*types.TypeParam {
	named, ok := types.Unalias(obj.Type()).(*types.Named)
	if !ok {
		return nil
	}
	return typeParamsOf(named.TypeParams())
}

// namedOf gives the named type of a receiver. It reads through a pointer.
func namedOf(t types.Type) *types.Named {
	u := types.Unalias(t)
	if p, ok := u.(*types.Pointer); ok {
		u = types.Unalias(p.Elem())
	}
	named, _ := u.(*types.Named)
	return named
}

// recvTypeArgs gives the dictionary and the Dart type of each type argument of
// the receiver type of a call to a method. A call inside the body of the type
// itself reads the type parameters, because such a receiver holds no argument.
func (e *emitter) recvTypeArgs(t types.Type) (dicts, args []string) {
	named := namedOf(t)
	if named == nil {
		return nil, nil
	}
	if list := named.TypeArgs(); list != nil && list.Len() > 0 {
		for i := 0; i < list.Len(); i++ {
			dicts = append(dicts, e.dictOf(list.At(i)))
			args = append(args, e.dartType(list.At(i)))
		}
		return dicts, args
	}
	for _, tp := range typeParamsOf(named.TypeParams()) {
		dicts = append(dicts, e.dictOf(tp))
		args = append(args, e.dartType(tp))
	}
	return dicts, args
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

func isComplexType(t types.Type) bool {
	if set, ok := typeset.Describe(t); ok && set.Complex {
		return true
	}
	b := basicOf(t)
	return b != nil && b.Info()&types.IsComplex != 0
}

func isFloat(t types.Type) bool {
	if set, ok := typeset.Describe(t); ok && set.Float {
		return true
	}
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

// cyclic reports whether the named type reaches itself through its underlying
// type. `type Number *Number` is such a type, and it has no finite Dart form.
// A struct breaks a cycle, because its class name stands in for its shape.
func (e *emitter) cyclic(obj *types.TypeName) bool {
	if e.typeCycle == nil {
		e.typeCycle = map[*types.TypeName]bool{}
		e.typeAcyclic = map[*types.TypeName]bool{}
	}
	if e.typeCycle[obj] {
		return true
	}
	if e.typeAcyclic[obj] {
		return false
	}
	seen := map[*types.TypeName]bool{}
	var walk func(t types.Type) bool
	walk = func(t types.Type) bool {
		switch u := types.Unalias(t).(type) {
		case *types.Named:
			if u.Obj() == obj {
				return true
			}
			if seen[u.Obj()] {
				return false
			}
			seen[u.Obj()] = true
			if _, ok := u.Underlying().(*types.Struct); ok {
				return false
			}
			return walk(u.Underlying())
		case *types.Pointer:
			return walk(u.Elem())
		case *types.Slice:
			return walk(u.Elem())
		case *types.Array:
			return walk(u.Elem())
		case *types.Chan:
			return walk(u.Elem())
		case *types.Map:
			return walk(u.Key()) || walk(u.Elem())
		case *types.Signature:
			// A function type that names itself, for example
			// `type stateFn func(*lexer) stateFn`, holds no finite Dart form
			// either. The depth guard of dartType wrote 30 levels of it, and
			// the Dart parser rejects that text.
			for i := 0; i < u.Params().Len(); i++ {
				if walk(u.Params().At(i).Type()) {
					return true
				}
			}
			for i := 0; i < u.Results().Len(); i++ {
				if walk(u.Results().At(i).Type()) {
					return true
				}
			}
			return false
		}
		return false
	}
	out := false
	if _, ok := obj.Type().Underlying().(*types.Struct); !ok {
		out = walk(obj.Type().Underlying())
	}
	if out {
		e.typeCycle[obj] = true
	} else {
		e.typeAcyclic[obj] = true
	}
	return out
}

// mentionsParam reports whether a type reaches the type parameter tp. A
// constraint of tp that holds such a type names itself, thus its Dart form
// never ends.
func mentionsParam(t types.Type, tp *types.TypeParam, seen map[*types.TypeName]bool) bool {
	switch u := types.Unalias(t).(type) {
	case *types.TypeParam:
		return u == tp
	case *types.Named:
		if seen[u.Obj()] {
			return false
		}
		seen[u.Obj()] = true
		for i := 0; i < u.TypeArgs().Len(); i++ {
			if mentionsParam(u.TypeArgs().At(i), tp, seen) {
				return true
			}
		}
		if _, ok := u.Underlying().(*types.Struct); ok {
			return false
		}
		return mentionsParam(u.Underlying(), tp, seen)
	case *types.Pointer:
		return mentionsParam(u.Elem(), tp, seen)
	case *types.Slice:
		return mentionsParam(u.Elem(), tp, seen)
	case *types.Array:
		return mentionsParam(u.Elem(), tp, seen)
	case *types.Chan:
		return mentionsParam(u.Elem(), tp, seen)
	case *types.Map:
		return mentionsParam(u.Key(), tp, seen) || mentionsParam(u.Elem(), tp, seen)
	case *types.Signature:
		for i := 0; i < u.Params().Len(); i++ {
			if mentionsParam(u.Params().At(i).Type(), tp, seen) {
				return true
			}
		}
		for i := 0; i < u.Results().Len(); i++ {
			if mentionsParam(u.Results().At(i).Type(), tp, seen) {
				return true
			}
		}
	}
	return false
}

// mentionsCyclic reports whether a type reaches a named type that names
// itself. Such a named type takes Object?, thus each type that holds it takes
// Object? as well.
func (e *emitter) mentionsCyclic(t types.Type, seen map[*types.TypeName]bool) bool {
	switch u := types.Unalias(t).(type) {
	case *types.Named:
		if seen[u.Obj()] {
			return false
		}
		seen[u.Obj()] = true
		if e.cyclic(u.Obj()) {
			return true
		}
		if _, ok := u.Underlying().(*types.Struct); ok {
			return false
		}
		return e.mentionsCyclic(u.Underlying(), seen)
	case *types.Pointer:
		return e.mentionsCyclic(u.Elem(), seen)
	case *types.Slice:
		return e.mentionsCyclic(u.Elem(), seen)
	case *types.Array:
		return e.mentionsCyclic(u.Elem(), seen)
	case *types.Chan:
		return e.mentionsCyclic(u.Elem(), seen)
	case *types.Map:
		return e.mentionsCyclic(u.Key(), seen) || e.mentionsCyclic(u.Elem(), seen)
	case *types.Signature:
		for i := 0; i < u.Params().Len(); i++ {
			if e.mentionsCyclic(u.Params().At(i).Type(), seen) {
				return true
			}
		}
		for i := 0; i < u.Results().Len(); i++ {
			if e.mentionsCyclic(u.Results().At(i).Type(), seen) {
				return true
			}
		}
	}
	return false
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
	case types.Complex64:
		return "$c64"
	}
	return ""
}

// holdsTypeParam reports whether a type reads a type parameter. A value of such
// a type takes no variable at package level.
func holdsTypeParam(t types.Type) bool {
	return typeset.HoldsParam(t)
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
	case types.Complex64:
		return "$dictComplex64"
	case types.Complex128, types.UntypedComplex:
		return "$dictComplex128"
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

// instanceRep gives the instance of a generic type whose struct object is the
// underlying struct of obj.
//
// `type X G[A]` gives X the struct object of G[A]. A pointer conversion between
// the 2 types names 1 value, and a write through the result reaches that value.
// The 2 types thus need 1 Dart form. X takes the form of the instance, and it
// declares no class of its own. Each method of X becomes a top level function,
// exactly as a method of a named type over a basic type does.
//
// A generic type stays out, and so does a type whose package names the instance
// nowhere. X keeps its own class in that case, and a conversion fails with the
// report of the 2 classes.
func (e *emitter) instanceRep(obj *types.TypeName) *types.Named {
	named, _ := types.Unalias(obj.Type()).(*types.Named)
	if named == nil || named.TypeParams().Len() > 0 || named.TypeArgs().Len() > 0 {
		return nil
	}
	st, _ := named.Underlying().(*types.Struct)
	if st == nil || obj.Pkg() == nil {
		return nil
	}
	inst := e.packageInstances(obj.Pkg())[st]
	// A recursive type argument needs its class name to end expansion.
	if inst != nil && instanceArgumentContains(inst, obj, map[types.Type]bool{}) {
		return nil
	}
	return inst
}

func instanceArgumentContains(t types.Type, obj *types.TypeName, seen map[types.Type]bool) bool {
	t = types.Unalias(t)
	if seen[t] {
		return false
	}
	seen[t] = true
	switch t := t.(type) {
	case *types.Named:
		if t.Obj() == obj {
			return true
		}
		for i := range t.TypeArgs().Len() {
			if instanceArgumentContains(t.TypeArgs().At(i), obj, seen) {
				return true
			}
		}
	case *types.Pointer:
		return instanceArgumentContains(t.Elem(), obj, seen)
	case *types.Slice:
		return instanceArgumentContains(t.Elem(), obj, seen)
	case *types.Array:
		return instanceArgumentContains(t.Elem(), obj, seen)
	case *types.Chan:
		return instanceArgumentContains(t.Elem(), obj, seen)
	case *types.Map:
		return instanceArgumentContains(t.Key(), obj, seen) || instanceArgumentContains(t.Elem(), obj, seen)
	}
	return false
}

// packageInstances gives the instance of a generic type that owns each struct
// object of a package. The scope of a package holds each exported name and each
// name of the package under compilation, thus the walk reaches the instance
// that a declaration of the package names.
func (e *emitter) packageInstances(pkg *types.Package) map[*types.Struct]*types.Named {
	if m, ok := e.instByStruct[pkg]; ok {
		return m
	}
	m := map[*types.Struct]*types.Named{}
	scope := pkg.Scope()
	if genericStructs(scope) {
		seen := map[types.Type]bool{}
		for _, n := range scope.Names() {
			obj := scope.Lookup(n)
			if obj != nil {
				collectInstances(obj.Type(), m, seen)
			}
		}
	}
	e.instByStruct[pkg] = m
	return m
}

// genericStructs reports whether a scope declares a generic type over a struct.
// A package with no such type owns no instance, thus the walk is not needed.
func genericStructs(scope *types.Scope) bool {
	for _, n := range scope.Names() {
		tn, _ := scope.Lookup(n).(*types.TypeName)
		if tn == nil || tn.IsAlias() {
			continue
		}
		named, _ := types.Unalias(tn.Type()).(*types.Named)
		if named == nil || named.TypeParams().Len() == 0 {
			continue
		}
		if _, ok := named.Underlying().(*types.Struct); ok {
			return true
		}
	}
	return false
}

// collectInstances records each instance of a generic struct type that a type
// names, keyed by the struct object of the instance.
func collectInstances(t types.Type, out map[*types.Struct]*types.Named,
	seen map[types.Type]bool) {
	if t == nil || seen[t] {
		return
	}
	seen[t] = true
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		if t.TypeArgs().Len() > 0 {
			if st, ok := t.Underlying().(*types.Struct); ok {
				if _, held := out[st]; !held {
					out[st] = t
				}
			}
			for i := range t.TypeArgs().Len() {
				collectInstances(t.TypeArgs().At(i), out, seen)
			}
		}
		collectInstances(t.Underlying(), out, seen)
		for i := range t.NumMethods() {
			collectInstances(t.Method(i).Type(), out, seen)
		}
	case *types.Struct:
		for i := range t.NumFields() {
			collectInstances(t.Field(i).Type(), out, seen)
		}
	case *types.Pointer:
		collectInstances(t.Elem(), out, seen)
	case *types.Slice:
		collectInstances(t.Elem(), out, seen)
	case *types.Array:
		collectInstances(t.Elem(), out, seen)
	case *types.Chan:
		collectInstances(t.Elem(), out, seen)
	case *types.Map:
		collectInstances(t.Key(), out, seen)
		collectInstances(t.Elem(), out, seen)
	case *types.Signature:
		collectInstances(t.Params(), out, seen)
		collectInstances(t.Results(), out, seen)
	case *types.Tuple:
		for i := range t.Len() {
			collectInstances(t.At(i).Type(), out, seen)
		}
	case *types.Interface:
		for i := range t.NumMethods() {
			collectInstances(t.Method(i).Type(), out, seen)
		}
	}
}

// structRef gives the named type that owns the Dart form of a named struct
// type. A defined type over an instance of a generic type reads the instance,
// and a defined type over a struct of another package reads that package.
func (e *emitter) structRef(named *types.Named) *types.Named {
	if rep := e.structOwner(named.Obj()); rep != nil {
		return rep
	}
	return named
}

func zeroSized(t types.Type) (bool, bool) {
	t = types.Unalias(t)
	if _, ok := t.(*types.TypeParam); ok {
		return false, false
	}
	switch u := t.Underlying().(type) {
	case *types.Array:
		if u.Len() == 0 {
			return true, true
		}
		return zeroSized(u.Elem())
	case *types.Struct:
		known := true
		for i := 0; i < u.NumFields(); i++ {
			zero, held := zeroSized(u.Field(i).Type())
			if held && !zero {
				return false, true
			}
			known = known && held
		}
		return known, known
	}
	return false, true
}

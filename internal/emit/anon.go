package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

// Anonymous structs share storage across packages and generic instantiations.
// Field dictionaries retain value semantics without a package-specific class.
func (e *emitter) anonClass(st *types.Struct) string {
	storage := storageShape(st)
	sum := sha256.Sum256([]byte(anonText(storage)))
	key := "$Anon" + hex.EncodeToString(sum[:])
	e.anon[key] = storage
	return key
}
func storageShape(st *types.Struct) *types.Struct {
	fields := make([]*types.Var, st.NumFields())
	for i := range fields {
		field := st.Field(i)
		param := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, fmt.Sprintf("$F%d", i), nil), types.NewInterfaceType(nil, nil).Complete())
		fields[i] = types.NewField(token.NoPos, field.Pkg(), field.Name(), param, field.Embedded())
	}
	return types.NewStruct(fields, nil)
}

func (e *emitter) anonType(st *types.Struct) string {
	class := e.anonClass(st)
	if st.NumFields() == 0 {
		return class
	}
	args := make([]string, st.NumFields())
	for i := range args {
		args[i] = e.fieldStorageType(st.Field(i).Type())
	}
	return class + "<" + strings.Join(args, ", ") + ">"
}

// typeParamArgs writes the type arguments of a class that takes 1 Dart type
// parameter for each Go parameter of the shape.
func (e *emitter) typeParamArgs(params []*types.TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, tp := range params {
		args[i] = e.dartType(tp)
	}
	return "<" + strings.Join(args, ", ") + ">"
}

func (e *emitter) anonDicts(st *types.Struct) []string {
	out := make([]string, st.NumFields())
	for i := range out {
		out[i] = e.dictOf(st.Field(i).Type())
	}
	return out
}

func walkTuple(t *types.Tuple, walk func(types.Type)) {
	for i := 0; i < t.Len(); i++ {
		walk(t.At(i).Type())
	}
}

// Tags belong to Go descriptors, not Dart storage. A conversion can share a
// pointer or slice whose unnamed element differs only in its tags.
func untaggedShape(t types.Type) types.Type {
	switch u := types.Unalias(t).(type) {
	case *types.Struct:
		fields := make([]*types.Var, u.NumFields())
		for i := range fields {
			f := u.Field(i)
			fields[i] = types.NewField(f.Pos(), f.Pkg(), f.Name(), untaggedShape(f.Type()), f.Embedded())
		}
		return types.NewStruct(fields, nil)
	case *types.Pointer:
		return types.NewPointer(untaggedShape(u.Elem()))
	case *types.Slice:
		return types.NewSlice(untaggedShape(u.Elem()))
	case *types.Array:
		return types.NewArray(untaggedShape(u.Elem()), u.Len())
	case *types.Map:
		return types.NewMap(untaggedShape(u.Key()), untaggedShape(u.Elem()))
	case *types.Chan:
		return types.NewChan(u.Dir(), untaggedShape(u.Elem()))
	case *types.Signature:
		tuple := func(in *types.Tuple) *types.Tuple {
			vars := make([]*types.Var, in.Len())
			for i := range vars {
				v := in.At(i)
				vars[i] = types.NewVar(v.Pos(), v.Pkg(), v.Name(), untaggedShape(v.Type()))
			}
			return types.NewTuple(vars...)
		}
		return types.NewSignatureType(nil, nil, nil, tuple(u.Params()), tuple(u.Results()), u.Variadic())
	}
	return t
}

// anonText writes the Go text of a type. A named type takes its import path,
// thus 2 packages of the same build give the same text.
func anonText(t types.Type) string {
	var b strings.Builder
	writeType(&b, t, qualifyPath, map[types.Type]bool{})
	return b.String()
}

// typeText writes the text of a type with the package name of each named type,
// exactly as types.TypeString with no qualifier writes it. The name of a
// descriptor reads it.
func typeText(t types.Type) string {
	var b strings.Builder
	writeType(&b, t, nil, map[types.Type]bool{})
	return b.String()
}

// writeType writes the text of a type that names the same text for 2 types
// that Go holds identical, and another text for each other pair.
//
// types.TypeString gives no such text for 2 shapes: it writes an embedded
// field as its type alone, thus `struct{ int }` and `struct{ Int }` of an
// alias Int give 1 text, and it writes an alias by its name, thus 2 local
// aliases of 1 name in 2 blocks give 1 text. The walk below writes each field
// name and it reads each alias.
func writeType(b *strings.Builder, t types.Type, qual types.Qualifier, seen map[types.Type]bool) {
	if t == nil {
		b.WriteString("<nil>")
		return
	}
	if seen[t] {
		b.WriteString("<cycle>")
		return
	}
	switch u := types.Unalias(t).(type) {
	case *types.Basic:
		b.WriteString(basicText(u))
	case *types.Named:
		b.WriteString(types.TypeString(u, qual))
	case *types.TypeParam:
		b.WriteString(types.TypeString(u, qual))
	case *types.Pointer:
		seen[t] = true
		b.WriteString("*")
		writeType(b, u.Elem(), qual, seen)
		delete(seen, t)
	case *types.Slice:
		b.WriteString("[]")
		writeType(b, u.Elem(), qual, seen)
	case *types.Array:
		fmt.Fprintf(b, "[%d]", u.Len())
		writeType(b, u.Elem(), qual, seen)
	case *types.Map:
		b.WriteString("map[")
		writeType(b, u.Key(), qual, seen)
		b.WriteString("]")
		writeType(b, u.Elem(), qual, seen)
	case *types.Chan:
		switch u.Dir() {
		case types.SendOnly:
			b.WriteString("chan<- ")
		case types.RecvOnly:
			b.WriteString("<-chan ")
		default:
			b.WriteString("chan ")
		}
		elem, ok := types.Unalias(u.Elem()).(*types.Chan)
		parenthesize := u.Dir() == types.SendRecv && ok && elem.Dir() == types.RecvOnly
		if parenthesize {
			b.WriteString("(")
		}
		writeType(b, u.Elem(), qual, seen)
		if parenthesize {
			b.WriteString(")")
		}
	case *types.Struct:
		seen[t] = true
		b.WriteString("struct{")
		for i := 0; i < u.NumFields(); i++ {
			f := u.Field(i)
			if i > 0 {
				b.WriteString("; ")
			}
			if f.Embedded() {
				b.WriteString("~")
			}
			b.WriteString(f.Name())
			if !f.Exported() && f.Pkg() != nil {
				b.WriteString("@" + f.Pkg().Path())
			}
			b.WriteString(" ")
			writeType(b, f.Type(), qual, seen)
			if tag := u.Tag(i); tag != "" {
				fmt.Fprintf(b, " %q", tag)
			}
		}
		b.WriteString("}")
		delete(seen, t)
	case *types.Signature:
		b.WriteString("func(")
		writeTuple(b, u.Params(), u.Variadic(), qual, seen)
		b.WriteString(")")
		switch u.Results().Len() {
		case 0:
		case 1:
			b.WriteString(" ")
			writeType(b, u.Results().At(0).Type(), qual, seen)
		default:
			b.WriteString(" (")
			writeTuple(b, u.Results(), false, qual, seen)
			b.WriteString(")")
		}
	case *types.Interface:
		seen[t] = true
		b.WriteString("interface{")
		for i := 0; i < u.NumMethods(); i++ {
			m := u.Method(i)
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(m.Name())
			if !m.Exported() && m.Pkg() != nil {
				b.WriteString("@" + m.Pkg().Path())
			}
			writeType(b, m.Type(), qual, seen)
		}
		b.WriteString("}")
		delete(seen, t)
	default:
		b.WriteString(types.TypeString(t, qual))
	}
}

// writeTuple writes the type of each element of a tuple.
func writeTuple(b *strings.Builder, tuple *types.Tuple, variadic bool, qual types.Qualifier, seen map[types.Type]bool) {
	for i := 0; i < tuple.Len(); i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		elem := tuple.At(i).Type()
		if variadic && i == tuple.Len()-1 {
			b.WriteString("...")
			elem = types.Unalias(elem).(*types.Slice).Elem()
		}
		writeType(b, elem, qual, seen)
	}
}

// qualifyPath names a package by its import path.
func qualifyPath(p *types.Package) string { return p.Path() }

// nilChan gives the name of the shared nil channel of an element type.
//
// The zero value of a channel type reaches each variable of a frame, and such
// a channel holds no state that a program can change. Thus 1 value serves each
// variable of the type.
func (e *emitter) nilChan(elem types.Type) string {
	text := e.dartType(elem)
	sum := sha256.Sum256([]byte(text))
	key := e.nilPrefix + "$nilChan" + hex.EncodeToString(sum[:])
	if _, ok := e.nilChans[key]; !ok {
		e.nilChans[key] = elem
	}
	return key
}

// instDescVar names the variable that holds the description of an instance
// of a generic type. The description of an instance can reach itself through
// a method signature or a field, thus the text stands in 1 variable, and each
// use names it.
func (e *emitter) instDescVar(named *types.Named, pointer bool) string {
	t := types.Type(named)
	if pointer {
		t = types.NewPointer(named)
	}
	// A description that reads a type parameter reads the dictionary of the
	// enclosing function, thus it cannot stand at package level. The name of a
	// local type of a generic function reads the same dictionary.
	if holdsTypeParam(t) || hasOuterType(t) {
		return e.instanceDesc(named, pointer)
	}
	name := e.typeName(t)
	sum := sha256.Sum256([]byte(name))
	key := e.nilPrefix + "$desc" + hex.EncodeToString(sum[:4])
	held, ok := e.instDescs[key]
	if !ok {
		e.instDescs[key] = t
		return key
	}
	// 1 variable serves 1 type. The text stands at the point of the use when 2
	// names meet at 1 key.
	if e.typeName(held) != name {
		return e.instanceDesc(named, pointer)
	}
	return key
}

// writeInstDescs writes the variable of each description that the package holds
// in a variable: the description of an instance of a generic type, and the
// description of a type with no name of its own. A description can name another
// one, thus the loop reads the map again, exactly as writeNilChans does.
func (e *emitter) writeInstDescs() {
	if e.instDescsDone == nil {
		e.instDescsDone = map[string]bool{}
	}
	written := e.instDescsDone
	for {
		names := make([]string, 0, len(e.instDescs))
		for key := range e.instDescs {
			if !written[key] {
				names = append(names, key)
			}
		}
		if len(names) == 0 {
			return
		}
		sort.Strings(names)
		for _, key := range names {
			written[key] = true
			if text, ok := e.descTexts[key]; ok {
				e.line("final TypeDesc %s = %s;", key, text)
				continue
			}
			t := e.instDescs[key]
			named, _ := types.Unalias(t).(*types.Named)
			pointer := false
			if p, ok := types.Unalias(t).(*types.Pointer); ok {
				named, _ = types.Unalias(p.Elem()).(*types.Named)
				pointer = true
			}
			e.line("final TypeDesc %s = %s;", key, e.instanceDesc(named, pointer))
		}
		e.line("")
	}
}

// writeNilChans writes the nil channel of each element type that the package
// uses. The order comes from the name, thus equal input gives an equal file.
// A second call writes only the new keys, because the init routine of the
// package variables registers its channels after the first call.
func (e *emitter) writeNilChans() {
	if e.nilChansDone == nil {
		e.nilChansDone = map[string]bool{}
	}
	written := e.nilChansDone
	for {
		names := make([]string, 0, len(e.nilChans))
		for key := range e.nilChans {
			if !written[key] {
				names = append(names, key)
			}
		}
		if len(names) == 0 {
			return
		}
		sort.Strings(names)
		for _, key := range names {
			// The zero value of the element type can need another such
			// variable, thus the loop reads the map again.
			written[key] = true
			elem := e.nilChans[key]
			e.line("final Chan<%s> %s = Chan<%s>.nil(%s);",
				e.dartType(elem), key, e.dartType(elem), e.zeroFunc(elem))
		}
		e.line("")
	}
}

// anonDir holds the file of each unnamed struct type. The path stands outside
// the directory of a package, because 2 packages share 1 file.
const anonDir = "lib/gart_anon/"

// anonFileName gives the file name of 1 shape. A Dart URI is a string literal,
// and a dollar sign in such a literal starts an interpolation, thus the name of
// the class does not fit.
func anonFileName(class string) string {
	return strings.ToLower(strings.TrimPrefix(class, "$")) + ".dart"
}

// anonPath gives the path of the file of 1 shape, relative to the output root.
func anonPath(class string) string {
	return anonDir + anonFileName(class)
}

// anonURI gives the Dart URI of the file of 1 shape.
func (e *emitter) anonURI(class string) string {
	return "package:" + e.cfg.DartPackage + "/gart_anon/" + anonFileName(class)
}

// anonURIs lists the URI of each shape that the package reads, in the order of
// the name.
func (e *emitter) anonURIs() []string {
	out := make([]string, 0, len(e.anon))
	for key := range e.anon {
		out = append(out, e.anonURI(key))
	}
	sort.Strings(out)
	return out
}

func (e *emitter) sharedFiles() map[string]string {
	out := map[string]string{}
	for class, st := range e.anon {
		out[anonPath(class)] = anonFile(class, st)
	}
	return out
}

// Storage parameters remove dependencies on caller packages and constraints.
func anonFile(class string, st *types.Struct) string {
	inner := &emitter{}
	inner.buf.WriteString(header)
	inner.line("")
	inner.line("import 'package:goruntime/goruntime.dart';")
	inner.line("")
	params := make([]*types.TypeParam, st.NumFields())
	for i := range params {
		params[i] = st.Field(i).Type().(*types.TypeParam)
	}
	inner.structClass(class, params, st, true, nil)
	inner.line("TypeDesc %s$storage(List<TypeDesc> types) {", class)
	for i, tp := range params {
		inner.line("return $withStorageType<TypeDesc>(types[%d], <%s>(Dict<%s> $d%s) {", i, inner.dartType(tp), inner.dartType(tp), inner.typeParamName(tp))
	}
	full := class + inner.typeArgsOfParams(params)
	dicts := make([]string, len(params))
	for i, tp := range params {
		dicts[i] = "$d" + inner.typeParamName(tp)
	}
	inner.line("return TypeDesc('', 'struct', const <String, Function>{},")
	inner.line("zero: () => %s.$zero(%s),", full, strings.Join(dicts, ", "))
	inner.line("assign: (a, b) => (a as %s).$assign(b as %s),", full, full)
	inner.line("withType: <R>(R Function<T>() f) => f<%s>(),", full)
	inner.line("view: (from, value) => value is %s ? value : %s.$viewFrom(%s),", full, full, strings.Join(append(dicts, "from", "value"), ", "))
	inner.line("fields: <FieldDesc>[")
	for i := 0; i < st.NumFields(); i++ {
		field := structFieldName(st.Field(i), i)
		inner.line("FieldDesc('', '', true, () => types[%d], (r) => (r as %s).%s, (r, v) => (r as %s).%s = v as %s),", i, full, field, full, field, inner.dartType(st.Field(i).Type()))
	}
	inner.line("]);")
	for range params {
		inner.line("});")
	}
	inner.line("}")
	return inner.buf.String()
}

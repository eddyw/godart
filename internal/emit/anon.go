package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"sort"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
)

// anonClass gives the name of the Dart class of an unnamed struct type, and it
// records the type for the file that holds the class.
//
// Go gives an unnamed struct type a structural identity, and a Dart class holds
// a nominal identity. The name comes from the shape of the struct, thus 2 equal
// Go types give 1 Dart class. The class stands in a file of its own, and each
// package that holds the shape writes that file with equal bytes.
//
// A field type of a package that no library holds keeps the class inside the
// package. Package main is such a package, and no other package names a type
// of it.
func (e *emitter) anonClass(st *types.Struct) string {
	if typeset.HoldsParam(st) {
		return e.openClass(st)
	}
	key := e.anonName(st)
	target := e.anon
	if !e.anonShared(st) {
		target = e.localAnon
	}
	if _, ok := target[key]; !ok {
		target[key] = st
	}
	return key
}

// anonType gives the Dart type of an unnamed struct value: the class name and,
// for a shape with a parameter, the type arguments. A ground shape whose open
// twin exists reads the class of the twin, because 1 Go shape must give 1 Dart
// class, or a type test and an equality through an interface give a wrong
// result.
func (e *emitter) anonType(st *types.Struct) string {
	if typeset.HoldsParam(st) {
		return e.openClass(st) + e.typeParamArgs(e.anonParams[e.anonName(st)])
	}
	if twin := e.anonTwin(st); twin != nil {
		return twin.class + "<" + strings.Join(twin.args, ", ") + ">"
	}
	return e.anonClass(st)
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

// structOfAnon gives the underlying struct of an unnamed struct type and of a
// local named type whose underlying type is a struct. Both take the class of
// a shape.
func structOfAnon(t types.Type) (*types.Struct, bool) {
	switch t := types.Unalias(t).(type) {
	case *types.Struct:
		return t, true
	case *types.Named:
		if isLocalType(t.Obj()) {
			if st, ok := t.Underlying().(*types.Struct); ok {
				return st, true
			}
		}
	}
	return nil, false
}

// anonDicts gives the dictionary argument of each parameter of the shape.
func (e *emitter) anonDicts(st *types.Struct) []string {
	if params, ok := e.anonParams[e.anonName(st)]; ok && typeset.HoldsParam(st) {
		out := make([]string, len(params))
		for i, tp := range params {
			out[i] = e.dictOf(tp)
		}
		return out
	}
	if twin := e.anonTwin(st); twin != nil {
		return twin.dicts
	}
	return nil
}

// openClass names the class of a shape that reads a type parameter, and it
// records the shape with its parameters. The class takes 1 Dart type parameter
// for each Go parameter of the shape, thus the ground shape and the open shape
// share the class.
func (e *emitter) openClass(st *types.Struct) string {
	key := e.anonName(st)
	if _, ok := e.anonParams[key]; !ok {
		e.anonParams[key] = shapeParams(st)
		e.localAnon[key] = st
	}
	return key
}

// shapeParams lists each type parameter of the shape, in the order of its
// first use in a field type.
func shapeParams(st *types.Struct) []*types.TypeParam {
	var out []*types.TypeParam
	seen := map[*types.TypeParam]bool{}
	var walk func(t types.Type)
	walk = func(t types.Type) {
		switch t := types.Unalias(t).(type) {
		case *types.TypeParam:
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		case *types.Pointer:
			walk(t.Elem())
		case *types.Slice:
			walk(t.Elem())
		case *types.Array:
			walk(t.Elem())
		case *types.Chan:
			walk(t.Elem())
		case *types.Map:
			walk(t.Key())
			walk(t.Elem())
		case *types.Signature:
			walkTuple(t.Params(), walk)
			walkTuple(t.Results(), walk)
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				walk(t.Field(i).Type())
			}
		case *types.Named:
			for i := 0; i < t.TypeArgs().Len(); i++ {
				walk(t.TypeArgs().At(i))
			}
		}
	}
	walk(st)
	return out
}

func walkTuple(t *types.Tuple, walk func(types.Type)) {
	for i := 0; i < t.Len(); i++ {
		walk(t.At(i).Type())
	}
}

// anonTwin gives the class and the arguments of the open twin of a ground
// shape. A nil result names no twin.
func (e *emitter) anonTwin(st *types.Struct) *anonTwin {
	key := e.anonName(st)
	if t, ok := e.groundTwins[key]; ok {
		if t == nil {
			return nil
		}
		return t
	}
	t := e.matchTwin(st)
	e.groundTwins[key] = t
	return t
}

// matchTwin compares the ground shape with each open shape of the package.
func (e *emitter) matchTwin(st *types.Struct) *anonTwin {
	keys := make([]string, 0, len(e.anonParams))
	for key := range e.anonParams {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		open, ok := e.localAnon[key]
		if !ok {
			continue
		}
		subs := map[*types.TypeParam]types.Type{}
		if !matchShape(open, st, subs) {
			continue
		}
		e.openClass(open)
		params := e.anonParams[key]
		t := &anonTwin{class: key}
		for _, tp := range params {
			g := subs[tp]
			t.args = append(t.args, e.dartType(g))
			t.dicts = append(t.dicts, e.dictOf(g))
		}
		return t
	}
	return nil
}

// matchShape reports whether the ground type fills the open type. It records
// the type of each parameter in subs, and 2 uses of 1 parameter must agree.
func matchShape(open, ground types.Type, subs map[*types.TypeParam]types.Type) bool {
	open, ground = types.Unalias(open), types.Unalias(ground)
	if tp, ok := open.(*types.TypeParam); ok {
		if prev, ok := subs[tp]; ok {
			return types.Identical(prev, ground)
		}
		subs[tp] = ground
		return true
	}
	if !typeset.HoldsParam(open) {
		return types.Identical(open, ground)
	}
	switch open := open.(type) {
	case *types.Struct:
		u := types.Unalias(ground)
		if named, ok := u.(*types.Named); ok {
			u = named.Underlying()
		}
		g, ok := u.(*types.Struct)
		if !ok || g.NumFields() != open.NumFields() {
			return false
		}
		for i := 0; i < g.NumFields(); i++ {
			if g.Field(i).Name() != open.Field(i).Name() ||
				g.Tag(i) != open.Tag(i) ||
				g.Field(i).Embedded() != open.Field(i).Embedded() {
				return false
			}
			if !matchShape(open.Field(i).Type(), g.Field(i).Type(), subs) {
				return false
			}
		}
		return true
	case *types.Pointer:
		g, ok := ground.(*types.Pointer)
		return ok && matchShape(open.Elem(), g.Elem(), subs)
	case *types.Slice:
		g, ok := ground.(*types.Slice)
		return ok && matchShape(open.Elem(), g.Elem(), subs)
	case *types.Array:
		g, ok := ground.(*types.Array)
		return ok && open.Len() == g.Len() && matchShape(open.Elem(), g.Elem(), subs)
	case *types.Chan:
		g, ok := ground.(*types.Chan)
		return ok && open.Dir() == g.Dir() && matchShape(open.Elem(), g.Elem(), subs)
	case *types.Map:
		g, ok := ground.(*types.Map)
		return ok && matchShape(open.Key(), g.Key(), subs) &&
			matchShape(open.Elem(), g.Elem(), subs)
	case *types.Signature:
		g, ok := ground.(*types.Signature)
		return ok && open.Variadic() == g.Variadic() &&
			matchTuple(open.Params(), g.Params(), subs) &&
			matchTuple(open.Results(), g.Results(), subs)
	case *types.Named:
		g, ok := ground.(*types.Named)
		if !ok || open.Obj() != g.Obj() || open.TypeArgs().Len() != g.TypeArgs().Len() {
			return false
		}
		for i := 0; i < open.TypeArgs().Len(); i++ {
			if !matchShape(open.TypeArgs().At(i), g.TypeArgs().At(i), subs) {
				return false
			}
		}
		return true
	}
	return false
}

func matchTuple(open, ground *types.Tuple, subs map[*types.TypeParam]types.Type) bool {
	if open.Len() != ground.Len() {
		return false
	}
	for i := 0; i < open.Len(); i++ {
		if !matchShape(open.At(i).Type(), ground.At(i).Type(), subs) {
			return false
		}
	}
	return true
}

// anonName gives the Dart class name of an unnamed struct type.
//
// The name comes from the text of the type and from the library of each field
// type. A test variant of a package holds a library of its own, thus a shape
// that names a type of such a variant takes a class of its own. A shape that
// reads a type parameter writes 1 marker for each parameter, thus the name
// does not depend on the name of the parameter.
func (e *emitter) anonName(st *types.Struct) string {
	text := anonText(st)
	if typeset.HoldsParam(st) {
		var reads []string
		text = anonText(e.substParams(st, &reads))
	}
	for _, im := range e.anonImports(st) {
		text += "\n" + im.URI
	}
	sum := sha256.Sum256([]byte(text))
	return "$Anon" + hex.EncodeToString(sum[:4])
}

// anonShared reports whether the shape reaches a file of its own.
//
// A library must hold each field type of such a shape, thus the file can import
// them. Each field must also carry an exported name: Go gives an unexported
// field the identity of the package that declares it, and Dart holds a name
// with a leading underscore private to 1 library.
func (e *emitter) anonShared(st *types.Struct) bool {
	if anonPrivate(st, map[types.Type]bool{}) {
		return false
	}
	paths := map[string]bool{}
	for i := 0; i < st.NumFields(); i++ {
		typePackages(st.Field(i).Type(), paths, map[types.Type]bool{})
	}
	for path := range paths {
		if e.cfg.URIs[path] == "" {
			return false
		}
	}
	return true
}

// anonComparable reports whether the class of the shape holds the equality. A
// field that reads a type parameter compares through its dictionary, and Go
// compares such a shape when each ground argument is comparable.
func anonComparable(st *types.Struct) bool {
	for i := 0; i < st.NumFields(); i++ {
		t := st.Field(i).Type()
		if _, ok := types.Unalias(t).(*types.TypeParam); ok {
			continue
		}
		if !types.Comparable(t) {
			return false
		}
	}
	return true
}

// anonPrivate reports whether an unnamed struct type inside the type holds an
// unexported field. The walk stops at a named type of a package scope, because
// such a type holds a class of its own.
//
// A named struct type of a function body holds no class of its own. It reads
// the class of its shape, thus the walk reads the fields of that shape. A test
// file of encoding/json declares `type S1 struct{ x, X int }` inside a
// function, and it embeds S1 in an unnamed struct type.
func anonPrivate(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		if isLocalType(t.Obj()) {
			return anonPrivate(t.Underlying(), seen)
		}
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)
			if !f.Exported() || anonPrivate(f.Type(), seen) {
				return true
			}
		}
	case *types.Pointer:
		return anonPrivate(t.Elem(), seen)
	case *types.Slice:
		return anonPrivate(t.Elem(), seen)
	case *types.Array:
		return anonPrivate(t.Elem(), seen)
	case *types.Chan:
		return anonPrivate(t.Elem(), seen)
	case *types.Map:
		return anonPrivate(t.Key(), seen) || anonPrivate(t.Elem(), seen)
	case *types.Signature:
		return anonPrivate(t.Params(), seen) || anonPrivate(t.Results(), seen)
	case *types.Tuple:
		for i := 0; i < t.Len(); i++ {
			if anonPrivate(t.At(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

// anonText writes the Go text of a type. A named type takes its import path,
// thus 2 packages of the same build give the same text.
func anonText(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Path() })
}

// nilChan gives the name of the shared nil channel of an element type.
//
// The zero value of a channel type reaches each variable of a frame, and such
// a channel holds no state that a program can change. Thus 1 value serves each
// variable of the type.
func (e *emitter) nilChan(elem types.Type) string {
	text := e.dartType(elem)
	sum := sha256.Sum256([]byte(text))
	key := e.nilPrefix + "$nilChan" + hex.EncodeToString(sum[:4])
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
	if holdsTypeParam(t) || len(outerTypeParams(named)) > 0 {
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
			e.line("final Chan<%s> %s = Chan<%s>.nil(() => %s);",
				e.dartType(elem), key, e.dartType(elem), e.zero(elem))
		}
		e.line("")
	}
}

// anonTwin holds the Dart class and the arguments of the open twin of a ground
// shape.
type anonTwin struct {
	class string
	args  []string
	dicts []string
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

// writeAnonClasses writes the class of each unnamed struct type that stays
// inside the package. Such a class can name another one, thus the loop reads
// the map again.
func (e *emitter) writeAnonClasses() {
	if e.anonDone == nil {
		e.anonDone = map[string]bool{}
	}
	written := e.anonDone
	for {
		names := make([]string, 0, len(e.localAnon))
		for key := range e.localAnon {
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
			st := e.localAnon[key]
			e.line("/// %s", anonText(st))
			e.structClass(key, e.anonParams[key], st, anonComparable(st), nil)
		}
	}
}

// sharedFiles writes the file of each unnamed struct type that the package
// uses. A field of such a type can hold another one, thus the walk reads the
// map again until it finds no new shape.
func (e *emitter) sharedFiles() map[string]string {
	out := map[string]string{}
	done := map[string]bool{}
	todo := map[string]*types.Struct{}
	for key, st := range e.anon {
		todo[key] = st
	}
	for len(todo) > 0 {
		keys := make([]string, 0, len(todo))
		for key := range todo {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		next := map[string]*types.Struct{}
		for _, key := range keys {
			st := todo[key]
			done[key] = true
			text, more := e.anonFile(key, st)
			out[anonPath(key)] = text
			for k, s := range more {
				if !done[k] {
					next[k] = s
				}
			}
		}
		todo = next
	}
	return out
}

// anonFile writes the library of 1 shape. The content reads the shape only,
// thus each package that holds the shape writes equal bytes.
func (e *emitter) anonFile(class string, st *types.Struct) (string, map[string]*types.Struct) {
	inner := &emitter{
		merges:       e.merges,
		instByStruct: e.instByStruct,
		prefixes:     map[string]string{},
		initNames:    map[*ir.Func]string{},
		anon:         map[string]*types.Struct{},
		localAnon:    map[string]*types.Struct{},
		shadowed:     map[string]map[string]bool{},
		nilChans:     map[string]types.Type{},
		instDescs:    map[string]types.Type{},
		descTexts:    map[string]string{},
		cfg:          e.cfg,
		nilPrefix:    class,
		anonParams:   map[string][]*types.TypeParam{},
		groundTwins:  map[string]*anonTwin{},
		funcIds:      e.funcIds, funcByObj: e.funcByObj, varLitId: e.varLitId,
		lits: &litPool{names: map[string]string{}},
	}
	inner.setPrefixes(e.anonImports(st))
	inner.line("/// %s", anonText(st))
	inner.structClass(class, nil, st, anonComparable(st), nil)
	inner.writeNilChans()
	inner.writeInstDescs()
	inner.flushFrames()
	inner.writeLits()
	body := inner.buf.String()

	inner.buf.Reset()
	inner.buf.WriteString(header)
	inner.line("")
	inner.line("import 'package:goruntime/goruntime.dart';")
	all := append(e.anonImports(st), inner.extra...)
	sort.Slice(all, func(i, j int) bool { return all[i].Path < all[j].Path })
	for _, im := range all {
		inner.line("import '%s' as %s;", im.URI, inner.prefixes[im.Path])
	}
	delete(inner.anon, class)
	for _, uri := range inner.anonURIs() {
		inner.line("import '%s';", uri)
	}
	inner.line("")
	inner.buf.WriteString(body)
	e.errs = append(e.errs, inner.errs...)
	return inner.buf.String(), inner.anon
}

// anonImports lists the Dart library of each package that a field of the shape
// names. The order comes from the import path.
func (e *emitter) anonImports(st *types.Struct) []Import {
	paths := map[string]bool{}
	for i := 0; i < st.NumFields(); i++ {
		typePackages(st.Field(i).Type(), paths, map[types.Type]bool{})
	}
	out := make([]Import, 0, len(paths))
	for path := range paths {
		if uri := e.cfg.URIs[path]; uri != "" {
			out = append(out, Import{Path: path, URI: uri})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// typePackages collects the import path of each named type that the type names.
func typePackages(t types.Type, out map[string]bool, seen map[types.Type]bool) {
	if t == nil || seen[t] {
		return
	}
	seen[t] = true
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		if pkg := t.Obj().Pkg(); pkg != nil {
			out[pkg.Path()] = true
		}
		for i := 0; i < t.TypeArgs().Len(); i++ {
			typePackages(t.TypeArgs().At(i), out, seen)
		}
		if _, ok := t.Underlying().(*types.Struct); !ok {
			typePackages(t.Underlying(), out, seen)
		}
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			typePackages(t.Field(i).Type(), out, seen)
		}
	case *types.Pointer:
		typePackages(t.Elem(), out, seen)
	case *types.Slice:
		typePackages(t.Elem(), out, seen)
	case *types.Array:
		typePackages(t.Elem(), out, seen)
	case *types.Chan:
		typePackages(t.Elem(), out, seen)
	case *types.Map:
		typePackages(t.Key(), out, seen)
		typePackages(t.Elem(), out, seen)
	case *types.Signature:
		typePackages(t.Params(), out, seen)
		typePackages(t.Results(), out, seen)
	case *types.Tuple:
		for i := 0; i < t.Len(); i++ {
			typePackages(t.At(i).Type(), out, seen)
		}
	case *types.Interface:
		for i := 0; i < t.NumMethods(); i++ {
			typePackages(t.Method(i).Type(), out, seen)
		}
	}
}

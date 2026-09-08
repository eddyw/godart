package emit

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
	"github.com/eddyw/godart/internal/typeset"
)

// class writes the Dart class of a named Go struct type. The class holds the
// fields, the copy operations that give the struct value semantics, and the
// methods.
func (e *emitter) class(obj *types.TypeName, named *types.Named, st *types.Struct) {
	if e.structOwner(obj) != nil {
		// The instance of the generic type owns the class, or the package that
		// declares the struct object owns it. Read structOwner.
		return
	}
	group := e.mergeGroup(obj)
	if len(group) > 0 && group[0] != obj {
		// The class of the group already holds the fields and each method.
		return
	}
	methods := e.methods[obj]
	if len(group) > 0 {
		methods = nil
		for _, member := range group {
			methods = append(methods, e.methods[member]...)
		}
	}
	e.structClass(name.Type(obj.Name()), typeParamsOf(named.TypeParams()), st,
		structComparable(named, st), methods)
}

// structComparable reports whether the class of a named struct type needs the
// == operator and the hashCode getter.
//
// types.Comparable gives false for a generic struct that holds a field of a
// type parameter whose constraint is any, because such a parameter is not
// strictly comparable. Go still accepts == on an instance whose type arguments
// are comparable, and it rejects each other instance at the use site. The
// dictionary of the parameter gives equal and hash, thus the class of such a
// struct holds the 2 members. Without them Dart compares 2 instances by
// identity, and a map with such a key finds nothing.
func structComparable(named *types.Named, st *types.Struct) bool {
	if types.Comparable(named) {
		return true
	}
	if named.TypeParams().Len() == 0 {
		return false
	}
	for i := range st.NumFields() {
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

// structClass writes the Dart class of a Go struct type.
//
// An unnamed struct type reaches this function with no type parameter and no
// method. Go gives such a type a structural identity, thus the name comes from
// the shape of the struct.
func (e *emitter) structClass(class string, params []*types.TypeParam,
	st *types.Struct, comparable bool, methods []*ir.Func) {
	head := class + e.typeParamDecl(params)
	fields := make([]string, st.NumFields())
	for i := range fields {
		fields[i] = structFieldName(st.Field(i), i)
		for _, tp := range params {
			if tp.Obj().Name() == st.Field(i).Name() {
				fields[i] += "$f"
			}
		}
	}
	dicts := make([]string, len(params))
	for i, tp := range params {
		dicts[i] = "$d" + typeParamName(tp)
	}
	e.line("final class %s {", head)
	for i, tp := range params {
		e.line("final Dict<%s> %s;", e.dartType(tp), dicts[i])
	}
	for i, f := range fields {
		e.line("%s %s;", e.dartType(st.Field(i).Type()), f)
	}
	e.line("")
	all := append(append([]string{}, dicts...), fields...)
	if len(all) == 0 {
		e.line("%s();", class)
		e.line("%s.$zero();", class)
	} else {
		e.line("%s(%s);", class, "this."+strings.Join(all, ", this."))
		zeros := make([]string, 0, len(fields))
		for i, f := range fields {
			zeros = append(zeros, fmt.Sprintf("%s = %s", f, e.zero(st.Field(i).Type())))
		}
		head := class + ".$zero("
		if len(dicts) > 0 {
			head += "this." + strings.Join(dicts, ", this.")
		}
		head += ")"
		if len(zeros) == 0 {
			e.line("%s;", head)
		} else {
			e.line("%s : %s;", head, strings.Join(zeros, ", "))
		}
	}
	e.line("")
	full := class + e.typeArgsOfParams(params)
	e.writeCopy(full, st, fields, dicts)
	e.writeAssign(full, st, fields)
	if comparable {
		e.writeEqual(full, st, fields)
	}
	for _, m := range methods {
		e.line("")
		e.method(m)
	}
	e.line("}")
	e.line("")
}

func (e *emitter) writeCopy(class string, st *types.Struct, fields, dicts []string) {
	parts := make([]string, 0, len(fields)+len(dicts))
	parts = append(parts, dicts...)
	for i, f := range fields {
		parts = append(parts, e.copyOf(f, st.Field(i).Type()))
	}
	e.line("%s $copy() => %s(%s);", class, class, strings.Join(parts, ", "))
	e.line("")
}

// typeParamDecl writes the type parameters of a declaration.
func (e *emitter) typeParamDecl(params []*types.TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, tp := range params {
		parts[i] = typeParamName(tp)
		if bound := e.typeParamBound(tp); bound != "" {
			parts[i] += " extends " + bound
		}
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

// typeArgsOfParams names the type parameters again, with no bound.
func (e *emitter) typeArgsOfParams(params []*types.TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, tp := range params {
		parts[i] = typeParamName(tp)
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

// dictParams writes 1 dictionary parameter for each type parameter.
func (e *emitter) dictParams(params []*types.TypeParam) []string {
	out := make([]string, len(params))
	for i, tp := range params {
		out[i] = fmt.Sprintf("Dict<%s> $d%s", e.dartType(tp), typeParamName(tp))
	}
	return out
}

func typeParamsOf(list *types.TypeParamList) []*types.TypeParam {
	if list == nil {
		return nil
	}
	out := make([]*types.TypeParam, list.Len())
	for i := range out {
		out[i] = list.At(i)
	}
	return out
}

// copyOf gives the expression that copies 1 field. A struct and an array hold a
// value, thus each one needs a fresh copy. A type parameter reads its copy
// from the dictionary, because the Dart type of the argument is unknown.
func (e *emitter) copyOf(read string, t types.Type) string {
	switch {
	case isStructType(t):
		return read + ".$copy()"
	case isArrayType(t):
		return e.copyArray(read, t)
	case typeset.HoldsParam(t) && typeset.Core(t) == nil:
		return fmt.Sprintf("%s.copyOf(%s)", e.dictOf(t), read)
	}
	return read
}

func (e *emitter) copyArray(read string, t types.Type) string {
	a, _ := types.Unalias(t).Underlying().(*types.Array)
	if a != nil && needsElementCopy(a.Elem()) {
		return fmt.Sprintf("$copyArray(%s, (v) => %s)", read, e.copyOf("v", a.Elem()))
	}
	return fmt.Sprintf("$copyArray(%s)", read)
}

func needsElementCopy(t types.Type) bool {
	return isStructType(t) || isArrayType(t)
}

func (e *emitter) writeAssign(class string, st *types.Struct, fields []string) {
	e.line("void $assign(%s $o) {", class)
	for i, f := range fields {
		t := st.Field(i).Type()
		switch {
		case typeset.HoldsParam(t) && typeset.Core(t) == nil:
			e.line("%s = %s.assignOf(%s, $o.%s);", f, e.dictOf(t), f, f)
		case isStructType(t):
			e.line("%s.$assign($o.%s);", f, f)
		case isArrayType(t):
			a, _ := types.Unalias(t).Underlying().(*types.Array)
			if a != nil && needsElementCopy(a.Elem()) {
				e.line("$assignArray(%s, $o.%s, (a, b) => %s);", f, f, e.assignElem("a", "b", a.Elem()))
				continue
			}
			e.line("$assignArray(%s, $o.%s);", f, f)
		default:
			e.line("%s = $o.%s;", f, f)
		}
	}
	e.line("}")
	e.line("")
}

func (e *emitter) assignElem(dst, src string, t types.Type) string {
	if isStructType(t) {
		return dst + ".$assign(" + src + ")"
	}
	return fmt.Sprintf("$assignArray(%s, %s)", dst, src)
}

func (e *emitter) writeEqual(class string, st *types.Struct, fields []string) {
	parts := make([]string, 0, len(fields))
	hashes := make([]string, 0, len(fields))
	for i, f := range fields {
		t := st.Field(i).Type()
		parts = append(parts, e.equalText(f, "$o."+f, t))
		hashes = append(hashes, e.hashText(f, t))
	}
	e.line("@override")
	if len(parts) == 0 {
		e.line("bool operator ==(Object $o) => $o is %s;", class)
	} else {
		e.line("bool operator ==(Object $o) => $o is %s && %s;", class, strings.Join(parts, " && "))
	}
	e.line("")
	e.line("@override")
	if len(hashes) == 0 {
		e.line("int get hashCode => %d;", 0)
	} else {
		e.line("int get hashCode => Object.hashAll(<Object?>[%s]);", strings.Join(hashes, ", "))
	}
	e.line("")
}

// method writes a method of a struct class.
//
// A method with a pointer receiver holds its body in a top level function, thus
// the class holds a call of that function. Go runs such a method with a nil
// receiver, and Dart calls no member of null.
func (e *emitter) method(f *ir.Func) {
	if !e.inClass[f] {
		e.methodForward(f)
		return
	}
	// The frame of the body takes the type parameters of the class and of
	// the method, exactly as the frame of a top level method does. The
	// signature keeps the own list, because the class head names the rest.
	own := f.TypeParams
	all := own
	if f.Obj != nil {
		if sig, ok := f.Obj.Type().(*types.Signature); ok && sig.RecvTypeParams() != nil {
			all = append(typeParamsOf(sig.RecvTypeParams()), own...)
		}
	}
	params := append(e.dictParams(own), e.paramList(f.Params)...)
	if p := e.frameParam(f.Machine, all); p != "" {
		params = append(params, p)
	}
	e.line("%s %s%s(%s) {", e.results(f), e.methodMember(f),
		e.typeParamDecl(own), strings.Join(params, ", "))
	e.receiver(f)
	f.TypeParams = all
	e.body(f)
	f.TypeParams = own
	e.line("}")
	e.frameOfWith(f, all, fmt.Sprintf("%s.%s%s(%s)", frameRecv(f.Recv), e.methodMember(f),
		typeArgNames(own),
		strings.Join(e.frameArgs(own, f.Params), ", ")))
}

// methodForward writes the member of the class that calls the top level
// function of a method with a pointer receiver.
func (e *emitter) methodForward(f *ir.Func) {
	obj := receiverTypeName(f.Recv.Type)
	if obj == nil {
		e.fail("the method %s has no named receiver type", f.Name)
		return
	}
	dicts := e.dictParams(f.TypeParams)
	params := append(append([]string{}, dicts...), e.paramList(f.Params)...)
	recv := recvTypeParams(obj)
	args := []string{"this"}
	for _, tp := range append(append([]*types.TypeParam{}, recv...), f.TypeParams...) {
		args = append(args, "$d"+typeParamName(tp))
	}
	// A boxed parameter takes a second name, thus the argument reads the same
	// name that the parameter list of this member declares.
	for _, p := range f.Params {
		args = append(args, paramName(p))
	}
	e.line("%s %s%s(%s) => %s%s(%s);", e.results(f), e.methodMember(f),
		e.typeParamDecl(f.TypeParams), strings.Join(params, ", "),
		methodFuncName(obj.Name(), f.Name),
		typeArgNames(append(append([]*types.TypeParam{}, recv...), f.TypeParams...)),
		strings.Join(args, ", "))
}

// frameRecv reads the receiver of a frame. A pointer receiver holds a Dart
// type that accepts null, thus the read needs a null check.
func frameRecv(v *ir.Var) string {
	out := varRead(v)
	if isPointerType(v.Type) {
		return out + "!"
	}
	return out
}

// frameOf writes the frame class of a function that can block.
func (e *emitter) frameOf(f *ir.Func, invoke string) {
	e.frameOfWith(f, f.TypeParams, invoke)
}

// frameOfWith writes the frame class with an explicit list of type parameters.
// The class of a method that stands in a top level function also names the type
// parameters of the receiver type.
func (e *emitter) frameOfWith(f *ir.Func, tps []*types.TypeParam, invoke string) {
	if f.Machine == nil || !f.Machine.Blocking {
		return
	}
	goFile := ""
	if e.cfg.Fset != nil && f.Pos.IsValid() {
		goFile = e.goFile(f.Pos)
	}
	e.writeFrameClass(f.Machine, tps, "", invoke, len(f.Results) > 0,
		e.goFuncName(f), goFile)
}

// typeArgNames writes the type arguments of a call to a generic function.
func typeArgNames(params []*types.TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, tp := range params {
		parts[i] = typeParamName(tp)
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

// receiver binds the receiver name. Go gives a value receiver a copy of the
// value, thus a write inside the method does not reach the caller.
func (e *emitter) receiver(f *ir.Func) {
	if f.Recv == nil {
		return
	}
	// The name takes no `final` keyword. A method of Go can write its own
	// receiver, and `container/ring` holds such a method.
	if isPointerType(f.Recv.Type) {
		e.line("%s %s = this;", e.dartType(f.Recv.Type), f.Recv.Name)
		return
	}
	e.line("%s %s = this.$copy();", e.dartType(f.Recv.Type), f.Recv.Name)
}

func (e *emitter) funcDecl(f *ir.Func) {
	if f.Recv != nil {
		e.methodFunc(f)
		return
	}
	params := append(e.dictParams(f.TypeParams), e.paramList(f.Params)...)
	if p := e.frameParam(f.Machine, f.TypeParams); p != "" {
		params = append(params, p)
	}
	e.line("%s %s%s(%s) {", e.results(f), e.funcName(f),
		e.typeParamDecl(f.TypeParams), strings.Join(params, ", "))
	e.body(f)
	e.line("}")
	e.line("")
	e.frameOf(f, fmt.Sprintf("%s%s(%s)", e.funcName(f), typeArgNames(f.TypeParams),
		strings.Join(e.frameArgs(f.TypeParams, f.Params), ", ")))
}

// methodFunc writes a method of a named type that is not a struct. Such a type
// maps to a Dart type that holds no method, thus the receiver becomes the first
// parameter.
func (e *emitter) methodFunc(f *ir.Func) {
	obj := receiverTypeName(f.Recv.Type)
	if obj == nil {
		e.fail("the method %s has no named receiver type", f.Name)
		return
	}
	// The function names the type parameters of the receiver type first. The
	// body reads the dictionary of each one, and the class of the receiver
	// holds no such name at top level.
	recv := recvTypeParams(obj)
	if f.Obj != nil {
		if sig, ok := f.Obj.Type().(*types.Signature); ok && sig.RecvTypeParams() != nil {
			// The body names the parameter objects of the method itself, and
			// a method can give the receiver parameter a blank name.
			recv = typeParamsOf(sig.RecvTypeParams())
		}
	}
	all := append(recv, f.TypeParams...)
	params := []string{e.dartType(f.Recv.Type) + " " + paramName(f.Recv)}
	params = append(params, e.dictParams(all)...)
	params = append(params, e.paramList(f.Params)...)
	if p := e.frameParam(f.Machine, all); p != "" {
		params = append(params, p)
	}
	prev := e.typeParams
	e.typeParams = all
	e.line("%s %s%s(%s) {", e.results(f), methodFuncName(obj.Name(), f.Name),
		e.typeParamDecl(all), strings.Join(params, ", "))
	// The frame of the body must take the same type parameters as its class,
	// and the class names the parameters of the receiver type first.
	own := f.TypeParams
	f.TypeParams = all
	e.body(f)
	f.TypeParams = own
	e.line("}")
	e.line("")
	e.typeParams = prev
	args := append([]string{varRead(f.Recv)}, e.frameArgs(all, f.Params)...)
	e.frameOfWith(f, all, fmt.Sprintf("%s%s(%s)", methodFuncName(obj.Name(), f.Name),
		typeArgNames(all), strings.Join(args, ", ")))
}

// structFieldName gives the Dart name of a struct field. Each blank field
// takes a name of its own, because Dart forbids 2 parameters of 1 name, and
// Go never reads a blank field.
func structFieldName(f *types.Var, i int) string {
	if f.Name() == "_" {
		return fmt.Sprintf("$b%d", i)
	}
	return fieldBaseName(f)
}

// fieldBaseName gives the Dart name of 1 struct field. An embedded field takes
// a public name even when Go does not export it. Go promotes each member of an
// embedded type across a package boundary, thus a reader of another Dart
// library names the field, and a private Dart name gives a compile error
// there.
func fieldBaseName(f *types.Var) string {
	if !f.Embedded() {
		return name.Package(f.Name())
	}
	out := name.Public(f.Name())
	if out == name.Type(f.Name()) {
		// The Dart class of an unexported Go type already carries the name
		// with 1 dollar sign. A field of that name would hide the class
		// inside the body of its own class.
		out += "$"
	}
	return out
}

// fieldDartName gives the Dart name of a field through its owner type. A
// field whose name equals a type parameter of its own generic type takes a
// suffix, because the Dart class holds the 2 names in 1 scope.
func fieldDartName(owner types.Type, field string) string {
	out := name.Package(field)
	u := types.Unalias(owner)
	if p, ok := typeset.Core(u).(*types.Pointer); ok {
		u = types.Unalias(p.Elem())
	}
	if st, ok := typeset.Core(u).Underlying().(*types.Struct); ok {
		for i := range st.NumFields() {
			if st.Field(i).Name() == field {
				out = fieldBaseName(st.Field(i))
				break
			}
		}
	}
	named, ok := u.(*types.Named)
	if !ok {
		return out
	}
	tps := named.Origin().TypeParams()
	for i := 0; i < tps.Len(); i++ {
		if tps.At(i).Obj().Name() == field {
			return out + "$f"
		}
	}
	return out
}

// methodFuncName builds the name of a method that becomes a top level
// function. The name stays private when Go does not export the method.
//
// The name of the type does not decide the privacy. An exported type can
// embed an unexported type, and the method of the embedded type then reaches
// another package. A private Dart name of another library gives a compile
// error at that call.
func methodFuncName(typeName, methodName string) string {
	n := strings.TrimPrefix(name.Package(typeName), "_") + "$" +
		strings.TrimPrefix(name.Package(methodName), "_")
	if !isExported(methodName) {
		return "_" + n
	}
	return n
}

func isExported(s string) bool {
	return s != "" && s[0] >= 'A' && s[0] <= 'Z'
}

func (e *emitter) funcName(f *ir.Func) string {
	if n, ok := e.initNames[f]; ok {
		return n
	}
	return e.topName(e.pkg, f.Name)
}

func (e *emitter) params(vars []*ir.Var) string {
	return strings.Join(e.paramList(vars), ", ")
}

func (e *emitter) paramList(vars []*ir.Var) []string {
	out := make([]string, len(vars))
	for i, v := range vars {
		out[i] = e.dartType(v.Type) + " " + paramName(v)
	}
	return out
}

// paramName gives the Dart name of a parameter. A boxed parameter takes a
// second name, because the body holds the box under the name of the Go
// parameter.
func paramName(v *ir.Var) string {
	if v.Boxed {
		return v.Name + "$p"
	}
	return v.Name
}

// boxRecv writes the box of a receiver whose address the body takes.
func (e *emitter) boxRecv(v *ir.Var) {
	if v == nil {
		return
	}
	e.boxParams([]*ir.Var{v})
}

// boxParams writes the box of each parameter whose address the body takes. A
// Dart parameter holds the value, thus the box comes at the start of the body.
func (e *emitter) boxParams(vars []*ir.Var) {
	keyword := "final "
	if e.machine != nil {
		keyword = ""
	}
	for _, v := range vars {
		if v.Boxed {
			e.line("%sBox<%s> %s = Box(%s);", keyword, e.dartType(v.Type), v.Name, paramName(v))
		}
	}
}

func (e *emitter) results(f *ir.Func) string {
	return e.resultText(f.Results)
}

// resultText gives the Dart result type. A function with more than 1 result
// returns a record.
func (e *emitter) resultText(results []*ir.Var) string {
	var base string
	switch len(results) {
	case 0:
		base = "void"
	case 1:
		base = e.dartType(results[0].Type)
	default:
		parts := make([]string, len(results))
		for i, r := range results {
			parts[i] = e.dartType(r.Type)
		}
		base = "(" + strings.Join(parts, ", ") + ")"
	}
	return base
}

// body writes the statements of a function. A named result becomes a local
// variable that holds the zero value until a return statement writes it.
func (e *emitter) body(f *ir.Func) {
	e.markFunc(f)
	prev := e.typeParams
	e.typeParams = f.TypeParams
	defer func() { e.typeParams = prev }()
	if f.NoBody {
		if f.DartLib != "" {
			e.dartCall(f)
			return
		}
		e.line("$panic(GoString.fromString(%q));",
			"the function "+f.Name+" holds no body in the subset")
		return
	}
	saved := e.curFuncId
	e.curFuncId = e.funcId(f)
	defer func() { e.curFuncId = saved }()
	if f.Machine != nil {
		if f.Recv != nil && !e.inClass[f] {
			e.boxRecv(f.Recv)
		}
		e.machineBody(f, "")
		return
	}
	e.boxParams(f.Params)
	e.boxRecv(f.Recv)
	e.emitBody(f.Results, f.Defers, f.Body, f.End)
}

// emitBody writes the statements of a function or of a closure. A function
// with a defer statement runs the deferred calls at each return, after the
// body, and on the panic path.
//
// Go lets a deferred call change a named result, thus the return statement
// writes the result variables and leaves a labeled block. The function reads
// the variables after the block.
func (e *emitter) emitBody(results []*ir.Var, defers bool, body *ir.Block, end token.Pos) {
	// A function that names 1 result declares each result. The return path
	// writes each one and reads them together, thus a blank result also needs
	// its variable.
	named := hasNamedResults(results)
	if named {
		for _, r := range results {
			e.declareOne(r, e.zero(r.Type))
		}
	}
	if !defers {
		e.list(body.List)
		e.closeBody(results, body)
		return
	}
	e.line("final DeferStack $defer = DeferStack();")
	// A return statement runs the deferred calls itself, thus the frame of a
	// deferred call names the line of that return, exactly as in Go. The run
	// after the body covers a fall off the end, and the mark of the closing
	// brace gives its line. A second run finds an empty stack and does
	// nothing.
	prevPlain := e.deferPlain
	e.deferPlain = true
	e.line("try {")
	if named {
		e.line("$body: {")
		prevLabel, prevVars := e.returnLabel, e.returnVars
		e.returnLabel, e.returnVars = "$body", results
		e.list(body.List)
		e.returnLabel, e.returnVars = prevLabel, prevVars
		e.line("}")
	} else {
		e.list(body.List)
	}
	e.mark(end)
	e.line("$defer.run();")
	e.line("} catch ($e) {")
	e.line("final ChainState $cs = $chainOpen($e);")
	e.mark(end)
	e.line("if (!$defer.runRecovered()) { $chainClose($cs); rethrow; }")
	e.line("$chainClose($cs);")
	e.line("}")
	e.deferPlain = prevPlain
	if named {
		e.line("return %s;", e.resultRead(results))
		return
	}
	// A panic that a deferred call recovered leaves the body at the catch,
	// and Go gives such a return the zero value.
	if len(results) > 0 {
		e.line("return %s;", e.zeroResult(results))
		return
	}
	e.closeBody(results, body)
}

// closeBody ends a function that gives a result. Go proves that the end of
// such a function is unreachable, but the flow analysis of Dart cannot.
func (e *emitter) closeBody(results []*ir.Var, body *ir.Block) {
	if len(results) == 0 || len(body.List) == 0 {
		return
	}
	if _, ok := body.List[len(body.List)-1].(*ir.Return); ok {
		return
	}
	e.line("$unreachable();")
}

func (e *emitter) resultRead(results []*ir.Var) string {
	if len(results) == 1 {
		return varRead(results[0])
	}
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = varRead(r)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// varRead reads a variable. A boxed variable holds a box, thus the read takes
// the field of the box.
func varRead(v *ir.Var) string {
	if v.Boxed {
		return v.Name + ".v"
	}
	return v.Name
}

// methodMember gives the name of a method inside the class of its receiver.
func (e *emitter) methodMember(f *ir.Func) string {
	if f.Obj != nil {
		return e.memberName(f.Obj)
	}
	return name.Package(f.Name)
}

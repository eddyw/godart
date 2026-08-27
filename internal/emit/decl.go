package emit

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
)

// class writes the Dart class of a Go struct type. The class holds the fields,
// the copy operations that give the struct value semantics, and the methods.
func (e *emitter) class(obj *types.TypeName, named *types.Named, st *types.Struct) {
	class := name.Type(obj.Name())
	params := typeParamsOf(named.TypeParams())
	head := class + e.typeParamDecl(params)
	fields := make([]string, st.NumFields())
	for i := range fields {
		fields[i] = name.Package(st.Field(i).Name())
	}
	dicts := make([]string, len(params))
	for i, tp := range params {
		dicts[i] = "$d" + name.Local(tp.Obj().Name())
	}
	e.line("final class %s {", head)
	for i, tp := range params {
		e.line("final Dict<%s> %s;", name.Local(tp.Obj().Name()), dicts[i])
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
	if types.Comparable(named) {
		e.writeEqual(full, st, fields)
	}
	for _, m := range e.methods[obj] {
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
		parts[i] = name.Local(tp.Obj().Name())
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
		parts[i] = name.Local(tp.Obj().Name())
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

// dictParams writes 1 dictionary parameter for each type parameter.
func (e *emitter) dictParams(params []*types.TypeParam) []string {
	out := make([]string, len(params))
	for i, tp := range params {
		n := name.Local(tp.Obj().Name())
		out[i] = fmt.Sprintf("Dict<%s> $d%s", n, n)
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
// value, thus each one needs a fresh copy.
func (e *emitter) copyOf(read string, t types.Type) string {
	switch {
	case isStructType(t):
		return read + ".$copy()"
	case isArrayType(t):
		return e.copyArray(read, t)
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
func (e *emitter) method(f *ir.Func) {
	params := append(e.dictParams(f.TypeParams), e.paramList(f.Params)...)
	e.line("%s %s%s(%s)%s {", e.results(f), name.Package(f.Name),
		e.typeParamDecl(f.TypeParams), strings.Join(params, ", "), asyncWord(f.Async))
	e.receiver(f)
	e.body(f)
	e.line("}")
}

// receiver binds the receiver name. Go gives a value receiver a copy of the
// value, thus a write inside the method does not reach the caller.
func (e *emitter) receiver(f *ir.Func) {
	if f.Recv == nil {
		return
	}
	if isPointerType(f.Recv.Type) {
		e.line("final %s %s = this;", e.dartType(f.Recv.Type), f.Recv.Name)
		return
	}
	e.line("final %s %s = this.$copy();", e.dartType(f.Recv.Type), f.Recv.Name)
}

func (e *emitter) funcDecl(f *ir.Func) {
	if f.Recv != nil {
		e.methodFunc(f)
		return
	}
	params := append(e.dictParams(f.TypeParams), e.paramList(f.Params)...)
	e.line("%s %s%s(%s)%s {", e.results(f), e.funcName(f),
		e.typeParamDecl(f.TypeParams), strings.Join(params, ", "), asyncWord(f.Async))
	e.body(f)
	e.line("}")
	e.line("")
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
	params := append([]string{e.dartType(f.Recv.Type) + " " + f.Recv.Name}, e.paramList(f.Params)...)
	e.line("%s %s(%s)%s {", e.results(f), methodFuncName(obj.Name(), f.Name),
		strings.Join(params, ", "), asyncWord(f.Async))
	e.body(f)
	e.line("}")
	e.line("")
}

// methodFuncName builds the name of a method that becomes a top level
// function. The name stays private when Go does not export the type or the
// method.
func methodFuncName(typeName, methodName string) string {
	n := strings.TrimPrefix(name.Package(typeName), "_") + "$" +
		strings.TrimPrefix(name.Package(methodName), "_")
	if !isExported(typeName) || !isExported(methodName) {
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
	return name.Package(f.Name)
}

func (e *emitter) params(vars []*ir.Var) string {
	return strings.Join(e.paramList(vars), ", ")
}

func (e *emitter) paramList(vars []*ir.Var) []string {
	out := make([]string, len(vars))
	for i, v := range vars {
		out[i] = e.dartType(v.Type) + " " + v.Name
	}
	return out
}

func asyncWord(async bool) string {
	if async {
		return " async"
	}
	return ""
}

func (e *emitter) results(f *ir.Func) string {
	return e.resultText(f.Results, f.Async)
}

// resultText gives the Dart result type. A function that can block returns a
// Future, because it compiles to a Dart async function.
func (e *emitter) resultText(results []*ir.Var, async bool) string {
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
	if async {
		return "Future<" + base + ">"
	}
	return base
}

// body writes the statements of a function. A named result becomes a local
// variable that holds the zero value until a return statement writes it.
func (e *emitter) body(f *ir.Func) {
	e.emitBody(f.Results, f.Defers, f.Body)
}

// emitBody writes the statements of a function or of a closure. A function
// with a defer statement runs the deferred calls through a finally block.
//
// Go lets a deferred call change a named result, thus the return statement
// writes the result variables and leaves a labeled block. The function reads
// the variables after the finally block.
func (e *emitter) emitBody(results []*ir.Var, defers bool, body *ir.Block) {
	for _, r := range results {
		if r.Named {
			e.line("%s %s = %s;", e.dartType(r.Type), r.Name, e.zero(r.Type))
		}
	}
	if !defers {
		for _, s := range body.List {
			e.stmt(s)
		}
		e.closeBody(results, body)
		return
	}
	e.line("final DeferStack $defer = DeferStack();")
	named := false
	for _, r := range results {
		named = named || r.Named
	}
	e.line("try {")
	if named {
		e.line("$body: {")
		prevLabel, prevVars := e.returnLabel, e.returnVars
		e.returnLabel, e.returnVars = "$body", results
		for _, s := range body.List {
			e.stmt(s)
		}
		e.returnLabel, e.returnVars = prevLabel, prevVars
		e.line("}")
	} else {
		for _, s := range body.List {
			e.stmt(s)
		}
	}
	e.line("} finally {")
	e.line("$defer.run();")
	e.line("}")
	if named {
		e.line("return %s;", e.resultRead(results))
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
		return results[0].Name
	}
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = r.Name
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

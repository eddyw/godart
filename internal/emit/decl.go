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
	fields := make([]string, st.NumFields())
	for i := range fields {
		fields[i] = name.Package(st.Field(i).Name())
	}
	e.line("final class %s {", class)
	for i, f := range fields {
		e.line("%s %s;", e.dartType(st.Field(i).Type()), f)
	}
	e.line("")
	if len(fields) == 0 {
		e.line("%s();", class)
		e.line("%s.$zero();", class)
	} else {
		e.line("%s(%s);", class, "this."+strings.Join(fields, ", this."))
		zeros := make([]string, len(fields))
		for i, f := range fields {
			zeros[i] = fmt.Sprintf("%s = %s", f, e.zero(st.Field(i).Type()))
		}
		e.line("%s.$zero() : %s;", class, strings.Join(zeros, ", "))
	}
	e.line("")
	e.writeCopy(class, st, fields)
	e.writeAssign(class, st, fields)
	if types.Comparable(named) {
		e.writeEqual(class, st, fields)
	}
	for _, m := range e.methods[obj] {
		e.line("")
		e.method(m)
	}
	e.line("}")
	e.line("")
}

func (e *emitter) writeCopy(class string, st *types.Struct, fields []string) {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = e.copyOf(f, st.Field(i).Type())
	}
	e.line("%s $copy() => %s(%s);", class, class, strings.Join(parts, ", "))
	e.line("")
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
		if isArrayType(t) {
			parts = append(parts, fmt.Sprintf("$equalArray(%s, $o.%s)", f, f))
			hashes = append(hashes, fmt.Sprintf("Object.hashAll(%s)", f))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s == $o.%s", f, f))
		hashes = append(hashes, f)
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
	e.line("%s %s(%s) {", e.results(f), name.Package(f.Name), e.params(f.Params))
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
	e.line("%s %s(%s) {", e.results(f), e.funcName(f), e.params(f.Params))
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
	e.line("%s %s(%s) {", e.results(f), methodFuncName(obj.Name(), f.Name), strings.Join(params, ", "))
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

func (e *emitter) results(f *ir.Func) string {
	switch len(f.Results) {
	case 0:
		return "void"
	case 1:
		return e.dartType(f.Results[0].Type)
	}
	parts := make([]string, len(f.Results))
	for i, r := range f.Results {
		parts[i] = e.dartType(r.Type)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// body writes the statements of a function. A named result becomes a local
// variable that holds the zero value until a return statement writes it.
func (e *emitter) body(f *ir.Func) {
	for _, r := range f.Results {
		if r.Named {
			e.line("%s %s = %s;", e.dartType(r.Type), r.Name, e.zero(r.Type))
		}
	}
	for _, s := range f.Body.List {
		e.stmt(s)
	}
}

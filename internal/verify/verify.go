// Package verify runs stage 2 of the compiler. It enforces the Go subset that
// the compiler supports, and it rejects all other code with a source position.
package verify

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/eddyw/godart/internal/overlay"
)

// Error reports 1 rejected construct.
type Error struct {
	Pos  token.Position
	Msg  string
	Hint string
}

func (e *Error) Error() string {
	if e.Hint == "" {
		return fmt.Sprintf("%s: %s", e.Pos, e.Msg)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Pos, e.Msg, e.Hint)
}

// Package rejects each construct of the files that falls outside the subset.
// The result joins the errors in source order.
func Package(fset *token.FileSet, files []*ast.File, info *types.Info) error {
	v := &verifier{fset: fset, info: info, bodyless: bodyless(files, info)}
	for _, f := range files {
		ast.Inspect(f, v.visit)
	}
	return errors.Join(v.errs...)
}

// bodyless collects each function of the package that holds no body. An
// assembly file holds the code of such a function, or a go:linkname directive
// only exports the name.
func bodyless(files []*ast.File, info *types.Info) map[types.Object]bool {
	out := map[types.Object]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body != nil {
				continue
			}
			if obj := info.Defs[fn.Name]; obj != nil {
				out[obj] = true
			}
		}
	}
	return out
}

type verifier struct {
	fset *token.FileSet
	info *types.Info
	errs []error
	// bodyless holds each function of the package that holds no body. A call of
	// one reaches a rejection, and the declaration alone stays in the subset.
	bodyless map[types.Object]bool
}

func (v *verifier) reject(pos token.Pos, hint string, format string, args ...any) {
	v.errs = append(v.errs, &Error{
		Pos:  v.fset.Position(pos),
		Msg:  fmt.Sprintf(format, args...),
		Hint: hint,
	})
}

func (v *verifier) visit(n ast.Node) bool {
	v.checkExported(n)
	v.checkTypeParamOp(n)
	v.checkBodyless(n)
	switch n := n.(type) {
	case *ast.ImportSpec:
		v.checkImport(n)
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			v.checkAddress(n)
		}
	case *ast.CallExpr:
		v.checkCall(n)
	case *ast.Ident:
		v.checkComplex(n)
	}
	return true
}

// checkComplex rejects a complex number. The runtime library holds no complex
// class yet, thus the emitter would write a value with no type.
func (v *verifier) checkComplex(e *ast.Ident) {
	t := v.typeOf(e)
	if t == nil || !isComplex(t) {
		return
	}
	v.reject(e.Pos(), "the runtime library holds no complex class",
		"the compiler rejects the type %s", types.TypeString(t, nil))
}

func isComplex(t types.Type) bool {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	return ok && b.Info()&types.IsComplex != 0
}

func isFloat(t types.Type) bool {
	b, ok := types.Unalias(t).Underlying().(*types.Basic)
	return ok && b.Info()&types.IsFloat != 0
}

func (v *verifier) checkImport(spec *ast.ImportSpec) {
	path, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		return
	}
	switch {
	case path == "unsafe":
		// A blank import performs no operation. The standard library writes it
		// before a go:linkname directive, and such a directive needs no Dart
		// form when it only exports a name.
		if spec.Name != nil && spec.Name.Name == "_" {
			return
		}
		v.reject(spec.Pos(), "the subset has no pointer arithmetic",
			"the compiler rejects the package unsafe")
	case path == "C":
		v.reject(spec.Pos(), "Dart cannot link a C object file",
			"the compiler rejects cgo")
	case overlay.Outside(path):
		v.reject(spec.Pos(), "the standard library surface is curated",
			"the package %s is outside the supported surface", path)
	}
}

// checkAddress rejects the address of a value that holds no reference and that
// names no location.
//
// A struct and an array give a Dart object, thus a pointer to one is the object
// itself. Each other value needs a cell. A variable and a composite literal each
// hold a box, and a field of a struct and an element of a slice or an array hold
// a $Ref of the object and the index. Each other form holds no location, thus
// the compiler rejects its address.
func (v *verifier) checkAddress(e *ast.UnaryExpr) {
	t := v.typeOf(e.X)
	if t == nil || isStruct(t) || isArray(t) {
		return
	}
	if !v.isVariable(e.X) {
		if v.isLocation(e.X) {
			return
		}
		v.reject(e.Pos(), "a value of such a type needs a cell, and only a variable or a location holds one",
			"the compiler rejects the address of %s", types.TypeString(t, nil))
		return
	}
	// A package variable that holds a box carries the box in its declaration,
	// thus each package that reads the variable must know the box. An exported
	// variable reaches another package, and the archive holds no such field.
	if id, ok := unparen(e.X).(*ast.Ident); ok && id.IsExported() {
		if obj, ok := v.info.Uses[id].(*types.Var); ok && isGlobal(obj) && importable(obj.Pkg()) {
			v.reject(e.Pos(), "the box of a package variable does not cross a package boundary",
				"the compiler rejects the address of the exported package variable %s", id.Name)
		}
	}
}

// importable reports whether an import of another package can name the
// package. No import names a main package or an external test package, thus a
// box of an exported variable of one stays inside the package.
func importable(pkg *types.Package) bool {
	return pkg != nil && pkg.Name() != "main" && !strings.HasSuffix(pkg.Name(), "_test")
}

// isGlobal reports whether the variable stands at package level.
func isGlobal(obj *types.Var) bool {
	return obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope()
}

// isLocation reports whether the expression names a field of a struct, an
// element of a slice or an array, or a composite literal. A pointer to a field
// and to an element holds the object and the index, and a pointer to a literal
// holds a box of its own.
func (v *verifier) isLocation(e ast.Expr) bool {
	switch e := unparen(e).(type) {
	case *ast.CompositeLit:
		return true
	case *ast.SelectorExpr:
		obj, ok := v.info.Uses[e.Sel].(*types.Var)
		return ok && obj.IsField()
	case *ast.IndexExpr:
		t := v.typeOf(e.X)
		if t == nil {
			return false
		}
		u := types.Unalias(t).Underlying()
		if p, ok := u.(*types.Pointer); ok {
			u = types.Unalias(p.Elem()).Underlying()
		}
		switch u.(type) {
		case *types.Slice, *types.Array:
			return true
		}
	}
	return false
}

// isVariable reports whether the expression names a variable. A parameter and a
// package variable each count as one.
func (v *verifier) isVariable(e ast.Expr) bool {
	id, ok := unparen(e).(*ast.Ident)
	if !ok {
		return false
	}
	if _, ok := v.info.Uses[id].(*types.Var); ok {
		return true
	}
	_, ok = v.info.Defs[id].(*types.Var)
	return ok
}

// checkBodyless rejects a read of a function of the package that holds no body.
// Such a function reaches no Dart code, thus a call of it would panic.
func (v *verifier) checkBodyless(n ast.Node) {
	id, ok := n.(*ast.Ident)
	if !ok {
		return
	}
	obj := v.info.Uses[id]
	if obj == nil || !v.bodyless[obj] {
		return
	}
	v.reject(id.Pos(), "an assembly file or a go:linkname directive holds the code",
		"the function %s holds no body, thus the compiler writes no Dart code for it",
		id.Name)
}

func (v *verifier) checkCall(call *ast.CallExpr) {
	id, _ := unparen(call.Fun).(*ast.Ident)
	if id == nil {
		return
	}
	b, _ := v.info.Uses[id].(*types.Builtin)
	if b == nil {
		return
	}
	switch b.Name() {
	case "complex", "real", "imag":
		v.reject(call.Pos(), "the runtime library holds no complex class",
			"the compiler rejects the builtin %s", b.Name())
	case "recover":
		v.reject(call.Pos(), "a panic stops the program",
			"the compiler rejects a call to recover")
	case "len", "cap", "print", "println":
		v.checkTypeParamBuiltin(call, b.Name())
	}
}

func (v *verifier) typeOf(e ast.Expr) types.Type {
	tv, ok := v.info.Types[e]
	if !ok {
		return nil
	}
	return tv.Type
}

func isStruct(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Struct)
	return ok
}

// isArray reports whether the type is an array. The Dart form of an array is a
// List, and a List already holds a reference, thus the address of an array
// needs no box.
func isArray(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Array)
	return ok
}

func unparen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

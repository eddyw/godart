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
	v := &verifier{fset: fset, info: info}
	for _, f := range files {
		ast.Inspect(f, v.visit)
	}
	return errors.Join(v.errs...)
}

type verifier struct {
	fset *token.FileSet
	info *types.Info
	errs []error
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

// checkAddress rejects the address of a value that is not a struct. A pointer
// to a struct is the Dart object reference, thus the runtime needs no pointer
// cell. All other locations hold a Dart value, and a reference to such a
// location does not exist.
// checkAddress rejects the address of a value that holds no reference and that
// is not a variable.
//
// A struct and an array give a Dart object, thus a pointer to one is the object
// itself. Each other value needs a box, and only a variable can hold a box. A
// field of a struct, an element of a slice, and an element of a map live inside
// another object, thus the compiler writes no box for them.
func (v *verifier) checkAddress(e *ast.UnaryExpr) {
	t := v.typeOf(e.X)
	if t == nil || isStruct(t) || isArray(t) {
		return
	}
	if v.isVariable(e.X) {
		return
	}
	v.reject(e.Pos(), "a value of such a type needs a box, and only a variable holds one",
		"the compiler rejects the address of %s", types.TypeString(t, nil))
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
	case "len", "cap":
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

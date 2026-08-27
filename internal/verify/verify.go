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

// supported lists the standard library packages of the curated surface. A
// package enters this set when it compiles inside the subset, or when an
// overlay replaces it.
var supported = supportedSet()

func supportedSet() map[string]bool {
	out := map[string]bool{}
	for _, path := range overlay.Paths() {
		out[path] = true
	}
	return out
}

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
	case *ast.FuncDecl:
		v.checkTestMain(n)
	}
	return true
}

// checkTestMain rejects a TestMain function. Such a function replaces the
// entry point of a test binary, and the compiler writes that entry point.
func (v *verifier) checkTestMain(d *ast.FuncDecl) {
	if d.Name.Name != "TestMain" || d.Recv != nil {
		return
	}
	if !strings.HasSuffix(v.fset.Position(d.Pos()).Filename, "_test.go") {
		return
	}
	v.reject(d.Pos(), "gart writes the entry point of a test binary",
		"the compiler rejects a TestMain function")
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
		v.reject(spec.Pos(), "the subset has no pointer arithmetic",
			"the compiler rejects the package unsafe")
	case path == "C":
		v.reject(spec.Pos(), "Dart cannot link a C object file",
			"the compiler rejects cgo")
	case isStandard(path) && !supported[path]:
		v.reject(spec.Pos(), "the standard library surface is curated",
			"the package %s is outside the supported surface", path)
	}
}

// isStandard reports whether path names a standard library package. The first
// element of a module path always holds a dot, and a standard path never does.
func isStandard(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

// checkAddress rejects the address of a value that is not a struct. A pointer
// to a struct is the Dart object reference, thus the runtime needs no pointer
// cell. All other locations hold a Dart value, and a reference to such a
// location does not exist.
func (v *verifier) checkAddress(e *ast.UnaryExpr) {
	t := v.typeOf(e.X)
	if t == nil || isStruct(t) {
		return
	}
	v.reject(e.Pos(), "only a pointer to a struct stays in the subset",
		"the compiler rejects the address of %s", types.TypeString(t, nil))
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
	case "new":
		v.checkNew(call)
	}
}

// checkNew rejects new(T) and new(x) when the result is not a pointer to a
// struct. Go 1.26 added the expression form of new.
func (v *verifier) checkNew(call *ast.CallExpr) {
	p, _ := types.Unalias(v.typeOf(call)).(*types.Pointer)
	if p == nil || isStruct(p.Elem()) {
		return
	}
	v.reject(call.Pos(), "only a pointer to a struct stays in the subset",
		"the compiler rejects new(%s)", types.TypeString(p.Elem(), nil))
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

func unparen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

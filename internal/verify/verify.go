// Package verify runs stage 2 of the compiler. It enforces the Go subset that
// the compiler supports, and it rejects all other code with a source position.
package verify

import (
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/overlay"
	"github.com/eddyw/godart/internal/typeset"
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
//
// embeds holds the path of each file that the loader rewrote for a
// `//go:embed` directive. The constructor of the package embed stands in such
// a file alone. localPackages identifies module packages and synthetic file lists.
func Package(fset *token.FileSet, files []*ast.File, info *types.Info, embeds, localPackages map[string]bool) error {
	v := &verifier{fset: fset, info: info, bodyless: bodyless(files, info), deadReads: deadBodylessReads(files, info), embeds: embeds, localPackages: localPackages}
	for _, f := range files {
		v.checkDartDecls(f)
	}
	for _, f := range files {
		v.unsafeValues = v.onlySafeUnsafeValues(f)
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
			// A //gart:dart directive names the Dart symbol of the
			// declaration, thus stage 5 writes a call and the read stands.
			if ir.DartMarker(fn.Doc) != nil {
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
	bodyless      map[types.Object]bool
	deadReads     map[*ast.Ident]bool
	localPackages map[string]bool
	// embeds holds the path of each file that the loader rewrote for a
	// `//go:embed` directive.
	embeds       map[string]bool
	unsafeValues bool
}

func (v *verifier) reject(pos token.Pos, hint string, format string, args ...any) {
	v.errs = append(v.errs, &Error{
		Pos:  v.fset.Position(pos),
		Msg:  fmt.Sprintf(format, args...),
		Hint: hint,
	})
}

func (v *verifier) visit(n ast.Node) bool {
	if call, ok := n.(*ast.CallExpr); ok && v.unsafeLayoutCall(call) {
		return false
	}
	v.checkTypeParamOp(n)
	v.checkBodyless(n)
	switch n := n.(type) {
	case *ast.ArrayType:
		if array, ok := v.info.TypeOf(n).(*types.Array); ok && oversized(array, 1<<50) {
			v.reject(n.Pos(), "", "type %s larger than address space", array)
		}
	case *ast.MapType:
		m := v.info.TypeOf(n).(*types.Map)
		if oversized(m.Elem(), 1<<31) {
			v.reject(n.Pos(), "", "map element type too large")
		}
		if oversized(m.Key(), 1<<31) {
			v.reject(n.Pos(), "", "map key type too large")
		}
	case *ast.ChanType:
		elem := v.info.TypeOf(n).(*types.Chan).Elem()
		if oversized(elem, 1<<16) {
			v.reject(n.Pos(), "", "channel element type too large (>64kB)")
		}
	case *ast.ImportSpec:
		v.checkImport(n)
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			v.checkAddress(n)
		}
	case *ast.CallExpr:
		if !v.safeUnsafeConversion(n) {
			v.reject(n.Pos(), "unsafe conversions support typed pointers, nil and constant zero", "the compiler rejects the package unsafe conversion")
		}
		v.checkCall(n)
	case *ast.SelectorExpr:
		v.checkEmbedNew(n)
	}
	return true
}

// checkEmbedNew rejects a read of the constructor of the package embed.
//
// The real package embed of Go holds no such name. The compiler of Go writes
// the file list of an FS into the data of the program, and this compiler holds
// no directive of the linker, thus the overlay takes a constructor and the
// loader writes 1 call of it for each `//go:embed` variable. A file of the
// user that names it would compile here and fail under the go command.
func (v *verifier) checkEmbedNew(sel *ast.SelectorExpr) {
	if sel.Sel.Name != "New" {
		return
	}
	obj, ok := v.info.Uses[sel.Sel].(*types.Func)
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "embed" {
		return
	}
	if v.embeds[v.fset.Position(sel.Pos()).Filename] {
		return
	}
	v.reject(sel.Pos(), "the package embed of Go holds no such name",
		"a value of embed.FS comes from a //go:embed directive")
}

func (v *verifier) checkImport(spec *ast.ImportSpec) {
	path, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		return
	}
	var object types.Object
	if spec.Name == nil {
		object = v.info.Implicits[spec]
	} else {
		object = v.info.Defs[spec.Name]
	}
	if pkg, ok := object.(*types.PkgName); ok {
		path = pkg.Imported().Path()
	}
	switch {
	case path == "unsafe":
		if v.unsafeValues {
			return
		}
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
	case !v.localPackages[path] && overlay.Outside(path):
		v.reject(spec.Pos(), "the standard library surface is curated",
			"the package %s is outside the supported surface", path)
	}
}

func (v *verifier) unsafeLayoutCall(call *ast.CallExpr) bool {
	var id *ast.Ident
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		id = fun.Sel
	case *ast.Ident:
		id = fun
	default:
		return false
	}
	obj, ok := v.info.Uses[id].(*types.Builtin)
	return ok && obj.Pkg() == types.Unsafe && (obj.Name() == "Sizeof" || obj.Name() == "Alignof" || obj.Name() == "Offsetof")
}

func (v *verifier) onlySafeUnsafeValues(file *ast.File) bool {
	allowed := true
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if v.unsafeLayoutCall(call) {
				return false
			}

		}
		if id, ok := n.(*ast.Ident); ok {
			if obj := v.info.Uses[id]; obj != nil && obj.Pkg() == types.Unsafe && obj.Name() != "Pointer" {
				allowed = false
			}
		}
		return true
	})
	return allowed
}

func (v *verifier) safeUnsafeConversion(call *ast.CallExpr) bool {
	if !v.info.Types[call.Fun].IsType() || len(call.Args) != 1 {
		return true
	}
	from, to := v.info.TypeOf(call.Args[0]), v.info.TypeOf(call)
	for _, typ := range []types.Type{from, to} {
		if parameter, ok := types.Unalias(typ).(*types.TypeParam); ok && typeset.Core(parameter) == nil {
			terms, _ := typeset.Terms(parameter)
			for _, term := range terms {
				if unsafePointerType(term) {
					return false
				}
			}
		}
	}
	if unsafePointerType(to) && !unsafePointerType(from) && from != types.Typ[types.UntypedNil] {
		value := v.info.Types[call.Args[0]].Value
		_, pointer := typeset.Core(from).(*types.Pointer)
		if !pointer && (value == nil || value.Kind() != constant.Int || constant.Sign(value) != 0) {
			return false
		}
	}
	if unsafePointerType(from) && !unsafePointerType(to) {
		if _, boxing := to.Underlying().(*types.Interface); !boxing {
			return false
		}
	}
	return true
}

func unsafePointerType(t types.Type) bool {
	basic, ok := typeset.Core(t).(*types.Basic)
	return ok && basic.Kind() == types.UnsafePointer
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

}

// isLocation reports whether the expression names a field of a struct, an
// element of a slice or an array, or a composite literal. A pointer to a field
// and to an element holds the object and the index, and a pointer to a literal
// holds a box of its own.
func (v *verifier) isLocation(e ast.Expr) bool {
	switch e := unparen(e).(type) {
	case *ast.CompositeLit, *ast.StarExpr:
		return true
	case *ast.SelectorExpr:
		obj, ok := v.info.Uses[e.Sel].(*types.Var)
		return ok && (obj.IsField() || obj.Parent() == obj.Pkg().Scope())
	case *ast.IndexExpr:
		t := v.typeOf(e.X)
		if t == nil {
			return false
		}
		if typeset.ArrayOrSlice(t) {
			return true
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
	if !ok || v.deadReads[id] {
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
	case "len", "cap", "print", "println":
		v.checkTypeParamBuiltin(call, b.Name())
		if b.Name() == "print" || b.Name() == "println" {
			for _, arg := range call.Args {
				t := v.info.TypeOf(arg)
				switch types.Unalias(t).Underlying().(type) {
				case *types.Array, *types.Struct:
					v.reject(call.Pos(), "", "illegal types for operand: %s\n\t%s", b.Name(), t)
				}
			}
		}
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

func oversized(t types.Type, limit int64) bool {
	if !knownLayout(t) {
		return false
	}
	size := types.SizesFor("gc", "amd64").Sizeof(t)
	return size < 0 || size >= limit
}

func knownLayout(t types.Type) bool {
	if _, ok := types.Unalias(t).(*types.TypeParam); ok {
		return false
	}
	switch t := t.Underlying().(type) {
	case *types.Array:
		return knownLayout(t.Elem())
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if !knownLayout(t.Field(i).Type()) {
				return false
			}
		}
	}
	return true
}

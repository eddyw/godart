package verify

import (
	"go/ast"
	"go/types"

	"github.com/eddyw/godart/internal/ir"
)

// The external Dart declaration.
//
// A Go function declaration with no body that carries //gart:dart names 1 Dart
// symbol, and stage 5 writes a direct call of it. The value boundary of such a
// call holds no descriptor, thus each parameter type and the result type must
// hold 1 Dart form that needs no conversion table.
//
// The rules below reject each other type with a source position. A wider set
// needs a conversion for each direction, and a wrong conversion would give a
// wrong value with no report.

// checkDartDecls rejects each external Dart declaration that the value
// boundary cannot carry.
func (v *verifier) checkDartDecls(f *ast.File) {
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body != nil {
			continue
		}
		ref := ir.DartMarker(fn.Doc)
		if ref == nil {
			continue
		}
		if fn.Recv != nil {
			v.reject(fn.Pos(), "declare a function of the package",
				"the method %s carries %s, and only a function carries it",
				fn.Name.Name, ir.DartDirective)
			continue
		}
		obj, _ := v.info.Defs[fn.Name].(*types.Func)
		if obj == nil {
			continue
		}
		sig, _ := obj.Type().(*types.Signature)
		if sig == nil {
			continue
		}
		if sig.TypeParams() != nil && sig.TypeParams().Len() > 0 {
			v.reject(fn.Pos(), "declare 1 function for each type",
				"the function %s carries %s and a type parameter",
				fn.Name.Name, ir.DartDirective)
			continue
		}
		if sig.Variadic() {
			v.reject(fn.Pos(), "declare each parameter",
				"the function %s carries %s and a variadic parameter",
				fn.Name.Name, ir.DartDirective)
			continue
		}
		for i := range sig.Params().Len() {
			p := sig.Params().At(i)
			if !dartBoundary(p.Type()) {
				v.reject(p.Pos(), "the boundary carries bool, int, float64, and string",
					"the parameter %s of %s holds the type %s, and %s carries no form of it",
					p.Name(), fn.Name.Name, p.Type(), ir.DartDirective)
			}
		}
		if sig.Results().Len() > 1 {
			v.reject(fn.Pos(), "a Dart function gives 1 value",
				"the function %s carries %s and %d results",
				fn.Name.Name, ir.DartDirective, sig.Results().Len())
			continue
		}
		if sig.Results().Len() == 1 {
			r := sig.Results().At(0)
			if !dartBoundary(r.Type()) {
				v.reject(fn.Pos(), "the boundary carries bool, int, float64, and string",
					"the result of %s holds the type %s, and %s carries no form of it",
					fn.Name.Name, r.Type(), ir.DartDirective)
			}
		}
	}
}

// dartBoundary reports whether a type crosses the boundary of an external Dart
// call with no conversion table.
//
// Each type below holds 1 Dart form. A bool and a float64 pass as they are, a
// Go int is an extension type over the Dart int, and a GoString holds 2
// methods that give the Dart String and read it back.
func dartBoundary(t types.Type) bool {
	basic, ok := types.Unalias(t).(*types.Basic)
	if !ok {
		return false
	}
	switch basic.Kind() {
	case types.Bool, types.Int, types.Float64, types.String:
		return true
	}
	return false
}

package verify

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/eddyw/godart/internal/typeset"
)

// checkExported rejects an unnamed struct type with an unexported field in an
// exported declaration.
//
// A shape whose fields all carry an exported name reaches a file of its own,
// and each package that holds the shape writes that file with equal bytes. Go
// gives an unexported field the identity of the package that declares it, and
// Dart holds a name with a leading underscore private to 1 library. Thus the
// class of such a shape stays inside 1 package, and 2 packages give 2 classes
// that Dart holds apart.
func (v *verifier) checkExported(n ast.Node) {
	switch n := n.(type) {
	case *ast.FuncDecl:
		if !v.exportedFunc(n) {
			return
		}
		obj := v.info.Defs[n.Name]
		if obj == nil {
			return
		}
		v.rejectAnon(n.Name.Pos(), obj.Type(), "the function "+n.Name.Name)
	case *ast.ValueSpec:
		for _, id := range n.Names {
			obj := v.info.Defs[id]
			if obj == nil || !id.IsExported() {
				continue
			}
			if _, ok := obj.(*types.Var); !ok {
				continue
			}
			v.rejectAnon(id.Pos(), obj.Type(), "the variable "+id.Name)
		}
	case *ast.TypeSpec:
		if !n.Name.IsExported() {
			return
		}
		obj := v.info.Defs[n.Name]
		if obj == nil {
			return
		}
		v.rejectAnon(n.Name.Pos(), obj.Type(), "the type "+n.Name.Name)
	}
}

// exportedFunc reports whether another package can call the function. A method
// needs an exported name and an exported receiver type.
func (v *verifier) exportedFunc(n *ast.FuncDecl) bool {
	if !n.Name.IsExported() {
		return false
	}
	if n.Recv == nil || len(n.Recv.List) == 0 {
		return true
	}
	return receiverExported(n.Recv.List[0].Type)
}

func receiverExported(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.StarExpr:
		return receiverExported(e.X)
	case *ast.IndexExpr:
		return receiverExported(e.X)
	case *ast.IndexListExpr:
		return receiverExported(e.X)
	case *ast.Ident:
		return e.IsExported()
	}
	return false
}

func (v *verifier) rejectAnon(pos token.Pos, t types.Type, what string) {
	if !mentionsAnon(t, map[types.Type]bool{}) {
		return
	}
	v.reject(pos, "Go gives an unexported field the identity of its package, and "+
		"Dart holds such a name private to 1 library",
		"%s gives an unnamed struct type with an unexported field to another "+
			"package. Declare a named type, or export each field.",
		what)
}

// mentionsAnon reports whether another package can reach an unnamed struct type
// with an unexported field through this type.
//
// A named type stops the walk at its unexported fields. Another package cannot
// name such a field, thus it never writes the type of the field.
func mentionsAnon(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	switch t := types.Unalias(t).(type) {
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)
			if !f.Exported() || mentionsAnon(f.Type(), seen) {
				return true
			}
		}
		return false
	case *types.Named:
		for i := 0; i < t.TypeArgs().Len(); i++ {
			if mentionsAnon(t.TypeArgs().At(i), seen) {
				return true
			}
		}
		st, ok := t.Underlying().(*types.Struct)
		if !ok {
			return mentionsAnon(t.Underlying(), seen)
		}
		return exportedFieldAnon(st, seen)
	case *types.Pointer:
		return mentionsAnon(t.Elem(), seen)
	case *types.Slice:
		return mentionsAnon(t.Elem(), seen)
	case *types.Array:
		return mentionsAnon(t.Elem(), seen)
	case *types.Chan:
		return mentionsAnon(t.Elem(), seen)
	case *types.Map:
		return mentionsAnon(t.Key(), seen) || mentionsAnon(t.Elem(), seen)
	case *types.Signature:
		return mentionsAnon(t.Params(), seen) || mentionsAnon(t.Results(), seen)
	case *types.Tuple:
		for i := 0; i < t.Len(); i++ {
			if mentionsAnon(t.At(i).Type(), seen) {
				return true
			}
		}
	case *types.Interface:
		for i := 0; i < t.NumMethods(); i++ {
			if mentionsAnon(t.Method(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

// exportedFieldAnon walks the fields that another package can name.
func exportedFieldAnon(st *types.Struct, seen map[types.Type]bool) bool {
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}
		if mentionsAnon(f.Type(), seen) {
			return true
		}
	}
	return false
}

// checkParamStruct rejects a struct type that reads a type parameter without
// the name of a package level generic type. The class of such a shape stands
// at package level, and that class cannot name the parameter.
func (v *verifier) checkParamStruct(n *ast.StructType) {
	if n == v.genericStruct || v.typeTargets[n] {
		return
	}
	tv, ok := v.info.Types[n]
	if !ok {
		return
	}
	st, ok := tv.Type.(*types.Struct)
	if !ok {
		return
	}
	for i := 0; i < st.NumFields(); i++ {
		if typeset.HoldsParam(st.Field(i).Type()) {
			v.reject(n.Pos(), "the class of a struct shape stands at package level",
				"the field %s reads a type parameter, and only a package level generic type holds one",
				st.Field(i).Name())
			return
		}
	}
}


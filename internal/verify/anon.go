package verify

import (
	"go/ast"
	"go/token"
	"go/types"
)

// checkExported rejects an unnamed struct type in an exported declaration.
//
// Go gives an unnamed struct type a structural identity, and a Dart class holds
// a nominal identity. The compiler writes 1 class into each package that uses
// such a type, thus 2 packages give 2 classes that Dart holds apart. A named
// type has 1 class in 1 library, and it crosses a package boundary.
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
	v.reject(pos, "a Dart class holds a nominal identity, and an unnamed struct "+
		"type holds a structural identity",
		"%s gives an unnamed struct type to another package. Declare a named type.",
		what)
}

// mentionsAnon reports whether another package can reach an unnamed struct
// type through this type.
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
		return true
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

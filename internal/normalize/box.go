package normalize

import (
	"go/ast"
	"go/token"
	"go/types"
)

// addressed collects each variable whose address the code takes.
//
// The Dart form of a pointer to a struct is the object of the struct, and
// another pointer needs a box. A variable of this set holds a box, thus each
// read of it reads the field of the box.
func (b *builder) addressed(body *ast.BlockStmt) map[*types.Var]bool {
	out := map[*types.Var]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		u, ok := n.(*ast.UnaryExpr)
		if !ok || u.Op != token.AND {
			return true
		}
		id, ok := unparenExpr(u.X).(*ast.Ident)
		if !ok {
			return true
		}
		v, ok := b.info.Uses[id].(*types.Var)
		if !ok {
			v, ok = b.info.Defs[id].(*types.Var)
		}
		if !ok || !needsBox(v.Type()) {
			return true
		}
		out[v] = true
		return true
	})
	return out
}

// needsBox reports whether the address of a value of the type needs a box.
//
// A struct and an array give a Dart object, and such an object already holds a
// reference. Each other type needs a box.
func needsBox(t types.Type) bool {
	switch types.Unalias(t).Underlying().(type) {
	case *types.Struct, *types.Array:
		return false
	}
	return true
}

// boxed reports whether the variable holds a box.
func (b *builder) boxed(obj types.Object) bool {
	v, ok := obj.(*types.Var)
	return ok && b.boxes[v]
}

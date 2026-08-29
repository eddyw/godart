package normalize

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/eddyw/godart/internal/ir"
)

// addressed collects each variable whose address the code takes.
//
// The Dart form of a pointer to a struct is the object of the struct, and
// another pointer needs a box. A variable of this set holds a box, thus each
// read of it reads the field of the box.
func (b *builder) addressed(body ast.Node) map[*types.Var]bool {
	out := map[*types.Var]bool{}
	b.markAddressed(body, out, false)
	return out
}

// markAddressed collects each variable of the node whose address the code
// takes. The global flag selects a package variable, and it drops a local.
func (b *builder) markAddressed(node ast.Node, out map[*types.Var]bool, global bool) {
	ast.Inspect(node, func(n ast.Node) bool {
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
		if !ok || !needsBox(v.Type()) || isGlobalVar(v) != global {
			return true
		}
		out[v] = true
		return true
	})
}

// isGlobalVar reports whether the variable stands at package level.
func isGlobalVar(v *types.Var) bool {
	return v.Pkg() != nil && v.Parent() == v.Pkg().Scope()
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

// captured collects each variable of a machine that a function literal reads.
//
// A machine writes each local into its frame, and a resume reads them into a
// new call. A closure holds the variable of the first call, thus the 2 calls
// must share 1 box.
//
// A struct and an array need no box. An assignment to such a variable copies
// the fields into the same object, thus the closure reads the new value.
func (b *builder) captured(body ast.Node, assigned map[*types.Var]bool) map[*types.Var]bool {
	out := map[*types.Var]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		inner := map[*types.Var]bool{}
		ast.Inspect(lit, func(m ast.Node) bool {
			if id, ok := m.(*ast.Ident); ok {
				if v, ok := b.info.Defs[id].(*types.Var); ok {
					inner[v] = true
				}
			}
			return true
		})
		ast.Inspect(lit, func(m ast.Node) bool {
			id, ok := m.(*ast.Ident)
			if !ok {
				return true
			}
			v, ok := b.info.Uses[id].(*types.Var)
			if !ok || inner[v] || !needsBox(v.Type()) || !assigned[v] {
				return true
			}
			if v.Parent() != nil && v.Pkg() != nil && v.Parent() == v.Pkg().Scope() {
				return true
			}
			out[v] = true
			return true
		})
		return true
	})
	return out
}

// litCaptures gives each boxed variable of an outer scope that the literal
// reads. The order follows the source, thus 2 equal builds give 1 output.
func (b *builder) litCaptures(lit *ast.FuncLit) []*ir.Var {
	inner := map[*types.Var]bool{}
	ast.Inspect(lit, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			if v, ok := b.info.Defs[id].(*types.Var); ok {
				inner[v] = true
			}
		}
		return true
	})
	seen := map[*types.Var]bool{}
	var out []*ir.Var
	ast.Inspect(lit, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		v, ok := b.info.Uses[id].(*types.Var)
		if !ok || inner[v] || seen[v] || !b.boxes[v] {
			return true
		}
		if v.Parent() != nil && v.Pkg() != nil && v.Parent() == v.Pkg().Scope() {
			return true
		}
		seen[v] = true
		out = append(out, &ir.Var{Name: b.names[v], Obj: v, Type: v.Type(), Boxed: true})
		return true
	})
	return out
}

// machineBoxes adds the box of each captured variable when a function of the
// declaration can block.
func (b *builder) machineBoxes(body *ast.BlockStmt, blocking bool) {
	assigned := b.assignedVars(body)
	if blocking {
		for v := range b.captured(body, assigned) {
			b.boxes[v] = true
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok || !b.blocks.Blocks(lit) {
			return true
		}
		for v := range b.captured(lit.Body, assigned) {
			b.boxes[v] = true
		}
		return true
	})
}

// assignedVars collects each variable that the code writes or binds after its
// declaration. A variable that keeps its first value needs no box, because each
// call of a machine reads the same value.
//
// A declaration binds a new variable at each pass of a loop, thus the set holds
// the names of a short declaration and of a range clause.
func (b *builder) assignedVars(body ast.Node) map[*types.Var]bool {
	out := map[*types.Var]bool{}
	mark := func(e ast.Expr) {
		id, ok := unparenExpr(e).(*ast.Ident)
		if !ok {
			return
		}
		v, ok := b.info.Uses[id].(*types.Var)
		if !ok {
			v, ok = b.info.Defs[id].(*types.Var)
		}
		if ok {
			out[v] = true
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, l := range n.Lhs {
				mark(l)
			}
		case *ast.IncDecStmt:
			mark(n.X)
		case *ast.RangeStmt:
			mark(n.Key)
			mark(n.Value)
		case *ast.ValueSpec:
			if len(n.Values) > 0 {
				for _, name := range n.Names {
					mark(name)
				}
			}
		}
		return true
	})
	return out
}

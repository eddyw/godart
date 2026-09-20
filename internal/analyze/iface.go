package analyze

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

// The interface of a method group.
//
// A call through an interface reaches only a type that implements that
// interface. The key of a group thus holds the interface of the call site,
// and a concrete method joins the group only when its receiver implements
// that interface.
//
// Each answer of this file is safe in 1 direction. A member that the group
// keeps but a call cannot reach costs precision alone. A member that the group
// drops but a call can reach makes the check unsound, thus each test below
// gives true when the answer is not certain.

// methodSource gives the group of an interface method call. x is the receiver
// expression, recv is the receiver type, and fn is the method that the
// selection names.
func (a *analyzer) methodSource(x ast.Expr, recv types.Type, fn *types.Func) (source, bool) {
	if candidates, ok := a.concreteInterfaces(x, a.pkg.TypesInfo, map[types.Object]bool{}); ok {
		var keys []any
		complete := true
		for _, candidate := range candidates {
			if basic, ok := candidate.Underlying().(*types.Basic); ok && basic.Kind() == types.UntypedNil {
				continue
			}
			method, _, _ := types.LookupFieldOrMethod(candidate, false, fn.Pkg(), fn.Name())
			if method, ok := method.(*types.Func); ok && !abstractMethod(method) {
				keys = append(keys, method.Origin())
			} else {
				complete = false
			}
		}
		if complete {
			return source{keys: keys}, true
		}
	}
	msig, ok := fn.Type().(*types.Signature)
	if !ok {
		return source{}, false
	}
	c := methodCell{key: methodKey(fn.Name(), msig)}
	it := a.narrowed(x)
	if it == nil {
		it = callIface(recv, fn)
	}
	if it != nil {
		c.iface = interfaceKey(it)
	}
	if c.iface == "" {
		it = nil
	}
	a.methodCells[c] = it
	return source{cells: []cell{c}}, true
}

// narrowed gives the interface of a value that a type assertion produced. Such
// a value implements the asserted interface and the interface that it came
// from, thus the group holds a type that implements the 2.
func (a *analyzer) narrowed(x ast.Expr) *types.Interface {
	id, ok := unparen(x).(*ast.Ident)
	if !ok {
		return nil
	}
	v, ok := a.pkg.TypesInfo.Uses[id].(*types.Var)
	if !ok || a.assertDrop[v] {
		return nil
	}
	return a.assertNarrow[v]
}

// callIface gives the interface that a method call dispatches on. It gives nil
// when the analysis cannot name 1, and the group then holds each method of the
// name and the signature.
func callIface(recv types.Type, fn *types.Func) *types.Interface {
	if it := basicIface(recv); it != nil {
		return it
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return nil
	}
	return basicIface(sig.Recv().Type())
}

// basicIface gives the interface under a type. A constraint that holds a type
// term gives nil, because types.Implements gives no answer for such a type.
func basicIface(t types.Type) *types.Interface {
	it, ok := types.Unalias(t).Underlying().(*types.Interface)
	if !ok || !it.IsMethodSet() {
		return nil
	}
	return it
}

// interfaceKey names an interface by its complete method set. Go implements an
// interface structurally, thus 2 interfaces of the same methods take 1 key. An
// unexported name carries its package, because 2 packages can hold 1 such name
// and the 2 names are distinct.
func interfaceKey(it *types.Interface) string {
	n := it.NumMethods()
	if n == 0 {
		return ""
	}
	parts := make([]string, 0, n)
	for i := range n {
		m := it.Method(i)
		sig, ok := m.Type().(*types.Signature)
		if !ok {
			return ""
		}
		name := m.Name()
		if !m.Exported() && m.Pkg() != nil {
			name = m.Pkg().Path() + "." + name
		}
		parts = append(parts, methodKey(name, sig))
	}
	sort.Strings(parts)
	return strings.Join(parts, keySep)
}

// reachesIface reports whether a dispatch through the interface can reach the
// method.
func reachesIface(member any, it *types.Interface) bool {
	fn, ok := member.(*types.Func)
	if !ok {
		return true
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return true
	}
	recv := sig.Recv().Type()
	if !plainRecv(recv) {
		return true
	}
	if types.Implements(recv, it) {
		return true
	}
	if _, isPtr := types.Unalias(recv).(*types.Pointer); isPtr {
		return false
	}
	// A value receiver puts the method in the method set of the pointer too.
	return types.Implements(types.NewPointer(recv), it)
}

// plainRecv reports whether types.Implements answers for the receiver. A
// generic receiver holds a type parameter, and the method set of such a type
// depends on the type argument.
func plainRecv(t types.Type) bool {
	if p, ok := types.Unalias(t).(*types.Pointer); ok {
		t = p.Elem()
	}
	n, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	return n.Origin().TypeParams().Len() == 0
}

// collectAsserts reads 1 file before the main walk, and it keeps each local
// value that a type assertion to an interface produced. The main walk asks for
// the answer while it reads a call, thus the whole module runs through this
// pass first.
func (a *analyzer) collectAsserts(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.UnaryExpr:
			if n.Op == token.AND {
				a.dropAssert(n.X)
			}
		case *ast.IncDecStmt:
			a.dropAssert(n.X)
		case *ast.RangeStmt:
			a.dropAssert(n.Key)
			a.dropAssert(n.Value)
		case *ast.AssignStmt:
			a.assertAssign(n)
		case *ast.TypeSwitchStmt:
			a.assertSwitch(n)
		}
		return true
	})
}

// dropAssert marks a name that another statement can write, thus the narrow
// answer of the assertion no longer holds for it.
func (a *analyzer) dropAssert(x ast.Expr) {
	if x == nil {
		return
	}
	id, ok := unparen(x).(*ast.Ident)
	if !ok {
		return
	}
	if v, ok := a.pkg.TypesInfo.Uses[id].(*types.Var); ok {
		a.assertDrop[v] = true
	}
}

func (a *analyzer) assertAssign(n *ast.AssignStmt) {
	for i, lhs := range n.Lhs {
		id, ok := unparen(lhs).(*ast.Ident)
		if !ok {
			continue
		}
		v, _ := a.pkg.TypesInfo.Defs[id].(*types.Var)
		if v == nil {
			a.dropAssert(lhs)
			continue
		}
		if i != 0 || len(n.Rhs) != 1 {
			continue
		}
		ta, ok := unparen(n.Rhs[0]).(*ast.TypeAssertExpr)
		if !ok || ta.Type == nil {
			continue
		}
		a.keepNarrow(v, a.typeOf(ta.X), a.typeOf(ta.Type))
	}
}

// assertSwitch reads a type switch. Each clause of 1 interface type holds its
// own implicit name, and that name carries the narrow answer of the clause.
func (a *analyzer) assertSwitch(n *ast.TypeSwitchStmt) {
	var from types.Type
	switch s := n.Assign.(type) {
	case *ast.AssignStmt:
		if len(s.Rhs) != 1 {
			return
		}
		ta, ok := unparen(s.Rhs[0]).(*ast.TypeAssertExpr)
		if !ok {
			return
		}
		from = a.typeOf(ta.X)
	case *ast.ExprStmt:
		ta, ok := unparen(s.X).(*ast.TypeAssertExpr)
		if !ok {
			return
		}
		from = a.typeOf(ta.X)
	default:
		return
	}
	for _, stmt := range n.Body.List {
		clause, ok := stmt.(*ast.CaseClause)
		if !ok || len(clause.List) != 1 {
			continue
		}
		v, _ := a.pkg.TypesInfo.Implicits[clause].(*types.Var)
		if v == nil {
			continue
		}
		a.keepNarrow(v, from, a.typeOf(clause.List[0]))
	}
}

// keepNarrow keeps the interface of a value that an assertion from the type
// from to the type to produced.
func (a *analyzer) keepNarrow(v *types.Var, from, to types.Type) {
	fromIface := basicIface(from)
	toIface := basicIface(to)
	if fromIface == nil || toIface == nil {
		return
	}
	if it := unionIface(fromIface, toIface); it != nil {
		a.assertNarrow[v] = it
	}
}

// unionIface builds the interface that holds each method of the 2 interfaces.
// A value that an assertion gives implements the 2, thus it implements this
// interface. It gives nil when 1 name carries 2 signatures, because no type
// implements such a pair and the analysis then keeps the wide group.
func unionIface(x, y *types.Interface) *types.Interface {
	byName := map[string]*types.Signature{}
	out := make([]*types.Func, 0, x.NumMethods()+y.NumMethods())
	for _, it := range [2]*types.Interface{x, y} {
		for i := range it.NumMethods() {
			m := it.Method(i)
			sig, ok := m.Type().(*types.Signature)
			if !ok {
				return nil
			}
			name := m.Name()
			if m.Pkg() != nil && !m.Exported() {
				name = m.Pkg().Path() + "." + name
			}
			if old, seen := byName[name]; seen {
				if !types.Identical(old, sig) {
					return nil
				}
				continue
			}
			byName[name] = sig
			// A fresh function keeps the receiver of the 2 inputs untouched,
			// because NewInterfaceType takes ownership of each method.
			out = append(out, types.NewFunc(token.NoPos, m.Pkg(), m.Name(),
				types.NewSignatureType(nil, nil, nil, sig.Params(), sig.Results(), sig.Variadic())))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return types.NewInterfaceType(out, nil).Complete()
}

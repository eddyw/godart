package analyze

import (
	"go/ast"
	"go/types"
	"strings"
)

// The value flow of stage 3.
//
// A call through a function value must wait when a function that can reach
// that value blocks. The flow follows each assignment, each argument, each
// result, and each struct field, thus the set of a call site holds only the
// functions that can reach it. A value that leaves this net falls into the
// escape group of its signature, and a read outside the net draws from that
// group. The group gives the old behavior of the signature key, thus a
// construct outside the net stays sound.

// A cell names a location that can hold a function value. The kinds:
//
//   - a *types.Var: a variable, a parameter, a result, or a struct field
//   - a retCell: 1 result of a function
//   - an outCell: 1 result of a call expression
//   - a sigCell: the escape group of a signature
//   - a methodCell: the group of an interface method
type cell any

// retCell names 1 result of a function. The key is the origin of the function.
type retCell struct {
	fn any
	i  int
}

// outCell names 1 result of a call expression.
type outCell struct {
	call *ast.CallExpr
	i    int
}

// sigCell names the escape group of a signature.
type sigCell struct{ key string }

// methodCell names the group of an interface method. A method of the name and
// the signature is a member when its receiver implements iface, because a
// dispatch through that interface can then reach it. An empty iface means that
// the analysis named no interface, and each method of the group is a member.
// Read iface.go.
type methodCell struct {
	key   string
	iface string
}

// methodExprKey marks a target that a method expression built. A call of such
// a value takes the receiver as its first argument, thus the wire shifts each
// argument by 1.
type methodExprKey struct{ fn *types.Func }

// A source names where a function value comes from: fixed targets, cells, or
// both.
type source struct {
	keys  []any
	cells []cell
}

func (s source) empty() bool { return len(s.keys) == 0 && len(s.cells) == 0 }

// callSite holds 1 call that the solver wires to its targets. The wire flows
// each argument into the parameters of a target, and each result of the
// target into the results of the call.
type callSite struct {
	onceVariable   *types.Var
	onceOperands   bool
	onceTargets    []any
	onceFallbacks  []string
	formatSource   source
	formatOperands bool
	formatMethods  []any
	summary        *summary
	call           *ast.CallExpr
	// rng marks the call of a range loop over a function. The loop holds no
	// ast.CallExpr, thus the async state lands in rangeAsync.
	rng    *ast.RangeStmt
	callee source
	// args holds the source of each function typed argument, by position.
	args map[int]source
	// argEsc holds the escape key of each such argument. An argument flows
	// there when the target holds no cell at its position.
	argEsc map[int]string
	// done marks the targets that the wire already covered.
	done map[any]bool

	// The answer of the flow, after the solve. targets holds the block key of
	// each function that the site can reach, and fallbacks holds each escape
	// group that the site reads.
	targets   []any
	fallbacks []string
}

// funcCell gives the canonical cell of a variable. An instance of a generic
// declaration holds a variable of its own, and Origin joins the instances.
func funcCell(v *types.Var) cell { return v.Origin() }

// addTarget puts 1 function into the set of a cell.
func (a *analyzer) addTarget(c cell, key any) {
	set := a.pts[c]
	if set == nil {
		set = map[any]bool{}
		a.pts[c] = set
	}
	if set[key] {
		return
	}
	set[key] = true
	a.dirty(c)
}

// addEdge lets the set of src flow into the set of dst.
func (a *analyzer) addEdge(src, dst cell) {
	if src == dst {
		return
	}
	set := a.edges[src]
	if set == nil {
		set = map[cell]bool{}
		a.edges[src] = set
	}
	if set[dst] {
		return
	}
	set[dst] = true
	if len(a.pts[src]) > 0 {
		a.dirty(src)
	}
}

func (a *analyzer) dirty(c cell) {
	if a.inWork[c] {
		return
	}
	a.inWork[c] = true
	a.work = append(a.work, c)
}

// flowTo wires a source into a cell.
func (a *analyzer) flowTo(dst cell, src source) {
	for _, k := range src.keys {
		a.addTarget(dst, k)
	}
	for _, c := range src.cells {
		a.addEdge(c, dst)
	}
}

// escape drops a function value into the escape group of its signature.
func (a *analyzer) escape(e ast.Expr) {
	sig := funcSig(a.typeOf(e))
	if sig == nil {
		return
	}
	a.flowTo(sigCell{signatureKey(sig)}, a.flowExpr(e))
}

// funcSig gives the signature under a type, or nil.
func funcSig(t types.Type) *types.Signature {
	if t == nil {
		return nil
	}
	sig, _ := types.Unalias(t).Underlying().(*types.Signature)
	return sig
}

// assign wires the flow of 1 assignment. A destination that holds no function
// type takes the value as an interface value, thus the value escapes. A
// destination outside the net, for example an element of a slice, does the
// same.
func (a *analyzer) assign(dst ast.Expr, src source, srcType types.Type) {
	if funcSig(srcType) == nil {
		return
	}
	c, blank := a.lhsCell(dst)
	if blank {
		return
	}
	if c == nil {
		a.flowTo(sigCell{signatureKey(funcSig(srcType))}, src)
		return
	}
	a.flowTo(c, src)
}

// lhsCell gives the cell of the left side of an assignment. A location that
// the flow does not track gives nil, and the value then escapes. A blank name
// drops the value.
func (a *analyzer) lhsCell(e ast.Expr) (c cell, blank bool) {
	e = unparen(e)
	switch e := e.(type) {
	case *ast.Ident:
		if e.Name == "_" {
			return nil, true
		}
		if v, ok := a.pkg.TypesInfo.Defs[e].(*types.Var); ok {
			return funcCell(v), false
		}
		if v, ok := a.pkg.TypesInfo.Uses[e].(*types.Var); ok {
			return funcCell(v), false
		}
	case *ast.SelectorExpr:
		if sel, ok := a.pkg.TypesInfo.Selections[e]; ok {
			if sel.Kind() == types.FieldVal {
				if v, ok := sel.Obj().(*types.Var); ok {
					return funcCell(v), false
				}
			}
			return nil, false
		}
		if v, ok := a.pkg.TypesInfo.Uses[e.Sel].(*types.Var); ok {
			return funcCell(v), false
		}
	}
	return nil, false
}

// flowExpr gives the source of a function typed expression. An expression
// outside the net draws from the escape group of its signature.
func (a *analyzer) flowExpr(e ast.Expr) source {
	e = unparen(e)
	switch e := e.(type) {
	case *ast.Ident:
		switch obj := a.pkg.TypesInfo.Uses[e].(type) {
		case *types.Func:
			return source{keys: []any{obj.Origin()}}
		case *types.Var:
			return source{cells: []cell{funcCell(obj)}}
		}
	case *ast.FuncLit:
		return source{keys: []any{e}}
	case *ast.SelectorExpr:
		sel, ok := a.pkg.TypesInfo.Selections[e]
		if !ok {
			// A qualified name of another package.
			switch obj := a.pkg.TypesInfo.Uses[e.Sel].(type) {
			case *types.Func:
				return source{keys: []any{obj.Origin()}}
			case *types.Var:
				return source{cells: []cell{funcCell(obj)}}
			}
			break
		}
		switch sel.Kind() {
		case types.FieldVal:
			if v, ok := sel.Obj().(*types.Var); ok {
				return source{cells: []cell{funcCell(v)}}
			}
		case types.MethodVal, types.MethodExpr:
			fn, ok := sel.Obj().(*types.Func)
			if !ok {
				break
			}
			if isInterface(sel.Recv()) {
				if src, ok := a.methodSource(e.X, sel.Recv(), fn); ok {
					return src
				}
				break
			}
			if sel.Kind() == types.MethodExpr {
				if sig, ok := fn.Type().(*types.Signature); ok &&
					sig.Recv() != nil && funcSig(sig.Recv().Type()) != nil {
					// A method on a named function type takes a function
					// value as its receiver, and the shift of the wire cannot
					// place it. Such a value stays in the escape group.
					break
				}
				return source{keys: []any{methodExprKey{fn.Origin()}}}
			}
			return source{keys: []any{fn.Origin()}}
		}
	case *ast.IndexExpr:
		if s := a.instanceSource(e.X); !s.empty() {
			return s
		}
	case *ast.IndexListExpr:
		if s := a.instanceSource(e.X); !s.empty() {
			return s
		}
	case *ast.CallExpr:
		if tv, ok := a.pkg.TypesInfo.Types[e.Fun]; ok && tv.IsType() {
			// A conversion keeps the value.
			if len(e.Args) == 1 {
				return a.flowExpr(e.Args[0])
			}
		}
		return source{cells: []cell{outCell{e, 0}}}
	}
	sig := funcSig(a.typeOf(e))
	if sig == nil {
		return source{}
	}
	return source{cells: []cell{sigCell{signatureKey(sig)}}}
}

// instanceSource gives the source of a generic function that an index
// expression instantiates. An index of a slice or of a map gives an empty
// source, and the caller then falls back to the escape group.
func (a *analyzer) instanceSource(x ast.Expr) source {
	switch x := unparen(x).(type) {
	case *ast.Ident:
		if fn, ok := a.pkg.TypesInfo.Uses[x].(*types.Func); ok {
			return source{keys: []any{fn.Origin()}}
		}
	case *ast.SelectorExpr:
		if fn, ok := a.pkg.TypesInfo.Uses[x.Sel].(*types.Func); ok {
			return source{keys: []any{fn.Origin()}}
		}
	}
	return source{}
}

// solve runs the flow to a fixed point. The wire of a call site adds new
// edges, thus the 2 parts repeat together until no set grows.
func (a *analyzer) solve() {
	for c, it := range a.methodCells {
		name, signature, _ := strings.Cut(c.key, keySep)
		keys := a.byGroup[a.find(signature)]
		if len(keys) == 0 {
			keys = []string{signature}
		}
		generic := strings.Contains(c.iface, wildcard)
		for _, key := range keys {
			for _, member := range a.byMethod[name+keySep+key] {
				if it == nil || generic || reachesIface(member, it) {
					a.addTarget(c, member)
				}
			}
		}
	}
	for _, site := range a.sites {
		for _, k := range site.callee.keys {
			a.wire(site, k)
		}
		for _, c := range site.callee.cells {
			a.watchers[c] = append(a.watchers[c], site)
			for k := range a.pts[c] {
				a.wire(site, k)
			}
		}
	}
	for len(a.work) > 0 {
		c := a.work[len(a.work)-1]
		a.work = a.work[:len(a.work)-1]
		a.inWork[c] = false
		for dst := range a.edges[c] {
			grew := false
			set := a.pts[dst]
			if set == nil {
				set = map[any]bool{}
				a.pts[dst] = set
			}
			for k := range a.pts[c] {
				if !set[k] {
					set[k] = true
					grew = true
				}
			}
			if grew {
				a.dirty(dst)
			}
		}
		for _, site := range a.watchers[c] {
			for k := range a.pts[c] {
				a.wire(site, k)
			}
		}
	}
}

// wire joins 1 call site with 1 target. Each function typed argument flows
// into the parameter of the target, and each result of the target flows into
// the result of the call.
func (a *analyzer) wire(site *callSite, key any) {
	if site.done[key] {
		return
	}
	site.done[key] = true
	shift := 0
	if _, ok := key.(methodExprKey); ok {
		shift = 1
	}
	params, results := a.targetShape(key)
	for i, src := range site.args {
		j := i - shift
		if j < 0 {
			// The first argument of a method expression is the receiver, and
			// a function typed receiver never reaches a wire.
			continue
		}
		if params == nil || j >= len(params) || params[j] == nil {
			if esc := site.argEsc[i]; esc != "" {
				a.flowTo(sigCell{esc}, src)
			}
			continue
		}
		a.flowTo(params[j], src)
	}
	if site.call != nil {
		for i, r := range results {
			if r != nil {
				a.addEdge(r, outCell{site.call, i})
			}
		}
	}
}

// targetShape gives the parameter cells and the result cells of a target. The
// parameter list of a range loop body holds the key variable and the value
// variable of the loop.
func (a *analyzer) targetShape(key any) (params, results []cell) {
	switch key := key.(type) {
	case *types.Func:
		sig, ok := key.Type().(*types.Signature)
		if !ok {
			return nil, nil
		}
		return a.sigShape(key, sig)
	case methodExprKey:
		sig, ok := key.fn.Type().(*types.Signature)
		if !ok {
			return nil, nil
		}
		return a.sigShape(key.fn, sig)
	case *ast.FuncLit:
		sig := a.litSigs[key]
		if sig == nil {
			return nil, nil
		}
		return a.sigShape(key, sig)
	case *ast.RangeStmt:
		return a.rangeShape[key], nil
	}
	return nil, nil
}

func (a *analyzer) sigShape(key any, sig *types.Signature) (params, results []cell) {
	params = make([]cell, sig.Params().Len())
	for i := range params {
		if funcSig(sig.Params().At(i).Type()) != nil {
			params[i] = funcCell(sig.Params().At(i))
		}
	}
	results = make([]cell, sig.Results().Len())
	for i := range results {
		if funcSig(sig.Results().At(i).Type()) != nil {
			results[i] = retCell{key, i}
		}
	}
	return params, results
}

// targets gives the functions that a source can hold after the solve.
func (a *analyzer) targets(src source) []any {
	seen := map[any]bool{}
	out := []any{}
	for _, k := range src.keys {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	for _, c := range src.cells {
		for k := range a.pts[c] {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}

// sigFallbacks gives the escape keys that a source reads. A member that joins
// the group after this call site still counts, thus the keys stay in the
// answer of the site.
func sigFallbacks(src source) []string {
	out := []string{}
	for _, c := range src.cells {
		if s, ok := c.(sigCell); ok && s.key != "" {
			out = append(out, s.key)
		}
	}
	return out
}

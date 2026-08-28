// Package analyze runs stage 3 of the compiler. It finds each function that can
// block, and it marks each call that must wait for a result.
//
// All Go code runs in 1 Dart isolate, and goroutines are cooperative. A
// function that can block compiles to a Dart async function, and a call to such
// a function compiles to an await expression.
package analyze

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Result holds the block state of each function and of each call.
type Result struct {
	// blocking marks a function that can block. A key is a *types.Func or a
	// *ast.FuncLit.
	blocking map[any]bool
	// callAsync marks a call that waits for a result.
	callAsync map[*ast.CallExpr]bool
	// sigAsync marks the signature of a function value that can block. Each
	// function value of the signature then takes the async form.
	sigAsync map[string]bool
	// methodAsync marks a method name with a signature that can block.
	methodAsync map[string]bool
}

func (r *Result) Blocks(key any) bool { return r.blocking[key] }

func (r *Result) CallAsync(call *ast.CallExpr) bool { return r.callAsync[call] }

// SignatureAsync reports whether a function value of this signature can block.
func (r *Result) SignatureAsync(sig *types.Signature) bool {
	return r.sigAsync[signatureKey(sig)]
}

// summary holds what 1 function does. The fixed point reads it.
type summary struct {
	key     any
	direct  bool
	callees []any
	sigs    []string
	methods []string
	calls   []*ast.CallExpr
	// callKeys names the target of each call, in the order of calls.
	callKeys [][]any
	callSigs []string
}

// Module computes the block state over each package of a module.
func Module(pkgs []*packages.Package) *Result {
	a := &analyzer{
		result: &Result{
			blocking:    map[any]bool{},
			callAsync:   map[*ast.CallExpr]bool{},
			sigAsync:    map[string]bool{},
			methodAsync: map[string]bool{},
		},
		summaries:  map[any]*summary{},
		bySig:      map[string][]any{},
		byMethod:   map[string][]any{},
		methodKeys: map[any]string{},
		sigKeys:    map[any]string{},
		callFuns:   map[ast.Expr]bool{},
	}
	for _, p := range pkgs {
		a.pkg = p
		for _, f := range p.Syntax {
			ast.Inspect(f, a.collect)
		}
	}
	a.fixedPoint()
	a.markCalls()
	return a.result
}

type analyzer struct {
	pkg       *packages.Package
	result    *Result
	summaries map[any]*summary
	order     []any

	// bySig groups each function value by its signature, and byMethod groups
	// each method by its name and its signature. A call through a value or
	// through an interface can reach each member of the group.
	bySig      map[string][]any
	byMethod   map[string][]any
	methodKeys map[any]string
	sigKeys    map[any]string
	// callFuns holds the callee expression of each call, thus a name that
	// stands alone reads as a function value.
	callFuns map[ast.Expr]bool
}

func (a *analyzer) collect(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.FuncDecl:
		obj, _ := a.pkg.TypesInfo.Defs[n.Name].(*types.Func)
		if obj == nil || n.Body == nil {
			return true
		}
		a.walkBody(obj, n.Body)
		if sig, ok := obj.Type().(*types.Signature); ok && sig.Recv() != nil {
			key := obj.Name() + "|" + signatureKey(sig)
			a.byMethod[key] = append(a.byMethod[key], any(obj))
			a.methodKeys[any(obj)] = key
		}
		return true
	case *ast.FuncLit:
		a.walkBody(n, n.Body)
		if sig, ok := a.pkg.TypesInfo.Types[n].Type.(*types.Signature); ok {
			a.addValue(any(n), sig)
		}
		return true
	case *ast.CallExpr:
		a.callFuns[unparen(n.Fun)] = true
		return true
	case *ast.Ident:
		if a.callFuns[n] {
			return true
		}
		if obj, ok := a.pkg.TypesInfo.Uses[n].(*types.Func); ok {
			if sig, ok := obj.Type().(*types.Signature); ok && sig.Recv() == nil {
				a.addValue(any(obj), sig)
			}
		}
		return true
	}
	return true
}

// addValue records a function that the code uses as a value. A call through
// such a value can reach each function of the same signature.
func (a *analyzer) addValue(key any, sig *types.Signature) {
	name := signatureKey(sig)
	for _, member := range a.bySig[name] {
		if member == key {
			return
		}
	}
	a.bySig[name] = append(a.bySig[name], key)
	a.sigKeys[key] = name
}

// walkBody reads 1 function body. It stops at a nested function literal,
// because that literal gets its own summary.
func (a *analyzer) walkBody(key any, body ast.Node) {
	s := &summary{key: key}
	a.summaries[key] = s
	a.order = append(a.order, key)
	var walk func(n ast.Node) bool
	walk = func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			if n.Body != body {
				return false
			}
		case *ast.SendStmt:
			s.direct = true
		case *ast.SelectStmt:
			s.direct = true
		case *ast.UnaryExpr:
			if n.Op.String() == "<-" {
				s.direct = true
			}
		case *ast.RangeStmt:
			if _, ok := types.Unalias(a.typeOf(n.X)).Underlying().(*types.Chan); ok {
				s.direct = true
			}
		case *ast.CallExpr:
			a.callTargets(s, n)
		}
		return true
	}
	ast.Inspect(body, func(n ast.Node) bool {
		if n == body {
			return true
		}
		if lit, ok := n.(*ast.FuncLit); ok {
			_ = lit
			return false
		}
		return walk(n)
	})
}

func (a *analyzer) typeOf(e ast.Expr) types.Type {
	if tv, ok := a.pkg.TypesInfo.Types[e]; ok {
		return tv.Type
	}
	return types.Typ[types.Invalid]
}

// callTargets names the functions that a call can reach.
func (a *analyzer) callTargets(s *summary, call *ast.CallExpr) {
	fun := unparen(call.Fun)
	if tv, ok := a.pkg.TypesInfo.Types[fun]; ok && tv.IsType() {
		return
	}
	var keys []any
	var sigKey string
	switch fun := fun.(type) {
	case *ast.Ident:
		obj := a.pkg.TypesInfo.Uses[fun]
		switch obj := obj.(type) {
		case *types.Func:
			if blockingIntrinsic(obj) {
				a.markIntrinsic(s, call)
				return
			}
			keys = append(keys, any(obj))
		case *types.Builtin:
			return
		case *types.Var:
			sig, _ := types.Unalias(obj.Type()).Underlying().(*types.Signature)
			if sig != nil {
				sigKey = signatureKey(sig)
			}
		}
	case *ast.SelectorExpr:
		sel, ok := a.pkg.TypesInfo.Selections[fun]
		if !ok {
			if obj, ok := a.pkg.TypesInfo.Uses[fun.Sel].(*types.Func); ok {
				if blockingIntrinsic(obj) {
					a.markIntrinsic(s, call)
					return
				}
				keys = append(keys, any(obj))
			}
			break
		}
		switch sel.Kind() {
		case types.MethodVal:
			if isInterface(sel.Recv()) {
				fn, _ := sel.Obj().(*types.Func)
				if sig, ok := fn.Type().(*types.Signature); ok {
					s.methods = append(s.methods, fn.Name()+"|"+signatureKey(sig))
					s.calls = append(s.calls, call)
					s.callKeys = append(s.callKeys, nil)
					s.callSigs = append(s.callSigs, fn.Name()+"|"+signatureKey(sig))
					return
				}
			}
			if fn, ok := sel.Obj().(*types.Func); ok {
				keys = append(keys, any(fn))
			}
		case types.FieldVal:
			sig, _ := types.Unalias(sel.Type()).Underlying().(*types.Signature)
			if sig != nil {
				sigKey = signatureKey(sig)
			}
		}
	default:
		sig, _ := types.Unalias(a.typeOf(fun)).Underlying().(*types.Signature)
		if sig != nil {
			sigKey = signatureKey(sig)
		}
	}
	s.callees = append(s.callees, keys...)
	if sigKey != "" {
		s.sigs = append(s.sigs, sigKey)
	}
	s.calls = append(s.calls, call)
	s.callKeys = append(s.callKeys, keys)
	s.callSigs = append(s.callSigs, sigKey)
}

// intrinsicKey marks a call to an interop function that gives up control.
const intrinsicKey = "$intrinsic"

// markIntrinsic records a call that the compiler replaces with a call to the
// runtime library. Such a call waits for a result, thus the caller can block.
func (a *analyzer) markIntrinsic(s *summary, call *ast.CallExpr) {
	s.direct = true
	s.calls = append(s.calls, call)
	s.callKeys = append(s.callKeys, nil)
	s.callSigs = append(s.callSigs, intrinsicKey)
	a.result.sigAsync[intrinsicKey] = true
}

// blockingIntrinsic reports whether an interop function gives up control. The
// body of such a function holds no block operation, thus the analysis needs
// this list.
func blockingIntrinsic(fn *types.Func) bool {
	if fn.Pkg() == nil || fn.Pkg().Path() != "syscall/dart" {
		return false
	}
	switch fn.Name() {
	case "Sleep", "Worker", "Catch":
		return true
	}
	return false
}

func isInterface(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Interface)
	return ok
}

// fixedPoint marks each function that can block. It repeats until no state
// changes, because a call graph holds cycles.
func (a *analyzer) fixedPoint() {
	for _, key := range a.order {
		if a.summaries[key].direct {
			a.result.blocking[key] = true
		}
	}
	for {
		changed := false
		for _, key := range a.order {
			if a.result.blocking[key] {
				continue
			}
			if a.blocksNow(a.summaries[key]) {
				a.result.blocking[key] = true
				changed = true
			}
		}
		if a.spread() {
			changed = true
		}
		if !changed {
			return
		}
	}
}

func (a *analyzer) blocksNow(s *summary) bool {
	for _, c := range s.callees {
		if a.result.blocking[c] {
			return true
		}
	}
	for _, sig := range s.sigs {
		if a.result.sigAsync[sig] {
			return true
		}
	}
	for _, m := range s.methods {
		if a.result.methodAsync[m] {
			return true
		}
	}
	return false
}

// spread gives the same state to each member of a group. A call through a
// function value or through an interface reaches each member, thus each member
// takes the async form.
func (a *analyzer) spread() bool {
	changed := false
	for key, group := range a.bySig {
		if a.result.sigAsync[key] {
			continue
		}
		for _, member := range group {
			if a.result.blocking[member] {
				a.result.sigAsync[key] = true
				changed = true
				break
			}
		}
	}
	for key, group := range a.byMethod {
		if a.result.methodAsync[key] {
			continue
		}
		for _, member := range group {
			if a.result.blocking[member] {
				a.result.methodAsync[key] = true
				changed = true
				break
			}
		}
	}
	return changed
}

// markCalls marks each call that must wait for a result.
func (a *analyzer) markCalls() {
	keys := make([]any, 0, len(a.summaries))
	keys = append(keys, a.order...)
	for _, key := range keys {
		s := a.summaries[key]
		for i, call := range s.calls {
			async := false
			for _, target := range s.callKeys[i] {
				async = async || a.result.blocking[target]
			}
			if sig := s.callSigs[i]; sig != "" {
				async = async || a.result.sigAsync[sig] || a.result.methodAsync[sig]
			}
			if async {
				a.result.callAsync[call] = true
			}
		}
	}
}

// signatureKey builds a stable text of a signature. The receiver stays out,
// because a method group needs 1 key for each implementation.
func signatureKey(sig *types.Signature) string {
	if sig == nil {
		return ""
	}
	out := "("
	for i := 0; i < sig.Params().Len(); i++ {
		if i > 0 {
			out += ","
		}
		out += types.TypeString(sig.Params().At(i).Type(), nil)
	}
	out += ")("
	for i := 0; i < sig.Results().Len(); i++ {
		if i > 0 {
			out += ","
		}
		out += types.TypeString(sig.Results().At(i).Type(), nil)
	}
	if sig.Variadic() {
		out += ",..."
	}
	return out + ")"
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

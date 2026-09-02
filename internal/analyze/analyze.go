// Package analyze runs stage 3 of the compiler. It finds each function that can
// block, and it marks each call that must wait for a result.
//
// All Go code runs in 1 Dart isolate, and goroutines are cooperative. A
// function that can block compiles to a state machine, and a call to such a
// function compiles to a wait for the result.
//
// A call through a function value reaches only the functions that flow to that
// value. The file flow.go holds the value flow.
package analyze

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/eddyw/godart/internal/typeset"
	"golang.org/x/tools/go/packages"
)

// Result holds the block state of each function and of each call.
type Result struct {
	// blocking marks a function that can block. A key is a *types.Func, a
	// *ast.FuncLit, or the *ast.RangeStmt of a range loop over a function,
	// whose body is a function of its own.
	blocking map[any]bool
	// callAsync marks a call that waits for a result.
	callAsync map[*ast.CallExpr]bool
	// rangeAsync marks a range loop over a function whose call of the
	// sequence waits for a result.
	rangeAsync map[*ast.RangeStmt]bool
	// sigAsync marks the escape group of a signature. A call through a value
	// that the flow does not track waits when its group holds a function that
	// blocks.
	sigAsync map[string]bool
	// syncErrs holds each rejection of the synchronous check. Read sync.go.
	syncErrs []*SyncError
}

func (r *Result) Blocks(key any) bool { return r.blocking[key] }

func (r *Result) CallAsync(call *ast.CallExpr) bool { return r.callAsync[call] }

// RangeAsync reports whether the call of the sequence of a range loop over a
// function waits for a result.
func (r *Result) RangeAsync(s *ast.RangeStmt) bool { return r.rangeAsync[s] }

// Digest gives a stable text of each decision that the emitter of 1 package
// reads.
//
// The cache key of a package holds this text. The analysis reads the whole
// module, thus a new function of a package that this package does not import
// can change the form of a call through a function value.
func (r *Result) Digest(p *packages.Package) string {
	h := sha256.New()
	for _, f := range p.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				fmt.Fprintf(h, "d%t\x00", r.blocking[any(p.TypesInfo.Defs[n.Name])])
			case *ast.FuncLit:
				fmt.Fprintf(h, "l%t\x00", r.blocking[any(n)])
			case *ast.CallExpr:
				fmt.Fprintf(h, "c%t\x00", r.callAsync[n])
			case *ast.RangeStmt:
				fmt.Fprintf(h, "r%t%t\x00", r.blocking[any(n)], r.rangeAsync[n])
			}
			return true
		})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// summary holds what 1 function does. The fixed point reads it.
type summary struct {
	key    any
	direct bool
	// directPos and directWhat name the first operation of the body that
	// blocks. The synchronous check of sync.go reports the pair, thus the
	// error names a position inside the function and not only its declaration.
	directPos  token.Pos
	directWhat string
	sites      []*callSite
	// intrinsics holds each call of an interop function that gives up
	// control. Such a call always waits.
	intrinsics []*ast.CallExpr
}

// blocksAt marks the summary and keeps the first operation that blocks.
func (s *summary) blocksAt(pos token.Pos, what string) {
	s.direct = true
	if !s.directPos.IsValid() {
		s.directPos = pos
		s.directWhat = what
	}
}

// Module computes the block state over each package of a module.
func Module(pkgs []*packages.Package) *Result {
	a := &analyzer{
		result: &Result{
			blocking:   map[any]bool{},
			callAsync:  map[*ast.CallExpr]bool{},
			rangeAsync: map[*ast.RangeStmt]bool{},
			sigAsync:   map[string]bool{},
		},
		summaries:    map[any]*summary{},
		byMethod:     map[string][]any{},
		methodCells:  map[methodCell]*types.Interface{},
		assertNarrow: map[*types.Var]*types.Interface{},
		assertDrop:   map[*types.Var]bool{},
		parent:       map[string]string{},
		pts:          map[cell]map[any]bool{},
		edges:        map[cell]map[cell]bool{},
		inWork:       map[cell]bool{},
		watchers:     map[cell][]*callSite{},
		litSigs:      map[*ast.FuncLit]*types.Signature{},
		rangeShape:   map[*ast.RangeStmt][]cell{},
	}
	for _, p := range pkgs {
		a.pkg = p
		for _, f := range p.Syntax {
			a.collectAsserts(f)
		}
	}
	for _, p := range pkgs {
		a.pkg = p
		if a.fset == nil {
			a.fset = p.Fset
		}
		for _, f := range p.Syntax {
			a.file(f)
		}
	}
	a.components()
	a.solve()
	a.resolveSites()
	a.fixedPoint()
	a.markCalls()
	a.checkSync()
	return a.result
}

type analyzer struct {
	pkg       *packages.Package
	fset      *token.FileSet
	result    *Result
	summaries map[any]*summary
	order     []any

	// syncTypes holds each named function type that carries //gart:sync. Read
	// sync.go.
	syncTypes map[*types.TypeName]bool

	// byMethod groups each method by its name and its signature. A call
	// through an interface reads the group of its own key. Read iface.go.
	byMethod map[string][]any
	// methodCells holds each group that a call site reads, with the interface
	// of that site. A nil interface means that the group holds each member.
	methodCells map[methodCell]*types.Interface
	// assertNarrow holds the interface of each local value that a type
	// assertion produced, and assertDrop marks a name that another statement
	// can write. A pass before the main walk fills the 2. Read iface.go.
	assertNarrow map[*types.Var]*types.Interface
	assertDrop   map[*types.Var]bool

	// parent joins the key of a generic signature with the key of each
	// instance of it. The analysis reads a generic function 1 time, thus the 2
	// texts of 1 signature differ. Read linkInstance.
	parent map[string]string
	// byGroup gives each key of 1 group, by the root of the group.
	byGroup map[string][]string

	// The value flow. Read flow.go.
	pts      map[cell]map[any]bool
	edges    map[cell]map[cell]bool
	work     []cell
	inWork   map[cell]bool
	sites    []*callSite
	watchers map[cell][]*callSite

	litSigs    map[*ast.FuncLit]*types.Signature
	rangeShape map[*ast.RangeStmt][]cell
}

// file reads 1 file. Each function gets a summary, and each package variable
// joins the flow.
func (a *analyzer) file(f *ast.File) {
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			obj, _ := a.pkg.TypesInfo.Defs[d.Name].(*types.Func)
			if obj == nil {
				continue
			}
			if sig, ok := obj.Type().(*types.Signature); ok && sig.Recv() != nil {
				key := methodKey(obj.Name(), sig)
				a.byMethod[key] = append(a.byMethod[key], any(obj))
			}
			if d.Body != nil {
				a.walkFunc(any(obj), d.Body)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					a.valueSpec(spec)
					for _, v := range spec.Values {
						a.walkTopExpr(v)
					}
				case *ast.TypeSpec:
					a.typeSpec(d, spec)
				}
			}
		}
	}
	// The links of the generic instances. An escaped value of a generic
	// signature and a read at an instance of it share 1 group.
	ast.Inspect(f, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			a.linkUse(id)
		}
		return true
	})
}

// walkTopExpr reads an expression of a package variable. A function literal
// inside it needs a summary of its own.
func (a *analyzer) walkTopExpr(e ast.Expr) {
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			a.registerLit(n)
			return false
		case *ast.CompositeLit:
			a.compositeLit(n)
		}
		return true
	})
}

func (a *analyzer) registerLit(lit *ast.FuncLit) {
	sig, _ := a.pkg.TypesInfo.Types[lit].Type.(*types.Signature)
	a.litSigs[lit] = sig
	a.walkFunc(any(lit), lit.Body)
}

// walkFunc reads 1 function body and builds its summary. A nested function
// literal and the body of a range loop over a function each get a summary of
// their own.
func (a *analyzer) walkFunc(key any, body ast.Node) {
	s := &summary{key: key}
	a.summaries[key] = s
	a.order = append(a.order, key)
	a.namedResults(key)
	var visit func(n ast.Node) bool
	visit = func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			a.registerLit(n)
			return false
		case *ast.SendStmt:
			s.blocksAt(n.Pos(), "a channel send")
			a.escape(n.Value)
		case *ast.SelectStmt:
			s.blocksAt(n.Pos(), "a select statement")
		case *ast.UnaryExpr:
			if n.Op == token.ARROW {
				s.blocksAt(n.Pos(), "a channel receive")
			}
			if n.Op == token.AND {
				a.escapeAddress(n.X)
			}
		case *ast.RangeStmt:
			return a.rangeStmt(s, n, visit)
		case *ast.AssignStmt:
			a.assignStmt(n)
		case *ast.ValueSpec:
			a.valueSpec(n)
		case *ast.ReturnStmt:
			a.returnStmt(key, n)
		case *ast.CompositeLit:
			a.compositeLit(n)
		case *ast.CallExpr:
			a.callExpr(s, n)
		}
		return true
	}
	ast.Inspect(body, func(n ast.Node) bool {
		if n == body {
			return true
		}
		return visit(n)
	})
}

// namedResults joins the cell of each named result with the result of the
// function, because a bare return and a deferred write each reach it.
func (a *analyzer) namedResults(key any) {
	sig := a.sigOf(key)
	if sig == nil {
		return
	}
	for i := 0; i < sig.Results().Len(); i++ {
		res := sig.Results().At(i)
		if res.Name() != "" && funcSig(res.Type()) != nil {
			a.addEdge(funcCell(res), retCell{key, i})
		}
	}
}

func (a *analyzer) sigOf(key any) *types.Signature {
	switch key := key.(type) {
	case *types.Func:
		sig, _ := key.Type().(*types.Signature)
		return sig
	case *ast.FuncLit:
		return a.litSigs[key]
	}
	return nil
}

// rangeStmt reads 1 range statement inside the visit of a body.
//
// A loop over a function holds an implicit call of the sequence, with the
// loop body as the yield function. The body becomes a Dart closure, thus a
// block operation inside it needs the body as a function of its own. Before
// this rule, such an operation was lost, and the closure ignored the park of
// a channel.
func (a *analyzer) rangeStmt(s *summary, n *ast.RangeStmt, visit func(ast.Node) bool) bool {
	if n.X == nil {
		return true
	}
	t := a.typeOf(n.X)
	if _, ok := typeset.Core(t).(*types.Chan); ok {
		s.blocksAt(n.Pos(), "a range loop over a channel")
		return true
	}
	sig, _ := typeset.Core(t).(*types.Signature)
	if sig == nil || sig.Params().Len() != 1 {
		a.rangeVars(n)
		return true
	}
	yield := funcSig(typeset.Core(sig.Params().At(0).Type()))
	if yield == nil {
		return true
	}
	// The expression of the sequence and the loop variables stay in the
	// enclosing function.
	ast.Inspect(n.X, visit)
	if n.Key != nil {
		ast.Inspect(n.Key, visit)
	}
	if n.Value != nil {
		ast.Inspect(n.Value, visit)
	}
	shape := make([]cell, yield.Params().Len())
	if len(shape) > 0 && n.Key != nil {
		if c, blank := a.lhsCell(n.Key); !blank {
			shape[0] = c
		}
	}
	if len(shape) > 1 && n.Value != nil {
		if c, blank := a.lhsCell(n.Value); !blank {
			shape[1] = c
		}
	}
	a.rangeShape[n] = shape
	site := &callSite{
		summary: s,
		rng:     n,
		callee:  a.flowExpr(n.X),
		args:    map[int]source{0: {keys: []any{n}}},
		argEsc:  map[int]string{0: signatureKey(yield)},
		done:    map[any]bool{},
	}
	a.sites = append(a.sites, site)
	s.sites = append(s.sites, site)
	a.walkFunc(any(n), n.Body)
	return false
}

// rangeVars reads the loop variables of a range over a container. A function
// value inside a container stands outside the net, thus the variable draws
// from the escape group.
func (a *analyzer) rangeVars(n *ast.RangeStmt) {
	for _, e := range []ast.Expr{n.Key, n.Value} {
		if e == nil {
			continue
		}
		sig := funcSig(a.typeOf(e))
		if sig == nil {
			continue
		}
		if c, blank := a.lhsCell(e); c != nil && !blank {
			a.addEdge(sigCell{signatureKey(sig)}, c)
		}
	}
}

// escapeAddress joins a variable with its escape group when the code takes
// its address. A pointer aliases the cell, and the flow does not follow a
// pointer.
func (a *analyzer) escapeAddress(e ast.Expr) {
	sig := funcSig(a.typeOf(e))
	if sig == nil {
		return
	}
	c, blank := a.lhsCell(e)
	if c == nil || blank {
		return
	}
	group := sigCell{signatureKey(sig)}
	a.addEdge(c, group)
	a.addEdge(group, c)
}

// assignStmt wires each pair of an assignment.
func (a *analyzer) assignStmt(n *ast.AssignStmt) {
	if len(n.Lhs) > 1 && len(n.Rhs) == 1 {
		call, ok := unparen(n.Rhs[0]).(*ast.CallExpr)
		if !ok {
			return
		}
		tuple, ok := a.typeOf(call).(*types.Tuple)
		if !ok {
			return
		}
		for i, lhs := range n.Lhs {
			if i >= tuple.Len() {
				break
			}
			a.assign(lhs, source{cells: []cell{outCell{call, i}}}, tuple.At(i).Type())
		}
		return
	}
	for i, lhs := range n.Lhs {
		if i >= len(n.Rhs) {
			break
		}
		a.assign(lhs, a.flowExpr(n.Rhs[i]), a.typeOf(n.Rhs[i]))
	}
}

// valueSpec wires each pair of a var declaration.
func (a *analyzer) valueSpec(n *ast.ValueSpec) {
	if len(n.Names) > 1 && len(n.Values) == 1 {
		call, ok := unparen(n.Values[0]).(*ast.CallExpr)
		if !ok {
			return
		}
		tuple, ok := a.typeOf(call).(*types.Tuple)
		if !ok {
			return
		}
		for i, name := range n.Names {
			if i >= tuple.Len() {
				break
			}
			a.assign(name, source{cells: []cell{outCell{call, i}}}, tuple.At(i).Type())
		}
		return
	}
	for i, name := range n.Names {
		if i >= len(n.Values) {
			break
		}
		a.assign(name, a.flowExpr(n.Values[i]), a.typeOf(n.Values[i]))
	}
}

// returnStmt wires each result of a return statement.
func (a *analyzer) returnStmt(key any, n *ast.ReturnStmt) {
	sig := a.sigOf(key)
	if sig == nil {
		return
	}
	if len(n.Results) == 1 && sig.Results().Len() > 1 {
		call, ok := unparen(n.Results[0]).(*ast.CallExpr)
		if !ok {
			return
		}
		for i := 0; i < sig.Results().Len(); i++ {
			if funcSig(sig.Results().At(i).Type()) != nil {
				a.addEdge(outCell{call, i}, retCell{key, i})
			}
		}
		return
	}
	for i, e := range n.Results {
		if i >= sig.Results().Len() {
			break
		}
		if funcSig(sig.Results().At(i).Type()) == nil {
			if funcSig(a.typeOf(e)) != nil {
				a.escape(e)
			}
			continue
		}
		a.flowTo(retCell{key, i}, a.flowExpr(e))
	}
}

// compositeLit wires the elements of a composite literal. A struct field is a
// cell, and an element of a slice, of an array, or of a map escapes.
func (a *analyzer) compositeLit(n *ast.CompositeLit) {
	t := a.typeOf(n)
	if p, ok := types.Unalias(t).Underlying().(*types.Pointer); ok {
		t = p.Elem()
	}
	st, _ := types.Unalias(t).Underlying().(*types.Struct)
	for i, elt := range n.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if st != nil {
				if id, ok := kv.Key.(*ast.Ident); ok {
					if v, ok := a.pkg.TypesInfo.Uses[id].(*types.Var); ok {
						a.fieldFlow(v, kv.Value)
						continue
					}
				}
			}
			if funcSig(a.typeOf(kv.Value)) != nil {
				a.escape(kv.Value)
			}
			continue
		}
		if st != nil && i < st.NumFields() {
			a.fieldFlow(st.Field(i), elt)
			continue
		}
		if funcSig(a.typeOf(elt)) != nil {
			a.escape(elt)
		}
	}
}

func (a *analyzer) fieldFlow(field *types.Var, e ast.Expr) {
	if funcSig(a.typeOf(e)) == nil {
		return
	}
	if funcSig(field.Type()) == nil {
		a.escape(e)
		return
	}
	a.flowTo(funcCell(field), a.flowExpr(e))
}

// callExpr reads 1 call. A conversion keeps the value, a builtin drops each
// function argument into its escape group, and each other call becomes a call
// site of the flow.
func (a *analyzer) callExpr(s *summary, call *ast.CallExpr) {
	fun := unparen(call.Fun)
	if tv, ok := a.pkg.TypesInfo.Types[fun]; ok && tv.IsType() {
		return
	}
	callee, handled := a.calleeSource(s, call, fun)
	if handled {
		return
	}
	site := &callSite{
		summary: s,
		call:    call,
		callee:  callee,
		args:    map[int]source{},
		argEsc:  map[int]string{},
		done:    map[any]bool{},
	}
	for i, arg := range call.Args {
		sig := funcSig(a.typeOf(arg))
		if sig == nil {
			continue
		}
		site.args[i] = a.flowExpr(arg)
		site.argEsc[i] = signatureKey(sig)
	}
	a.sites = append(a.sites, site)
	s.sites = append(s.sites, site)
}

// calleeSource gives the source of the callee of a call. The flag reports
// that the call needs no site: a builtin, or an interop function.
func (a *analyzer) calleeSource(s *summary, call *ast.CallExpr, fun ast.Expr) (source, bool) {
	switch fun := fun.(type) {
	case *ast.Ident:
		switch obj := a.pkg.TypesInfo.Uses[fun].(type) {
		case *types.Func:
			return a.funcCallee(s, call, obj)
		case *types.Builtin:
			a.escapeArgs(call)
			return source{}, true
		case *types.Var:
			return source{cells: []cell{funcCell(obj)}}, false
		}
	case *ast.SelectorExpr:
		sel, ok := a.pkg.TypesInfo.Selections[fun]
		if !ok {
			switch obj := a.pkg.TypesInfo.Uses[fun.Sel].(type) {
			case *types.Func:
				return a.funcCallee(s, call, obj)
			case *types.Var:
				return source{cells: []cell{funcCell(obj)}}, false
			}
			break
		}
		switch sel.Kind() {
		case types.MethodVal, types.MethodExpr:
			fn, ok := sel.Obj().(*types.Func)
			if !ok {
				break
			}
			if isInterface(sel.Recv()) || abstractMethod(fn) {
				if src, ok := a.methodSource(fun.X, sel.Recv(), fn); ok {
					return src, false
				}
				break
			}
			a.recvFlow(fn, fun.X)
			if sel.Kind() == types.MethodExpr {
				return source{keys: []any{methodExprKey{fn.Origin()}}}, false
			}
			return source{keys: []any{fn.Origin()}}, false
		case types.FieldVal:
			if v, ok := sel.Obj().(*types.Var); ok {
				return source{cells: []cell{funcCell(v)}}, false
			}
		}
	case *ast.IndexExpr:
		if src := a.instanceSource(fun.X); !src.empty() {
			if fn, ok := src.keys[0].(*types.Func); ok {
				return a.funcCallee(s, call, fn)
			}
		}
	case *ast.IndexListExpr:
		if src := a.instanceSource(fun.X); !src.empty() {
			if fn, ok := src.keys[0].(*types.Func); ok {
				return a.funcCallee(s, call, fn)
			}
		}
	case *ast.FuncLit:
		return source{keys: []any{fun}}, false
	case *ast.CallExpr:
		return source{cells: []cell{outCell{fun, 0}}}, false
	}
	return a.flowExpr(fun), false
}

// funcCallee gives the source of a call of a named function. An interop
// function that gives up control marks the caller, and each other interop
// function drops its function arguments into the escape group, because the
// runtime library calls them outside the net.
func (a *analyzer) funcCallee(s *summary, call *ast.CallExpr, fn *types.Func) (source, bool) {
	if blockingIntrinsic(fn) {
		s.blocksAt(call.Pos(), "a call of "+fn.Pkg().Path()+"."+fn.Name())
		s.intrinsics = append(s.intrinsics, call)
		return source{}, true
	}
	if fn.Pkg() != nil && fn.Pkg().Path() == "syscall/dart" {
		a.escapeArgs(call)
	}
	return source{keys: []any{fn.Origin()}}, false
}

// recvFlow joins the receiver of a method call with the receiver cell, when
// the receiver is a function value. A method on a named function type calls
// through its receiver.
func (a *analyzer) recvFlow(fn *types.Func, recv ast.Expr) {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return
	}
	if funcSig(sig.Recv().Type()) == nil {
		return
	}
	origin, ok := fn.Origin().Type().(*types.Signature)
	if !ok || origin.Recv() == nil {
		return
	}
	a.flowTo(funcCell(origin.Recv()), a.flowExpr(recv))
}

// escapeArgs drops each function argument of a call into its escape group.
func (a *analyzer) escapeArgs(call *ast.CallExpr) {
	for _, arg := range call.Args {
		if funcSig(a.typeOf(arg)) != nil {
			a.escape(arg)
		}
	}
}

// abstractMethod reports whether the method belongs to an interface type. A
// promoted method of an embedded interface field reaches a call with a
// concrete receiver, and the dispatch still runs through the interface.
func abstractMethod(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}
	return isInterface(sig.Recv().Type())
}

// linkUse joins the key of a generic signature with the key of the instance
// that stands at this name.
func (a *analyzer) linkUse(n *ast.Ident) {
	inst, ok := a.pkg.TypesInfo.Instances[n]
	if !ok || inst.Type == nil {
		return
	}
	if obj, ok := a.pkg.TypesInfo.Uses[n].(*types.Func); ok {
		a.linkInstance(obj.Origin().Type(), inst.Type)
	}
}

// linkInstance walks a generic type and 1 instance of it at the same time, and
// it joins the key of each function type that the 2 hold.
func (a *analyzer) linkInstance(generic, concrete types.Type) {
	seen := map[[2]types.Type]bool{}
	var walk func(g, c types.Type)
	walk = func(g, c types.Type) {
		if g == nil || c == nil {
			return
		}
		pair := [2]types.Type{g, c}
		if seen[pair] {
			return
		}
		seen[pair] = true
		gu, cu := types.Unalias(g), types.Unalias(c)
		if gn, ok := gu.(*types.Named); ok {
			if cn, ok := cu.(*types.Named); ok {
				walk(gn.Underlying(), cn.Underlying())
				return
			}
		}
		switch gt := gu.(type) {
		case *types.Signature:
			ct, ok := cu.(*types.Signature)
			if !ok || gt.Params().Len() != ct.Params().Len() ||
				gt.Results().Len() != ct.Results().Len() {
				return
			}
			a.link(signatureKey(gt), signatureKey(ct))
			for i := 0; i < gt.Params().Len(); i++ {
				walk(gt.Params().At(i).Type(), ct.Params().At(i).Type())
			}
			for i := 0; i < gt.Results().Len(); i++ {
				walk(gt.Results().At(i).Type(), ct.Results().At(i).Type())
			}
		case *types.Pointer:
			if ct, ok := cu.(*types.Pointer); ok {
				walk(gt.Elem(), ct.Elem())
			}
		case *types.Slice:
			if ct, ok := cu.(*types.Slice); ok {
				walk(gt.Elem(), ct.Elem())
			}
		case *types.Array:
			if ct, ok := cu.(*types.Array); ok {
				walk(gt.Elem(), ct.Elem())
			}
		case *types.Chan:
			if ct, ok := cu.(*types.Chan); ok {
				walk(gt.Elem(), ct.Elem())
			}
		case *types.Map:
			if ct, ok := cu.(*types.Map); ok {
				walk(gt.Key(), ct.Key())
				walk(gt.Elem(), ct.Elem())
			}
		case *types.Struct:
			ct, ok := cu.(*types.Struct)
			if !ok || gt.NumFields() != ct.NumFields() {
				return
			}
			for i := 0; i < gt.NumFields(); i++ {
				walk(gt.Field(i).Type(), ct.Field(i).Type())
			}
		}
	}
	walk(generic, concrete)
}

// link joins the 2 keys in 1 group.
func (a *analyzer) link(x, y string) {
	if x == "" || y == "" || x == y {
		return
	}
	rx, ry := a.find(x), a.find(y)
	if rx != ry {
		a.parent[ry] = rx
	}
}

func (a *analyzer) find(k string) string {
	p, ok := a.parent[k]
	if !ok {
		a.parent[k] = k
		return k
	}
	if p == k {
		return k
	}
	root := a.find(p)
	a.parent[k] = root
	return root
}

// components groups each key that names the same signature. A key of a generic
// signature and a key of an instance of it stand in 1 group.
func (a *analyzer) components() {
	a.byGroup = map[string][]string{}
	for key := range a.parent {
		root := a.find(key)
		a.byGroup[root] = append(a.byGroup[root], key)
	}
}

// spreadKey marks each key of the group of the key.
func (a *analyzer) spreadKey(key string) bool {
	changed := false
	for _, other := range a.byGroup[a.find(key)] {
		if !a.result.sigAsync[other] {
			a.result.sigAsync[other] = true
			changed = true
		}
	}
	return changed
}

// resolveSites reads the answer of the flow into each call site. The flow is
// at its fixed point, thus the set of each site is final.
func (a *analyzer) resolveSites() {
	for _, site := range a.sites {
		site.targets = nil
		for _, t := range a.targets(site.callee) {
			site.targets = append(site.targets, blockTarget(t))
		}
		site.fallbacks = sigFallbacks(site.callee)
	}
}

// blockTarget gives the key that holds the block state of a target.
func blockTarget(t any) any {
	if m, ok := t.(methodExprKey); ok {
		return any(m.fn)
	}
	return t
}

func (a *analyzer) typeOf(e ast.Expr) types.Type {
	if tv, ok := a.pkg.TypesInfo.Types[e]; ok {
		return tv.Type
	}
	return types.Typ[types.Invalid]
}

// blockingIntrinsic reports whether an interop function gives up control. The
// body of such a function holds no block operation, thus the analysis needs
// this list.
func blockingIntrinsic(fn *types.Func) bool {
	if fn.Pkg() == nil || fn.Pkg().Path() != "syscall/dart" {
		return false
	}
	switch fn.Name() {
	case "Sleep", "Worker", "Catch", "GC",
		"DialTCP", "DialTLS", "DialTLSProxy", "ConnRead", "ConnWrite",
		"ListenTCP", "Accept",
		"ListenTLS", "AcceptTLS",
		"DialUnix", "ListenUnix",
		"DialUDP", "ListenUDP", "PacketRead", "PacketWrite",
		"ProcStart", "ProcWrite", "ProcCloseStdin", "ProcRead",
		"ProcReadErr", "ProcWait", "SignalWait", "ValueRecv",
		"H2Fetch", "H2Read", "H2Next",
		"FetchStart", "FetchRead":
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
	for _, site := range s.sites {
		if a.siteAsync(site) {
			return true
		}
	}
	return false
}

// siteAsync reports whether a call site must wait. A target that blocks or an
// escape group that holds a function that blocks each give true.
func (a *analyzer) siteAsync(site *callSite) bool {
	for _, t := range site.targets {
		if a.result.blocking[t] {
			return true
		}
	}
	for _, key := range site.fallbacks {
		if a.result.sigAsync[key] {
			return true
		}
	}
	return false
}

// spread marks the escape group of each escaped function that blocks, and it
// gives the same state to each key of a linked group.
func (a *analyzer) spread() bool {
	changed := false
	for c, set := range a.pts {
		group, ok := c.(sigCell)
		if !ok || a.result.sigAsync[group.key] {
			continue
		}
		for member := range set {
			if a.result.blocking[blockTarget(member)] {
				a.result.sigAsync[group.key] = true
				changed = true
				break
			}
		}
	}
	async := make([]string, 0, len(a.result.sigAsync))
	for key, on := range a.result.sigAsync {
		if on {
			async = append(async, key)
		}
	}
	for _, key := range async {
		if a.spreadKey(key) {
			changed = true
		}
	}
	return changed
}

// markCalls marks each call that must wait for a result.
func (a *analyzer) markCalls() {
	for _, key := range a.order {
		s := a.summaries[key]
		for _, call := range s.intrinsics {
			a.result.callAsync[call] = true
		}
		for _, site := range s.sites {
			if !a.siteAsync(site) {
				continue
			}
			if site.rng != nil {
				a.result.rangeAsync[site.rng] = true
				continue
			}
			a.result.callAsync[site.call] = true
		}
	}
}

// keySep joins the parts of a signature key, and keyResults marks the start of
// the results. The 2 bytes stand outside the text of a Go type.
const (
	keySep     = "\x00"
	keyResults = "\x01"
	// wildcard stands for a type that holds a type parameter. The analysis
	// reads a generic function 1 time, and the call site holds the type
	// argument, thus the 2 texts of 1 signature differ.
	wildcard = "\x02"
)

// methodKey names a method of an interface by its name and its signature.
func methodKey(name string, sig *types.Signature) string {
	return name + keySep + signatureKey(sig)
}

// typeKey names 1 type of a signature. A type parameter takes the wildcard, and
// each other part of the type keeps its text. Thus 2 signatures that differ
// only in a type argument keep 2 keys.
func typeKey(t types.Type) string {
	switch t := types.Unalias(t).(type) {
	case *types.TypeParam:
		return wildcard
	case *types.Pointer:
		return "*" + typeKey(t.Elem())
	case *types.Slice:
		return "[]" + typeKey(t.Elem())
	case *types.Array:
		return "[" + strconv.FormatInt(t.Len(), 10) + "]" + typeKey(t.Elem())
	case *types.Chan:
		return "chan" + strconv.Itoa(int(t.Dir())) + " " + typeKey(t.Elem())
	case *types.Map:
		return "map[" + typeKey(t.Key()) + "]" + typeKey(t.Elem())
	case *types.Signature:
		parts := make([]string, 0, t.Params().Len()+t.Results().Len()+1)
		for i := 0; i < t.Params().Len(); i++ {
			parts = append(parts, typeKey(t.Params().At(i).Type()))
		}
		parts = append(parts, keyResults)
		for i := 0; i < t.Results().Len(); i++ {
			parts = append(parts, typeKey(t.Results().At(i).Type()))
		}
		return "func(" + strings.Join(parts, ",") + ")"
	case *types.Struct:
		parts := make([]string, t.NumFields())
		for i := range parts {
			parts[i] = t.Field(i).Name() + " " + typeKey(t.Field(i).Type())
		}
		return "struct{" + strings.Join(parts, ";") + "}"
	case *types.Named:
		if t.TypeArgs().Len() == 0 {
			return types.TypeString(t, nil)
		}
		args := make([]string, t.TypeArgs().Len())
		for i := range args {
			args[i] = typeKey(t.TypeArgs().At(i))
		}
		return types.TypeString(t.Origin(), nil) + "[" + strings.Join(args, ",") + "]"
	}
	return types.TypeString(t, nil)
}

// signatureKey builds a stable text of a signature. The receiver stays out,
// because a method group needs 1 key for each implementation.
func signatureKey(sig *types.Signature) string {
	if sig == nil {
		return ""
	}
	parts := make([]string, 0, sig.Params().Len()+sig.Results().Len()+2)
	for i := 0; i < sig.Params().Len(); i++ {
		parts = append(parts, typeKey(sig.Params().At(i).Type()))
	}
	parts = append(parts, keyResults)
	for i := 0; i < sig.Results().Len(); i++ {
		parts = append(parts, typeKey(sig.Results().At(i).Type()))
	}
	if sig.Variadic() {
		parts = append(parts, "...")
	}
	return strings.Join(parts, keySep)
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

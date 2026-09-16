package normalize

import (
	"fmt"
	"go/token"
	"go/types"
	"strconv"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
)

// A function that can block gives up control with a plain return, thus it keeps
// no structured control flow. This file rewrites such a body into a flat state
// machine: a loop with a switch on a state variable.
//
// The rewrite also holds each goto statement, because a goto needs the same
// form. A function with a goto and with no block operation takes the machine
// with no frame.

// flatten builds the state machine of 1 body.
//
// blocking marks a body that can block. Such a body needs a frame, and each
// call that can block needs a resume point.
func (b *builder) flatten(body *ir.Block, params []*ir.Var, recv *ir.Var,
	blocking, defers, deferBlocks bool, results []*ir.Var,
	class string, pos token.Pos) *ir.Machine {
	m := &ir.Machine{Class: class, Recv: recv, Defer: defers, Blocking: blocking,
		DeferBlocks: deferBlocks}
	f := &flat{b: b, m: m, pos: pos, labels: map[string]int{}}
	if recv != nil {
		m.Locals = append(m.Locals, recv)
	}
	m.Locals = append(m.Locals, params...)
	f.newState()
	if deferBlocks {
		f.openDeferChain(results)
	}
	f.markLabels(body.List)
	f.block(body.List)
	if deferBlocks {
		f.writeDeferChain()
	}
	f.close()
	placeLocalFuncs(m)
	return m
}

// openDeferChain reserves the 3 states of the defer chain and gives each result
// a variable of the frame. A return statement writes the variables and jumps to
// the chain, thus the deferred calls run inside the machine.
func (f *flat) openDeferChain(results []*ir.Var) {
	f.m.DeferState = f.state()
	f.deferBody = f.state()
	f.deferDone = f.state()
	for i, r := range results {
		if r.Named {
			f.m.ResultVars = append(f.m.ResultVars, r)
			continue
		}
		v := &ir.Var{Name: fmt.Sprintf("$dret%d", i), Type: r.Type}
		f.m.ResultVars = append(f.m.ResultVars, v)
		f.m.Locals = append(f.m.Locals, v)
	}
}

// writeDeferChain writes the 3 states that run the deferred calls.
func (f *flat) writeDeferChain() {
	f.cur = f.m.DeferState
	f.emit(&ir.DeferNext{Body: f.deferBody, Done: f.deferDone})
	f.cur = f.deferBody
	f.emit(&ir.Resume{})
	f.emit(&ir.Jump{State: f.m.DeferState})
	f.cur = f.deferDone
	f.emit(&ir.DeferEnd{})
	values := make([]ir.Expr, len(f.m.ResultVars))
	for i, v := range f.m.ResultVars {
		values[i] = identOf(v)
	}
	f.emit(&ir.Return{Values: values})
}

// deferReturn writes each result into its variable and jumps to the chain of
// the deferred calls.
//
// A return of 1 call that gives each result writes the record of the call into
// the whole set of variables. A plain assignment would put the record into the
// first variable, and the Dart output would then hold a type error.
func (f *flat) deferReturn(s *ir.Return) {
	if len(s.Values) == 1 && len(f.m.ResultVars) > 1 {
		lhs := make([]ir.Expr, len(f.m.ResultVars))
		for i, v := range f.m.ResultVars {
			lhs[i] = identOf(v)
		}
		f.emit(&ir.AssignMulti{Lhs: lhs, Rhs: s.Values[0]})
		f.emit(&ir.Jump{State: f.m.DeferState})
		return
	}
	// Go reads each operand of a return first and it writes the result
	// variables after that, thus an operand that panics leaves each variable
	// with its old value.
	if len(s.Values) > 1 {
		tmps := make([]*ir.Var, 0, len(s.Values))
		for i, v := range s.Values {
			if i >= len(f.m.ResultVars) {
				break
			}
			t := f.b.tempVar(f.pos, "rv"+strconv.Itoa(i), f.m.ResultVars[i].Type)
			tmps = append(tmps, t)
			f.emit(&ir.Declare{Vars: []*ir.Var{t}, Init: v})
		}
		for i, t := range tmps {
			f.emit(&ir.Assign{
				Lhs: identOf(f.m.ResultVars[i]),
				Rhs: &ir.Ident{Name: t.Name, Type: t.Type},
			})
		}
		f.emit(&ir.Jump{State: f.m.DeferState})
		return
	}
	for i, v := range s.Values {
		if i >= len(f.m.ResultVars) {
			break
		}
		f.emit(&ir.Assign{Lhs: identOf(f.m.ResultVars[i]), Rhs: v})
	}
	f.emit(&ir.Jump{State: f.m.DeferState})
}

type flat struct {
	b   *builder
	m   *ir.Machine
	cur int
	// pos holds the position of the statement that the pass reads. Each
	// statement that the pass builds takes it, thus the source map of the
	// output holds a state of a machine.
	pos    token.Pos
	seq    int
	loops  []flatLoop
	labels map[string]int

	// deferBody runs 1 deferred call, and deferDone ends the chain.
	deferBody int
	deferDone int
}

// flatLoop names the state that a break statement and a continue statement
// reach. cont is -1 for a switch and for a select, because such a construct
// takes no continue statement.
type flatLoop struct {
	label string
	brk   int
	cont  int
}

func (f *flat) newState() int {
	f.m.States = append(f.m.States, &ir.Block{})
	f.cur = len(f.m.States) - 1
	return f.cur
}

// state adds a state and keeps the current one.
func (f *flat) state() int {
	prev := f.cur
	n := f.newState()
	f.cur = prev
	return n
}

func (f *flat) emit(s ir.Stmt) {
	ir.SetPos(s, f.pos)
	blk := f.m.States[f.cur]
	blk.List = append(blk.List, s)
}

func (f *flat) jump(n int) {
	if f.done() {
		return
	}
	f.emit(&ir.Jump{State: n})
}

// done reports whether the current state already leaves through a transfer.
func (f *flat) done() bool {
	list := f.m.States[f.cur].List
	if len(list) == 0 {
		return false
	}
	switch list[len(list)-1].(type) {
	case *ir.Jump, *ir.Return:
		return true
	}
	return false
}

// close ends each state that holds no transfer. Such a state stands at the end
// of the body, because each construct writes the transfer of its own states.
// A Dart case clause needs such a transfer.
func (f *flat) close() {
	for i := range f.m.States {
		f.cur = i
		if f.done() {
			continue
		}
		if f.m.DeferBlocks {
			f.emit(&ir.Jump{State: f.m.DeferState})
			continue
		}
		// The return of a fall off the end keeps no position. The emitter
		// then marks the closing brace, exactly as Go names such an exit.
		blk := f.m.States[f.cur]
		blk.List = append(blk.List, &ir.Return{})
	}
}

// markLabels gives each goto target its own state. A goto can stand before its
// label, thus the pass reads each label first.
func (f *flat) markLabels(list []ir.Stmt) {
	for _, s := range list {
		switch s := s.(type) {
		case *ir.Labeled:
			if s.Goto {
				f.labels[s.Label] = f.state()
			}
			f.markLabels([]ir.Stmt{s.Stmt})
		case *ir.Block:
			f.markLabels(s.List)
		case *ir.If:
			f.markLabels(s.Then.List)
			if s.Else != nil {
				f.markLabels([]ir.Stmt{s.Else})
			}
		case *ir.For:
			f.markLabels(s.Body.List)
		case *ir.Switch:
			for _, c := range s.Cases {
				f.markLabels(c.Body.List)
			}
		case *ir.Select:
			for _, c := range s.Cases {
				f.markLabels(c.Body.List)
			}
		}
	}
}

func (f *flat) block(list []ir.Stmt) {
	for _, s := range list {
		f.stmt(s)
	}
}

func (f *flat) stmt(s ir.Stmt) {
	if p := ir.PosOf(s); p.IsValid() {
		f.pos = p
	}
	if !f.split(s) {
		f.plain(s)
		return
	}
	switch s := s.(type) {
	case *ir.Block:
		f.block(s.List)
	case *ir.If:
		f.ifStmt(s)
	case *ir.For:
		f.forStmt(s, s.Label)
	case *ir.Switch:
		f.switchStmt(s, s.Label)
	case *ir.Select:
		f.selectStmt(s, s.Label)
	case *ir.Labeled:
		f.labeled(s)
	case *ir.Goto:
		f.gotoStmt(s)
	case *ir.Break:
		f.branch(s.Label, true, s)
	case *ir.Continue:
		f.branch(s.Label, false, s)
	case *ir.Return:
		if f.m.DeferBlocks {
			f.deferReturn(&ir.Return{Values: f.operands(s.Values)})
			return
		}
		f.emit(&ir.Return{Values: f.operands(s.Values)})
	case *ir.ExprStmt:
		f.exprStmt(s)
	case *ir.Assign:
		f.assign(s)
	case *ir.AssignMulti:
		f.assignMulti(s)
	case *ir.Declare:
		f.declare(s)
	case *ir.Send:
		f.send(s)
	case *ir.Go:
		out := &ir.Go{Node: s.Node, Call: f.callArgs(s.Call)}
		f.spawn(out, out.Call)
	case *ir.Defer:
		out := &ir.Defer{Node: s.Node, Call: f.callArgs(s.Call)}
		f.spawn(out, out.Call)
	default:
		f.b.fail(f.pos, "the statement %T has no state machine form", s)
	}
}

// plain adds a statement that holds no operation that blocks. A declaration
// still becomes a variable of the frame, because a later state reads it.
func (f *flat) plain(s ir.Stmt) {
	switch s := s.(type) {
	case *ir.Declare:
		for _, out := range f.lift(s) {
			f.emit(out)
		}
	case *ir.Go:
		f.spawn(s, s.Call)
	case *ir.Defer:
		f.spawn(s, s.Call)
	default:
		f.emit(s)
	}
}

// spawn writes a go statement or a defer statement of a machine.
//
// Go reads the function value and each argument at the point of the
// statement, and the Dart closure of the goroutine or of the deferred call
// reads the variable and not the value. Each local of a machine stands in
// front of the loop, thus a second pass of the loop would change what a
// closure of the first pass reads.
//
// The statement thus takes a block of its own, and the block declares a copy
// of each argument that names a local of the machine. A Dart block gives a new
// binding at each pass, thus the closure of each pass reads its own copy.
func (f *flat) spawn(s ir.Stmt, target ir.Expr) {
	local := map[string]bool{}
	for _, v := range f.m.Locals {
		local[v.Name] = true
	}
	var list []ir.Stmt
	snap := func(e ir.Expr) ir.Expr {
		id, ok := e.(*ir.Ident)
		if !ok || !local[id.Name] {
			return e
		}
		v := f.b.tempVar(f.pos, fmt.Sprintf("gc%d", len(list)), id.GoType())
		list = append(list, &ir.Declare{Vars: []*ir.Var{v}, Init: id})
		return identOf(v)
	}
	switch call := target.(type) {
	case *ir.Call:
		if m, ok := call.Fun.(*ir.Method); ok {
			call.Fun = &ir.Method{Recv: snap(m.Recv), Obj: m.Obj, Type: m.Type,
				TypeArgs: m.TypeArgs}
		} else {
			call.Fun = snap(call.Fun)
		}
		for i, a := range call.Args {
			call.Args[i] = snap(a)
		}
	case *ir.Builtin:
		for i, a := range call.Args {
			call.Args[i] = snap(a)
		}
	}
	if len(list) == 0 {
		f.emit(s)
		return
	}
	f.emit(&ir.Block{List: append(list, s)})
}

// lift turns a declaration into a variable of the frame and an assignment. A
// Dart case clause and the next case clause share no scope, thus each variable
// of the machine stands in front of the loop.
func (f *flat) lift(d *ir.Declare) []ir.Stmt {
	f.keep(d.Vars)
	if d.Init == nil {
		// A loop runs the declaration again, thus the variable takes the zero
		// value again.
		out := make([]ir.Stmt, 0, len(d.Vars))
		for _, v := range d.Vars {
			if v.Name == "_" {
				continue
			}
			out = append(out, &ir.Assign{
				Lhs:    identOf(v),
				Rhs:    &ir.Zero{Type: v.Type},
				Define: true,
			})
		}
		return out
	}
	if len(d.Vars) == 1 {
		// A blank name holds no value, and Dart takes no write to it.
		if d.Vars[0].Name == "_" {
			return []ir.Stmt{&ir.ExprStmt{X: d.Init}}
		}
		return []ir.Stmt{&ir.Assign{Lhs: identOf(d.Vars[0]), Rhs: d.Init, Define: true}}
	}
	lhs := make([]ir.Expr, len(d.Vars))
	for i, v := range d.Vars {
		lhs[i] = identOf(v)
	}
	return []ir.Stmt{&ir.AssignMulti{Lhs: lhs, Rhs: d.Init, Define: true}}
}

// keep adds each variable that the frame holds. A blank name holds no value.
func (f *flat) keep(vars []*ir.Var) {
	for _, v := range vars {
		if v.Name == "_" {
			continue
		}
		f.m.Locals = append(f.m.Locals, v)
	}
}

func identOf(v *ir.Var) ir.Expr {
	out := &ir.Ident{Name: v.Name, Type: v.Type, Boxed: v.Boxed}
	// A nil object of a generated variable must stay nil inside the interface.
	if v.Obj != nil {
		out.Obj = v.Obj
	}
	return out
}

// local adds a variable to the frame and gives its name.
func (f *flat) local(t types.Type) *ir.Var {
	f.seq++
	n := fmt.Sprintf("$t%d", f.seq)
	for f.b.used[n] {
		f.seq++
		n = fmt.Sprintf("$t%d", f.seq)
	}
	f.b.used[n] = true
	v := &ir.Var{Name: n, Type: t}
	f.m.Locals = append(f.m.Locals, v)
	return v
}

// temp writes an expression into a variable of the frame. Go evaluates each
// operand in order, thus an operand that stands before a call that blocks needs
// its own variable.
// temp writes an expression into a variable of the frame, and it gives the
// read of that variable.
//
// A variable of the frame lives as long as the machine, and 1 loop writes it
// again at each turn. A struct and an array each hold a Dart object, thus a
// read of such a variable must take a copy. Without the copy a composite
// literal of an earlier turn holds the object that the next turn writes.
// Stage 4 marks no copy point here, because the source expression is a call or
// another fresh value, and Go needs no copy of such a value.
func (f *flat) temp(x ir.Expr) ir.Expr {
	switch x.(type) {
	case *ir.Const, *ir.Zero, *ir.Blank, nil:
		return x
	}
	v := f.local(x.GoType())
	f.emit(&ir.Assign{Lhs: identOf(v), Rhs: x})
	return f.readLocal(v)
}

// readLocal gives the read of a variable of the frame.
//
// A struct and an array each hold a Dart object, and a variable of the frame
// lives as long as the machine. 1 loop writes such a variable again at each
// turn, thus a read must take a copy. Without the copy a composite literal of
// an earlier turn holds the object that the next turn writes. Stage 4 marks no
// copy point on the source expression, because a call and each other fresh
// value need none in Go.
func (f *flat) readLocal(v *ir.Var) ir.Expr {
	out := identOf(v)
	if needsCopy(v.Type) {
		return &ir.Copy{X: out, Type: v.Type}
	}
	return out
}

// pause ends the state after a call that blocks. The next state re-enters the
// call and leaves the machine when the call gave up control.
func (f *flat) pause(v *ir.Var) {
	n := f.state()
	f.jump(n)
	f.cur = n
	f.emit(&ir.Resume{Var: v, State: n})
}

func (f *flat) ifStmt(s *ir.If) {
	cond := f.value(s.Cond)
	thenSplit := f.splitBlock(s.Then)
	elseSplit := s.Else != nil && f.split(s.Else)
	if !thenSplit && !elseSplit {
		f.emit(&ir.If{Cond: cond, Then: s.Then, Else: s.Else})
		return
	}
	thenState := f.state()
	elseState := -1
	if s.Else != nil {
		elseState = f.state()
	}
	end := f.state()
	next := end
	if elseState >= 0 {
		next = elseState
	}
	f.emit(&ir.If{
		Cond: cond,
		Then: &ir.Block{List: []ir.Stmt{&ir.Jump{State: thenState}}},
		Else: &ir.Jump{State: next},
	})
	f.cur = thenState
	f.block(s.Then.List)
	f.jump(end)
	if elseState >= 0 {
		f.cur = elseState
		f.stmt(s.Else)
		f.jump(end)
	}
	f.cur = end
}

func (f *flat) forStmt(s *ir.For, label string) {
	if s.Init != nil {
		f.stmt(s.Init)
	}
	top := f.state()
	body := f.state()
	post := f.state()
	end := f.state()
	f.jump(top)
	f.cur = top
	if s.Cond != nil {
		// The declaration of each temporary of the condition runs at the top
		// of each pass, in front of the check.
		f.block(s.CondPre)
		cond := f.value(s.Cond)
		f.emit(&ir.If{
			Cond: &ir.Unary{Op: token.NOT, X: cond, Type: types.Typ[types.Bool]},
			Then: &ir.Block{List: []ir.Stmt{&ir.Jump{State: end}}},
		})
	}
	f.jump(body)
	f.cur = body
	f.loops = append(f.loops, flatLoop{label: label, brk: end, cont: post})
	f.block(s.Body.List)
	f.loops = f.loops[:len(f.loops)-1]
	f.jump(post)
	f.cur = post
	if s.Post != nil {
		f.stmt(s.Post)
	}
	f.jump(top)
	f.cur = end
}

func (f *flat) switchStmt(s *ir.Switch, label string) {
	var tag ir.Expr
	if s.Tag != nil && !isTrueConst(s.Tag) {
		tag = f.temp(f.value(s.Tag))
	}
	end := f.state()
	states := make([]int, len(s.Cases))
	for i := range s.Cases {
		states[i] = f.state()
	}
	def := -1
	for i, c := range s.Cases {
		if len(c.Values) == 0 {
			def = i
			continue
		}
		if !caseNeedsPre(c) {
			f.emit(&ir.If{
				Cond: f.caseCond(tag, c.Values),
				Then: &ir.Block{List: []ir.Stmt{&ir.Jump{State: states[i]}}},
			})
			continue
		}
		// A value that needs a declaration takes a test of its own. A jump
		// leaves the state at the first match, thus no later value runs.
		for k, v := range c.Values {
			if k < len(c.Pre) {
				f.block(c.Pre[k])
			}
			f.emit(&ir.If{
				Cond: f.caseCond(tag, []ir.Expr{v}),
				Then: &ir.Block{List: []ir.Stmt{&ir.Jump{State: states[i]}}},
			})
		}
	}
	if def >= 0 {
		f.jump(states[def])
	} else {
		f.jump(end)
	}
	f.loops = append(f.loops, flatLoop{label: label, brk: end, cont: -1})
	for i, c := range s.Cases {
		f.cur = states[i]
		f.block(c.Body.List)
		if c.Fallthrough && i+1 < len(s.Cases) {
			f.jump(states[i+1])
			continue
		}
		f.jump(end)
	}
	f.loops = f.loops[:len(f.loops)-1]
	f.cur = end
}

// caseNeedsPre reports whether a value of the case needs a declaration in
// front of its test.
func caseNeedsPre(c *ir.Case) bool {
	for _, pre := range c.Pre {
		if len(pre) > 0 {
			return true
		}
	}
	return false
}

// caseCond builds the test of 1 case. A switch with no tag reads each value as
// a condition of its own.
func (f *flat) caseCond(tag ir.Expr, values []ir.Expr) ir.Expr {
	var out ir.Expr
	for _, v := range values {
		test := v
		if tag != nil {
			test = &ir.Binary{Op: token.EQL, X: tag, Y: v, Type: types.Typ[types.Bool]}
		}
		if out == nil {
			out = test
			continue
		}
		out = &ir.Binary{Op: token.LOR, X: out, Y: test, Type: types.Typ[types.Bool]}
	}
	return out
}

func isTrueConst(x ir.Expr) bool {
	c, ok := x.(*ir.Const)
	return ok && c.Value != nil && c.Value.String() == "true"
}

func (f *flat) labeled(s *ir.Labeled) {
	if s.Goto {
		n := f.labels[s.Label]
		f.jump(n)
		f.cur = n
		if inner, ok := s.Stmt.(*ir.Block); ok {
			f.block(inner.List)
			return
		}
		f.stmt(s.Stmt)
		return
	}
	switch inner := s.Stmt.(type) {
	case *ir.For:
		f.forStmt(inner, s.Label)
	case *ir.Switch:
		f.switchStmt(inner, s.Label)
	case *ir.Select:
		f.selectStmt(inner, s.Label)
	default:
		end := f.state()
		f.loops = append(f.loops, flatLoop{label: s.Label, brk: end, cont: -1})
		f.stmt(s.Stmt)
		f.loops = f.loops[:len(f.loops)-1]
		f.jump(end)
		f.cur = end
	}
}

func (f *flat) gotoStmt(s *ir.Goto) {
	n, ok := f.labels[s.Label]
	if !ok {
		f.b.fail(f.pos, "the label %s of a goto statement has no block", s.Label)
		return
	}
	f.jump(n)
}

// branch writes the jump of a break statement or of a continue statement. A
// target outside the machine keeps the Dart statement.
func (f *flat) branch(label string, isBreak bool, s ir.Stmt) {
	for i := len(f.loops) - 1; i >= 0; i-- {
		l := f.loops[i]
		if label != "" && l.label != label {
			continue
		}
		if !isBreak && l.cont < 0 {
			if label == "" {
				continue
			}
			break
		}
		if isBreak {
			f.jump(l.brk)
		} else {
			f.jump(l.cont)
		}
		return
	}
	f.emit(s)
}

func (f *flat) exprStmt(s *ir.ExprStmt) {
	call, ok := s.X.(*ir.Call)
	if !ok || !call.Async {
		f.emit(&ir.ExprStmt{X: f.value(s.X)})
		return
	}
	f.emit(&ir.ExprStmt{X: f.callArgs(call)})
	f.pause(nil)
}

func (f *flat) assign(s *ir.Assign) {
	lhs := s.Lhs
	if blocksExpr(s.Rhs) {
		lhs = f.lvalue(lhs)
	}
	f.emit(&ir.Assign{Lhs: lhs, Rhs: f.value(s.Rhs)})
}

func (f *flat) assignMulti(s *ir.AssignMulti) {
	lhs := s.Lhs
	if blocksExpr(s.Rhs) {
		lhs = make([]ir.Expr, len(s.Lhs))
		for i, l := range s.Lhs {
			lhs[i] = f.lvalue(l)
		}
	}
	f.emit(&ir.AssignMulti{Lhs: lhs, Rhs: f.value(s.Rhs)})
}

func (f *flat) declare(s *ir.Declare) {
	init := f.value(s.Init)
	for _, out := range f.lift(&ir.Declare{Vars: s.Vars, Init: init}) {
		f.emit(out)
	}
}

func (f *flat) send(s *ir.Send) {
	parts := f.operands([]ir.Expr{s.Chan, s.Value})
	f.emit(&ir.Send{Chan: parts[0], Value: parts[1]})
	f.pause(nil)
}

// lvalue writes the parts of a target into variables of the frame. Go
// evaluates an index expression of the target before the value.
func (f *flat) lvalue(x ir.Expr) ir.Expr {
	switch x := x.(type) {
	case *ir.Index:
		return &ir.Index{Pos: x.Pos, X: f.indexBase(x.X), Index: f.temp(f.value(x.Index)), Type: x.Type}
	case *ir.Field:
		return &ir.Field{Pos: x.Pos, X: f.lvalueBase(x.X), Name: x.Name, Type: x.Type}
	case *ir.Unary:
		return &ir.Unary{Op: x.Op, X: f.temp(f.value(x.X)), Type: x.Type}
	}
	return x
}

// indexBase keeps the base of an index write.
//
// A slice, a map, and a pointer each hold a reference. A variable of the frame
// thus names the same target, and the hoist holds the order of Go: the value
// that stands in front of the call reaches the write.
//
// A Dart list holds an array by value, thus each assignment of an array takes
// a copy. A hoist of an array would write into that copy, and the array of the
// program would keep its old element. An array keeps its location.
func (f *flat) indexBase(x ir.Expr) ir.Expr {
	if isArrayType(x.GoType()) {
		return f.lvalueBase(x)
	}
	return f.temp(f.value(x))
}

// isArrayType reports whether the core type of t is an array.
func isArrayType(t types.Type) bool {
	if t == nil {
		return false
	}
	_, ok := typeset.Core(t).(*types.Array)
	return ok
}

// lvalueBase keeps the location of the base of a field write. A name is a
// stable location. A variable of the frame holds a copy of a struct value,
// thus a hoist of such a base would take the write onto the copy. A slice, an
// array, and a pointer each share their target through a copy, thus a hoist
// of them stays safe.
func (f *flat) lvalueBase(x ir.Expr) ir.Expr {
	switch x := x.(type) {
	case *ir.Ident:
		return x
	case *ir.Field:
		return &ir.Field{Pos: x.Pos, X: f.lvalueBase(x.X), Name: x.Name, Type: x.Type}
	case *ir.Index:
		return &ir.Index{Pos: x.Pos, X: f.indexBase(x.X), Index: f.temp(f.value(x.Index)), Type: x.Type}
	case *ir.Unary:
		return &ir.Unary{Op: x.Op, X: f.temp(f.value(x.X)), Type: x.Type}
	}
	return f.temp(f.value(x))
}

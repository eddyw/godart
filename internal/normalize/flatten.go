package normalize

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/eddyw/godart/internal/ir"
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
	blocking, defers bool, class string, pos token.Pos) *ir.Machine {
	m := &ir.Machine{Class: class, Recv: recv, Defer: defers, Blocking: blocking}
	f := &flat{b: b, m: m, pos: pos, labels: map[string]int{}}
	if recv != nil {
		m.Locals = append(m.Locals, recv)
	}
	m.Locals = append(m.Locals, params...)
	f.newState()
	f.markLabels(body.List)
	f.block(body.List)
	f.close()
	return m
}

type flat struct {
	b      *builder
	m      *ir.Machine
	cur    int
	pos    token.Pos
	seq    int
	loops  []flatLoop
	labels map[string]int
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
		f.emit(&ir.Return{})
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
		f.emit(&ir.Go{Call: f.callArgs(s.Call)})
	case *ir.Defer:
		f.emit(&ir.Defer{Call: f.callArgs(s.Call)})
	default:
		f.b.fail(f.pos, "the statement %T has no state machine form", s)
	}
}

// plain adds a statement that holds no operation that blocks. A declaration
// still becomes a variable of the frame, because a later state reads it.
func (f *flat) plain(s ir.Stmt) {
	d, ok := s.(*ir.Declare)
	if !ok {
		f.emit(s)
		return
	}
	for _, out := range f.lift(d) {
		f.emit(out)
	}
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
func (f *flat) temp(x ir.Expr) ir.Expr {
	switch x.(type) {
	case *ir.Const, *ir.Zero, *ir.Blank, nil:
		return x
	}
	v := f.local(x.GoType())
	f.emit(&ir.Assign{Lhs: identOf(v), Rhs: x})
	return identOf(v)
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
		f.emit(&ir.If{
			Cond: f.caseCond(tag, c.Values),
			Then: &ir.Block{List: []ir.Stmt{&ir.Jump{State: states[i]}}},
		})
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
		return &ir.Index{X: f.temp(f.value(x.X)), Index: f.temp(f.value(x.Index)), Type: x.Type}
	case *ir.Field:
		return &ir.Field{X: f.temp(f.value(x.X)), Name: x.Name, Type: x.Type}
	case *ir.Unary:
		return &ir.Unary{Op: x.Op, X: f.temp(f.value(x.X)), Type: x.Type}
	}
	return x
}

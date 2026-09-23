package normalize

import (
	"go/token"
	"go/types"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
)

// This file holds the expression side of the state machine. A call that blocks
// must stand alone as a statement, because the machine takes a new state after
// it. An operand that stands in front of such a call and holds a call of its
// own takes a variable, thus the 2 calls keep their order.
//
// A plain read keeps its place. gc reads such an operand after each call of
// the statement, and `order.go` holds the rule for the path that blocks no
// more.

// value rewrites an expression so that it holds no operation that blocks.
func (f *flat) value(x ir.Expr) ir.Expr {
	if x == nil || !blocksExpr(x) {
		return x
	}
	switch x := x.(type) {
	case *ir.Call:
		return f.call(x)
	case *ir.Recv:
		return f.recv(x)
	case *ir.Binary:
		if x.Op == token.LAND || x.Op == token.LOR {
			return f.shortCircuit(x)
		}
		parts := f.operands([]ir.Expr{x.X, x.Y})
		return &ir.Binary{Op: x.Op, X: parts[0], Y: parts[1], Type: x.Type}
	case *ir.Unary:
		return &ir.Unary{Op: x.Op, X: f.value(x.X), Type: x.Type}
	case *ir.Conv:
		return &ir.Conv{X: f.value(x.X), From: x.From, Type: x.Type}
	case *ir.Copy:
		return &ir.Copy{X: f.value(x.X), Type: x.Type}
	case *ir.Field:
		return &ir.Field{Pos: x.Pos, X: f.value(x.X), Name: x.Name, Type: x.Type}
	case *ir.Index:
		parts := f.operands([]ir.Expr{x.X, x.Index})
		return &ir.Index{Pos: x.Pos, X: parts[0], Index: parts[1], Type: x.Type}
	case *ir.MapRead:
		parts := f.operands([]ir.Expr{x.X, x.Key})
		return &ir.MapRead{X: parts[0], Key: parts[1], Type: x.Type}
	case *ir.Slice:
		parts := f.operands([]ir.Expr{x.X, x.Low, x.High, x.Max})
		return &ir.Slice{Pos: x.Pos, X: parts[0], Low: parts[1], High: parts[2], Max: parts[3], Type: x.Type}
	case *ir.Composite:
		out := &ir.Composite{Keys: x.Keys, Elems: f.operands(x.Elems),
			Indices: x.Indices, Length: x.Length, Type: x.Type, ResultsVar: x.ResultsVar}
		if x.Results != nil {
			out.Results = f.value(x.Results)
		}
		return out
	case *ir.Builtin:
		out := &ir.Builtin{Name: x.Name, Args: f.operands(x.Args),
			Type: x.Type, Spread: x.Spread, ResultsVar: x.ResultsVar}
		if x.Results != nil {
			out.Results = f.value(x.Results)
		}
		return out
	case *ir.Record:
		return &ir.Record{Elems: f.operands(x.Elems), Type: x.Type}
	case *ir.RecordField:
		return &ir.RecordField{X: f.value(x.X), Index: x.Index, Type: x.Type}
	case *ir.TypeAssert:
		return &ir.TypeAssert{X: f.value(x.X), Assert: x.Assert, Comma: x.Comma, Type: x.Type}
	case *ir.TypeTest:
		return &ir.TypeTest{X: f.value(x.X), Assert: x.Assert, Type: x.Type}
	case *ir.Method:
		return &ir.Method{Recv: f.value(x.Recv), Obj: x.Obj, Type: x.Type, TypeArgs: x.TypeArgs}
	}
	return x
}

// operands rewrites each expression of a list up to the last one that blocks.
//
// An operand in front of that one takes a variable only when it holds a call
// that stays in place. The machine runs the call that blocks in a state of its
// own, thus such a call would else run second. Each other operand keeps its
// place and reads its value after the state, exactly as gc does.
func (f *flat) operands(list []ir.Expr) []ir.Expr {
	last := -1
	for i, x := range list {
		if x != nil && blocksExpr(x) {
			last = i
		}
	}
	if last < 0 {
		return list
	}
	out := make([]ir.Expr, len(list))
	for i, x := range list {
		switch {
		case x == nil:
			out[i] = nil
		case i < last && holdsSyncCall(x):
			out[i] = f.temp(f.value(x))
		case i <= last:
			out[i] = f.value(x)
		default:
			out[i] = x
		}
	}
	return out
}

// holdsSyncCall reports whether the expression holds a call that stays in
// place. A call that blocks takes a state of its own, thus the machine
// already runs it in front of the statement. The walk stops at a function
// literal, because such a literal is its own function.
func holdsSyncCall(x ir.Expr) bool {
	switch x := x.(type) {
	case nil:
		return false
	case *ir.Call:
		if !x.Async {
			return true
		}
		return holdsSyncCall(x.Fun) || holdsSyncCall(x.Results) || holdsSyncCallAny(x.Args)
	case *ir.Builtin:
		return holdsSyncCall(x.Results) || holdsSyncCallAny(x.Args)
	case *ir.Method:
		return holdsSyncCall(x.Recv)
	case *ir.Recv:
		return holdsSyncCall(x.Chan)
	case *ir.Binary:
		return holdsSyncCall(x.X) || holdsSyncCall(x.Y)
	case *ir.Unary:
		return holdsSyncCall(x.X)
	case *ir.Conv:
		return holdsSyncCall(x.X)
	case *ir.Copy:
		return holdsSyncCall(x.X)
	case *ir.Field:
		return holdsSyncCall(x.X)
	case *ir.Index:
		return holdsSyncCall(x.X) || holdsSyncCall(x.Index)
	case *ir.MapRead:
		return holdsSyncCall(x.X) || holdsSyncCall(x.Key)
	case *ir.Slice:
		return holdsSyncCall(x.X) || holdsSyncCall(x.Low) ||
			holdsSyncCall(x.High) || holdsSyncCall(x.Max)
	case *ir.Composite:
		return holdsSyncCall(x.Results) || holdsSyncCallAny(x.Elems)
	case *ir.Record:
		return holdsSyncCallAny(x.Elems)
	case *ir.RecordField:
		return holdsSyncCall(x.X)
	case *ir.TypeAssert:
		return holdsSyncCall(x.X)
	case *ir.TypeTest:
		return holdsSyncCall(x.X)
	}
	return false
}

func holdsSyncCallAny(list []ir.Expr) bool {
	for _, x := range list {
		if holdsSyncCall(x) {
			return true
		}
	}
	return false
}

// hoistRecv writes the receiver of a method call into a variable of the
// frame, because an argument of the call blocks and the receiver must keep
// its place in the order of the evaluation.
//
// A method with a pointer receiver reads a location, and a copy of the value
// has no address. The variable then holds the address, and the call reads the
// location through it. A struct and an array need the address too: a variable
// of such a type takes a copy at each assignment, thus a temporary of the
// value type would hide each write of the method.
func (f *flat) hoistRecv(m *ir.Method, recv ir.Expr) ir.Expr {
	t := recv.GoType()
	sig, ok := m.Obj.Type().(*types.Signature)
	pointer := ok && sig.Recv() != nil &&
		isPointerUnder(sig.Recv().Type()) && !isPointerUnder(t)
	if pointer {
		return f.temp(&ir.Unary{Op: token.AND, X: recv, Type: types.NewPointer(t)})
	}
	return f.temp(recv)
}

// isPointerUnder reports whether the core type is a pointer.
func isPointerUnder(t types.Type) bool {
	_, ok := typeset.Core(t).(*types.Pointer)
	return ok
}

// call writes a call that blocks into a variable of the frame.
func (f *flat) call(x *ir.Call) ir.Expr {
	out := f.callArgs(x)
	call, _ := out.(*ir.Call)
	if call == nil || !call.Async {
		return out
	}
	v := f.local(call.Type)
	f.emit(&ir.Assign{Lhs: identOf(v), Rhs: call})
	f.pause(v)
	return f.readLocal(v)
}

// callArgs rewrites the callee and each argument of a call. The call itself
// keeps its form.
func (f *flat) callArgs(x ir.Expr) ir.Expr {
	call, ok := x.(*ir.Call)
	if !ok {
		return f.value(x)
	}
	argsBlock := false
	for _, a := range call.Args {
		if blocksExpr(a) {
			argsBlock = true
		}
	}
	out := *call
	if m, ok := call.Fun.(*ir.Method); ok {
		recv := f.value(m.Recv)
		if argsBlock {
			recv = f.hoistRecv(m, recv)
		}
		out.Fun = &ir.Method{Recv: recv, Obj: m.Obj, Type: m.Type, TypeArgs: m.TypeArgs}
	} else if blocksExpr(call.Fun) {
		out.Fun = f.value(call.Fun)
	}
	out.Args = f.operands(call.Args)
	if call.Results != nil {
		out.Results = f.value(call.Results)
	}
	return &out
}

// recv writes a channel read into a variable of the frame. The variable holds
// the value and the flag of the comma form.
func (f *flat) recv(x *ir.Recv) ir.Expr {
	ch := f.value(x.Chan)
	elem := x.Type
	if x.Comma {
		if t, ok := x.Type.(*types.Tuple); ok && t.Len() == 2 {
			elem = t.At(0).Type()
		}
	}
	pair := types.NewTuple(anonVar(elem), anonVar(types.Typ[types.Bool]))
	v := f.local(pair)
	f.emit(&ir.Assign{
		Lhs: identOf(v),
		Rhs: &ir.Recv{Chan: ch, Comma: true, Type: pair},
	})
	f.pause(v)
	if x.Comma {
		return identOf(v)
	}
	return &ir.RecordField{X: identOf(v), Index: 0, Type: elem}
}

// shortCircuit rewrites a logical operator whose right side blocks. Go runs
// that side only for 1 value of the left side, thus the machine needs a
// condition and not a variable.
func (f *flat) shortCircuit(x *ir.Binary) ir.Expr {
	v := f.local(types.Typ[types.Bool])
	f.emit(&ir.Assign{Lhs: identOf(v), Rhs: f.value(x.X)})
	if !blocksExpr(x.Y) {
		return &ir.Binary{Op: x.Op, X: identOf(v), Y: x.Y, Type: x.Type}
	}
	cond := identOf(v)
	if x.Op == token.LOR {
		cond = &ir.Unary{Op: token.NOT, X: cond, Type: types.Typ[types.Bool]}
	}
	then := f.state()
	end := f.state()
	f.emit(&ir.If{
		Cond: cond,
		Then: &ir.Block{List: []ir.Stmt{&ir.Jump{State: then}}},
		Else: &ir.Jump{State: end},
	})
	f.cur = then
	f.emit(&ir.Assign{Lhs: identOf(v), Rhs: f.value(x.Y)})
	f.jump(end)
	f.cur = end
	return identOf(v)
}

// selectStmt writes a select statement. The runtime gives the number of the
// case that it takes, and each body becomes a state.
func (f *flat) selectStmt(s *ir.Select, label string) {
	cases := make([]*ir.SelectClause, len(s.Cases))
	hasDefault := false
	for i, c := range s.Cases {
		clause := *c
		if c.Kind == ir.SelectDefault {
			hasDefault = true
			cases[i] = &clause
			continue
		}
		// A later case can change an earlier channel or send value.
		clause.Chan = f.temp(f.value(c.Chan))
		clause.Value = f.temp(f.value(c.Value))
		cases[i] = &clause
	}
	list := f.local(nil)
	list.Dart = "List<SelectCase>"
	list.DartZero = "<SelectCase>[]"
	f.emit(&ir.Assign{Lhs: identOf(list), Rhs: &ir.SelectList{Cases: cases}})
	choice := f.local(types.Typ[types.Int])
	f.emit(&ir.Assign{
		Lhs: identOf(choice),
		Rhs: &ir.SelectRun{
			List:       identOf(list),
			HasDefault: hasDefault,
			Type:       types.Typ[types.Int],
		},
	})
	f.pause(choice).Select = true
	end := f.state()
	states := make([]int, len(cases))
	for i := range cases {
		states[i] = f.state()
	}
	index := 0
	def := -1
	for i, c := range cases {
		if c.Kind == ir.SelectDefault {
			def = i
			continue
		}
		f.emit(&ir.If{
			Cond: &ir.Binary{
				Op:   token.EQL,
				X:    identOf(choice),
				Y:    intConst(int64(index)),
				Type: types.Typ[types.Bool],
			},
			Then: &ir.Block{List: []ir.Stmt{&ir.Jump{State: states[i]}}},
		})
		index++
	}
	if def >= 0 {
		f.jump(states[def])
	} else {
		f.jump(end)
	}
	f.loops = append(f.loops, flatLoop{label: label, brk: end, cont: -1})
	index = 0
	for i, c := range cases {
		f.cur = states[i]
		if c.Kind != ir.SelectDefault {
			if c.Kind == ir.SelectRecv {
				f.keep(c.Vars)
				f.emit(&ir.SelectBind{List: identOf(list), Index: index, Clause: c})
			}
			index++
		}
		f.block(c.Body.List)
		f.jump(end)
	}
	f.loops = f.loops[:len(f.loops)-1]
	f.cur = end
}

// split reports whether a statement must become states of its own.
func (f *flat) split(s ir.Stmt) bool {
	if f.m.Blocking && holdsLoop(s) {
		return true
	}
	switch s.(type) {
	case *ir.Goto, *ir.Break, *ir.Continue, *ir.Return:
		return true
	}
	if blocksStmt(s) || holdsGoto(s) {
		return true
	}
	if f.m.DeferBlocks && holdsReturn(s) {
		return true
	}
	return len(f.loops) > 0 && escapes(s)
}

// splitBlock reports whether the statements of a block must become states.
func (f *flat) splitBlock(b *ir.Block) bool {
	if b == nil {
		return false
	}
	for _, s := range b.List {
		if f.m.Blocking && holdsLoop(s) {
			return true
		}
		if blocksStmt(s) || holdsGoto(s) {
			return true
		}
		if f.m.DeferBlocks && holdsReturn(s) {
			return true
		}
		if len(f.loops) > 0 && escapes(s) {
			return true
		}
	}
	return false
}

func holdsLoop(s ir.Stmt) bool {
	switch s := s.(type) {
	case *ir.For:
		return true
	case *ir.Labeled:
		return holdsLoop(s.Stmt)
	case *ir.Block:
		for _, child := range s.List {
			if holdsLoop(child) {
				return true
			}
		}
	case *ir.If:
		return holdsLoop(s.Then) || holdsLoop(s.Else)
	case *ir.Switch:
		for _, c := range s.Cases {
			if holdsLoop(c.Body) {
				return true
			}
		}
	case *ir.Select:
		for _, c := range s.Cases {
			if holdsLoop(c.Body) {
				return true
			}
		}
	}
	return false
}

// holdsReturn reports whether a statement holds a return statement.
//
// A machine whose deferred calls can block runs the chain in states of its
// own. A return must jump to that chain. A return inside a plain statement
// writes a Dart return, and each deferred call then runs no more. The walk
// stops at a function literal, because such a literal is its own function.
func holdsReturn(s ir.Stmt) bool {
	switch s := s.(type) {
	case nil:
		return false
	case *ir.Return:
		return true
	case *ir.Labeled:
		return holdsReturn(s.Stmt)
	case *ir.Block:
		return holdsReturnList(s.List)
	case *ir.If:
		return holdsReturnList(s.Then.List) || holdsReturn(s.Else)
	case *ir.For:
		return holdsReturnList(s.Body.List)
	case *ir.Switch:
		for _, c := range s.Cases {
			if holdsReturnList(c.Body.List) {
				return true
			}
		}
	case *ir.Select:
		for _, c := range s.Cases {
			if holdsReturnList(c.Body.List) {
				return true
			}
		}
	}
	return false
}

func holdsReturnList(list []ir.Stmt) bool {
	for _, s := range list {
		if holdsReturn(s) {
			return true
		}
	}
	return false
}

// blocksStmt reports whether a statement holds an operation that blocks. The
// walk stops at a function literal, because such a literal is its own
// function.
func blocksStmt(s ir.Stmt) bool {
	switch s := s.(type) {
	case nil:
		return false
	case *ir.Block:
		return blocksList(s.List)
	case *ir.ExprStmt:
		return blocksExpr(s.X)
	case *ir.Declare:
		return blocksExpr(s.Init)
	case *ir.Assign:
		return blocksExpr(s.Lhs) || blocksExpr(s.Rhs)
	case *ir.AssignMulti:
		for _, l := range s.Lhs {
			if blocksExpr(l) {
				return true
			}
		}
		return blocksExpr(s.Rhs)
	case *ir.If:
		return blocksExpr(s.Cond) || blocksList(s.Then.List) || blocksStmt(s.Else)
	case *ir.For:
		return blocksStmt(s.Init) || blocksList(s.CondPre) || blocksExpr(s.Cond) ||
			blocksStmt(s.Post) || blocksList(s.Body.List)
	case *ir.Return:
		for _, v := range s.Values {
			if blocksExpr(v) {
				return true
			}
		}
		return false
	case *ir.Labeled:
		return blocksStmt(s.Stmt)
	case *ir.Switch:
		if blocksExpr(s.Tag) {
			return true
		}
		for _, c := range s.Cases {
			for _, v := range c.Values {
				if blocksExpr(v) {
					return true
				}
			}
			for _, pre := range c.Pre {
				if blocksList(pre) {
					return true
				}
			}
			if blocksList(c.Body.List) {
				return true
			}
		}
		return false
	case *ir.Send:
		return true
	case *ir.Select:
		return true
	case *ir.Go:
		return blocksArgs(s.Call)
	case *ir.Defer:
		return blocksArgs(s.Call)
	}
	return false
}

func blocksList(list []ir.Stmt) bool {
	for _, s := range list {
		if blocksStmt(s) {
			return true
		}
	}
	return false
}

// blocksArgs reads the arguments of a call that runs later. Go evaluates each
// argument of a go statement and of a defer statement at the statement.
func blocksArgs(x ir.Expr) bool {
	call, ok := x.(*ir.Call)
	if !ok {
		return blocksExpr(x)
	}
	for _, a := range call.Args {
		if blocksExpr(a) {
			return true
		}
	}
	return false
}

// blocksExpr reports whether an expression holds an operation that blocks.
func blocksExpr(x ir.Expr) bool {
	switch x := x.(type) {
	case nil:
		return false
	case *ir.Call:
		if x.Async {
			return true
		}
		if blocksExpr(x.Fun) || blocksExpr(x.Results) {
			return true
		}
		return blocksAny(x.Args)
	case *ir.Recv:
		return true
	case *ir.SelectRun:
		return true
	case *ir.Method:
		return blocksExpr(x.Recv)
	case *ir.Binary:
		return blocksExpr(x.X) || blocksExpr(x.Y)
	case *ir.Unary:
		return blocksExpr(x.X)
	case *ir.Conv:
		return blocksExpr(x.X)
	case *ir.Copy:
		return blocksExpr(x.X)
	case *ir.Field:
		return blocksExpr(x.X)
	case *ir.Index:
		return blocksExpr(x.X) || blocksExpr(x.Index)
	case *ir.MapRead:
		return blocksExpr(x.X) || blocksExpr(x.Key)
	case *ir.Slice:
		return blocksExpr(x.X) || blocksExpr(x.Low) || blocksExpr(x.High) || blocksExpr(x.Max)
	case *ir.Composite:
		return blocksExpr(x.Results) || blocksAny(x.Elems) || blocksAny(x.Keys)
	case *ir.Builtin:
		return blocksExpr(x.Results) || blocksAny(x.Args)
	case *ir.Record:
		return blocksAny(x.Elems)
	case *ir.RecordField:
		return blocksExpr(x.X)
	case *ir.TypeAssert:
		return blocksExpr(x.X)
	case *ir.TypeTest:
		return blocksExpr(x.X)
	}
	return false
}

func blocksAny(list []ir.Expr) bool {
	for _, x := range list {
		if blocksExpr(x) {
			return true
		}
	}
	return false
}

// holdsGoto reports whether a statement holds a goto statement or the label of
// one. Such a statement needs a state of its own.
func holdsGoto(s ir.Stmt) bool {
	switch s := s.(type) {
	case nil:
		return false
	case *ir.Goto:
		return true
	case *ir.Labeled:
		return s.Goto || holdsGoto(s.Stmt)
	case *ir.Block:
		return holdsGotoList(s.List)
	case *ir.If:
		return holdsGotoList(s.Then.List) || holdsGoto(s.Else)
	case *ir.For:
		return holdsGotoList(s.Body.List)
	case *ir.Switch:
		for _, c := range s.Cases {
			if holdsGotoList(c.Body.List) {
				return true
			}
		}
	case *ir.Select:
		for _, c := range s.Cases {
			if holdsGotoList(c.Body.List) {
				return true
			}
		}
	}
	return false
}

func holdsGotoList(list []ir.Stmt) bool {
	for _, s := range list {
		if holdsGoto(s) {
			return true
		}
	}
	return false
}

// escapes reports whether a branch statement inside s leaves s. Such a branch
// reaches a construct of the machine, thus s needs states of its own.
func escapes(s ir.Stmt) bool {
	return leaves(s, map[string]bool{}, true, true)
}

// leaves walks a statement. inner holds each label that s declares. brk marks
// a position where an unlabelled break leaves s, and cont does the same for an
// unlabelled continue.
func leaves(s ir.Stmt, inner map[string]bool, brk, cont bool) bool {
	switch s := s.(type) {
	case nil:
		return false
	case *ir.Break:
		if s.Label == "" {
			return brk
		}
		return !inner[s.Label]
	case *ir.Continue:
		if s.Label == "" {
			return cont
		}
		return !inner[s.Label]
	case *ir.Block:
		return leavesList(s.List, inner, brk, cont)
	case *ir.If:
		return leavesList(s.Then.List, inner, brk, cont) || leaves(s.Else, inner, brk, cont)
	case *ir.For:
		return leavesList(s.Body.List, inner, false, false)
	case *ir.Switch:
		for _, c := range s.Cases {
			if leavesList(c.Body.List, inner, false, cont) {
				return true
			}
		}
	case *ir.Select:
		for _, c := range s.Cases {
			if leavesList(c.Body.List, inner, false, cont) {
				return true
			}
		}
	case *ir.Labeled:
		next := map[string]bool{s.Label: true}
		for k := range inner {
			next[k] = true
		}
		return leaves(s.Stmt, next, brk, cont)
	}
	return false
}

func leavesList(list []ir.Stmt, inner map[string]bool, brk, cont bool) bool {
	for _, s := range list {
		if leaves(s, inner, brk, cont) {
			return true
		}
	}
	return false
}

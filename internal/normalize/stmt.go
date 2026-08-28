package normalize

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
	"github.com/eddyw/godart/internal/typeset"
)

func (b *builder) block(s *ast.BlockStmt) *ir.Block {
	out := &ir.Block{}
	if s == nil {
		return out
	}
	for _, st := range s.List {
		out.List = append(out.List, b.stmt(st)...)
	}
	return out
}

// stmt builds 1 statement. A function literal that can block becomes a local
// function, and its declaration stands in front of the statement that holds
// it.
func (b *builder) stmt(s ast.Stmt) []ir.Stmt {
	out := b.stmtOf(s)
	if len(b.pending) == 0 {
		return out
	}
	pending := b.pending
	b.pending = nil
	return append(pending, out...)
}

func (b *builder) stmtOf(s ast.Stmt) []ir.Stmt {
	switch s := s.(type) {
	case nil, *ast.EmptyStmt:
		return nil
	case *ast.BlockStmt:
		return []ir.Stmt{b.block(s)}
	case *ast.ExprStmt:
		return []ir.Stmt{&ir.ExprStmt{X: b.expr(s.X)}}
	case *ast.DeclStmt:
		return b.declStmt(s)
	case *ast.AssignStmt:
		return b.assign(s)
	case *ast.IncDecStmt:
		return b.incDec(s)
	case *ast.IfStmt:
		return b.ifStmt(s)
	case *ast.ForStmt:
		return b.forStmt(s, "")
	case *ast.RangeStmt:
		return b.rangeStmt(s, "")
	case *ast.SwitchStmt:
		return b.switchStmt(s, "")
	case *ast.TypeSwitchStmt:
		return b.typeSwitch(s, "")
	case *ast.ReturnStmt:
		return b.returnStmt(s)
	case *ast.BranchStmt:
		return b.branch(s)
	case *ast.LabeledStmt:
		return b.labeled(s)
	case *ast.DeferStmt:
		return b.deferStmt(s)
	case *ast.GoStmt:
		return b.goStmt(s)
	case *ast.SendStmt:
		return []ir.Stmt{&ir.Send{Chan: b.expr(s.Chan), Value: b.copyPoint(b.exprAs(s.Value, chanElem(b.typeOf(s.Chan))))}}
	case *ast.SelectStmt:
		return b.selectStmt(s, "")
	}
	b.fail(s.Pos(), "the statement %T is outside the subset", s)
	return nil
}

func (b *builder) labeled(s *ast.LabeledStmt) []ir.Stmt {
	label := name.Local(s.Label.Name)
	if b.gotoLabels[s.Label.Name] {
		// A goto reaches this label, thus the label marks the start of a block
		// of the state machine that stage 4 builds.
		return []ir.Stmt{&ir.Labeled{Label: label, Goto: true, Stmt: &ir.Block{List: b.stmt(s.Stmt)}}}
	}
	switch inner := s.Stmt.(type) {
	case *ast.ForStmt:
		return b.forStmt(inner, label)
	case *ast.RangeStmt:
		return b.rangeStmt(inner, label)
	case *ast.SwitchStmt:
		return b.switchStmt(inner, label)
	case *ast.TypeSwitchStmt:
		return b.typeSwitch(inner, label)
	case *ast.SelectStmt:
		return b.selectStmt(inner, label)
	}
	body := b.stmt(s.Stmt)
	if len(body) == 0 {
		return []ir.Stmt{&ir.Labeled{Label: label, Stmt: &ir.Block{}}}
	}
	return []ir.Stmt{&ir.Labeled{Label: label, Stmt: &ir.Block{List: body}}}
}

func (b *builder) branch(s *ast.BranchStmt) []ir.Stmt {
	label := ""
	if s.Label != nil {
		label = name.Local(s.Label.Name)
	}
	switch s.Tok {
	case token.BREAK, token.CONTINUE:
		return b.breakOrContinue(s, label)
	case token.GOTO:
		return []ir.Stmt{&ir.Goto{Label: label}}
	}
	b.fail(s.Pos(), "fallthrough must be the last statement of a case")
	return nil
}

func (b *builder) declStmt(s *ast.DeclStmt) []ir.Stmt {
	d, ok := s.Decl.(*ast.GenDecl)
	if !ok {
		return nil
	}
	var out []ir.Stmt
	for _, spec := range d.Specs {
		switch spec := spec.(type) {
		case *ast.TypeSpec:
			// A local struct type takes the Dart class of its shape, exactly as
			// an unnamed struct type does, thus the declaration needs no output.
			if !b.localStructType(spec) {
				b.fail(spec.Pos(), "a type declaration inside a function is "+
					"outside the subset, and only a struct type is")
			}
		case *ast.ValueSpec:
			out = append(out, b.localValueSpec(spec, d.Tok)...)
		}
	}
	return out
}

func (b *builder) localValueSpec(spec *ast.ValueSpec, tok token.Token) []ir.Stmt {
	if tok == token.CONST {
		return nil
	}
	var out []ir.Stmt
	if len(spec.Values) == 0 {
		for _, n := range spec.Names {
			obj, _ := b.info.Defs[n].(*types.Var)
			if obj == nil {
				continue
			}
			v := b.declareLocal(obj, obj.Name())
			out = append(out, &ir.Declare{Vars: []*ir.Var{v}})
		}
		return out
	}
	if len(spec.Values) == len(spec.Names) {
		for i, n := range spec.Names {
			obj, _ := b.info.Defs[n].(*types.Var)
			if obj == nil {
				continue
			}
			init := b.copyPoint(b.exprAs(spec.Values[i], obj.Type()))
			v := b.declareLocal(obj, obj.Name())
			out = append(out, &ir.Declare{Vars: []*ir.Var{v}, Init: init})
		}
		return out
	}
	init := b.multiValue(spec.Values[0])
	vars := make([]*ir.Var, 0, len(spec.Names))
	for _, n := range spec.Names {
		obj, _ := b.info.Defs[n].(*types.Var)
		if obj == nil {
			continue
		}
		vars = append(vars, b.declareLocal(obj, obj.Name()))
	}
	return append(out, &ir.Declare{Vars: vars, Init: init})
}

func (b *builder) incDec(s *ast.IncDecStmt) []ir.Stmt {
	pre, get, set := b.lvalue(s.X)
	op := token.ADD
	if s.Tok == token.DEC {
		op = token.SUB
	}
	one := &ir.Const{Value: constant.MakeInt64(1), Type: get.GoType()}
	return append(pre, set(&ir.Binary{Op: op, X: get, Y: one, Type: get.GoType()}))
}

func (b *builder) assign(s *ast.AssignStmt) []ir.Stmt {
	if op, ok := opAssign[s.Tok]; ok {
		pre, get, set := b.lvalue(s.Lhs[0])
		y := b.exprAs(s.Rhs[0], shiftOperand(op, get.GoType(), b.typeOf(s.Rhs[0])))
		return append(pre, set(&ir.Binary{Op: op, X: get, Y: y, Type: get.GoType()}))
	}
	if len(s.Lhs) > 1 && len(s.Rhs) == 1 {
		return b.assignMulti(s)
	}
	if s.Tok == token.DEFINE {
		var out []ir.Stmt
		for i, lhs := range s.Lhs {
			id, _ := lhs.(*ast.Ident)
			obj, _ := b.info.Defs[id].(*types.Var)
			if obj == nil {
				out = append(out, b.assignOne(lhs, s.Rhs[i])...)
				continue
			}
			init := b.copyPoint(b.exprAs(s.Rhs[i], obj.Type()))
			v := b.declareLocal(obj, obj.Name())
			out = append(out, &ir.Declare{Vars: []*ir.Var{v}, Init: init})
		}
		return out
	}
	if len(s.Lhs) == 1 {
		return b.assignOne(s.Lhs[0], s.Rhs[0])
	}
	// Go evaluates each value on the right before it writes a location, thus
	// the stage hoists the values into temporary variables.
	var out []ir.Stmt
	tmps := make([]ir.Expr, len(s.Rhs))
	for i, rhs := range s.Rhs {
		t := b.typeOf(s.Lhs[i])
		v := b.tempVar(rhs.Pos(), "a", t)
		out = append(out, &ir.Declare{Vars: []*ir.Var{v}, Init: b.exprAs(rhs, t)})
		tmps[i] = &ir.Ident{Name: v.Name, Type: t}
	}
	for i, lhs := range s.Lhs {
		pre, _, set := b.lvalue(lhs)
		out = append(out, pre...)
		out = append(out, set(b.copyPoint(tmps[i])))
	}
	return out
}

// assignMulti writes the parts of 1 expression with multiple values.
//
// A short variable declaration may hold a name that already exists. Go declares
// only the new names, thus each name needs its own form.
func (b *builder) assignMulti(s *ast.AssignStmt) []ir.Stmt {
	rhs := b.multiValue(s.Rhs[0])
	var news []*types.Var
	if s.Tok == token.DEFINE {
		news = make([]*types.Var, len(s.Lhs))
		allNew := true
		for i, lhs := range s.Lhs {
			id, _ := lhs.(*ast.Ident)
			news[i], _ = b.info.Defs[id].(*types.Var)
			if news[i] == nil {
				allNew = false
			}
		}
		if allNew {
			vars := make([]*ir.Var, len(s.Lhs))
			for i, obj := range news {
				vars[i] = b.declareLocal(obj, obj.Name())
			}
			return []ir.Stmt{&ir.Declare{Vars: vars, Init: rhs}}
		}
	}
	var out []ir.Stmt
	tmps := make([]*ir.Var, len(s.Lhs))
	for i, lhs := range s.Lhs {
		tmps[i] = &ir.Var{Name: b.temp(s.Pos(), "m") + strconv.Itoa(i), Type: b.typeOf(lhs)}
	}
	out = append(out, &ir.Declare{Vars: tmps, Init: rhs})
	for i, lhs := range s.Lhs {
		if isBlank(lhs) {
			continue
		}
		read := &ir.Ident{Name: tmps[i].Name, Type: tmps[i].Type}
		if news != nil && news[i] != nil {
			v := b.declareLocal(news[i], news[i].Name())
			out = append(out, &ir.Declare{Vars: []*ir.Var{v}, Init: read})
			continue
		}
		pre, _, set := b.lvalue(lhs)
		out = append(out, pre...)
		out = append(out, set(read))
	}
	return out
}

func (b *builder) assignOne(lhs, rhs ast.Expr) []ir.Stmt {
	if isBlank(lhs) {
		return []ir.Stmt{&ir.ExprStmt{X: b.expr(rhs)}}
	}
	t := b.typeOf(lhs)
	pre, _, set := b.lvalue(lhs)
	return append(pre, set(b.copyPoint(b.exprAs(rhs, t))))
}

// deferStmt reads the function value and each argument at the point of the
// defer statement, exactly as Go does. The call itself runs later.
func (b *builder) deferStmt(s *ast.DeferStmt) []ir.Stmt {
	if b.fn != nil {
		b.fn.Defers = true
	}
	if b.blocks.CallAsync(s.Call) && b.fn != nil {
		// The chain of the deferred calls holds states of its own, thus such a
		// call gives up control like each other call.
		b.fn.DeferBlocks = true
	}
	call := b.expr(s.Call)
	var pre []ir.Stmt
	switch c := call.(type) {
	case *ir.Call:
		switch fun := c.Fun.(type) {
		case *ir.Method:
			hoisted, recv := b.forceHoist(b.receiverValue(fun), s.Pos(), "dr")
			pre = append(pre, hoisted...)
			c.Fun = &ir.Method{Recv: recv, Obj: fun.Obj, Type: fun.Type}
		default:
			if !directFunc(fun) {
				hoisted, value := b.forceHoist(fun, s.Pos(), "df")
				pre = append(pre, hoisted...)
				c.Fun = value
			}
		}
		pre = append(pre, b.hoistArgs(c.Args, s.Pos())...)
	case *ir.Builtin:
		pre = append(pre, b.hoistArgs(c.Args, s.Pos())...)
	default:
		if call != nil {
			b.fail(s.Pos(), "defer needs a call to a function")
		}
		return nil
	}
	return append(pre, &ir.Defer{Call: call})
}

// goStmt starts a goroutine. Go reads the function value and each argument at
// the point of the statement, exactly as a defer statement does.
func (b *builder) goStmt(s *ast.GoStmt) []ir.Stmt {
	call := b.expr(s.Call)
	var pre []ir.Stmt
	switch c := call.(type) {
	case *ir.Call:
		switch fun := c.Fun.(type) {
		case *ir.Method:
			hoisted, recv := b.forceHoist(b.receiverValue(fun), s.Pos(), "gr")
			pre = append(pre, hoisted...)
			c.Fun = &ir.Method{Recv: recv, Obj: fun.Obj, Type: fun.Type}
		default:
			if !directFunc(fun) {
				hoisted, value := b.forceHoist(fun, s.Pos(), "gf")
				pre = append(pre, hoisted...)
				c.Fun = value
			}
		}
		pre = append(pre, b.hoistArgs(c.Args, s.Pos())...)
	case *ir.Builtin:
		pre = append(pre, b.hoistArgs(c.Args, s.Pos())...)
	default:
		if call != nil {
			b.fail(s.Pos(), "go needs a call to a function")
		}
		return nil
	}
	return append(pre, &ir.Go{Call: call})
}

// selectStmt builds 1 select statement. Each case keeps its own body, and the
// runtime gives the number of the case that it takes.
func (b *builder) selectStmt(s *ast.SelectStmt, label string) []ir.Stmt {
	b.pushLoop(loopSwitch, label)
	defer b.popLoop()
	node := &ir.Select{Label: label}
	var pre []ir.Stmt
	for _, clause := range s.Body.List {
		c, ok := clause.(*ast.CommClause)
		if !ok {
			continue
		}
		out := &ir.SelectClause{Kind: ir.SelectDefault}
		switch comm := c.Comm.(type) {
		case nil:
		case *ast.SendStmt:
			out.Kind = ir.SelectSend
			out.Chan = b.expr(comm.Chan)
			out.Elem = chanElem(b.typeOf(comm.Chan))
			out.Value = b.copyPoint(b.exprAs(comm.Value, out.Elem))
		case *ast.ExprStmt:
			recv, ok := unparenExpr(comm.X).(*ast.UnaryExpr)
			if !ok || recv.Op != token.ARROW {
				b.fail(comm.Pos(), "a select case needs a send or a receive")
				continue
			}
			out.Kind = ir.SelectRecv
			out.Chan = b.expr(recv.X)
			out.Elem = chanElem(b.typeOf(recv.X))
		case *ast.AssignStmt:
			recv, ok := unparenExpr(comm.Rhs[0]).(*ast.UnaryExpr)
			if !ok || recv.Op != token.ARROW {
				b.fail(comm.Pos(), "a select case needs a send or a receive")
				continue
			}
			out.Kind = ir.SelectRecv
			out.Chan = b.expr(recv.X)
			out.Elem = chanElem(b.typeOf(recv.X))
			for _, lhs := range comm.Lhs {
				if comm.Tok == token.DEFINE {
					id, _ := lhs.(*ast.Ident)
					obj, _ := b.info.Defs[id].(*types.Var)
					if obj == nil {
						out.Vars = append(out.Vars, &ir.Var{Name: "_", Type: b.typeOf(lhs)})
						continue
					}
					out.Vars = append(out.Vars, b.declareLocal(obj, obj.Name()))
					continue
				}
				out.Lhs = append(out.Lhs, b.expr(lhs))
			}
		default:
			b.fail(c.Pos(), "a select case needs a send or a receive")
			continue
		}
		body := &ir.Block{}
		for _, st := range c.Body {
			body.List = append(body.List, b.stmt(st)...)
		}
		out.Body = body
		node.Cases = append(node.Cases, out)
	}
	return append(pre, node)
}

// receiverValue reads the receiver of a deferred call or of a go statement. Go
// copies a value receiver at that moment, and it keeps the reference of a
// pointer receiver.
func (b *builder) receiverValue(m *ir.Method) ir.Expr {
	sig, ok := m.Obj.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return m.Recv
	}
	if _, isPointer := types.Unalias(sig.Recv().Type()).(*types.Pointer); isPointer {
		return m.Recv
	}
	return b.copyPoint(m.Recv)
}

// directFunc reports whether the callee names a function or holds a fresh
// closure. Such a callee needs no temporary variable, because no later write
// changes it.
func directFunc(fun ir.Expr) bool {
	switch fun := fun.(type) {
	case *ir.FuncLit:
		return true
	case *ir.Ident:
		_, ok := fun.Obj.(*types.Func)
		return ok
	}
	return false
}

func chanElem(t types.Type) types.Type {
	c, ok := typeset.Core(t).(*types.Chan)
	if !ok {
		return nil
	}
	return c.Elem()
}

// hoistArgs reads each argument at the point of the statement.
func (b *builder) hoistArgs(args []ir.Expr, pos token.Pos) []ir.Stmt {
	var pre []ir.Stmt
	for i, a := range args {
		hoisted, value := b.forceHoist(a, pos, fmt.Sprintf("da%d", i))
		pre = append(pre, hoisted...)
		args[i] = value
	}
	return pre
}

var opAssign = map[token.Token]token.Token{
	token.ADD_ASSIGN: token.ADD, token.SUB_ASSIGN: token.SUB,
	token.MUL_ASSIGN: token.MUL, token.QUO_ASSIGN: token.QUO,
	token.REM_ASSIGN: token.REM, token.AND_ASSIGN: token.AND,
	token.OR_ASSIGN: token.OR, token.XOR_ASSIGN: token.XOR,
	token.SHL_ASSIGN: token.SHL, token.SHR_ASSIGN: token.SHR,
	token.AND_NOT_ASSIGN: token.AND_NOT,
}

// shiftOperand gives the type of the right operand. A shift count keeps its own
// type, and each other operand takes the type of the left operand.
func shiftOperand(op token.Token, left, right types.Type) types.Type {
	if op == token.SHL || op == token.SHR {
		return right
	}
	return left
}

func isBlank(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "_"
}

func (b *builder) ifStmt(s *ast.IfStmt) []ir.Stmt {
	out := b.stmt(s.Init)
	node := &ir.If{Cond: b.expr(s.Cond), Then: b.block(s.Body)}
	if s.Else != nil {
		inner := b.stmt(s.Else)
		switch len(inner) {
		case 0:
		case 1:
			node.Else = inner[0]
		default:
			node.Else = &ir.Block{List: inner}
		}
	}
	return append(out, node)
}

// forStmt writes a loop with 3 clauses.
//
// Go 1.22 gives each iteration its own copy of a variable that the first
// clause declares. A Dart for loop gives the same copy when the declaration
// stands in its header, thus the declaration stays inside the loop. A boxed
// variable holds a reference, and the copy of Dart then shares 1 box, thus the
// third clause takes a new box in front of it.
func (b *builder) forStmt(s *ast.ForStmt, label string) []ir.Stmt {
	out := b.stmt(s.Init)
	loop := &ir.For{Label: label}
	if len(out) == 1 {
		if d, ok := out[0].(*ir.Declare); ok {
			loop.Init = d
			out = nil
		}
	}
	b.pushLoop(loopPlain, label)
	loop.Body = b.block(s.Body)
	b.popLoop()
	if s.Cond != nil {
		loop.Cond = b.expr(s.Cond)
	}
	var post []ir.Stmt
	if d, ok := loop.Init.(*ir.Declare); ok {
		for _, v := range d.Vars {
			if v.Boxed {
				post = append(post, &ir.Assign{
					Lhs:    identOf(v),
					Rhs:    identOf(v),
					Define: true,
				})
			}
		}
	}
	if s.Post != nil {
		post = append(post, b.stmt(s.Post)...)
	}
	if len(post) == 1 {
		loop.Post = post[0]
	} else if len(post) > 1 {
		loop.Post = &ir.Block{List: post}
	}
	return append(out, loop)
}

func (b *builder) returnStmt(s *ast.ReturnStmt) []ir.Stmt {
	if b.insideYield() {
		return b.yieldReturn(s)
	}
	f := b.fn
	if f == nil {
		return nil
	}
	if len(s.Results) == 0 {
		if hasNamedResult(f) {
			values := make([]ir.Expr, len(f.Results))
			for i, r := range f.Results {
				values[i] = &ir.Ident{Name: r.Name, Type: r.Type, Boxed: r.Boxed}
			}
			return []ir.Stmt{&ir.Return{Values: values}}
		}
		return []ir.Stmt{&ir.Return{}}
	}
	if len(s.Results) == 1 && len(f.Results) > 1 {
		return []ir.Stmt{&ir.Return{Values: []ir.Expr{b.multiValue(s.Results[0])}}}
	}
	values := make([]ir.Expr, len(s.Results))
	for i, r := range s.Results {
		values[i] = b.copyPoint(b.exprAs(r, f.Results[i].Type))
	}
	return []ir.Stmt{&ir.Return{Values: values}}
}

// typeSwitch rewrites a type switch into a switch with a type test in each
// case. The test reads the descriptor of the interface value.
func (b *builder) typeSwitch(s *ast.TypeSwitchStmt, label string) []ir.Stmt {
	b.pushLoop(loopSwitch, label)
	defer b.popLoop()
	out := b.stmt(s.Init)
	subject := typeSwitchSubject(s.Assign)
	if subject == nil {
		b.fail(s.Pos(), "the type switch has no subject")
		return out
	}
	t := b.typeOf(subject)
	v := b.tempVar(s.Pos(), "ts", t)
	out = append(out, &ir.Declare{Vars: []*ir.Var{v}, Init: b.expr(subject)})
	read := &ir.Ident{Name: v.Name, Type: t}
	node := &ir.Switch{Label: label}
	for _, clause := range s.Body.List {
		c, ok := clause.(*ast.CaseClause)
		if !ok {
			continue
		}
		cs := &ir.Case{}
		for _, target := range c.List {
			cs.Values = append(cs.Values, b.typeCase(read, target))
		}
		body := &ir.Block{}
		if obj, ok := b.info.Implicits[c].(*types.Var); ok {
			bound := b.declareLocal(obj, obj.Name())
			var init ir.Expr = read
			if len(c.List) == 1 && !isInterface(obj.Type()) && !isNilCase(c.List[0]) {
				init = &ir.TypeAssert{X: read, Assert: obj.Type(), Type: obj.Type()}
			}
			body.List = append(body.List, &ir.Declare{Vars: []*ir.Var{bound}, Init: init})
		}
		for _, st := range c.Body {
			body.List = append(body.List, b.stmt(st)...)
		}
		cs.Body = body
		node.Cases = append(node.Cases, cs)
	}
	return append(out, node)
}

// typeCase writes the test of 1 case. A case that names nil tests the value,
// because a nil interface holds no descriptor.
func (b *builder) typeCase(read ir.Expr, target ast.Expr) ir.Expr {
	if isNilCase(target) {
		return &ir.Binary{Op: token.EQL, X: read, Y: &ir.Zero{Type: read.GoType()},
			Type: types.Typ[types.Bool]}
	}
	return &ir.TypeTest{X: read, Assert: b.typeOf(target), Type: types.Typ[types.Bool]}
}

func isNilCase(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}

func isInterface(t types.Type) bool {
	return isIfaceType(t)
}

func typeSwitchSubject(s ast.Stmt) ast.Expr {
	switch s := s.(type) {
	case *ast.ExprStmt:
		if a, ok := s.X.(*ast.TypeAssertExpr); ok {
			return a.X
		}
	case *ast.AssignStmt:
		if len(s.Rhs) == 1 {
			if a, ok := s.Rhs[0].(*ast.TypeAssertExpr); ok {
				return a.X
			}
		}
	}
	return nil
}

func (b *builder) switchStmt(s *ast.SwitchStmt, label string) []ir.Stmt {
	b.pushLoop(loopSwitch, label)
	defer b.popLoop()
	out := b.stmt(s.Init)
	node := &ir.Switch{Label: label}
	if s.Tag != nil {
		t := b.typeOf(s.Tag)
		v := b.tempVar(s.Tag.Pos(), "tag", t)
		out = append(out, &ir.Declare{Vars: []*ir.Var{v}, Init: b.expr(s.Tag)})
		node.Tag = &ir.Ident{Name: v.Name, Type: t}
	}
	for _, clause := range s.Body.List {
		c, ok := clause.(*ast.CaseClause)
		if !ok {
			continue
		}
		cs := &ir.Case{}
		for _, value := range c.List {
			want := types.Type(nil)
			if node.Tag != nil {
				want = node.Tag.GoType()
			}
			cs.Values = append(cs.Values, b.exprAs(value, want))
		}
		body := c.Body
		if n := len(body); n > 0 {
			if br, ok := body[n-1].(*ast.BranchStmt); ok && br.Tok == token.FALLTHROUGH {
				cs.Fallthrough = true
				body = body[:n-1]
			}
		}
		block := &ir.Block{}
		for _, st := range body {
			block.List = append(block.List, b.stmt(st)...)
		}
		cs.Body = block
		node.Cases = append(node.Cases, cs)
	}
	return append(out, node)
}

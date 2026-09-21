package normalize

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
)

// loopKind names the construct that a break statement or a continue statement
// leaves.
type loopKind int

const (
	// loopPlain is a for loop or a range loop. It catches a break and a
	// continue.
	loopPlain loopKind = iota
	// loopSwitch is a switch statement or a select statement. It catches a
	// break only.
	loopSwitch
	// loopYield is the body of a range loop over a function. The body becomes a
	// Dart closure, thus a break gives `return false` and a continue gives
	// `return true`.
	loopYield
)

// loop holds 1 entry of the stack of the constructs that hold the current
// statement.
type loop struct {
	kind  loopKind
	label string
	yield *ast.RangeStmt

	// flag names the variable that marks a branch that leaves the body of a
	// range loop over a function and reaches this construct. It is empty when
	// no such branch exists.
	flag string
}

func (b *builder) pushLoop(kind loopKind, label string) {
	b.loops = append(b.loops, loop{kind: kind, label: label})
}

func (b *builder) popLoop() { b.loops = b.loops[:len(b.loops)-1] }

// branchTarget finds the construct that a break statement or a continue
// statement leaves. It gives the index in the stack, and -1 when the stack holds
// no such construct.
//
// A continue statement leaves a loop, thus a switch statement does not catch it.
func (b *builder) branchTarget(label string, isBreak bool) int {
	for i := len(b.loops) - 1; i >= 0; i-- {
		l := b.loops[i]
		if label != "" {
			if l.label == label {
				return i
			}
			continue
		}
		if isBreak || l.kind != loopSwitch {
			return i
		}
	}
	return -1
}

// crossesYield reports whether a branch to the construct at index k leaves the
// body of a range loop over a function.
//
// Such a body is a Dart closure. A Dart closure leaves no loop outside itself,
// thus the branch writes a flag and gives false to the sequence.
func (b *builder) crossesYield(k int) bool {
	for i := len(b.loops) - 1; i > k; i-- {
		if b.loops[i].kind == loopYield {
			return true
		}
	}
	return false
}

// breakOrContinue writes a break statement or a continue statement.
//
// The body of a range loop over a function is a Dart closure. A break of such a
// loop gives `return false`, and a continue gives `return true`.
func (b *builder) breakOrContinue(s *ast.BranchStmt, label string) []ir.Stmt {
	isBreak := s.Tok == token.BREAK
	k := b.branchTarget(label, isBreak)
	if k < 0 {
		b.fail(s.Pos(), "the %s statement leaves no construct of the subset", s.Tok)
		return nil
	}
	if b.crossesYield(k) {
		return b.yieldBranch(k, isBreak)
	}
	if b.loops[k].kind == loopYield {
		return []ir.Stmt{&ir.Return{Values: []ir.Expr{boolConst(isBreak == false)}}}
	}
	if isBreak {
		return []ir.Stmt{&ir.Break{Label: label}}
	}
	return []ir.Stmt{&ir.Continue{Label: label}}
}

// insideYield reports whether the current statement is in the body of a range
// loop over a function.
func (b *builder) insideYield() bool {
	for _, l := range b.loops {
		if l.kind == loopYield {
			return true
		}
	}
	return false
}

// rangeFunc rewrites a range loop over a function.
//
//	for k, v := range seq { BODY }
//
// becomes
//
//	seq(func(k K, v V) bool { BODY; return true })
//
// A break gives `return false`, and a continue gives `return true`. A return
// sets a flag, and the code after the call reads it.
func (b *builder) rangeFunc(s *ast.RangeStmt, label string, sig *types.Signature) []ir.Stmt {
	if sig.Params().Len() != 1 {
		b.fail(s.Pos(), "a range loop needs a function with 1 parameter")
		return nil
	}
	yield, _ := typeset.Core(sig.Params().At(0).Type()).(*types.Signature)
	if yield == nil || yield.Results().Len() != 1 {
		b.fail(s.Pos(), "a range loop needs a yield function that gives 1 result")
		return nil
	}
	params := make([]*ir.Var, yield.Params().Len())
	base := b.temp(s.Pos(), "y")
	for i := range params {
		params[i] = &ir.Var{
			Name: base + strconv.Itoa(i),
			Type: yield.Params().At(i).Type(),
		}
	}
	body := &ir.Block{}
	if len(params) > 0 {
		body.List = append(body.List, b.bind(s, s.Key,
			&ir.Ident{Name: params[0].Name, Type: params[0].Type})...)
	}
	if len(params) > 1 {
		body.List = append(body.List, b.bind(s, s.Value,
			&ir.Ident{Name: params[1].Name, Type: params[1].Type})...)
	}
	b.pushLoop(loopYield, label)
	b.loops[len(b.loops)-1].yield = s
	previousYield := b.currentYield
	b.currentYield = s
	body.List = append(body.List, b.block(s.Body).List...)
	b.currentYield = previousYield
	b.popLoop()
	body.List = append(body.List, &ir.Return{Values: []ir.Expr{boolConst(true)}})

	fn := &ir.FuncLit{
		Pos:     s.Pos(),
		Params:  params,
		Results: []*ir.Var{{Name: "$ok", Type: yield.Results().At(0).Type()}},
		Body:    body,
		Type:    yield,
		Async:   b.blocks.Blocks(s),
	}
	// A body that can block becomes a local state machine, exactly as a
	// function literal does. Before this rule, the body stayed a plain
	// closure, and a park of a channel inside it was lost.
	var arg ir.Expr = fn
	if fn.Async || len(gotoTargets(s.Body)) > 0 {
		fn.Machine = b.flatten(fn.Body, fn.Params, nil, fn.Async, false, false,
			fn.Results, b.frameClass(s.Pos()), s.Pos())
		addResults(fn.Machine, fn.Results)
	}
	if fn.Async {
		fn.Captured = b.rangeCaptures(s)
		local := b.litName(s.Pos())
		b.pending = append(b.pending, &ir.LocalFunc{Name: local, Lit: fn})
		arg = &ir.Ident{Name: local, Type: yield}
	}
	call := &ir.Call{
		Fun:   b.expr(s.X),
		Args:  []ir.Expr{arg},
		Type:  sig.Results(),
		Async: b.blocks.RangeAsync(s),
	}
	out := []ir.Stmt{&ir.ExprStmt{X: call}}
	if check := b.returnCheck(); check != nil {
		out = append(out, check)
	}
	out = append(out, b.branchChecks()...)
	return append(out, b.gotoChecks()...)
}

type yieldGoto struct {
	label  string
	target *ast.RangeStmt
	flag   *ir.Var
}

func (b *builder) rangeLabels(body *ast.BlockStmt) map[string]*ast.RangeStmt {
	labels := map[string]*ast.RangeStmt{}
	var visit func(ast.Node, *ast.RangeStmt)
	visit = func(node ast.Node, owner *ast.RangeStmt) {
		ast.Inspect(node, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncLit:
				return false
			case *ast.LabeledStmt:
				labels[n.Label.Name] = owner
			case *ast.RangeStmt:
				if _, ok := typeset.Core(b.typeOf(n.X)).(*types.Signature); ok {
					visit(n.Body, n)
					return false
				}
			}
			return true
		})
	}
	visit(body, nil)
	return labels
}

func (b *builder) gotoBranch(s *ast.BranchStmt, label string) []ir.Stmt {
	target := b.labelYields[s.Label.Name]
	if target == b.currentYield {
		return []ir.Stmt{&ir.Goto{Label: label}}
	}
	var flag *ir.Var
	for _, jump := range b.yieldGotos {
		if jump.label == label {
			flag = jump.flag
			break
		}
	}
	if flag == nil {
		flag = b.tempVar(s.Pos(), "ygt", types.Typ[types.Int])
		flag.Boxed = true
		b.yieldGotos = append(b.yieldGotos, yieldGoto{label: label, target: target, flag: flag})
		b.branchFlags = append(b.branchFlags, flag)
	}
	return []ir.Stmt{
		&ir.Assign{Lhs: identOf(flag), Rhs: intConst(1)},
		&ir.Return{Values: []ir.Expr{boolConst(false)}},
	}
}

func (b *builder) gotoChecks() []ir.Stmt {
	var out []ir.Stmt
	for _, jump := range b.yieldGotos {
		active := jump.target == nil
		for _, loop := range b.loops {
			if loop.yield == jump.target {
				active = true
			}
		}
		if !active {
			continue
		}
		body := []ir.Stmt{&ir.Return{Values: []ir.Expr{boolConst(false)}}}
		if jump.target == b.currentYield {
			body = []ir.Stmt{&ir.Assign{Lhs: identOf(jump.flag), Rhs: intConst(0)}, &ir.Goto{Label: jump.label}}
		}
		out = append(out, &ir.If{
			Cond: &ir.Binary{Op: token.NEQ, X: identOf(jump.flag), Y: intConst(0), Type: types.Typ[types.Bool]},
			Then: &ir.Block{List: body},
		})
	}
	return out
}

// yieldBranch writes a branch that leaves the body of a range loop over a
// function.
//
// Such a body is a Dart closure. The branch writes the flag of its target and
// gives false to the sequence. The code after the call of the sequence reads the
// flag: it gives false again while another such body holds it, and it runs the
// real branch at the level of the target.
func (b *builder) yieldBranch(k int, isBreak bool) []ir.Stmt {
	target := &b.loops[k]
	if target.label == "" {
		// An unlabelled branch reaches the innermost construct, and that
		// construct stands inside the closure.
		b.fail(b.fn.Pos, "a branch with no label leaves no closure")
		return nil
	}
	if target.flag == "" {
		target.flag = "$ybr" + target.label
		b.branchFlags = append(b.branchFlags,
			&ir.Var{Name: target.flag, Type: types.Typ[types.Int], Boxed: true})
	}
	mark := branchBreak
	if !isBreak {
		mark = branchContinue
	}
	return []ir.Stmt{
		&ir.Assign{
			Lhs: &ir.Ident{Name: target.flag, Type: types.Typ[types.Int], Boxed: true},
			Rhs: intConst(int64(mark)),
		},
		&ir.Return{Values: []ir.Expr{boolConst(false)}},
	}
}

// The value of a branch flag. 0 marks a body that no branch left.
const (
	branchBreak    = 1
	branchContinue = 2
)

// branchChecks writes the test of each branch flag after the call of a
// sequence.
func (b *builder) branchChecks() []ir.Stmt {
	inner := -1
	for i, l := range b.loops {
		if l.kind == loopYield {
			inner = i
		}
	}
	var out []ir.Stmt
	for i := len(b.loops) - 1; i >= 0; i-- {
		l := b.loops[i]
		if l.flag == "" {
			continue
		}
		read := &ir.Ident{Name: l.flag, Type: types.Typ[types.Int], Boxed: true}
		if inner > i {
			// Another closure stands between this point and the target, thus
			// the flag travels out 1 more level.
			out = append(out, &ir.If{
				Cond: &ir.Binary{Op: token.NEQ, X: read, Y: intConst(0),
					Type: types.Typ[types.Bool]},
				Then: &ir.Block{List: []ir.Stmt{
					&ir.Return{Values: []ir.Expr{boolConst(false)}},
				}},
			})
			continue
		}
		out = append(out, b.branchRun(l, read, branchBreak, true))
		if l.kind != loopSwitch {
			out = append(out, b.branchRun(l, read, branchContinue, false))
		}
	}
	return out
}

// branchRun writes the real branch of 1 value of a flag. The flag takes 0
// again, because the target construct runs its next pass.
func (b *builder) branchRun(l loop, read ir.Expr, mark int, isBreak bool) ir.Stmt {
	var jump ir.Stmt = &ir.Continue{Label: l.label}
	if isBreak {
		jump = &ir.Break{Label: l.label}
	}
	if l.kind == loopYield {
		jump = &ir.Return{Values: []ir.Expr{boolConst(!isBreak)}}
	}
	return &ir.If{
		Cond: &ir.Binary{Op: token.EQL, X: read, Y: intConst(int64(mark)),
			Type: types.Typ[types.Bool]},
		Then: &ir.Block{List: []ir.Stmt{
			&ir.Assign{
				Lhs: &ir.Ident{Name: l.flag, Type: types.Typ[types.Int], Boxed: true},
				Rhs: intConst(0),
			},
			jump,
		}},
	}
}

func boolConst(v bool) ir.Expr {
	return &ir.Const{Value: constant.MakeBool(v), Type: types.Typ[types.Bool]}
}

// yieldReturn rewrites a return statement inside the body of a range loop over a
// function.
//
// Such a body is a Dart closure, thus a return of it leaves the closure and not
// the function. The statement writes each value into a temporary variable, it
// sets a flag, and it gives false to the sequence. The code after the call of
// the sequence reads the flag.
func (b *builder) yieldReturn(s *ast.ReturnStmt) []ir.Stmt {
	f := b.fn
	if f == nil {
		return nil
	}
	b.needReturnState(f)
	var out []ir.Stmt
	switch {
	case len(s.Results) == 0 && hasNamedResult(f):
		for i, r := range f.Results {
			out = append(out, &ir.Assign{
				Lhs: identOf(b.retVars[i]),
				Rhs: &ir.Ident{Name: r.Name, Type: r.Type, Boxed: r.Boxed},
			})
		}
	case len(s.Results) == 1 && len(f.Results) > 1:
		lhs := make([]ir.Expr, len(b.retVars))
		for i, v := range b.retVars {
			lhs[i] = identOf(v)
		}
		out = append(out, &ir.AssignMulti{Lhs: lhs, Rhs: b.multiValue(s.Results[0])})
	default:
		for i, r := range s.Results {
			if i >= len(b.retVars) {
				break
			}
			out = append(out, &ir.Assign{
				Lhs: identOf(b.retVars[i]),
				Rhs: b.copyPoint(b.exprAs(r, f.Results[i].Type)),
			})
		}
	}
	out = append(out, &ir.Assign{
		Lhs: &ir.Ident{Name: b.retFlag, Type: types.Typ[types.Bool], Boxed: true},
		Rhs: boolConst(true),
	})
	return append(out, &ir.Return{Values: []ir.Expr{boolConst(false)}})
}

// needReturnState allocates the flag and the result variables of the function.
// The call is idempotent, thus each return of the function shares 1 set.
func (b *builder) needReturnState(f *ir.Func) {
	if b.retFlag != "" {
		return
	}
	b.retFlag = b.temp(f.Pos, "yret")
	b.retVars = make([]*ir.Var, len(f.Results))
	for i, r := range f.Results {
		b.retVars[i] = &ir.Var{Name: b.temp(f.Pos, "yr"+strconv.Itoa(i)), Type: r.Type, Boxed: true}
	}
}

// returnCheck writes the statement that reads the flag after the call of a
// sequence. The result is nil when no return statement of the function reaches
// the body of a range loop over a function.
//
// A call inside another such body gives false to its own sequence, thus the flag
// travels out 1 level for each call.
func (b *builder) returnCheck() ir.Stmt {
	if b.retFlag == "" {
		return nil
	}
	read := &ir.Ident{Name: b.retFlag, Type: types.Typ[types.Bool], Boxed: true}
	if b.insideYield() {
		return &ir.If{Cond: read, Then: &ir.Block{List: []ir.Stmt{
			&ir.Return{Values: []ir.Expr{boolConst(false)}},
		}}}
	}
	values := make([]ir.Expr, len(b.retVars))
	for i, v := range b.retVars {
		values[i] = identOf(v)
	}
	return &ir.If{Cond: read, Then: &ir.Block{List: []ir.Stmt{
		&ir.Return{Values: values},
	}}}
}

// returnState writes the declaration of the flag and of each result variable.
// The declarations go at the start of the function body.
func (b *builder) returnState() []ir.Stmt {
	var out []ir.Stmt
	if b.retFlag != "" {
		out = append(out, &ir.Declare{
			Vars: []*ir.Var{{Name: b.retFlag, Type: types.Typ[types.Bool], Boxed: true}},
			Init: boolConst(false),
		})
		for _, v := range b.retVars {
			out = append(out, &ir.Declare{Vars: []*ir.Var{v}})
		}
	}
	for _, v := range b.branchFlags {
		out = append(out, &ir.Declare{Vars: []*ir.Var{v}, Init: intConst(0)})
	}
	return out
}

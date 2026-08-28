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
// Such a body is a Dart closure. A Dart closure cannot leave a loop outside
// itself, thus the compiler rejects the branch.
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
		b.fail(s.Pos(), "the %s statement leaves the body of a range loop over a "+
			"function, and such a body is a Dart closure", s.Tok)
		return nil
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
	body.List = append(body.List, b.block(s.Body).List...)
	b.popLoop()
	body.List = append(body.List, &ir.Return{Values: []ir.Expr{boolConst(true)}})

	fn := &ir.FuncLit{
		Params:  params,
		Results: []*ir.Var{{Name: "$ok", Type: yield.Results().At(0).Type()}},
		Body:    body,
		Type:    yield,
		Async:   b.blocks.SignatureAsync(yield),
	}
	call := &ir.Call{
		Fun:   b.expr(s.X),
		Args:  []ir.Expr{fn},
		Type:  sig.Results(),
		Async: b.blocks.SignatureAsync(sig),
	}
	out := []ir.Stmt{&ir.ExprStmt{X: call}}
	if check := b.returnCheck(); check != nil {
		out = append(out, check)
	}
	return out
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
				Lhs: &ir.Ident{Name: b.retVars[i].Name, Type: b.retVars[i].Type},
				Rhs: &ir.Ident{Name: r.Name, Type: r.Type, Boxed: r.Boxed},
			})
		}
	case len(s.Results) == 1 && len(f.Results) > 1:
		lhs := make([]ir.Expr, len(b.retVars))
		for i, v := range b.retVars {
			lhs[i] = &ir.Ident{Name: v.Name, Type: v.Type}
		}
		out = append(out, &ir.AssignMulti{Lhs: lhs, Rhs: b.multiValue(s.Results[0])})
	default:
		for i, r := range s.Results {
			if i >= len(b.retVars) {
				break
			}
			out = append(out, &ir.Assign{
				Lhs: &ir.Ident{Name: b.retVars[i].Name, Type: b.retVars[i].Type},
				Rhs: b.copyPoint(b.exprAs(r, f.Results[i].Type)),
			})
		}
	}
	out = append(out, &ir.Assign{
		Lhs: &ir.Ident{Name: b.retFlag, Type: types.Typ[types.Bool]},
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
		b.retVars[i] = &ir.Var{Name: b.temp(f.Pos, "yr"+strconv.Itoa(i)), Type: r.Type}
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
	read := &ir.Ident{Name: b.retFlag, Type: types.Typ[types.Bool]}
	if b.insideYield() {
		return &ir.If{Cond: read, Then: &ir.Block{List: []ir.Stmt{
			&ir.Return{Values: []ir.Expr{boolConst(false)}},
		}}}
	}
	values := make([]ir.Expr, len(b.retVars))
	for i, v := range b.retVars {
		values[i] = &ir.Ident{Name: v.Name, Type: v.Type}
	}
	return &ir.If{Cond: read, Then: &ir.Block{List: []ir.Stmt{
		&ir.Return{Values: values},
	}}}
}

// returnState writes the declaration of the flag and of each result variable.
// The declarations go at the start of the function body.
func (b *builder) returnState() []ir.Stmt {
	if b.retFlag == "" {
		return nil
	}
	out := []ir.Stmt{&ir.Declare{
		Vars: []*ir.Var{{Name: b.retFlag, Type: types.Typ[types.Bool]}},
		Init: boolConst(false),
	}}
	for _, v := range b.retVars {
		out = append(out, &ir.Declare{Vars: []*ir.Var{v}})
	}
	return out
}

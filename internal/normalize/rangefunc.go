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
	return []ir.Stmt{&ir.ExprStmt{X: call}}
}

func boolConst(v bool) ir.Expr {
	return &ir.Const{Value: constant.MakeBool(v), Type: types.Typ[types.Bool]}
}

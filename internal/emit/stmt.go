package emit

import (
	"fmt"
	"strings"

	"github.com/eddyw/godart/internal/ir"
)

func (e *emitter) stmt(s ir.Stmt) {
	switch s := s.(type) {
	case *ir.Block:
		e.line("{")
		for _, inner := range s.List {
			e.stmt(inner)
		}
		e.line("}")
	case *ir.ExprStmt:
		e.line("%s;", e.statementExpr(s.X))
	case *ir.Declare:
		e.declare(s)
	case *ir.Assign:
		e.assign(s.Lhs, s.Rhs)
	case *ir.AssignMulti:
		parts := make([]string, len(s.Lhs))
		for i, l := range s.Lhs {
			parts[i] = e.expr(l)
		}
		e.line("(%s) = %s;", strings.Join(parts, ", "), e.expr(s.Rhs))
	case *ir.If:
		e.ifStmt(s)
	case *ir.For:
		e.forStmt(s)
	case *ir.Switch:
		e.switchStmt(s)
	case *ir.Return:
		e.returnStmt(s)
	case *ir.Break:
		e.line("break %s;", e.breakTarget(s.Label))
	case *ir.Continue:
		if s.Label == "" {
			e.line("continue;")
			return
		}
		e.line("continue %s;", s.Label)
	case *ir.Labeled:
		e.line("%s:", s.Label)
		e.stmt(s.Stmt)
	case *ir.Defer:
		e.line("$defer.push(() => %s);", e.expr(s.Call))
	case *ir.Send:
		e.line("await %s.send(%s);", e.expr(s.Chan), e.expr(s.Value))
	case *ir.Go:
		e.goStmt(s)
	case *ir.Select:
		e.selectStmt(s)
	case *ir.Goto:
		e.fail("the label %s of a goto statement is outside the function body", s.Label)
	default:
		e.fail("the statement %T has no Dart form", s)
	}
}

// statementExpr gives an expression that stands alone as a Dart statement. A
// read that Dart drops still runs its bounds check, thus the value goes to a
// function that ignores it.
func (e *emitter) statementExpr(x ir.Expr) string {
	switch x.(type) {
	case *ir.Call, *ir.Builtin:
		return e.expr(x)
	}
	return "$drop(" + e.expr(x) + ")"
}

func (e *emitter) declare(s *ir.Declare) {
	if len(s.Vars) == 1 {
		v := s.Vars[0]
		if s.Init == nil {
			e.line("%s %s = %s;", e.dartType(v.Type), v.Name, e.zero(v.Type))
			return
		}
		e.line("%s %s = %s;", e.dartType(v.Type), v.Name, e.expr(s.Init))
		return
	}
	if s.Init == nil {
		for _, v := range s.Vars {
			e.line("%s %s = %s;", e.dartType(v.Type), v.Name, e.zero(v.Type))
		}
		return
	}
	names := make([]string, len(s.Vars))
	for i, v := range s.Vars {
		names[i] = v.Name
	}
	e.line("var (%s) = %s;", strings.Join(names, ", "), e.expr(s.Init))
}

// assign writes 1 value into 1 location. A struct and an array keep their
// identity, thus the write copies the fields and does not replace the object.
func (e *emitter) assign(lhs, rhs ir.Expr) {
	t := lhs.GoType()
	value := e.expr(rhs)
	if idx, ok := lhs.(*ir.Index); ok && isMapType(idx.X.GoType()) {
		e.line("%s.set(%s, %s);", e.expr(idx.X), e.expr(idx.Index), value)
		return
	}
	if isStructType(t) {
		e.line("%s.$assign(%s);", e.expr(lhs), value)
		return
	}
	if isArrayType(t) {
		e.line("%s;", e.assignArray(e.expr(lhs), t, value))
		return
	}
	if idx, ok := lhs.(*ir.Index); ok {
		if target := e.indexTarget(idx); target != "" {
			e.line("%s, %s);", target, value)
			return
		}
	}
	e.line("%s = %s;", e.expr(lhs), value)
}

// goStmt starts a goroutine. Go waits for the scheduler, thus the body does not
// run at the point of the statement.
func (e *emitter) goStmt(s *ir.Go) {
	text := e.expr(s.Call)
	if call, ok := s.Call.(*ir.Call); ok && call.Async {
		e.line("$go(() async { %s; });", text)
		return
	}
	e.line("$go(() { %s; });", text)
}

// selectStmt writes a select statement. The runtime gives the number of the
// case that it takes, and a labeled block holds the bodies.
func (e *emitter) selectStmt(s *ir.Select) {
	label := s.Label
	if label == "" {
		e.labelSeq++
		label = fmt.Sprintf("$sel%d", e.labelSeq)
	}
	list := label + "$c"
	choice := label + "$i"
	hasDefault := false
	parts := make([]string, 0, len(s.Cases))
	for _, c := range s.Cases {
		switch c.Kind {
		case ir.SelectRecv:
			parts = append(parts, fmt.Sprintf("RecvCase<%s>(%s)", e.dartType(c.Elem), e.expr(c.Chan)))
		case ir.SelectSend:
			parts = append(parts, fmt.Sprintf("SendCase<%s>(%s, %s)",
				e.dartType(c.Elem), e.expr(c.Chan), e.expr(c.Value)))
		default:
			hasDefault = true
		}
	}
	e.line("%s:", label)
	e.line("{")
	e.line("final List<SelectCase> %s = <SelectCase>[%s];", list, strings.Join(parts, ", "))
	e.line("final int %s = await $select(%s, %t);", choice, list, hasDefault)
	e.breakables = append(e.breakables, label)
	index := 0
	first := true
	for _, c := range s.Cases {
		if c.Kind == ir.SelectDefault {
			continue
		}
		keyword := "else if"
		if first {
			keyword = "if"
			first = false
		}
		e.line("%s (%s == %d) {", keyword, choice, index)
		if c.Kind == ir.SelectRecv {
			e.selectRecvBind(c, list, index, label)
		}
		for _, inner := range c.Body.List {
			e.stmt(inner)
		}
		e.line("}")
		index++
	}
	for _, c := range s.Cases {
		if c.Kind != ir.SelectDefault {
			continue
		}
		if first {
			e.line("{")
		} else {
			e.line("else {")
		}
		for _, inner := range c.Body.List {
			e.stmt(inner)
		}
		e.line("}")
	}
	e.breakables = e.breakables[:len(e.breakables)-1]
	e.line("}")
}

// selectRecvBind writes the value and the ok flag of a receive case.
func (e *emitter) selectRecvBind(c *ir.SelectClause, list string, index int, label string) {
	read := fmt.Sprintf("%s$r%d", label, index)
	e.line("final %s = (%s[%d] as RecvCase<%s>).result!;", read, list, index, e.dartType(c.Elem))
	fields := []string{read + ".$1", read + ".$2"}
	for i, v := range c.Vars {
		if i >= len(fields) || v.Name == "_" {
			continue
		}
		e.line("%s %s = %s;", e.dartType(v.Type), v.Name, fields[i])
	}
	for i, lhs := range c.Lhs {
		if i >= len(fields) {
			continue
		}
		e.line("%s = %s;", e.expr(lhs), fields[i])
	}
}

func (e *emitter) ifStmt(s *ir.If) {
	e.line("if (%s) {", e.expr(s.Cond))
	for _, inner := range s.Then.List {
		e.stmt(inner)
	}
	if s.Else == nil {
		e.line("}")
		return
	}
	e.line("} else {")
	e.stmt(s.Else)
	e.line("}")
}

func (e *emitter) forStmt(s *ir.For) {
	init := e.forClause(s.Init)
	cond := ""
	if s.Cond != nil {
		cond = e.expr(s.Cond)
	}
	post := e.forClause(s.Post)
	if s.Label != "" {
		e.line("%s:", s.Label)
	}
	e.line("for (%s; %s; %s) {", init, cond, post)
	e.breakables = append(e.breakables, "")
	for _, inner := range s.Body.List {
		e.stmt(inner)
	}
	e.breakables = e.breakables[:len(e.breakables)-1]
	e.line("}")
}

// forClause writes the first clause or the third clause of a Dart for loop.
func (e *emitter) forClause(s ir.Stmt) string {
	switch s := s.(type) {
	case nil:
		return ""
	case *ir.Declare:
		if len(s.Vars) != 1 {
			e.fail("a for clause holds 1 variable")
			return ""
		}
		v := s.Vars[0]
		if s.Init == nil {
			return fmt.Sprintf("%s %s = %s", e.dartType(v.Type), v.Name, e.zero(v.Type))
		}
		return fmt.Sprintf("%s %s = %s", e.dartType(v.Type), v.Name, e.expr(s.Init))
	case *ir.Assign:
		if isStructType(s.Lhs.GoType()) {
			return fmt.Sprintf("%s.$assign(%s)", e.expr(s.Lhs), e.expr(s.Rhs))
		}
		return fmt.Sprintf("%s = %s", e.expr(s.Lhs), e.expr(s.Rhs))
	case *ir.ExprStmt:
		return e.statementExpr(s.X)
	}
	e.fail("the statement %T does not fit a for clause", s)
	return ""
}

// switchStmt writes a Go switch. Each case takes an index, and the bodies run
// from the index of the match. Thus fallthrough needs no second form.
func (e *emitter) switchStmt(s *ir.Switch) {
	label := s.Label
	if label == "" {
		e.labelSeq++
		label = fmt.Sprintf("$sw%d", e.labelSeq)
	}
	index := label + "$i"
	e.line("%s:", label)
	e.line("{")
	e.line("int %s = %d;", index, len(s.Cases))
	defaultCase := -1
	first := true
	for i, c := range s.Cases {
		if len(c.Values) == 0 {
			defaultCase = i
			continue
		}
		parts := make([]string, len(c.Values))
		for k, v := range c.Values {
			if s.Tag == nil {
				parts[k] = e.expr(v)
				continue
			}
			parts[k] = e.equal(s.Tag, v)
		}
		keyword := "else if"
		if first {
			keyword = "if"
			first = false
		}
		e.line("%s (%s) { %s = %d; }", keyword, strings.Join(parts, " || "), index, i)
	}
	if defaultCase >= 0 {
		if first {
			e.line("%s = %d;", index, defaultCase)
		} else {
			e.line("else { %s = %d; }", index, defaultCase)
		}
	}
	e.breakables = append(e.breakables, label)
	for i, c := range s.Cases {
		e.line("if (%s <= %d) {", index, i)
		for _, inner := range c.Body.List {
			e.stmt(inner)
		}
		if !c.Fallthrough {
			e.line("break %s;", label)
		}
		e.line("}")
	}
	e.breakables = e.breakables[:len(e.breakables)-1]
	e.line("}")
}

// breakTarget gives the label of a break statement. A Go switch becomes a
// labeled block, and a Dart block needs the label on each break.
func (e *emitter) breakTarget(label string) string {
	if label != "" {
		return label
	}
	if n := len(e.breakables); n > 0 {
		return e.breakables[n-1]
	}
	return ""
}

func (e *emitter) returnStmt(s *ir.Return) {
	if e.returnLabel != "" {
		e.namedReturn(s)
		return
	}
	switch len(s.Values) {
	case 0:
		e.line("return;")
	case 1:
		e.line("return %s;", e.expr(s.Values[0]))
	default:
		parts := make([]string, len(s.Values))
		for i, v := range s.Values {
			parts[i] = e.expr(v)
		}
		e.line("return (%s);", strings.Join(parts, ", "))
	}
}

// namedReturn writes the results into their variables and leaves the block. A
// deferred call runs after this point, thus it can still change a result.
func (e *emitter) namedReturn(s *ir.Return) {
	switch {
	case len(s.Values) == 0:
	case len(s.Values) == 1 && len(e.returnVars) > 1:
		parts := make([]string, len(e.returnVars))
		for i, r := range e.returnVars {
			parts[i] = r.Name
		}
		e.line("(%s) = %s;", strings.Join(parts, ", "), e.expr(s.Values[0]))
	default:
		for i, v := range s.Values {
			if i >= len(e.returnVars) {
				break
			}
			e.assign(&ir.Ident{Name: e.returnVars[i].Name, Type: e.returnVars[i].Type}, v)
		}
	}
	e.line("break %s;", e.returnLabel)
}

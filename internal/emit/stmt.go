package emit

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/typeset"
)

func (e *emitter) stmt(s ir.Stmt) {
	e.list([]ir.Stmt{s})
}

// list writes each statement of a list.
//
// A machine keeps each local in 1 variable and writes a new box at each pass of
// a loop, thus a literal that a statement builds takes a copy of the box. The
// copy stands in a block in front of the statement. A local function keeps its
// name for the statements that follow it, thus the block of such a function
// ends with the list.
func (e *emitter) list(stmts []ir.Stmt) {
	prev := e.captures
	open := 0
	for _, s := range stmts {
		e.captures = nil
		mark := e.buf.Len()
		e.stmtOne(s)
		if len(e.captures) == 0 {
			continue
		}
		e.openCaptures(mark)
		if _, ok := s.(*ir.LocalFunc); ok {
			open++
			continue
		}
		e.line("}")
	}
	for ; open > 0; open-- {
		e.line("}")
	}
	e.captures = prev
}

// openCaptures puts the copy of each box in front of the text that starts at
// the mark, and it leaves the block open.
func (e *emitter) openCaptures(mark int) {
	text := e.buf.String()
	head, body := text[:mark], text[mark:]
	e.buf.Reset()
	e.buf.WriteString(head)
	e.line("{")
	for _, c := range e.captures {
		e.line("final %s %s = %s;", c.typ, c.name, c.from)
	}
	e.buf.WriteString(body)
}

func (e *emitter) stmtOne(s ir.Stmt) {
	e.mark(ir.PosOf(s))
	switch s := s.(type) {
	case *ir.Block:
		e.line("{")
		e.list(s.List)
		e.line("}")
	case *ir.ExprStmt:
		e.line("%s;", e.statementExpr(s.X))
	case *ir.Declare:
		e.declare(s)
	case *ir.Assign:
		if s.Define {
			e.defineAssign(s)
			return
		}
		e.assign(s.Lhs, s.Rhs)
	case *ir.AssignMulti:
		e.assignMulti(s)
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
		// Go resolves the method of an interface receiver at the statement,
		// and a nil receiver panics there. A function value keeps its check
		// for the run of the deferred call.
		if m, ok := s.Call.(*ir.Call); ok {
			if fun, ok := m.Fun.(*ir.Method); ok && isInterfaceType(fun.Recv.GoType()) {
				recv := e.expr(fun.Recv)
				e.line("%s == null ? $panicNil() : null;", recv)
			}
			// The chain runs the body of the deferred function under the
			// token of the function, and recover reads the token. Go gives a
			// recover the panic value only in the function that the defer
			// pushed, and each function that it calls reads nil.
			if id := e.deferredId(m.Fun); id != 0 {
				e.markDrop()
				e.line("$defer.push(() => $dcall(%d, () => %s));", id, e.expr(s.Call))
				return
			}
		}
		e.markDrop()
		e.line("$defer.push(() => %s);", e.expr(s.Call))
	case *ir.Send:
		e.line("%s.send(%s);", e.chanExpr(s.Chan), e.expr(s.Value))
	case *ir.Go:
		e.goStmt(s)
	case *ir.Jump:
		e.line("$s = %d;", s.State)
		e.line("continue $m;")
	case *ir.Resume:
		e.resume(s)
	case *ir.DeferNext:
		e.deferNext(s)
	case *ir.DeferEnd:
		e.deferEnd()
	case *ir.LocalFunc:
		e.localFunc(s)
	case *ir.SelectBind:
		e.selectBind(s)
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

// declareOne writes 1 variable with its value. A boxed variable holds a box, and
// the value goes inside the box.
func (e *emitter) declareOne(v *ir.Var, value string) {
	if v.Name == "_" {
		// Dart holds no setter of the wildcard, and the value read stands.
		e.line("final $blank%d = %s;", e.labelSeq, value)
		e.labelSeq++
		return
	}
	e.line("%s;", e.declareText(v, value))
}

// declareText writes the declaration of 1 variable with no terminator.
func (e *emitter) declareText(v *ir.Var, value string) string {
	if v.Boxed {
		return fmt.Sprintf("Box<%s> %s = Box(%s)", e.dartType(v.Type), v.Name, value)
	}
	return fmt.Sprintf("%s %s = %s", e.dartType(v.Type), v.Name, value)
}

func (e *emitter) declare(s *ir.Declare) {
	if len(s.Vars) == 1 {
		v := s.Vars[0]
		if s.Init == nil {
			e.declareOne(v, e.zero(v.Type))
			return
		}
		if lit, ok := s.Init.(*ir.FuncLit); ok {
			// The machine hoists the literal of a deferred call into a
			// variable, and the defer statement reads the token of the
			// literal through this name.
			e.varLitId[v.Name] = e.funcId(lit)
		}
		e.declareOne(v, e.expr(s.Init))
		return
	}
	if s.Init == nil {
		for _, v := range s.Vars {
			e.declareOne(v, e.zero(v.Type))
		}
		return
	}
	boxed := false
	names := make([]string, len(s.Vars))
	for i, v := range s.Vars {
		names[i] = v.Name
		boxed = boxed || v.Boxed
	}
	if boxed {
		e.labelSeq++
		tmp := fmt.Sprintf("$rec%d", e.labelSeq)
		e.line("final %s = %s;", tmp, e.expr(s.Init))
		for i, v := range s.Vars {
			e.declareOne(v, fmt.Sprintf("%s.$%d", tmp, i+1))
		}
		return
	}
	e.line("var (%s) = %s;", strings.Join(names, ", "), e.expr(s.Init))
}

// assignMulti writes each value of a record. A record pattern of Dart takes no
// boxed variable, thus such a write reads the record from a temporary variable.
func (e *emitter) assignMulti(s *ir.AssignMulti) {
	if !anyBoxed(s.Lhs) {
		parts := make([]string, len(s.Lhs))
		for i, l := range s.Lhs {
			parts[i] = e.expr(l)
		}
		e.line("(%s) = %s;", strings.Join(parts, ", "), e.expr(s.Rhs))
		return
	}
	e.labelSeq++
	tmp := fmt.Sprintf("$rec%d", e.labelSeq)
	e.line("final %s = %s;", tmp, e.expr(s.Rhs))
	for i, l := range s.Lhs {
		id, ok := l.(*ir.Ident)
		if ok && id.Name == "_" {
			continue
		}
		if ok && id.Boxed && s.Define {
			e.line("%s = Box(%s.$%d);", id.Name, tmp, i+1)
			continue
		}
		e.line("%s = %s.$%d;", e.expr(l), tmp, i+1)
	}
}

// anyBoxed reports whether 1 target of a multiple assignment holds a box.
func anyBoxed(list []ir.Expr) bool {
	for _, l := range list {
		if id, ok := l.(*ir.Ident); ok && id.Boxed {
			return true
		}
	}
	return false
}

// defineAssign writes the value of a declaration that a machine lifted. Go
// binds a new variable at such a point, thus the write replaces the value and
// does not write through the old one.
func (e *emitter) defineAssign(s *ir.Assign) {
	value := e.expr(s.Rhs)
	if id, ok := s.Lhs.(*ir.Ident); ok && id.Boxed {
		e.line("%s = Box(%s);", id.Name, value)
		return
	}
	if idn, ok := s.Rhs.(*ir.Ident); ok {
		if tok, ok := e.varLitId[idn.Name]; ok {
			if li, ok := s.Lhs.(*ir.Ident); ok {
				e.varLitId[li.Name] = tok
			}
		}
	}
	e.line("%s = %s;", e.expr(s.Lhs), value)
}

// assign writes 1 value into 1 location. A struct and an array keep their
// identity, thus the write copies the fields and does not replace the object.
func (e *emitter) assign(lhs, rhs ir.Expr) {
	// A write to the blank name keeps the evaluation only. Dart holds no
	// setter of the wildcard.
	if id, ok := lhs.(*ir.Ident); ok && id.Name == "_" {
		e.line("final $blank%d = %s;", e.labelSeq, e.expr(rhs))
		e.labelSeq++
		return
	}
	t := lhs.GoType()
	value := e.expr(rhs)
	// A copy of a variable that holds the literal of a deferred call keeps
	// the token of the literal, because the defer statement reads it through
	// the copy.
	if idn, ok := rhs.(*ir.Ident); ok {
		if tok, ok := e.varLitId[idn.Name]; ok {
			if li, ok := lhs.(*ir.Ident); ok {
				e.varLitId[li.Name] = tok
			}
		}
	}
	// The case list of a select statement has no Go type, thus it takes the
	// plain form of an assignment.
	if t == nil {
		e.line("%s = %s;", e.expr(lhs), value)
		return
	}
	if idx, ok := lhs.(*ir.Index); ok && isMapType(idx.X.GoType()) {
		e.line("%s.set(%s, %s);", e.expr(idx.X), e.expr(idx.Index), value)
		return
	}
	// The Dart form of a pointer to a type parameter holds no field, thus the
	// write reads the description of the type argument.
	if u, ok := lhs.(*ir.Unary); ok && u.Op == token.MUL {
		if p, ok := typeset.Core(u.X.GoType()).(*types.Pointer); ok {
			if tp, ok := types.Unalias(p.Elem()).(*types.TypeParam); ok {
				e.line("$setDeref<%s>(%s, %s, %s);", e.dartType(tp), e.expr(u.X),
					value, e.dictOf(tp))
				return
			}
		}
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
		if head, tail := e.indexTarget(idx); head != "" {
			e.line("%s, %s%s);", head, value, tail)
			return
		}
	}
	e.line("%s = %s;", e.expr(lhs), value)
}

// goStmt starts a goroutine. Go waits for the scheduler, thus the body does not
// run at the point of the statement.
// deferredId gives the token of the function that a defer statement pushes,
// and 0 when the compiler holds no tree of it.
func (e *emitter) deferredId(fun ir.Expr) int {
	switch fun := fun.(type) {
	case *ir.Ident:
		if f, ok := e.funcByObj[fun.Obj]; ok {
			return e.funcId(f)
		}
		// A machine hoists the literal of a deferred call into a variable.
		if id, ok := e.varLitId[fun.Name]; ok {
			return id
		}
	case *ir.Method:
		if f, ok := e.funcByObj[fun.Obj]; ok {
			return e.funcId(f)
		}
	case *ir.FuncLit:
		return e.funcId(fun)
	}
	return 0
}

func (e *emitter) goStmt(s *ir.Go) {
	e.line("$go(() { %s; });", e.expr(s.Call))
}

func (e *emitter) ifStmt(s *ir.If) {
	e.line("if (%s) {", e.expr(s.Cond))
	e.list(s.Then.List)
	for s.Else != nil {
		next, ok := s.Else.(*ir.If)
		if !ok {
			e.line("} else {")
			e.stmt(s.Else)
			break
		}
		// A chain of else-if branches stays flat, because the Dart front end
		// caps the nesting of a statement.
		e.mark(next.Pos)
		e.line("} else if (%s) {", e.expr(next.Cond))
		e.list(next.Then.List)
		s = next
	}
	e.line("}")
}

func (e *emitter) forStmt(s *ir.For) {
	init, initFits := e.forClause(s.Init)
	post, postFits := e.forClause(s.Post)
	if initFits && postFits {
		cond := ""
		if s.Cond != nil {
			cond = e.expr(s.Cond)
		}
		e.writeFor(s, init, cond, post, "")
		return
	}
	e.wideFor(s, init, initFits, postFits)
}

// wideFor writes a loop whose first clause or third clause needs more than 1
// statement. A Go assignment of 2 variables reads each right side first, thus
// the normalizer gives it a temporary variable and a block.
//
// The third clause moves to the top of the body, behind a flag. A continue
// statement of Go runs the third clause, and a continue statement of Dart
// leaves the body, thus the clause cannot stay at the end of the body.
func (e *emitter) wideFor(s *ir.For, init string, initFits, postFits bool) {
	e.line("{")
	if !initFits {
		e.stmtList(s.Init)
		init = ""
	}
	if postFits {
		post, _ := e.forClause(s.Post)
		cond := ""
		if s.Cond != nil {
			cond = e.expr(s.Cond)
		}
		e.writeFor(s, init, cond, post, "")
		e.line("}")
		return
	}
	e.labelSeq++
	flag := fmt.Sprintf("$post%d", e.labelSeq)
	e.line("bool %s = false;", flag)
	e.writeFor(s, init, "", "", flag)
	e.line("}")
}

// writeFor writes the loop and its body. A flag that is not empty moves the
// third clause and the condition to the top of the body.
func (e *emitter) writeFor(s *ir.For, init, cond, post, flag string) {
	if s.Label != "" {
		e.line("%s:", s.Label)
	}
	e.line("for (%s; %s; %s) {", init, cond, post)
	if flag != "" {
		e.line("if (%s) {", flag)
		e.stmtList(s.Post)
		e.line("}")
		e.line("%s = true;", flag)
		if s.Cond != nil {
			e.line("if (!(%s)) break;", e.expr(s.Cond))
		}
	}
	e.breakables = append(e.breakables, "")
	e.list(s.Body.List)
	e.breakables = e.breakables[:len(e.breakables)-1]
	e.line("}")
}

// stmtList writes a statement with no block of its own. A variable of the
// statement stays visible to the code that follows it.
func (e *emitter) stmtList(s ir.Stmt) {
	if block, ok := s.(*ir.Block); ok {
		e.list(block.List)
		return
	}
	e.stmt(s)
}

// forClause writes the first clause or the third clause of a Dart for loop. It
// reports whether the statement fits such a clause.
func (e *emitter) forClause(s ir.Stmt) (string, bool) {
	switch s := s.(type) {
	case nil:
		return "", true
	case *ir.Declare:
		if len(s.Vars) != 1 {
			return "", false
		}
		v := s.Vars[0]
		if s.Init == nil {
			return e.declareText(v, e.zero(v.Type)), true
		}
		return e.declareText(v, e.expr(s.Init)), true
	case *ir.Assign:
		if s.Define {
			return "", false
		}
		if isStructType(s.Lhs.GoType()) {
			return fmt.Sprintf("%s.$assign(%s)", e.expr(s.Lhs), e.expr(s.Rhs)), true
		}
		return fmt.Sprintf("%s = %s", e.expr(s.Lhs), e.expr(s.Rhs)), true
	case *ir.ExprStmt:
		return e.statementExpr(s.X), true
	}
	return "", false
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
				// A switch with no tag compares each case with true. A case
				// of an interface type needs that comparison, and a plain
				// bool is its own condition.
				if isInterfaceType(v.GoType()) {
					parts[k] = e.expr(v) + " == Iface($typeBool, true)"
					continue
				}
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
		e.list(c.Body.List)
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
	if e.machine != nil {
		e.machineReturn(s)
		return
	}
	if e.returnLabel != "" {
		e.namedReturn(s)
		return
	}
	if e.deferPlain {
		// Go evaluates the values of the return first, and the deferred calls
		// run after that.
		switch len(s.Values) {
		case 0:
			e.line("$defer.run();")
			e.line("return;")
		case 1:
			e.line("{ final $ret = %s; $defer.run(); return $ret; }", e.expr(s.Values[0]))
		default:
			parts := make([]string, len(s.Values))
			for i, v := range s.Values {
				parts[i] = e.expr(v)
			}
			e.line("{ final $ret = (%s); $defer.run(); return $ret; }",
				strings.Join(parts, ", "))
		}
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
			if r.Boxed {
				e.fail("the address of the result %s needs a box, and a return "+
					"of multiple values writes no box yet", r.Name)
			}
			parts[i] = r.Name
		}
		e.line("(%s) = %s;", strings.Join(parts, ", "), e.expr(s.Values[0]))
	default:
		for i, v := range s.Values {
			if i >= len(e.returnVars) {
				break
			}
			r := e.returnVars[i]
			e.assign(&ir.Ident{Name: r.Name, Type: r.Type, Boxed: r.Boxed}, v)
		}
	}
	if e.deferPlain {
		e.line("$defer.run();")
	}
	e.line("break %s;", e.returnLabel)
}

package emit

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
)

// A function that can block writes each local into a frame and gives up control
// with a plain return. This file writes that form: the frame class, the
// preamble that reads a frame, the states, and the epilogue that writes one.

// machineCtx holds the machine of the function that the emitter writes.
type machineCtx struct {
	m       *ir.Machine
	results []*ir.Var
	// fn names the local function of a literal. The frame keeps it, because a
	// literal has no name at package level.
	fn string
	// locals holds the name of each local of the frame.
	locals map[string]bool
}

// capture holds the copy of 1 box that a state declares.
type capture struct {
	name, from, typ string
}

// captureRenames registers a copy of each box that a literal of a machine
// reads, and it gives the name of each copy.
//
// The machine writes a new box at each pass of a loop, thus a literal that
// reads the local of the machine reads the box of the last pass. The state
// declares a copy in front of the statement, and the body of the literal reads
// that copy.
func (e *emitter) captureRenames(vars []*ir.Var) map[string]string {
	if e.machine == nil || len(vars) == 0 {
		return nil
	}
	var out map[string]string
	for _, v := range vars {
		if !e.machine.locals[v.Name] || e.renames[v.Name] != "" {
			continue
		}
		if out == nil {
			out = map[string]string{}
		}
		name := "$c" + v.Name
		out[v.Name] = name
		// 2 literals of 1 statement read 1 copy, thus the list holds each name
		// 1 time.
		if !hasCapture(e.captures, name) {
			e.captures = append(e.captures,
				capture{name: name, from: v.Name, typ: e.fieldType(v)})
		}
	}
	return out
}

func hasCapture(list []capture, name string) bool {
	for _, c := range list {
		if c.name == name {
			return true
		}
	}
	return false
}

// withRenames gives the map that a literal of the machine reads.
func (e *emitter) withRenames(add map[string]string) map[string]string {
	if len(add) == 0 {
		return e.renames
	}
	out := map[string]string{}
	for k, v := range e.renames {
		out[k] = v
	}
	for k, v := range add {
		out[k] = v
	}
	return out
}

// frameType gives the Dart type of a frame with its type arguments.
func (e *emitter) frameType(m *ir.Machine, tps []*types.TypeParam) string {
	if len(tps) == 0 {
		return m.Class
	}
	parts := make([]string, len(tps))
	for i, tp := range tps {
		parts[i] = typeParamName(tp)
	}
	return m.Class + "<" + strings.Join(parts, ", ") + ">"
}

// frameParam gives the last parameter of a function that can block. A resume
// passes the frame of the call, and the first call passes nothing.
func (e *emitter) frameParam(m *ir.Machine, tps []*types.TypeParam) string {
	if m == nil || !m.Blocking {
		return ""
	}
	return "[" + e.frameType(m, tps) + "? $f]"
}

// fieldType gives the Dart type of 1 field of a frame.
func (e *emitter) fieldType(v *ir.Var) string {
	if v.Dart != "" {
		return v.Dart
	}
	if v.Boxed {
		return "Box<" + e.dartType(v.Type) + ">"
	}
	return e.dartType(v.Type)
}

// localZero gives the first value of 1 variable of a machine.
func (e *emitter) localZero(v *ir.Var) string {
	if v.DartZero != "" {
		return v.DartZero
	}
	if v.Boxed {
		return "Box(" + e.zero(v.Type) + ")"
	}
	return e.zero(v.Type)
}

// writeFrameClass queues the class of 1 frame. A part file writes it at package
// level, because Dart holds no class inside a function and no class inside
// another class.
func (e *emitter) writeFrameClass(m *ir.Machine, tps []*types.TypeParam,
	fnType, invoke string, hasResult bool) {
	var b strings.Builder
	w := func(format string, args ...any) {
		if len(args) == 0 {
			b.WriteString(format)
		} else {
			fmt.Fprintf(&b, format, args...)
		}
		b.WriteByte('\n')
	}
	w("final class %s%s implements Frame {", m.Class, e.typeParamDecl(tps))
	w("int $s = 0;")
	w("bool $c = false;")
	w("Frame? $r;")
	if m.Defer {
		w("late DeferStack $defer;")
	}
	if m.DeferBlocks {
		w("Object? $p;")
	}
	if fnType != "" {
		w("late %s $fn;", fnType)
	}
	// A local function reads the dictionary of the enclosing function through
	// its closure, thus its frame holds no dictionary. The field $fn marks
	// such a frame.
	if fnType == "" {
		for _, tp := range tps {
			w("late Dict<%s> $d%s;", e.dartType(tp), typeParamName(tp))
		}
	}
	for _, v := range m.Locals {
		w("late %s %s;", e.fieldType(v), v.Name)
	}
	w("@override")
	if hasResult {
		w("Object? resume() => %s;", invoke)
	} else {
		w("Object? resume() {")
		w("%s;", invoke)
		w("return null;")
		w("}")
	}
	w("}")
	w("")
	e.frames = append(e.frames, b.String())
}

// frameArgs lists the arguments that a resume gives the function again.
func (e *emitter) frameArgs(tps []*types.TypeParam, params []*ir.Var) []string {
	out := make([]string, 0, len(tps)+len(params)+1)
	for _, tp := range tps {
		n := typeParamName(tp)
		out = append(out, "$d"+n)
	}
	for _, p := range params {
		if p.Boxed {
			out = append(out, p.Name+".v")
			continue
		}
		out = append(out, p.Name)
	}
	return append(out, "this")
}

// machineBody writes the body of a function whose form is a state machine.
func (e *emitter) machineBody(f *ir.Func, fn string) {
	m := f.Machine
	prev := e.machine
	locals := make(map[string]bool, len(m.Locals))
	for _, v := range m.Locals {
		locals[v.Name] = true
	}
	e.machine = &machineCtx{m: m, results: f.Results, fn: fn, locals: locals}
	e.boxParams(f.Params)
	skip := map[*ir.Var]bool{f.Recv: true}
	for _, p := range f.Params {
		skip[p] = true
	}
	for _, v := range m.Locals {
		if skip[v] {
			continue
		}
		e.line("%s %s = %s;", e.fieldType(v), v.Name, e.localZero(v))
	}
	e.line("int $s = 0;")
	if m.Blocking {
		e.line("bool $c = false;")
		e.line("Frame? $r;")
	}
	if m.Defer {
		e.line("DeferStack $defer = DeferStack();")
	}
	if m.DeferBlocks {
		e.line("Object? $p;")
	}
	if m.Blocking {
		e.line("if ($f != null) {")
		e.line("$s = $f.$s;")
		e.line("$c = $f.$c;")
		e.line("$r = $f.$r;")
		if m.Defer {
			e.line("$defer = $f.$defer;")
		}
		if m.DeferBlocks {
			e.line("$p = $f.$p;")
		}
		for _, v := range m.Locals {
			e.line("%s = $f.%s;", v.Name, v.Name)
		}
		e.line("}")
	}
	// A machine whose deferred calls can block holds the try inside the loop.
	// A panic then jumps to the chain of the deferred calls, and such a call
	// gives up control exactly as each other call does.
	if m.Defer && !m.DeferBlocks {
		e.line("try {")
	}
	e.line("$m: while (true) {")
	if m.DeferBlocks {
		e.line("try {")
	}
	e.line("switch ($s) {")
	for i, state := range m.States {
		e.line("case %d:", i)
		e.list(state.List)
	}
	e.line("}")
	if m.DeferBlocks {
		e.line("} catch ($e) {")
		e.line("if ($defer.isEmpty || $p != null) rethrow;")
		e.line("$p = $e;")
		e.line("$s = %d;", m.DeferState)
		e.line("continue $m;")
		e.line("}")
	}
	e.line("}")
	if m.Blocking {
		e.machineSave(f, fn)
	}
	if m.Defer && !m.DeferBlocks {
		e.line("} catch ($e) {")
		e.line("$defer.run();")
		e.line("rethrow;")
		e.line("}")
	}
	e.machine = prev
}

// deferNext writes the state that runs the next deferred call.
func (e *emitter) deferNext(s *ir.DeferNext) {
	e.line("if ($defer.isEmpty) { $s = %d; continue $m; }", s.Done)
	e.line("$defer.runNext();")
	e.line("$s = %d;", s.Body)
	e.line("continue $m;")
}

// deferEnd writes the end of the defer chain. A panic that the machine caught
// reaches the caller here.
func (e *emitter) deferEnd() {
	e.line("if ($p != null) { final Object $pv = $p!; $p = null; throw $pv; }")
}

// machineSave writes the frame of a call that gave up control.
func (e *emitter) machineSave(f *ir.Func, fn string) {
	m := f.Machine
	tps := f.TypeParams
	e.line("$blocked = %s()", e.frameType(m, tps))
	e.line("..$s = $s")
	e.line("..$c = true")
	e.line("..$r = $blocked")
	if m.Defer {
		e.line("..$defer = $defer")
	}
	if m.DeferBlocks {
		e.line("..$p = $p")
	}
	if fn != "" {
		e.line("..$fn = %s", fn)
	}
	// The frame of a local function holds no dictionary. Read writeFrameClass.
	if fn == "" {
		for _, tp := range tps {
			n := typeParamName(tp)
			e.line("..$d%s = $d%s", n, n)
		}
	}
	for _, v := range m.Locals {
		e.line("..%s = %s", v.Name, v.Name)
	}
	e.line(";")
	e.line("return %s;", e.zeroResult(f.Results))
}

// zeroResult gives a result that no code reads. A call that gave up control
// gives it, and the caller drops it.
func (e *emitter) zeroResult(results []*ir.Var) string {
	switch len(results) {
	case 0:
		return ""
	case 1:
		return e.zero(results[0].Type)
	}
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = e.zero(r.Type)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// machineReturn writes a return statement of a machine. Each deferred call runs
// in front of the return, thus such a call can still write a named result.
func (e *emitter) machineReturn(s *ir.Return) {
	ctx := e.machine
	named := hasNamedResults(ctx.results)
	switch {
	case named:
		e.writeNamedResults(s, ctx.results)
	case len(s.Values) == 0 && len(ctx.results) > 0:
		// Go proves that this point is unreachable, and Dart cannot.
		e.line("$unreachable();")
		return
	}
	// A machine whose deferred calls can block runs the chain in its own
	// states, thus the return statement of the chain runs no call.
	if ctx.m.Defer && !ctx.m.DeferBlocks {
		e.line("$defer.run();")
	}
	if named {
		e.line("return %s;", e.resultRead(ctx.results))
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

func hasNamedResults(results []*ir.Var) bool {
	for _, r := range results {
		if r.Named {
			return true
		}
	}
	return false
}

// writeNamedResults writes the values of a return statement into the result
// variables.
func (e *emitter) writeNamedResults(s *ir.Return, results []*ir.Var) {
	switch {
	case len(s.Values) == 0:
	case len(s.Values) == 1 && len(results) > 1:
		parts := make([]string, len(results))
		for i, r := range results {
			parts[i] = varRead(r)
		}
		e.line("(%s) = %s;", strings.Join(parts, ", "), e.expr(s.Values[0]))
	default:
		for i, v := range s.Values {
			if i >= len(results) {
				break
			}
			e.line("%s = %s;", varRead(results[i]), e.expr(v))
		}
	}
}

// resume writes the re-entry of the call of the state before it.
func (e *emitter) resume(s *ir.Resume) {
	if s.Var != nil {
		e.line("if ($c) { $c = false; %s = $r!.resume() as %s; }",
			s.Var.Name, e.fieldType(s.Var))
	} else {
		e.line("if ($c) { $c = false; $r!.resume(); }")
	}
	e.line("if ($blocked != null) break $m;")
}

// localFunc writes a function literal that can block. Such a literal becomes a
// local function, because its frame must name it.
func (e *emitter) localFunc(s *ir.LocalFunc) {
	lit := s.Lit
	m := lit.Machine
	prevNames := e.renames
	e.renames = e.withRenames(e.captureRenames(lit.Captured))
	defer func() { e.renames = prevNames }()
	// The frame class stands at package level, thus it takes the type
	// parameters of the enclosing function. A local of the literal can hold a
	// value of such a parameter.
	tps := e.typeParams
	params := e.paramList(lit.Params)
	if p := e.frameParam(m, tps); p != "" {
		params = append(params, p)
	}
	result := e.resultText(lit.Results)
	e.line("%s %s(%s) {", result, s.Name, strings.Join(params, ", "))
	inner := &ir.Func{
		Params:     lit.Params,
		Results:    lit.Results,
		Body:       lit.Body,
		Machine:    m,
		TypeParams: tps,
	}
	e.machineBody(inner, s.Name)
	e.line("}")
	types := make([]string, len(lit.Params))
	for i, p := range lit.Params {
		types[i] = e.dartType(p.Type)
	}
	frame := e.frameType(m, tps)
	fnType := fmt.Sprintf("%s Function(%s, [%s?])", result,
		strings.Join(types, ", "), frame)
	if len(types) == 0 {
		fnType = fmt.Sprintf("%s Function([%s?])", result, frame)
	}
	invoke := fmt.Sprintf("$fn(%s)", strings.Join(e.frameArgs(nil, lit.Params), ", "))
	e.writeFrameClass(m, tps, fnType, invoke, len(lit.Results) > 0)
}

// selectBind writes the value and the flag of a receive case of a select
// statement. Each variable of the case lives in the frame, thus the statement
// writes it and does not declare it.
func (e *emitter) selectBind(s *ir.SelectBind) {
	list := e.expr(s.List)
	read := fmt.Sprintf("(%s[%d] as RecvCase<%s>).result!",
		list, s.Index, e.dartType(s.Clause.Elem))
	fields := []string{read + ".$1", read + ".$2"}
	for i, v := range s.Clause.Vars {
		if i >= len(fields) || v.Name == "_" {
			continue
		}
		if v.Boxed {
			// The case declares its variable, thus each pass takes a fresh
			// box, and a pointer of an earlier pass keeps its value.
			e.line("%s = Box(%s);", v.Name, fields[i])
			continue
		}
		e.line("%s = %s;", varRead(v), fields[i])
	}
	for i, lhs := range s.Clause.Lhs {
		if i >= len(fields) {
			continue
		}
		e.line("%s = %s;", e.expr(lhs), fields[i])
	}
}

// selectList writes the case list of a select statement.
func (e *emitter) selectList(x *ir.SelectList) string {
	parts := make([]string, 0, len(x.Cases))
	for _, c := range x.Cases {
		switch c.Kind {
		case ir.SelectRecv:
			parts = append(parts, fmt.Sprintf("RecvCase<%s>(%s)",
				e.dartType(c.Elem), e.expr(c.Chan)))
		case ir.SelectSend:
			parts = append(parts, fmt.Sprintf("SendCase<%s>(%s, %s)",
				e.dartType(c.Elem), e.expr(c.Chan), e.expr(c.Value)))
		}
	}
	return "<SelectCase>[" + strings.Join(parts, ", ") + "]"
}

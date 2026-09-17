package emit

import (
	"fmt"
	"go/token"
	"go/types"
	"strconv"
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
	// end holds the closing brace of the body. The run of the defer chain on
	// the panic path and at a fall off the end marks that line.
	end token.Pos
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
	fnType, invoke string, hasResult bool, goName, goFile string) {
	// A frame of a local function resumes under the token of the function,
	// because the call gave up control inside the chain of a defer
	// statement, and a recover after the yield still reads the chain panic.
	tok := e.curFuncId
	var b strings.Builder
	w := func(format string, args ...any) {
		if len(args) == 0 {
			b.WriteString(format)
		} else {
			fmt.Fprintf(&b, format, args...)
		}
		b.WriteByte('\n')
	}
	w("final class %s%s implements TraceFrame {", m.Class, e.typeParamDecl(tps))
	w("int $s = 0;")
	w("bool $c = false;")
	w("Frame? $r;")
	w("static const List<int> $lines = <int>[%s];", e.stateLines(m))
	w("@override")
	w("String get $name => %s;", dartString(goName))
	w("@override")
	w("String get $goFile => %s;", dartString(goFile))
	w("@override")
	w("int get $goLine => $s >= 0 && $s < $lines.length ? $lines[$s] : 0;")
	w("@override")
	w("Frame? get $next => $r;")
	if m.Defer {
		w("late DeferStack $defer;")
	}
	if m.DeferBlocks {
		w("ChainState? $chain;")
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
		w("Object? resume() => $dcall<Object?>(%d, () => %s);", tok, invoke)
	} else {
		w("Object? resume() {")
		w("$dcall<Object?>(%d, () => %s);", tok, invoke)
		w("return null;")
		w("}")
	}
	w("}")
	w("")
	// placeLocalFuncs of stage 4 writes 1 declaration of a literal into each
	// state that reads it, because a case clause of a Dart switch holds a
	// scope of its own. The class stands at package level, thus the file takes
	// 1 copy only. The text above still reads each type of the frame, thus the
	// import of that type reaches the library file.
	if e.frameSeen == nil {
		e.frameSeen = map[string]bool{}
	}
	if e.frameSeen[m.Class] {
		return
	}
	e.frameSeen[m.Class] = true
	e.frames = append(e.frames, b.String())
}

// stateLines writes the source line of each state of a machine. The line of a
// state is the line of its first statement that holds a position. The trace of
// a goroutine that waits reads the line of the state that resumes.
func (e *emitter) stateLines(m *ir.Machine) string {
	if e.cfg.Fset == nil {
		return ""
	}
	parts := make([]string, len(m.States))
	for i, s := range m.States {
		parts[i] = "0"
		for _, st := range s.List {
			if p := ir.PosOf(st); p.IsValid() {
				parts[i] = strconv.Itoa(e.cfg.Fset.Position(p).Line)
				break
			}
		}
	}
	return strings.Join(parts, ", ")
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
	e.machine = &machineCtx{m: m, results: f.Results, fn: fn, locals: locals,
		end: f.End}
	e.boxParams(f.Params)
	if f.Recv != nil && !e.inClass[f] {
		e.boxRecv(f.Recv)
	}
	skip := map[*ir.Var]bool{f.Recv: true}
	for _, p := range f.Params {
		skip[p] = true
	}
	for _, v := range m.Locals {
		if skip[v] {
			continue
		}
		// Array temporaries can receive an existing array before any read.
		// A lazy zero avoids allocation but still permits frame saves before assignment.
		if v.Dart == "" && !v.Boxed && isArrayType(v.Type) {
			e.line("late %s %s = %s;", e.fieldType(v), v.Name, e.localZero(v))
		} else {
			e.line("%s %s = %s;", e.fieldType(v), v.Name, e.localZero(v))
		}
	}
	e.line("int $s = 0;")
	if m.Blocking {
		e.line("bool $c = false;")
		e.line("Frame? $r;")
	}
	if m.Defer {
		e.line("DeferStack $defer = DeferStack();")
	}
	// The chain of the deferred calls of such a machine can give up control.
	// The state of the chain thus rides the frame, and not a local.
	if m.DeferBlocks {
		e.line("ChainState? $chain;")
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
			e.line("$chain = $f.$chain;")
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
		e.line("if ($defer.isEmpty) rethrow;")
		// A value that reaches the catch while the chain already stands open
		// must not save the state again. The chain of the caller stands in the
		// value that the first open gave.
		e.line("if ($chain == null) { $chain = $chainOpen($e); } else { $chainReset($e); }")
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
		e.line("final ChainState $chain = $chainOpen($e);")
		e.mark(f.End)
		e.line("if (!$defer.runRecovered()) { $chainClose($chain); rethrow; }")
		e.line("$chainClose($chain);")
		if hasNamedResults(f.Results) {
			e.line("return %s;", e.resultRead(f.Results))
		} else {
			e.line("return %s;", e.zeroResult(f.Results))
		}
		e.line("}")
	}
	e.machine = prev
}

// deferNext writes the state that runs the next deferred call.
func (e *emitter) deferNext(s *ir.DeferNext) {
	// A return of the function also reaches this state, and the chain of the
	// caller still stands open then. The chain of this function must hold no
	// panic in that case, thus a recover inside it gives null. The open stands
	// in front of the test, because the end of the chain always closes it.
	//
	// The name of the variable holds no form that a capture can take. A
	// capture writes "$c" and the Go name. A local named s thus took the
	// shorter name of the chain before.
	e.line("$chain ??= $chainOpen(null);")
	e.line("if ($defer.isEmpty) { $s = %d; continue $m; }", s.Done)
	// Each exit of the function shares this state, thus the frame of a
	// deferred call names the closing brace. Go names the line of a return
	// statement when the exit is a return, and the shared state cannot.
	if e.machine != nil {
		e.mark(e.machine.end)
	}
	e.line("$defer.runNext();")
	e.line("$s = %d;", s.Body)
	e.line("continue $m;")
}

// deferEnd writes the end of the defer chain. A panic that the chain holds
// reaches the caller here, unless a deferred call recovered it.
func (e *emitter) deferEnd() {
	e.line("final ChainState $chainOuter = $chain!;")
	e.line("$chain = null;")
	e.line("if ($chainPanic != null && !$recovered) {")
	e.line("final GoPanic $pv = $chainPanic!;")
	e.line("$chainClose($chainOuter);")
	e.line("throw $pv;")
	e.line("}")
	e.line("$chainClose($chainOuter);")
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
		e.line("..$chain = $chain")
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
	runs := ctx.m.Defer && !ctx.m.DeferBlocks
	if runs && !s.Pos.IsValid() {
		// A return that stage 4 built for a fall off the end holds no
		// position, and Go names the closing brace there.
		e.mark(ctx.end)
	}
	if named {
		if runs {
			e.line("$defer.run();")
		}
		e.line("return %s;", e.resultRead(ctx.results))
		return
	}
	// Go evaluates each value of the return first, and the deferred calls run
	// after that. A value that a deferred call changes thus keeps the answer of
	// Go. A named result reads the other order, because such a call writes it.
	switch len(s.Values) {
	case 0:
		if runs {
			e.line("$defer.run();")
		}
		e.line("return;")
	case 1:
		if runs {
			e.line("{ final $ret = %s; $defer.run(); return $ret; }",
				e.expr(s.Values[0]))
			return
		}
		e.line("return %s;", e.expr(s.Values[0]))
	default:
		parts := make([]string, len(s.Values))
		for i, v := range s.Values {
			parts[i] = e.expr(v)
		}
		if runs {
			e.line("{ final $ret = (%s); $defer.run(); return $ret; }",
				strings.Join(parts, ", "))
			return
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
	case len(s.Values) > 1:
		// Go reads each operand first and it writes the result variables after
		// that, thus an operand that panics leaves each variable with its old
		// value.
		parts := make([]string, 0, len(s.Values))
		for i, v := range s.Values {
			if i >= len(results) {
				break
			}
			parts = append(parts, e.expr(v))
		}
		e.labelSeq++
		tmp := fmt.Sprintf("$rv%d", e.labelSeq)
		e.line("final %s = (%s);", tmp, strings.Join(parts, ", "))
		for i := range parts {
			e.line("%s = %s.$%d;", varRead(results[i]), tmp, i+1)
		}
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
	if s.Select {
		e.line("if ($c) { $c = false; %s = $gi($r!.resume() as int); }", s.Var.Name)
	} else if s.Var != nil {
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
	// The chain of the deferred calls runs the local function of a defer
	// statement under its token, and recover reads it.
	saved := e.curFuncId
	e.curFuncId = e.funcId(s.Name)
	defer func() { e.curFuncId = saved }()
	e.varLitId[s.Name] = e.curFuncId
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
		End:        lit.End,
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
	// A second declaration of 1 literal takes no number of its own, because
	// writeFrameClass drops its class. Go numbers each literal 1 time.
	if !e.frameSeen[m.Class] {
		e.litSeq++
	}
	e.writeFrameClass(m, tps, fnType, invoke, len(lit.Results) > 0,
		fmt.Sprintf("%s.func%d", e.curGoName, e.litSeq), e.funcFile)
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

package normalize

import (
	"go/token"
	"go/types"
	"strconv"

	"github.com/eddyw/godart/internal/ir"
)

// This file gives an expression the order of the evaluation of gc.
//
// The Go specification orders 4 operations of an expression from left to
// right: a function call, a method call, a receive operation, and a binary
// logical operation. It leaves the order of each other operand open.
//
// gc closes that gap 1 way. Its order pass writes each such operation into a
// temporary variable in front of the statement, thus each other operand reads
// its value after the operations ran. Real Go source depends on that order.
// `crypto/x509.ParseOID` holds `return o, o.unmarshalOIDText(oid)`, and the
// method writes the value that the first result reads.
//
// The pass walks the operands of 1 statement in the order of the evaluation.
// It marks each read that no operation consumed. A call behind such a read
// must run first, thus the pass writes each operation of that statement into
// a temporary variable, in lexical order. A statement that holds no such read
// keeps its form, and the output then holds no extra variable. Only that 1
// shape differs. A call in front of a read already matches, because both
// forms run the call first.
//
// The set of the operations follows gc and not the specification, because gc
// writes a variable for more than the 4. `orderedBuiltin` holds the builtins,
// and a slice expression takes 1 too.
//
// The pass leaves the 3 clauses of a for loop, the value of a case, and the
// initializer of a package variable. No Dart statement can stand in front of
// such a position. `docs/parity.md` holds a witness for each 1, and
// decision 324 holds the whole rule.

// orderer rewrites the expressions of 1 statement.
type orderer struct {
	b   *builder
	pos token.Pos

	// pre holds the declaration of each temporary variable, in the order of
	// the evaluation.
	pre []ir.Stmt

	// stack holds 1 flag for each operation that the walk entered. The flag
	// marks an operand of that operation that the walk read. The operation
	// consumes such a read only when it runs, thus a call behind the read
	// must run in front of it.
	stack []bool

	// pend counts the flags of the stack that stand.
	pend int

	// apply marks the second pass. The first pass reports only whether the
	// order of the statement differs from the order of gc.
	apply bool

	// need marks a call that stands behind a read.
	need bool
}

// order writes each call of a statement into a temporary variable when the
// order of the operands differs from the order of gc. The result holds each
// declaration in front of the statement that needs it.
func (b *builder) order(list []ir.Stmt, pos token.Pos) []ir.Stmt {
	if b.noOrder > 0 {
		return list
	}
	var out []ir.Stmt
	for i, s := range list {
		pre := b.orderStmt(s, pos)
		if len(pre) == 0 {
			if out != nil {
				out = append(out, s)
			}
			continue
		}
		if out == nil {
			out = append(out, list[:i]...)
		}
		out = append(out, pre...)
		out = append(out, s)
	}
	if out == nil {
		return list
	}
	return out
}

// orderStmt reads the operands of 1 statement. A statement whose operands run
// more than 1 time keeps its form, thus the walk reaches no loop and no case.
func (b *builder) orderStmt(s ir.Stmt, pos token.Pos) []ir.Stmt {
	switch s := s.(type) {
	case *ir.Return:
		return b.orderRun(pos, func(o *orderer) { o.list(s.Values) })
	case *ir.ExprStmt:
		return b.orderRun(pos, func(o *orderer) { s.X = o.expr(s.X) })
	case *ir.Declare:
		return b.orderRun(pos, func(o *orderer) { s.Init = o.expr(s.Init) })
	case *ir.Assign:
		return b.orderRun(pos, func(o *orderer) {
			// A write to a plain name reads no value, thus a call of the
			// right side needs no variable. gc holds the same rule.
			if _, name := s.Lhs.(*ir.Ident); !name {
				s.Lhs = o.expr(s.Lhs)
			}
			s.Rhs = o.expr(s.Rhs)
		})
	case *ir.AssignMulti:
		return b.orderRun(pos, func(o *orderer) {
			o.list(s.Lhs)
			s.Rhs = o.expr(s.Rhs)
		})
	case *ir.If:
		return b.orderRun(pos, func(o *orderer) { s.Cond = o.expr(s.Cond) })
	case *ir.Send:
		return b.orderRun(pos, func(o *orderer) {
			s.Chan = o.expr(s.Chan)
			s.Value = o.expr(s.Value)
		})
	}
	return nil
}

// orderValues reads a list of operands that stands in front of a statement
// which the stage builds from more than 1 part. A parallel assignment, a
// short variable declaration, a defer statement, and a go statement each hold
// such a list.
func (b *builder) orderValues(list []ir.Expr, pos token.Pos) []ir.Stmt {
	return b.orderRun(pos, func(o *orderer) { o.list(list) })
}

// orderCall reads the callee and each argument of a call that runs later. A
// defer statement and a go statement each read the 2 at the statement, thus
// each call of the list runs in front of the receiver and of each argument.
func (b *builder) orderCall(c *ir.Call, pos token.Pos) []ir.Stmt {
	list := make([]ir.Expr, 0, len(c.Args)+1)
	list = append(list, c.Fun)
	list = append(list, c.Args...)
	out := b.orderValues(list, pos)
	c.Fun = list[0]
	copy(c.Args, list[1:])
	return out
}

// orderRun walks the operands 2 times. The first pass reads only, and the
// second pass writes the temporary variables. A statement whose order already
// matches gc thus keeps each node of its tree.
func (b *builder) orderRun(pos token.Pos, walk func(o *orderer)) []ir.Stmt {
	if b.noOrder > 0 {
		return nil
	}
	o := &orderer{b: b, pos: pos}
	o.push()
	walk(o)
	o.pop()
	if !o.need {
		return nil
	}
	o.apply = true
	o.stack = o.stack[:0]
	o.pend = 0
	o.push()
	walk(o)
	o.pop()
	return o.pre
}

// push opens the frame of 1 operation.
func (o *orderer) push() { o.stack = append(o.stack, false) }

// pop closes the frame of 1 operation. It reports whether an operand of that
// operation read a value.
func (o *orderer) pop() bool {
	i := len(o.stack) - 1
	flag := o.stack[i]
	if flag {
		o.pend--
	}
	o.stack = o.stack[:i]
	return flag
}

// mark records a read in the frame of the operation that consumes it.
func (o *orderer) mark() {
	i := len(o.stack) - 1
	if i < 0 || o.stack[i] {
		return
	}
	o.stack[i] = true
	o.pend++
}

func (o *orderer) list(list []ir.Expr) {
	for i, e := range list {
		list[i] = o.expr(e)
	}
}

// expr walks 1 expression in the order of the evaluation.
func (o *orderer) expr(x ir.Expr) ir.Expr {
	switch x := x.(type) {
	case nil:
		return nil
	case *ir.Const, *ir.Zero, *ir.Blank, *ir.FuncLit, *ir.MethodExpr, *ir.FuncInst:
		return x
	case *ir.Ident:
		if !stableName(x) {
			o.mark()
		}
		return x
	case *ir.Call:
		return o.operation(x, func() {
			x.Fun = o.expr(x.Fun)
			x.Results = o.expr(x.Results)
			o.list(x.Args)
		})
	case *ir.Recv:
		return o.operation(x, func() { x.Chan = o.expr(x.Chan) })
	case *ir.Slice:
		return o.operation(x, func() {
			x.X = o.expr(x.X)
			x.Low = o.expr(x.Low)
			x.High = o.expr(x.High)
			x.Max = o.expr(x.Max)
		})
	case *ir.Builtin:
		if !orderedBuiltin[x.Name] {
			break
		}
		return o.operation(x, func() {
			x.Results = o.expr(x.Results)
			o.list(x.Args)
		})
	case *ir.Binary:
		if x.Op != token.LAND && x.Op != token.LOR {
			break
		}
		// The specification orders a binary logical operation exactly as it
		// orders a call, and gc writes such an operation into a temporary
		// variable. The right side runs only for 1 value of the left side,
		// thus it keeps its place inside the operator. Dart holds the same
		// short circuit.
		return o.operation(x, func() { x.X = o.expr(x.X) })
	}
	o.push()
	o.children(x)
	if o.pop() {
		o.mark()
	}
	return x
}

// operation walks 1 operation that the order pass can write into a temporary
// variable. The result of such an operation is a fresh value, thus it marks no
// read of the operation that holds it.
func (o *orderer) operation(x ir.Expr, walk func()) ir.Expr {
	take := hoistable(x.GoType())
	if !o.apply && take && o.pend > 0 {
		o.need = true
	}
	o.push()
	walk()
	o.pop()
	if o.apply && take {
		return o.hoist(x)
	}
	return x
}

// orderedBuiltin lists each builtin that gc writes into a temporary variable
// exactly as it writes a call. `cmd/compile/internal/walk/order.go` holds the
// list. Each other builtin either gives no value or belongs to the stage.
var orderedBuiltin = map[string]bool{
	"append": true, "cap": true, "complex": true, "copy": true,
	"imag": true, "len": true, "make": true, "max": true,
	"min": true, "new": true, "real": true, "recover": true,
}

// children walks the operands of each other node, in the order of the
// evaluation.
func (o *orderer) children(x ir.Expr) {
	switch x := x.(type) {
	case *ir.Field:
		x.X = o.expr(x.X)
	case *ir.Index:
		x.X = o.expr(x.X)
		x.Index = o.expr(x.Index)
	case *ir.MapRead:
		x.X = o.expr(x.X)
		x.Key = o.expr(x.Key)
	case *ir.Binary:
		x.X = o.expr(x.X)
		x.Y = o.expr(x.Y)
	case *ir.Unary:
		x.X = o.expr(x.X)
	case *ir.Conv:
		x.X = o.expr(x.X)
	case *ir.Copy:
		x.X = o.expr(x.X)
	case *ir.TypeAssert:
		x.X = o.expr(x.X)
	case *ir.TypeTest:
		x.X = o.expr(x.X)
	case *ir.Method:
		x.Recv = o.expr(x.Recv)
	case *ir.Composite:
		x.Results = o.expr(x.Results)
		o.list(x.Elems)
	case *ir.Builtin:
		x.Results = o.expr(x.Results)
		o.list(x.Args)
	case *ir.Record:
		o.list(x.Elems)
	case *ir.RecordField:
		x.X = o.expr(x.X)
	}
}

// hoist writes an operation into a temporary variable in front of the
// statement.
func (o *orderer) hoist(x ir.Expr) ir.Expr {
	o.b.orderN++
	v := &ir.Var{
		Name: o.b.temp(o.pos, "ev") + "$" + strconv.Itoa(o.b.orderN),
		Type: x.GoType(),
	}
	o.pre = append(o.pre, &ir.Declare{Vars: []*ir.Var{v}, Init: x})
	return &ir.Ident{Name: v.Name, Type: v.Type}
}

// stableName reports whether a call can change the value that the name reads.
// A function name holds 1 function, and a temporary of the stage belongs to
// the statement alone.
func stableName(x *ir.Ident) bool {
	if x.Obj == nil {
		return true
	}
	_, ok := x.Obj.(*types.Func)
	return ok
}

// hoistable reports whether 1 variable can hold the result of an operation.
//
// A call with multiple results feeds the whole list of the operands, thus it
// needs no variable of its own. A call with no result stands alone as a
// statement.
func hoistable(t types.Type) bool {
	if t == nil {
		return false
	}
	if _, tuple := t.(*types.Tuple); tuple {
		return false
	}
	if b, basic := types.Unalias(t).(*types.Basic); basic && b.Kind() == types.Invalid {
		return false
	}
	return true
}

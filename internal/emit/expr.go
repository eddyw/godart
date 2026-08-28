package emit

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"math"
	"strconv"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
	"github.com/eddyw/godart/internal/typeset"
)

func (e *emitter) expr(x ir.Expr) string {
	switch x := x.(type) {
	case nil:
		return ""
	case *ir.Const:
		return e.constant(x.Value, x.Type)
	case *ir.Ident:
		return e.identExpr(x)
	case *ir.Blank:
		return "_"
	case *ir.Zero:
		return e.zero(x.Type)
	case *ir.Field:
		return e.deref(x.X) + "." + name.Package(x.Name)
	case *ir.Index:
		return e.index(x)
	case *ir.TypeAssert:
		return e.typeAssert(x)
	case *ir.TypeTest:
		return e.typeTest(x.X, x.Assert)
	case *ir.MapRead:
		return fmt.Sprintf("%s.get2(%s)", e.expr(x.X), e.expr(x.Key))
	case *ir.Slice:
		return e.sliceExpr(x)
	case *ir.Call:
		return e.call(x)
	case *ir.Method:
		return e.methodValue(x)
	case *ir.Builtin:
		return e.builtin(x)
	case *ir.Binary:
		return e.binary(x)
	case *ir.Unary:
		return e.unary(x)
	case *ir.Conv:
		return e.conv(x)
	case *ir.Composite:
		return e.composite(x)
	case *ir.FuncLit:
		return e.funcLit(x)
	case *ir.Copy:
		return e.copyExpr(x)
	case *ir.Recv:
		if x.Comma {
			return fmt.Sprintf("(await %s.receive())", e.expr(x.Chan))
		}
		return fmt.Sprintf("(await %s.receive()).$1", e.expr(x.Chan))
	case *ir.Record:
		parts := make([]string, len(x.Elems))
		for i, el := range x.Elems {
			parts[i] = e.expr(el)
		}
		return "(" + strings.Join(parts, ", ") + ")"
	}
	e.fail("the expression %T has no Dart form", x)
	return "null"
}

func (e *emitter) identExpr(x *ir.Ident) string {
	if x.Obj == nil {
		return x.Name
	}
	if x.Obj.Pkg() != nil && x.Obj.Parent() == x.Obj.Pkg().Scope() {
		return e.qualify(x.Obj, name.Package(x.Obj.Name()))
	}
	return x.Name
}

// deref reads through a pointer. A pointer to a struct is the Dart object
// reference, and the type is nullable, thus the read needs a null check.
func (e *emitter) deref(x ir.Expr) string {
	text := e.expr(x)
	if isPointerType(x.GoType()) {
		return text + "!"
	}
	return text
}

func (e *emitter) constant(v constant.Value, t types.Type) string {
	switch {
	case isString(t):
		return stringConst(constant.StringVal(v))
	case isFloat(t):
		f, _ := constant.Float64Val(constant.ToFloat(v))
		return floatConst(f)
	case isInteger(t):
		return intConst(v, t)
	}
	if v.Kind() == constant.Bool {
		return strconv.FormatBool(constant.BoolVal(v))
	}
	e.fail("the constant %s has no Dart form", v)
	return "null"
}

func intConst(v constant.Value, t types.Type) string {
	i := constant.ToInt(v)
	if n, ok := constant.Int64Val(i); ok {
		if n < 0 {
			return "(" + strconv.FormatInt(n, 10) + ")"
		}
		return strconv.FormatInt(n, 10)
	}
	if n, ok := constant.Uint64Val(i); ok {
		// A value above the range of int64 keeps its 64 bit pattern. Dart
		// reads the pattern as a negative number, and the unsigned helpers
		// give the Go result.
		return strconv.FormatInt(int64(n), 10)
	}
	return "0"
}

func floatConst(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "double.infinity"
	case math.IsInf(f, -1):
		return "double.negativeInfinity"
	case math.IsNaN(f):
		return "double.nan"
	}
	text := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	if f < 0 {
		return "(" + text + ")"
	}
	return text
}

// stringConst writes a Go string constant. The bytes go into the output when
// the text is not valid UTF-8, because a Dart string cannot hold such bytes.
func stringConst(s string) string {
	if !isPrintableUTF8(s) {
		parts := make([]string, len(s))
		for i := 0; i < len(s); i++ {
			parts[i] = strconv.Itoa(int(s[i]))
		}
		return "GoString.fromBytes(<int>[" + strings.Join(parts, ", ") + "])"
	}
	var b strings.Builder
	b.WriteString("GoString.fromString('")
	for _, r := range s {
		switch r {
		case '\'':
			b.WriteString(`\'`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '$':
			b.WriteString(`\$`)
		default:
			if r < 0x20 || r == 0x7F {
				fmt.Fprintf(&b, `\u{%x}`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteString("')")
	return b.String()
}

func isPrintableUTF8(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return validUTF8(s)
		}
	}
	return true
}

func validUTF8(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			return false
		}
	}
	return true
}

// index reads 1 element. A Go array keeps a plain Dart list, thus the read
// needs the bounds check of the runtime.
func (e *emitter) index(x *ir.Index) string {
	container := x.X.GoType()
	if isMapType(container) {
		return fmt.Sprintf("%s.get(%s)", e.expr(x.X), e.expr(x.Index))
	}
	if isPointerType(container) {
		return fmt.Sprintf("$arrayIndex(%s, %s)", e.deref(x.X), e.expr(x.Index))
	}
	if isArrayType(container) {
		return fmt.Sprintf("$arrayIndex(%s, %s)", e.expr(x.X), e.expr(x.Index))
	}
	return fmt.Sprintf("%s[%s]", e.expr(x.X), e.expr(x.Index))
}

// indexTarget gives the start of a write into an array. The result is empty
// when the write uses the index operator of Dart.
func (e *emitter) indexTarget(x *ir.Index) string {
	container := x.X.GoType()
	if isPointerType(container) {
		return fmt.Sprintf("$arraySet(%s, %s", e.deref(x.X), e.expr(x.Index))
	}
	if isArrayType(container) {
		return fmt.Sprintf("$arraySet(%s, %s", e.expr(x.X), e.expr(x.Index))
	}
	return ""
}

func (e *emitter) assignArray(dst string, t types.Type, value string) string {
	a, _ := types.Unalias(t).Underlying().(*types.Array)
	if a != nil && needsElementCopy(a.Elem()) {
		return fmt.Sprintf("$assignArray(%s, %s, (a, b) => %s)", dst, value, e.assignElem("a", "b", a.Elem()))
	}
	return fmt.Sprintf("$assignArray(%s, %s)", dst, value)
}

func (e *emitter) sliceExpr(x *ir.Slice) string {
	low := "0"
	if x.Low != nil {
		low = e.expr(x.Low)
	}
	source := x.X.GoType()
	if isString(source) {
		high := fmt.Sprintf("%s.length", e.expr(x.X))
		if x.High != nil {
			high = e.expr(x.High)
		}
		return fmt.Sprintf("$substring(%s, %s, %s)", e.expr(x.X), low, high)
	}
	if isArrayType(source) || isPointerType(source) {
		text := e.expr(x.X)
		if isPointerType(source) {
			text = e.deref(x.X)
		}
		high := fmt.Sprintf("%s.length", text)
		if x.High != nil {
			high = e.expr(x.High)
		}
		max := fmt.Sprintf("%s.length", text)
		if x.Max != nil {
			max = e.expr(x.Max)
		}
		return fmt.Sprintf("Slice(%s, 0, %s, %s).range(%s, %s, %s)",
			text, max, max, low, high, max)
	}
	x0 := e.expr(x.X)
	high := fmt.Sprintf("%s.len", x0)
	if x.High != nil {
		high = e.expr(x.High)
	}
	max := fmt.Sprintf("%s.cap", x0)
	if x.Max != nil {
		max = e.expr(x.Max)
	}
	return fmt.Sprintf("%s.range(%s, %s, %s)", x0, low, high, max)
}

func (e *emitter) copyExpr(x *ir.Copy) string {
	if isArrayType(x.Type) {
		return e.copyArray(e.expr(x.X), x.Type)
	}
	return e.deref(x.X) + ".$copy()"
}

func (e *emitter) funcLit(x *ir.FuncLit) string {
	var b strings.Builder
	fmt.Fprintf(&b, "(%s)%s {", e.params(x.Params), asyncWord(x.Async))
	e.buf.WriteString("")
	inner := &emitter{pkg: e.pkg, prefixes: e.prefixes, base: e.base,
		methods: e.methods, inClass: e.inClass, initNames: e.initNames,
		breakables: e.breakables, labelSeq: e.labelSeq}
	inner.emitBody(x.Results, x.Defers, x.Body)
	e.errs = append(e.errs, inner.errs...)
	e.labelSeq = inner.labelSeq
	b.WriteString(inner.buf.String())
	b.WriteString("}")
	return b.String()
}

// equal writes the Go comparison `a == b` for 2 values of the same type.
func (e *emitter) equal(a, b ir.Expr) string {
	if isArrayType(a.GoType()) {
		return fmt.Sprintf("$equalArray(%s, %s)", e.expr(a), e.expr(b))
	}
	if text, ok := e.nilTest(a, b); ok {
		return text
	}
	if text, ok := e.nilTest(b, a); ok {
		return text
	}
	return fmt.Sprintf("%s == %s", e.expr(a), e.expr(b))
}

// nilTest writes the comparison of a value with nil. A slice and a map hold the
// nil state in a field, because the zero value is an object and not null.
func (e *emitter) nilTest(x, other ir.Expr) (string, bool) {
	if _, ok := other.(*ir.Zero); !ok {
		return "", false
	}
	switch typeset.Core(x.GoType()).(type) {
	case *types.Slice, *types.Map:
		return e.expr(x) + ".isNil", true
	case *types.Pointer, *types.Signature, *types.Chan, *types.Interface:
		return e.expr(x) + " == null", true
	}
	return "", false
}

func (e *emitter) binary(x *ir.Binary) string {
	left, right := e.expr(x.X), e.expr(x.Y)
	t := x.X.GoType()
	switch x.Op {
	case token.LAND:
		return fmt.Sprintf("(%s && %s)", left, right)
	case token.LOR:
		return fmt.Sprintf("(%s || %s)", left, right)
	case token.EQL, token.NEQ:
		text := e.equal(x.X, x.Y)
		if x.Op == token.NEQ {
			return "(!(" + text + "))"
		}
		return "(" + text + ")"
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		return e.compare(x.Op, x.X, x.Y)
	case token.QUO:
		if isInteger(t) {
			if isUnsigned64(t) {
				return fmt.Sprintf("$udiv(%s, %s)", left, right)
			}
			return e.narrow(t, fmt.Sprintf("$div(%s, %s)", left, right))
		}
		return e.narrow(t, fmt.Sprintf("(%s / %s)", left, right))
	case token.REM:
		if isUnsigned64(t) {
			return fmt.Sprintf("$umod(%s, %s)", left, right)
		}
		return e.narrow(t, fmt.Sprintf("$mod(%s, %s)", left, right))
	case token.SHL:
		return e.narrow(t, fmt.Sprintf("$shl(%s, %s)", left, right))
	case token.SHR:
		if isUnsigned(t) {
			return e.narrow(t, fmt.Sprintf("$ushr(%s, %s)", left, right))
		}
		return e.narrow(t, fmt.Sprintf("$shr(%s, %s)", left, right))
	case token.AND_NOT:
		return e.narrow(t, fmt.Sprintf("(%s & ~%s)", left, right))
	case token.ADD:
		if isString(t) {
			return e.narrow(x.Type, fmt.Sprintf("$concat(%s, %s)", left, right))
		}
	}
	return e.narrow(t, fmt.Sprintf("(%s %s %s)", left, x.Op.String(), right))
}

// compare writes an ordered comparison. A string compares by byte, and a 64 bit
// unsigned value needs the unsigned helper.
func (e *emitter) compare(op token.Token, a, b ir.Expr) string {
	left, right := e.expr(a), e.expr(b)
	if tp, ok := types.Unalias(a.GoType()).(*types.TypeParam); ok && e.mixedForm(tp) {
		return fmt.Sprintf("(%s.order(%s, %s) %s 0)", e.dictOf(tp), left, right, op.String())
	}
	if isString(a.GoType()) {
		return fmt.Sprintf("($compare(%s, %s) %s 0)", left, right, op.String())
	}
	if isUnsigned64(a.GoType()) {
		switch op {
		case token.LSS:
			return fmt.Sprintf("$ult(%s, %s)", left, right)
		case token.LEQ:
			return fmt.Sprintf("$ule(%s, %s)", left, right)
		case token.GTR:
			return fmt.Sprintf("$ult(%s, %s)", right, left)
		case token.GEQ:
			return fmt.Sprintf("$ule(%s, %s)", right, left)
		}
	}
	return fmt.Sprintf("(%s %s %s)", left, op.String(), right)
}

// narrow keeps the value of a narrow type inside its range. A Dart int holds 64
// bits, thus a smaller Go type needs a mask after each operation.
//
// A type parameter needs a cast. Go gives the operation the type of the
// parameter, and Dart gives it the bound of the parameter.
func (e *emitter) narrow(t types.Type, text string) string {
	if m := mask(t); m != "" {
		return m + "(" + text + ")"
	}
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		return "(" + text + " as " + name.Local(tp.Obj().Name()) + ")"
	}
	return text
}

func (e *emitter) unary(x *ir.Unary) string {
	switch x.Op {
	case token.AND:
		return e.expr(x.X)
	case token.MUL:
		return e.deref(x.X)
	case token.SUB:
		return e.narrow(x.Type, "(-"+e.expr(x.X)+")")
	case token.NOT:
		return "(!" + e.expr(x.X) + ")"
	case token.XOR:
		return e.narrow(x.Type, "(~"+e.expr(x.X)+")")
	case token.ADD:
		return e.expr(x.X)
	}
	e.fail("the operator %s has no Dart form", x.Op)
	return "null"
}

func (e *emitter) composite(x *ir.Composite) string {
	u := types.Unalias(x.Type).Underlying()
	parts := make([]string, len(x.Elems))
	for i, el := range x.Elems {
		parts[i] = e.expr(el)
	}
	switch u := u.(type) {
	case *types.Struct:
		if named, ok := types.Unalias(x.Type).(*types.Named); ok {
			parts = append(e.dictArgs(named.TypeArgs()), parts...)
		}
		return e.dartType(x.Type) + "(" + strings.Join(parts, ", ") + ")"
	case *types.Array:
		for len(parts) < int(u.Len()) {
			parts = append(parts, e.zero(u.Elem()))
		}
		return fmt.Sprintf("<%s>[%s]", e.dartType(u.Elem()), strings.Join(parts, ", "))
	case *types.Slice:
		return e.sliceLiteral(u.Elem(), parts)
	case *types.Map:
		var text strings.Builder
		fmt.Fprintf(&text, "(GoMap<%s, %s>(() => %s)",
			e.dartType(u.Key()), e.dartType(u.Elem()), e.zero(u.Elem()))
		for i, key := range x.Keys {
			fmt.Fprintf(&text, "..set(%s, %s)", e.expr(key), parts[i])
		}
		text.WriteString(")")
		return text.String()
	}
	e.fail("a literal of type %s has no Dart form", x.Type)
	return "null"
}

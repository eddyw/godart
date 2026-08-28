package emit

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
	"github.com/eddyw/godart/internal/overlay"
	"github.com/eddyw/godart/internal/typeset"
)

func (e *emitter) call(x *ir.Call) string {
	text := e.callText(x)
	if !x.Async {
		return text
	}
	// Dart rejects a void expression inside parentheses, thus a call with no
	// result keeps the plain form.
	if sig := signatureOf(x.Fun.GoType()); sig != nil && sig.Results().Len() == 0 {
		return "await " + text
	}
	return "(await " + text + ")"
}

func (e *emitter) callText(x *ir.Call) string {
	args := e.callArgs(x)
	if text, ok := e.intrinsic(x, args); ok {
		return text
	}
	targs, dicts := e.typeArguments(x.TypeArgs)
	args = append(dicts, args...)
	switch fun := x.Fun.(type) {
	case *ir.Method:
		return e.methodCall(fun, args, targs)
	case *ir.Ident:
		if _, ok := fun.Obj.(*types.Func); ok {
			return e.identExpr(fun) + targs + "(" + strings.Join(args, ", ") + ")"
		}
	}
	return e.expr(x.Fun) + "!(" + strings.Join(args, ", ") + ")"
}

// intrinsic replaces a call to the interop package with a call to the runtime
// library.
func (e *emitter) intrinsic(x *ir.Call, args []string) (string, bool) {
	id, ok := x.Fun.(*ir.Ident)
	if !ok {
		return "", false
	}
	fn, ok := id.Obj.(*types.Func)
	if !ok || fn.Pkg() == nil || !overlay.Intrinsic(fn.Pkg().Path()) {
		return "", false
	}
	switch fn.Name() {
	case "Print":
		return fmt.Sprintf("$stdout(%s)", args[0]), true
	case "Format":
		return fmt.Sprintf("$goFormat(%s, %s)", args[1], args[0]), true
	case "Kind":
		return fmt.Sprintf("$goKind(%s)", args[0]), true
	case "Upper":
		return fmt.Sprintf("$upper(%s)", args[0]), true
	case "Lower":
		return fmt.Sprintf("$lower(%s)", args[0]), true
	case "ParseFloat":
		return fmt.Sprintf("$parseFloat(%s)", args[0]), true
	case "Sleep":
		return fmt.Sprintf("$sleep(%s)", args[0]), true
	case "NowNanos":
		return "$nowNanos()", true
	case "MonoNanos":
		return "$monoNanos()", true
	case "Float64Bits":
		return fmt.Sprintf("$f64ToBits(%s)", args[0]), true
	case "Float64FromBits":
		return fmt.Sprintf("$f64FromBits(%s)", args[0]), true
	case "Float32Bits":
		return fmt.Sprintf("$f32ToBits(%s)", args[0]), true
	case "Float32FromBits":
		return fmt.Sprintf("$f32FromBits(%s)", args[0]), true
	case "FileOpen":
		return fmt.Sprintf("$fileOpen(%s, %s)", args[0], args[1]), true
	case "FileRead":
		return fmt.Sprintf("$fileRead(%s, %s)", args[0], args[1]), true
	case "FileWrite":
		return fmt.Sprintf("$fileWrite(%s, %s)", args[0], args[1]), true
	case "FileSeek":
		return fmt.Sprintf("$fileSeek(%s, %s, %s)", args[0], args[1], args[2]), true
	case "FileClose":
		return fmt.Sprintf("$fileClose(%s)", args[0]), true
	case "FileSync":
		return fmt.Sprintf("$fileSync(%s)", args[0]), true
	case "ReadFile":
		return fmt.Sprintf("$readFile(%s)", args[0]), true
	case "WriteFile":
		return fmt.Sprintf("$writeFile(%s, %s)", args[0], args[1]), true
	case "Stat":
		return fmt.Sprintf("$stat(%s)", args[0]), true
	case "Remove":
		return fmt.Sprintf("$remove(%s)", args[0]), true
	case "RemoveAll":
		return fmt.Sprintf("$removeAll(%s)", args[0]), true
	case "Mkdir":
		return fmt.Sprintf("$mkdir(%s, %s)", args[0], args[1]), true
	case "Rename":
		return fmt.Sprintf("$rename(%s, %s)", args[0], args[1]), true
	case "ReadDir":
		return fmt.Sprintf("$readDir(%s)", args[0]), true
	case "Getenv":
		return fmt.Sprintf("$getenv(%s)", args[0]), true
	case "Environ":
		return "$environ()", true
	case "Getwd":
		return "$getwd()", true
	case "Chdir":
		return fmt.Sprintf("$chdir(%s)", args[0]), true
	case "TempDir":
		return "$tempDir()", true
	case "PathSeparator":
		return "$pathSeparator()", true
	case "Stderr":
		return fmt.Sprintf("$stderr(%s)", args[0]), true
	case "StdinRead":
		return fmt.Sprintf("$stdinRead(%s)", args[0]), true
	case "Worker":
		return fmt.Sprintf("$worker(%s, %s)", args[0], args[1]), true
	case "Stop":
		return "$stop()", true
	case "Catch":
		return fmt.Sprintf("$catch(%s)", args[0]), true
	case "Exit":
		return fmt.Sprintf("$exit(%s)", args[0]), true
	case "Args":
		return "$args()", true
	case "SliceLen":
		return fmt.Sprintf("$sliceLen(%s)", args[0]), true
	case "SliceSwap":
		return fmt.Sprintf("$swap(%s, %s, %s)", args[0], args[1], args[2]), true
	}
	e.fail("the interop function %s has no runtime form", fn.Name())
	return "null", true
}

// typeArguments writes the type arguments of a call and the dictionary of
// each one.
func (e *emitter) typeArguments(args []types.Type) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	names := make([]string, len(args))
	dicts := make([]string, len(args))
	for i, t := range args {
		names[i] = e.dartType(t)
		dicts[i] = e.dictOf(t)
	}
	return "<" + strings.Join(names, ", ") + ">", dicts
}

// callArgs writes the arguments. A variadic call packs the last arguments into
// a slice, exactly as Go does.
func (e *emitter) callArgs(x *ir.Call) []string {
	sig := e.calleeSignature(x)
	if sig == nil || !x.Variadic || x.Spread {
		out := make([]string, len(x.Args))
		for i, a := range x.Args {
			out[i] = e.expr(a)
		}
		return out
	}
	last := sig.Params().Len() - 1
	out := make([]string, 0, last+1)
	for i := 0; i < last && i < len(x.Args); i++ {
		out = append(out, e.expr(x.Args[i]))
	}
	slice, _ := types.Unalias(sig.Params().At(last).Type()).Underlying().(*types.Slice)
	if slice == nil {
		e.fail("the last parameter of a variadic function is not a slice")
		return out
	}
	var rest []string
	for i := last; i < len(x.Args); i++ {
		rest = append(rest, e.expr(x.Args[i]))
	}
	return append(out, e.sliceLiteral(slice.Elem(), rest))
}

func (e *emitter) sliceLiteral(elem types.Type, parts []string) string {
	if list := typedList(elem); list != "" {
		return fmt.Sprintf("Slice.of(%s.fromList(<%s>[%s]))",
			list, e.dartType(elem), strings.Join(parts, ", "))
	}
	return fmt.Sprintf("Slice.of(<%s>[%s])", e.dartType(elem), strings.Join(parts, ", "))
}

// calleeSignature gives the signature of the callee. A call of a generic
// function reads the identifier of the declaration, thus the signature still
// holds the type parameters. The type arguments of the call replace them.
func (e *emitter) calleeSignature(x *ir.Call) *types.Signature {
	sig := signatureOf(x.Fun.GoType())
	if sig == nil || len(x.TypeArgs) == 0 ||
		sig.TypeParams().Len() != len(x.TypeArgs) {
		return sig
	}
	out, err := types.Instantiate(nil, sig, x.TypeArgs, false)
	if err != nil {
		e.fail("the call of %s takes no type argument: %v", x.Fun.GoType(), err)
		return sig
	}
	return signatureOf(out)
}

func signatureOf(t types.Type) *types.Signature {
	if t == nil {
		return nil
	}
	sig, _ := types.Unalias(t).Underlying().(*types.Signature)
	return sig
}

func (e *emitter) methodCall(m *ir.Method, args []string, targs string) string {
	if isInterfaceType(m.Recv.GoType()) {
		return e.interfaceCall(m, args)
	}
	obj := receiverTypeName(m.Recv.GoType())
	list := strings.Join(args, ", ")
	if obj != nil && !isStructType(obj.Type()) {
		all := append([]string{e.expr(m.Recv)}, args...)
		return e.qualify(obj, methodFuncName(obj.Name(), m.Obj.Name())) +
			"(" + strings.Join(all, ", ") + ")"
	}
	return fmt.Sprintf("%s.%s%s(%s)", e.deref(m.Recv), methodName(m.Obj), targs, list)
}

// methodValue reads a method without a call. Go binds a copy of the receiver at
// that moment when the method takes a value receiver.
func (e *emitter) methodValue(m *ir.Method) string {
	obj := receiverTypeName(m.Recv.GoType())
	if obj != nil && !isStructType(obj.Type()) {
		e.fail("a method value of the type %s is not supported yet", obj.Name())
		return "null"
	}
	sig, _ := m.Obj.Type().(*types.Signature)
	if sig != nil && sig.Recv() != nil && !isPointerType(sig.Recv().Type()) {
		return fmt.Sprintf("%s.$copy().%s", e.deref(m.Recv), methodName(m.Obj))
	}
	return fmt.Sprintf("%s.%s", e.deref(m.Recv), methodName(m.Obj))
}

func (e *emitter) builtin(x *ir.Builtin) string {
	switch x.Name {
	case "len":
		return e.length(x.Args[0])
	case "cap":
		return e.capacity(x.Args[0])
	case "append":
		return e.appendCall(x)
	case "copy":
		return e.copyCall(x)
	case "make":
		return e.makeCall(x)
	case "new":
		return e.newCall(x)
	case "panic":
		return fmt.Sprintf("$panic(%s)", e.expr(x.Args[0]))
	case "print":
		return fmt.Sprintf("$print(<String>[%s])", strings.Join(e.textList(x.Args), ", "))
	case "println":
		return fmt.Sprintf("$println(<String>[%s])", strings.Join(e.textList(x.Args), ", "))
	case "decoderune":
		return fmt.Sprintf("$decodeRune(%s, %s)", e.expr(x.Args[0]), e.expr(x.Args[1]))
	case "min", "max":
		return e.minMax(x)
	case "delete":
		return fmt.Sprintf("%s.remove(%s)", e.expr(x.Args[0]), e.expr(x.Args[1]))
	case "clear":
		if isMapType(x.Args[0].GoType()) {
			return e.expr(x.Args[0]) + ".clear()"
		}
		e.fail("clear does not apply to a slice yet")
		return "null"
	case "mapkeys":
		return e.expr(x.Args[0]) + ".keys()"
	case "close":
		return e.expr(x.Args[0]) + ".close()"
	case "maphas":
		return fmt.Sprintf("%s.has(%s)", e.expr(x.Args[0]), e.expr(x.Args[1]))
	}
	e.fail("the builtin %s has no Dart form yet", x.Name)
	return "null"
}

func (e *emitter) length(x ir.Expr) string {
	t := x.GoType()
	switch {
	case isString(t):
		return e.expr(x) + ".length"
	case isSliceType(t):
		return e.expr(x) + ".len"
	case isArrayType(t):
		return e.expr(x) + ".length"
	case isMapType(t):
		return e.expr(x) + ".length"
	case isChanType(t):
		return e.expr(x) + ".length"
	case isPointerType(t):
		return e.deref(x) + ".length"
	}
	e.fail("len does not apply to the type %s", t)
	return "0"
}

func (e *emitter) capacity(x ir.Expr) string {
	t := x.GoType()
	switch {
	case isSliceType(t):
		return e.expr(x) + ".cap"
	case isArrayType(t):
		return e.expr(x) + ".length"
	case isChanType(t):
		return e.expr(x) + ".capacity"
	case isPointerType(t):
		return e.deref(x) + ".length"
	}
	e.fail("cap does not apply to the type %s", t)
	return "0"
}

// isUntypedNil reports whether the type is the type of the literal nil.
func isUntypedNil(t types.Type) bool {
	b, ok := types.Unalias(t).(*types.Basic)
	return ok && b.Kind() == types.UntypedNil
}

func (e *emitter) appendCall(x *ir.Builtin) string {
	slice, _ := typeset.Core(x.Type).(*types.Slice)
	if slice == nil {
		e.fail("append needs a slice")
		return "null"
	}
	if x.Spread {
		source := e.expr(x.Args[1])
		if isString(x.Args[1].GoType()) {
			// Go allows append of a string to a byte slice.
			source = fmt.Sprintf("$stringToBytes(%s)", source)
		}
		return fmt.Sprintf("$appendSlice(%s, %s, %s)",
			e.expr(x.Args[0]), source, e.alloc(slice.Elem()))
	}
	parts := make([]string, 0, len(x.Args)-1)
	for _, a := range x.Args[1:] {
		parts = append(parts, e.expr(a))
	}
	return fmt.Sprintf("$append(%s, <%s>[%s], %s)",
		e.expr(x.Args[0]), e.dartType(slice.Elem()), strings.Join(parts, ", "), e.alloc(slice.Elem()))
}

func (e *emitter) copyCall(x *ir.Builtin) string {
	dst, src := x.Args[0], x.Args[1]
	source := e.expr(src)
	if isString(src.GoType()) {
		source = fmt.Sprintf("$stringToBytes(%s)", source)
	}
	slice, _ := typeset.Core(dst.GoType()).(*types.Slice)
	if slice != nil && needsElementCopy(slice.Elem()) {
		return fmt.Sprintf("$copySlice(%s, %s, (a, b) => %s)",
			e.expr(dst), source, e.assignElem("a", "b", slice.Elem()))
	}
	return fmt.Sprintf("$copySlice(%s, %s)", e.expr(dst), source)
}

func (e *emitter) makeCall(x *ir.Builtin) string {
	switch u := typeset.Core(x.Type).(type) {
	case *types.Slice:
		length := "0"
		if len(x.Args) > 0 {
			length = e.expr(x.Args[0])
		}
		capacity := length
		if len(x.Args) > 1 {
			capacity = e.expr(x.Args[1])
		}
		return fmt.Sprintf("$makeSlice<%s>(%s, %s, %s)",
			e.dartType(u.Elem()), length, capacity, e.alloc(u.Elem()))
	case *types.Map:
		return fmt.Sprintf("GoMap<%s, %s>(() => %s)",
			e.dartType(u.Key()), e.dartType(u.Elem()), e.zero(u.Elem()))
	case *types.Chan:
		capacity := "0"
		if len(x.Args) > 0 {
			capacity = e.expr(x.Args[0])
		}
		return fmt.Sprintf("Chan<%s>(%s, () => %s)",
			e.dartType(u.Elem()), capacity, e.zero(u.Elem()))
	}
	e.fail("make does not apply to the type %s yet", x.Type)
	return "null"
}

func (e *emitter) newCall(x *ir.Builtin) string {
	p, _ := types.Unalias(x.Type).Underlying().(*types.Pointer)
	if p == nil {
		e.fail("new gives a pointer")
		return "null"
	}
	if len(x.Args) == 1 {
		return e.expr(x.Args[0]) + ".$copy()"
	}
	return e.zero(p.Elem())
}

func (e *emitter) minMax(x *ir.Builtin) string {
	op := "<"
	if x.Name == "max" {
		op = ">"
	}
	out := e.expr(x.Args[0])
	for _, a := range x.Args[1:] {
		next := e.expr(a)
		out = fmt.Sprintf("((%s %s %s) ? %s : %s)", out, op, next, out, next)
	}
	return out
}

// textList writes each value of a print builtin as Dart text.
func (e *emitter) textList(args []ir.Expr) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = e.textOf(a)
	}
	return out
}

func (e *emitter) textOf(x ir.Expr) string {
	t := x.GoType()
	switch {
	case isString(t):
		return e.expr(x) + ".toDartString()"
	case isFloat(t):
		return fmt.Sprintf("$fstring(%s)", e.expr(x))
	case isUnsigned64(t):
		return fmt.Sprintf("$ustring(%s)", e.expr(x))
	case isInteger(t):
		return e.expr(x) + ".toString()"
	}
	if b, ok := types.Unalias(t).Underlying().(*types.Basic); ok && b.Kind() == types.Bool {
		return e.expr(x) + ".toString()"
	}
	e.fail("print does not accept the type %s yet", t)
	return "''"
}

// conv makes a Go conversion explicit in Dart.
func (e *emitter) conv(x *ir.Conv) string {
	from, to := x.From, x.Type
	text := e.expr(x.X)
	switch {
	case isString(to) && isSliceType(from):
		if isRuneSlice(from) {
			return fmt.Sprintf("$runesToString(%s)", text)
		}
		return fmt.Sprintf("$bytesToString(%s)", text)
	case isString(to) && isInteger(from):
		return fmt.Sprintf("$runeToString(%s)", text)
	case isSliceType(to) && isString(from):
		if isRuneSlice(to) {
			return fmt.Sprintf("$stringToRunes(%s)", text)
		}
		return fmt.Sprintf("$stringToBytes(%s)", text)
	case isFloat(to) && isInteger(from):
		if isUnsigned64(from) {
			return e.narrow(to, fmt.Sprintf("$utodouble(%s)", text))
		}
		return e.narrow(to, fmt.Sprintf("%s.toDouble()", text))
	case isInteger(to) && isFloat(from):
		return e.narrow(to, fmt.Sprintf("%s.toInt()", text))
	case isFloat(to) && isFloat(from):
		return e.narrow(to, text)
	case isInteger(to) && isInteger(from):
		return e.narrow(to, text)
	case isString(to) && isString(from):
		return text
	}
	if isInterfaceType(to) {
		if isInterfaceType(from) {
			return text
		}
		return e.box(x.X, from)
	}
	if types.Identical(types.Unalias(from).Underlying(), types.Unalias(to).Underlying()) {
		if isStructType(to) {
			e.fail("a conversion between 2 struct types is not supported yet")
			return "null"
		}
		return text
	}
	// A conversion of nil gives the zero value of the target type. Go writes
	// `S(nil)` for the nil slice of a type parameter.
	if isUntypedNil(from) {
		return e.zero(to)
	}
	e.fail("the conversion from %s to %s is not supported yet", from, to)
	return "null"
}

func isRuneSlice(t types.Type) bool {
	s, ok := types.Unalias(t).Underlying().(*types.Slice)
	if !ok {
		return false
	}
	b, ok := types.Unalias(s.Elem()).Underlying().(*types.Basic)
	return ok && (b.Kind() == types.Int32 || b.Kind() == types.Rune)
}

func methodName(fn *types.Func) string {
	return name.Package(fn.Name())
}

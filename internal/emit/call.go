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
	return e.callText(x)
}

func (e *emitter) callText(x *ir.Call) string {
	if x.Results != nil {
		return e.spreadCall(x)
	}
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

// spreadCall writes a call whose arguments come from the results of another
// call. Go passes each result of such a call as its own argument.
//
// A Dart record holds the results. The closure runs the inner call 1 time, and
// each argument reads 1 field of the record.
func (e *emitter) spreadCall(x *ir.Call) string {
	inner := x.Results
	x.Results = nil
	body := e.callText(x)
	x.Results = inner
	return fmt.Sprintf("((%s %s) => %s)(%s)",
		e.dartType(inner.GoType()), x.ResultsVar, body, e.expr(inner))
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
	case "DialTLS":
		return fmt.Sprintf("$dialTLS(%s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4]), true
	case "GC":
		return "$gc()", true
	case "AddCleanup":
		return fmt.Sprintf("$addCleanup(%s!.value as Object, %s)", args[0], args[1]), true
	case "StopCleanup":
		return fmt.Sprintf("$stopCleanup(%s)", args[0]), true
	case "Kind":
		return fmt.Sprintf("$goKind(%s)", args[0]), true
	case "Stack":
		return fmt.Sprintf("$stack(%s)", args[0]), true
	case "TypeInfo":
		return fmt.Sprintf("$typeInfo(%s)", args[0]), true
	case "DeepEqual":
		return fmt.Sprintf("$deepEqual(%s, %s)", args[0], args[1]), true
	case "AssignError":
		return fmt.Sprintf("$assignError(%s, %s)", args[0], args[1]), true
	case "DescOf":
		return fmt.Sprintf("$descOf(%s)", args[0]), true
	case "DescInfo":
		return fmt.Sprintf("$descInfo(%s)", args[0]), true
	case "DescNumIn":
		return fmt.Sprintf("$descNumIn(%s)", args[0]), true
	case "DescNumOut":
		return fmt.Sprintf("$descNumOut(%s)", args[0]), true
	case "DescIn":
		return fmt.Sprintf("$descIn(%s, %s)", args[0], args[1]), true
	case "DescOut":
		return fmt.Sprintf("$descOut(%s, %s)", args[0], args[1]), true
	case "DescKey":
		return fmt.Sprintf("$descKey(%s)", args[0]), true
	case "CallValue":
		return fmt.Sprintf("$callValue(%s, %s)", args[0], args[1]), true
	case "MakeMap":
		return fmt.Sprintf("$makeMap(%s)", args[0]), true
	case "MapSet":
		return fmt.Sprintf("$mapSet(%s, %s, %s, %s)",
			args[0], args[1], args[2], args[3]), true
	case "DescLength":
		return fmt.Sprintf("$descLength(%s)", args[0]), true
	case "FieldCount":
		return fmt.Sprintf("$fieldCount(%s)", args[0]), true
	case "FieldInfo":
		return fmt.Sprintf("$fieldInfo(%s, %s)", args[0], args[1]), true
	case "FieldRead":
		return fmt.Sprintf("$fieldRead(%s, %s)", args[0], args[1]), true
	case "FieldWrite":
		return fmt.Sprintf("$fieldWrite(%s, %s, %s)", args[0], args[1], args[2]), true
	case "ValueLen":
		return fmt.Sprintf("$valueLen(%s)", args[0]), true
	case "ValueIndex":
		return fmt.Sprintf("$valueIndex(%s, %s)", args[0], args[1]), true
	case "ValueSetIndex":
		return fmt.Sprintf("$valueSetIndex(%s, %s, %s)", args[0], args[1], args[2]), true
	case "AsInt":
		return fmt.Sprintf("$asInt(%s)", args[0]), true
	case "AsFloat":
		return fmt.Sprintf("$asFloat(%s)", args[0]), true
	case "AsBool":
		return fmt.Sprintf("$asBool(%s)", args[0]), true
	case "AsString":
		return fmt.Sprintf("$asString(%s)", args[0]), true
	case "WithInt":
		return fmt.Sprintf("$withInt(%s, %s)", args[0], args[1]), true
	case "WithFloat":
		return fmt.Sprintf("$withFloat(%s, %s)", args[0], args[1]), true
	case "WithBool":
		return fmt.Sprintf("$withBool(%s, %s)", args[0], args[1]), true
	case "WithString":
		return fmt.Sprintf("$withString(%s, %s)", args[0], args[1]), true
	case "NewValue":
		return fmt.Sprintf("$newValue(%s)", args[0]), true
	case "MakeValue":
		return fmt.Sprintf("$makeValue(%s, %s)", args[0], args[1]), true
	case "DescElem":
		return fmt.Sprintf("$descElem(%s)", args[0]), true
	case "DescZero":
		return fmt.Sprintf("$descZero(%s)", args[0]), true
	case "PointerElem":
		return fmt.Sprintf("$pointerElem(%s)", args[0]), true
	case "PointerSet":
		return fmt.Sprintf("$pointerSet(%s, %s)", args[0], args[1]), true
	case "ZeroText":
		return fmt.Sprintf("$zeroText(%s)", args[0]), true
	case "OS":
		return "$goos()", true
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
	case "ZoneOffset":
		return fmt.Sprintf("$zoneOffset(%s)", args[0]), true
	case "MonoNanos":
		return "$monoNanos()", true
	case "MarshalJSON":
		return fmt.Sprintf("$jsonMarshal(%s, %s)", args[0], args[1]), true
	case "UnmarshalJSON":
		return fmt.Sprintf("$jsonUnmarshal(%s, %s)", args[0], args[1]), true
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
	case "DialTCP":
		return fmt.Sprintf("$dialTCP(%s, %s)", args[0], args[1]), true
	case "ConnRead":
		return fmt.Sprintf("$connRead(%s, %s)", args[0], args[1]), true
	case "ConnWrite":
		return fmt.Sprintf("$connWrite(%s, %s)", args[0], args[1]), true
	case "ConnClose":
		return fmt.Sprintf("$connClose(%s)", args[0]), true
	case "ListenTCP":
		return fmt.Sprintf("$listenTCP(%s, %s)", args[0], args[1]), true
	case "Accept":
		return fmt.Sprintf("$accept(%s)", args[0]), true
	case "ListenClose":
		return fmt.Sprintf("$listenClose(%s)", args[0]), true
	case "DialUnix":
		return fmt.Sprintf("$dialUnix(%s)", args[0]), true
	case "ListenUnix":
		return fmt.Sprintf("$listenUnix(%s)", args[0]), true
	case "ConnDeadline":
		return fmt.Sprintf("$connDeadline(%s, %s, %s)", args[0], args[1], args[2]), true
	case "ListenUDP":
		return fmt.Sprintf("$listenUDP(%s, %s)", args[0], args[1]), true
	case "DialUDP":
		return fmt.Sprintf("$dialUDP(%s, %s)", args[0], args[1]), true
	case "PacketRead":
		return fmt.Sprintf("$packetRead(%s, %s)", args[0], args[1]), true
	case "PacketWrite":
		return fmt.Sprintf("$packetWrite(%s, %s, %s, %s)",
			args[0], args[1], args[2], args[3]), true
	case "PacketClose":
		return fmt.Sprintf("$packetClose(%s)", args[0]), true
	case "PacketDeadline":
		return fmt.Sprintf("$packetDeadline(%s, %s, %s)", args[0], args[1], args[2]), true
	case "StartTimer":
		return fmt.Sprintf("$startTimer(%s, %s)", args[0], args[1]), true
	case "StopTimer":
		return fmt.Sprintf("$stopTimer(%s)", args[0]), true
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
	// A method of a named type that is not a struct, and a method with a
	// pointer receiver, each hold the body in a top level function. Go runs a
	// method with a nil pointer receiver, and Dart calls no member of null.
	if obj != nil && (!isStructType(obj.Type()) || pointerMethod(m.Obj)) {
		dicts, typeArgs := e.recvTypeArgs(m.Recv.GoType())
		all := append(append([]string{e.topLevelRecv(m)}, dicts...), args...)
		return e.qualify(obj, methodFuncName(obj.Name(), m.Obj.Name())) +
			joinTypeArgs(typeArgs, targs) +
			"(" + strings.Join(all, ", ") + ")"
	}
	return fmt.Sprintf("%s.%s%s(%s)", e.deref(m.Recv), methodName(m.Obj), targs, list)
}

// topLevelRecv writes the receiver of a call to a method whose body stands in a
// top level function.
//
// Go reads the value behind a pointer for a method with a value receiver, and it
// takes the address of a value for a method with a pointer receiver.
func (e *emitter) topLevelRecv(m *ir.Method) string {
	t := m.Recv.GoType()
	if pointerMethod(m.Obj) && !isPointerType(t) && needsBox(t) {
		return e.addressExpr(m.Recv)
	}
	if !pointerMethod(m.Obj) && isPointerType(t) {
		return e.deref(m.Recv)
	}
	return e.expr(m.Recv)
}

// methodValue reads a method without a call. Go binds a copy of the receiver at
// that moment when the method takes a value receiver.
func (e *emitter) methodValue(m *ir.Method) string {
	obj := receiverTypeName(m.Recv.GoType())
	if obj != nil && (!isStructType(obj.Type()) || pointerMethod(m.Obj)) {
		return e.methodFuncValue(m, obj)
	}
	sig, _ := m.Obj.Type().(*types.Signature)
	if sig != nil && sig.Recv() != nil && !isPointerType(sig.Recv().Type()) {
		return fmt.Sprintf("%s.$copy().%s", e.deref(m.Recv), methodName(m.Obj))
	}
	return fmt.Sprintf("%s.%s", e.deref(m.Recv), methodName(m.Obj))
}

// methodExpr writes a method expression, for example `(*T).M`. The value is a
// closure whose first parameter is the receiver.
// funcInst writes an instance of a generic function that reaches a value. Dart
// binds no argument of a function, thus the value is a closure that gives the
// dictionary of each type argument.
func (e *emitter) funcInst(x *ir.FuncInst) string {
	sig, ok := x.Type.(*types.Signature)
	if !ok {
		e.fail("the instance of %s has no signature", x.Fun.Name)
		return "null"
	}
	targs, dicts := e.typeArguments(x.TypeArgs)
	params := make([]string, sig.Params().Len())
	names := make([]string, sig.Params().Len())
	for i := range params {
		names[i] = fmt.Sprintf("$a%d", i)
		params[i] = e.dartType(sig.Params().At(i).Type()) + " " + names[i]
	}
	all := append(append([]string{}, dicts...), names...)
	call := e.identExpr(x.Fun) + targs + "(" + strings.Join(all, ", ") + ")"
	return fmt.Sprintf("((%s) => %s)", strings.Join(params, ", "), call)
}

func (e *emitter) methodExpr(x *ir.MethodExpr) string {
	sig, ok := x.Type.(*types.Signature)
	if !ok || sig.Params().Len() == 0 {
		e.fail("the method expression %s has no signature", x.Obj.Name())
		return "null"
	}
	params := make([]string, sig.Params().Len())
	names := make([]string, sig.Params().Len())
	for i := range params {
		names[i] = fmt.Sprintf("$a%d", i)
		params[i] = e.dartType(sig.Params().At(i).Type()) + " " + names[i]
	}
	obj := receiverTypeName(x.Recv)
	if obj != nil && (!isStructType(obj.Type()) || pointerMethod(x.Obj)) {
		dicts, typeArgs := e.recvTypeArgs(x.Recv)
		all := append(append([]string{names[0]}, dicts...), names[1:]...)
		call := e.qualify(obj, methodFuncName(obj.Name(), x.Obj.Name())) +
			joinTypeArgs(typeArgs, "") + "(" + strings.Join(all, ", ") + ")"
		return fmt.Sprintf("((%s) => %s)", strings.Join(params, ", "), call)
	}
	recv := names[0]
	if isPointerType(x.Recv) {
		recv += "!"
	} else {
		recv += ".$copy()"
	}
	call := fmt.Sprintf("%s.%s(%s)", recv, methodName(x.Obj), strings.Join(names[1:], ", "))
	return fmt.Sprintf("((%s) => %s)", strings.Join(params, ", "), call)
}

// methodFuncValue reads a method of a named type that is not a struct. Such a
// method is a top level function, thus the value becomes a closure that gives
// the receiver as the first argument.
//
// Go binds the receiver at the point of the value. The outer closure runs 1
// time, thus a later write to the source of the receiver does not reach it.
// joinTypeArgs writes the type arguments of a call to a top level function of a
// method. The list holds the arguments of the receiver type first, and the text
// holds the arguments of the method itself.
func joinTypeArgs(recv []string, method string) string {
	if len(recv) == 0 {
		return method
	}
	out := strings.Join(recv, ", ")
	if method != "" {
		out += ", " + strings.TrimSuffix(strings.TrimPrefix(method, "<"), ">")
	}
	return "<" + out + ">"
}

func (e *emitter) methodFuncValue(m *ir.Method, obj *types.TypeName) string {
	sig, _ := m.Obj.Type().(*types.Signature)
	if sig == nil {
		e.fail("the method %s has no signature", m.Obj.Name())
		return "null"
	}
	params := make([]string, sig.Params().Len())
	args := make([]string, sig.Params().Len()+1)
	args[0] = "$recv"
	for i := range params {
		name := fmt.Sprintf("$a%d", i)
		params[i] = e.dartType(sig.Params().At(i).Type()) + " " + name
		args[i+1] = name
	}
	dicts, typeArgs := e.recvTypeArgs(m.Recv.GoType())
	args = append(append([]string{args[0]}, dicts...), args[1:]...)
	call := e.qualify(obj, methodFuncName(obj.Name(), m.Obj.Name())) +
		joinTypeArgs(typeArgs, "") + "(" + strings.Join(args, ", ") + ")"
	return fmt.Sprintf("((%s $recv) => (%s) => %s)(%s)",
		e.dartType(m.Recv.GoType()), strings.Join(params, ", "), call, e.expr(m.Recv))
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
		if slice, ok := typeset.Core(x.Args[0].GoType()).(*types.Slice); ok {
			return fmt.Sprintf("$clearSlice(%s, () => %s)",
				e.expr(x.Args[0]), e.zero(slice.Elem()))
		}
		e.fail("clear does not apply to the type %s", x.Args[0].GoType())
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
	if needsBox(p.Elem()) {
		if len(x.Args) == 1 {
			return fmt.Sprintf("Box<%s>(%s)", e.dartType(p.Elem()), e.expr(x.Args[0]))
		}
		return fmt.Sprintf("Box<%s>(%s)", e.dartType(p.Elem()), e.zero(p.Elem()))
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
	case isArrayType(to) && isSliceType(from):
		// Go 1.20 converts a slice to an array. It panics when the slice holds
		// fewer elements than the array.
		a, _ := typeset.Core(to).(*types.Array)
		return fmt.Sprintf("$sliceToArray<%s>(%s, %d)",
			e.dartType(a.Elem()), text, a.Len())
	case isPointerType(to) && isSliceType(from):
		// Go 1.17 converts a slice to a pointer to an array. The array shares
		// the elements of the slice, thus a write through the pointer reaches
		// the slice.
		p, _ := typeset.Core(to).(*types.Pointer)
		a, _ := typeset.Core(p.Elem()).(*types.Array)
		if a != nil {
			return fmt.Sprintf("$sliceToArrayPtr<%s>(%s, %d)",
				e.dartType(a.Elem()), text, a.Len())
		}
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
		// The untyped nil gives a nil interface. A typed nil keeps its
		// description, exactly as in Go.
		if isInterfaceType(from) || isUntypedNil(from) {
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
	// A pointer converts to another pointer when the 2 base types share 1
	// underlying type. The Dart form of such a pointer is a box or a list, and
	// the 2 forms are equal, thus the conversion performs no operation.
	//
	// A pointer to a struct holds the class of the struct, and 2 named struct
	// types give 2 classes. Such a conversion stays outside the subset.
	if pf, ok := types.Unalias(from).Underlying().(*types.Pointer); ok {
		if pt, ok := types.Unalias(to).Underlying().(*types.Pointer); ok {
			if !isStructType(pt.Elem()) &&
				types.Identical(types.Unalias(pf.Elem()).Underlying(),
					types.Unalias(pt.Elem()).Underlying()) {
				return text
			}
		}
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

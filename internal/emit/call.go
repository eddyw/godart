package emit

import (
	"fmt"
	"go/constant"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
	"github.com/eddyw/godart/internal/name"
	"github.com/eddyw/godart/internal/overlay"
	"github.com/eddyw/godart/internal/typeset"
)

func (e *emitter) call(x *ir.Call) string {
	text := e.callText(x)
	if x.Sync {
		text = "$syncCall(() => " + text + ")"
	}
	return e.sourceCall(x.Pos, text, x.GoType())
}

func (e *emitter) callText(x *ir.Call) string {
	if x.Results != nil {
		return e.spreadCall(x)
	}
	args := e.callArgs(x)
	e.mark(x.Pos)
	if text, ok := e.intrinsic(x, args); ok {
		return text
	}
	targs, dicts := e.typeArguments(x.TypeArgs)
	args = append(dicts, args...)
	switch fun := x.Fun.(type) {
	case *ir.Method:
		return e.eraseCast(fun.Obj, x, e.methodCall(fun, args, targs))
	case *ir.Ident:
		if fn, ok := fun.Obj.(*types.Func); ok {
			return e.eraseCast(fn, x,
				e.identExpr(fun)+targs+"("+strings.Join(args, ", ")+")")
		}
	}
	// Go panics with a run time error when a call goes through a nil
	// function value, and Dart gives a null check error. An operand with an
	// effect takes a parameter of its own, thus the operand runs 1 time.
	recv := e.expr(x.Fun)
	joined := strings.Join(args, ", ")
	if _, ok := x.Fun.(*ir.FuncLit); ok {
		return "(" + recv + ")(" + joined + ")"
	}
	// A function value whose type erases to Object?, for example a type that
	// names itself, holds no Dart function type. Such a call goes through a
	// dynamic call.
	if e.dartType(x.Fun.GoType()) == "Object?" {
		if exprIsSimple(x.Fun) {
			return "(" + recv + " == null ? $panicNil() : (" + recv +
				"! as dynamic)(" + joined + "))"
		}
		return fmt.Sprintf(
			"((Object? $fn) => ($fn == null ? $panicNil() : ($fn! as dynamic)(%s)))(%s)",
			joined, recv)
	}
	if exprIsSimple(x.Fun) {
		return "(" + recv + " == null ? $panicNil() : " + recv + "!(" + joined + "))"
	}
	// The parameter takes Object?, because the type text of a chain of calls
	// grows with each level, and the analyzer rejects the deep text.
	return fmt.Sprintf(
		"((Object? $fn) => ($fn == null ? $panicNil() : ($fn! as dynamic)(%s)))(%s)",
		joined, recv)
}

// eraseCast adds a cast when the declaration of a generic function erases the
// Dart type of its result. A pointer to a type parameter takes the Dart type
// Object?, because the form of a pointer follows the type argument. The call
// site holds that argument, thus the cast gives the real type.
func (e *emitter) eraseCast(obj types.Object, x *ir.Call, text string) string {
	fn, ok := obj.(*types.Func)
	if !ok {
		return text
	}
	origin, ok := fn.Origin().Type().(*types.Signature)
	if !ok {
		return text
	}
	return e.eraseResults(origin.Results(), x.GoType(), text)
}

func (e *emitter) eraseResults(origin *types.Tuple, result types.Type, text string) string {
	if tuple, ok := result.(*types.Tuple); ok {
		if tuple.Len() == 0 {
			return text
		}
		if tuple.Len() > 1 {
			parts := make([]string, tuple.Len())
			params := make([]string, tuple.Len())
			changed := false
			for i := range parts {
				part := fmt.Sprintf("$r.$%d", i+1)
				parts[i] = e.eraseResult(origin.At(i).Type(), tuple.At(i).Type(), part)
				params[i] = e.dartType(tuple.At(i).Type())
				if parts[i] != part {
					changed = true
					params[i] = "Object?"
				}
			}
			if !changed {
				return text
			}
			return fmt.Sprintf("(((%s) $r) => (%s))(%s)", strings.Join(params, ", "), strings.Join(parts, ", "), text)
		}
		return e.eraseResult(origin.At(0).Type(), tuple.At(0).Type(), text)
	}
	if result == nil {
		return text
	}
	return e.eraseResult(origin.At(0).Type(), result, text)
}

func (e *emitter) eraseResult(origin, result types.Type, text string) string {
	if e.dartType(origin) != "Object?" {
		return text
	}
	dart := e.dartType(result)
	if dart == "Object?" {
		return text
	}
	// A generic function writes a cell for the address of a value, because it
	// holds no type argument. The Dart form of a pointer to a struct or to an
	// array is the object itself, thus the call site reads the cell out.
	if ptr, ok := types.Unalias(result).(*types.Pointer); ok && !needsBox(ptr.Elem()) {
		return fmt.Sprintf("$ptrTo<%s>(%s)", e.dartType(ptr.Elem()), text)
	}
	return "(" + text + " as " + dart + ")"
}

// spreadCall writes a call whose arguments come from the results of another
// call. Go passes each result of such a call as its own argument.
//
// A Dart record holds the results. The closure runs the inner call 1 time, and
// each argument reads 1 field of the record.
func (e *emitter) spreadCall(x *ir.Call) string {
	inner := x.Results
	x.Results = nil
	defer func() { x.Results = inner }()
	// Go reads the function operand first, and the inner call second. An
	// operand with an effect takes a parameter of its own, thus the order
	// stays the order of Go.
	switch fun := x.Fun.(type) {
	case *ir.Ident, *ir.Const:
	case *ir.Method:
		switch fun.Recv.(type) {
		case *ir.Ident, *ir.Const, *ir.Zero:
		default:
			out := *x
			out.Fun = &ir.Method{Recv: &ir.Ident{Name: "$fn", Type: fun.Recv.GoType()},
				Obj: fun.Obj, Type: fun.Type, TypeArgs: fun.TypeArgs}
			return fmt.Sprintf("((%s $fn, %s %s) => %s)(%s, %s)",
				e.dartType(fun.Recv.GoType()), e.dartType(inner.GoType()), x.ResultsVar,
				e.callText(&out), e.expr(fun.Recv), e.expr(inner))
		}
	default:
		out := *x
		out.Fun = &ir.Ident{Name: "$fn", Type: fun.GoType()}
		return fmt.Sprintf("((%s $fn, %s %s) => %s)(%s, %s)",
			e.dartType(fun.GoType()), e.dartType(inner.GoType()), x.ResultsVar,
			e.callText(&out), e.expr(fun), e.expr(inner))
	}
	body := e.callText(x)
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
	// The boundary of the interop: each Go integer argument becomes a plain
	// Dart int, and each integer result comes back as a Go integer. Thus the
	// runtime library keeps the plain int inside. The raw set opts out: the
	// float bit functions carry a 64-bit pattern that a JS number cannot
	// hold, and the worker carries a function value of byte slices.
	sig, _ := fn.Type().(*types.Signature)
	raw := intrinsicRaw[fn.Name()]
	if !raw && sig != nil {
		for i := range args {
			if i < sig.Params().Len() && isInteger(sig.Params().At(i).Type()) {
				args[i] = "$ix(" + args[i] + ")"
			}
		}
	}
	text, ok := e.intrinsicText(fn.Name(), args)
	if !ok {
		return "", false
	}
	if !raw && sig != nil {
		text = e.goIntResult(text, sig.Results())
	}
	return text, true
}

// intrinsicRaw lists each interop function whose boundary the mapped text
// handles itself.
var intrinsicRaw = map[string]bool{
	"SelectValue":   true,
	"SliceSwap":     true,
	"SliceLen":      true,
	"GrowValue":     true,
	"ValueLen":      true,
	"ValueCap":      true,
	"ValueIndex":    true,
	"ValueSetIndex": true,
	"ValueSlice":    true,
	"ValueAddr":     true,
	"MakeValue":     true,
	"CopyValue":     true,

	"GODEBUGCount":    true,
	"DescSize":        true,
	"DescLength":      true,
	"ArrayOf":         true,
	"DescOffset":      true,
	"AsInt":           true,
	"WithInt":         true,
	"NowNanos":        true,
	"MonoNanos":       true,
	"Float64Bits":     true,
	"Float64FromBits": true,
	"Float32Bits":     true,
	"Float32FromBits": true,
	"Worker":          true,
	"IntText":         true,
	"UintText":        true,
	"IntRadix":        true,
	"UintRadix":       true,
}

// goIntResult wraps the integer results of an interop call. A single integer
// takes $gi, a byte slice takes $gils, and a tuple with such a member reads
// each member through a small closure.
func (e *emitter) goIntResult(text string, rs *types.Tuple) string {
	member := func(t types.Type, text string) (string, bool) {
		if isInteger(t) {
			return "$gi(" + text + ")", true
		}
		if slice, ok := types.Unalias(t).Underlying().(*types.Slice); ok && isInteger(slice.Elem()) {
			return "$gils(" + text + ")", true
		}
		return text, false
	}
	switch rs.Len() {
	case 0:
		return text
	case 1:
		out, _ := member(rs.At(0).Type(), text)
		return out
	}
	needs := false
	parts := make([]string, rs.Len())
	raw := make([]string, rs.Len())
	for i := 0; i < rs.Len(); i++ {
		t := rs.At(i).Type()
		out, changed := member(t, fmt.Sprintf("$r.$%d", i+1))
		parts[i] = out
		if changed {
			needs = true
		}
		raw[i] = e.dartType(t)
		if isInteger(t) {
			raw[i] = "int"
		} else if slice, ok := types.Unalias(t).Underlying().(*types.Slice); ok && isInteger(slice.Elem()) {
			raw[i] = "Slice<int>"
		}
	}
	if !needs {
		return text
	}
	return fmt.Sprintf("(((%s) $r) => (%s))(%s)",
		strings.Join(raw, ", "), strings.Join(parts, ", "), text)
}

func (e *emitter) intrinsicText(name string, args []string) (string, bool) {
	switch name {
	case "Print":
		return fmt.Sprintf("$stdout(%s)", args[0]), true
	case "Format":
		return fmt.Sprintf("$goFormat(%s, %s)", args[1], args[0]), true
	case "FormatVerb":
		return fmt.Sprintf("$goFormatVerb(%s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4]), true
	case "VerifySystemCertificates":
		return fmt.Sprintf("$verifySystemCertificates(%s, %s, (%s).toDouble(), %s)", args[0], args[1], args[2], args[3]), true
	case "DialTLS":
		return fmt.Sprintf("$dialTLS(%s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4]), true
	case "DialTLSProxy":
		return fmt.Sprintf("$dialTLSProxy(%s, %s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4], args[5]), true
	case "ConnOption":
		return fmt.Sprintf("$connOption(%s, %s, %s)", args[0], args[1], args[2]), true
	case "PacketOption":
		return fmt.Sprintf("$packetOption(%s, %s, %s)", args[0], args[1], args[2]), true
	case "GC":
		return "$gc()", true
	case "AddCleanup":
		return fmt.Sprintf("$addCleanup(%s!.value as Object, %s)", args[0], args[1]), true
	case "StopCleanup":
		return fmt.Sprintf("$stopCleanup(%s)", args[0]), true
	case "Kind":
		return fmt.Sprintf("$goKind(%s)", args[0]), true
	case "SetValueHook":
		return fmt.Sprintf("$setValueHook(%s)", args[0]), true
	case "SetFormatHooks":
		return fmt.Sprintf("$setFormatHooks(%s, %s)", args[0], args[1]), true
	case "SliceOverlap":
		return fmt.Sprintf("$sliceOverlap(%s, %s)", args[0], args[1]), true
	case "IntVerb":
		return fmt.Sprintf("$goIntVerb(%s)", args[0]), true
	case "RecordPC":
		return fmt.Sprintf("$recordPC(%s)", args[0]), true
	case "PCInfo":
		return fmt.Sprintf("$pcInfo(%s)", args[0]), true
	case "PCEntry":
		return fmt.Sprintf("$pcEntry(%s)", args[0]), true
	case "Stack":
		return fmt.Sprintf("$stack(%s)", args[0]), true
	case "GoID":
		return "$goid()", true
	case "CaptureOut":
		return fmt.Sprintf("$captureOut(%s)", args[0]), true
	case "Parked":
		return "$parked()", true
	case "Gosched":
		return "$gosched()", true
	case "Goexit":
		return "$goexit()", true
	case "Fatal":
		return fmt.Sprintf("$fatalError(%s.toString())", args[0]), true
	case "SetTypeAssertionError":
		return fmt.Sprintf("$setTypeAssertionError(%s)", args[0]), true
	case "SetPanicNil":
		return fmt.Sprintf("$setPanicNil(%s)", args[0]), true
	case "SetRuntimeGODEBUG":
		return fmt.Sprintf("$setRuntimeGODEBUG(%s)", args[0]), true
	case "RuntimeGODEBUGVersion":
		return "$runtimeGODEBUGVersion", true
	case "DefaultGODEBUG":
		return "GoString.fromString($defaultGODEBUG)", true
	case "RegisterGODEBUG":
		return "$registerGODEBUG()", true
	case "IncGODEBUG":
		return fmt.Sprintf("$incGODEBUG(%s)", args[0]), true
	case "GODEBUGCount":
		return fmt.Sprintf("$godebugCount(%s)", args[0]), true
	case "TypeInfo":
		return fmt.Sprintf("$typeInfo(%s)", args[0]), true
	case "DeepEqual":
		return fmt.Sprintf("$deepEqual(%s, %s)", args[0], args[1]), true
	case "AssignError":
		return fmt.Sprintf("$assignError(%s, %s)", args[0], args[1]), true
	case "CheckErrorTarget":
		return fmt.Sprintf("$checkErrorTarget(%s)", args[0]), true
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
	case "DescNumMethod":
		return fmt.Sprintf("$descNumMethod(%s)", args[0]), true
	case "EmbeddedFieldError":
		return fmt.Sprintf("$embeddedFieldError(%s, %s, %s)", args[0], args[1], args[2]), true
	case "StructOf":
		return fmt.Sprintf("$structOf(%s, %s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4], args[5]), true
	case "DescMethodFunc":
		return fmt.Sprintf("$descMethodFunc(%s, %s)", args[0], args[1]), true
	case "DescMethodName":
		return fmt.Sprintf("$descMethodName(%s, %s)", args[0], args[1]), true
	case "DescMethodPkgPath":
		return fmt.Sprintf("$descMethodPkgPath(%s, %s)", args[0], args[1]), true
	case "MethodValue":
		return fmt.Sprintf("$methodValue(%s, %s, %s)", args[0], args[1], args[2]), true
	case "ValueIsNil":
		return fmt.Sprintf("$valueIsNil(%s)", args[0]), true
	case "MapKeys":
		return fmt.Sprintf("$mapKeys(%s)", args[0]), true
	case "MapIndex":
		return fmt.Sprintf("$mapIndex(%s, %s)", args[0], args[1]), true
	case "MapHas":
		return fmt.Sprintf("$mapHas(%s, %s)", args[0], args[1]), true
	case "FuncOf":
		return fmt.Sprintf("$funcOf(%s, %s, %s)", args[0], args[1], args[2]), true
	case "MapIterOf":
		return fmt.Sprintf("$mapIterOf(%s)", args[0]), true
	case "MapIterNext":
		return fmt.Sprintf("$mapIterNext(%s)", args[0]), true
	case "MapIterKey":
		return fmt.Sprintf("$mapIterKey(%s, %s)", args[0], args[1]), true
	case "MapIterValue":
		return fmt.Sprintf("$mapIterValue(%s, %s)", args[0], args[1]), true
	case "ValueRecv":
		return fmt.Sprintf("$valueRecv(%s)", args[0]), true
	case "ValueAddr":
		return fmt.Sprintf("$valueAddr(%s, %s, %s, $ix(%s))",
			args[0], args[1], args[2], args[3]), true
	case "ValueEqual":
		return fmt.Sprintf("$valueEqual(%s, %s)", args[0], args[1]), true
	case "ValueConvert":
		return fmt.Sprintf("$valueConvert(%s, %s)", args[0], args[1]), true
	case "ValueSlice":
		return fmt.Sprintf("$valueSlice(%s, %s, %s, %s)",
			args[0], args[1], args[2], args[3]), true
	case "AsComplex":
		return fmt.Sprintf("$asComplex(%s)", args[0]), true
	case "WithComplex":
		return fmt.Sprintf("$withComplex(%s, %s)", args[0], args[1]), true
	case "ValuePointer":
		return fmt.Sprintf("$valuePointer(%s)", args[0]), true
	case "DescImplements":
		return fmt.Sprintf("$descImplements(%s, %s)", args[0], args[1]), true
	case "DescAssignable":
		return fmt.Sprintf("$descAssignable(%s, %s)", args[0], args[1]), true
	case "DescConvertible":
		return fmt.Sprintf("$descConvertible(%s, %s)", args[0], args[1]), true
	case "FieldEmbedded":
		return fmt.Sprintf("$fieldEmbedded(%s, %s)", args[0], args[1]), true
	case "DescPkgPath":
		return fmt.Sprintf("$descPkgPath(%s)", args[0]), true
	case "DescVariadic":
		return fmt.Sprintf("$descVariadic(%s)", args[0]), true
	case "DescChanDir":
		return fmt.Sprintf("$descChanDir(%s)", args[0]), true
	case "SliceOf":
		return fmt.Sprintf("$sliceOf(%s)", args[0]), true
	case "ArrayOf":
		return fmt.Sprintf("$arrayOf(%s, %s)", args[0], args[1]), true
	case "MapOf":
		return fmt.Sprintf("$mapOf(%s, %s)", args[0], args[1]), true
	case "PointerTo":
		return fmt.Sprintf("$pointerTo(%s)", args[0]), true
	case "ChanOf":
		return fmt.Sprintf("$chanOf(%s, %s)", args[0], args[1]), true
	case "MakeChan":
		return fmt.Sprintf("$makeChan(%s, %s)", args[0], args[1]), true
	case "ValueSend":
		return fmt.Sprintf("$valueSend(%s, %s)", args[0], args[1]), true
	case "ValueClose":
		return fmt.Sprintf("$valueClose(%s)", args[0]), true
	case "ValueTryRecv":
		return fmt.Sprintf("$valueTryRecv(%s)", args[0]), true
	case "ValueTrySend":
		return fmt.Sprintf("$valueTrySend(%s, %s)", args[0], args[1]), true
	case "SelectValue":
		return fmt.Sprintf("$selectValue(%s, %s, %s)",
			args[0], args[1], args[2]), true
	case "ValueClear":
		return fmt.Sprintf("$valueClear(%s)", args[0]), true
	case "DescMethodType":
		return fmt.Sprintf("$descMethodType(%s, %s)", args[0], args[1]), true
	case "MakeFunc":
		return fmt.Sprintf("$makeFunc(%s, %s)", args[0], args[1]), true
	case "CallSliceValue":
		return fmt.Sprintf("$callSliceValue(%s, %s)", args[0], args[1]), true
	case "GrowValue":
		return fmt.Sprintf("$growValue(%s!, %s)", args[0], args[1]), true
	case "ValueCap":
		return fmt.Sprintf("$valueCap(%s)", args[0]), true
	case "AppendValue":
		return fmt.Sprintf("$appendValue(%s, %s)", args[0], args[1]), true
	case "AppendSliceValue":
		return fmt.Sprintf("$appendSliceValue(%s, %s)", args[0], args[1]), true
	case "CopyValue":
		return fmt.Sprintf("$copyValue(%s, %s)", args[0], args[1]), true
	case "MakeMap":
		return fmt.Sprintf("$makeMap(%s)", args[0]), true
	case "MapSet":
		return fmt.Sprintf("$mapSet(%s, %s, %s, %s)",
			args[0], args[1], args[2], args[3]), true
	case "DescLength":
		return fmt.Sprintf("$descLength(%s)", args[0]), true
	case "DescSize":
		return fmt.Sprintf("$descSize(%s)", args[0]), true
	case "DescAlign":
		return fmt.Sprintf("$descAlign(%s)", args[0]), true
	case "DescOffset":
		return fmt.Sprintf("$descOffset(%s, %s)", args[0], args[1]), true
	case "FieldCount":
		return fmt.Sprintf("$fieldCount(%s)", args[0]), true
	case "FieldPkgPath":
		return fmt.Sprintf("$fieldPkgPath(%s, %s)", args[0], args[1]), true
	case "FieldInfo":
		return fmt.Sprintf("$fieldInfo(%s, %s)", args[0], args[1]), true
	case "CopyInterface":
		return fmt.Sprintf("$copyInterface(%s)", args[0]), true
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
		return fmt.Sprintf("$withInt($ix(%s), %s)", args[0], args[1]), true
	case "WithFloat":
		return fmt.Sprintf("$withFloat(%s, %s)", args[0], args[1]), true
	case "WithBool":
		return fmt.Sprintf("$withBool(%s, %s)", args[0], args[1]), true
	case "WithString":
		return fmt.Sprintf("$withString(%s, %s)", args[0], args[1]), true
	case "NewValue":
		return fmt.Sprintf("$newValue(%s)", args[0]), true
	case "MakeValue":
		return fmt.Sprintf("$makeValue($ix(%s), %s, %s)", args[0], args[1], args[2]), true
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
	case "ParseFloat32":
		return fmt.Sprintf("$parseFloat32(%s)", args[0]), true
	case "Sleep":
		return fmt.Sprintf("$sleep(%s)", args[0]), true
	case "NowNanos":
		return "$nowNanos()", true
	case "ZoneOffset":
		return fmt.Sprintf("$zoneOffset(%s)", args[0]), true
	case "MonoNanos":
		return "$monoNanos()", true
	case "RandomBytes":
		return fmt.Sprintf("$randomBytes(%s)", args[0]), true
	case "WeakRef":
		return fmt.Sprintf("$weakRef(%s)", args[0]), true
	case "WeakTarget":
		return fmt.Sprintf("$weakTarget(%s)", args[0]), true
	case "JSONValid":
		return fmt.Sprintf("$jsonValid(%s)", args[0]), true
	case "JSONUnmarshal":
		return fmt.Sprintf("$jsonUnmarshal(%s, %s)", args[0], args[1]), true
	case "IntText":
		return fmt.Sprintf("$intText(%s)", args[0]), true
	case "UintText":
		return fmt.Sprintf("$uintText(%s)", args[0]), true
	case "IntRadix":
		return fmt.Sprintf("$intRadix(%s, $ix(%s))", args[0], args[1]), true
	case "UintRadix":
		return fmt.Sprintf("$uintRadix(%s, $ix(%s))", args[0], args[1]), true
	case "Sqrt":
		return fmt.Sprintf("$sqrt(%s)", args[0]), true
	case "Floor":
		return fmt.Sprintf("(%s).floorToDouble()", args[0]), true
	case "Ceil":
		return fmt.Sprintf("(%s).ceilToDouble()", args[0]), true
	case "Trunc":
		return fmt.Sprintf("(%s).truncateToDouble()", args[0]), true
	case "Abs":
		return fmt.Sprintf("(%s).abs()", args[0]), true
	case "Float64Bits":
		return fmt.Sprintf("$f64ToBits(%s)", args[0]), true
	case "Float64FromBits":
		return fmt.Sprintf("$f64FromBits(%s)", args[0]), true
	case "Float32Bits":
		return fmt.Sprintf("$f32ToBits(%s)", args[0]), true
	case "Float32FromBits":
		return fmt.Sprintf("$f32FromBits(%s)", args[0]), true
	case "FileOpenFlags":
		return fmt.Sprintf("$fileOpenFlags(%s, %s, %s)", args[0], args[1], args[2]), true
	case "FileOpen":
		return fmt.Sprintf("$fileOpen(%s, %s)", args[0], args[1]), true
	case "FileFd":
		return fmt.Sprintf("$fileFd(%s, %s)", args[0], args[1]), true
	case "SyscallOpen":
		return fmt.Sprintf("$syscallOpen(%s, %s, %s)", args[0], args[1], args[2]), true
	case "SyscallSeek":
		return fmt.Sprintf("$syscallSeek(%s, %s, %s)", args[0], args[1], args[2]), true
	case "SyscallReadlink":
		return fmt.Sprintf("$syscallReadlink(%s, %s)", args[0], args[1]), true
	case "SyscallUnlink":
		return fmt.Sprintf("$syscallUnlink(%s, %s)", args[0], args[1]), true
	case "SyscallClose":
		return fmt.Sprintf("$syscallClose(%s)", args[0]), true
	case "SyscallReadDirent":
		return fmt.Sprintf("$syscallReadDirent(%s, %s)", args[0], args[1]), true
	case "DirectoryOpen":
		return fmt.Sprintf("$directoryOpen(%s)", args[0]), true
	case "DirectoryFd":
		return fmt.Sprintf("$directoryFd(%s)", args[0]), true
	case "DirectoryChdir":
		return fmt.Sprintf("$directoryChdir(%s)", args[0]), true
	case "DirectoryClose":
		return fmt.Sprintf("$directoryClose(%s)", args[0]), true
	case "FileRead":
		return fmt.Sprintf("$fileRead(%s, %s)", args[0], args[1]), true
	case "FileWrite":
		return fmt.Sprintf("$fileWrite(%s, %s)", args[0], args[1]), true
	case "FileSeek":
		return fmt.Sprintf("$fileSeek(%s, %s, %s)", args[0], args[1], args[2]), true
	case "FileClose":
		return fmt.Sprintf("$fileClose(%s)", args[0]), true
	case "NativeTruncate":
		return fmt.Sprintf("$nativeTruncate(%s, %s)", args[0], args[1]), true
	case "NativePread":
		return fmt.Sprintf("$nativePread(%s, %s, %s)", args[0], args[1], args[2]), true
	case "NativePwrite":
		return fmt.Sprintf("$nativePwrite(%s, %s, %s)", args[0], args[1], args[2]), true
	case "NativeFtruncate":
		return fmt.Sprintf("$nativeFtruncate(%s, %s)", args[0], args[1]), true
	case "NativeFsync":
		return fmt.Sprintf("$nativeFsync(%s)", args[0]), true
	case "FileSync":
		return fmt.Sprintf("$fileSync(%s)", args[0]), true
	case "FileTruncate":
		return fmt.Sprintf("$fileTruncate(%s, %s)", args[0], args[1]), true
	case "ReadFile":
		return fmt.Sprintf("$readFile(%s)", args[0]), true
	case "HasNativeStat":
		return "$hasNativeStat", true
	case "NativeStat":
		return fmt.Sprintf("$nativeStat(%s, %s)", args[0], args[1]), true
	case "NativeLstatat":
		return fmt.Sprintf("$nativeLstatat(%s, %s)", args[0], args[1]), true
	case "NativeFstat":
		return fmt.Sprintf("$nativeFstat(%s)", args[0]), true
	case "Stat":
		return fmt.Sprintf("$stat(%s)", args[0]), true
	case "Lstat":
		return fmt.Sprintf("$lstat(%s)", args[0]), true
	case "NativeLink":
		return fmt.Sprintf("$nativeLink(%s, %s)", args[0], args[1]), true
	case "NativeSymlink":
		return fmt.Sprintf("$nativeSymlink(%s, %s)", args[0], args[1]), true
	case "Symlink":
		return fmt.Sprintf("$symlink(%s, %s)", args[0], args[1]), true
	case "Link":
		return fmt.Sprintf("$link(%s, %s)", args[0], args[1]), true
	case "Umask":
		return "$umask()", true
	case "FileChmod":
		return fmt.Sprintf("$fileChmod(%s, %s)", args[0], args[1]), true
	case "FileChown":
		return fmt.Sprintf("$fileChown(%s, %s, %s)", args[0], args[1], args[2]), true
	case "NativeChmod":
		return fmt.Sprintf("$nativeChmod(%s, %s)", args[0], args[1]), true
	case "NativeRename":
		return fmt.Sprintf("$nativeRename(%s, %s)", args[0], args[1]), true
	case "NativeChown":
		return fmt.Sprintf("$nativeChown(%s, %s, %s, %s)",
			args[0], args[1], args[2], args[3]), true
	case "Chmod":
		return fmt.Sprintf("$chmod(%s, %s)", args[0], args[1]), true
	case "Chown":
		return fmt.Sprintf("$chown(%s, %s, %s, %s)",
			args[0], args[1], args[2], args[3]), true
	case "NativeChtimes":
		return fmt.Sprintf("$nativeChtimes(%s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4]), true
	case "Chtimes":
		return fmt.Sprintf("$chtimes(%s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4]), true
	case "NativeReadlink":
		return fmt.Sprintf("$nativeReadlink(%s)", args[0]), true
	case "ReadLink":
		return fmt.Sprintf("$readLink(%s)", args[0]), true
	case "ResolveSymlinks":
		return fmt.Sprintf("$resolveSymlinks(%s)", args[0]), true
	case "Remove":
		return fmt.Sprintf("$remove(%s)", args[0]), true
	case "RemoveAll":
		return fmt.Sprintf("$removeAll(%s)", args[0]), true
	case "DirectoryCreate":
		return fmt.Sprintf("$directoryCreate(%s, %s)", args[0], args[1]), true
	case "Mkdir":
		return fmt.Sprintf("$mkdir(%s)", args[0]), true
	case "Rename":
		return fmt.Sprintf("$rename(%s, %s)", args[0], args[1]), true
	case "FileIdentical":
		return fmt.Sprintf("$fileIdentical(%s, %s)", args[0], args[1]), true
	case "ReadDir":
		return fmt.Sprintf("$readDir(%s)", args[0]), true
	case "Getenv":
		return fmt.Sprintf("$getenv(%s)", args[0]), true
	case "Environ":
		return "$environ()", true
	case "SyscallGetwd":
		return "$syscallGetwd()", true
	case "NativeGetwd":
		return "$nativeGetwd()", true
	case "NativeChdir":
		return fmt.Sprintf("$nativeChdir(%s)", args[0]), true
	case "Getwd":
		return "$getwd()", true
	case "Getpid", "Getuid", "Geteuid", "Getgid", "Getegid", "Getppid", "Getpgrp", "Getpagesize", "Getgroups":
		return "$" + strings.ToLower(name) + "()", true
	case "Getpgid", "Getsid":
		return "$" + strings.ToLower(name) + "(" + args[0] + ")", true
	case "Hostname":
		return "$hostname()", true
	case "Executable":
		return "$executable()", true
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
		return fmt.Sprintf("$dialTCP(%s, %s, %s, %s)", args[0], args[1], args[2], args[3]), true
	case "ProcStart":
		return fmt.Sprintf("$procStart(%s, %s, %s, %s)",
			args[0], args[1], args[2], args[3]), true
	case "ProcWrite":
		return fmt.Sprintf("$procWrite(%s, %s)", args[0], args[1]), true
	case "ProcCloseStdin":
		return fmt.Sprintf("$procCloseStdin(%s)", args[0]), true
	case "ProcCloseIO":
		return fmt.Sprintf("$procCloseIO(%s)", args[0]), true
	case "SendSignal":
		return fmt.Sprintf("$sendSignal(%s, %s)", args[0], args[1]), true
	case "KillPid":
		return fmt.Sprintf("$killPid(%s, %s)", args[0], args[1]), true
	case "SignalWatch":
		return fmt.Sprintf("$signalWatch(%s)", args[0]), true
	case "SignalWait":
		return fmt.Sprintf("$signalWait(%s)", args[0]), true
	case "SignalStop":
		return fmt.Sprintf("$signalStop(%s)", args[0]), true
	case "ProcRead":
		return fmt.Sprintf("$procRead(%s, %s)", args[0], args[1]), true
	case "ProcReadErr":
		return fmt.Sprintf("$procReadErr(%s, %s)", args[0], args[1]), true
	case "ProcWait":
		return fmt.Sprintf("$procWait(%s)", args[0]), true
	case "ProcKill":
		return fmt.Sprintf("$procKill(%s, %s)", args[0], args[1]), true
	case "H2Client":
		return fmt.Sprintf("$h2Client(%s)", args[0]), true
	case "H2Fetch":
		return fmt.Sprintf("$h2Fetch(%s, %s, %s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4], args[5], args[6]), true
	case "H2Read":
		return fmt.Sprintf("$h2Read(%s, %s)", args[0], args[1]), true
	case "H2Server":
		return fmt.Sprintf("$h2Server(%s)", args[0]), true
	case "H2Next":
		return fmt.Sprintf("$h2Next(%s)", args[0]), true
	case "H2WriteHead":
		return fmt.Sprintf("$h2WriteHead(%s, %s, %s, %s)",
			args[0], args[1], args[2], args[3]), true
	case "H2Write":
		return fmt.Sprintf("$h2Write(%s, %s)", args[0], args[1]), true
	case "H2End":
		return fmt.Sprintf("$h2End(%s)", args[0]), true
	case "H2Close":
		return fmt.Sprintf("$h2Close(%s)", args[0]), true
	case "FetchOpen":
		return "$fetchOpen()", true
	case "FetchCancel":
		return fmt.Sprintf("$fetchCancel(%s)", args[0]), true
	case "FetchStart":
		return fmt.Sprintf("$fetchStart(%s, %s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4], args[5]), true
	case "FetchRead":
		return fmt.Sprintf("$fetchRead(%s, %s)", args[0], args[1]), true
	case "FetchClose":
		return fmt.Sprintf("$fetchClose(%s)", args[0]), true
	case "ConnAddrs":
		return fmt.Sprintf("$connAddrs(%s)", args[0]), true
	case "ConnRead":
		return fmt.Sprintf("$connRead(%s, %s)", args[0], args[1]), true
	case "ConnWrite":
		return fmt.Sprintf("$connWrite(%s, %s)", args[0], args[1]), true
	case "ConnClose":
		return fmt.Sprintf("$connClose(%s)", args[0]), true
	case "ListenTCP":
		return fmt.Sprintf("$listenTCP(%s, %s, %s)", args[0], args[1], args[2]), true
	case "ListenTCPHost":
		return fmt.Sprintf("$listenTCPHost(%s)", args[0]), true
	case "ListenTLS":
		return fmt.Sprintf("$listenTLS(%s, %s, %s, %s, %s)",
			args[0], args[1], args[2], args[3], args[4]), true
	case "AcceptTLS":
		return fmt.Sprintf("$acceptTLS(%s)", args[0]), true
	case "Accept":
		return fmt.Sprintf("$accept(%s)", args[0]), true
	case "ListenDeadline":
		return fmt.Sprintf("$listenDeadline(%s, %s)", args[0], args[1]), true
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
	case "ListenMulticastUDP":
		return fmt.Sprintf("$listenMulticastUDP(%s, %s)", args[0], args[1]), true
	case "JoinMulticast":
		return fmt.Sprintf("$joinMulticast(%s, %s, %s)",
			args[0], args[1], args[2]), true
	case "DialUDP":
		return fmt.Sprintf("$dialUDP(%s, %s, %s, %s)", args[0], args[1], args[2], args[3]), true
	case "PacketRead":
		return fmt.Sprintf("$packetRead(%s, %s)", args[0], args[1]), true
	case "PacketWrite":
		return fmt.Sprintf("$packetWrite(%s, %s, %s, %s)",
			args[0], args[1], args[2], args[3]), true
	case "PacketClose":
		return fmt.Sprintf("$packetClose(%s)", args[0]), true
	case "PacketDeadline":
		return fmt.Sprintf("$packetDeadline(%s, %s, %s)", args[0], args[1], args[2]), true
	case "InterfaceMulticastAddrs":
		return fmt.Sprintf("$interfaceMulticastAddrs(%s)", args[0]), true
	case "InterfaceTable":
		return "$interfaceTable()", true
	case "Interfaces":
		return "$interfaces()", true
	case "StartTimer":
		return fmt.Sprintf("$startTimer(%s, %s)", args[0], args[1]), true
	case "MarkTimerChannel":
		return fmt.Sprintf("$markTimerChannel(%s)", args[0]), true
	case "StopTimer":
		return fmt.Sprintf("$stopTimer(%s)", args[0]), true
	case "ResetTimer":
		return fmt.Sprintf("$resetTimer(%s)", args[0]), true
	case "LockToken":
		return "$lockToken()", true
	case "LockWait":
		return fmt.Sprintf("$lockWait(%s)", args[0]), true
	case "LockWake":
		return fmt.Sprintf("$lockWake(%s)", args[0]), true
	case "InBubble":
		return "$inBubble()", true
	case "BubbleRun":
		return fmt.Sprintf("$bubbleRun(%s)", args[0]), true
	case "BubbleWait":
		return "$bubbleWait()", true
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
	e.fail("the interop function %s has no runtime form", name)
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
	decl := e.declSignature(x)
	if sig == nil || !x.Variadic || x.Spread {
		out := make([]string, len(x.Args))
		for i, a := range x.Args {
			out[i] = e.adaptArg(e.expr(a), a, decl, i)
		}
		return out
	}
	last := sig.Params().Len() - 1
	out := make([]string, 0, last+1)
	for i := 0; i < last && i < len(x.Args); i++ {
		out = append(out, e.adaptArg(e.expr(x.Args[i]), x.Args[i], decl, i))
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
	if localStorageCycle(types.NewSlice(elem)) {
		return fmt.Sprintf("Slice<%s>.of(<%s>[%s])", e.dartType(elem), e.dartType(elem), strings.Join(parts, ", "))
	}
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
	// A method call on a type parameter with no core type dispatches through
	// the method table of its dictionary, exactly as an interface call does.
	// A parameter whose core type is an interface already holds an Iface.
	if recv, ok := e.typeParamRecv(m); ok {
		sig, _ := m.Obj.Type().(*types.Signature)
		call := fmt.Sprintf("$call(%s, %s, <Object?>[%s])",
			recv, e.methodKeyText(m.Obj.Id(), sig), strings.Join(args, ", "))
		if result := e.resultType(sig); result != "void" {
			return fmt.Sprintf("(%s as %s)", call, result)
		}
		return call
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
	return fmt.Sprintf("%s.%s%s(%s)", e.deref(m.Recv), e.memberName(m.Obj), targs, list)
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
	if isInterfaceType(m.Recv.GoType()) {
		return e.interfaceMethodValue(m)
	}
	if recv, ok := e.typeParamRecv(m); ok {
		return e.interfaceMethodValueOf(m, recv)
	}
	obj := receiverTypeName(m.Recv.GoType())
	if obj != nil && (!isStructType(obj.Type()) || pointerMethod(m.Obj)) {
		return e.methodFuncValue(m, obj)
	}
	if len(m.TypeArgs) > 0 || e.methodValueCast(m, "$value") != "$value" {
		return e.genericMethodValue(m)
	}
	sig, _ := m.Obj.Type().(*types.Signature)
	if sig != nil && sig.Recv() != nil && !isPointerType(sig.Recv().Type()) {
		return fmt.Sprintf("%s.$copy().%s", e.deref(m.Recv), e.memberName(m.Obj))
	}
	return fmt.Sprintf("%s.%s", e.deref(m.Recv), e.memberName(m.Obj))
}

// genericMethodValue reads an instance of a generic method without a call. A
// Dart tear-off binds no type argument, thus the value is a closure. The
// closure holds the receiver and the dictionary of each type argument.
func (e *emitter) genericMethodValue(m *ir.Method) string {
	sig, _ := m.Type.(*types.Signature)
	if sig == nil {
		e.fail("the method value %s has no signature", m.Obj.Name())
		return "null"
	}
	targs, dicts := e.typeArguments(m.TypeArgs)
	params := make([]string, sig.Params().Len())
	names := make([]string, sig.Params().Len())
	for i := range params {
		names[i] = fmt.Sprintf("$a%d", i)
		params[i] = e.dartType(sig.Params().At(i).Type()) + " " + names[i]
	}
	recv := e.deref(m.Recv)
	msig, _ := m.Obj.Type().(*types.Signature)
	if msig != nil && msig.Recv() != nil && !isPointerType(msig.Recv().Type()) {
		recv += ".$copy()"
	}
	rt := m.Recv.GoType()
	if p, ok := typeset.Core(rt).(*types.Pointer); ok {
		rt = p.Elem()
	}
	all := append(append([]string{}, dicts...), names...)
	call := "$recv." + e.memberName(m.Obj) + targs + "(" + strings.Join(all, ", ") + ")"
	call = e.methodValueCast(m, call)
	return fmt.Sprintf("((%s $recv) => (%s) => %s)(%s)",
		e.dartType(rt), strings.Join(params, ", "), call, recv)
}

func (e *emitter) methodValueCast(m *ir.Method, text string) string {
	origin := m.Obj.Origin().Type().(*types.Signature)
	inst := m.Type.(*types.Signature)
	return e.eraseResults(origin.Results(), inst.Results(), text)
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
	origin := x.Fun.Obj.(*types.Func).Origin().Type().(*types.Signature)
	call = e.eraseResults(origin.Results(), sig.Results(), call)
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
	msig, _ := x.Obj.Type().(*types.Signature)
	mtargs, mdicts := e.typeArguments(x.TypeArgs)
	closure := func(call string) string {
		origin := x.Obj.Origin().Type().(*types.Signature)
		call = e.eraseResults(origin.Results(), sig.Results(), call)
		text := fmt.Sprintf("((%s) => %s)", strings.Join(params, ", "), call)
		if tp, ok := types.Unalias(x.Recv).(*types.TypeParam); ok {
			return fmt.Sprintf("$copyFunctionPC(%s, %s.desc.methods[%s])", text, e.dictOf(tp), e.methodKeyText(x.Obj.Id(), msig))
		}
		if x.Ordinal > 0 && e.cfg.Fset != nil {
			return e.genericWrapperValue(x.Pos, x.Ordinal, text)
		}
		return e.methodFunctionValue(x.Recv, x.Obj, text)
	}
	// The method of a type parameter with no single Dart class dispatches
	// through the method table, exactly as an interface receiver does. A
	// parameter with no core type boxes the value with its dictionary.
	if tp, ok := types.Unalias(x.Recv).(*types.TypeParam); ok {
		core := typeset.Core(tp)
		_, coreIface := core.(*types.Interface)
		if coreDispatch(core) || coreIface {
			recv := names[0]
			if !coreIface {
				recv = fmt.Sprintf("$boxDesc(%s.desc, %s)", e.dictOf(tp), recv)
			}
			call := fmt.Sprintf("$call(%s, %s, <Object?>[%s])",
				recv, e.methodKeyText(x.Obj.Id(), msig), strings.Join(names[1:], ", "))
			if cast := e.resultCast(msig); cast != "" {
				call = "(" + call + " as " + cast + ")"
			}
			return closure(call)
		}
	}
	// The method of an interface type dispatches through the method table of
	// the receiver argument.
	if isInterfaceType(x.Recv) {
		call := fmt.Sprintf("$call(%s, %s, <Object?>[%s])",
			names[0], e.methodKeyText(x.Obj.Id(), msig), strings.Join(names[1:], ", "))
		if cast := e.resultCast(msig); cast != "" {
			call = "(" + call + " as " + cast + ")"
		}
		return closure(call)
	}
	// A promoted method reads its embedded fields first, exactly as the
	// adapter of a method table does.
	recv := names[0]
	current := x.Recv
	var receiver ir.Expr = &ir.Ident{Name: recv, Type: current}
	if pkg := x.Obj.Pkg(); pkg != nil {
		_, index, _ := types.LookupFieldOrMethod(x.Recv, true, pkg, x.Obj.Name())
		if len(index) > 1 {
			for _, fi := range index[:len(index)-1] {
				st := structOf(current)
				if st == nil {
					break
				}
				field := st.Field(fi)
				receiver = &ir.Field{X: receiver, Name: field.Name(), Type: field.Type()}
				recv = e.expr(receiver)
				current = field.Type()
			}
		}
	}
	if isInterfaceType(current) {
		call := fmt.Sprintf("$call(%s, %s, <Object?>[%s])",
			recv, e.methodKeyText(x.Obj.Id(), msig), strings.Join(names[1:], ", "))
		if cast := e.resultCast(msig); cast != "" {
			call = "(" + call + " as " + cast + ")"
		}
		return closure(call)
	}
	obj := receiverTypeName(current)
	if obj != nil && (!isStructType(obj.Type()) || pointerMethod(x.Obj)) {
		// The body stands in a top level function. A value method through a
		// pointer receiver argument reads the value behind the pointer.
		arg0 := recv
		if pointerMethod(x.Obj) && !isPointerType(current) && needsBox(current) {
			arg0 = e.addressExpr(receiver)
		}
		if !pointerMethod(x.Obj) && isPointerType(current) {
			if p, ok := typeset.Core(current).(*types.Pointer); ok && needsBox(p.Elem()) {
				arg0 += "!.v"
			} else {
				arg0 += "!"
			}
		}
		dicts, typeArgs := e.recvTypeArgs(current)
		all := append(append(append([]string{arg0}, dicts...), mdicts...), names[1:]...)
		call := e.qualify(obj, methodFuncName(obj.Name(), x.Obj.Name())) +
			joinTypeArgs(typeArgs, mtargs) + "(" + strings.Join(all, ", ") + ")"
		return closure(call)
	}
	// A class method. A value receiver takes a copy at the call, exactly as
	// Go copies it.
	if isPointerType(current) {
		recv += "!"
	}
	if msig != nil && msig.Recv() != nil && !isPointerType(msig.Recv().Type()) {
		recv += ".$copy()"
	}
	return closure(fmt.Sprintf("%s.%s%s(%s)", recv, e.memberName(x.Obj), mtargs,
		strings.Join(append(append([]string{}, mdicts...), names[1:]...), ", ")))
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
	if len(m.TypeArgs) > 0 {
		if inst, ok := m.Type.(*types.Signature); ok {
			sig = inst
		}
	}
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
	mtargs, mdicts := e.typeArguments(m.TypeArgs)
	dicts, typeArgs := e.recvTypeArgs(m.Recv.GoType())
	args = append(append(append([]string{args[0]}, dicts...), mdicts...), args[1:]...)
	call := e.qualify(obj, methodFuncName(obj.Name(), m.Obj.Name())) +
		joinTypeArgs(typeArgs, mtargs) + "(" + strings.Join(args, ", ") + ")"
	// The receiver takes the form of the top level function: a pointer method
	// takes the address of a value, and a value method reads behind a pointer.
	rt := m.Recv.GoType()
	if pointerMethod(m.Obj) && !isPointerType(rt) && needsBox(rt) {
		rt = types.NewPointer(rt)
	} else if !pointerMethod(m.Obj) && isPointerType(rt) {
		if p, ok := typeset.Core(rt).(*types.Pointer); ok {
			rt = p.Elem()
		}
	}
	return fmt.Sprintf("((%s $recv) => (%s) => %s)(%s)",
		e.dartType(rt), strings.Join(params, ", "), call, e.topLevelRecv(m))
}

// interfaceMethodValue reads a method of an interface value without a call. The
// closure holds the value, and each call reads the method table of the
// descriptor.
func (e *emitter) interfaceMethodValue(m *ir.Method) string {
	return e.interfaceMethodValueOf(m, e.expr(m.Recv))
}

// typeParamRecv gives the Iface form of a receiver whose type is a type
// parameter with no single Dart class. A parameter with no core type boxes
// the value with the descriptor of its dictionary, and a parameter whose
// core type is an interface already holds an Iface.
func (e *emitter) typeParamRecv(m *ir.Method) (string, bool) {
	tp, ok := types.Unalias(m.Recv.GoType()).(*types.TypeParam)
	if !ok {
		return "", false
	}
	core := typeset.Core(tp)
	if _, ok := core.(*types.Interface); ok {
		return e.expr(m.Recv), true
	}
	if coreDispatch(core) {
		return fmt.Sprintf("$boxDesc(%s.desc, %s)", e.dictOf(tp), e.expr(m.Recv)), true
	}
	return "", false
}

func (e *emitter) interfaceMethodValueOf(m *ir.Method, recv string) string {
	sig, _ := m.Obj.Type().(*types.Signature)
	if sig == nil {
		e.fail("the method %s has no signature", m.Obj.Name())
		return "null"
	}
	params := make([]string, sig.Params().Len())
	types := make([]string, sig.Params().Len()+1)
	args := make([]string, sig.Params().Len()+1)
	types[0] = "Object?"
	args[0] = "$recv!.value"
	for i := range params {
		name := fmt.Sprintf("$a%d", i)
		text := e.dartType(sig.Params().At(i).Type())
		params[i] = text + " " + name
		types[i+1] = text
		args[i+1] = name
	}
	call := fmt.Sprintf("($method($recv, %s) as %s Function(%s))(%s)",
		e.methodKeyText(m.Obj.Id(), sig), e.resultType(sig),
		strings.Join(types, ", "), strings.Join(args, ", "))
	if paramsHoldParam(sig) {
		call = fmt.Sprintf("$call($recv, %s, <Object?>[%s])",
			e.methodKeyText(m.Obj.Id(), sig), strings.Join(args[1:], ", "))
		if result := e.resultType(sig); result != "void" {
			call = fmt.Sprintf("(%s as %s)", call, result)
		}
	}
	return fmt.Sprintf("((Iface? $recv) { if ($recv == null) $panicNil(); return (%s) => %s; })(%s)",
		strings.Join(params, ", "), call, recv)
}

func (e *emitter) builtin(x *ir.Builtin) string {
	if x.Results != nil {
		inner := x.Results
		x.Results = nil
		body := e.builtin(x)
		x.Results = inner
		return fmt.Sprintf("((%s %s) => %s)(%s)",
			e.dartType(inner.GoType()), x.ResultsVar, body, e.expr(inner))
	}
	switch x.Name {
	case "unsafe.Sizeof":
		return fmt.Sprintf("$typeSize(%s)", e.descriptorOf(x.Args[0].GoType()))
	case "unsafe.Alignof":
		return fmt.Sprintf("$gi($typeAlign(%s))", e.descriptorOf(x.Args[0].GoType()))
	case "unsafe.Offsetof":
		return fmt.Sprintf("$fieldOffset(%s, $ix(%s))", e.descriptorOf(x.Args[0].GoType()), e.expr(x.Args[1]))
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
	case "recover":
		return fmt.Sprintf("$recover(%d)", e.curFuncId)
	case "print":
		return fmt.Sprintf("$print(<String>[%s])", strings.Join(e.textList(x.Args), ", "))
	case "println":
		return fmt.Sprintf("$println(<String>[%s])", strings.Join(e.textList(x.Args), ", "))
	case "decoderune":
		return fmt.Sprintf("$decodeRune(%s, %s)", e.expr(x.Args[0]), e.expr(x.Args[1]))
	case "min", "max":
		return e.minMax(x)
	case "delete":
		text := e.container(x.Args[0])
		if typeset.Core(x.Args[0].GoType()) == nil {
			text = "(" + text + " as GoMap<Object?, Object?>)"
		}
		return fmt.Sprintf("%s.remove(%s)", text, e.expr(x.Args[1]))
	case "clear":
		if isMapType(x.Args[0].GoType()) {
			return e.container(x.Args[0]) + ".clear()"
		}
		if slice, ok := typeset.Core(x.Args[0].GoType()).(*types.Slice); ok {
			return fmt.Sprintf("$clearSlice(%s, () => %s%s)",
				e.sliceText(x.Args[0]), e.zero(slice.Elem()), e.elemAssign(slice.Elem()))
		}
		e.fail("clear does not apply to the type %s", x.Args[0].GoType())
		return "null"
	case "mapiter":
		return fmt.Sprintf("GoMapIter(%s)", e.container(x.Args[0]))
	case "iternext":
		return e.expr(x.Args[0]) + ".next()"
	case "iterkey":
		return e.expr(x.Args[0]) + ".key"
	case "iterval":
		return e.expr(x.Args[0]) + ".value"
	case "close":
		text := e.chanExpr(x.Args[0])
		if typeset.Core(x.Args[0].GoType()) == nil {
			text = "(" + text + " as Chan<Object?>)"
		}
		return text + ".close()"
	case "real":
		if b := basicOf(x.Type); b != nil && b.Kind() == types.Float32 {
			return fmt.Sprintf("(%s as GoComplex32).real32", e.expr(x.Args[0]))
		}
		return fmt.Sprintf("%s.real64", e.expr(x.Args[0]))
	case "imag":
		if b := basicOf(x.Type); b != nil && b.Kind() == types.Float32 {
			return fmt.Sprintf("(%s as GoComplex32).imag32", e.expr(x.Args[0]))
		}
		return fmt.Sprintf("%s.imag64", e.expr(x.Args[0]))
	case "complex":
		if b := basicOf(x.Type); b != nil && b.Kind() == types.Complex64 {
			return fmt.Sprintf("GoComplex32(%s, %s)", e.expr(x.Args[0]), e.expr(x.Args[1]))
		}
		return fmt.Sprintf("GoComplex.fromParts(%s, %s)", e.expr(x.Args[0]), e.expr(x.Args[1]))
	}
	e.fail("the builtin %s has no Dart form yet", x.Name)
	return "null"
}

func (e *emitter) length(x ir.Expr) string {
	t := x.GoType()
	if typeset.BytesOrText(t) {
		return fmt.Sprintf("%s.length!(%s)", e.dictOf(t), e.expr(x))
	}
	switch {
	case isString(t):
		return "$gi(" + e.expr(x) + ".length)"
	case isSliceType(t):
		return e.sliceText(x) + ".$length"
	case isArrayType(t):
		return "$arrayLength(" + e.expr(x) + ")"
	case isMapType(t):
		return "$gi(" + e.container(x) + ".length)"
	case isChanType(t):
		return "$gi(" + e.chanExpr(x) + ".length)"
	case isPointerType(t):
		// The length of a pointer to an array comes from the type, and Go
		// evaluates the operand for a constant length in no case, thus a nil
		// pointer also gives the length. The closure keeps the side effect of
		// the operand.
		if p, ok := typeset.Core(t).(*types.Pointer); ok {
			if arr, ok := types.Unalias(p.Elem()).Underlying().(*types.Array); ok {
				return fmt.Sprintf("((Object? $p) => %s)(%s)", intConst(constant.MakeInt64(arr.Len()), types.Typ[types.Int]), e.expr(x))
			}
		}
		return "$gi(" + e.deref(x) + ".length)"
	}
	e.fail("len does not apply to the type %s", t)
	return "0"
}

func (e *emitter) capacity(x ir.Expr) string {
	t := x.GoType()
	switch {
	case isSliceType(t):
		return e.sliceText(x) + ".$capacity"
	case isArrayType(t):
		return "$arrayLength(" + e.expr(x) + ")"
	case isChanType(t):
		return "$gi(" + e.chanExpr(x) + ".capacity)"
	case isPointerType(t):
		// Read the note of length: the value comes from the type.
		if p, ok := typeset.Core(t).(*types.Pointer); ok {
			if arr, ok := types.Unalias(p.Elem()).Underlying().(*types.Array); ok {
				return fmt.Sprintf("((Object? $p) => %s)(%s)", intConst(constant.MakeInt64(arr.Len()), types.Typ[types.Int]), e.expr(x))
			}
		}
		return "$gi(" + e.deref(x) + ".length)"
	}
	e.fail("cap does not apply to the type %s", t)
	return "0"
}

// mapNew writes the constructor of a map. A key of an array type compares by
// value in Go, and a Dart list compares by identity, thus such a map takes
// its own equality and its own hash.
func (e *emitter) mapNew(m *types.Map) string {
	return fmt.Sprintf("GoMap<%s, %s>(%s%s)",
		e.dartType(m.Key()), e.dartType(m.Elem()), e.zeroFunc(m.Elem()), e.mapKeyOptions(m))
}

func (e *emitter) mapKeyOptions(m *types.Map) string {
	if tp, ok := types.Unalias(m.Key()).(*types.TypeParam); ok {
		dict := e.dictOf(tp)
		return fmt.Sprintf(", equals: %s.equal, hashCode: %s.hash", dict, dict)
	}
	// A pointer compares by identity in Go. A pointer to a struct is the Dart
	// object of the struct, and the class compares the fields, thus such a
	// key needs the identity of Dart.
	if p, ok := typeset.Core(m.Key()).(*types.Pointer); ok && isStructType(p.Elem()) {
		return ", equals: identical, hashCode: identityHashCode"
	}
	// A NaN key never compares equal, thus each write of one adds an entry,
	// and a read of one finds nothing. The map of Dart takes 2 identical
	// doubles as 1 key, thus a float key and a complex key need the plain
	// equality. Each NaN takes a fresh hash, and GoMap.nanHash tells the
	// cause.
	if isFloat(m.Key()) || isComplexType(m.Key()) {
		return ", equals: ($ka, $kb) => $ka == $kb, " +
			"hashCode: ($k) => $k == $k ? $k.hashCode : GoMap.nanHash()"
	}
	if _, ok := typeset.Core(m.Key()).(*types.Array); !ok {
		return ""
	}
	return fmt.Sprintf(", equals: ($ka, $kb) => %s, hashCode: ($k) => %s",
		e.arrayKeyEqual(m.Key(), "$ka", "$kb", 0),
		e.arrayKeyHash(m.Key(), "$k", 0))
}

// arrayKeyEqual writes the value equality of a map key of an array type. An
// array inside the array compares the same way.
func (e *emitter) arrayKeyEqual(t types.Type, a, b string, depth int) string {
	arr, ok := typeset.Core(t).(*types.Array)
	if !ok {
		return e.equalText(a, b, t)
	}
	x := fmt.Sprintf("$x%d", depth)
	y := fmt.Sprintf("$y%d", depth)
	return fmt.Sprintf("$equalArray(%s, %s, (%s, %s) => %s)",
		a, b, x, y, e.arrayKeyEqual(arr.Elem(), x, y, depth+1))
}

// arrayKeyHash writes the hash of a map key of an array type. The hash reads
// each element, thus 2 equal keys give 1 hash.
func (e *emitter) arrayKeyHash(t types.Type, v string, depth int) string {
	arr, ok := typeset.Core(t).(*types.Array)
	if !ok {
		return e.hashText(v, t)
	}
	zero, known := zeroSized(arr)
	if zero {
		return "0"
	}
	if !known {
		return fmt.Sprintf("$goHash(%s, %s)", v, e.descriptorOf(t))
	}
	x := fmt.Sprintf("$e%d", depth)
	return fmt.Sprintf("Object.hashAll(%s.map((%s) => %s))",
		v, x, e.arrayKeyHash(arr.Elem(), x, depth+1))
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
	args := ""
	if localStorageCycle(slice) {
		args = "<" + e.dartType(slice.Elem()) + ">"
	}
	if x.Spread {
		source := e.sliceText(x.Args[1])
		if isString(x.Args[1].GoType()) {
			// Go allows append of a string to a byte slice. append reads the
			// source and holds it not, thus the bytes need no copy and no
			// slice header.
			return fmt.Sprintf("$appendString(%s, %s, %s)",
				e.sliceText(x.Args[0]), source, e.alloc(slice.Elem()))
		} else if tp, ok := types.Unalias(x.Args[1].GoType()).(*types.TypeParam); ok &&
			typeset.Core(tp) == nil {
			// A union of a string and a byte slice branches at run time.
			source = fmt.Sprintf("$spreadBytes(%s)", source)
		}
		return fmt.Sprintf("$appendSlice%s(%s, %s, %s%s)",
			args, e.sliceText(x.Args[0]), source, e.alloc(slice.Elem()),
			e.elemAssign(slice.Elem()))
	}
	parts := make([]string, 0, len(x.Args)-1)
	for _, a := range x.Args[1:] {
		parts = append(parts, e.expr(a))
	}
	return fmt.Sprintf("$append%s(%s, <%s>[%s], %s%s)",
		args, e.sliceText(x.Args[0]), e.dartType(slice.Elem()), strings.Join(parts, ", "),
		e.alloc(slice.Elem()), e.elemAssign(slice.Elem()))
}

// elemAssign writes the assign argument of $append and of $appendSlice. A
// struct element and an array element each need a field copy on the path that
// grows. Each other element type gives an empty text, thus the call takes the
// reference write.
func (e *emitter) elemAssign(elem types.Type) string {
	if _, ok := types.Unalias(elem).(*types.TypeParam); ok {
		return ", " + e.dictOf(elem) + ".assign"
	}
	if !needsElementCopy(elem) {
		return ""
	}
	return fmt.Sprintf(", (a, b) => %s", e.assignElem("a", "b", elem))
}

// sliceText writes a slice expression. The Dart type of a type that names
// itself erases to Object?, and the operation needs the slice type back.
func (e *emitter) sliceText(x ir.Expr) string {
	text := e.expr(x)
	if e.dartType(x.GoType()) != "Object?" {
		return text
	}
	slice, ok := typeset.Core(x.GoType()).(*types.Slice)
	if !ok {
		return text
	}
	return fmt.Sprintf("(%s as Slice<%s>)", text, e.dartType(slice.Elem()))
}

func (e *emitter) copyCall(x *ir.Builtin) string {
	dst, src := x.Args[0], x.Args[1]
	source := e.sliceText(src)
	if isString(src.GoType()) {
		// copy reads the source and holds it not, thus the slice shares the
		// bytes of the string.
		source = fmt.Sprintf("$stringBytes(%s)", source)
	}
	slice, _ := typeset.Core(dst.GoType()).(*types.Slice)
	return fmt.Sprintf("$copySlice(%s, %s%s)",
		e.sliceText(dst), source, e.elemAssign(slice.Elem()))
}

func (e *emitter) makeCall(x *ir.Builtin) string {
	switch u := typeset.Core(x.Type).(type) {
	case *types.Slice:
		length := "$gi(0)"
		if len(x.Args) > 0 {
			length = e.expr(x.Args[0])
		}
		// A make with no capacity reads its length expression 1 time, because
		// the expression can hold a side effect.
		if len(x.Args) < 2 {
			return fmt.Sprintf("$makeSliceN<%s>(%s, %s)",
				e.dartType(u.Elem()), length, e.alloc(u.Elem()))
		}
		return fmt.Sprintf("$makeSlice<%s>(%s, %s, %s)",
			e.dartType(u.Elem()), length, e.expr(x.Args[1]), e.alloc(u.Elem()))
	case *types.Map:
		return e.mapNew(u)
	case *types.Chan:
		capacity := "0"
		if len(x.Args) > 0 {
			capacity = "$ix(" + e.expr(x.Args[0]) + ")"
		}
		return fmt.Sprintf("Chan<%s>($chanCapacity(%s, %s), %s)",
			e.dartType(u.Elem()), capacity, e.descriptorOf(u.Elem()), e.zeroFunc(u.Elem()))
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
	if tp, ok := types.Unalias(p.Elem()).(*types.TypeParam); ok {
		value := e.zero(tp)
		if len(x.Args) == 1 {
			value = e.expr(x.Args[0])
		}
		return fmt.Sprintf("$allocPtr(%s, %s)", value, e.dictOf(tp))
	}
	if needsBox(p.Elem()) {
		if len(x.Args) == 1 {
			return fmt.Sprintf("Box<%s>(%s)", e.dartType(p.Elem()), e.expr(x.Args[0]))
		}
		return fmt.Sprintf("Box<%s>(%s)", e.dartType(p.Elem()), e.zero(p.Elem()))
	}
	if len(x.Args) == 1 {
		return e.copyOf(e.expr(x.Args[0]), p.Elem())
	}
	return e.zero(p.Elem())
}

func (e *emitter) minMax(x *ir.Builtin) string {
	fn := "$minOf"
	if x.Name == "max" {
		fn = "$maxOf"
	}
	// A Go value of an unsigned 64-bit type above 2 to the power 63 holds a
	// negative Dart int, thus the signed compare of the 2 helpers inverts the
	// order. A narrow unsigned type holds no such value.
	unsigned := isUnsigned64(x.Type)
	if unsigned {
		fn = "$uminOf"
		if x.Name == "max" {
			fn = "$umaxOf"
		}
	}
	out := e.expr(x.Args[0])
	for _, a := range x.Args[1:] {
		out = fmt.Sprintf("%s(%s, %s)", fn, out, e.expr(a))
	}
	// The 2 unsigned helpers each give a GoInt, thus a cast of the result
	// would read as unnecessary.
	if unsigned {
		return out
	}
	return fmt.Sprintf("(%s as %s)", out, e.dartType(x.Type))
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
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		return fmt.Sprintf("$genericPrintText(%s, %s.desc)", e.expr(x), e.dictOf(tp))
	}
	switch {
	case isString(t):
		return e.expr(x) + ".toDartString()"
	case isFloat(t), isComplexType(t):
		return fmt.Sprintf("$goFormat(%s, GoString.ascii('g')).toDartString()", e.box(x, t))
	case isPointerType(t), isMapType(t), isChanType(t), isSliceType(t), signatureOf(t) != nil:
		return fmt.Sprintf("$referenceText(%s)", e.box(x, t))
	case isInterfaceType(t):
		return fmt.Sprintf("$interfaceText(%s)", e.expr(x))
	case isUnsigned64(t):
		return fmt.Sprintf("$ustring(%s)", e.expr(x))
	case isInteger(t):
		return e.expr(x) + ".toString()"
	}
	if b, ok := types.Unalias(t).Underlying().(*types.Basic); ok {
		switch b.Kind() {
		case types.Bool:
			return e.expr(x) + ".toString()"
		case types.UnsafePointer:
			return fmt.Sprintf("$referenceText(%s)", e.box(x, t))
		}
	}
	e.fail("print does not accept the type %s yet", t)
	return "''"
}

func (e *emitter) syncCallback(t types.Type, value string) string {
	sig := signatureOf(t)
	params := make([]string, sig.Params().Len())
	args := make([]string, len(params))
	for i := range params {
		args[i] = fmt.Sprintf("$a%d", i)
		params[i] = e.dartType(sig.Params().At(i).Type()) + " " + args[i]
	}
	return fmt.Sprintf("((%s $f) => $f == null ? null : $copyFunctionPC((%s) => $syncCall(() => $f(%s)), $f))(%s)", e.dartType(t), strings.Join(params, ", "), strings.Join(args, ", "), value)
}

// conv makes a Go conversion explicit in Dart.
func (e *emitter) conv(x *ir.Conv) string {
	if x.Sync {
		return e.syncCallback(x.Type, e.expr(x.X))
	}
	from, to := x.From, x.Type
	text := e.expr(x.X)
	if basic := basicOf(to); basic != nil && basic.Kind() == types.UnsafePointer {
		if value, ok := x.X.(*ir.Const); ok && value.Value.Kind() == constant.Int && constant.Sign(value.Value) == 0 {
			return "null"
		}
	}
	fs, fok := typeset.Describe(from)
	ts, tok := typeset.Describe(to)
	_, fp := types.Unalias(from).(*types.TypeParam)
	_, tp := types.Unalias(to).(*types.TypeParam)
	if c, ok := x.X.(*ir.Const); ok && tp && tok && ts.Numeric {
		return e.constant(c.Value, to)
	}
	if (fp || tp) && (isInteger(from) || isFloat(from) || isComplexType(from) || (fok && fs.Numeric)) &&
		(isInteger(to) || isFloat(to) || isComplexType(to) || (tok && ts.Numeric)) {
		return fmt.Sprintf("$convert<%s>(%s, %s, %s)", e.dartType(to), text,
			e.descriptorOf(from), e.descriptorOf(to))
	}
	if (fp || tp) && typeset.Pointers(from) && typeset.Pointers(to) {
		return fmt.Sprintf("$convert<%s>(%s, %s, %s)", e.dartType(to), text,
			e.descriptorOf(from), e.descriptorOf(to))
	}
	// A type set that holds a byte slice and a string holds 2 Dart types, thus
	// the dictionary carries the 2 conversions.
	if typeset.BytesOrRunes(from) && isString(to) {
		return fmt.Sprintf("%s.text!(%s)", e.dictOf(from), text)
	}
	if typeset.BytesOrRunes(to) && isString(from) {
		return fmt.Sprintf("%s.fromText!(%s)", e.dictOf(to), text)
	}
	if typeset.BytesOrText(from) {
		switch {
		case isString(to):
			return fmt.Sprintf("%s.text!(%s)", e.dictOf(from), text)
		case isByteSlice(to):
			return fmt.Sprintf("%s.bytes!(%s)", e.dictOf(from), text)
		}
	}
	if typeset.BytesOrText(to) {
		switch {
		case isString(from):
			return fmt.Sprintf("%s.fromText!(%s)", e.dictOf(to), text)
		case isByteSlice(from):
			return fmt.Sprintf("%s.fromBytes!(%s)", e.dictOf(to), text)
		}
	}
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
		return fmt.Sprintf("$sliceToArray<%s>(%s, %s)",
			e.dartType(a.Elem()), text, intConst(constant.MakeInt64(a.Len()), types.Typ[types.Int]))
	case isPointerType(to) && isSliceType(from):
		// Go 1.17 converts a slice to a pointer to an array. The array shares
		// the elements of the slice, thus a write through the pointer reaches
		// the slice.
		p, _ := typeset.Core(to).(*types.Pointer)
		a, _ := typeset.Core(p.Elem()).(*types.Array)
		if a != nil {
			return fmt.Sprintf("$sliceToArrayPtr<%s>(%s, %s)",
				e.dartType(a.Elem()), text, intConst(constant.MakeInt64(a.Len()), types.Typ[types.Int]))
		}
	case isSliceType(to) && isString(from):
		if isRuneSlice(to) {
			return fmt.Sprintf("$stringToRunes(%s)", text)
		}
		return fmt.Sprintf("$stringToBytes(%s)", text)
	case isFloat(to) && isInteger(from):
		if isUnsigned64(from) {
			text = fmt.Sprintf("$utodouble(%s)", text)
		} else {
			text = fmt.Sprintf("$todouble(%s)", text)
		}
		if basicOf(to).Kind() == types.Float32 {
			return "$f32(" + text + ")"
		}
		return "$f64(" + text + ")"
	case isInteger(to) && isFloat(from):
		text = "(" + text + ").toDouble()"
		// truncate of Dart stops at the largest signed value, thus a target of
		// an unsigned 64-bit type needs the wrap of the unsigned range.
		if isUnsigned64(to) {
			return e.narrow(to, fmt.Sprintf("$fromF64U(%s)", text))
		}
		return e.narrow(to, fmt.Sprintf("$fromF64(%s)", text))
	case isFloat(to) && isFloat(from):
		if basicOf(from).Kind() == types.Float32 {
			if basicOf(to).Kind() == types.Float32 {
				return text
			}
			return "$f64((" + text + ").toDouble())"
		}
		if basicOf(to).Kind() == types.Float32 {
			return "$f32((" + text + ").toDouble())"
		}
		return text
	case isComplexType(to) && isComplexType(from):
		if basicOf(from).Kind() == types.Complex64 && basicOf(to).Kind() != types.Complex64 {
			return "$c128(" + text + ")"
		}
		return e.narrow(to, text)
	case isInteger(to) && isInteger(from):
		return e.narrow(to, text)
	case isString(to) && isString(from), isChanType(to) && isChanType(from):
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
	if isStructType(from) && isStructType(to) && !e.sameClass(from, to) {
		return fmt.Sprintf("$convert<%s>(%s, %s, %s)", e.dartType(to), text,
			e.descriptorOf(from), e.descriptorOf(to))
	}
	if types.Identical(types.Unalias(from).Underlying(), types.Unalias(to).Underlying()) {
		return text
	}
	if isSliceType(from) && isSliceType(to) && types.IdenticalIgnoreTags(from.Underlying(), to.Underlying()) {
		return text
	}
	// Distinct Dart classes need a view that retains the source fields.
	if pf, ok := typeset.Core(from).(*types.Pointer); ok {
		if pt, ok := typeset.Core(to).(*types.Pointer); ok {
			if isStructType(pt.Elem()) {
				if e.sameClass(pf.Elem(), pt.Elem()) {
					return text
				}
				return fmt.Sprintf("$convert<%s>(%s, %s, %s)", e.dartType(to), text,
					e.descriptorOf(from), e.descriptorOf(to))
			} else if types.IdenticalIgnoreTags(types.Unalias(pf.Elem()).Underlying(),
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

// sameClass reports whether 2 named struct types share 1 Dart class.
func (e *emitter) sameClass(a, b types.Type) bool {
	na, nb := namedOf(a), namedOf(b)
	if na == nil || nb == nil {
		return false
	}
	// A defined type over an instance of a generic type reads the class of the
	// instance, and the 2 instances of 1 instantiation hold 1 struct object.
	// Read instanceRep.
	ra, rb := e.structRef(na), e.structRef(nb)
	if ra != na || rb != nb {
		sa, _ := ra.Underlying().(*types.Struct)
		sb, _ := rb.Underlying().(*types.Struct)
		return sa != nil && sa == sb
	}
	return e.mergeRep(na.Obj()) == e.mergeRep(nb.Obj())
}

// isByteSlice reports whether a type is a slice of bytes.
func isByteSlice(t types.Type) bool {
	s, ok := typeset.Core(t).(*types.Slice)
	if !ok {
		return false
	}
	b, ok := types.Unalias(s.Elem()).Underlying().(*types.Basic)
	return ok && b.Kind() == types.Uint8
}

// funcAdapt wraps a function value whose Dart type differs from the Dart type
// of the target.
//
// The Dart declaration of a generic type erases a pointer to a type parameter
// to Object?, and the substituted type of an instance holds the class of the
// type argument. A Dart function parameter is contravariant, thus
// `Function(Encoder$?)` is no subtype of `Function(Object?)` and the reverse
// pair is no subtype either. The wrapper takes the parameter types of the
// target and casts each parameter that differs. A result position stays
// covariant, thus it needs no cast.
//
// A nil function value keeps its nil. Go panics at the call of such a value,
// and a wrapper of it would hide that panic.
func (e *emitter) funcAdapt(text string, from, want types.Type) string {
	fs, ws := signatureOf(from), signatureOf(want)
	if fs == nil || ws == nil {
		return text
	}
	if e.dartType(want) == "Object?" {
		return text
	}
	if fs.Params().Len() != ws.Params().Len() || fs.Variadic() != ws.Variadic() {
		return text
	}
	// A type parameter of the declaration keeps its name in the Dart type, and
	// Dart replaces that name with the type argument of the call. Only a
	// parameter that the declaration erases to Object? needs the wrapper.
	params := make([]string, ws.Params().Len())
	args := make([]string, len(params))
	erased := false
	for i := range params {
		wt := e.dartType(ws.Params().At(i).Type())
		ft := e.dartType(fs.Params().At(i).Type())
		n := fmt.Sprintf("$w%d", i)
		if wt != ft && (wt == "Object?" || ft == "Object?") {
			erased = true
			params[i] = wt + " " + n
			args[i] = "(" + n + " as " + ft + ")"
			continue
		}
		params[i] = ft + " " + n
		args[i] = n
	}
	if !erased {
		return text
	}
	return fmt.Sprintf("((%s $f) => $f == null ? null : (%s) => $f!(%s))(%s)",
		e.dartType(from), strings.Join(params, ", "),
		strings.Join(args, ", "), text)
}

// declSignature gives the signature that the Dart declaration of the callee
// writes. 1 Dart function serves each instance of a generic declaration, thus
// the parameter types of an instance read the origin.
func (e *emitter) declSignature(x *ir.Call) *types.Signature {
	switch fun := x.Fun.(type) {
	case *ir.Method:
		if fun.Obj != nil {
			return signatureOf(fun.Obj.Origin().Type())
		}
	case *ir.Ident:
		if fn, ok := fun.Obj.(*types.Func); ok {
			return signatureOf(fn.Origin().Type())
		}
	}
	return signatureOf(x.Fun.GoType())
}

// adaptArg wraps 1 argument of a call against the parameter type that the
// declaration of the callee writes.
func (e *emitter) adaptArg(text string, a ir.Expr, decl *types.Signature, i int) string {
	if decl == nil || i >= decl.Params().Len() {
		return text
	}
	return e.funcAdapt(text, a.GoType(), decl.Params().At(i).Type())
}

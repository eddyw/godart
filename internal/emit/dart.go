package emit

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
)

// The external Dart declaration.
//
// A Go function declaration with no body that carries //gart:dart names 1 Dart
// symbol. This file writes the import of the library and the body of the call.
// Stage 2 already rejected each type that the boundary cannot carry, thus the
// conversions below cover the whole set.

// dartCoreLib names the library that each Dart file holds already. A prefixed
// import of it would hide the plain names, thus double and String would leave
// the file.
const dartCoreLib = "dart:core"

// dartPrefix gives the import prefix of a Dart library, and it holds the order
// of the first read. The core library takes no prefix and no import.
func (e *emitter) dartPrefix(uri string) string {
	if uri == dartCoreLib {
		return ""
	}
	if p, ok := e.dartLibs[uri]; ok {
		return p
	}
	if e.dartLibs == nil {
		e.dartLibs = map[string]string{}
	}
	p := fmt.Sprintf("$d%d", len(e.dartLibs))
	e.dartLibs[uri] = p
	e.dartOrder = append(e.dartOrder, uri)
	return p
}

// dartCall writes the body of an external Dart declaration.
func (e *emitter) dartCall(f *ir.Func) {
	args := make([]string, len(f.Params))
	for i, p := range f.Params {
		args[i] = dartArg(p.Type, paramName(p))
	}
	target := f.DartName
	if prefix := e.dartPrefix(f.DartLib); prefix != "" {
		target = prefix + "." + target
	}
	call := fmt.Sprintf("%s(%s)", target, strings.Join(args, ", "))
	if len(f.Results) == 0 {
		e.line("%s;", call)
		return
	}
	e.line("return %s;", dartResult(f.Results[0].Type, call))
}

// dartArg gives the Dart expression of 1 argument of an external call.
//
// A bool and a float64 hold the Dart form already, and a Go int is an
// extension type over the Dart int. Only a string needs a conversion.
func dartArg(t types.Type, name string) string {
	if isDartString(t) {
		return name + ".toDartString()"
	}
	return name
}

// dartResult gives the Dart expression of the result of an external call.
//
// A Dart String and a Dart int each need the Go form, because GoString holds
// bytes and GoInt is a type of its own.
func dartResult(t types.Type, call string) string {
	switch {
	case isDartString(t):
		return "GoString.fromString(" + call + ")"
	case isDartInt(t):
		return "$gi(" + call + ")"
	}
	return call
}

func isDartString(t types.Type) bool {
	b, ok := types.Unalias(t).(*types.Basic)
	return ok && b.Kind() == types.String
}

func isDartInt(t types.Type) bool {
	b, ok := types.Unalias(t).(*types.Basic)
	return ok && b.Kind() == types.Int
}

// dartObjectPath and dartObjectName name the Go type that holds 1 Dart object.
const (
	dartObjectPath = "syscall/dart"
	dartObjectName = "Object"
)

// isDartObject reports whether a named type is the opaque Dart object.
func isDartObject(obj *types.TypeName) bool {
	return obj != nil && obj.Name() == dartObjectName &&
		obj.Pkg() != nil && obj.Pkg().Path() == dartObjectPath
}

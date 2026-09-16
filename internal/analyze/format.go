package analyze

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"
)

// The string format functions can block only through operand methods.
// Keep their concrete method targets when all operand types are known.
func (a *analyzer) formatTargets(call *ast.CallExpr, callee source) (source, bool) {
	if call.Ellipsis.IsValid() || len(callee.keys) != 1 || len(callee.cells) != 0 {
		return source{}, false
	}
	fn, ok := callee.keys[0].(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "fmt" {
		return source{}, false
	}
	first := 0
	switch fn.Name() {
	case "Sprintf":
		first = 1
	case "Sprint", "Sprintln":
	default:
		return source{}, false
	}
	stringable := true
	if first == 1 {
		value := a.pkg.TypesInfo.Types[call.Args[0]].Value
		stringable = value != nil && value.Kind() == constant.String && simpleStringFormat(constant.StringVal(value), len(call.Args)-1)
	}
	var out source
	for _, arg := range call.Args[first:] {
		t := a.typeOf(arg)
		if targets, ok := formatTypeTargets([]types.Type{t}); ok {
			out.keys = append(out.keys, targets...)
			continue
		}
		it := basicIface(t)
		if !stringable || it == nil || !types.Implements(it, types.Universe.Lookup("error").Type().Underlying().(*types.Interface)) {
			return source{}, false
		}
		// A string verb calls Formatter first, then Error. Neither path reads
		// the underlying fields. Keep both groups and their recovered panics.
		formatter := fn.Pkg().Scope().Lookup("Formatter").Type().Underlying().(*types.Interface)
		for _, name := range []string{"Error", "Format"} {
			var method *types.Func
			if name == "Format" {
				method = formatter.Method(0)
			} else {
				obj, _, _ := types.LookupFieldOrMethod(it, false, nil, name)
				method = obj.(*types.Func)
			}
			cell := methodCell{key: methodKey(name, method.Type().(*types.Signature)), iface: interfaceKey(it)}
			a.methodCells[cell] = it
			out.cells = append(out.cells, cell)
		}
	}
	return out, true
}

// Restrict the proof to plain string verbs. Flags, widths, argument indexes,
// and extra operands can make fmt inspect the underlying value.
func simpleStringFormat(format string, operands int) bool {
	count := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i == len(format) {
			return false
		}
		if format[i] == '%' {
			continue
		}
		if !strings.ContainsRune("vsxXq", rune(format[i])) {
			return false
		}
		count++
	}
	return count == operands
}

func formatTypeTargets(operands []types.Type) ([]any, bool) {
	var targets []any
	seen := map[types.Type]bool{}
	var visit func(types.Type) bool
	visit = func(t types.Type) bool {
		t = types.Unalias(t)
		if seen[t] {
			return true
		}
		seen[t] = true
		if n, ok := t.(*types.Named); ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == "reflect" && n.Obj().Name() == "Value" {
			return false
		}
		if _, ok := t.(*types.TypeParam); ok {
			return false
		}
		methods := types.NewMethodSet(t)
		for i := range methods.Len() {
			m := methods.At(i).Obj().(*types.Func)
			switch m.Name() {
			case "Format", "String", "GoString", "Error":
				targets = append(targets, m.Origin())
			}
		}
		switch t := t.Underlying().(type) {
		case *types.Interface:
			return false
		case *types.Pointer:
			return visit(t.Elem())
		case *types.Slice:
			return visit(t.Elem())
		case *types.Array:
			return visit(t.Elem())
		case *types.Map:
			return visit(t.Key()) && visit(t.Elem())
		case *types.Struct:
			for i := range t.NumFields() {
				if !visit(t.Field(i).Type()) {
					return false
				}
			}
		}
		return true
	}
	for _, operand := range operands {
		if !visit(operand) {
			return nil, false
		}
	}
	return targets, true
}

// fmt prints a recovered method panic through the panic value's own methods.
func (a *analyzer) formatMethodAsync(key any, seen map[any]bool) bool {
	if a.result.blocking[key] {
		return true
	}
	if seen[key] {
		return false
	}
	seen[key] = true
	s := a.summaries[key]
	if s == nil {
		return false
	}
	if s.unknownPanic {
		return true
	}
	for _, fn := range s.panicMethods {
		if a.formatMethodAsync(fn, seen) {
			return true
		}
	}
	for _, site := range s.sites {
		targets := site.targets
		if site.formatOperands {
			targets = site.formatMethods
		}
		for _, fn := range targets {
			if a.formatMethodAsync(fn, seen) {
				return true
			}
		}
	}
	return false
}

package main

import (
	"go/types"
	"slices"
	"strconv"
	"strings"
)

type APIReport struct {
	GoVersion string       `json:"go_version"`
	GOOS      string       `json:"goos"`
	GOARCH    string       `json:"goarch"`
	Absent    []string     `json:"unassessed_packages"`
	Packages  []PackageAPI `json:"packages"`
}

type PackageAPI struct {
	Package string      `json:"package"`
	Symbols []SymbolAPI `json:"symbols"`
}

type SymbolAPI struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Native string `json:"native"`
	Gart   string `json:"gart,omitempty"`
}

func compareAPI(path string, native, gart *types.Scope) PackageAPI {
	want, got := contracts(native), contracts(gart)
	result := PackageAPI{Package: path}
	var names []string
	for name := range want {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		status := "match"
		if _, ok := got[name]; !ok {
			status = "missing"
		} else if got[name] != want[name] {
			status = "different"
		}
		result.Symbols = append(result.Symbols, SymbolAPI{name, status, want[name], got[name]})
	}
	return result
}

func contractType(t types.Type) string {
	return contractTypeParams(t, map[*types.TypeParam]string{})
}

func contractTypeParams(t types.Type, params map[*types.TypeParam]string) string {
	render := func(t types.Type) string { return contractTypeParams(t, params) }
	switch t := t.(type) {
	case *types.TypeParam:
		if name, ok := params[t]; ok {
			return name
		}
	case *types.Signature:
		local := make(map[*types.TypeParam]string, len(params)+t.TypeParams().Len())
		for param, name := range params {
			local[param] = name
		}
		for i := 0; i < t.TypeParams().Len(); i++ {
			local[t.TypeParams().At(i)] = "$T" + strconv.Itoa(len(params)+i)
		}
		render = func(t types.Type) string { return contractTypeParams(t, local) }
		var declarations []string
		for i := 0; i < t.TypeParams().Len(); i++ {
			p := t.TypeParams().At(i)
			declarations = append(declarations, local[p]+" "+render(p.Constraint()))
		}
		tuple := func(tuple *types.Tuple, variadic bool) string {
			parts := make([]string, tuple.Len())
			for i := range parts {
				typ := tuple.At(i).Type()
				if variadic && i == len(parts)-1 {
					parts[i] = "..." + render(typ.(*types.Slice).Elem())
				} else {
					parts[i] = render(typ)
				}
			}
			return strings.Join(parts, ", ")
		}
		text := "func"
		if len(declarations) > 0 {
			text += "[" + strings.Join(declarations, ", ") + "]"
		}
		text += "(" + tuple(t.Params(), t.Variadic()) + ")"
		switch t.Results().Len() {
		case 0:
		case 1:
			text += " " + tuple(t.Results(), false)
		default:
			text += " (" + tuple(t.Results(), false) + ")"
		}
		return text
	case *types.Pointer:
		return "*" + render(t.Elem())
	case *types.Slice:
		return "[]" + render(t.Elem())
	case *types.Array:
		return "[" + strconv.FormatInt(t.Len(), 10) + "]" + render(t.Elem())
	case *types.Map:
		return "map[" + render(t.Key()) + "]" + render(t.Elem())
	case *types.Chan:
		prefix := "chan "
		if t.Dir() == types.SendOnly {
			prefix = "chan<- "
		} else if t.Dir() == types.RecvOnly {
			prefix = "<-chan "
		}
		return prefix + render(t.Elem())
	case *types.Union:
		parts := make([]string, t.Len())
		for i := range parts {
			term := t.Term(i)
			parts[i] = render(term.Type())
			if term.Tilde() {
				parts[i] = "~" + parts[i]
			}
		}
		slices.Sort(parts)
		return strings.Join(parts, " | ")
	case *types.Interface:
		var parts []string
		for i := 0; i < t.NumExplicitMethods(); i++ {
			method := t.ExplicitMethod(i)
			parts = append(parts, method.Name()+strings.TrimPrefix(render(method.Type()), "func"))
		}
		for i := 0; i < t.NumEmbeddeds(); i++ {
			parts = append(parts, render(t.EmbeddedType(i)))
		}
		slices.Sort(parts)
		return "interface{" + strings.Join(parts, "; ") + "}"
	case *types.Named:
		if t.TypeArgs().Len() > 0 {
			args := make([]string, t.TypeArgs().Len())
			for i := range args {
				args[i] = render(t.TypeArgs().At(i))
			}
			return t.Obj().Pkg().Path() + "." + t.Obj().Name() + "[" + strings.Join(args, ", ") + "]"
		}
	}
	return types.TypeString(t, func(pkg *types.Package) string { return pkg.Path() })
}

// Struct layout and private state do not define an exported API contract.
// Public fields and methods receive separate entries.
func contracts(scope *types.Scope) map[string]string {
	out := map[string]string{}
	if scope == nil {
		return out
	}
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		switch obj := obj.(type) {
		case *types.Const:
			out[name] = "const " + contractType(obj.Type()) + " = " + obj.Val().ExactString()
		case *types.Func:
			out[name] = contractType(obj.Type())
		case *types.Var:
			out[name] = "var " + contractType(obj.Type())
		case *types.TypeName:
			typ := types.Unalias(obj.Type())
			switch underlying := typ.Underlying().(type) {
			case *types.Struct:
				out[name] = "struct"
				for i := 0; i < underlying.NumFields(); i++ {
					field := underlying.Field(i)
					if field.Exported() {
						value := contractType(field.Type()) + " tag=" + underlying.Tag(i)
						if field.Embedded() {
							value += " embedded"
						}
						out[name+"."+field.Name()] = value
					}
				}
			case *types.Interface:
				out[name] = "interface"
			default:
				out[name] = contractType(typ.Underlying())
			}
			if obj.IsAlias() {
				out[name] = "alias " + contractType(typ)
			}
			for _, receiver := range []types.Type{types.NewPointer(typ), typ} {
				set := types.NewMethodSet(receiver)
				for i := 0; i < set.Len(); i++ {
					method := set.At(i).Obj()
					if method.Exported() {
						kind := "pointer "
						if receiver == typ {
							kind = "value "
						}
						out[name+"."+method.Name()] = kind + contractType(method.Type())
					}
				}
			}
		}
	}
	return out
}

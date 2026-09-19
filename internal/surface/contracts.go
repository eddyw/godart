package main

import (
	"go/types"
	"slices"
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
	if signature, ok := t.(*types.Signature); ok {
		anonymous := func(tuple *types.Tuple) *types.Tuple {
			vars := make([]*types.Var, tuple.Len())
			for i := range vars {
				vars[i] = types.NewVar(0, nil, "", tuple.At(i).Type())
			}
			return types.NewTuple(vars...)
		}
		var params []string
		for i := 0; i < signature.TypeParams().Len(); i++ {
			param := signature.TypeParams().At(i)
			params = append(params, param.Obj().Name()+" "+contractType(param.Constraint()))
		}
		t = types.NewSignatureType(nil, nil, nil, anonymous(signature.Params()), anonymous(signature.Results()), signature.Variadic())
		text := types.TypeString(t, func(pkg *types.Package) string { return pkg.Path() })
		if len(params) > 0 {
			text = "func[" + strings.Join(params, ", ") + "]" + strings.TrimPrefix(text, "func")
		}
		return text
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

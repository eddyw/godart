package emit

import (
	"go/types"

	"github.com/eddyw/godart/internal/typeset"
)

func dictionaryParams(t types.Type) []*types.TypeParam {
	var params []*types.TypeParam
	seen := map[types.Type]bool{}
	var walk func(types.Type)
	walk = func(t types.Type) {
		t = types.Unalias(t)
		if seen[t] {
			return
		}
		seen[t] = true
		switch t := t.(type) {
		case *types.TypeParam:
			params = append(params, t)
			terms, _ := typeset.Terms(t.Constraint())
			for _, term := range terms {
				walk(term)
			}
		case *types.Named:
			for i := 0; i < t.TypeArgs().Len(); i++ {
				walk(t.TypeArgs().At(i))
			}
		case *types.Pointer:
			walk(t.Elem())
		case *types.Slice:
			walk(t.Elem())
		case *types.Array:
			walk(t.Elem())
		case *types.Chan:
			walk(t.Elem())
		case *types.Map:
			walk(t.Key())
			walk(t.Elem())
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				walk(t.Field(i).Type())
			}
		case *types.Interface:
			for i := 0; i < t.NumMethods(); i++ {
				walk(t.Method(i).Type())
			}
		case *types.Signature:
			walkTuple(t.Params(), walk)
			walkTuple(t.Results(), walk)
		}
	}
	walk(t)
	return params
}

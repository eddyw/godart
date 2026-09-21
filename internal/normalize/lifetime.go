package normalize

import (
	"go/types"
	"reflect"

	"github.com/eddyw/godart/internal/ir"
)

// A suspended frame must not retain a temporary after its last possible use.
// Source variables and closure captures retain their existing lifetime.
func releaseTemporaries(m *ir.Machine) {
	if !m.Blocking {
		return
	}
	uses := make([]map[string]bool, len(m.States))
	next := make([][]int, len(m.States))
	captured := map[string]bool{}
	for i, state := range m.States {
		uses[i] = map[string]bool{}
		walkLifetime(reflect.ValueOf(state), func(node any) bool {
			switch n := node.(type) {
			case *ir.Ident:
				uses[i][n.Name] = true
			case *ir.Jump:
				next[i] = append(next[i], n.State)
			case *ir.DeferNext:
				next[i] = append(next[i], n.Body, n.Done)
			case *ir.FuncLit:
				walkLifetime(reflect.ValueOf(n), func(inner any) bool {
					if id, ok := inner.(*ir.Ident); ok {
						captured[id.Name] = true
					}
					return true
				})
				return false
			}
			return true
		})
		if m.DeferBlocks {
			next[i] = append(next[i], m.DeferState)
		}
	}
	for changed := true; changed; {
		changed = false
		for i := len(m.States) - 1; i >= 0; i-- {
			for _, successor := range next[i] {
				for name := range uses[successor] {
					if !uses[i][name] {
						uses[i][name] = true
						changed = true
					}
				}
			}
		}
	}
	for i, state := range m.States {
		var clear []ir.Stmt
		for _, v := range m.Locals {
			if v.Obj != nil || v.Boxed || v.Dart != "" || captured[v.Name] || uses[i][v.Name] {
				continue
			}
			switch v.Type.Underlying().(type) {
			case *types.Slice, *types.Map, *types.Pointer, *types.Interface, *types.Signature, *types.Tuple:
				clear = append(clear, &ir.Assign{Lhs: identOf(v), Rhs: &ir.Zero{Type: v.Type}})
			}
		}
		state.List = append(clear, state.List...)
	}
}

func walkLifetime(v reflect.Value, visit func(any) bool) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return
		}
		if v.Kind() == reflect.Pointer && v.Type().Elem().PkgPath() == irPkgPath {
			if !visit(v.Interface()) {
				return
			}
		}
		walkLifetime(v.Elem(), visit)
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkLifetime(v.Index(i), visit)
		}
	case reflect.Struct:
		if v.Type().PkgPath() != irPkgPath {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			walkLifetime(v.Field(i), visit)
		}
	}
}

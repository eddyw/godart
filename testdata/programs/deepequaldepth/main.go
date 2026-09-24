package main

import (
	"fmt"
	"reflect"
)

type Box struct{ Value any }

func nest(wrap func(any) any, leaf any) any {
	for i := 0; i < 20000; i++ {
		leaf = wrap(leaf)
	}
	return leaf
}

func main() {
	wrappers := []func(any) any{
		func(v any) any { return &Box{v} },
		func(v any) any { return []any{v} },
		func(v any) any { return map[string]any{"next": v} },
		func(v any) any { return [1]any{v} },
		func(v any) any { return Box{v} },
	}
	for i, wrap := range wrappers {
		a, b, c := nest(wrap, 1), nest(wrap, 1), nest(wrap, 2)
		if !reflect.DeepEqual(a, b) || reflect.DeepEqual(a, c) {
			panic(i)
		}
		fmt.Println(i, true)
	}
}

// The program proves that `any` and `interface{}` name 1 type. A type
// assertion compares the identity of the type, thus the 2 spellings must give
// 1 name. The decode of encoding/json into an empty interface builds the same
// map, and an assertion of the result must hold.
package main

import (
	"encoding/json"
	"fmt"
)

type alias = any

func main() {
	var m any = map[string]interface{}{"k": 1}
	_, ok := m.(map[string]any)
	fmt.Println("spelling", ok)

	var s any = []any{1, 2}
	_, ok = s.([]interface{})
	fmt.Println("slice", ok)

	var a any = map[string]alias{}
	_, ok = a.(map[string]interface{})
	fmt.Println("alias", ok)

	var f any = func(x any) any { return x }
	_, ok = f.(func(interface{}) interface{})
	fmt.Println("func", ok)

	var v any
	if err := json.Unmarshal([]byte(`{"a":{"b":1},"c":[1,2]}`), &v); err != nil {
		fmt.Println("err", err)
		return
	}
	fmt.Printf("top %T\n", v)
	top, ok := v.(map[string]any)
	fmt.Println("top assert", ok)
	fmt.Printf("inner %T\n", top["a"])
	_, ok = top["a"].(map[string]any)
	fmt.Println("inner assert", ok)
	fmt.Printf("list %T\n", top["c"])
	_, ok = top["c"].([]any)
	fmt.Println("list assert", ok)
}

// The program reads a pointer field through encoding/json. The decoder builds a
// cell, and that cell must hold the Dart type of the element, because the setter
// of the field casts to that exact type.
package main

import (
	"encoding/json"
	"fmt"
)

type inner struct {
	N int `json:"n"`
}

type record struct {
	P *float64 `json:"p"`
	Q *int     `json:"q"`
	S *string  `json:"s"`
	B *bool    `json:"b"`
	L *[]int   `json:"l"`
	M *inner   `json:"m"`
	E *int     `json:"e"`
}

func main() {
	var back record
	text := `{"p":1.5,"q":2,"s":"x","b":true,"l":[1,2],"m":{"n":3},"e":null}`
	fmt.Println(json.Unmarshal([]byte(text), &back))
	fmt.Println(*back.P, *back.Q, *back.S, *back.B)
	fmt.Println(*back.L, back.M.N, back.E == nil)

	// The round trip writes the same text.
	out, err := json.Marshal(&back)
	fmt.Println(string(out), err)

	// A pointer to a pointer also reads.
	type deep struct {
		D **int `json:"d"`
	}
	var two deep
	fmt.Println(json.Unmarshal([]byte(`{"d":7}`), &two), **two.D)

	// A slice of pointers reads each element.
	var list []*int
	fmt.Println(json.Unmarshal([]byte(`[1,2,3]`), &list), *list[0], *list[2])

	// A map of pointers reads each value.
	var table map[string]*float64
	fmt.Println(json.Unmarshal([]byte(`{"a":0.5}`), &table), *table["a"])
}

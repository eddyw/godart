// jsonptr reads a pointer field of each kind through encoding/json.
package main

import (
	"encoding/json"
	"fmt"
)

type inner struct{ N int }

type r struct {
	P *float64        `json:"p"`
	Q *int            `json:"q"`
	S *string         `json:"s"`
	B *bool           `json:"b"`
	I *inner          `json:"i"`
	A *[2]int         `json:"a"`
	L *[]int          `json:"l"`
	M *map[string]int `json:"m"`
	Z *int            `json:"z"`
}

func main() {
	var back r
	text := `{"p":1.5,"q":2,"s":"x","b":true,"i":{"N":3},` +
		`"a":[1,2],"l":[4,5],"m":{"k":6},"z":null}`
	fmt.Println(json.Unmarshal([]byte(text), &back))
	fmt.Println(*back.P, *back.Q, *back.S, *back.B, back.I.N)
	fmt.Println(*back.A, *back.L, (*back.M)["k"], back.Z == nil)

	out, err := json.Marshal(back)
	fmt.Println(string(out), err)
}

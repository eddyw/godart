package main

import (
	"encoding/json"
	"fmt"
)

type box struct {
	N int64   `json:"n"`
	U uint64  `json:"u"`
	F float64 `json:"f"`
}

func main() {
	var b box
	err := json.Unmarshal([]byte(`{"n":9007199254740993,"u":18446744073709551615,"f":1.5}`), &b)
	fmt.Println(err, b.N, b.U, b.F)

	var a any
	err = json.Unmarshal([]byte(`{"n":9007199254740993}`), &a)
	fmt.Println(err, a)
}

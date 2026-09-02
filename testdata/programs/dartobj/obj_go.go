//go:build !dart

package main

import "encoding/json"

type object = any

func decode(text string) object {
	var out object
	json.Unmarshal([]byte(text), &out)
	return out
}

func encode(value object) string {
	out, _ := json.Marshal(value)
	return string(out)
}

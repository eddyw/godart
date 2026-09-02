package main

import "fmt"

// The 2 declarations of obj_dart.go carry a Dart object between them. Neither
// Go nor the Dart output reads a field of that object.
func main() {
	for _, text := range []string{
		`{"a":1,"b":[2,3],"c":"x"}`,
		`[1,2,3]`,
		`"a text"`,
		`true`,
		`null`,
	} {
		fmt.Printf("%s -> %s\n", text, encode(decode(text)))
	}
}

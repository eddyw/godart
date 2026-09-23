package main

import "fmt"

type F24758 struct{ A int }
type F50385 struct{ B int }

func main() {
	var first chan F24758
	var second chan F50385
	fmt.Println(first == nil, second == nil)
}

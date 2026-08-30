// The program proves the go/format subtree: go/token, go/scanner, go/ast,
// go/parser, go/printer, and text/tabwriter. It also proves 4 subset rules
// that the subtree needed: the elided address of a pointer literal, a package
// variable pair from 1 call, an identifier that Dart rejects, and the cast of
// the erased result of a generic call.
package main

import (
	"fmt"
	"go/format"
	"strings"
)

func pair() (int, string) { return 42, "pair" }

var num, text = pair()

type node struct {
	name string
}

func main() {
	src := "package  x\n\nfunc  f(a int,b string ) (int ,error) {\n\treturn a,nil\n}\n"
	out, err := format.Source([]byte(src))
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	fmt.Print(string(out))

	χ2 := 0.5
	χ2 = χ2 * 4
	fmt.Println("chi:", χ2)
	fmt.Println("vars:", num, text)

	nodes := []*node{{name: "a"}, {name: "b"}}
	names := []string{}
	for _, n := range nodes {
		names = append(names, n.name)
	}
	fmt.Println("nodes:", strings.Join(names, ","))
}

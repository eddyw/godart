package main

import (
	"fmt"
	"path"
)

// A local struct type takes the Dart class of its shape.
func spans(text string) int {
	type span struct {
		start int
		end   int
	}
	out := make([]span, 0, 4)
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '/' {
			out = append(out, span{start: start, end: i})
			start = i + 1
		}
	}
	return len(out)
}

func main() {
	fmt.Println(path.Join("a", "b", "c"), path.Base("/x/y.txt"), path.Ext("y.txt"))
	fmt.Println(path.Dir("/x/y.txt"), path.Clean("a//b/../c"), path.IsAbs("/a"))
	m, err := path.Match("*.go", "main.go")
	fmt.Println(m, err == nil)
	fmt.Println(path.Split("/x/y.txt"))
	fmt.Println(path.Join(), path.Base(""), path.Clean(""))
	fmt.Println(spans("a/b/c/"))
}

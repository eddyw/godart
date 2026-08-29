package main

import (
	"fmt"
	"strconv"
)

func main() {
	for _, in := range []string{
		`"plain"`,
		`"a\tb\n"`,
		`"é\U0001F600"`,
		"`raw \\n text`",
		`'x'`,
		`'\n'`,
		`"bad`,
		`"\q"`,
	} {
		out, err := strconv.Unquote(in)
		fmt.Printf("%q -> %q %v\n", in, out, err)
	}

	v, mb, tail, err := strconv.UnquoteChar(`érest`, '"')
	fmt.Println(v, mb, tail, err)

	fmt.Printf("%#x %#X %#o %#b\n", 255, 255, 8, 5)
	fmt.Printf("%#x %#x\n", -255, 0)
}

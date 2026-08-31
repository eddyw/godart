package main

import (
	"fmt"
	"runtime"
	"strings"
)

// here gives the file and the line of its caller.
func here() (string, int) {
	_, file, line, _ := runtime.Caller(1)
	return file, line
}

// The text of a line directive passes into runtime.Caller verbatim. Go names
// a true file with the path of the build machine, thus the first case prints
// the suffix only.
func main() {
	f1, l1 := here()
	fmt.Println(strings.HasSuffix(f1, "main.go"), l1)
//line lied.go:100
	f2, l2 := here()
	fmt.Println(f2, l2)
//line other/dir/x.go:200
	f3, l3 := here()
	fmt.Println(f3, l3)
	f4, l4 := here()
	fmt.Println(f4, l4)
}

package main

import (
	"fmt"
	"log"
	"strings"
)

// Go writes the file of Llongfile as the build machine holds it, and gart
// writes the module form. The 2 share the text after the last separator, thus
// the program prints that part only.
func main() {
	var b strings.Builder
	l := log.New(&b, "p ", log.Llongfile)
	l.Print("hello")
	text := strings.TrimSuffix(b.String(), "\n")
	fmt.Println(strings.HasPrefix(text, "p "))
	fmt.Println(text[strings.LastIndexByte(text, '/')+1:])

	b.Reset()
	l.SetFlags(log.Llongfile | log.Lshortfile)
	l.Print("short wins")
	fmt.Println(strings.TrimSuffix(b.String(), "\n"))
}
